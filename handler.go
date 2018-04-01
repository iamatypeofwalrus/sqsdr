package sqsdr

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

// Handler represents any type that can process SQS messages. Messages returned by the handler will be removed from
// the source queue if there is one. In other words the handler need not worry about the lifecyle of the SQS messages.
type Handler interface {
	Handle([]*sqs.Message) ([]*sqs.Message, error)
}
