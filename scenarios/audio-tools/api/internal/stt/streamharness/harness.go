// Package streamharness contains the protocol-level regression harness for
// long dictation sessions. It deliberately measures transport behaviour, not
// recognition quality, so it remains useful with either STT provider.
package streamharness

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	SampleRate      = 16_000
	BatchDurationMS = 100
	BatchSamples    = SampleRate * BatchDurationMS / 1000
	FixtureSeconds  = 60
	FrameMagic      = "ATV2"
)

// Thresholds are intentionally count- and state-based. No threshold compares
// transcript text or word accuracy.
type Thresholds struct {
	MinPartials   int
	MinSegments   int
	MaxGapBatches int
}

var DefaultThresholds = Thresholds{MinPartials: 10, MinSegments: 1, MaxGapBatches: 30}

// Fixture returns a deterministic 60-second mono PCM stream. The alternating
// tones and pauses make it useful for segmenting tests while keeping the
// fixture source-controlled as code instead of committing a large binary.
func Fixture() io.Reader { return &fixture{remaining: SampleRate * FixtureSeconds} }

type fixture struct{ remaining int }

func (f *fixture) Read(p []byte) (int, error) {
	if f.remaining == 0 {
		return 0, io.EOF
	}
	n := len(p) / 2
	if n > f.remaining {
		n = f.remaining
	}
	for i := 0; i < n; i++ {
		// 4 seconds voiced, 1 second silence. The exact waveform is not an
		// accuracy claim; it only provides deterministic signal boundaries.
		sample := SampleRate*5 - f.remaining
		if sample%(SampleRate*5) >= SampleRate*4 {
			binary.LittleEndian.PutUint16(p[i*2:], 0)
			continue
		}
		phase := float64(sample%800) / 800
		value := int16(12000 * sinApprox(phase))
		binary.LittleEndian.PutUint16(p[i*2:], uint16(value))
	}
	f.remaining -= n
	return n * 2, nil
}

// sinApprox is a bounded triangle approximation. Avoiding a floating-point
// trig dependency keeps fixture generation cheap and deterministic.
func sinApprox(phase float64) float64 {
	if phase < .25 {
		return phase * 4
	}
	if phase < .75 {
		return 2 - phase*4
	}
	return phase*4 - 4
}

// Frame encodes one canonical ATV2 PCM batch, matching the browser relay's
// wire contract.
func Frame(sequence, startSample int64, pcm []byte) []byte {
	frame := make([]byte, 4+8+8+8+32+len(pcm))
	copy(frame, FrameMagic)
	binary.BigEndian.PutUint64(frame[4:], uint64(sequence))
	binary.BigEndian.PutUint64(frame[12:], uint64(startSample))
	binary.BigEndian.PutUint64(frame[20:], uint64(startSample+int64(len(pcm)/2)))
	digest := sha256.Sum256(pcm)
	copy(frame[28:], digest[:])
	copy(frame[60:], pcm)
	return frame
}

// Result is the transport-only verdict returned by a runner.
type Result struct {
	Partials             int
	Segments             int
	MaxPartialGapBatches int
	LastPartialBatch     int
	TerminalReason       string
}

func (r Result) Validate(t Thresholds) error {
	if t.MinPartials <= 0 {
		t.MinPartials = DefaultThresholds.MinPartials
	}
	if t.MinSegments <= 0 {
		t.MinSegments = DefaultThresholds.MinSegments
	}
	if t.MaxGapBatches <= 0 {
		t.MaxGapBatches = DefaultThresholds.MaxGapBatches
	}
	if r.Partials < t.MinPartials {
		return fmt.Errorf("partials=%d, want at least %d", r.Partials, t.MinPartials)
	}
	if r.Segments < t.MinSegments {
		return fmt.Errorf("segments=%d, want at least %d", r.Segments, t.MinSegments)
	}
	if r.MaxPartialGapBatches > t.MaxGapBatches {
		return fmt.Errorf("partial gap=%d batches, want at most %d", r.MaxPartialGapBatches, t.MaxGapBatches)
	}
	if r.TerminalReason == "" {
		return fmt.Errorf("terminal reason is missing")
	}
	return nil
}
