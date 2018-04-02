package main

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/iamatypeofwalrus/sqsdr"
)

func newRedrive(sourceQueueName string, sourceRegion string, destQueueName string, destRegion string, concurrency int) (*redrive, error) {
	srcSess := session.New(&aws.Config{Region: aws.String(sourceRegion)})
	srcClient := sqs.New(srcSess)
	srcQueueURL, err := getQueueURL(sourceQueueName, sourceRegion, srcClient)
	if err != nil {
		return nil, err
	}

	destSess := session.New(&aws.Config{Region: aws.String(destRegion)})
	destClient := sqs.New(destSess)
	destQueueURL, err := getQueueURL(destQueueName, destRegion, destClient)
	if err != nil {
		return nil, err
	}

	r := &redrive{
		sourceClient:   srcClient,
		sourceQueueURL: srcQueueURL,

		destClient:   destClient,
		destQueueURL: destQueueURL,

		concurrency: concurrency,
	}
	return r, nil
}

type redrive struct {
	sourceClient   *sqs.SQS
	sourceQueueURL string

	destClient   *sqs.SQS
	destQueueURL string

	concurrency int
}

func (r *redrive) redrive() error {
	p := sqsdr.NewPoller(
		r.sourceQueueURL,
		r.sourceClient,
		r,
	)

	return p.Process(context.Background())
}

func (r *redrive) Handle(ctx context.Context, msgs []*sqs.Message) ([]*sqs.Message, error) {
	processed := make([]*sqs.Message, 0, len(msgs))

	entries := make([]*sqs.SendMessageBatchRequestEntry, len(msgs))
	for i, msg := range msgs {
		entry := &sqs.SendMessageBatchRequestEntry{
			Id:                msg.MessageId,
			MessageAttributes: msg.MessageAttributes,
			MessageBody:       msg.Body,
		}
		entries[i] = entry
	}

	resp, err := r.destClient.SendMessageBatchWithContext(
		ctx,
		&sqs.SendMessageBatchInput{
			QueueUrl: aws.String(r.destQueueURL),
			Entries:  entries,
		},
	)
	if err != nil {
		return processed, err
	}

	if len(resp.Failed) > 0 {
		var errBuffer bytes.Buffer
		errBuffer.WriteString("The following error messages were received:\n\n")

		for _, f := range resp.Failed {
			errBuffer.WriteString(
				fmt.Sprintf("%v\n\n", *f.Message),
			)
		}

		return processed, fmt.Errorf("failed to batch send messages: %v", errBuffer.String())
	}

	if len(resp.Successful) < 1 {
		return processed, nil
	}

	// Create a map of MessageId -> SQS Message to construct the final array of
	// processed messages
	idToMessage := make(map[string]*sqs.Message)
	for _, msg := range msgs {
		idToMessage[*msg.MessageId] = msg
	}

	for _, s := range resp.Successful {
		msg := idToMessage[*s.Id]
		processed = append(processed, msg)
	}

	return processed, nil
}
