package strategy

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	"audio-tools/internal/protoint"
	voice "audio-tools/internal/stt/pipeline"

	"github.com/vrooli/api-core/schedule"
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
//
// Boundary accuracy is preserved via three coupled mechanisms (see
// docs/reference/streaming-tuning.md):
//
//   - PreRollMs: the next segment's start is rewound by this many ms of
//     PCM so Whisper has the same pre-word audio context a human would
//     hear before the first phoneme.
//   - TrailingPadMs: the cut keeps real silence after the last voiced
//     frame, instead of slicing flush against the silence boundary.
//   - InitialPromptWords: the last K words of the previous segment's
//     emitted text feed Whisper's initial_prompt for the next segment.
//
// Pre-roll duplicates words by design; emitted text is deduped against
// the per-session committed string via pipeline.DeduplicateOverlap so
// consumers never see the repeats.
type VADSegmenter struct {
	Provider sttchain.Provider

	// SilenceMs is the minimum sustained silence window that closes a
	// segment. Matches StreamConfig.VADSilenceMs (default 1200).
	SilenceMs int

	// SampleRate of the inbound PCM. Default 16000.
	SampleRate int

	// SilenceRMS is the RMS amplitude threshold below which a frame is
	// considered silent. Default 64. The browser capture path treats samples
	// above 256 as signal, but host-device loopback can attenuate voiced frames
	// substantially before they reach the server; 64 preserves a wide silence
	// margin while allowing the preview lane to run on those valid frames.
	SilenceRMS float64

	// FrameMs is the frame size used for RMS evaluation. Default 20 ms.
	FrameMs int

	// PreRollMs (default 800) carries the trailing N ms of the previous
	// segment into the next, so Whisper has pre-word audio context.
	PreRollMs int

	// TrailingPadMs (default 200) is how much real silence to keep on
	// the trailing edge of an emitted segment.
	TrailingPadMs int

	// InitialPromptWords (default 20) is the count of previous-segment
	// words forwarded as the next request's initial_prompt.
	InitialPromptWords int

	// PreviewIntervalMs is the minimum amount of NEW audio that must
	// arrive before another in-flight preview transcription is launched.
	// 0 disables previews entirely.
	//
	// Without previews this strategy emits text only at silence
	// boundaries, so an operator speaking continuously sees nothing until
	// they pause. That is the whole live-text experience on every host
	// without a streaming engine — which today is every macOS and Windows
	// host, and every Linux host without an NVIDIA GPU, because
	// kyutai-stt is the only streaming provider and it requires CUDA.
	//
	// Previews are best-effort and strictly additive: they never advance
	// segStart, never touch the committed transcript, and never surface an
	// error. A preview that fails or returns nothing is simply not shown.
	PreviewIntervalMs int

	// PreviewWindowMs bounds how much trailing audio a preview
	// transcribes. Preview cost must not grow with segment length: during
	// continuous speech no segment commits, so the in-flight segment grows
	// without bound and previewing all of it would be quadratic in session
	// length — the exact failure this pipeline has been repaired for.
	// Clamped to the tail of the in-flight segment. 0 means no clamp.
	PreviewWindowMs int

	// MaxSegmentMs bounds a voiced segment when the speaker never produces
	// enough silence for VAD to cut it. Without this safety boundary a
	// continuous turn becomes one ever-growing Whisper request and the raw PCM
	// buffer retains the whole turn. The cut uses the same pre-roll and
	// bounded text deduplication as a normal silence boundary. Zero uses the
	// production default.
	MaxSegmentMs int

	// Clock is the wall-clock seam used for per-segment latency
	// measurement. Defaults to schedule.System().
	Clock schedule.Clock
	// Logger records VAD lifecycle diagnostics through the scenario seam.
	Logger logx.Logger
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
	cursor *sttchain.ConsumptionCursor,
) error {
	if v.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: VADSegmenter requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	// Resolve defaults on a private copy so concurrent runs never mutate the
	// reusable strategy configuration.
	resolved := *v
	resolved.applyDefaults()
	logger := resolved.Logger
	if logger == nil {
		logger = logx.Std{}
	}
	logger.Printf("[stt-vad] session start: silence_ms=%d silence_rms=%.0f sample_rate=%d frame_ms=%d preroll_ms=%d trailing_pad_ms=%d preview_interval_ms=%d preview_window_ms=%d",
		resolved.SilenceMs, resolved.SilenceRMS, resolved.SampleRate, resolved.FrameMs, resolved.PreRollMs, resolved.TrailingPadMs, resolved.PreviewIntervalMs, resolved.PreviewWindowMs)
	defer logger.Printf("[stt-vad] session end")

	s := newVADRunState(ctx, &resolved, start, chunks, events, cursor, logger)
	return s.run()
}

