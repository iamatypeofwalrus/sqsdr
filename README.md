# sqsdr
## CLI Usage
``` sqsdr --help
NAME:
   sqsdr - AWS Simple Queue Service (SQS) Doctor

USAGE:
   sqsdr [global options] command [command options] [arguments...]

VERSION:
   0.1.0

DESCRIPTION:
   SQS tool for helping you investigate, debug, and redrive SQS messages

COMMANDS:
     redrive, r  redrive messages from source queue to a destination queue
     dump, d     dump messages from a source queue to disk
     send, s     send JSON messages piped through STDIN to a destination queue
     help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loquacious, -l  log loquaciously (read: verbosely, loudly, a lot) (default: false)
   --help, -h        show help
   --version, -v     print the version
```

## Redrive
### Help
```
$ sqsdr redrive --help
NAME:
   sqsdr redrive - redrive messages from source queue to a destination queue

USAGE:
   sqsdr redrive [command options] [arguments...]

OPTIONS:
   --source value, -s value       source queue name (required)
   --destination value, -d value  destination queue name (required)
   --regex value, -x value        only message bodies that match the regex will be sent to the destination queue (optional)
   --jmespath value, -j value     JMESPath expression applied to the message body. output is passed to the regular expression (optional)
   --region value, -r value       AWS region of the queues region (default: "us-east-1")
```
### Example
Imagine you have a DLQ with messages that look like this. Where "SOME_TEXT" is some text that is
different from messages to message.

```
{"a": {"b": {"c": "SOME_TEXT"}}}
```

If you want to redrive all the messages from the Dead Letter Queue to the main queue you can run
this command

```
sqsdr -loquacious redrive --source rdrv-int-test-dlq -destination rdrv-int-test --region us-west-2
```

What if you want to redrive a subset of those messages where "SOME_TEXT" is 42.

We'll use a [JMESPath](http://jmespath.org/) to 

```
sqsdr --loquacious redrive --source rdrv-int-test-dlq -destination rdrv-int-test --region us-west-2 --jmespath "a.b.c." --regex "42"
```

Messages that pass the JMESPath and the Regex will be sent to the destination queue. All others are sent to a fallthrough queue. Once every messages is either in the destination queue or the fallthrough queue the `redrive` command will put the messages in the fallthrough queue back in the source queue.

## Dump Messages to Disk
## Send Messages from STDIN to a Queue
