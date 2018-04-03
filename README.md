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
Imagine you have a DLQ with messages that look like this with some variations:

```
{
  "user_id": "2",
  "book_id": 1,
  "review": {
    "text": "noice",
    "lang": "en-US"
  }
}
```
and you have the following queues:

```
Source Queue Name: my-queue-dlq
Destination Queue Name: my-queue
```

If you want to redrive all the messages from the dead letter queue (DLQ) to the main queue you can run
this command:

```
sqsdr redrive \
  --source my-queue-dlq \
  --destination my-queue \
  --region us-west-2
```

If you want to redrive all of the messages that have reviews in English you can use the following [JMESPath](http://jmespath.org/) `reviews.lang` to drive down into the `"lang"` attribute and then use the regex flag
to specify that we want languages that match the pattern `"en-US"`.

```
sqsdr redrive \
  --source my-queue-dlq \
  --destination my-queue \
  --region us-west-2 \
  --jmespath "review.lang" \
  --regex "en-US" 
```

Messages that pass the JMESPath and the Regex will be sent to the destination queue. All others are sent to a fallthrough queue that `redrive` creates at run time.

Once every messages is either in the destination queue or the fallthrough queue the `redrive` command will put the messages that are in the fallthrough queue back into the source queue. Once that is completed `redrive` will remove the fallthrough queue.

## Dump Messages to Disk
## Send Messages from STDIN to a Queue
