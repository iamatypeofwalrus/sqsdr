package sqsdr

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/sqs"
)

// Pipeline is a simple struct to manage the interaction between a source, a chooser, and sinks.
// Pipeline satisfies the Handle interface
type Pipeline struct {
	Chooser   Chooser
	LeftSink  Sinker
	RightSink Sinker
}

// Handle is the entry point into the pipeline
func (p *Pipeline) Handle(msgs []*sqs.Message) ([]*sqs.Message, error) {
	leftMsgs, rightMsgs := p.Chooser.Choose(msgs)

	var leftError error
	if len(leftMsgs) > 0 {
		leftError = p.LeftSink.Sink(leftMsgs)
	}

	var rightError error
	if len(rightMsgs) > 0 {
		rightError = p.RightSink.Sink(rightMsgs)
	}

	var err error
	if rightError != nil && leftError != nil {
		err = fmt.Errorf("rightSink error: %v\nleftSink error: %v", rightError, leftError)
	} else if rightError != nil {
		err = rightError
	} else if leftError != nil {
		err = leftError
	}

	return msgs, err
}

// Sinker is an interface that captures any type that accepts an array of SQS messages and
// puts them somewhere. Where somewhere could be into another SQS queue or to disk.
type Sinker interface {
	Sink([]*sqs.Message) error
}

// Chooser is an interface for any struct that given an array of SQS messages will choose
// if the SQS messages go to the left sink, the right sink, or both.
type Chooser interface {
	Choose([]*sqs.Message) ([]*sqs.Message, []*sqs.Message)
}
