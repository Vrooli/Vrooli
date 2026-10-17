package logx

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestStandardForwardsPrintfAndNilIsSafe(t *testing.T) {
	var buffer bytes.Buffer
	Standard{Logger: log.New(&buffer, "", 0)}.Printf("event %d", 7)
	if got := buffer.String(); got != "event 7\n" {
		t.Fatalf("logged output = %q", got)
	}
	Standard{}.Printf("ignored")
}

func TestDefaultReturnsUsableAdapter(t *testing.T) {
	if Default().Logger == nil || !strings.Contains(Default().Logger.Prefix(), "") {
		t.Fatal("Default did not return a standard logger adapter")
	}
}
