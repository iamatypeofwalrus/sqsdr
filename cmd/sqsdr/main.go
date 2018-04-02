package main

import (
	"fmt"
	"log"
	"os"

	cli "gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "sqsdr - AWS Simple Queue Service (SQS) Doctor"
	app.Usage = ""
	app.Description = "SQS tool for helping you investigate, debug, and redrive SQS messages"
	app.Version = "0.1.0"

	// construct client, validate
	app.Commands = []cli.Command{
		{
			Name:    "redrive",
			Aliases: []string{"r"},
			Usage:   "redrive messages from source queue to a destination queue",
			Action: func(c *cli.Context) error {
				fmt.Println(c.Args())
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "source, s",
					Usage: "source queue name",
				},
				cli.StringFlag{
					Name:  "destination, d",
					Usage: "destination queue name",
				},
				cli.IntFlag{
					Name:  "concurrency, c",
					Usage: "number of concurrent workers to use",
					Value: 10,
				},
				cli.StringFlag{
					Name:  "region, r",
					Usage: "AWS region of the queues region",
					Value: "us-east-1",
				},
			},
		},
		{
			Name:    "dump",
			Aliases: []string{"d"},
			Usage:   "dump messages from a source queue to disk",
			Action: func(c *cli.Context) error {
				fmt.Println(c.Args())
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "source, s",
					Usage: "source queue name",
				},
				cli.StringFlag{
					Name:  "region, r",
					Usage: "AWS region of the queues region",
					Value: "us-east-1",
				},
			},
		},
		{
			Name:    "send",
			Aliases: []string{"s"},
			Usage:   "send JSON messages piped through STDIN to a destination queue",
			Action: func(c *cli.Context) error {
				fmt.Println(c.Args())
				return nil
			},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "destination, d",
					Usage: "destination queue name",
				},
				cli.StringFlag{
					Name:  "region, r",
					Usage: "AWS region of the queues region",
					Value: "us-east-1",
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
