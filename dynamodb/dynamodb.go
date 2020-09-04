package dynamodb

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type Option struct {
	ID             string  `json:"id"`
	CreatedDate    string  `json:"createdUtc"`
	Stock          int     `json:"stock" dynamodbav:",omitempty"`
	ShaftStiffness float64 `json:"shaftStiffness" dynamodbav:"shaftStiffness,omitempty"`
	Size           string  `json:"size" dynamodbav:"size,omitempty"`     // TODO enum?
	Socket         string  `json:"socket" dynamodbav:"socket,omitempty"` // TODO enum?
	Color          string  `json:"socket" dynamodbav:"color,omitempty"`  // TODO enum?
}

type Product struct {
	ID          string   `json:"id"`
	CreatedDate string   `json:"createdUtc"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       int      `json:"price"`
	Weight      int      `json:"weight"`
	Image       string   `json:"image"`
	Thumbnail   string   `json:"thumbNail"`
	Options     []Option `json:"options" dynamodbav:",omitempty"`
}

type Basket struct {
	ID string `json:"id"`
}

type DynamoDB struct {
	db        *dynamodb.DynamoDB
	tableName string
	endpoint  string
}

func New(endpoint, tableName string) (*DynamoDB, error) {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := dynamodb.New(sess, &aws.Config{
		Endpoint: aws.String(endpoint),
	})

	return &DynamoDB{
		db:        svc,
		tableName: tableName,
	}, nil
}

// AddProduct take a Product p and attempts to put that item into DynamoDB.
func (db *DynamoDB) AddProduct(p Product) (*Product, error) {

	p.CreatedDate = time.Now().Format(time.RFC3339)
	p.ID = uuid.New().String()

	pk := fmt.Sprintf("PRODUCT#%s", p.ID)
	sort := "METADATA#"

	item, err := dynamodbattribute.MarshalMap(&p)
	if err != nil {
		return &p, err
	}
	item["type"] = &dynamodb.AttributeValue{S: aws.String("product")}
	item["PK"] = &dynamodb.AttributeValue{S: aws.String(pk)}
	item["SK"] = &dynamodb.AttributeValue{S: aws.String(sort)}

	_, err = db.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(db.tableName),
		Item:      item,
	})

	return &p, err
}

// AddOptionToProduct adds a single option to a product.
func (db *DynamoDB) AddOptionToProduct(id string, option Option) (*Option, error) {
	option.ID = uuid.New().String()
	option.CreatedDate = time.Now().Format(time.RFC3339)

	pk := fmt.Sprintf("PRODUCT#%s", id)
	sort := fmt.Sprintf("OPTION#%s", option.ID)

	item, err := dynamodbattribute.MarshalMap(&option)
	if err != nil {
		return nil, err
	}
	item["type"] = &dynamodb.AttributeValue{S: aws.String("product_option")}
	item["PK"] = &dynamodb.AttributeValue{S: aws.String(pk)}
	item["SK"] = &dynamodb.AttributeValue{S: aws.String(sort)}

	_, err = db.db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String(db.tableName),
		Item:      item,
	})

	return &option, err
}

// GetProduct fetches the product will all their options included.
func (db *DynamoDB) GetProduct(id string) (*Product, error) {
	var result Product

	res, err := db.db.Query(&dynamodb.QueryInput{
		TableName:              aws.String(db.tableName),
		KeyConditionExpression: aws.String("#PK = :pk"),
		ExpressionAttributeNames: map[string]*string{
			"#PK": aws.String("PK"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pk": {
				S: aws.String(fmt.Sprintf("PRODUCT#%s", id)),
			},
		},
		ScanIndexForward: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if len(res.Items) == 0 {
		// TODO error not found here?
		return nil, nil
	}

	metadata, options := res.Items[0], res.Items[1:]

	err = dynamodbattribute.UnmarshalMap(metadata, &result)
	if err != nil {
		return nil, err
	}

	err = dynamodbattribute.UnmarshalListOfMaps(options, &result.Options)
	if err != nil {
		return nil, err
	}

	return &result, err
}
