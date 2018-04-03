package sqsdr

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
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
	// Create a new SQS queue based off of the destination queue name
	// Any message that doesn't make pass the filter will end up in this queue.
	// At the end of the function we'll put messages in this queue back into the
	// source queue, and then delete this queue.
	filterQueueURL, err := createFilterSink(r.SourceClient, r.SourceQueueURL)
	if err != nil {
		return err
	}

	log.Println("created temporary fallthrough queue:", filterQueueURL)

	// Prep for the source -> destination pipeline
	leftSink := &SQSSink{QueueURL: r.DestQueueURL, Client: r.DestClient}
	rightSink := &SQSSink{QueueURL: filterQueueURL, Client: r.SourceClient}

	chooser, err := NewFilterChooser(r.JMESPath, r.Regex)
	if err != nil {
		return err
	}

	pipeline := &Pipeline{
		Chooser:   chooser,
		LeftSink:  leftSink,
		RightSink: rightSink,
	}

	// Run filter over all messages in the DLQ. If messages pass the filter successfully
	// they will end up in the left sink else in the right sink (which is the SQS queue
	// we just created)
	log.Println("passing messages from source queue through filter to destination queue")
	poller := NewPoller(r.SourceQueueURL, r.SourceClient, pipeline)
	err = poller.Process(context.Background())
	if err != nil {
		return err
	}

	// Now we have a whole bunch of messages in the right sink and we need to put
	// them back in the source
	log.Println("redriving messages that ended up in the temporary fallthrough queue back to the source")
	passthrough := &PassthroughChooser{}
	sourceSink := &SQSSink{
		QueueURL: r.SourceQueueURL,
		Client:   r.SourceClient,
	}

	reversePipeline := &Pipeline{
		Chooser:   passthrough,
		LeftSink:  sourceSink,
		RightSink: &NoOpSink{},
	}

	rightPoller := NewPoller(filterQueueURL, r.SourceClient, reversePipeline)
	err = rightPoller.Process(context.Background())
	if err != nil {
		return err
	}

	// Huzzah! Let's remove the queue that we created at the top of the function
	log.Println("removing temporary fallthrough queue", filterQueueURL)
	return deleteFilterSink(r.DestClient, filterQueueURL)
}

func createFilterSink(client sqsiface.SQSAPI, queueURL string) (string, error) {
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

func deleteFilterSink(client sqsiface.SQSAPI, queueURL string) error {
	req := &sqs.DeleteQueueInput{QueueUrl: aws.String(queueURL)}
	_, err := client.DeleteQueue(req)
	if err != nil {
		return fmt.Errorf("could not delete queue created for filter sink: %v", err)
	}

	return nil
}
