package strategy

import (
	"context"
	"fmt"
	"log"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/clock"
	voice "audio-tools/internal/stt/pipeline"
)

// OverlapAgree is the LocalAgreement-N streaming strategy (Macháček et
// al. 2023) for batch-only providers.
//
// It runs over a single GROWING audio buffer: on each iteration it
// re-transcribes pcm[committedAudioBytes:] (the same audio prefix as
// before, plus newly arrived chunks). Because consecutive hypotheses
// share a real audio prefix, their text shares a real text prefix —
// LocalAgreement can extract it. When CommitRuns consecutive hypotheses
// agree on a prefix, that prefix commits as a Segment and the audio
// cursor (committedAudioBytes) advances to the END timestamp of the last
// committed word, so the next iteration's audio starts immediately after
// the committed material and committed words can never be re-emitted.
//
// This replaces a prior sliding-window implementation whose windows
// covered DIFFERENT audio offsets and therefore could never share a
// prefix to agree on; that scheme only ever committed at end-of-stream.
// See docs/internal/PROBLEMS.md "OverlapAgree commit gap" for the
// original failure mode.
//
// Tuning levers:
//
//   - WindowMs: also doubles as the lower bound on the first call's
//     audio (capped at 1.5s so the first hypothesis arrives in real
//     time). Matches StreamConfig.OverlapWindowMs (default 2000).
//   - CommitRuns: how many consecutive iterations must agree on a
//     prefix before it commits. Matches StreamConfig.OverlapCommitRuns
//     (default 2).
//   - AdvanceMs: minimum new audio between iterations. Default
//     WindowMs/2.
//   - MaxWindowMs: cap on the uncommitted audio buffer. If no commit
//     happens in this much audio, force-advance the cursor so per-call
//     latency stays bounded (default 25 000ms, matches faster-whisper's
//     preferred chunk size).
//
// TriggerVAD makes the strategy run a settle attempt on silence
// boundaries detected from frame RMS analysis (production default).
// Whisper sees clean audio edges, transcripts are more stable, and
// LocalAgreement agreement happens at a much higher rate than the
// stopwatch alternative.
const TriggerVAD = "vad"

// TriggerStopwatch makes the strategy run a settle attempt every
// AdvanceMs of accumulated audio, regardless of voice state. Used as
// a legacy fallback and by scripted-text behaviour tests where the
// fixture audio is silent zeros that the VAD trigger would never
// classify as voiced.
const TriggerStopwatch = "stopwatch"

type OverlapAgree struct {
	Provider sttchain.Provider

	WindowMs    int
	CommitRuns  int
	SampleRate  int
	AdvanceMs   int
	MaxWindowMs int
	// MaxAgreedTokens caps the per-iteration agreement walk so the
	// comparison cost and variance accumulation stay bounded even when
	// the uncommitted buffer holds many seconds of speech. 0 means
	// unbounded; default applied by applyDefaults is 30 tokens.
	MaxAgreedTokens int

	// MaxStallRejects is the stall-fallback commit policy. When
	// LocalAgreement keeps producing hypotheses that DIVERGE from the
	// committed prefix (the model wandered on hard audio / jittery word
	// timestamps), the divergence-reject path commits nothing and the
	// uncommitted tail grows toward the MaxWindowMs net — every settle
	// attempt then re-transcribes an ever-larger window, saturating the
	// Whisper semaphore and making finalization SLOWER than a single
	// batch call. After this many CONSECUTIVE divergence-rejects, the
	// strategy force-commits the freshest hypothesis tail and advances
	// the cursor, bounding tail growth / re-transcription cost well
	// before the 25s net. The counter resets on any forward commit.
	//
	// 0 disables the fallback (only the MaxWindowMs net applies — the
	// pre-fallback behavior). This field carries no applyDefaults
	// default precisely so that a directly-constructed OverlapAgree
	// preserves legacy behavior and so 0 ("disabled") survives end to
	// end; the operator default (3) is applied at the config layer
	// (selector.Defaults / streamCfgDoc). Acceptable operator range
	// [1,10].
	MaxStallRejects int

	// Trigger selects the settle-attempt source: TriggerVAD (default)
	// or TriggerStopwatch. See the constants above.
	Trigger string

	// VAD-trigger fields. Defaults applied by applyDefaults match
	// VADSegmenter's so a silence that closes a VADSegment also
	// closes an OverlapAgree settle attempt on the same audio.
	SilenceMs  int     // sustained silence window that triggers a settle (default 1200)
	SilenceRMS float64 // amplitude threshold below which a frame is silent (default 250)
	FrameMs    int     // frame size for RMS evaluation (default 20)

	Clock clock.Clock
}