func (s *vadRunState) emitVad(voiced bool, silenceElapsedMs int64, frameIdx int, timedOut bool) {
	s.tickSeq++
	s.lastEmitFrameIdx = frameIdx
	s.lastEmittedVoiced = voiced
	s.anyTickEmitted = true
	s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventVadState, VadState: &sttchain.VadStateEvent{
		Voiced: voiced, SilenceElapsedMs: silenceElapsedMs, SilenceTimeoutMs: s.silenceTimeoutMs,
		TickSeq: s.tickSeq, SilenceTimedOut: timedOut,
	}}
}

// flushSegment emits one committed segment and compacts its consumed PCM.
func (s *vadRunState) flushSegment(callCtx context.Context, end int) bool {
	const sampleBytes = 2
	if !s.hasVoiced || end-s.segStart <= 0 {
		s.segStart = end
		s.hasVoiced = false
		return true
	}
	seg := make([]byte, end-s.segStart)
	copy(seg, s.buf[s.segStart:end])
	prompt := s.start.InitialPrompt
	if s.v.InitialPromptWords > 0 && s.lastPromptText != "" {
		tail := voice.LastNWords(s.lastPromptText, s.v.InitialPromptWords)
		prompt = strings.TrimSpace(prompt + " " + tail)
	}
	req := sttchain.Request{
		Audio: seg, Format: s.start.InputFormat, Language: s.start.Language, InitialPrompt: prompt,
		SkipSpeakerVerification: s.start.SkipSpeakerVerification, VADFilter: s.start.VADFilter,
		BYOKProvider: s.start.BYOKProvider, BYOKKey: s.start.BYOKKey, LPBSToken: s.start.LPBSToken,
		UserIdentity: s.start.UserIdentity,
	}
	t0 := s.clk.Now()
	res, err := s.v.Provider.Transcribe(callCtx, req)
	latency := s.clk.Now().Sub(t0)
	if err != nil {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		return false
	}
	if res == nil {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf("stt-vad: provider returned no result for %d bytes; retaining audio for retry", len(seg))}
		return false
	}
	if strings.TrimSpace(res.Text) == "" {
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf("stt-vad: provider returned an empty transcript for %d bytes; retaining audio for retry", len(seg))}
		return false
	}
	s.lastTier, s.lastProviderID, s.lastModelID = res.Tier, res.ProviderID, res.ModelID
	s.totalLatencyMs += float64(latency.Milliseconds())
	merged := voice.DeduplicateOverlapBounded(s.committed, res.Text, s.maxTextOverlapWords)
	newTail := strings.TrimSpace(strings.TrimPrefix(merged, s.committed))
	s.committed, s.lastPromptText = merged, res.Text
	if newTail != "" {
		s.segmentOrdinal++
		startSample := s.bufStartSample + int64(s.segStart/sampleBytes)
		endSample := s.bufStartSample + int64(end/sampleBytes)
		s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
			Text: newTail, SegmentID: fmt.Sprintf("%s:%d:vad:%d", s.start.SessionID, s.start.Generation, s.segmentOrdinal),
			Generation: s.start.Generation, StartSample: startSample, EndSample: endSample,
			Words:            timedWordsForCommittedTail(newTail, res.Words),
			DetectedLanguage: res.DetectedLanguage, ProviderTier: res.Tier, ProviderID: res.ProviderID, ModelID: res.ModelID,
			LatencyMs: float64(latency.Milliseconds()), Confidence: res.Confidence, Audio: seg,
		}}
	}
	s.segStart = advanceWithPreRoll(end, s.preRollBytes)
	s.hasVoiced = false
	if s.segStart > 0 {
		prefix := s.segStart
		copy(s.buf, s.buf[prefix:])
		s.buf = s.buf[:len(s.buf)-prefix]
		s.bufStartSample += int64(prefix / sampleBytes)
		s.nextFrame -= prefix
		if s.nextFrame < 0 {
			s.nextFrame = 0
		}
		s.lastPreviewEnd -= prefix
		if s.lastPreviewEnd < 0 {
			s.lastPreviewEnd = 0
		}
		s.lastEmitFrameIdx -= prefix / s.frameBytes
		if s.lastEmitFrameIdx < 0 {
			s.lastEmitFrameIdx = 0
		}
		s.segStart = 0
	}
	return true
}

