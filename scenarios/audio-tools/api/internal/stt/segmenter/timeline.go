package segmenter

import (
	"fmt"
	"sync"

	"audio-tools/internal/ai/sttchain"
)

// AudioTimeline is the Segmenter's bounded canonical-PCM ownership seam. It
// stores sample-addressed audio once, independent of a provider adapter, and
// makes missing/evicted ranges explicit instead of silently verifying with
// unrelated bytes.
type AudioTimeline struct {
	mu         sync.Mutex
	maxSamples int64
	chunks     []sttchain.AudioChunk
}

type AudioRangeStatus string

const (
	AudioRangeAttached AudioRangeStatus = "attached"
	AudioRangeMissing  AudioRangeStatus = "missing"
	AudioRangeEvicted  AudioRangeStatus = "evicted"
)

type AudioRange struct {
	Status AudioRangeStatus
	PCM    []byte
}

func NewAudioTimeline(maxSamples int64) *AudioTimeline {
	if maxSamples <= 0 {
		maxSamples = 16_000 * 60
	}
	return &AudioTimeline{maxSamples: maxSamples}
}

func (t *AudioTimeline) Append(chunk sttchain.AudioChunk) error {
	if chunk.EndSample < chunk.StartSample {
		return fmt.Errorf("audio timeline: invalid sample range")
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.chunks) > 0 {
		last := t.chunks[len(t.chunks)-1]
		if chunk.StartSample != last.EndSample {
			return fmt.Errorf("audio timeline: non-contiguous chunk range")
		}
	}
	chunk.Audio = append([]byte(nil), chunk.Audio...)
	t.chunks = append(t.chunks, chunk)
	t.evictLocked()
	return nil
}

func (t *AudioTimeline) Lookup(start, end int64) AudioRange {
	t.mu.Lock()
	defer t.mu.Unlock()
	if end <= start || len(t.chunks) == 0 {
		return AudioRange{Status: AudioRangeMissing}
	}
	if start < t.chunks[0].StartSample {
		return AudioRange{Status: AudioRangeEvicted}
	}
	if end > t.chunks[len(t.chunks)-1].EndSample {
		return AudioRange{Status: AudioRangeMissing}
	}
	var out []byte
	for _, chunk := range t.chunks {
		if chunk.EndSample <= start || chunk.StartSample >= end {
			continue
		}
		from := maxInt64(start, chunk.StartSample)
		to := minInt64(end, chunk.EndSample)
		byteStart := (from - chunk.StartSample) * 2
		byteEnd := (to - chunk.StartSample) * 2
		if byteStart < 0 || byteEnd > int64(len(chunk.Audio)) {
			return AudioRange{Status: AudioRangeMissing}
		}
		out = append(out, chunk.Audio[byteStart:byteEnd]...)
	}
	if len(out) == 0 {
		return AudioRange{Status: AudioRangeMissing}
	}
	return AudioRange{Status: AudioRangeAttached, PCM: out}
}

func (t *AudioTimeline) evictLocked() {
	for len(t.chunks) > 1 && t.chunks[len(t.chunks)-1].EndSample-t.chunks[0].StartSample > t.maxSamples {
		t.chunks = t.chunks[1:]
	}
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