// Kind reports the strategy kind for selector enforcement.
func (o *OverlapAgree) Kind() sttchain.StrategyKind { return sttchain.StrategyOverlapAgree }

// hypothesis is one iteration's transcription, kept in `recent` so the
// next iteration can agree against it. words is empty when the backend
// doesn't supply word timestamps; the cursor advance then degrades to
// "leave committedAudioBytes alone" and mergeAgreed prevents re-emission.
type hypothesis struct {
	text  string
	words []sttchain.TimedWord
}

// Run consumes chunks, transcribes the growing buffer, emits Partial +
// Segment events under LocalAgreement-N, and a terminal Done.
func (o *OverlapAgree) Run(
	ctx context.Context,
	start sttchain.StreamStart,
	chunks <-chan sttchain.AudioChunk,
	events chan<- sttchain.StreamEvent,
) error {
	if o.Provider == nil {
		err := fmt.Errorf("audio-tools/stt/strategy: OverlapAgree requires a Provider")
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{}}
		return err
	}
	o.applyDefaults()
	log.Printf("[stt-overlap] session start: trigger=%s window_ms=%d advance_ms=%d commit_runs=%d max_window_ms=%d max_stall_rejects=%d silence_ms=%d silence_rms=%.0f frame_ms=%d max_agreed_tokens=%d sample_rate=%d",
		o.Trigger, o.WindowMs, o.AdvanceMs, o.CommitRuns, o.MaxWindowMs, o.MaxStallRejects, o.SilenceMs, o.SilenceRMS, o.FrameMs, o.MaxAgreedTokens, o.SampleRate)
	defer log.Printf("[stt-overlap] session end")

	const sampleBytes = 2
	advanceBytes := o.SampleRate * o.AdvanceMs / 1000 * sampleBytes
	if advanceBytes <= 0 {
		advanceBytes = o.SampleRate * o.WindowMs / 2000 * sampleBytes
	}
	// minWindowBytes is the floor on the FIRST call's audio size so we
	// don't waste a Whisper call on a tiny opening fragment. Capped at
	// 1.5s so the first hypothesis still arrives in real time on a
	// 2s+ WindowMs.
	const minWindowCapMs = 1500
	minWindowBytes := o.SampleRate * o.WindowMs / 1000 * sampleBytes
	if cap := o.SampleRate * minWindowCapMs / 1000 * sampleBytes; minWindowBytes > cap {
		minWindowBytes = cap
	}
	maxWindowBytes := o.SampleRate * o.MaxWindowMs / 1000 * sampleBytes

	var pcm []byte
	committedAudioBytes := 0
	committed := ""
	var recent []hypothesis
	// lastAdvanced is true when the previous commit moved
	// committedAudioBytes forward via word-aligned advance. While true,
	// the NEXT successful agreement covers genuinely new (post-advance)
	// audio, so we use appendAfterAdvance instead of mergeAgreed —
	// mergeAgreed's divergence detector would otherwise reject the
	// expected "no overlap with committed" state as a wander.
	lastAdvanced := false
	// stallRejects counts CONSECUTIVE divergence-rejects (model wander
	// with no commit). When it reaches MaxStallRejects the stall-fallback
	// force-commits the freshest hypothesis tail (see the divergence
	// branch in processIteration). Reset to 0 on any forward commit
	// (normal agreement commit or forceCommitAll).
	stallRejects := 0
	// nextTriggerAt is the pcm length at which the next iteration may
	// run. Initialized so the first iteration waits for max(advance,
	// minWindow) bytes of audio.
	nextTriggerAt := advanceBytes
	if minWindowBytes > nextTriggerAt {
		nextTriggerAt = minWindowBytes
	}

	var lastTier sttchain.ProviderTier
	var lastProviderID, lastModelID string
	var totalLatencyMs float64

	emitDone := func() {
		events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{
			FinalText:  committed,
			LockedTier: lastTier,
			ProviderID: lastProviderID,
			ModelID:    lastModelID,
			LatencyMs:  totalLatencyMs,
		}}
	}

	transcribe := func(audio []byte) (*sttchain.Result, error) {
		req := sttchain.Request{
			Audio:                   audio,
			Format:                  start.InputFormat,
			Language:                start.Language,
			InitialPrompt:           committed,
			SkipSpeakerVerification: start.SkipSpeakerVerification,
			VADFilter:               start.VADFilter,
			BYOKProvider:            start.BYOKProvider,
			BYOKKey:                 start.BYOKKey,
			LPBSToken:               start.LPBSToken,
			UserIdentity:            start.UserIdentity,
		}
		clk := o.Clock
		if clk == nil {
			clk = clock.System{}
		}
		t0 := clk.Now()
		res, err := o.Provider.Transcribe(ctx, req)
		totalLatencyMs += float64(clk.Now().Sub(t0).Milliseconds())
		return res, err
	}

	// wordEndBytes converts the END timestamp of the n-th word into a byte
	// offset (s16le, mono). Returns -1 when n is out of range, words is
	// empty, or the timestamp is non-positive — callers then skip the
	// cursor advance and rely on mergeAgreed alone.
	wordEndBytes := func(words []sttchain.TimedWord, n int) int {
		if n <= 0 || n > len(words) {
			return -1
		}
		endSec := words[n-1].End
		if endSec <= 0 {
			return -1
		}
		return int(endSec*float64(o.SampleRate)) * sampleBytes
	}

	// VAD frame analysis state. Used only in TriggerVAD mode; the
	// stopwatch path ignores these. nextFrame is a byte offset into
	// pcm — it advances monotonically as more audio is scanned.
	frameBytes := o.SampleRate * o.FrameMs / 1000 * sampleBytes
	silenceFramesNeeded := o.SilenceMs / o.FrameMs
	if silenceFramesNeeded < 1 {
		silenceFramesNeeded = 1
	}
	nextFrame := 0
	silentFrames := 0
	hasVoiced := false
	// Diagnostic state — log the first voiced detection so it's
	// obvious when (or whether) the VAD is seeing audio at all, and
	// track silent-run high-water-marks so we can log "buffer N
	// frames of silence, never reached threshold M".
	loggedFirstVoiced := false
	maxSilentRunSeen := 0

	// scanVADBoundary scans frames starting at nextFrame, advancing
	// the cursor and silentFrames counter. Returns the byte offset of
	// the first detected end-of-utterance silence (the frame right
	// AFTER the silence threshold is reached), or -1 when no
	// boundary forms before the buffer runs out. On a boundary the
	// VAD state resets so the next call detects the NEXT utterance.
	scanVADBoundary := func() int {
		for nextFrame+frameBytes <= len(pcm) {
			rms := frameRMS(pcm[nextFrame : nextFrame+frameBytes])
			isSilent := rms < o.SilenceRMS
			if isSilent {
				silentFrames++
				if silentFrames > maxSilentRunSeen {
					maxSilentRunSeen = silentFrames
				}
			} else {
				if !loggedFirstVoiced {
					loggedFirstVoiced = true
					log.Printf("[stt-overlap] first voiced frame: rms=%.0f threshold=%.0f frame_idx=%d",
						rms, o.SilenceRMS, nextFrame/frameBytes)
				}
				silentFrames = 0
				hasVoiced = true
			}
			nextFrame += frameBytes
			if hasVoiced && silentFrames >= silenceFramesNeeded {
				boundary := nextFrame
				log.Printf("[stt-overlap] silence boundary: silence_ms=%d threshold_ms=%d boundary_byte=%d uncommitted_ms=%d",
					silentFrames*o.FrameMs, o.SilenceMs, boundary,
					(boundary-committedAudioBytes)*1000/(o.SampleRate*sampleBytes))
				silentFrames = 0
				hasVoiced = false
				return boundary
			}
		}
		return -1
	}

	// forceCommitAll transcribes ALL uncommitted audio as a single
	// single-shot commit and advances the cursor to the end of the
	// buffer. This is the safety-net path for "no commit happened for
	// too long" (continuous speech with no silence boundary): rather
	// than dropping audio, we trust one transcription pass over the
	// whole uncommitted window. The egress hallucination filter
	// downstream still drops known false phrases. Returns true when a
	// force-commit was performed so the caller can skip the normal
	// iteration body.
	forceCommitAll := func() bool {
		if maxWindowBytes <= 0 || len(pcm)-committedAudioBytes <= maxWindowBytes {
			return false
		}
		uncommittedMs := (len(pcm) - committedAudioBytes) * 1000 / (o.SampleRate * sampleBytes)
		log.Printf("[overlap-agree] force commit: max_window_ms=%d exceeded, transcribing %dms of uncommitted audio (no silence boundary detected)",
			o.MaxWindowMs, uncommittedMs)
		n := len(pcm) - committedAudioBytes
		audio := make([]byte, n)
		copy(audio, pcm[committedAudioBytes:])
		res, err := transcribe(audio)
		if err != nil {
			// Transcribe genuinely failed — only path that actually
			// loses audio. Surface the error and advance the cursor
			// (otherwise we'd retry forever and the buffer would only
			// grow). The error event is the user-visible signal that
			// audio was lost.
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: fmt.Errorf(
				"overlap-agree: force-commit transcribe failed after max_window_ms=%d; %dms of audio unavailable: %w",
				o.MaxWindowMs, uncommittedMs, err)}
			committedAudioBytes = len(pcm)
			recent = nil
			lastAdvanced = true
			stallRejects = 0
			return true
		}
		if res.Text != "" {
			newCommit, tail, _ := appendAfterAdvance(committed, strings.TrimSpace(res.Text))
			if len(newCommit) > len(committed) && tail != "" {
				events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
					Text:             tail,
					DetectedLanguage: res.DetectedLanguage,
					ProviderTier:     res.Tier,
					ProviderID:       res.ProviderID,
					ModelID:          res.ModelID,
					LatencyMs:        float64(res.Latency.Milliseconds()),
					Confidence:       res.Confidence,
				}}
				committed = newCommit
				lastTier = res.Tier
				lastProviderID = res.ProviderID
				lastModelID = res.ModelID
			}
		}
		committedAudioBytes = len(pcm)
		recent = nil
		lastAdvanced = true
		stallRejects = 0
		return true
	}

	processIteration := func(rightEdge int) {
		// Safety net: when uncommitted audio exceeds MaxWindowMs (no
		// silence detected for too long), force-commit the whole
		// window via a single transcribe call rather than skipping
		// any audio. This guarantees the "never lose audio"
		// contract even under continuous speech.
		if forceCommitAll() {
			return
		}

		if rightEdge <= committedAudioBytes {
			return
		}
		n := rightEdge - committedAudioBytes
		if n < minWindowBytes {
			return
		}
		audio := make([]byte, n)
		copy(audio, pcm[committedAudioBytes:rightEdge])
		iterAudioMs := n * 1000 / (o.SampleRate * sampleBytes)
		log.Printf("[stt-overlap] settle attempt: audio_ms=%d cursor_byte=%d right_edge=%d recent=%d/%d last_advanced=%t",
			iterAudioMs, committedAudioBytes, rightEdge, len(recent), o.CommitRuns, lastAdvanced)
		res, err := transcribe(audio)
		nextTriggerAt = len(pcm) + advanceBytes
		if err != nil {
			// Transient provider failure: surface one Error and continue.
			// The growing buffer means the next iteration covers a
			// superset of the same audio, so a single failure is
			// self-healing.
			log.Printf("[stt-overlap] settle error: %v", err)
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventError, Error: err}
			return
		}
		lastTier = res.Tier
		lastProviderID = res.ProviderID
		lastModelID = res.ModelID

		recent = append(recent, hypothesis{text: res.Text, words: res.Words})
		if len(recent) > o.CommitRuns {
			recent = recent[len(recent)-o.CommitRuns:]
		}
		texts := make([]string, len(recent))
		for i, h := range recent {
			texts[i] = h.text
		}
		agreed := longestAgreedPrefix(texts, o.CommitRuns, o.MaxAgreedTokens)
		log.Printf("[stt-overlap] hypothesis: text=%q words=%d agreed=%q recent_now=%d",
			voice.TruncateForLog(res.Text, 80), len(res.Words),
			voice.TruncateForLog(agreed, 80), len(recent))

		// Two-mode merge based on lastAdvanced:
		//
		// - normal mode: hypotheses cover overlapping audio (cursor
		//   didn't advance last commit, or it's the very first
		//   commit). mergeAgreed's prefix/overlap/divergence logic
		//   applies — "no overlap with committed" means the model
		//   wandered, reject.
		//
		// - post-advance mode: cursor JUST moved forward via word
		//   timestamps; the next agreement is over genuinely new
		//   audio. appendAfterAdvance accepts the append cleanly
		//   (with prompt-regurg defense) instead of rejecting it.
		var newCommit, tail string
		var ok bool
		if lastAdvanced {
			newCommit, tail, ok = appendAfterAdvance(committed, agreed)
		} else {
			newCommit, tail, ok = mergeAgreed(committed, agreed)
		}
		if agreed != "" && !ok {
			stallRejects++
			// Stall-fallback: after MaxStallRejects consecutive
			// divergence-rejects, stop waiting for an agreement that
			// isn't coming and commit the freshest hypothesis tail. This
			// bounds tail growth / re-transcription cost well before the
			// MaxWindowMs net (the pathology this lever fixes). It commits
			// a best-guess — it never silently drops audio.
			if o.MaxStallRejects > 0 && stallRejects >= o.MaxStallRejects {
				log.Printf("[stt-overlap] stall-fallback: %d consecutive divergence-rejects >= max_stall_rejects=%d — force-committing freshest hypothesis tail=%q",
					stallRejects, o.MaxStallRejects, voice.TruncateForLog(res.Text, 80))
				if res.Text != "" {
					newCommit, tail, _ := appendAfterAdvance(committed, strings.TrimSpace(res.Text))
					if len(newCommit) > len(committed) && tail != "" {
						events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
							Text:             tail,
							DetectedLanguage: res.DetectedLanguage,
							ProviderTier:     res.Tier,
							ProviderID:       res.ProviderID,
							ModelID:          res.ModelID,
							LatencyMs:        float64(res.Latency.Milliseconds()),
							Confidence:       res.Confidence,
						}}
						committed = newCommit
						lastTier = res.Tier
						lastProviderID = res.ProviderID
						lastModelID = res.ModelID
					}
				}
				// Advance the cursor past the window we just force-
				// committed (its whole transcript is now committed), so
				// the next iteration starts on genuinely new audio.
				if rightEdge > committedAudioBytes && rightEdge <= len(pcm) {
					committedAudioBytes = rightEdge
				}
				recent = nil
				lastAdvanced = true
				stallRejects = 0
				return
			}
			log.Printf("[stt-overlap] divergence-reject: committed=%q agreed=%q (in-stream wander — no commit, stall=%d/%d)",
				voice.TruncateForLog(committed, 60), voice.TruncateForLog(agreed, 60), stallRejects, o.MaxStallRejects)
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: res.Text}}
			return
		}
		if len(newCommit) > len(committed) {
			log.Printf("[stt-overlap] commit: tail=%q committed_now=%q",
				voice.TruncateForLog(tail, 80), voice.TruncateForLog(newCommit, 100))
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
				Text:             tail,
				DetectedLanguage: res.DetectedLanguage,
				ProviderTier:     res.Tier,
				ProviderID:       res.ProviderID,
				ModelID:          res.ModelID,
				LatencyMs:        float64(res.Latency.Milliseconds()),
				Confidence:       res.Confidence,
			}}
			committed = newCommit
			// A forward commit happened — the model is making progress
			// again, so the consecutive-divergence streak resets.
			stallRejects = 0
			// Advance committedAudioBytes to the END time of the last
			// agreed word so the next iteration's audio starts where
			// the committed material ends. When word timestamps are
			// absent (non-Whisper backends or the scripted-text test
			// providers), the cursor stays put and mergeAgreed
			// continues to prevent re-emission — buffer growth is
			// bounded by the MaxWindowMs forced advance.
			if adv := wordEndBytes(res.Words, len(strings.Fields(agreed))); adv > 0 {
				newOffset := committedAudioBytes + adv
				if newOffset > len(pcm) {
					newOffset = len(pcm)
				}
				committedAudioBytes = newOffset
				// Stale pre-advance hypotheses would block future
				// agreement (they share no first-word with post-advance
				// hypotheses), so clear them and switch to
				// appendAfterAdvance for the next merge.
				recent = nil
				lastAdvanced = true
			} else {
				// No audio advance: keep the sliding window of
				// hypotheses so consecutive commits agree on growing
				// prefixes incrementally. Stay in normal-merge mode.
				lastAdvanced = false
			}
		} else if res.Text != "" {
			events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: res.Text}}
		}
	}

	for {
		select {
		case <-ctx.Done():
			emitDone()
			return ctx.Err()
		case ch, ok := <-chunks:
			if !ok {
				// Final transcribe over the remaining uncommitted tail.
				// This is the "last-chance" path: unsettled audio MUST
				// reach the user, otherwise long utterances that never
				// hit a clean agreement boundary silently lose
				// everything after the first commit.
				//
				// Strategy: transcribe the tail, run only the
				// prompt-regurgitation defense (not the full divergence
				// detector), and emit a Segment with whatever new
				// content remains. The egress hallucination filter
				// still drops known Whisper silence-hallucination
				// phrases downstream.
				if len(pcm)-committedAudioBytes > 0 {
					tailBytes := make([]byte, len(pcm)-committedAudioBytes)
					copy(tailBytes, pcm[committedAudioBytes:])
					res, err := transcribe(tailBytes)
					if err == nil && res.Text != "" {
						newCommit, tail, _ := appendAfterAdvance(committed, strings.TrimSpace(res.Text))
						if len(newCommit) > len(committed) && tail != "" {
							events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
								Text:         tail,
								ProviderTier: res.Tier,
								ProviderID:   res.ProviderID,
								ModelID:      res.ModelID,
								Confidence:   res.Confidence,
							}}
							committed = newCommit
						}
					}
				}
				emitDone()
				return nil
			}
			pcm = append(pcm, ch.Audio...)
			switch o.Trigger {
			case TriggerStopwatch:
				if len(pcm) >= nextTriggerAt && len(pcm)-committedAudioBytes >= minWindowBytes {
					processIteration(len(pcm))
				}
			default: // TriggerVAD
				// Drain every silence boundary that newly arrived.
				// Each one is its own settle attempt against a clean
				// right-edge — Whisper transcribes a chunk that ends
				// at the end of an utterance, not mid-word.
				for {
					boundary := scanVADBoundary()
					if boundary < 0 {
						break
					}
					processIteration(boundary)
				}
				// MaxWindowMs safety net: if voiced audio has piled
				// up without a silence (continuous speech), force a
				// settle to bound per-call latency. The forced-advance
				// branch inside processIteration handles the cursor
				// drop and logs an Error event.
				if maxWindowBytes > 0 && len(pcm)-committedAudioBytes > maxWindowBytes {
					processIteration(len(pcm))
				}
			}
		}
	}
}

