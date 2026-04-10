package logger

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	instance := New("app-monitor")
	if instance == nil {
		t.Fatal("expected logger instance")
	}
	if instance.prefix != "app-monitor" {
		t.Fatalf("expected prefix app-monitor, got %q", instance.prefix)
	}
}

func TestLoggerInfoIncludesLevelAndPrefix(t *testing.T) {
	var buffer bytes.Buffer
	originalWriter := log.Writer()
	originalFlags := log.Flags()
	log.SetOutput(&buffer)
	log.SetFlags(0)
	defer log.SetOutput(originalWriter)
	defer log.SetFlags(originalFlags)

	instance := New("app-monitor")
	instance.Info("hello world")

	output := buffer.String()
	if !strings.Contains(output, "[INFO]") {
		t.Fatalf("expected INFO level in log output, got %q", output)
	}
	if !strings.Contains(output, "[app-monitor]") {
		t.Fatalf("expected prefix in log output, got %q", output)
	}
	if !strings.Contains(output, "hello world") {
		t.Fatalf("expected message in log output, got %q", output)
	}
}
