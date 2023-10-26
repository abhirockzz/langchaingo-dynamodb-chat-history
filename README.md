# LangChain chat history memory implementation using Amazon DynamoDB

**Use DynamoDB as the memory/history backend for your Go applications using LangChain!**

Here is a sample application you can use to try it. This will create a DynamoDB table - once that's done, you can start the conversation. Press ctrl+c to exit - the program will exit after deleting the table.

- Please ensure the role used has access to Amazon Bedrock. Below is an example IAM permission policy:
	```
	{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": "bedrock:*",
				"Resource": "*"
			}
		]
	}
	```

- Please also ensure that you have access to the Claude base model. More info [here](https://docs.aws.amazon.com/bedrock/latest/userguide/model-access.html)

Save the below to `main.go` and `go run main.go` to try. The example uses Claude implementation for [Amazon Bedrock](https://github.com/abhirockzz/amazon-bedrock-langchain-go/tree/master/llm/claude) but any other LLM supported by LangChain Go should work (e.g. OpenAI)

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	ddbhist "github.com/abhirockzz/langchaingo-dynamodb-chat-history/dynamodb_chat_history"

	"github.com/abhirockzz/amazon-bedrock-langchain-go/llm"
	"github.com/abhirockzz/amazon-bedrock-langchain-go/llm/claude"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
)

const template = "{{.chat_history}}\n\nHuman:{{.human_input}}\n\nAssistant:"

const (
	tableName = "test-table"
	pkName    = "chat_id"
	pkValue   = "42"
	region    = "us-east-1"
)

func main() {

	llm, err := claude.New(region, llm.DontUseHumanAssistantPrompt())

	if err != nil {
		log.Fatal(err)
	}

	client := createClient()

	if err := checkTable(client); err != nil {
		if errCreate := createTable(client); errCreate != nil {
			log.Fatalf("Failed to create table: %v", errCreate)
		}
	}

	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT)
	go func() {
		<-exit
		deleteTable()
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)

	ddbcmh, err := ddbhist.New(region, ddbhist.WithTableName(tableName), ddbhist.WithPrimaryKeyName(pkName), ddbhist.WithPrimaryKeyValue(pkValue))

	if err != nil {
		log.Fatal(err)
	}

	chain := chains.LLMChain{
		Prompt: prompts.NewPromptTemplate(
			template,
			[]string{"chat_history", "human_input"},
		),
		LLM: llm,
		Memory: memory.NewConversationBuffer(
			memory.WithMemoryKey("chat_history"),
			memory.WithAIPrefix("\n\nAssistant"),
			memory.WithHumanPrefix("\n\nHuman"),
			memory.WithChatHistory(ddbcmh),
		),
		OutputParser: outputparser.NewSimple(),
		OutputKey:    "text",
	}

	ctx := context.Background()

	for {
		fmt.Print("\nEnter your message: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		_, err = chains.Call(ctx, chain, map[string]any{"human_input": input}, chains.WithMaxTokens(8191),
			chains.WithStreamingFunc(
				func(ctx context.Context, chunk []byte) error {
					fmt.Print(string(chunk))
					return nil
				},
			))

		if err != nil {
			log.Fatal(err)
		}
	}

}

func createClient() *dynamodb.Client {

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		log.Fatalf("Failed to load configuration, %v", err)
	}
	return dynamodb.NewFromConfig(cfg)
}

func checkTable(client *dynamodb.Client) error {
	_, err := client.DescribeTable(context.Background(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	return err
}

func createTable(client *dynamodb.Client) error {

	_, err := client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(pkName),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(pkName),
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return err
	}
	waiter := dynamodb.NewTableExistsWaiter(client)

	params := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	maxWaitTime := 45 * time.Second

	fmt.Println("waiting for dynamodb table to be ready")

	err = waiter.Wait(context.Background(), params, maxWaitTime)
	if err != nil {
		return err
	}

	fmt.Println("dynamodb table is ready")

	return nil
}

func deleteTable() error {

	region := "us-east-1"

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return err
	}
	client := dynamodb.NewFromConfig(cfg)

	_, err = client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return err
	}

	waiter := dynamodb.NewTableNotExistsWaiter(client)

	params := &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	}

	maxWaitTime := 45 * time.Second

	fmt.Println("waiting for dynamodb table to be deleted")

	err = waiter.Wait(context.Background(), params, maxWaitTime)
	if err != nil {
		return err
	}

	fmt.Println("dynamodb table has been deleted")

	return nil
}
```