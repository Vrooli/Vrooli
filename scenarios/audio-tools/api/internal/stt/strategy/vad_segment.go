package strategy

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"audio-tools/internal/ai/sttchain"
)

// VADSegmenter is the per-session VAD-bounded segment strategy. It
// buffers raw 16-bit PCM @ 16 kHz, detects sustained silence using a
// short-term RMS threshold, and calls Provider.Transcribe once per
// VAD-bounded segment.
//
// Inputs are raw little-endian 16-bit PCM. Transports that send
// container-framed audio (browser MediaRecorder → WebM/Opus) must
// transcode to PCM before pushing AudioChunks.
type VADSegmenter struct {
	Provider sttchain.Provider

	// SilenceMs is the minimum sustained silence window that closes a
	// segment. Matches StreamConfig.VADSilenceMs (default 700).
	SilenceMs int

	// SampleRate of the inbound PCM. Default 16000.
	SampleRate int

	// SilenceRMS is the RMS amplitude threshold below which a frame is
	// considered silent. Default 250 (≈0.7% of int16 max).
	SilenceRMS float64

	// FrameMs is the frame size used for RMS evaluation. Default 20 ms.
	FrameMs int
}

// Kind reports the strategy kind for selector enforcement.
func (v *VADSegmenter) Kind() sttchain.StrategyKind { return sttchain.StrategyVADSegment }

// Run consumes chunks until the channel closes (or ctx fires),
// emitting Segment + Done events.
func (v *VADSegmenter) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	if v.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: VADSegmenter requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	v.applyDefaults()

	const sampleBytes = 2
	frameBytes := v.SampleRate * v.FrameMs / 1000 * sampleBytes
	silenceFramesNeeded := v.SilenceMs / v.FrameMs
	if silenceFramesNeeded < 1 {
		silenceFramesNeeded = 1
	}

	var buf []byte
	segStart := 0       // offset in buf where the current segment begins
	nextFrame := 0      // offset of the next frame to evaluate
	silentFrames := 0   // consecutive silent frames observed
	hasVoiced := false  // any voiced frame seen in the current segment

	var lastTier sttchain.ProviderTier
	var lastProviderID, lastModelID string
	var totalLatencyMs float64

	flushSegment := func(end int) {
		if !hasVoiced || end-segStart <= 0 {
			segStart = end
			hasVoiced = false
			return
		}
		seg := make([]byte, end-segStart)
		copy(seg, buf[segStart:end])
		req := sttchain.Request{
			Audio:                   seg,
			Language:                start.Language,
			InitialPrompt:           start.InitialPrompt,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}
		t0 := time.Now()
		res, err := v.Provider.Transcribe(ctx, req)
		latency := time.Since(t0)
		if err != nil {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
			segStart = end
			hasVoiced = false
			return
		}
		lastTier = res.Tier
		lastProviderID = res.ProviderID
		lastModelID = res.ModelID
		totalLatencyMs += float64(latency.Milliseconds())
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
			Text:             res.Text,
			DetectedLanguage: res.DetectedLanguage,
			ProviderTier:     res.Tier,
			ProviderID:       res.ProviderID,
			ModelID:          res.ModelID,
			LatencyMs:        float64(latency.Milliseconds()),
		}}
		segStart = end
		hasVoiced = false
	}

	emitDone := func() {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
			LockedTier: lastTier,
			ProviderID: lastProviderID,
			ModelID:    lastModelID,
			LatencyMs:  totalLatencyMs,
		}}
	}

	scan := func() {
		for nextFrame+frameBytes <= len(buf) {
			rms := frameRMS(buf[nextFrame : nextFrame+frameBytes])
			if rms < v.SilenceRMS {
				silentFrames++
				if hasVoiced && silentFrames >= silenceFramesNeeded {
					// Cut the segment at the start of the silence run.
					cut := nextFrame + frameBytes - silentFrames*frameBytes
					if cut < segStart {
						cut = segStart
					}
					flushSegment(cut)
					silentFrames = 0
				}
			} else {
				silentFrames = 0
				hasVoiced = true
			}
			nextFrame += frameBytes
		}
	}

	for {
		select {
		case <-ctx.Done():
			scan()
			flushSegment(len(buf))
			emitDone()
			return ctx.Err()
		case ch, ok := <-chunks:
			if !ok {
				scan()
				flushSegment(len(buf))
				emitDone()
				return nil
			}
			buf = append(buf, ch.Audio...)
			scan()
		}
	}
}

func (v *VADSegmenter) applyDefaults() {
	if v.SampleRate == 0 {
		v.SampleRate = 16000
	}
	if v.SilenceMs == 0 {
		v.SilenceMs = 700
	}
	if v.SilenceRMS == 0 {
		v.SilenceRMS = 250
	}
	if v.FrameMs == 0 {
		v.FrameMs = 20
	}
}

func frameRMS(frame []byte) float64 {
	if len(frame) < 2 {
		return 0
	}
	var sum float64
	n := len(frame) / 2
	for i := 0; i < n; i++ {
		s := int16(binary.LittleEndian.Uint16(frame[i*2:]))
		f := float64(s)
		sum += f * f
	}
	return math.Sqrt(sum / float64(n))
}