func (o *OverlapAgree) applyDefaults() {
	if o.SampleRate == 0 {
		o.SampleRate = 16000
	}
	if o.WindowMs == 0 {
		o.WindowMs = 2000
	}
	if o.CommitRuns < 2 {
		o.CommitRuns = 2
	}
	if o.AdvanceMs == 0 {
		o.AdvanceMs = o.WindowMs / 2
	}
	if o.MaxWindowMs == 0 {
		o.MaxWindowMs = 25000
	}
	if o.MaxAgreedTokens == 0 {
		o.MaxAgreedTokens = 30
	}
	if o.Trigger == "" {
		o.Trigger = TriggerVAD
	}
	if o.SilenceMs == 0 {
		// OverlapAgree settles at SHORTER pauses than VADSegment by
		// design — the whole point of the strategy is to emit Segments
		// mid-utterance, not wait for end-of-turn. 500ms catches natural
		// inter-phrase pauses without firing inside words.
		o.SilenceMs = 500
	}
	if o.SilenceRMS == 0 {
		o.SilenceRMS = 250
	}
	if o.FrameMs == 0 {
		o.FrameMs = 20
	}
}

// mergeAgreed merges a newly-agreed run-prefix into the committed text.
//
// Returns (newCommit, tail, ok).
//
//   - ok=false: divergence — agreed is non-empty, does not extend committed
//     as a prefix, AND shares no word-suffix↔prefix overlap with
//     committed. Caller emits Partial only.
//   - newCommit > committed: caller emits Segment with `tail` and stores
//     newCommit.
//   - newCommit unchanged: caller may emit Partial.
//
// Three accepted cases (ok=true):
//
//  1. agreed=="" — no new agreement; returns committed, "", true.
//  2. committed is a prefix of agreed (the classic LocalAgreement
//     growth case) — returns the trimmed agreed text. The candidate
//     tail is run through DeduplicateOverlap before emitting because
//     Whisper sometimes regurgitates the initial_prompt (= committed)
//     at the start of its output; without dedupe the tail would be a
//     literal duplicate of committed.
//  3. committed's tail-words overlap agreed's head-words — happens when
//     the cursor didn't word-advance (no word timestamps) and the next
//     hypothesis covers the same audio. Delegates to
//     pipeline.DeduplicateOverlap.
func mergeAgreed(committed, agreed string) (newCommit, tail string, ok bool) {
	if agreed == "" {
		return committed, "", true
	}
	if committed == "" {
		nc := strings.TrimSpace(agreed)
		return nc, nc, true
	}
	if strings.HasPrefix(agreed, committed) {
		nc := strings.TrimSpace(agreed)
		if len(nc) <= len(committed) {
			return committed, "", true
		}
		candidate := nc[len(committed):]
		candidateTrimmed := strings.TrimSpace(candidate)
		if candidateTrimmed == "" {
			return committed, "", true
		}
		merged := voice.DeduplicateOverlap(committed, candidateTrimmed)
		if len(merged) <= len(committed) {
			return committed, "", true
		}
		return merged, strings.TrimPrefix(merged, committed), true
	}
	if strings.HasPrefix(committed, agreed) {
		return committed, "", true
	}
	merged := voice.DeduplicateOverlap(committed, agreed)
	if merged == committed+" "+agreed {
		return committed, "", false
	}
	if len(merged) <= len(committed) {
		return committed, "", true
	}
	return merged, strings.TrimPrefix(merged, committed), true
}

