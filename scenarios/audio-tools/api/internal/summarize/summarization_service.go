package summarize

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"audio-tools/internal/clock"
	"audio-tools/internal/text/normalizer"
)

const (
	defaultSummarizeConcurrency  = 2
	autoSummarizeFailureCooldown = 30 * time.Second
)

// ErrSummarizeCoolingDown is returned by SummarizationService.Summarize when
// the auto path is in backoff after a recent failure.
var ErrSummarizeCoolingDown = errors.New("summarization cooling down after a recent failure")

// ErrSummarizeBudgetInThink / ErrSummarizeTruncated / ErrSummarizeEmptyAfterStrip
// / ErrSummarizeTrulyEmpty are the four categorized empty-result sentinels used
// by the service and error-message helpers. Each names a distinct failure mode
// so logs and user-facing banners can say something actionable.
var (
	ErrSummarizeBudgetInThink   = summarizeError("budget exhausted inside <think>")
	ErrSummarizeTruncated       = summarizeError("response truncated (done_reason=length)")
	ErrSummarizeEmptyAfterStrip = summarizeError("empty after stripping <think> block")
	ErrSummarizeTrulyEmpty      = summarizeError("truly empty response from model")
)

type summarizeError string

func (e summarizeError) Error() string { return string(e) }

// SummarizeRequest is the input to SummarizationService.Summarize.
type SummarizeRequest struct {
	EventID string
	Path    string
	Text    string
}

// SummarizeResult carries the summary and diagnostic fields.
type SummarizeResult struct {
	Summary    string
	Paragraphs []string
	Config     SummarizeConfig
	ElapsedMs  int64
	// Diagnostics carried from the underlying Summarizer response so the
	// unified tts-summarize log line can distinguish real empty responses from
	// token-budget-exhausted truncation.
	DoneReason string
	EvalCount  int
	RawLen     int
}

type summarizeFuture struct {
	done   chan struct{}
	result SummarizeResult
	err    error
}

// SummarizationService coordinates Ollama summarization with cooldown,
// inflight deduplication, and empty-summary classification.
type SummarizationService struct {
	summarizer *Summarizer
	getConfig  func() SummarizeConfig
	sem        chan struct{}
	clk        clock.Clock

	mu          sync.Mutex
	inflight    map[string]*summarizeFuture
	autoBackoff map[string]time.Time
}

// NewSummarizationService constructs a service backed by the given summarizer
// and config accessor, using the system clock for backoff bookkeeping.
func NewSummarizationService(summarizer *Summarizer, getConfig func() SummarizeConfig) *SummarizationService {
	return NewSummarizationServiceWith(summarizer, getConfig, clock.System{})
}

// NewSummarizationServiceWith is the clock-injected constructor. Tests
// pass mocks.FakeClock to drive the autoBackoff TTL deterministically.
func NewSummarizationServiceWith(summarizer *Summarizer, getConfig func() SummarizeConfig, clk clock.Clock) *SummarizationService {
	if clk == nil {
		clk = clock.System{}
	}
	return &SummarizationService{
		summarizer:  summarizer,
		getConfig:   getConfig,
		sem:         make(chan struct{}, defaultSummarizeConcurrency),
		clk:         clk,
		inflight:    make(map[string]*summarizeFuture),
		autoBackoff: make(map[string]time.Time),
	}
}

func (s *SummarizationService) now() time.Time {
	if s.clk == nil {
		return clock.System{}.Now()
	}
	return s.clk.Now()
}

