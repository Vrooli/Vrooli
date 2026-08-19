package strategy

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"sync"

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
	// considered silent. Default 250 (≈0.7% of int16 max).
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
	v.applyDefaults()
	logger := v.Logger
	if logger == nil {
		logger = logx.Std{}
	}
	logger.Printf("[stt-vad] session start: silence_ms=%d silence_rms=%.0f sample_rate=%d frame_ms=%d preroll_ms=%d trailing_pad_ms=%d",
		v.SilenceMs, v.SilenceRMS, v.SampleRate, v.FrameMs, v.PreRollMs, v.TrailingPadMs)
	defer logger.Printf("[stt-vad] session end")

	const sampleBytes = 2
	frameBytes := v.SampleRate * v.FrameMs / 1000 * sampleBytes
	silenceFramesNeeded := v.SilenceMs / v.FrameMs
	if silenceFramesNeeded < 1 {
		silenceFramesNeeded = 1
	}
	// The vad-state ticks echo the frame-QUANTISED silence threshold actually
	// used for the cut (silenceFramesNeeded * FrameMs), not the raw SilenceMs.
	// The client's auto-stop fires on silenceElapsedMs >= silenceTimeoutMs, and
	// silenceElapsedMs only ever takes frame-quantised values; echoing the raw
	// SilenceMs would leave the threshold unreachable whenever FrameMs does not
	// divide SilenceMs evenly.
	silenceTimeoutMs := int64(silenceFramesNeeded * v.FrameMs)
	preRollBytes := v.SampleRate * v.PreRollMs / 1000 * sampleBytes
	trailingPadBytes := v.SampleRate * v.TrailingPadMs / 1000 * sampleBytes
	// A segment can only duplicate the audio carried in its pre-roll (plus
	// the small trailing-pad context). Keep the text merge bounded to that
	// physical overlap. Searching the whole committed transcript is unsafe:
	// repeated phrases later in a long dictation can otherwise be mistaken for
	// pre-roll and silently deleted.
	maxTextOverlapWords := (v.PreRollMs + v.TrailingPadMs + 99) / 100
	if maxTextOverlapWords < 1 {
		maxTextOverlapWords = 1
	}

	var buf []byte
	segStart := 0      // offset in buf where the current segment begins
	nextFrame := 0     // offset of the next frame to evaluate
	silentFrames := 0  // consecutive silent frames observed
	hasVoiced := false // any voiced frame seen in the current segment

	// Rolling cross-segment context. committed accumulates emitted
	// (deduped) text for overlap detection; lastPromptText is the raw
	// previous-segment text used to build the next request's
	// initial_prompt.
	committed := ""
	lastPromptText := ""

	var lastTier sttchain.ProviderTier
	var lastProviderID, lastModelID string
	var totalLatencyMs float64
	var lastPreviewEnd int

	var (
		tickSeq           uint64
		lastEmitFrameIdx  int
		lastEmittedVoiced bool
		anyTickEmitted    bool
	)
	clk := v.Clock
	if clk == nil {
		clk = schedule.System()
	}
	emitVad := func(voiced bool, silenceElapsedMs int64, frameIdx int, timedOut bool) {
		tickSeq++
		lastEmitFrameIdx = frameIdx
		lastEmittedVoiced = voiced
		anyTickEmitted = true
		events <- sttchain.StreamEvent{
			Kind: sttchain.StreamEventVadState,
			VadState: &sttchain.VadStateEvent{
				Voiced:           voiced,
				SilenceElapsedMs: silenceElapsedMs,
				SilenceTimeoutMs: silenceTimeoutMs,
				TickSeq:          tickSeq,
				SilenceTimedOut:  timedOut,
			},
		}
	}

	// flushSegment cuts [segStart, end) from buf, transcribes it with
	// rolling text context, deduplicates against `committed`, and emits
	// only the new tail as a SegmentEvent. It then advances segStart
	// backwards by preRollBytes so the next segment overlaps the
	// trailing audio of the one just emitted — Whisper sees pre-word
	// context, callers see deduped text.
	flushSegment := func(end int) {
		if !hasVoiced || end-segStart <= 0 {
			segStart = end
			hasVoiced = false
			return
		}
		seg := make([]byte, end-segStart)
		copy(seg, buf[segStart:end])

		// Build the per-call initial_prompt: operator hint + last K
		// words of the previous segment. LastNWords is empty when
		// InitialPromptWords <= 0 or lastPromptText is empty, so this
		// is a no-op for the first segment of a session.
		prompt := start.InitialPrompt
		if v.InitialPromptWords > 0 && lastPromptText != "" {
			tail := voice.LastNWords(lastPromptText, v.InitialPromptWords)
			prompt = strings.TrimSpace(prompt + " " + tail)
		}

		req := sttchain.Request{
			Audio:                   seg,
			Format:                  start.InputFormat,
			Language:                start.Language,
			InitialPrompt:           prompt,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			VADFilter:               start.VADFilter,
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
			// Keep the segment pending. The next voiced/silence boundary
			// retries the complete audio window; advancing segStart here
			// would make a provider outage indistinguishable from successful
			// transcription and silently discard the user's speech.
			return
		}
		if res == nil {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
				"stt-vad: provider returned no result for %d bytes; retaining audio for retry", len(seg))}
			return
		}
		if strings.TrimSpace(res.Text) == "" {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
				"stt-vad: provider returned an empty transcript for %d bytes; retaining audio for retry", len(seg))}
			return
		}
		lastTier = res.Tier
		lastProviderID = res.ProviderID
		lastModelID = res.ModelID
		totalLatencyMs += float64(latency.Milliseconds())

		// Dedup against the per-session committed string. The new tail
		// (everything after the overlap with committed) is what we emit.
		merged := voice.DeduplicateOverlapBounded(committed, res.Text, maxTextOverlapWords)
		newTail := strings.TrimSpace(strings.TrimPrefix(merged, committed))
		committed = merged
		lastPromptText = res.Text

		if newTail != "" {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
				Text:             newTail,
				DetectedLanguage: res.DetectedLanguage,
				ProviderTier:     res.Tier,
				ProviderID:       res.ProviderID,
				ModelID:          res.ModelID,
				LatencyMs:        float64(latency.Milliseconds()),
				// Egress-gate inputs (stripped before the wire): the
				// segment's confidence signals and its canonical-PCM bytes.
				Confidence: res.Confidence,
				Audio:      seg,
			}}
		}
		segStart = advanceWithPreRoll(end, preRollBytes)
		hasVoiced = false

		// The segmenter owns the PCM buffer, so discard the committed prefix
		// immediately. Preview goroutines already own copies of their windows;
		// no asynchronous reader depends on this backing array. Keep only the
		// configured pre-roll for the next segment and translate all offsets
		// back to the compacted buffer coordinate system.
		if segStart > 0 {
			prefix := segStart
			copy(buf, buf[prefix:])
			buf = buf[:len(buf)-prefix]
			nextFrame -= prefix
			if nextFrame < 0 {
				nextFrame = 0
			}
			lastPreviewEnd -= prefix
			if lastPreviewEnd < 0 {
				lastPreviewEnd = 0
			}
			segStart = 0
		}
	}

	// Preview lane. Single-flight and asynchronous: the provider call must
	// not sit in the chunk loop, or every preview would delay silence
	// detection by its own latency and make the auto-stop boundary drift.
	var previewWG sync.WaitGroup
	previewSlot := make(chan struct{}, 1)
	previewIntervalBytes := v.SampleRate * v.PreviewIntervalMs / 1000 * sampleBytes
	previewWindowBytes := v.SampleRate * v.PreviewWindowMs / 1000 * sampleBytes
	maybePreview := func() {
		if v.PreviewIntervalMs <= 0 || !hasVoiced {
			return
		}
		end := len(buf)
		if end-lastPreviewEnd < previewIntervalBytes {
			return
		}
		select {
		case previewSlot <- struct{}{}:
		default:
			// A preview is still running. Skipping is correct: previews are
			// replaceable, so the next one supersedes whatever this would
			// have said, and queueing them would let a slow provider build
			// an unbounded backlog of stale hypotheses.
			return
		}
		lastPreviewEnd = end
		windowStart := segStart
		if previewWindowBytes > 0 && end-windowStart > previewWindowBytes {
			windowStart = end - previewWindowBytes
		}
		window := make([]byte, end-windowStart)
		copy(window, buf[windowStart:end])

		prompt := start.InitialPrompt
		if v.InitialPromptWords > 0 && lastPromptText != "" {
			prompt = strings.TrimSpace(prompt + " " + voice.LastNWords(lastPromptText, v.InitialPromptWords))
		}
		req := sttchain.Request{
			Audio:                   window,
			Format:                  start.InputFormat,
			Language:                start.Language,
			InitialPrompt:           prompt,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			VADFilter:               start.VADFilter,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}

		previewWG.Add(1)
		go func() {
			defer previewWG.Done()
			defer func() { <-previewSlot }()
			res, err := v.Provider.Transcribe(ctx, req)
			if err != nil || res == nil {
				return
			}
			text := strings.TrimSpace(res.Text)
			if text == "" {
				return
			}
			select {
			case events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: text}}:
			case <-ctx.Done():
			}
		}()
	}

	// drainPreviews blocks until no preview can still send. A partial that
	// arrived after Done would be text the consumer can never settle.
	drainPreviews := func() { previewWG.Wait() }

	emitDone := func() {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
			FinalText:  committed,
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

			currentFrameIdx := nextFrame / frameBytes
			voicedNow := !isSilent
			silenceElapsedMs := int64(silentFrames) * int64(v.FrameMs)
			timedOut := silentFrames >= silenceFramesNeeded

			// Threshold-crossing tick MUST fire regardless of `hasVoiced`.
			// `hasVoiced` resets to false after each segment cut (see
			// flushSegment), so on the second-and-subsequent silences in a
			// turn the throttle gate below would never see hasVoiced=true and
			// the timedOut tick would be lost — the client's auto-stop latch
			// (SilenceTimedOut) would never trigger and one-shot recording
			// would hang with the ring visually full. The cut-the-segment
			// branch below stays gated on hasVoiced (no audio = nothing to
			// flush), but the tick itself is engine-of-record for the client
			// and must always emit at threshold.
			// Note on counter reset: when hasVoiced=true we want the
			// segment-cut branch below to run on this frame, which needs
			// `silentFrames >= silenceFramesNeeded` to still hold — so we do
			// NOT reset silentFrames in that case (the cut branch resets it).
			// When hasVoiced=false there's no cut to do, but we still want to
			// avoid spamming a timedOut tick every subsequent frame, so we
			// reset silentFrames here.
			if timedOut {
				emitVad(voicedNow, silenceElapsedMs, currentFrameIdx, true)
				if !hasVoiced {
					silentFrames = 0
				}
			} else if hasVoiced {
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
					emitVad(voicedNow, silenceElapsedMs, currentFrameIdx, false)
				}
			}

			if isSilent && hasVoiced && silentFrames >= silenceFramesNeeded {
				// Cut math:
				//   - "silence start" = nextFrame + frameBytes - silentFrames*frameBytes
				//   - keep TrailingPadMs of real silence after the last voiced frame
				silenceStart := nextFrame + frameBytes - silentFrames*frameBytes
				cut := silenceStart + trailingPadBytes
				if cut > nextFrame+frameBytes {
					cut = nextFrame + frameBytes
				}
				if cut < segStart+frameBytes {
					cut = segStart + frameBytes
				}
				// Observability: diagnosing user-reported "stops too early"
				// requires knowing the silence window and audio extent of
				// each cut. Cheap log — one line per cut, not per frame.
				logger.Printf("[stt-vad] segment cut: silence_ms=%d threshold_ms=%d segment_bytes=%d segment_ms=%d rms=%.0f silence_rms=%.0f",
					silentFrames*v.FrameMs, v.SilenceMs,
					cut-segStart,
					(cut-segStart)/(v.SampleRate*sampleBytes/1000),
					rms, v.SilenceRMS,
				)
				flushSegment(cut)
				silentFrames = 0
			} else if hasVoiced && v.MaxSegmentMs > 0 &&
				nextFrame+frameBytes-segStart >= v.SampleRate*v.MaxSegmentMs/1000*sampleBytes {
				// Continuous speech has no VAD silence boundary. Cut at the
				// current complete frame so the next provider request remains
				// bounded and the committed PCM can be released immediately.
				cut := nextFrame + frameBytes
				logger.Printf("[stt-vad] forced segment cut: max_segment_ms=%d segment_bytes=%d segment_ms=%d",
					v.MaxSegmentMs, cut-segStart,
					(cut-segStart)/(v.SampleRate*sampleBytes/1000),
				)
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
			drainPreviews()
			emitDone()
			return ctx.Err()
		case ch, ok := <-chunks:
			if !ok {
				scan()
				flushSegment(len(buf))
				drainPreviews()
				emitDone()
				return nil
			}
			cursor.Observe(ch)
			buf = append(buf, ch.Audio...)
			scan()
			maybePreview()
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
		v.SilenceRMS = 250
	}
	if v.FrameMs == 0 {
		v.FrameMs = 20
	}
	if v.PreviewWindowMs == 0 {
		// Long enough for Whisper to have real context, short enough that a
		// preview costs about the same at minute 60 as at minute 1.
		v.PreviewWindowMs = 8000
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
