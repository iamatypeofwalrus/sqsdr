package main

import (
	"fmt"
	"log"

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
	return nil
}

func send(c *cli.Context) error {
	return nil
}
