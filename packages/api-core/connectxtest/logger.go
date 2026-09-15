package connectxtest

import (
	"bytes"
	"log"
	"testing"
)

// NewLogger returns a logger and the buffer it writes to.
//
// Tests can pass the logger into code under test and inspect the returned
// buffer after the behavior has run.
func NewLogger(t *testing.T) (*log.Logger, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	return log.New(buf, "", 0), buf
}
