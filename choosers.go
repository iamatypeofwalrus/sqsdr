package sqsdr

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"

	"github.com/jmespath/go-jmespath"

	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	emptyMessages = make([]*sqs.Message, 0, 0)
)

// Chooser is an interface that given an array of SQS messages will choose
// if the SQS messages go to the left sink, the right sink, or both. It is assumed
// all messages that are passed in are passed back out.
//
// By convention the right sink is always the fallthrough sink. Pass any message that
// don't meet your criteria to the
type Chooser interface {
	Choose([]*sqs.Message) ([]*sqs.Message, []*sqs.Message)
}

// PassthroughChooser always passes messages to the left
type PassthroughChooser struct{}

// Choose always passes messages through to the left
func (p *PassthroughChooser) Choose(msgs []*sqs.Message) (left []*sqs.Message, right []*sqs.Message) {
	left = msgs
	right = emptyMessages
	return
}

// NewFilterChooser returns an initialized FilterChooser if the passed in regular expression
// can be compiled. It returns an error otherwise.
func NewFilterChooser(jmespath string, regex string) (*FilterChooser, error) {
	r, err := regexp.Compile(regex)
	if err != nil {
		return nil, fmt.Errorf("could not compile regular expression in NewFilterChooser: %v", err)
	}

	f := &FilterChooser{
		JMESPath: jmespath,
		Regex:    r,
	}
	return f, nil
}

// FilterChooser passes the body of a SQS message through a JMESPath, if it is present,
// and through a Regular Expression. Messages that satisfy the regular expression go to
// the left sink and all other go to the right sink.
type FilterChooser struct {
	JMESPath string
	Regex    *regexp.Regexp
}

// Choose will pass the body of the SQS Messages through the JMESPath, if it is
// present, and then through the regular expression. If a message body fails anywhere
// in the path an error message is logged and the SQS message is put in the right sink.
//
// If a JMESPath is present and is successfully run against the SQS Message body
// the JMESPath output replaces the SQS message body in the filter. This allows you to
// run a JMESPath against a large object to filter it down and then run a regular expression
// against the output to choose between the left sink or the right sink.
//
// NOTE: the JMESPath expression must return a valid JSON object. If the output
// is not valid the SQS message will be put in the right sink.
func (f *FilterChooser) Choose(msgs []*sqs.Message) ([]*sqs.Message, []*sqs.Message) {
	left := make([]*sqs.Message, 0, len(msgs))
	right := make([]*sqs.Message, 0, len(msgs))

	for _, msg := range msgs {
		body := msg.Body
		if body == nil {
			continue
		}
		strBody := *body

		// If a JMESPath is provided will make a best effort to run it against the
		// SQS Body. If an error has occured will throw it in the right sink and continue.
		//
		// In the happy path we'll convert whatever JMESPath returned into a string
		// and let the Regex filter have at it.
		if f.JMESPath != "" {
			var jsonBody interface{}
			jsonErr := json.Unmarshal([]byte(strBody), &jsonBody)
			// TODO: handle non JSON bodies and fail quietly
			if jsonErr != nil {
				log.Printf("could not marshal sqs body into json: %v\n", jsonErr)
				log.Printf("SQS Body: %v\n", strBody)

				// Wasn't JSON? put in the right sink
				right = append(right, msg)
				continue
			}

			out, jmesErr := jmespath.Search(f.JMESPath, jsonBody)
			if jmesErr != nil {
				log.Printf("jmespath threw an error: %v\n", jmesErr)

				right = append(right, msg)
				continue
			}

			b, err := json.Marshal(out)
			if err != nil {
				log.Println("could not convert JMES Path output to JSON:", err)
				right = append(right, msg)
				continue
			}

			strBody = string(b)
		}

		if f.Regex.MatchString(strBody) {
			left = append(left, msg)
		} else {
			right = append(right, msg)
		}
	}

	return left, right
}