// appendAfterAdvance is the post-cursor-advance merge function.
//
// When committedAudioBytes has just moved forward via word-aligned
// advance, the next hypothesis covers GENUINELY NEW audio that
// follows what committed represents. mergeAgreed's divergence
// detector would reject this expected state ("no overlap, no
// prefix relationship → wander → REJECT"). appendAfterAdvance
// instead accepts the append, with only the prompt-regurgitation
// defense applied (Whisper occasionally echoes its initial_prompt
// at the start of post-advance output, which DeduplicateOverlap
// will strip).
//
// Returns (newCommit, tail, ok). ok is always true — the caller's
// divergence-reject branch is intentionally unreachable for
// post-advance commits.
//
// Three accepted shapes:
//
//  1. agreed == "" → no change.
//  2. committed == "" → first commit; trimmed agreed becomes
//     newCommit; tail equals newCommit.
//  3. otherwise → DeduplicateOverlap(committed, agreed) merges
//     prompt-regurg overlap if present; pure append otherwise.
//     tail is the new content beyond committed.
func appendAfterAdvance(committed, agreed string) (newCommit, tail string, ok bool) {
	if agreed == "" {
		return committed, "", true
	}
	if committed == "" {
		nc := strings.TrimSpace(agreed)
		return nc, nc, true
	}
	// Whisper can regurgitate the initial_prompt in two shapes:
	//
	//  (a) `committed + " " + newWords`   — full prefix echo
	//  (b) `lastFewCommittedWords + " " + newWords` — partial overlap
	//
	// And sometimes BOTH at once:
	//  `committed + " " + committed + " " + newWords` — full echo
	//  followed by another partial overlap on the second copy.
	//
	// Strip the prefix echo first (shape a), then dedupe overlap on
	// the remainder (shape b). This mirrors mergeAgreed's case-2
	// defense but unconditionally — no divergence rejection.
	candidate := strings.TrimSpace(agreed)
	if strings.HasPrefix(candidate, committed) {
		tail := strings.TrimSpace(candidate[len(committed):])
		if tail == "" {
			return committed, "", true
		}
		merged := voice.DeduplicateOverlap(committed, tail)
		if len(merged) <= len(committed) {
			return committed, "", true
		}
		return merged, strings.TrimPrefix(merged, committed), true
	}
	merged := voice.DeduplicateOverlap(committed, candidate)
	if len(merged) <= len(committed) {
		return committed, "", true
	}
	return merged, strings.TrimPrefix(merged, committed), true
}

