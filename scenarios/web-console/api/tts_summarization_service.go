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
		summarizer: summarizer,
		getConfig:  getConfig,
		sem:        make(chan struct{}, defaultTTSSummarizeConcurrency),
		inflight:   make(map[string]*ttsSummarizeFuture),
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
	summary, err := s.summarizer.Summarize(runCtx, normalized, cfg.Model, cfg.Level)
	elapsedMs := time.Since(started).Milliseconds()
	if err != nil {
		return TTSSummarizeResult{
			Config:    cfg,
			ElapsedMs: elapsedMs,
		}, err
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		return TTSSummarizeResult{
			Config:    cfg,
			ElapsedMs: elapsedMs,
		}, emptySummaryErr
	}

	return TTSSummarizeResult{
		Summary:    summary,
		Paragraphs: SplitIntoSpeechParagraphs(summary),
		Config:     cfg,
		ElapsedMs:  elapsedMs,
	}, nil
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
	case errors.Is(err, emptySummaryErr):
		return "Summarizer returned empty result"
	case errors.Is(err, errTTSSummarizeCoolingDown):
		return "Summarization is cooling down after a recent timeout"
	case isDeadlineExceededError(err):
		return "Summarization timed out before Ollama returned a result"
	default:
		return fmt.Sprintf("Summarization failed: %v", err)
	}
}
