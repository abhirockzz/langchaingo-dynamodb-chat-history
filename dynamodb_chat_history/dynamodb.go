package dynamodb_chat_history

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/tmc/langchaingo/schema"
)

// DynamoDBChatMessageHistory is a struct that stores chat messages.
type DynamoDBChatMessageHistory struct {
	tableName       string
	primaryKeyName  string
	PrimaryKeyValue string
	client          *dynamodb.Client
}

func New(region string, options ...ConfigOption) (*DynamoDBChatMessageHistory, error) {

	ddbHistory := &DynamoDBChatMessageHistory{}

	opts := &ConfigOptions{}
	for _, opt := range options {
		opt(opts)
	}

	ddbHistory.tableName = opts.TableName
	ddbHistory.primaryKeyName = opts.PrimaryKeyName
	ddbHistory.primaryKeyValue = opts.PrimaryKeyValue

	if opts.DynamoDBClient == nil {

		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))

		if err != nil {
			return nil, err
		}

		ddbHistory.client = dynamodb.NewFromConfig(cfg)
	} else {
		ddbHistory.client = opts.DynamoDBClient
	}

	return ddbHistory, nil
}

// Statically assert that DynamoDBChatMessageHistory implement the chat message history interface.
var _ schema.ChatMessageHistory = &DynamoDBChatMessageHistory{}

func (h DynamoDBChatMessageHistory) AddMessage(_ context.Context, message schema.ChatMessage) error {
	//fmt.Println("===== DynamoDBChatMessageHistory/AddMessage =====")

	return h.addMessage(context.Background(), message.GetContent(), string(message.GetType()))

}

// AddUserMessage adds an user to the chat message history.
func (h DynamoDBChatMessageHistory) AddUserMessage(_ context.Context, text string) error {
	//fmt.Println("===== DynamoDBChatMessageHistory/AddUserMessage =====")

	return h.addMessage(context.Background(), text, "human")
}

func (h DynamoDBChatMessageHistory) AddAIMessage(_ context.Context, text string) error {
	//fmt.Println("===== DynamoDBChatMessageHistory/AddAIMessage =====")

	return h.addMessage(context.Background(), text, "ai")

}

func (h DynamoDBChatMessageHistory) Clear(_ context.Context) error {
	//fmt.Println("===== DynamoDBChatMessageHistory/Clear called =====")

	_, err := h.client.DeleteTable(context.Background(), &dynamodb.DeleteTableInput{
		TableName: aws.String(h.tableName),
	})
	if err != nil {
		return err
	}

	return nil
}

func (h DynamoDBChatMessageHistory) SetMessages(_ context.Context, messages []schema.ChatMessage) error {
	//fmt.Println("===== DynamoDBChatMessageHistory/SetMessages called =====")

	for _, message := range messages {
		err := h.AddMessage(context.Background(), message)

		if err != nil {
			return err
		}
	}

	return nil
}

// Messages returns all messages stored.
func (h DynamoDBChatMessageHistory) Messages(_ context.Context) ([]schema.ChatMessage, error) {
	//fmt.Println("===== DynamoDBChatMessageHistory/Messages =====")

	return h.getMessages()
}

const updateExpression = "SET #messages = list_append(if_not_exists(#messages, :empty_list), :newMessage)"

func (h *DynamoDBChatMessageHistory) addMessage(_ context.Context, text, messageType string) error {

	expressionAttributeNames := map[string]string{
		"#messages": "messages",
	}

	expressionAttributeValues := map[string]types.AttributeValue{
		":newMessage": &types.AttributeValueMemberL{
			Value: []types.AttributeValue{
				&types.AttributeValueMemberM{
					Value: map[string]types.AttributeValue{
						"type":    &types.AttributeValueMemberS{Value: messageType},
						"content": &types.AttributeValueMemberS{Value: text},
					},
				},
			},
		},
		":empty_list": &types.AttributeValueMemberL{Value: []types.AttributeValue{}},
	}

	primaryKey := map[string]types.AttributeValue{
		h.primaryKeyName: &types.AttributeValueMemberS{Value: h.primaryKeyValue},
	}

	updateItemInput := &dynamodb.UpdateItemInput{
		TableName:                 aws.String(h.tableName),
		Key:                       primaryKey,
		UpdateExpression:          aws.String(updateExpression),
		ExpressionAttributeNames:  expressionAttributeNames,
		ExpressionAttributeValues: expressionAttributeValues,
	}

	_, err := h.client.UpdateItem(context.Background(), updateItemInput)
	if err != nil {
		return err
	}

	return nil
}

func (h *DynamoDBChatMessageHistory) getMessages() ([]schema.ChatMessage, error) {

	getItemInput := &dynamodb.GetItemInput{
		TableName: aws.String(h.tableName),
		Key: map[string]types.AttributeValue{
			h.primaryKeyName: &types.AttributeValueMemberS{Value: h.primaryKeyValue},
		},
	}

	result, err := h.client.GetItem(context.Background(), getItemInput)
	if err != nil {
		return nil, err
	}

	ddbMessages := result.Item["messages"]

	if ddbMessages == nil {
		return nil, nil
	}

	var chatMessages []schema.ChatMessage

	listOfMessages := ddbMessages.(*types.AttributeValueMemberL)

	for _, m := range listOfMessages.Value {

		message := m.(*types.AttributeValueMemberM).Value

		content := message["content"].(*types.AttributeValueMemberS).Value
		messageType := message["type"].(*types.AttributeValueMemberS).Value

		var chatMessage schema.ChatMessage

		if messageType == string(schema.ChatMessageTypeAI) {
			chatMessage = schema.AIChatMessage{Content: content}
		} else if messageType == string(schema.ChatMessageTypeHuman) {
			chatMessage = schema.HumanChatMessage{Content: content}
		}

		chatMessages = append(chatMessages, chatMessage)

	}

	return chatMessages, nil
}