// trimSilentPrefix bounds storage during long leading or inter-segment silence.
func (s *vadRunState) trimSilentPrefix() {
	const sampleBytes = 2
	if s.hasVoiced || s.nextFrame <= 0 {
		return
	}
	keep := s.preRollBytes + s.frameBytes
	if keep < 2*s.frameBytes {
		keep = 2 * s.frameBytes
	}
	prefix := s.nextFrame - keep
	if prefix < s.v.SampleRate*sampleBytes {
		return
	}
	prefix -= prefix % sampleBytes
	remaining := len(s.buf) - prefix
	compacted := make([]byte, remaining)
	copy(compacted, s.buf[prefix:])
	s.buf = compacted
	s.bufStartSample += int64(prefix / sampleBytes)
	s.nextFrame -= prefix
	if s.nextFrame < 0 {
		s.nextFrame = 0
	}
	s.segStart -= prefix
	if s.segStart < 0 {
		s.segStart = 0
	}
	s.lastPreviewEnd -= prefix
	if s.lastPreviewEnd < 0 {
		s.lastPreviewEnd = 0
	}
	s.lastEmitFrameIdx -= prefix / s.frameBytes
	if s.lastEmitFrameIdx < 0 {
		s.lastEmitFrameIdx = 0
	}
}

func (s *vadRunState) emitDone() {
	s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
		FinalText: s.committed, LockedTier: s.lastTier, ProviderID: s.lastProviderID, ModelID: s.lastModelID, LatencyMs: s.totalLatencyMs,
	}}
}

// scan evaluates all complete frames currently buffered and applies cuts.
func (s *vadRunState) scan() {
	for s.nextFrame+s.frameBytes <= len(s.buf) {
		rms := frameRMS(s.buf[s.nextFrame : s.nextFrame+s.frameBytes])
		isSilent := rms < s.v.SilenceRMS
		if isSilent {
			s.silentFrames++
		} else {
			s.silentFrames = 0
			s.hasVoiced = true
		}
		currentFrameIdx := s.nextFrame / s.frameBytes
		if !isSilent && !s.firstVoicedLogged {
			s.firstVoicedLogged = true
			s.logger.Printf("[stt-vad] first voiced frame sample=%d rms=%.0f", currentFrameIdx*s.v.SampleRate*s.v.FrameMs/1000, rms)
		}
		voicedNow := !isSilent
		silenceElapsedMs := int64(s.silentFrames) * int64(s.v.FrameMs)
		timedOut := s.silentFrames >= s.silenceFramesNeeded
		s.emitVadTick(voicedNow, silenceElapsedMs, currentFrameIdx, timedOut)
		s.applySegmentCut(isSilent, rms)
		s.nextFrame += s.frameBytes
	}
	s.trimSilentPrefix()
}

func (s *vadRunState) emitVadTick(voiced bool, silenceElapsedMs int64, frameIdx int, timedOut bool) {
	if timedOut {
		s.emitVad(voiced, silenceElapsedMs, frameIdx, true)
		if !s.hasVoiced {
			s.silentFrames = 0
		}
		return
	}
	if !s.hasVoiced {
		return
	}
	shouldEmit := !s.anyTickEmitted || voiced != s.lastEmittedVoiced
	if !shouldEmit {
		minIntervalMs := vadEmitVoicedMinIntervalMs
		if !voiced {
			minIntervalMs = vadEmitSilenceMinIntervalMs
		}
		elapsedMs := (frameIdx - s.lastEmitFrameIdx) * s.v.FrameMs
		shouldEmit = elapsedMs >= minIntervalMs
	}
	if shouldEmit {
		s.emitVad(voiced, silenceElapsedMs, frameIdx, false)
	}
}

