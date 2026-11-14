package logging

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestStandardLoggerSatisfiesLoggerSeam(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)
	var seam Logger = logger
	seam.Printf("route %s", "ready")

	if got := strings.TrimSpace(output.String()); got != "route ready" {
		t.Fatalf("logger output = %q, want %q", got, "route ready")
	}
}

func TestDefaultReturnsLogger(t *testing.T) {
	if Default() == nil {
		t.Fatal("Default() returned nil")
	}
}
