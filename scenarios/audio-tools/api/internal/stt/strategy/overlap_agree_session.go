package strategy

import (
	"context"
	"strings"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	voice "audio-tools/internal/stt/pipeline"

	"github.com/vrooli/api-core/schedule"
)

// overlapAgreeSession owns every mutable value for one Run invocation. A
// strategy can therefore serve concurrent streams without sharing audio,
// cursors, agreement history, or provider attribution.
type overlapAgreeSession struct {
	strategy *OverlapAgree
	ctx      context.Context
	start    sttchain.StreamStart
	chunks   <-chan sttchain.AudioChunk
	events   chan<- sttchain.StreamEvent
	cursor   *sttchain.ConsumptionCursor
	logger   logx.Logger

	pcm                         []byte
	committedAudioBytes         int
	baseSample                  int64
	hasAudioBase                bool
	committed                   string
	recent                      []hypothesis
	lastAdvanced                bool
	stallRejects                int
	forceRetryAfterBytes        int
	nextTriggerAt               int
	lastTier                    sttchain.ProviderTier
	lastProviderID, lastModelID string
	totalLatencyMs              float64
	thresholds                  windowThresholds
	vadScan                     vadBoundaryScanner
}

const overlapSampleBytes = 2

func newOverlapAgreeSession(ctx context.Context, o *OverlapAgree, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent, cursor *sttchain.ConsumptionCursor, logger logx.Logger) *overlapAgreeSession {
	thresholds := calculateWindowThresholds(o.SampleRate, o.AdvanceMs, o.WindowMs, o.MaxWindowMs)
	nextTriggerAt := thresholds.advanceBytes
	if thresholds.minWindowBytes > nextTriggerAt {
		nextTriggerAt = thresholds.minWindowBytes
	}
	frameBytes := o.SampleRate * o.FrameMs / 1000 * overlapSampleBytes
	silenceFramesNeeded := o.SilenceMs / o.FrameMs
	if silenceFramesNeeded < 1 {
		silenceFramesNeeded = 1
	}
	return &overlapAgreeSession{
		strategy: o, ctx: ctx, start: start, chunks: chunks, events: events, cursor: cursor, logger: logger,
		nextTriggerAt: nextTriggerAt, thresholds: thresholds,
		vadScan: vadBoundaryScanner{frameBytes: frameBytes, silenceFramesNeeded: silenceFramesNeeded, frameMs: o.FrameMs, silenceRMS: o.SilenceRMS},
	}
}

func (s *overlapAgreeSession) run() error {
	o := s.strategy
	s.logger.Printf("[stt-overlap] session start: trigger=%s window_ms=%d advance_ms=%d commit_runs=%d max_window_ms=%d max_stall_rejects=%d silence_ms=%d silence_rms=%.0f frame_ms=%d max_agreed_tokens=%d sample_rate=%d",
		o.Trigger, o.WindowMs, o.AdvanceMs, o.CommitRuns, o.MaxWindowMs, o.MaxStallRejects, o.SilenceMs, o.SilenceRMS, o.FrameMs, o.MaxAgreedTokens, o.SampleRate)
	defer s.logger.Printf("[stt-overlap] session end")
	for {
		select {
		case <-s.ctx.Done():
			if len(s.pcm)-s.committedAudioBytes > 0 {
				if flushCtx, cancel, ok := bestEffortTailContext(s.ctx); ok {
					absoluteStartSample := s.baseSample + int64(s.committedAudioBytes/overlapSampleBytes)
					s.committed = flushOverlapTail(flushCtx, s.pcm, s.committedAudioBytes, absoluteStartSample, s.committed, s.vadScan.hasVoiced, s.transcribeWith, s.events)
					cancel()
				}
			}
			s.emitDone()
			return s.ctx.Err()
		case ch, ok := <-s.chunks:
			if !ok {
				absoluteStartSample := s.baseSample + int64(s.committedAudioBytes/overlapSampleBytes)
				s.committed = flushOverlapTail(s.ctx, s.pcm, s.committedAudioBytes, absoluteStartSample, s.committed, s.vadScan.hasVoiced, s.transcribeWith, s.events)
				s.emitDone()
				return nil
			}
			s.cursor.Observe(ch)
			if !s.hasAudioBase {
				s.baseSample = ch.StartSample
				s.hasAudioBase = true
			}
			s.pcm = append(s.pcm, ch.Audio...)
			s.processAudioTriggers()
		}
	}
}