func (s *SummarizationService) Summarize(ctx context.Context, req SummarizeRequest) (SummarizeResult, error) {
	if s == nil || s.summarizer == nil {
		return SummarizeResult{}, errors.New("summarizer unavailable")
	}

	cfg := s.getConfig()
	if req.Path == "auto" {
		if blocked := s.autoBackoffUntil(req.EventID); !blocked.IsZero() && s.now().Before(blocked) {
			return SummarizeResult{Config: cfg}, ErrSummarizeCoolingDown
		}
	}

	s.mu.Lock()
	if future, ok := s.inflight[req.EventID]; ok {
		s.mu.Unlock()
		return waitForSummarizeFuture(ctx, future)
	}
	future := &summarizeFuture{done: make(chan struct{})}
	s.inflight[req.EventID] = future
	s.mu.Unlock()

	result, err := s.run(ctx, req, cfg)

	s.mu.Lock()
	delete(s.inflight, req.EventID)
	if err != nil && req.Path == "auto" && isDeadlineExceededError(err) {
		s.autoBackoff[req.EventID] = s.now().Add(autoSummarizeFailureCooldown)
	} else if err == nil {
		delete(s.autoBackoff, req.EventID)
	}
	future.result = result
	future.err = err
	close(future.done)
	s.mu.Unlock()

	return result, err
}

func (s *SummarizationService) autoBackoffUntil(eventID string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	until := s.autoBackoff[eventID]
	if !until.IsZero() && s.now().After(until) {
		delete(s.autoBackoff, eventID)
		return time.Time{}
	}
	return until
}

func (s *SummarizationService) run(ctx context.Context, req SummarizeRequest, cfg SummarizeConfig) (SummarizeResult, error) {
	select {
	case s.sem <- struct{}{}:
	case <-ctx.Done():
		return SummarizeResult{Config: cfg}, ctx.Err()
	}
	defer func() { <-s.sem }()

	normalized := normalizer.NormalizeTextForSpeech(req.Text)
	if strings.TrimSpace(normalized) == "" {
		return SummarizeResult{Config: cfg}, errors.New("normalized text is empty")
	}

	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := s.now()
	resp, err := s.summarizer.Summarize(runCtx, normalized, cfg.Model, cfg.Level)
	elapsedMs := s.now().Sub(started).Milliseconds()
	if err != nil {
		return SummarizeResult{
			Config:     cfg,
			ElapsedMs:  elapsedMs,
			DoneReason: resp.DoneReason,
			EvalCount:  resp.EvalCount,
			RawLen:     len(resp.RawContent),
		}, err
	}

	summary := strings.TrimSpace(resp.Content)
	result := SummarizeResult{
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
	result.Paragraphs = normalizer.SplitIntoSpeechParagraphs(summary)
	return result, nil
}

// classifyEmptySummary picks the most descriptive sentinel error for an empty
// result so the unified error message and metrics can tell apart token-budget
// starvation from an actually-empty model response.
func classifyEmptySummary(resp SummarizerResponse) error {
	raw := resp.RawContent
	startsInThink := strings.HasPrefix(raw, "<think>")
	hadThinkBlock := strings.Contains(raw, "<think>")
	truncated := resp.DoneReason == "length"

	switch {
	case truncated && startsInThink:
		return ErrSummarizeBudgetInThink
	case truncated:
		return ErrSummarizeTruncated
	case hadThinkBlock:
		// Closed think block but no content after strip.
		return ErrSummarizeEmptyAfterStrip
	default:
		return ErrSummarizeTrulyEmpty
	}
}

func waitForSummarizeFuture(ctx context.Context, future *summarizeFuture) (SummarizeResult, error) {
	select {
	case <-ctx.Done():
		return SummarizeResult{}, ctx.Err()
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

// SummarizeErrorMessage maps service errors to user-facing strings.
func SummarizeErrorMessage(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrSummarizeBudgetInThink):
		return "Model spent its entire token budget on internal reasoning. Try a shorter input or a non-reasoning model."
	case errors.Is(err, ErrSummarizeTruncated):
		return "Model response was truncated before producing a summary. Increase the token budget or try a smaller level."
	case errors.Is(err, ErrSummarizeEmptyAfterStrip):
		return "Model produced only reasoning, no summary. Try a different model."
	case errors.Is(err, ErrSummarizeTrulyEmpty):
		return "Summarizer returned empty response."
	case errors.Is(err, ErrSummarizeCoolingDown):
		return "Summarization is cooling down after a recent timeout"
	case isDeadlineExceededError(err):
		return "Summarization timed out before Ollama returned a result"
	default:
		return fmt.Sprintf("Summarization failed: %v", err)
	}
}
