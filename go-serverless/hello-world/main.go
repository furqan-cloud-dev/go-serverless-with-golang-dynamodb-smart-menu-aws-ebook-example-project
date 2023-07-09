package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-playground/validator"
)

type MenuItem struct {
	PK       string `json:"pk"`
	SK       string `json:"sk"`
	Entity   string `json:"entity"`
	EntityId string `json:"entityId"`
	GSI1PK   string `json:"indexGSI1PK"`

	Title       string   `json:"title"`
	Description string   `json:"description"`
	ImageUrl    string   `json:"imageUrl"`
	Price       float64  `json:"price"`
	PriceUnit   string   `json:"priceUnit"`
	Category    string   `json:"category"`
	CategoryId  string   `json:"categoryId"`
	SubCategory string   `json:"subCategory"`
	Tags        []string `json:"tags"`
	IsAvailable bool     `json:"isAvailable"`
}

type Category struct {
	PK         string `json:"pk"`
	SK         string `json:"sk"`
	Category   string `json:"category"`
	CategoryId string `json:"categoryId"`
}

type CategoriesRequest struct {
	CountryCode string `json:"countryCode" validate:"required"`
	BranchId    string `json:"branchId" validate:"required"`
}

type MenuItemsRequest struct {
	CountryCode string `json:"countryCode" validate:"required"`
	BranchId    string `json:"branchId" validate:"required"`
	CategoryId  string `json:"categoryId" validate:"required"`
}

type BranchAllMenuItemsRequest struct {
	BranchId string `json:"branchId" validate:"required"`
}

type SingleMenuItemRequest struct {
	PK string `json:"pk" validate:"required"`
	SK string `json:"sk" validate:"required"`
}

var db *dynamodb.Client

const DYNAMO_TABLE_NAME = "SmartMenu"

func main() {
	lambda.Start(handler)
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	if db == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO(), func(o *config.LoadOptions) error {
			o.Region = "us-east-1"
			return nil
		})

		if err != nil {
			return errorResponse(err)
		}

		db = dynamodb.NewFromConfig(cfg)
	}

	var marshalledItem []byte
	validate := validator.New()
	path := request.Path
	method := request.HTTPMethod

	if method == "POST" {
		switch path {
		case "/api/categories":
			{
				var categoriesRequest CategoriesRequest
				err := json.Unmarshal([]byte(request.Body), &categoriesRequest)
				if err != nil {
					return badRequest(err)
				}

				err = validate.Struct(categoriesRequest)
				if err != nil {
					return badRequest(err)
				}

				categories, err := queryAllCategories(categoriesRequest)
				if err != nil {
					return errorResponse(err)
				}

				if categories == nil {
					return entityNotFound("no entity found")
				}

				jsonMarshalled, err := json.Marshal(categories)
				if err != nil {
					return errorResponse(err)
				}

				marshalledItem = jsonMarshalled
			}
		case "/api/menu/items":
			{
				var menuItemsRequest MenuItemsRequest
				err := json.Unmarshal([]byte(request.Body), &menuItemsRequest)
				if err != nil {
					return badRequest(err)
				}

				err = validate.Struct(menuItemsRequest)
				if err != nil {
					return badRequest(err)
				}

				menuItems, err := queryMenuItemsByCategory(menuItemsRequest)
				if err != nil {
					return errorResponse(err)
				}

				if menuItems == nil {
					return entityNotFound("no entity found")
				}

				jsonMarshalled, err := json.Marshal(menuItems)
				if err != nil {
					return errorResponse(err)
				}

				marshalledItem = jsonMarshalled
			}

		case "/api/menu/item":
			{
				var singleMenuItemRequest SingleMenuItemRequest
				err := json.Unmarshal([]byte(request.Body), &singleMenuItemRequest)
				if err != nil {
					return badRequest(err)
				}

				err = validate.Struct(singleMenuItemRequest)
				if err != nil {
					return badRequest(err)
				}

				menuItem, err := getSingleMenuItem(singleMenuItemRequest)
				if err != nil {
					return errorResponse(err)
				}

				if menuItem == nil {
					return entityNotFound("no entity found - invalid menu-item id")
				}

				jsonMarshalled, err := json.Marshal(menuItem)
				if err != nil {
					return errorResponse(err)
				}

				marshalledItem = jsonMarshalled
			}

		case "/api/branch/menu":
			{
				var branchAllMenuItemsRequest BranchAllMenuItemsRequest
				err := json.Unmarshal([]byte(request.Body), &branchAllMenuItemsRequest)
				if err != nil {
					return badRequest(err)
				}

				err = validate.Struct(branchAllMenuItemsRequest)
				if err != nil {
					return badRequest(err)
				}

				menuItems, err := queryAllMenuItems(branchAllMenuItemsRequest)
				if err != nil {
					return errorResponse(err)
				}

				if menuItems == nil {
					return entityNotFound("no entity found - invalid branch id")
				}

				jsonMarshalled, err := json.Marshal(menuItems)
				if err != nil {
					return errorResponse(err)
				}

				marshalledItem = jsonMarshalled
			}

		default:
			return defaultReturn()
		}
	} else if method == "GET" {
		return defaultReturn()
	} else {
		return defaultReturn()
	}

	return sendResponse(string(marshalledItem), 200), nil
}

