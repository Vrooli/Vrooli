package audio

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// fakeRunner captures invocations and returns canned responses.
type fakeRunner struct {
	calls   []fakeCall
	stdout  []byte
	err     error
	respond func(name string, args []string) ([]byte, error)
}

type fakeCall struct {
	Name  string
	Stdin []byte
	Args  []string
}

func (f *fakeRunner) Run(_ context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	f.calls = append(f.calls, fakeCall{Name: name, Stdin: append([]byte(nil), stdin...), Args: append([]string(nil), args...)})
	if f.respond != nil {
		return f.respond(name, args)
	}
	return f.stdout, f.err
}

func swapRunner(t *testing.T, r Runner) {
	t.Helper()
	// Seed the lookup-path cache to "available" so the fake takes
	// effect even when ffmpeg/ffprobe aren't installed on this host.
	ffmpegOnce.Do(func() {})
	ffmpegAvailable = true
	ffprobeOnce.Do(func() {})
	ffprobeAvailable = true
	orig := DefaultRunner
	DefaultRunner = r
	t.Cleanup(func() {
		DefaultRunner = orig
	})
}

func TestTranscodeArgvShape(t *testing.T) {
	fake := &fakeRunner{stdout: []byte("WAVDATA")}
	swapRunner(t, fake)

	out, err := Transcode(context.Background(), []byte("INPUT"))
	if err != nil {
		t.Fatalf("Transcode: %v", err)
	}
	if string(out) != "WAVDATA" {
		t.Fatalf("expected fake stdout, got %q", out)
	}
	if len(fake.calls) != 1 {
		t.Fatalf("expected 1 invocation, got %d", len(fake.calls))
	}
	call := fake.calls[0]
	if call.Name != "ffmpeg" {
		t.Fatalf("expected ffmpeg, got %s", call.Name)
	}
	if !containsAll(call.Args, "-i", "pipe:0", "-ar", "16000", "-ac", "1", "-f", "wav", "pipe:1") {
		t.Fatalf("argv missing required tokens: %v", call.Args)
	}
	if string(call.Stdin) != "INPUT" {
		t.Fatalf("stdin not forwarded: %q", call.Stdin)
	}
}

func TestTrimArgvIncludesStartAndEnd(t *testing.T) {
	fake := &fakeRunner{stdout: []byte("OUT")}
	swapRunner(t, fake)

	if _, err := Trim(context.Background(), []byte("X"), "mp3", 1.5, 4.0, ""); err != nil {
		t.Fatalf("Trim: %v", err)
	}
	args := fake.calls[0].Args
	if !containsAll(args, "-ss", "1.5", "-to", "4") {
		t.Fatalf("missing -ss/-to in argv: %v", args)
	}
	if !containsAll(args, "-f", "mp3") {
		t.Fatalf("expected outFormat default to input format mp3, got %v", args)
	}
}

func TestTrimRejectsNegativeStart(t *testing.T) {
	if _, err := Trim(context.Background(), nil, "wav", -1, 2, ""); err == nil {
		t.Fatalf("expected negative-start rejection")
	}
}

func TestVolumeAppliesGainFilter(t *testing.T) {
	fake := &fakeRunner{stdout: []byte("VOL")}
	swapRunner(t, fake)

	if _, err := Volume(context.Background(), []byte("Y"), "wav", -3, ""); err != nil {
		t.Fatalf("Volume: %v", err)
	}
	args := fake.calls[0].Args
	if !containsAll(args, "-af") || !containsAny(args, "volume=-3dB") {
		t.Fatalf("expected gain filter -3dB in argv: %v", args)
	}
}

func TestFadeOmitsFilterWhenZero(t *testing.T) {
	fake := &fakeRunner{stdout: []byte("F")}
	swapRunner(t, fake)

	if _, err := Fade(context.Background(), []byte("X"), "wav", 0, 0, ""); err != nil {
		t.Fatalf("Fade: %v", err)
	}
	args := fake.calls[0].Args
	for i, a := range args {
		if a == "-af" {
			t.Fatalf("expected no -af when both fades are zero, got %v (index %d)", args, i)
		}
	}
}

func TestMetadataParsesFfprobeJSON(t *testing.T) {
	const probeJSON = `{
	  "streams": [
	    {"codec_name": "mp3", "sample_rate": "48000", "channels": 2, "bit_rate": "192000"}
	  ],
	  "format": {"format_name": "mp3", "duration": "12.5", "bit_rate": "192000", "tags": {"title": "hello"}}
	}`
	fake := &fakeRunner{stdout: []byte(probeJSON)}
	swapRunner(t, fake)

	meta, err := Probe(context.Background(), []byte("xxx"))
	if err != nil {
		t.Fatalf("Metadata: %v", err)
	}
	if meta.DurationSeconds != 12.5 {
		t.Fatalf("duration: %v", meta.DurationSeconds)
	}
	if meta.SampleRate != 48000 {
		t.Fatalf("sample rate: %d", meta.SampleRate)
	}
	if meta.Channels != 2 {
		t.Fatalf("channels: %d", meta.Channels)
	}
	if meta.Codec != "mp3" {
		t.Fatalf("codec: %q", meta.Codec)
	}
	if meta.Tags["title"] != "hello" {
		t.Fatalf("tags missing title: %v", meta.Tags)
	}
}

func TestRunnerErrorPropagates(t *testing.T) {
	want := errors.New("synthetic ffmpeg fail")
	fake := &fakeRunner{err: want}
	swapRunner(t, fake)

	_, err := Transcode(context.Background(), []byte("x"))
	if err == nil {
		t.Fatalf("expected propagated error")
	}
	if !strings.Contains(err.Error(), want.Error()) && !errors.Is(err, want) {
		t.Fatalf("error did not surface underlying cause: %v", err)
	}
}

func containsAll(haystack []string, needles ...string) bool {
	for _, n := range needles {
		found := false
		for _, h := range haystack {
			if h == n {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func containsAny(haystack []string, needles ...string) bool {
	for _, n := range needles {
		for _, h := range haystack {
			if h == n {
				return true
			}
		}
	}
	return false
}
