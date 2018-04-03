package sqsdr

import (
	"io"

	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

// Dump persists all messages in a queue to disk and then makes sure all of the
// messages end up back in the source queue.
type Dump struct {
	SourceClient   sqsiface.SQSAPI
	SourceQueueURL string
	Out            io.Writer
}

// Dump uses a FallthroughPipeline to place all messages in a temporary queue after
// they've been written to disk. The messages will be placed back into the source
// queue using a pipline in the reverse direction.
func (d *Dump) Dump() error {
	chooser := &RightPassthroughChooser{}
	leftSink := &NoOpSink{}
	rightSinkFunc := func(queueURL string, client sqsiface.SQSAPI) Sinker {
		pass := &SQSSink{
			QueueURL: queueURL,
			Client:   client,
		}

		w := &WriterSink{
			Writer:      d.Out,
			Passthrough: pass,
		}

		return w
	}

	f := &FallthroughPipeline{
		Chooser:        chooser,
		LeftSink:       leftSink,
		RightSinkFunc:  rightSinkFunc,
		SourceClient:   d.SourceClient,
		SourceQueueURL: d.SourceQueueURL,
	}

	return f.Run()
}
