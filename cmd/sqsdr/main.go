package main

import (
	"fmt"
	"io/ioutil"
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
			Action:  redrive,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "source, s",
					Usage: "source queue name (required)",
				},
				cli.StringFlag{
					Name:  "destination, d",
					Usage: "destination queue name (required)",
				},
				cli.StringFlag{
					Name:  "regex, x",
					Usage: "only message bodies that match the regex will be sent to the destination queue (optional)",
				},
				cli.StringFlag{
					Name:  "jmespath, j",
					Usage: "JMESPath expression applied to the message body. output is passed to the regular expression (optional)",
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
			Action:  dump,
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
			Action:  send,
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

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "loquacious, l",
			Usage: "log loquaciously (read: verbosely, loudly, a lot) (default: false)",
		},
	}

	app.Before = setVerboseLogging

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func setVerboseLogging(c *cli.Context) error {
	if c.Bool("loquacious") {
		log.SetOutput(os.Stdout)
	} else {
		log.SetOutput(ioutil.Discard)
	}
	return nil
}
