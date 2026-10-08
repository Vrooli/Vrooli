package lifecycle

import (
	"bytes"
	"testing"

	"github.com/vrooli/vrooli/internal/shell"
)

func TestWriterSupportsStreamingRejectsBuffer(t *testing.T) {
	if shell.WriterSupportsStreaming(&bytes.Buffer{}) {
		t.Fatal("WriterSupportsStreaming() should not treat bytes.Buffer as streaming")
	}
}
