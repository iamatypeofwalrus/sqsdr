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

	// Args with default values
	region := c.String("region")

	log.Println("command: redrive")
	log.Printf("\tsource: %v\n", src)
	log.Printf("\tdest: %v\n", dest)
	log.Printf("\tregion: %v\n", region)

	// not regex and jmespath? throw an error

	r, err := sqsdr.NewRedrive(
		src,
		region,
		dest,
		region,
	)
	if err != nil {
		return err
	}

	return r.Redrive()
}

func dump(c *cli.Context) error {
	return nil
}

func send(c *cli.Context) error {
	return nil
}
