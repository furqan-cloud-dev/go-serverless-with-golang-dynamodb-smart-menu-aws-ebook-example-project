package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		Body:       "{\"message\": \"hello world 14\"}",
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		IsBase64Encoded: false,
	}, nil
}

func main() {
	lambda.Start(handler)
}
