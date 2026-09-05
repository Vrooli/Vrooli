package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

// bufferLogger returns a logger writing JSON to buf so tests can assert on the
// emitted records.
func bufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestFromContextReturnsDefaultWhenUnset(t *testing.T) {
	got := FromContext(context.Background())
	if got == nil {
		t.Fatal("FromContext returned nil; expected the default logger")
	}
	if got != slog.Default() {
		t.Error("FromContext should return slog.Default() when no logger is attached")
	}
}

func TestWithLoggerRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	logger := bufferLogger(&buf)

	ctx := WithLogger(context.Background(), logger)
	got := FromContext(ctx)
	if got != logger {
		t.Fatal("FromContext did not return the logger stored by WithLogger")
	}

	got.Info("hello", "k", "v")
	if buf.Len() == 0 {
		t.Fatal("expected the round-tripped logger to write a record")
	}
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("emitted record is not valid JSON: %v", err)
	}
	if rec["msg"] != "hello" || rec["k"] != "v" {
		t.Errorf("record missing expected fields: %v", rec)
	}
}

func TestWithLoggerIsolatesContexts(t *testing.T) {
	var buf bytes.Buffer
	logger := bufferLogger(&buf)
	parent := context.Background()
	child := WithLogger(parent, logger)

	// The parent context must remain unaffected by the child's logger.
	if FromContext(parent) == logger {
		t.Error("WithLogger mutated the parent context")
	}
	if FromContext(child) != logger {
		t.Error("child context lost its logger")
	}
}

func TestWithAttachesAttributes(t *testing.T) {
	var buf bytes.Buffer
	ctx := WithLogger(context.Background(), bufferLogger(&buf))

	child := With(ctx, "scenario", "swarm-manager")
	if child == nil {
		t.Fatal("With returned nil")
	}
	child.Info("event")

	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("emitted record is not valid JSON: %v", err)
	}
	if rec["scenario"] != "swarm-manager" {
		t.Errorf("With did not attach the attribute; got %v", rec)
	}
}

func TestWithFallsBackToDefaultLogger(t *testing.T) {
	// With on a bare context must not panic and must return a usable logger.
	got := With(context.Background(), "k", "v")
	if got == nil {
		t.Fatal("With returned nil for a context without a logger")
	}
}

func TestInitSetsJSONDefault(t *testing.T) {
	// Preserve and restore the process-wide default so this test is isolated.
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })

	Init()
	if slog.Default() == nil {
		t.Fatal("Init left a nil default logger")
	}
	// The default must be usable after Init.
	FromContext(context.Background()).Info("init-smoke")
}
