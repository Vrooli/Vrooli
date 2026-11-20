package audio

import (
	"bytes"
	"context"
	"encoding/binary"
	"math"
	"testing"
)

// Real-ffmpeg regression tests for the behaviors the fake Runner cannot
// observe: actual seek/cut semantics, fade placement, filter-graph labels,
// and muxer choice. They skip on hosts without ffmpeg/ffprobe, matching
// TestTranscode_Success.

func requireRealFFmpeg(t *testing.T) {
	t.Helper()
	if !ffmpegOnPath() {
		t.Skip("ffmpeg not available")
	}
	if !hasFfprobe() {
		t.Skip("ffprobe not available")
	}
}

// sineWAV builds a mono 16-bit PCM WAV with a 440 Hz tone.
func sineWAV(t *testing.T, seconds float64, sampleRate int) []byte {
	t.Helper()
	const frequency = 440.0
	samples := int(seconds * float64(sampleRate))
	data := make([]byte, samples*2)
	for i := 0; i < samples; i++ {
		v := int16(12000 * math.Sin(2*math.Pi*frequency*float64(i)/float64(sampleRate)))
		binary.LittleEndian.PutUint16(data[i*2:], uint16(v))
	}

	var buf bytes.Buffer
	write := func(v any) {
		if err := binary.Write(&buf, binary.LittleEndian, v); err != nil {
			t.Fatalf("build wav: %v", err)
		}
	}
	buf.WriteString("RIFF")
	write(uint32(36 + len(data)))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	write(uint32(16))
	write(uint16(1)) // PCM
	write(uint16(1)) // mono
	write(uint32(sampleRate))
	write(uint32(sampleRate * 2)) // byte rate
	write(uint16(2))              // block align
	write(uint16(16))             // bits per sample
	buf.WriteString("data")
	write(uint32(len(data)))
	buf.Write(data)
	return buf.Bytes()
}

// wavSamples extracts the PCM payload from a RIFF/WAVE byte stream.
func wavSamples(t *testing.T, wav []byte) []int16 {
	t.Helper()
	if len(wav) < 12 || string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("not a RIFF/WAVE payload (len=%d)", len(wav))
	}
	offset := 12
	for offset+8 <= len(wav) {
		id := string(wav[offset : offset+4])
		size := int(binary.LittleEndian.Uint32(wav[offset+4:]))
		body := offset + 8
		if id == "data" {
			end := body + size
			if end > len(wav) {
				end = len(wav)
			}
			raw := wav[body:end]
			out := make([]int16, len(raw)/2)
			for i := range out {
				out[i] = int16(binary.LittleEndian.Uint16(raw[i*2:]))
			}
			return out
		}
		offset = body + size + (size & 1) // chunks are word-aligned
	}
	t.Fatalf("no data chunk in wav")
	return nil
}

func rms16(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}

func probeDuration(t *testing.T, audio []byte) float64 {
	t.Helper()
	meta, err := Probe(context.Background(), audio)
	if err != nil {
		t.Fatalf("probe duration: %v", err)
	}
	return meta.DurationSeconds
}

// TestIntegrationTrimUsesAbsoluteEnd is the regression for the -to
// placement bug: with an input -ss, an output -to was measured on the
// post-seek timeline, so Trim(2,5) produced 5 seconds instead of 3.
func TestIntegrationTrimUsesAbsoluteEnd(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 4, 8000)

	out, err := Trim(context.Background(), src, "wav", 1, 2, "")
	if err != nil {
		t.Fatalf("Trim: %v", err)
	}
	got := probeDuration(t, out)
	if math.Abs(got-1) > 0.1 {
		t.Fatalf("trimmed duration = %.3fs, want ~1.0s", got)
	}
}

// TestIntegrationFadeOutFadesTheTail is the regression for the misplaced
// fade-out, which previously silenced everything after the fade window.
func TestIntegrationFadeOutFadesTheTail(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 2, 8000)

	out, err := Fade(context.Background(), src, "wav", 0, 0.5, "")
	if err != nil {
		t.Fatalf("Fade: %v", err)
	}
	samples := wavSamples(t, out)
	rate := 8000
	head := rms16(samples[:rate/10]) // first 100ms: unfaded start
	if head < 1000 {
		t.Fatalf("head RMS %.1f, expected an unfaded start", head)
	}
	// The middle must still be at full level: the old bug silenced
	// everything after the 0.5s fade window.
	middle := rms16(samples[rate*3/4 : rate*5/4])
	if middle < head*0.8 {
		t.Fatalf("middle RMS %.1f collapsed (head %.1f): fade-out is silencing the body", middle, head)
	}
	// And the final 50ms must be attenuated by the fade.
	tail := rms16(samples[len(samples)-rate/20:])
	if tail > middle/2 {
		t.Fatalf("tail RMS %.1f is not faded out (middle %.1f)", tail, middle)
	}
}

