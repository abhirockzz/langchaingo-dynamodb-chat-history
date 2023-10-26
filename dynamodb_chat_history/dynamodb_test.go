package dynamodb_chat_history

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/schema"
)

// go test -run ^TestDynamoDBChatMessageHistory_AddUserMessage$ langchaingo/dynamodb_chat_history
func TestDynamoDBChatMessageHistory_AddUserMessage(t *testing.T) {

	tableName := "test-table"
	pkName := "chat_id"
	pkValue := "42"
	region := "us-east-1"

	history, err := New(region, WithTableName(tableName), WithPrimaryKeyName(pkName), WithPrimaryKeyValue(pkValue))
	assert.Nil(t, err)

	err = createTable(history)
	assert.Nil(t, err)

	defer deleteTable(history)

	messsage := "test"
	err = history.AddUserMessage(context.Background(), messsage)
	assert.Nil(t, err)

	found, err := queryByPrimaryKey(history, messsage, schema.ChatMessageTypeHuman)
	assert.Nil(t, err)
	assert.True(t, found)
}

func TestDynamoDBChatMessageHistory_AddAIMessage(t *testing.T) {

	tableName := "test-table"
	pkName := "chat_id"
	pkValue := "42"
	region := "us-east-1"

	history, err := New(region, WithTableName(tableName), WithPrimaryKeyName(pkName), WithPrimaryKeyValue(pkValue))
	assert.Nil(t, err)

	err = createTable(history)
	assert.Nil(t, err)

	defer deleteTable(history)

	messsage := "test"
	err = history.AddAIMessage(context.Background(), messsage)
	assert.Nil(t, err)

	found, err := queryByPrimaryKey(history, messsage, schema.ChatMessageTypeAI)
	assert.Nil(t, err)
	assert.True(t, found)
}

func TestDynamoDBChatMessageHistory_AddMessage_Human(t *testing.T) {

	tableName := "test-table"
	pkName := "chat_id"
	pkValue := "42"
	region := "us-east-1"

	history, err := New(region, WithTableName(tableName), WithPrimaryKeyName(pkName), WithPrimaryKeyValue(pkValue))
	assert.Nil(t, err)

	err = createTable(history)
	assert.Nil(t, err)

	defer deleteTable(history)

	messsage := "test-human"
	err = history.AddMessage(context.Background(), schema.HumanChatMessage{Content: messsage})
	assert.Nil(t, err)

	found, err := queryByPrimaryKey(history, messsage, schema.ChatMessageTypeHuman)
	assert.Nil(t, err)
	assert.True(t, found)
}

func TestDynamoDBChatMessageHistory_AddMessage_AI(t *testing.T) {

	tableName := "test-table"
	pkName := "chat_id"
	pkValue := "42"
	region := "us-east-1"

	history, err := New(region, WithTableName(tableName), WithPrimaryKeyName(pkName), WithPrimaryKeyValue(pkValue))
	assert.Nil(t, err)

	err = createTable(history)
	assert.Nil(t, err)

	defer deleteTable(history)

	messsage := "test-ai"
	err = history.AddMessage(context.Background(), schema.AIChatMessage{Content: messsage})
	assert.Nil(t, err)

	found, err := queryByPrimaryKey(history, messsage, schema.ChatMessageTypeAI)
	assert.Nil(t, err)
	assert.True(t, found)
}

func queryByPrimaryKey(h *DynamoDBChatMessageHistory, expectedValue string, expectedType schema.ChatMessageType) (bool, error) {

	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			h.primaryKeyName: &types.AttributeValueMemberS{Value: h.PrimaryKeyValue},
		},
	}

	result, err := h.client.GetItem(context.Background(), getItemInput)
	if err != nil {
		return false, err
	}

	ddbMessages := result.Item["messages"]

	if ddbMessages == nil {
		return false, nil
	}

	listOfMessages := ddbMessages.(*types.AttributeValueMemberL)
	message := listOfMessages.Value[0].(*types.AttributeValueMemberM).Value

	content := message["content"].(*types.AttributeValueMemberS).Value
	messageType := message["type"].(*types.AttributeValueMemberS).Value

	return (content == expectedValue && messageType == string(expectedType)), nil

}

func createTable(h *DynamoDBChatMessageHistory) error {

	_, err := h.client.CreateTable(context.Background(), &dynamodb.CreateTableInput{
		TableName: aws.String(h.tableName),
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: &h.primaryKeyName,
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: &h.primaryKeyName,
				KeyType:       types.KeyTypeHash,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
	})

	if err != nil {
		return err
	}

	//wait before completion
	waiter := dynamodb.NewTableExistsWaiter(h.client)

	params := &dynamodb.DescribeTableInput{
		TableName: aws.String(h.tableName),
	}

	maxWaitTime := 45 * time.Second

	// Wait until it table is created, or max wait time
	// expires.
	err = waiter.Wait(context.Background(), params, maxWaitTime)
	if err != nil {
		return err
	}

	return nil
}

func deleteTable(h *DynamoDBChatMessageHistory) error {
	_, err := h.client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(h.tableName),
	})
	if err != nil {
		return err
	}

	waiter := dynamodb.NewTableNotExistsWaiter(h.client)

	params := &dynamodb.DescribeTableInput{
		TableName: aws.String(h.tableName),
	}

	maxWaitTime := 45 * time.Second

	// Wait until it table is created, or max wait time
	// expires.
	err = waiter.Wait(context.Background(), params, maxWaitTime)
	if err != nil {
		return err
	}

	return nil
}
