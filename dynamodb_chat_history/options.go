package dynamodb_chat_history

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type ConfigOption func(*ConfigOptions)

type ConfigOptions struct {
	TableName       string
	PrimaryKeyName  string
	DynamoDBClient  *dynamodb.Client
	PrimaryKeyValue string
}

func WithTableName(tableName string) ConfigOption {
	return func(o *ConfigOptions) {
		o.TableName = tableName
	}
}

func WithPrimaryKeyName(primaryKeyName string) ConfigOption {
	return func(o *ConfigOptions) {
		o.PrimaryKeyName = primaryKeyName
	}
}

func WithPrimaryKeyValue(primaryKeyValue string) ConfigOption {
	return func(o *ConfigOptions) {
		o.PrimaryKeyValue = primaryKeyValue
	}
}

func WithDynamoDBClient(client *dynamodb.Client) ConfigOption {
	return func(o *ConfigOptions) {
		o.DynamoDBClient = client
	}
}
