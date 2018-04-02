package sqsdr

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// Sinker is an interface that accepts an array of SQS messages and puts them
// somewhere. Where "somewhere" could be into another SQS queue, a file on disk,
// or whatever.
type Sinker interface {
	Sink(context.Context, []*sqs.Message) error
}

// NullSink drops the messages on the floor. Use it only as a signal to other developers
// that your other sink is doing all of the work.
type NullSink struct{}

// Sink does nothing with the messages and returns a nil error
func (n NullSink) Sink(ctx context.Context, msgs []*sqs.Message) error {
	return nil
}

// SQSSink pass all messages to the QueueURL with the provided SQS Client
type SQSSink struct {
	QueueURL string
	Client   sqsiface.SQSAPI
}

// Sink performs a BatchSend with the passed in messages
func (s *SQSSink) Sink(ctx context.Context, msgs []*sqs.Message) error {
	entries := make([]*sqs.SendMessageBatchRequestEntry, len(msgs))
	for i, msg := range msgs {
		entry := &sqs.SendMessageBatchRequestEntry{
			Id:                msg.MessageId,
			MessageAttributes: msg.MessageAttributes,
			MessageBody:       msg.Body,
		}
		entries[i] = entry
	}

	resp, err := s.Client.SendMessageBatchWithContext(
		ctx,
		&sqs.SendMessageBatchInput{
			QueueUrl: aws.String(s.QueueURL),
			Entries:  entries,
		},
	)
	if err != nil {
		return err
	}

	if len(resp.Failed) > 0 {
		var errBuffer bytes.Buffer
		errBuffer.WriteString("The following error messages were received:\n\n")

		for _, f := range resp.Failed {
			errBuffer.WriteString(
				fmt.Sprintf("%v\n\n", *f.Message),
			)
		}

		return fmt.Errorf("failed to batch send messages: %v", errBuffer.String())
	}

	return nil
}
