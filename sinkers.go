package sqsdr

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"

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

// NoOpSink drops the messages on the floor. Use it only as a signal to other developers
// that your other sink is doing all of the work.
type NoOpSink struct{}

// Sink does nothing with the messages and returns a nil error
func (n NoOpSink) Sink(ctx context.Context, msgs []*sqs.Message) error {
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

// MessageOutput is a simplified version of the SQS Message that's appropriate to write to disk or
// STDOUT.
//
// TODO: will have to handle the case when putting messages from STDIN to queue
//       where the message body is JSON.
type MessageOutput struct {
	Body              *string                               `json:",omitempty"`
	MessageAttributes map[string]*sqs.MessageAttributeValue `json:",omitempty"`
	MessageId         *string
	ReceiptHandle     *string
}

// WriterSink will write SQS Message in the MessageOutput format to the Writer with the delimiter
// as a separator
type WriterSink struct {
	Writer      io.Writer
	Passthrough Sinker
}

// Sink writes converts the SQS Message to a MessageOutput and writes the message
// the Writer.
func (w *WriterSink) Sink(ctx context.Context, msgs []*sqs.Message) error {
	errors := make([]error, 0)
	encoder := json.NewEncoder(w.Writer)

	for _, msg := range msgs {
		msgOut := MessageOutput{
			Body:              msg.Body,
			MessageAttributes: msg.MessageAttributes,
			MessageId:         msg.MessageId,
			ReceiptHandle:     msg.ReceiptHandle,
		}

		err := encoder.Encode(msgOut)
		if err != nil {
			log.Println("an error occurred while dumping SQS message to JSON:", err)
			errors = append(errors, err)
			continue
		}
	}

	if len(errors) > 0 {
		var buffer bytes.Buffer
		for _, e := range errors {
			buffer.WriteString(e.Error())
			buffer.WriteString("\n===\n")
		}

		return fmt.Errorf("the following errors occurred while dumping SQS messages: %v", buffer.String())
	}

	return w.Passthrough.Sink(ctx, msgs)
}
