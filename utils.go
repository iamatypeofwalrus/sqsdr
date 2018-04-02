package sqsdr

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
