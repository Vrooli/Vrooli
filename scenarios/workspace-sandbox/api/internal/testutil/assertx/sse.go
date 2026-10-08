package assertx

import (
	"bytes"
	"strings"
	"testing"

	"workspace-sandbox/internal/sse"
)

// FrameSpec describes one expected frame in an AssertSSEFrameSequence
// call. Event matches exactly. DataPredicate is optional; when nil,
// any data is accepted. DataContains is sugar for "must contain this
// substring."
type FrameSpec struct {
	Event         string
	DataContains  string
	DataPredicate func(data []byte) bool
}

// AssertSSEFrameSequence checks that `frames` matches `want` in order.
// Extra frames at the end fail the assertion (with a clear message
// listing the extras). Missing frames fail. Mismatched events at any
// position fail.
func AssertSSEFrameSequence(t *testing.T, frames []sse.Frame, want []FrameSpec) {
	t.Helper()
	if len(frames) != len(want) {
		t.Errorf("AssertSSEFrameSequence: got %d frames, want %d", len(frames), len(want))
		describeFrames(t, "actual", frames)
		describeSpecs(t, "expected", want)
		return
	}
	for i, spec := range want {
		got := frames[i]
		if got.Event != spec.Event {
			t.Errorf("frame[%d]: event = %q, want %q", i, got.Event, spec.Event)
		}
		if spec.DataContains != "" && !bytes.Contains(got.Data, []byte(spec.DataContains)) {
			t.Errorf("frame[%d]: data missing substring %q (got %q)", i, spec.DataContains, string(got.Data))
		}
		if spec.DataPredicate != nil && !spec.DataPredicate(got.Data) {
			t.Errorf("frame[%d]: data predicate failed (data=%q)", i, string(got.Data))
		}
	}
}

func describeFrames(t *testing.T, label string, frames []sse.Frame) {
	t.Helper()
	var b strings.Builder
	b.WriteString(label + ":\n")
	for i, f := range frames {
		b.WriteString("  [")
		b.WriteString(itoa(i))
		b.WriteString("] event=")
		b.WriteString(f.Event)
		b.WriteString(" data=")
		b.WriteString(string(f.Data))
		b.WriteByte('\n')
	}
	t.Log(b.String())
}

func describeSpecs(t *testing.T, label string, specs []FrameSpec) {
	t.Helper()
	var b strings.Builder
	b.WriteString(label + ":\n")
	for i, s := range specs {
		b.WriteString("  [")
		b.WriteString(itoa(i))
		b.WriteString("] event=")
		b.WriteString(s.Event)
		if s.DataContains != "" {
			b.WriteString(" contains=")
			b.WriteString(s.DataContains)
		}
		b.WriteByte('\n')
	}
	t.Log(b.String())
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	var buf [16]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
