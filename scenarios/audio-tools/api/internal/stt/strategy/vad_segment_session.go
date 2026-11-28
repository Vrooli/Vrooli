package strategy

import (
	"context"
	"strings"
	"sync"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/logx"
	voice "audio-tools/internal/stt/pipeline"

	"github.com/vrooli/api-core/schedule"
)

// vadRunState owns all mutable data for one Run call. VADSegmenter itself is
// reusable configuration; this state is deliberately never shared by sessions.
type vadRunState struct {
	v      *VADSegmenter
	ctx    context.Context
	start  sttchain.StreamStart
	chunks <-chan sttchain.AudioChunk
	events chan<- sttchain.StreamEvent
	cursor *sttchain.ConsumptionCursor
	logger logx.Logger
	clk    schedule.Clock

	frameBytes, silenceFramesNeeded int
	silenceTimeoutMs                int64
	preRollBytes, trailingPadBytes  int
	maxTextOverlapWords             int

	buf            []byte
	bufStartSample int64
	segStart       int
	nextFrame      int
	silentFrames   int
	hasVoiced      bool

	committed, lastPromptText        string
	segmentOrdinal                   int
	lastTier                         sttchain.ProviderTier
	lastProviderID, lastModelID      string
	totalLatencyMs                   float64
	lastPreviewEnd, chunkCount       int
	retryStepBytes, nextFlushAttempt int

	tickSeq           uint64
	lastEmitFrameIdx  int
	lastEmittedVoiced bool
	anyTickEmitted    bool
	firstVoicedLogged bool

	previewWG            sync.WaitGroup
	previewSlot          chan struct{}
	previewRetry         chan struct{}
	previewIntervalBytes int
	previewWindowBytes   int
}

func newVADRunState(ctx context.Context, v *VADSegmenter, start sttchain.StreamStart, chunks <-chan sttchain.AudioChunk, events chan<- sttchain.StreamEvent, cursor *sttchain.ConsumptionCursor, logger logx.Logger) *vadRunState {
	const sampleBytes = 2
	frameBytes := v.SampleRate * v.FrameMs / 1000 * sampleBytes
	silenceFramesNeeded := v.SilenceMs / v.FrameMs
	if silenceFramesNeeded < 1 {
		silenceFramesNeeded = 1
	}
	maxTextOverlapWords := (v.PreRollMs + v.TrailingPadMs + 99) / 100
	if maxTextOverlapWords < 1 {
		maxTextOverlapWords = 1
	}
	retryStepBytes := v.SampleRate / 4 * sampleBytes
	if retryStepBytes <= 0 {
		retryStepBytes = frameBytes
	}
	clk := v.Clock
	if clk == nil {
		clk = schedule.System()
	}
	return &vadRunState{
		v: v, ctx: ctx, start: start, chunks: chunks, events: events, cursor: cursor, logger: logger, clk: clk,
		frameBytes: frameBytes, silenceFramesNeeded: silenceFramesNeeded,
		silenceTimeoutMs:    int64(silenceFramesNeeded * v.FrameMs),
		preRollBytes:        v.SampleRate * v.PreRollMs / 1000 * sampleBytes,
		trailingPadBytes:    v.SampleRate * v.TrailingPadMs / 1000 * sampleBytes,
		maxTextOverlapWords: maxTextOverlapWords, retryStepBytes: retryStepBytes,
		previewSlot: make(chan struct{}, 1), previewRetry: make(chan struct{}, 1),
		previewIntervalBytes: v.SampleRate * v.PreviewIntervalMs / 1000 * sampleBytes,
		previewWindowBytes:   v.SampleRate * v.PreviewWindowMs / 1000 * sampleBytes,
	}
}

// maybePreview launches one bounded, replaceable hypothesis asynchronously.
func (s *vadRunState) maybePreview() {
	window, ok := s.nextPreviewWindow()
	if !ok {
		return
	}
	req := s.previewRequest(window)
	s.previewWG.Add(1)
	go s.transcribePreview(req)
}

// nextPreviewWindow preserves the interval, retry, and single-flight gates.
func (s *vadRunState) nextPreviewWindow() ([]byte, bool) {
	if s.v.PreviewIntervalMs <= 0 {
		return nil, false
	}
	end := len(s.buf)
	retry := false
	select {
	case <-s.previewRetry:
		retry = true
	default:
	}
	if !retry && end-s.lastPreviewEnd < s.previewIntervalBytes {
		return nil, false
	}
	select {
	case s.previewSlot <- struct{}{}:
	default:
		return nil, false
	}
	s.lastPreviewEnd = end
	windowStart := s.segStart
	if s.previewWindowBytes > 0 && end-windowStart > s.previewWindowBytes {
		windowStart = end - s.previewWindowBytes
	}
	window := make([]byte, end-windowStart)
	copy(window, s.buf[windowStart:end])
	return window, true
}

func (s *vadRunState) previewRequest(window []byte) sttchain.Request {
	prompt := s.start.InitialPrompt
	if s.v.InitialPromptWords > 0 && s.lastPromptText != "" {
		prompt = strings.TrimSpace(prompt + " " + voice.LastNWords(s.lastPromptText, s.v.InitialPromptWords))
	}
	return sttchain.Request{
		Audio: window, Format: s.start.InputFormat, Language: s.start.Language, InitialPrompt: prompt,
		SkipSpeakerVerification: s.start.SkipSpeakerVerification, VADFilter: false,
		BYOKProvider: s.start.BYOKProvider, BYOKKey: s.start.BYOKKey, LPBSToken: s.start.LPBSToken, UserIdentity: s.start.UserIdentity,
	}
}

func (s *vadRunState) transcribePreview(req sttchain.Request) {
	defer s.previewWG.Done()
	defer func() { <-s.previewSlot }()
	res, err := s.v.Provider.Transcribe(s.ctx, req)
	if err != nil || res == nil {
		return
	}
	text := strings.TrimSpace(res.Text)
	if text == "" {
		select {
		case s.previewRetry <- struct{}{}:
		default:
		}
		return
	}
	select {
	case s.events <- sttchain.StreamEvent{Kind: sttchain.StreamEventPartial, Partial: &sttchain.PartialEvent{Text: text}}:
	case <-s.ctx.Done():
	}
}

func (s *vadRunState) drainPreviews() { s.previewWG.Wait() }
