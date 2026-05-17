package audio

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRunFfprobeJSONArgvShape(t *testing.T) {
	const probeJSON = `{"streams":[],"format":{"format_name":"wav"}}`
	fake := &fakeRunner{stdout: []byte(probeJSON)}
	swapRunner(t, fake)

	out, err := runFfprobeJSON(context.Background(), []byte("X"))
	if err != nil {
		t.Fatalf("runFfprobeJSON: %v", err)
	}
	if string(out) != probeJSON {
		t.Fatalf("expected fake stdout, got %q", out)
	}
	if len(fake.calls) != 1 || fake.calls[0].Name != "ffprobe" {
		t.Fatalf("expected one ffprobe call, got %v", fake.calls)
	}
	args := fake.calls[0].Args
	if !containsAll(args, "-show_streams", "-show_format", "-of", "json") {
		t.Fatalf("argv missing standard probe flags: %v", args)
	}
}

func TestProbeFailsCleanlyOnGarbage(t *testing.T) {
	fake := &fakeRunner{stdout: []byte("not-json")}
	swapRunner(t, fake)

	if _, err := Probe(context.Background(), []byte("x")); err == nil {
		t.Fatalf("expected Probe to fail on non-JSON output")
	}
}

func TestRunFfprobeJSONPropagatesError(t *testing.T) {
	want := errors.New("synthetic ffprobe fail")
	fake := &fakeRunner{err: want}
	swapRunner(t, fake)

	_, err := runFfprobeJSON(context.Background(), []byte("x"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, want) && !strings.Contains(err.Error(), want.Error()) {
		t.Fatalf("error did not propagate: %v", err)
	}
}
