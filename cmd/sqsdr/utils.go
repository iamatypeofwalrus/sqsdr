package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

func getQueueURL(queueName string, region string, client sqsiface.SQSAPI) (string, error) {
	output, err := client.GetQueueUrl(
		&sqs.GetQueueUrlInput{
			QueueName: aws.String(queueName),
		},
	)
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == sqs.ErrCodeQueueDoesNotExist {
			err = fmt.Errorf("could not find source queue with name '%v' in %v", queueName, region)
		}

		return "", err
	}

	return *output.QueueUrl, nil
}
