package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/iamatypeofwalrus/sqsdr"
	cli "gopkg.in/urfave/cli.v1"
)

func redrive(c *cli.Context) error {
	src := c.String("source")
	if src == "" {
		return fmt.Errorf("the source flag must be present")
	}

	dest := c.String("destination")
	if dest == "" {
		return fmt.Errorf("the destination flag must be present")
	}

	// Optional
	jmesPath := c.String("jmespath")
	regex := c.String("regex")

	// Args with default values
	region := c.String("region")

	log.Println("command: redrive")
	log.Printf("\tsource: %v\n", src)
	log.Printf("\tdest: %v\n", dest)
	log.Printf("\tregion: %v\n", region)

	if jmesPath != "" {
		log.Printf("\tjmespath: %v\n", jmesPath)
	}

	if regex != "" {
		log.Printf("\tregex: %v\n", regex)
	}

	srcClient, srcURL, err := sqsdr.CreateClientAndValidateQueue(region, src)
	if err != nil {
		return err
	}

	destClient, destURL, err := sqsdr.CreateClientAndValidateQueue(region, dest)
	if err != nil {
		return err
	}

	r := &sqsdr.Redrive{
		SourceClient:   srcClient,
		SourceQueueURL: srcURL,

		DestClient:   destClient,
		DestQueueURL: destURL,

		JMESPath: jmesPath,
		Regex:    regex,
	}

	return r.Redrive()
}

func dump(c *cli.Context) error {
	src := c.String("source")
	if src == "" {
		return fmt.Errorf("the source flag must be present")
	}

	// Args with default values
	region := c.String("region")

	log.Println("command: dump")
	log.Printf("\tsource: %v\n", src)
	log.Printf("\tregion: %v\n", region)

	srcClient, srcURL, err := sqsdr.CreateClientAndValidateQueue(region, src)
	if err != nil {
		return err
	}

	d := sqsdr.Dump{
		SourceClient:   srcClient,
		SourceQueueURL: srcURL,
		Out:            os.Stdout,
	}

	return d.Dump()
}

func send(c *cli.Context) error {
	// read from STDIN or a file
	// need to take in a queue url
	// region?
	dest := c.String("destination")
	if dest == "" {
		return fmt.Errorf("the destination flag must be present")
	}

	region := c.String("region")

	log.Println("command: send")
	log.Printf("\tdestination: %v\n", dest)
	log.Printf("\tregion: %v\n", region)

	srcClient, srcURL, err := sqsdr.CreateClientAndValidateQueue(region, dest)
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		rawText := scanner.Text()

		input := &sqs.SendMessageInput{
			QueueUrl:    &srcURL,
			MessageBody: &rawText,
		}

		output, err := srcClient.SendMessage(input)
		if err != nil {
			log.Println("encountered error while sending message to SQS")
			return err
		}

		log.Printf("messageId: %v\n", output.MessageId)

	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
