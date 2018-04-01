package sqsdr

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

const (
	maxEmptyReceives          = 2
	waitTimeSeconds     int64 = 20
	maxNumberofMessages int64 = 10
)

type sqsClient interface {
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessageBatch(*sqs.DeleteMessageBatchInput) (*sqs.DeleteMessageBatchOutput, error)
}

// NewPoller returns a Poller that defaults to long polling and receiving at most 10 messages at a time
func NewPoller(queueURL string, client sqsiface.SQSAPI, handler Handler) *Poller {
	return &Poller{
		QueueURL:            queueURL,
		Client:              client,
		Handler:             handler,
		MaxEmptyReceives:    maxEmptyReceives,
		WaitTimeSeconds:     waitTimeSeconds,
		MaxNumberOfMessages: maxNumberofMessages,
	}
}

// Poller manages the business logic of polling a queue for messages, handing them off to a Handler, and deleting the succesfully
// processed messages from the queue.
type Poller struct {
	QueueURL         string
	Handler          Handler
	Client           sqsiface.SQSAPI
	MaxEmptyReceives int

	// SQS ReceiveMessage API pass through
	WaitTimeSeconds     int64
	MaxNumberOfMessages int64
}

// Process is the entry point for the Poller. It is a blocking function. If you desire more concurrency call Process() in a separate
// go routine as many times as needed.
func (p *Poller) Process(ctx context.Context) error {
	numEmptyReceives := 0
	for {
		numProcessed, err := p.processOnce(ctx)
		if err != nil {
			return err
		}

		if numProcessed == 0 {
			numEmptyReceives++
		}

		if numEmptyReceives >= p.MaxEmptyReceives {
			return nil
		}
	}
}

// ProcessOnce polls, handles, and deletes successfully processed messages from the queue one time.
func (p *Poller) ProcessOnce(ctx context.Context) (int, error) {
	numProcessed := 0
	msgs, err := p.receiveMessages(ctx)
	if err != nil {
		return numProcessed, err
	}

	processed, err := p.Handler.Handle(msgs)
	numProcessed = len(processed)
	if err != nil {
		return numProcessed, err
	}

	err = p.deleteMessages(ctx, processed)
	return numProcessed, nil
}

func (p *Poller) receiveMessages(ctx context.Context) ([]*sqs.Message, error) {
	req := &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(p.QueueURL),
		WaitTimeSeconds:     aws.Int64(int64(p.WaitTimeSeconds)),
		AttributeNames:      []*string{aws.String("ALL")},
		MaxNumberOfMessages: aws.Int64(p.MaxNumberOfMessages),
	}

	resp, err := p.Client.ReceiveMessageWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	return resp.Messages, nil
}

func (p *Poller) deleteMessages(ctx context.Context, msgs []*sqs.Message) error {
	entries := make([]*sqs.DeleteMessageBatchRequestEntry, len(msgs))

	for i, msg := range msgs {
		entries[i] = &sqs.DeleteMessageBatchRequestEntry{
			Id:            msg.MessageId,
			ReceiptHandle: msg.ReceiptHandle,
		}
	}

	failed, err := p.deleteEntries(ctx, entries)
	if err != nil {
		return err
	}

	if senderFault(failed) {
		return compileFailedErrors("failed to batch delete messages", failed)
	}

	if len(failed) > 0 {
		err = p.deleteFailedEntries(ctx, entries, failed, 2)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Poller) deleteEntries(ctx context.Context, entries []*sqs.DeleteMessageBatchRequestEntry) ([]*sqs.BatchResultErrorEntry, error) {
	req := &sqs.DeleteMessageBatchInput{
		QueueUrl: aws.String(p.QueueURL),
		Entries:  entries,
	}

	resp, err := p.Client.DeleteMessageBatchWithContext(ctx, req)
	return resp.Failed, err
}

// deleteFailedEntries handles retrying messages that failed to send. it is assumed that the messages failed due to availabilty
// errors with SQS.
func (p *Poller) deleteFailedEntries(ctx context.Context, entries []*sqs.DeleteMessageBatchRequestEntry, failed []*sqs.BatchResultErrorEntry, maxRetries int) error {
	// BatchResultErrorEntry only contains the Id of the request and not the ReceiptHandle. We need both to make a delete request.
	// Let's create a map from the original entry Id to ReceiptHandle
	entriesMap := make(map[string]string)
	for _, entry := range entries {
		entriesMap[*entry.Id] = *entry.ReceiptHandle
	}

	retries := 0
	for retries < maxRetries {
		// Create new batch entries using Id in the ErrorEntry and the corresponding ReceiptHandle from the map we generated above
		entries := make([]*sqs.DeleteMessageBatchRequestEntry, len(failed))
		for i, fail := range failed {
			entry := &sqs.DeleteMessageBatchRequestEntry{
				Id:            fail.Id,
				ReceiptHandle: aws.String(entriesMap[*fail.Id]),
			}

			entries[i] = entry
		}

		failed, err := p.deleteEntries(ctx, entries)
		if err != nil {
			return err
		}

		// Success!
		if len(failed) == 0 {
			return nil
		}

		// Looks like we still have some more messages that failed to delete. let's loop around again
		retries++
	}

	// This may or may not be the same slice that was passed in. return a useful error message to the caller
	// so they can debug.
	return compileFailedErrors(
		fmt.Sprintf("unable to delete sqs messages with %v retries", maxRetries),
		failed,
	)
}

// compileFailedErrors builds an error from all of the separate errors in the failed array
func compileFailedErrors(msg string, failed []*sqs.BatchResultErrorEntry) error {
	messages := make([]string, len(failed))

	for i, entry := range failed {
		var sqsMsg string
		if entry.Message != nil {
			sqsMsg = *entry.Message
		} else {
			sqsMsg = "No Error Message"
		}

		errMsg := fmt.Sprintf("%v: %v", *entry.Id, sqsMsg)

		messages[i] = errMsg
	}

	return fmt.Errorf("%v: %v", msg, strings.Join(messages, "\n============\n"))
}

// senderFault for better or worse assumes if any entry is a sender fault it's likely that the entire batch
// failed due to a sender error
func senderFault(failed []*sqs.BatchResultErrorEntry) bool {
	for _, entry := range failed {
		if *entry.SenderFault {
			return true
		}
	}
	return false
}
