package sqsdr

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// FallthroughPipeline is a higher level pipeline that manages creating fallthrough
// queues, running the forward pipeline, redriving the messages in the fallthrough
// queues back into the source queues, and removing the fallthrough queue.
//
// It's written to be as general as possible so you may want to check out filtered redrive
// or Dump for examples on how it's used
type FallthroughPipeline struct {
	Chooser       Chooser
	LeftSink      Sinker
	RightSinkFunc func(queueURL string, client sqsiface.SQSAPI) Sinker

	SourceClient   sqsiface.SQSAPI
	SourceQueueURL string
}

// Run is the entrypoint for running the FilterRunner
func (f *FallthroughPipeline) Run() error {
	// Any message that doesn't make it past the filter (i.e. ends up in the right sink)
	// will end up in this queue. At the end of the function we'll put messages
	// in this queue back into the source queue, and then delete this queue.
	fallthroughQueueURL, err := createFallthroughQueue(
		f.SourceClient,
		f.SourceQueueURL,
	)
	if err != nil {
		return err
	}

	rightSink := f.RightSinkFunc(fallthroughQueueURL, f.SourceClient)
	pipeline := &Pipeline{
		Chooser:   f.Chooser,
		LeftSink:  f.LeftSink,
		RightSink: rightSink,
	}

	// Run filter over all messages in the source queue. If messages pass the filter successfully
	// they will end up in the left sink else in the right sink (which is the SQS queue
	// we just created)
	log.Println("passing messages from source queue through filter")
	poller := NewPoller(f.SourceQueueURL, f.SourceClient, pipeline)
	err = poller.Process(context.Background())
	if err != nil {
		return err
	}

	// Now we have a whole bunch of messages in the right sink and we need to put
	// them back in the source
	log.Println("redriving messages that ended up in the temporary fallthrough queue back to the source")
	passthrough := &PassthroughChooser{}
	sourceSink := &SQSSink{
		QueueURL: f.SourceQueueURL,
		Client:   f.SourceClient,
	}

	reversePipeline := &Pipeline{
		Chooser:   passthrough,
		LeftSink:  sourceSink,
		RightSink: &NoOpSink{},
	}

	rightPoller := NewPoller(fallthroughQueueURL, f.SourceClient, reversePipeline)
	err = rightPoller.Process(context.Background())
	if err != nil {
		return err
	}

	// Huzzah! Let's remove the queue that we created at the top of the function
	log.Println("removing temporary fallthrough queue", fallthroughQueueURL)
	return deleteFallthroughQueue(f.SourceClient, fallthroughQueueURL)
}

func createFallthroughQueue(client sqsiface.SQSAPI, queueURL string) (string, error) {
	// this is _super_ gross. ¯\_(ツ)_/¯
	split := strings.Split(queueURL, "/")
	queueName := split[len(split)-1]
	name := aws.String(fmt.Sprintf("sqsdr-%v-fallthrough", queueName))

	req := &sqs.CreateQueueInput{QueueName: name}
	// NOTE: if the queue already exists SQS is more than happy to return a successful
	//       response. Nice!
	resp, err := client.CreateQueue(req)
	if err != nil {
		return "", fmt.Errorf("could not create queue for filter sink: %v", err)
	}

	return *resp.QueueUrl, nil
}

func deleteFallthroughQueue(client sqsiface.SQSAPI, queueURL string) error {
	req := &sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)}
	_, err := client.DeleteQueue(req)
	if err != nil {
		return fmt.Errorf("could not delete queue created for filter sink: %v", err)
	}

	return nil
}