// longestAgreedPrefix returns the longest token-level prefix common to
// every transcript in `runs`, when at least `commitRuns` runs are
// present. Token-level (whitespace-separated) so partial words don't
// commit until the next iteration confirms them.
//
// Comparison is normalized via pipeline.NormalizeToken (case-insensitive
// + trailing-punctuation stripped) because Whisper jitters
// capitalization and punctuation across calls even when the underlying
// audio is identical (it re-decides sentence boundaries as more audio
// arrives). Without normalization, "Hello world" vs "hello world."
// would yield zero agreement and the algorithm would never commit.
// The returned prefix uses the FIRST run's tokens verbatim so committed
// text preserves Whisper's chosen capitalization/punctuation rather
// than an arbitrary normalized form.
//
// When maxTokens > 0 the comparison is capped at that many tokens — the
// agreement walk only considers the first maxTokens positions. This
// bounds variance accumulation on long uncommitted buffers: each
// settle attempt asks Whisper to be self-consistent over a fixed text
// length regardless of utterance duration. maxTokens <= 0 means
// unbounded (entire prefix is considered).
func longestAgreedPrefix(runs []string, commitRuns int, maxTokens int) string {
	if len(runs) < commitRuns {
		return ""
	}
	tokens := make([][]string, len(runs))
	for i, r := range runs {
		tokens[i] = strings.Fields(r)
	}
	minLen := len(tokens[0])
	for _, t := range tokens {
		if len(t) < minLen {
			minLen = len(t)
		}
	}
	if maxTokens > 0 && minLen > maxTokens {
		minLen = maxTokens
	}
	var prefix []string
	for i := 0; i < minLen; i++ {
		head := voice.NormalizeToken(tokens[0][i])
		ok := true
		for j := 1; j < len(tokens); j++ {
			if voice.NormalizeToken(tokens[j][i]) != head {
				ok = false
				break
			}
		}
		if !ok {
			break
		}
		prefix = append(prefix, tokens[0][i])
	}
	return strings.Join(prefix, " ")
}
