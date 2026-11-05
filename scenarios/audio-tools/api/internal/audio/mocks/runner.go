// Package mocks holds the hoisted test fakes for the audio domain.
// FakeRunner stands in for the ffmpeg/ffprobe os/exec Runner so handler
// and op tests can pin argv shape without depending on a binary.
package mocks

import (
	"context"

	"audio-tools/internal/audio"
)

// Call records one invocation of FakeRunner.Run for assertion.
type Call struct {
	Name  string
	Stdin []byte
	Args  []string
}

// FakeRunner is the canonical audio.Runner fake. Tests configure it
// with Stdout/Err for static responses, or Respond for argv-aware
// per-call behavior. Calls is the recorded transcript.
type FakeRunner struct {
	Calls   []Call
	Stdout  []byte
	Err     error
	Respond func(name string, args []string) ([]byte, error)
}

// NewFakeRunner constructs a FakeRunner pre-seeded with stdout/err.
// Pass nil bytes and nil error for the default "no-op runner".
func NewFakeRunner(stdout []byte, err error) *FakeRunner {
	return &FakeRunner{Stdout: stdout, Err: err}
}

// Run records the invocation and returns the configured response.
func (f *FakeRunner) Run(_ context.Context, name string, stdin []byte, args ...string) ([]byte, error) {
	f.Calls = append(f.Calls, Call{
		Name:  name,
		Stdin: append([]byte(nil), stdin...),
		Args:  append([]string(nil), args...),
	})
	if f.Respond != nil {
		return f.Respond(name, args)
	}
	return f.Stdout, f.Err
}

// Compile-time guarantee that *FakeRunner satisfies audio.Runner.
var _ audio.Runner = (*FakeRunner)(nil)