func (s *overlapAgreeSession) emitDone() {
	s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventDone, Done: &sttchain.DoneEvent{FinalText: s.committed, LockedTier: s.lastTier, ProviderID: s.lastProviderID, ModelID: s.lastModelID, LatencyMs: s.totalLatencyMs}}
}

func (s *overlapAgreeSession) transcribeWith(callCtx context.Context, audio []byte) (*sttchain.Result, error) {
	prompt := s.start.InitialPrompt
	if s.committed != "" {
		prompt = strings.TrimSpace(prompt + " " + voice.LastNWords(s.committed, maxPromptOverlapWords))
	}
	req := sttchain.Request{
		Audio: audio, Format: s.start.InputFormat, Language: s.start.Language, InitialPrompt: prompt,
		SkipSpeakerVerification: s.start.SkipSpeakerVerification, VADFilter: s.start.VADFilter, BYOKProvider: s.start.BYOKProvider,
		BYOKKey: s.start.BYOKKey, LPBSToken: s.start.LPBSToken, UserIdentity: s.start.UserIdentity,
	}
	clk := s.strategy.Clock
	if clk == nil {
		clk = schedule.System()
	}
	t0 := clk.Now()
	res, err := s.strategy.Provider.Transcribe(callCtx, req)
	s.totalLatencyMs += float64(clk.Now().Sub(t0).Milliseconds())
	return res, err
}

func (s *overlapAgreeSession) transcribe(audio []byte) (*sttchain.Result, error) {
	return s.transcribeWith(s.ctx, audio)
}

func (s *overlapAgreeSession) emitSegment(text string, result *sttchain.Result) {
	s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventSegment, Segment: &sttchain.SegmentEvent{
		Text:             text,
		StartSample:      s.baseSample + int64(s.committedAudioBytes/overlapSampleBytes),
		Words:            timedWordsForUniqueSpan(text, result.Words),
		DetectedLanguage: result.DetectedLanguage, ProviderTier: result.Tier, ProviderID: result.ProviderID, ModelID: result.ModelID,
		LatencyMs: float64(result.Latency.Milliseconds()), Confidence: result.Confidence,
	}}
}

func (s *overlapAgreeSession) wordEndBytes(words []sttchain.TimedWord, n int) int {
	if n <= 0 || n > len(words) {
		return -1
	}
	endSec := words[n-1].End
	if endSec <= 0 {
		return -1
	}
	return int(endSec*float64(s.strategy.SampleRate)) * overlapSampleBytes
}

func (s *overlapAgreeSession) scanVADBoundary() int {
	o := s.strategy
	return s.vadScan.scan(s.pcm, func(rms float64, frameIndex int) {
		s.logger.Printf("[stt-overlap] first voiced frame: rms=%.0f threshold=%.0f frame_idx=%d", rms, o.SilenceRMS, frameIndex)
	}, func(boundary, silentFrames int) {
		s.logger.Printf("[stt-overlap] silence boundary: silence_ms=%d threshold_ms=%d boundary_byte=%d uncommitted_ms=%d",
			silentFrames*s.vadScan.frameMs, o.SilenceMs, boundary, (boundary-s.committedAudioBytes)*1000/(o.SampleRate*overlapSampleBytes))
	})
}

func (s *overlapAgreeSession) compactConsumed() {
	if s.committedAudioBytes <= 0 || s.committedAudioBytes > len(s.pcm) || s.committedAudioBytes < s.strategy.SampleRate*overlapSampleBytes {
		return
	}
	prefix := s.committedAudioBytes
	remaining := len(s.pcm) - prefix
	compacted := make([]byte, remaining)
	copy(compacted, s.pcm[prefix:])
	s.pcm = compacted
	s.baseSample += int64(prefix / overlapSampleBytes)
	s.committedAudioBytes = 0
	s.vadScan.nextFrame -= prefix
	if s.vadScan.nextFrame < 0 {
		s.vadScan.nextFrame = 0
	}
	if s.vadScan.nextFrame > len(s.pcm) {
		s.vadScan.nextFrame = len(s.pcm)
	}
	s.nextTriggerAt -= prefix
	if s.nextTriggerAt < 0 {
		s.nextTriggerAt = 0
	}
	if maxFrames := s.vadScan.nextFrame / s.vadScan.frameBytes; s.vadScan.silentFrames > maxFrames {
		s.vadScan.silentFrames = maxFrames
	}
}

func (s *overlapAgreeSession) processAudioTriggers() {
	o := s.strategy
	processAudioTriggers(o.Trigger, len(s.pcm), s.committedAudioBytes, s.nextTriggerAt,
		s.thresholds.minWindowBytes, s.thresholds.maxWindowBytes, s.scanVADBoundary, s.processIteration)
}

