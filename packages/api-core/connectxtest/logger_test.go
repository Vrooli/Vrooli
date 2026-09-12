package connectxtest

import (
	"strings"
	"testing"
)

func TestNewLoggerCapturesOutput(t *testing.T) {
	logger, buf := NewLogger(t)

	logger.Print("handler failed")

	if got := buf.String(); !strings.Contains(got, "handler failed") {
		t.Fatalf("buffer = %q, want logged text", got)
	}
}

func TestNewLoggerReturnsFreshBuffers(t *testing.T) {
	loggerA, bufA := NewLogger(t)
	loggerB, bufB := NewLogger(t)

	loggerA.Print("a")
	loggerB.Print("b")

	if strings.Contains(bufA.String(), "b") {
		t.Fatalf("first buffer captured second logger output: %q", bufA.String())
	}
	if strings.Contains(bufB.String(), "a") {
		t.Fatalf("second buffer captured first logger output: %q", bufB.String())
	}
}
