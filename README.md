# LangChain chat history memory implementation using Amazon DynamoDB

**Use DynamoDB as the memory/history backend for your Go applications using LangChain**

Here is a sample application you can use to try it.

Start by creating a DynamoDB table:

```shell
export DYNAMODB_TABLE_NAME=test-table
export PARTITION_KEY=chat_id

aws dynamodb create-table \
    --table-name $DYNAMODB_TABLE_NAME \
    --attribute-definitions AttributeName=$PARTITION_KEY,AttributeType=S \
    --key-schema AttributeName=$PARTITION_KEY,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST
```

The sample app below, uses Amazon Bedrock (see line - `llm, err := claude.New(region, llm.DontUseHumanAssistantPrompt())`). For this to work, you will need to make ensure proper IAM permissions. Refer to the ["Before You Begin" section](https://community.aws/concepts/amazon-bedrock-golang-getting-started#before-you-begin).

```json
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

Save the program to a file named `main.go`. 

Then, create a new Go module, run the program and start chatting.

```shell
go mod init demo
go mod tidy

export DYNAMODB_TABLE_NAME=test-table
export PARTITION_KEY=chat_id

go run main.go
```

Complete program:

```go
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	ddbhist "github.com/abhirockzz/langchaingo-dynamodb-chat-history/dynamodb_chat_history"
	"github.com/google/uuid"

	"github.com/abhirockzz/amazon-bedrock-langchain-go/llm"
	"github.com/abhirockzz/amazon-bedrock-langchain-go/llm/claude"
	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/memory"
	"github.com/tmc/langchaingo/outputparser"
	"github.com/tmc/langchaingo/prompts"
)

const template = "{{.chat_history}}\n\nHuman:{{.human_input}}\n\nAssistant:"

func main() {

	region := "us-east-1"

	llm, err := claude.New(region, llm.DontUseHumanAssistantPrompt())

	if err != nil {
		log.Fatal(err)
	}

	//a random uuid is used as primary key value. typically, in an application this would be a unique identifier such as a session ID.
	pkValue := uuid.New().String()

	ddbcmh, err := ddbhist.New(region, ddbhist.WithTableName(os.Getenv("DYNAMODB_TABLE_NAME")), ddbhist.WithPrimaryKeyName(os.Getenv("PARTITION_KEY")), ddbhist.WithPrimaryKeyValue(pkValue))

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

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nEnter your message: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		_, err := chains.Call(ctx, chain, map[string]any{"human_input": input}, chains.WithMaxTokens(8191),
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
```

Verify chat history in DynamoDB:

```shell
export DYNAMODB_TABLE_NAME=test-table

aws dynamodb scan --table-name $DYNAMODB_TABLE_NAME
```

To delete the table:

```shell
export DYNAMODB_TABLE_NAME=test-table

aws dynamodb delete-table --table-name $DYNAMODB_TABLE_NAME
```