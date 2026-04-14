package scenarioexec

import (
	"bytes"
	"testing"
)

func TestWriterSupportsStreamingRejectsBuffer(t *testing.T) {
	if WriterSupportsStreaming(&bytes.Buffer{}) {
		t.Fatal("WriterSupportsStreaming() should not treat bytes.Buffer as streaming")
	}
}