//--------DYNAMODB QURIES---------------

func queryAllCategories(request CategoriesRequest) (*[]Category, error) {
	pk := request.CountryCode + "#CAT#BR#" + request.BranchId
	result, err := db.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(DYNAMO_TABLE_NAME),
		KeyConditionExpression: aws.String("pk = :hashKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":hashKey": &types.AttributeValueMemberS{Value: pk},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	categories := []Category{}
	err = attributevalue.UnmarshalListOfMaps(result.Items, &categories)
	if err != nil {
		return nil, err
	}

	return &categories, nil
}

// "US#MI#BR#01FHZXHK8PTP9FVK99Z66GXQTX"
func queryMenuItemsByCategory(request MenuItemsRequest) (*[]MenuItem, error) {
	pk := request.CountryCode + "#MI#BR#" + request.BranchId
	result, err := db.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(DYNAMO_TABLE_NAME),
		KeyConditionExpression: aws.String("pk = :hashKey and begins_with(sk, :rangeKey)"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":hashKey":  &types.AttributeValueMemberS{Value: pk},
			":rangeKey": &types.AttributeValueMemberS{Value: request.CategoryId + "#"},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	menuItems := []MenuItem{}
	err = attributevalue.UnmarshalListOfMaps(result.Items, &menuItems)
	if err != nil {
		return nil, err
	}

	return &menuItems, nil
}

func queryAllMenuItems(branchAllMenuItemsRequest BranchAllMenuItemsRequest) (*[]MenuItem, error) {
	result, err := db.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(DYNAMO_TABLE_NAME),
		KeyConditionExpression: aws.String("pk = :hashKey"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":hashKey": &types.AttributeValueMemberS{Value: branchAllMenuItemsRequest.BranchId},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	menuItems := []MenuItem{}
	err = attributevalue.UnmarshalListOfMaps(result.Items, &menuItems)
	if err != nil {
		return nil, err
	}

	return &menuItems, nil
}

func getSingleMenuItem(singleMenuItemRequest SingleMenuItemRequest) (*MenuItem, error) {
	result, err := db.GetItem(context.TODO(), &dynamodb.GetItemInput{
		TableName: aws.String(DYNAMO_TABLE_NAME),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: singleMenuItemRequest.PK},
			"sk": &types.AttributeValueMemberS{Value: singleMenuItemRequest.SK},
		},
	})

	if err != nil {
		return nil, err
	}

	if len(result.Item) == 0 {
		return nil, nil
	}

	item := MenuItem{}
	err = attributevalue.UnmarshalMap(result.Item, &item)
	if err != nil {
		return nil, err
	}

	return &item, nil
}

//---------Responses-------------

func sendResponse(body string, statusCode int) events.APIGatewayProxyResponse {
	return events.APIGatewayProxyResponse{
		Body:       body,
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		IsBase64Encoded: false,
	}
}

func defaultReturn() (events.APIGatewayProxyResponse, error) {
	resp := make(map[string]string)
	resp["error"] = "route not found"
	jsonResp, _ := json.Marshal(resp)
	return sendResponse(string(jsonResp), 404), nil
}

func errorResponse(e error) (events.APIGatewayProxyResponse, error) {
	resp := make(map[string]string)
	resp["error"] = e.Error()
	jsonResp, _ := json.Marshal(resp)
	return sendResponse(string(jsonResp), 500), nil
}

func unAuthorizedAccess() (events.APIGatewayProxyResponse, error) {
	resp := make(map[string]string)
	resp["error"] = "unauthorized access"
	jsonResp, _ := json.Marshal(resp)
	return sendResponse(string(jsonResp), 401), nil
}

func badRequest(e error) (events.APIGatewayProxyResponse, error) {
	resp := make(map[string]string)
	resp["error"] = e.Error()
	jsonResp, _ := json.Marshal(resp)
	return sendResponse(string(jsonResp), 400), nil
}

func entityNotFound(message string) (events.APIGatewayProxyResponse, error) {
	resp := make(map[string]string)
	resp["error"] = message
	jsonResp, _ := json.Marshal(resp)
	return sendResponse(string(jsonResp), 200), nil
}
