package strategy

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
)

// VAD state-emission cadence. See VadStateEvent doc. Throttling lives
// here on the server (per plan §12); UI consumers must not add their
// own throttle layer.
const (
	vadEmitSilenceMinIntervalMs = 50  // ≤ ~20 Hz during sustained silence
	vadEmitVoicedMinIntervalMs  = 500 // ≤ ~2 Hz during sustained voiced
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
	// segment. Matches StreamConfig.VADSilenceMs (default 1200).
	SilenceMs int

	// SampleRate of the inbound PCM. Default 16000.
	SampleRate int

	// SilenceRMS is the RMS amplitude threshold below which a frame is
	// considered silent. Default 250 (≈0.7% of int16 max).
	SilenceRMS float64

	// FrameMs is the frame size used for RMS evaluation. Default 20 ms.
	FrameMs int

	// Clock is the wall-clock seam used for per-segment latency
	// measurement. Defaults to clock.System{}.
	Clock clock.Clock
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
	segStart := 0      // offset in buf where the current segment begins
	nextFrame := 0     // offset of the next frame to evaluate
	silentFrames := 0  // consecutive silent frames observed
	hasVoiced := false // any voiced frame seen in the current segment

	var lastTier sttchain.ProviderTier
	var lastProviderID, lastModelID string
	var totalLatencyMs float64

	// VAD state-emission bookkeeping. tickSeq is per-stream monotonic so
	// out-of-order clients can drop stale frames. lastEmittedVoiced lets
	// us force a tick on state transitions even when throttled. zeroEmit
	// is the sentinel for "no tick emitted yet" — we always emit on the
	// first voiced→silence transition (no event before first speech, per
	// plan §8 contract).
	var (
		tickSeq             uint64
		lastEmitFrameIdx    int // frame index of the last emission
		lastEmittedVoiced   bool
		anyTickEmitted      bool
	)
	clk := v.Clock
	if clk == nil {
		clk = clock.System{}
	}
	emitVad := func(voiced bool, silenceElapsedMs int64, frameIdx int) {
		tickSeq++
		lastEmitFrameIdx = frameIdx
		lastEmittedVoiced = voiced
		anyTickEmitted = true
		events <- sttchain.StreamEvent{
			Kind: sttchain.StreamEventVadState,
			VadState: &sttchain.VadStateEvent{
				Voiced:           voiced,
				SilenceElapsedMs: silenceElapsedMs,
				SilenceTimeoutMs: int64(v.SilenceMs),
				TickSeq:          tickSeq,
			},
		}
	}

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
		t0 := clk.Now()
		res, err := v.Provider.Transcribe(ctx, req)
		latency := clk.Now().Sub(t0)
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
			isSilent := rms < v.SilenceRMS
			if isSilent {
				silentFrames++
			} else {
				silentFrames = 0
				hasVoiced = true
			}

			// VAD state emission. Per plan §8:
			//  - no event before the first voiced frame in a segment
			//  - emit on voiced↔silence transitions
			//  - otherwise throttle by audio-time elapsed:
			//      ≤20 Hz silence (≥50 ms / tick), ≤2 Hz voiced (≥500 ms / tick).
			//
			// Throttle is intentionally audio-time, not wall-clock: the
			// chain feeds frames in real time in production but tests
			// batch all PCM in one chunk, and the contract talks about
			// silence-clock ticks (the user-visible mic ring), not CPU
			// time. Audio-time throttling gives both worlds the same
			// cadence guarantees.
			currentFrameIdx := nextFrame / frameBytes
			if hasVoiced {
				voicedNow := !isSilent
				silenceElapsedMs := int64(silentFrames) * int64(v.FrameMs)
				shouldEmit := false
				if !anyTickEmitted {
					shouldEmit = true
				} else if voicedNow != lastEmittedVoiced {
					shouldEmit = true
				} else {
					minIntervalMs := vadEmitVoicedMinIntervalMs
					if !voicedNow {
						minIntervalMs = vadEmitSilenceMinIntervalMs
					}
					elapsedMs := (currentFrameIdx - lastEmitFrameIdx) * v.FrameMs
					if elapsedMs >= minIntervalMs {
						shouldEmit = true
					}
				}
				if shouldEmit {
					emitVad(voicedNow, silenceElapsedMs, currentFrameIdx)
				}
			}

			if isSilent && hasVoiced && silentFrames >= silenceFramesNeeded {
				// Cut the segment at the start of the silence run.
				cut := nextFrame + frameBytes - silentFrames*frameBytes
				if cut < segStart {
					cut = segStart
				}
				flushSegment(cut)
				silentFrames = 0
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
		v.SilenceMs = 1200
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
