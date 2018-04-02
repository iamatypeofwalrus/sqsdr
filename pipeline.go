package sqsdr

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/sqs"
)

// Pipeline is a simple struct to manage the interaction between a source, a chooser, and sinks.
// Pipeline satisfies the Handler interface and can be passed to a Poller.
type Pipeline struct {
	Chooser   Chooser
	LeftSink  Sinker
	RightSink Sinker
}

// Handle is the entry point into the pipeline
func (p *Pipeline) Handle(ctx context.Context, msgs []*sqs.Message) ([]*sqs.Message, error) {
	leftMsgs, rightMsgs := p.Chooser.Choose(msgs)

	var leftError error
	if len(leftMsgs) > 0 {
		leftError = p.LeftSink.Sink(ctx, leftMsgs)
	}

	var rightError error
	if len(rightMsgs) > 0 {
		rightError = p.RightSink.Sink(ctx, rightMsgs)
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