// TestIntegrationMergeConcatAndCrossfade exercises both filter graphs on
// real ffmpeg; the crossfade case asserts the graph's final label is
// actually mapped.
func TestIntegrationMergeConcatAndCrossfade(t *testing.T) {
	requireRealFFmpeg(t)
	a := sineWAV(t, 1, 8000)
	b := sineWAV(t, 1, 8000)

	concat, err := Merge(context.Background(), [][]byte{a, b}, []string{"wav", "wav"}, "wav", 0)
	if err != nil {
		t.Fatalf("concat merge: %v", err)
	}
	if got := probeDuration(t, concat); math.Abs(got-2) > 0.15 {
		t.Fatalf("concat duration = %.3fs, want ~2.0s", got)
	}

	crossfaded, err := Merge(context.Background(), [][]byte{a, b}, []string{"wav", "wav"}, "wav", 0.5)
	if err != nil {
		t.Fatalf("crossfade merge: %v", err)
	}
	if got := probeDuration(t, crossfaded); math.Abs(got-1.5) > 0.2 {
		t.Fatalf("crossfade duration = %.3fs, want ~1.5s", got)
	}
}

// TestIntegrationAACTranscodeProducesADTS is the regression for the
// "-f aac" muxer bug: ffmpeg has no aac muxer, so every AAC output
// request used to fail.
func TestIntegrationAACTranscodeProducesADTS(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 0.5, 44100)

	out, err := TranscodeOpts(context.Background(), src, "aac", 0, 0, 0)
	if err != nil {
		t.Fatalf("TranscodeOpts(aac): %v", err)
	}
	if len(out) < 2 {
		t.Fatalf("aac output too short: %d bytes", len(out))
	}
	// ADTS sync word: 12 bits set, layer 00.
	if out[0] != 0xFF || out[1]&0xF0 != 0xF0 {
		t.Fatalf("output is not ADTS (first bytes % x)", out[:2])
	}
	// AAC priming/padding makes the measured duration slightly longer
	// than the source; the point of the assertion is that a real stream
	// was produced at roughly the requested length.
	if got := probeDuration(t, out); got < 0.4 || got > 0.8 {
		t.Fatalf("aac duration = %.3fs, want roughly 0.5s", got)
	}
}

// TestIntegrationSplitProducesExpectedChunks verifies chunk count, starts,
// and per-chunk content on real ffmpeg.
func TestIntegrationSplitProducesExpectedChunks(t *testing.T) {
	requireRealFFmpeg(t)
	src := sineWAV(t, 2, 8000)

	chunks, err := Split(context.Background(), src, "wav", 0.75, nil, "wav")
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(chunks) != 3 {
		t.Fatalf("chunk count = %d, want 3", len(chunks))
	}
	wantStarts := []float64{0, 0.75, 1.5}
	for i, want := range wantStarts {
		if math.Abs(chunks[i].Start-want) > 1e-9 {
			t.Fatalf("chunk %d start = %v, want %v", i, chunks[i].Start, want)
		}
		if len(chunks[i].Audio) == 0 {
			t.Fatalf("chunk %d has no audio", i)
		}
	}
	if chunks[0].Duration != 0.75 || chunks[1].Duration != 0.75 {
		t.Fatalf("chunk durations = %v/%v, want 0.75/0.75", chunks[0].Duration, chunks[1].Duration)
	}
	if math.Abs(chunks[2].Duration-0.5) > 1e-6 {
		t.Fatalf("last chunk duration = %v, want 0.5 (input duration - start)", chunks[2].Duration)
	}
	if got := probeDuration(t, chunks[0].Audio); math.Abs(got-0.75) > 0.1 {
		t.Fatalf("first chunk duration = %.3fs, want ~0.75s", got)
	}
}

func TestIntegrationTrimBeyondEndFails(t *testing.T) {
	if !ffmpegOnPath() || !HasFfprobe() {
		t.Skip("ffmpeg/ffprobe not available")
	}
	src := sineWAV(t, 1, 8000)
	if _, err := Trim(context.Background(), src, "wav", 5, 0, "wav"); err == nil {
		t.Fatal("trimming past the input duration must fail, not return a header-only clip")
	}
}
