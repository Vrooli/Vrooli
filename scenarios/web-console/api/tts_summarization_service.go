package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	defaultTTSSummarizeConcurrency = 2
	autoSummarizeFailureCooldown   = 30 * time.Second
)

var errTTSSummarizeCoolingDown = errors.New("summarization cooling down after a recent failure")

// errSummarizeBudgetInThink / errSummarizeTruncated / errSummarizeEmptyAfterStrip
// / errSummarizeTrulyEmpty are the four categorized empty-result sentinels used
// by the service and error-message helpers. Each names a distinct failure mode
// so logs and user-facing banners can say something actionable.
var (
	errSummarizeBudgetInThink   = summarizeError("budget exhausted inside <think>")
	errSummarizeTruncated       = summarizeError("response truncated (done_reason=length)")
	errSummarizeEmptyAfterStrip = summarizeError("empty after stripping <think> block")
	errSummarizeTrulyEmpty      = summarizeError("truly empty response from model")
)

type summarizeError string

func (e summarizeError) Error() string { return string(e) }

type TTSSummarizeRequest struct {
	EventID string
	Path    string
	Text    string
}

type TTSSummarizeResult struct {
	Summary    string
	Paragraphs []string
	Config     TTSSummarizeConfig
	ElapsedMs  int64
	// Diagnostics carried from the underlying TTSSummarizer response so the
	// unified tts-summarize log line can distinguish real empty responses from
	// token-budget-exhausted truncation.
	DoneReason string
	EvalCount  int
	RawLen     int
}

type ttsSummarizeFuture struct {
	done   chan struct{}
	result TTSSummarizeResult
	err    error
}

type TTSSummarizationService struct {
	summarizer *TTSSummarizer
	getConfig  func() TTSSummarizeConfig
	sem        chan struct{}

	mu          sync.Mutex
	inflight    map[string]*ttsSummarizeFuture
	autoBackoff map[string]time.Time
}

func NewTTSSummarizationService(summarizer *TTSSummarizer, getConfig func() TTSSummarizeConfig) *TTSSummarizationService {
	return &TTSSummarizationService{
		summarizer:  summarizer,
		getConfig:   getConfig,
		sem:         make(chan struct{}, defaultTTSSummarizeConcurrency),
		inflight:    make(map[string]*ttsSummarizeFuture),
		autoBackoff: make(map[string]time.Time),
	}
}

func (s *TTSSummarizationService) Summarize(ctx context.Context, req TTSSummarizeRequest) (TTSSummarizeResult, error) {
	if s == nil || s.summarizer == nil {
		return TTSSummarizeResult{}, errors.New("summarizer unavailable")
	}

	cfg := s.getConfig()
	if req.Path == "auto" {
		if blocked := s.autoBackoffUntil(req.EventID); !blocked.IsZero() && time.Now().Before(blocked) {
			return TTSSummarizeResult{Config: cfg}, errTTSSummarizeCoolingDown
		}
	}

	s.mu.Lock()
	if future, ok := s.inflight[req.EventID]; ok {
		s.mu.Unlock()
		return waitForTTSSummarizeFuture(ctx, future)
	}
	future := &ttsSummarizeFuture{done: make(chan struct{})}
	s.inflight[req.EventID] = future
	s.mu.Unlock()

	result, err := s.run(ctx, req, cfg)

	s.mu.Lock()
	delete(s.inflight, req.EventID)
	if err != nil && req.Path == "auto" && isDeadlineExceededError(err) {
		s.autoBackoff[req.EventID] = time.Now().Add(autoSummarizeFailureCooldown)
	} else if err == nil {
		delete(s.autoBackoff, req.EventID)
	}
	future.result = result
	future.err = err
	close(future.done)
	s.mu.Unlock()

	return result, err
}

func (s *TTSSummarizationService) autoBackoffUntil(eventID string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	until := s.autoBackoff[eventID]
	if !until.IsZero() && time.Now().After(until) {
		delete(s.autoBackoff, eventID)
		return time.Time{}
	}
	return until
}

func (s *TTSSummarizationService) run(ctx context.Context, req TTSSummarizeRequest, cfg TTSSummarizeConfig) (TTSSummarizeResult, error) {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return TTSSummarizeResult{Config: cfg}, ctx.Err()
	}
	defer func() { <-s.sem }()

	normalized := NormalizeTextForSpeech(req.Text)
	if strings.TrimSpace(normalized) == "" {
		return TTSSummarizeResult{Config: cfg}, errors.New("normalized text is empty")
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	resp, err := s.summarizer.Summarize(runCtx, normalized, cfg.Model, cfg.Level)
	elapsedMs := time.Since(started).Milliseconds()
	if err != nil {
		return TTSSummarizeResult{
			Config:     cfg,
			ElapsedMs:  elapsedMs,
			DoneReason: resp.DoneReason,
			EvalCount:  resp.EvalCount,
			RawLen:     len(resp.RawContent),
		}, err
	}

	summary := strings.TrimSpace(resp.Content)
	result := TTSSummarizeResult{
		Config:     cfg,
		ElapsedMs:  elapsedMs,
		DoneReason: resp.DoneReason,
		EvalCount:  resp.EvalCount,
		RawLen:     len(resp.RawContent),
	}
	if summary == "" {
		return result, classifyEmptySummary(resp)
	}

	result.Summary = summary
	result.Paragraphs = SplitIntoSpeechParagraphs(summary)
	return result, nil
}

// classifyEmptySummary picks the most descriptive sentinel error for an empty
// result so the unified error message and metrics can tell apart token-budget
// starvation from an actually-empty model response.
func classifyEmptySummary(resp TTSSummarizerResponse) error {
	raw := resp.RawContent
	startsInThink := strings.HasPrefix(raw, "<think>")
	hadThinkBlock := strings.Contains(raw, "<think>")
	truncated := resp.DoneReason == "length"

	switch {
	case truncated && startsInThink:
		return errSummarizeBudgetInThink
	case truncated:
		return errSummarizeTruncated
	case hadThinkBlock:
		// Closed think block but no content after strip.
		return errSummarizeEmptyAfterStrip
	default:
		return errSummarizeTrulyEmpty
	}
}

func waitForTTSSummarizeFuture(ctx context.Context, future *ttsSummarizeFuture) (TTSSummarizeResult, error) {
	select {
	case <-ctx.Done():
		return TTSSummarizeResult{}, ctx.Err()
	case <-future.done:
		return future.result, future.err
	}
}

func isDeadlineExceededError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "context deadline exceeded")
}

func summarizeErrorMessage(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errSummarizeBudgetInThink):
		return "Model spent its entire token budget on internal reasoning. Try a shorter input or a non-reasoning model."
	case errors.Is(err, errSummarizeTruncated):
		return "Model response was truncated before producing a summary. Increase the token budget or try a smaller level."
	case errors.Is(err, errSummarizeEmptyAfterStrip):
		return "Model produced only reasoning, no summary. Try a different model."
	case errors.Is(err, errSummarizeTrulyEmpty):
		return "Summarizer returned empty response."
	case errors.Is(err, errTTSSummarizeCoolingDown):
		return "Summarization is cooling down after a recent timeout"
	case isDeadlineExceededError(err):
		return "Summarization timed out before Ollama returned a result"
	default:
		return fmt.Sprintf("Summarization failed: %v", err)
	}
}
