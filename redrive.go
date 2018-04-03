package sqsdr

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// Redrive is a simple strategy that moves messages from a source queue to a destination queue.
type Redrive struct {
	SourceClient   *sqs.SQS
	SourceQueueURL string

	DestClient   *sqs.SQS
	DestQueueURL string

	JMESPath string
	Regex    string

	// Not implemented yet
	concurrency int
}

// Redrive is the entry point into the redriving strategy
func (r *Redrive) Redrive() error {
	if r.Regex != "" {
		return r.filteredRedrive()
	}

	return r.simpleRedrive()
}

func (r *Redrive) simpleRedrive() error {
	log.Println("starting simple redrive")
	sink := &SQSSink{QueueURL: r.DestQueueURL, Client: r.DestClient}

	pipeline := &Pipeline{
		Chooser:   &PassthroughChooser{},
		LeftSink:  sink,
		RightSink: NoOpSink{},
	}

	poller := NewPoller(r.SourceQueueURL, r.SourceClient, pipeline)

	return poller.Process(context.Background())
}

func (r *Redrive) filteredRedrive() error {
	chooser, err := NewFilterChooser(r.JMESPath, r.Regex)
	if err != nil {
		return err
	}

	leftSink := &SQSSink{QueueURL: r.DestQueueURL, Client: r.DestClient}
	rightSinkFunc := func(queueURL string, client sqsiface.SQSAPI) Sinker {
		return &SQSSink{QueueURL: queueURL, Client: client}
	}

	f := FallthroughPipeline{
		Chooser:        chooser,
		LeftSink:       leftSink,
		RightSinkFunc:  rightSinkFunc,
		SourceClient:   r.SourceClient,
		SourceQueueURL: r.SourceQueueURL,
	}

	return f.Run()
}
