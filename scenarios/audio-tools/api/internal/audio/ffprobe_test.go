package audio_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"audio-tools/internal/audio"
	"audio-tools/internal/audio/mocks"
)

func TestProbeArgvShape(t *testing.T) {
	const probeJSON = `{"streams":[],"format":{"format_name":"wav"}}`
	fake := mocks.NewFakeRunner([]byte(probeJSON), nil)
	swapRunner(t, fake)

	if _, err := audio.Probe(context.Background(), []byte("X")); err != nil {
		t.Fatalf("Probe: %v", err)
	}
	if len(fake.Calls) != 1 || fake.Calls[0].Name != "ffprobe" {
		t.Fatalf("expected one ffprobe call, got %v", fake.Calls)
	}
	args := fake.Calls[0].Args
	if !containsAll(args, "-show_streams", "-show_format", "-of", "json") {
		t.Fatalf("argv missing standard probe flags: %v", args)
	}
}

func TestProbeFailsCleanlyOnGarbage(t *testing.T) {
	fake := mocks.NewFakeRunner([]byte("not-json"), nil)
	swapRunner(t, fake)

	if _, err := audio.Probe(context.Background(), []byte("x")); err == nil {
		t.Fatalf("expected Probe to fail on non-JSON output")
	}
}

func TestProbePropagatesRunnerError(t *testing.T) {
	want := errors.New("synthetic ffprobe fail")
	fake := mocks.NewFakeRunner(nil, want)
	swapRunner(t, fake)

	_, err := audio.Probe(context.Background(), []byte("x"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, want) && !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("error did not propagate: %v", err)
	}
}
