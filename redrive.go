package sqsdr

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

// NewRedrive returns an initialized Redrive struct
func NewRedrive(sourceQueueName string, sourceRegion string, destQueueName string, destRegion string) (*Redrive, error) {
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

	r := &Redrive{
		sourceClient:   srcClient,
		sourceQueueURL: srcQueueURL,

		destClient:   destClient,
		destQueueURL: destQueueURL,
	}
	return r, nil
}

// Redrive is a simple strategy that moves messages from a source queue to a destination queue.
type Redrive struct {
	sourceClient   *sqs.SQS
	sourceQueueURL string

	destClient   *sqs.SQS
	destQueueURL string

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
	sink := &SQSSink{QueueURL: r.destQueueURL, Client: r.destClient}

	pipeline := &Pipeline{
		Chooser:   &PassthroughChooser{},
		LeftSink:  sink,
		RightSink: NullSink{},
	}

	poller := NewPoller(r.sourceQueueURL, r.sourceClient, pipeline)

	return poller.Process(context.Background())
}

func (r *Redrive) filteredRedrive() error {
	// Create a new SQS queue based off of the destination queue name
	// Any message that doesn't make pass the filter will end up in this queue.
	// At the end of the function we'll put messages in this queue back into the
	// source queue, and then delete this queue.
	filterQueueURL, err := createFilterSink(r.destClient, r.destQueueURL)
	if err != nil {
		return err
	}

	// Prep for the source -> destination pipeline
	leftSink := &SQSSink{QueueURL: r.destQueueURL, Client: r.destClient}
	rightSink := &SQSSink{QueueURL: filterQueueURL, Client: r.destClient}

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
	log.Println("passing messages in source queue through filter")
	poller := NewPoller(r.sourceQueueURL, r.sourceClient, pipeline)
	err = poller.Process(context.Background())
	if err != nil {
		return err
	}

	// Now we have a whole bunch of messages in the right sink and we need to put
	// them back in the source
	log.Println("redriving messages that ended up in the right sink back to the source")
	passthrough := &PassthroughChooser{}
	sourceSink := &SQSSink{
		QueueURL: r.sourceQueueURL,
		Client:   r.sourceClient,
	}

	reversePipeline := &Pipeline{
		Chooser:   passthrough,
		LeftSink:  sourceSink,
		RightSink: &NullSink{},
	}

	rightPoller := NewPoller(filterQueueURL, r.destClient, reversePipeline)
	err = rightPoller.Process(context.Background())
	if err != nil {
		return err
	}

	// Huzzah! Let's remove the queue that we created at the top of the function
	return deleteFilterSink(r.destClient, filterQueueURL)
}

func createFilterSink(client sqsiface.SQSAPI, queueURL string) (string, error) {
	// this is _super_ gross. ¯\_(ツ)_/¯
	split := strings.Split(queueURL, "/")
	name := split[len(split)-1]
	log.Println("creating queue with name:", name)
	req := &sqs.CreateQueueInput{
		QueueName: aws.String(fmt.Sprintf("sqsdr-%v", name)),
	}
	// TODO: we don't care if this queue already exists that means something failed
	// and the user is running the command again.
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