func (s *vadRunState) applySegmentCut(isSilent bool, rms float64) {
	const sampleBytes = 2
	if isSilent && s.hasVoiced && s.silentFrames >= s.silenceFramesNeeded {
		silenceStart := s.nextFrame + s.frameBytes - s.silentFrames*s.frameBytes
		cut := silenceStart + s.trailingPadBytes
		if cut > s.nextFrame+s.frameBytes {
			cut = s.nextFrame + s.frameBytes
		}
		if cut < s.segStart+s.frameBytes {
			cut = s.segStart + s.frameBytes
		}
		s.logger.Printf("[stt-vad] segment cut: silence_ms=%d threshold_ms=%d segment_bytes=%d segment_ms=%d rms=%.0f silence_rms=%.0f",
			s.silentFrames*s.v.FrameMs, s.v.SilenceMs, cut-s.segStart,
			(cut-s.segStart)/(s.v.SampleRate*sampleBytes/1000), rms, s.v.SilenceRMS)
		s.finishSegmentCut(cut)
	} else if s.hasVoiced && s.v.MaxSegmentMs > 0 &&
		s.nextFrame+s.frameBytes-s.segStart >= s.v.SampleRate*s.v.MaxSegmentMs/1000*sampleBytes &&
		(s.nextFlushAttempt == 0 || s.nextFrame+s.frameBytes >= s.nextFlushAttempt) {
		cut := s.nextFrame + s.frameBytes
		s.logger.Printf("[stt-vad] forced segment cut: max_segment_ms=%d segment_bytes=%d segment_ms=%d",
			s.v.MaxSegmentMs, cut-s.segStart, (cut-s.segStart)/(s.v.SampleRate*sampleBytes/1000))
		s.finishSegmentCut(cut)
	}
}

func (s *vadRunState) finishSegmentCut(cut int) {
	if !s.flushSegment(s.ctx, cut) {
		s.nextFlushAttempt = s.nextFrame + s.frameBytes + s.retryStepBytes
	} else {
		s.nextFlushAttempt = 0
	}
	s.silentFrames = 0
}

func (s *vadRunState) run() error {
	for {
		select {
		case <-s.ctx.Done():
			s.scan()
			if flushCtx, cancel, ok := bestEffortTailContext(s.ctx); ok {
				s.flushSegment(flushCtx, len(s.buf))
				cancel()
			}
			s.drainPreviews()
			s.emitDone()
			return s.ctx.Err()
		case ch, ok := <-s.chunks:
			if !ok {
				s.scan()
				s.flushSegment(s.ctx, len(s.buf))
				s.drainPreviews()
				s.emitDone()
				return nil
			}
			s.chunkCount++
			if s.chunkCount <= 8 {
				s.logger.Printf("[stt-vad] chunk_received n=%d bytes=%d buffered_bytes=%d", s.chunkCount, len(ch.Audio), len(s.buf)+len(ch.Audio))
			}
			s.cursor.Observe(ch)
			s.buf = append(s.buf, ch.Audio...)
			s.scan()
			s.maybePreview()
		}
	}
}

// advanceWithPreRoll rewinds the next-segment start by preRollBytes so
// the new segment overlaps the trailing audio of the one just emitted.
// Clamped to 0 to avoid underflow.
func advanceWithPreRoll(end, preRollBytes int) int {
	if preRollBytes <= 0 {
		return end
	}
	start := end - preRollBytes
	if start < 0 {
		start = 0
	}
	return start
}

func (v *VADSegmenter) applyDefaults() {
	if v.SampleRate == 0 {
		v.SampleRate = 16000
	}
	if v.SilenceMs == 0 {
		// Defensive self-default for direct test construction. In production the
		// selector always supplies SilenceMs from cfg.VADSilenceMs, which derives
		// from stt.DefaultVADSilenceMs — keep this literal equal to it. It cannot
		// import internal/stt here: strategy is imported BY that package (cycle).
		v.SilenceMs = 1200
	}
	if v.SilenceRMS == 0 {
		v.SilenceRMS = 64
	}
	if v.FrameMs == 0 {
		v.FrameMs = 20
	}
	if v.PreviewWindowMs == 0 {
		// Long enough for Whisper to have useful context while keeping the
		// first preview responsive and the per-call cost bounded.
		v.PreviewWindowMs = 3000
	}
	if v.MaxSegmentMs == 0 {
		// Keep the native batch request comfortably below Whisper's fixed
		// audio context while still giving it enough speech context to decode
		// natural phrases. This is a boundedness default, not a latency claim.
		v.MaxSegmentMs = 15000
	}
	// PreRollMs/TrailingPadMs/InitialPromptWords are additive — zero is
	// a valid "disable" value for each. Operator config supplies real
	// defaults at the selector layer.
}

func frameRMS(frame []byte) float64 {
	if len(frame) < 2 {
		return 0
	}
	var sum float64
	n := len(frame) / 2
	for i := 0; i < n; i++ {
		s := protoint.PCMInt16(binary.LittleEndian.Uint16(frame[i*2:]))
		f := float64(s)
		sum += f * f
	}
	return math.Sqrt(sum / float64(n))
}
