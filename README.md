# sqsdr
Simple Queue Service (SQS) doctor wants to be your one stop shop for triaging and redriving messages in
your SQS queues.

## CLI Usage
```
$ sqsdr --help
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
     help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --loquacious, -l  log loquaciously (read: verbosely, loudly, a lot) (default: false)
   --help, -h        show help
   --version, -v     print the version
```

## Redrive
`redrive` is a generic command for moving messages from one queue to another. It also exposes filtering
functionality with the `--regex` and `--jmespath` flags allowing you to send a subset of the messages
in your source queue to the destination queue.

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
Imagine you have the following queues:

```
Source Queue Name: my-queue-dlq
Destination Queue Name: my-queue
```

With messages that look like this with some variations in `my-queue-dlq`:

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

If you want to redrive all the messages from `my-queue-dlq` to `my-queue` you can run
this command:

```
sqsdr redrive \
  --source my-queue-dlq \
  --destination my-queue \
  --region us-west-2
```

If you want to redrive all of the messages that have reviews in English you can use the following [JMESPath(http://jmespath.org/) expression `reviews.lang` with the `--jmespath` flag to drive down into the `"lang"` attribute in concert with the regex flag `--regex` to specify that we want languages that match the pattern `"en-US"`.

```
sqsdr redrive \
  --source my-queue-dlq \
  --destination my-queue \
  --region us-west-2 \
  --jmespath "review.lang" \
  --regex "en-US" 
```

Messages that pass the JMESPath and the Regex will be sent to the destination queue.

## Dump Messages to Disk
### Help
```
$ sqsdr dump --help
NAME:
   sqsdr dump - dump messages from a source queue to STDOUT

USAGE:
   sqsdr dump [command options] [arguments...]

OPTIONS:
   --source value, -s value  source queue name
   --region value, -r value  AWS region of the queues region (default: "us-east-1")
```
