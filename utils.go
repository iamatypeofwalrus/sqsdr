package sqsdr

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type queueURLer interface {
	GetQueueUrl(*sqs.GetQueueUrlInput) (*sqs.GetQueueUrlOutput, error)
}

func getQueueURL(queueName string, region string, client queueURLer) (string, error) {
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

// CreateClientAndValidateQueue takes in an AWS region and a Queue name and returns
// an intialized SQS client, the Queue URL for a given and an error if one exists.
func CreateClientAndValidateQueue(region, queueName string) (*sqs.SQS, string, error) {
	sess := session.New(&aws.Config{Region: aws.String(region)})
	client := sqs.New(sess)
	queueURL, err := getQueueURL(queueName, region, client)
	if err != nil {
		return nil, "", err
	}

	return client, queueURL, nil
}