func (s *overlapAgreeSession) forceCommit() forceCommitState {
	return forceCommitState{
		strategy: s.strategy, pcm: &s.pcm, committedAudioBytes: &s.committedAudioBytes,
		maxWindowBytes: s.thresholds.maxWindowBytes, forceRetryAfterBytes: &s.forceRetryAfterBytes,
		forceRetryStepBytes: s.thresholds.forceRetryStepBytes, committed: &s.committed, recent: &s.recent,
		lastAdvanced: &s.lastAdvanced, stallRejects: &s.stallRejects, lastTier: &s.lastTier,
		lastProviderID: &s.lastProviderID, lastModelID: &s.lastModelID, logger: s.logger, events: s.events,
		sampleBytes: overlapSampleBytes, transcribe: s.transcribe, emitSegment: s.emitSegment, compactConsumed: s.compactConsumed,
	}
}

// processIteration handles one settle attempt while keeping every state update
// on the owning session; event order matches the former Run-local closure.
func (s *overlapAgreeSession) processIteration(rightEdge int) {
	if s.forceCommit().run() {
		return
	}
	if s.thresholds.maxWindowBytes > 0 {
		uncommittedBytes := len(s.pcm) - s.committedAudioBytes
		if uncommittedBytes > s.thresholds.maxWindowBytes && s.forceRetryAfterBytes > 0 && uncommittedBytes < s.forceRetryAfterBytes {
			return
		}
	}
	if rightEdge <= s.committedAudioBytes {
		return
	}
	n := rightEdge - s.committedAudioBytes
	if n < s.thresholds.minWindowBytes {
		return
	}
	audio := append([]byte(nil), s.pcm[s.committedAudioBytes:rightEdge]...)
	iterAudioMs := n * 1000 / (s.strategy.SampleRate * overlapSampleBytes)
	s.logger.Printf("[stt-overlap] settle attempt: audio_ms=%d cursor_byte=%d right_edge=%d recent=%d/%d last_advanced=%t",
		iterAudioMs, s.committedAudioBytes, rightEdge, len(s.recent), s.strategy.CommitRuns, s.lastAdvanced)
	settle := settleAttemptState{
		pcm: s.pcm, advanceBytes: s.thresholds.advanceBytes, nextTriggerAt: &s.nextTriggerAt,
		transcribe: s.transcribe, logger: s.logger, events: s.events,
	}
	res, ok := settle.run(audio, n)
	if !ok {
		return
	}
	s.lastTier, s.lastProviderID, s.lastModelID = res.Tier, res.ProviderID, res.ModelID
	postForce := cleanPostForceBoundary{
		strategy: s.strategy, pcm: s.pcm, rightEdge: rightEdge,
		committedAudioBytes: &s.committedAudioBytes, committed: &s.committed, result: res,
		forceRetryAfterBytes: &s.forceRetryAfterBytes, events: s.events, emitSegment: s.emitSegment,
		compactConsumed: s.compactConsumed, recent: &s.recent, lastAdvanced: &s.lastAdvanced, stallRejects: &s.stallRejects,
	}
	if postForce.handle() {
		return
	}
	s.recent = append(s.recent, hypothesis{text: res.Text, words: res.Words})
	if len(s.recent) > s.strategy.CommitRuns {
		s.recent = s.recent[len(s.recent)-s.strategy.CommitRuns:]
	}
	texts := make([]string, len(s.recent))
	for i, h := range s.recent {
		texts[i] = h.text
	}
	agreed := longestAgreedPrefix(texts, s.strategy.CommitRuns, s.strategy.MaxAgreedTokens)
	s.logger.Printf("[stt-overlap] hypothesis: text=%q words=%d agreed=%q recent_now=%d",
		voice.TruncateForLog(res.Text, 80), len(res.Words), voice.TruncateForLog(agreed, 80), len(s.recent))
	agreement := agreementCommitState{
		strategy: s.strategy, pcm: s.pcm, rightEdge: rightEdge,
		committedAudioBytes: &s.committedAudioBytes, committed: &s.committed, recent: &s.recent,
		lastAdvanced: &s.lastAdvanced, stallRejects: &s.stallRejects, forceRetryAfterBytes: &s.forceRetryAfterBytes,
		lastTier: &s.lastTier, lastProviderID: &s.lastProviderID, lastModelID: &s.lastModelID,
		logger: s.logger, events: s.events, result: res, agreed: agreed, emitSegment: s.emitSegment,
		wordEndBytes: s.wordEndBytes, compactConsumed: s.compactConsumed,
	}
	agreement.handle()
}
