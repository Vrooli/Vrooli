// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the RecommendationWorker which handles background extraction
// of recommendations from completed investigation runs. It runs as a background
// service that periodically:
// - Polls for investigation runs that need recommendation extraction
// - Extracts recommendations using the Ollama-based extractor
// - Caches results in the run record for instant UI access
// - Handles retries with exponential backoff on failure
//
// EXTRACTION LOOP:
//   1. Query for runs with recommendation_status = 'pending' or 'failed'
//   2. Claim the run atomically (prevents duplicate processing)
//   3. Extract recommendations using the extractor adapter
//   4. Update run with result (success) or increment attempts (failure)
//   5. Broadcast status change for real-time UI updates

package orchestration

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/recommendation"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
)

// AllowlistProvider provides access to the investigation tag allowlist.
// This abstraction allows the worker to get the current allowlist without
// depending on the full orchestrator or settings repository directly.
type AllowlistProvider interface {
	GetAllowlist(ctx context.Context) []domain.InvestigationTagRule
}

// SettingsAllowlistProvider wraps an InvestigationSettingsRepository to provide allowlist access.
type SettingsAllowlistProvider struct {
	settings repository.InvestigationSettingsRepository
}

// NewSettingsAllowlistProvider creates a new provider from an InvestigationSettingsRepository.
func NewSettingsAllowlistProvider(settings repository.InvestigationSettingsRepository) *SettingsAllowlistProvider {
	return &SettingsAllowlistProvider{settings: settings}
}

// GetAllowlist returns the current investigation tag allowlist from settings.
func (p *SettingsAllowlistProvider) GetAllowlist(ctx context.Context) []domain.InvestigationTagRule {
	if p.settings == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	settings, err := p.settings.Get(ctx)
	if err != nil || settings == nil {
		return domain.DefaultInvestigationTagAllowlist()
	}
	return domain.NormalizeInvestigationTagAllowlist(settings.InvestigationTagAllowlist)
}

// RecommendationWorkerConfig holds configuration for the extraction worker.
type RecommendationWorkerConfig struct {
	// Interval is how often to poll for pending extractions
	Interval time.Duration

	// MaxRetries is the maximum extraction attempts before marking as failed
	MaxRetries int

	// RetryBackoff is the base duration for exponential backoff
	RetryBackoff time.Duration

	// MaxConcurrent is the number of extractions to process at once (1 = serial)
	MaxConcurrent int

	// StaleTimeout is how long an extraction can be "extracting" before being
	// considered stuck and reset to pending. This handles worker crashes.
	StaleTimeout time.Duration

	// ExtractionTimeout is the maximum time allowed for a single extraction.
	// This prevents indefinite hangs when calling the LLM.
	ExtractionTimeout time.Duration
}

// DefaultRecommendationWorkerConfig returns sensible defaults.
// Serial processing (MaxConcurrent=1) avoids overloading local Ollama.
func DefaultRecommendationWorkerConfig() RecommendationWorkerConfig {
	return RecommendationWorkerConfig{
		Interval:          30 * time.Second,
		MaxRetries:        3,
		RetryBackoff:      1 * time.Minute,
		MaxConcurrent:     1,                // Serial processing to avoid overloading Ollama
		StaleTimeout:      5 * time.Minute,  // Recover stuck extractions after 5 minutes
		ExtractionTimeout: 3 * time.Minute,  // Max time for a single LLM extraction
	}
}

// RecommendationWorker processes recommendation extractions in the background.
type RecommendationWorker struct {
	runs      repository.RunRepository
	events    event.Store
	extractor recommendation.Extractor

	config RecommendationWorkerConfig

	// Allowlist provider for filtering eligible runs
	allowlistProvider AllowlistProvider

	// State
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	lastRunTime time.Time
	lastStats   RecommendationWorkerStats

	// Broadcaster for real-time updates
	broadcaster EventBroadcaster
}

// RecommendationWorkerStats contains statistics from an extraction cycle.
type RecommendationWorkerStats struct {
	Timestamp          time.Time
	Duration           time.Duration
	RunsProcessed      int
	ExtractionsSuccess int
	ExtractionsRetried int
	ExtractionsFailed  int
	Errors             []string
}

// NewRecommendationWorker creates a new worker with the given dependencies.
func NewRecommendationWorker(
	runs repository.RunRepository,
	events event.Store,
	extractor recommendation.Extractor,
	opts ...RecommendationWorkerOption,
) *RecommendationWorker {
	w := &RecommendationWorker{
		runs:      runs,
		events:    events,
		extractor: extractor,
		config:    DefaultRecommendationWorkerConfig(),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

// RecommendationWorkerOption configures the worker.
type RecommendationWorkerOption func(*RecommendationWorker)

// WithRecommendationWorkerConfig sets custom configuration.
func WithRecommendationWorkerConfig(cfg RecommendationWorkerConfig) RecommendationWorkerOption {
	return func(w *RecommendationWorker) {
		w.config = cfg
	}
}

// WithRecommendationWorkerBroadcaster sets the event broadcaster.
func WithRecommendationWorkerBroadcaster(b EventBroadcaster) RecommendationWorkerOption {
	return func(w *RecommendationWorker) {
		w.broadcaster = b
	}
}

// WithRecommendationWorkerAllowlist sets the allowlist provider.
// This is required for proper filtering of eligible investigation runs.
func WithRecommendationWorkerAllowlist(p AllowlistProvider) RecommendationWorkerOption {
	return func(w *RecommendationWorker) {
		w.allowlistProvider = p
	}
}

// Start begins the extraction loop.
func (w *RecommendationWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return domain.NewStateError("RecommendationWorker", "running", "start", "already running")
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.doneCh = make(chan struct{})
	w.mu.Unlock()

	// Seed the queue with existing unextracted investigation runs on startup
	seeded := w.seedExistingRuns(ctx)
	if seeded > 0 {
		log.Printf("[recommendation-worker] Seeded %d existing investigation runs for extraction", seeded)
	}

	go w.loop(ctx)
	log.Printf("[recommendation-worker] Started with interval=%v, maxRetries=%d",
		w.config.Interval, w.config.MaxRetries)
	return nil
}

// Stop gracefully stops the extraction loop.
func (w *RecommendationWorker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.mu.Unlock()

	close(w.stopCh)
	<-w.doneCh

	w.mu.Lock()
	w.running = false
	w.mu.Unlock()

	log.Printf("[recommendation-worker] Stopped")
	return nil
}

// IsRunning returns whether the worker is active.
func (w *RecommendationWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// LastStats returns the statistics from the last extraction cycle.
func (w *RecommendationWorker) LastStats() RecommendationWorkerStats {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.lastStats
}

// loop runs the extraction loop.
func (w *RecommendationWorker) loop(ctx context.Context) {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()

	// Run once immediately on startup
	stats := w.processQueue(ctx)
	w.updateStats(stats)

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := w.processQueue(ctx)
			w.updateStats(stats)
		}
	}
}

// getAllowlist returns the current investigation tag allowlist.
// Falls back to defaults if no provider is configured.
func (w *RecommendationWorker) getAllowlist(ctx context.Context) []domain.InvestigationTagRule {
	if w.allowlistProvider != nil {
		return w.allowlistProvider.GetAllowlist(ctx)
	}
	return domain.DefaultInvestigationTagAllowlist()
}

// isEligibleForExtraction checks if a run is eligible for recommendation extraction.
func (w *RecommendationWorker) isEligibleForExtraction(ctx context.Context, run *domain.Run) bool {
	allowlist := w.getAllowlist(ctx)
	return domain.MatchesInvestigationTag(run.Tag, allowlist)
}

// updateStats updates the last run statistics.
func (w *RecommendationWorker) updateStats(stats RecommendationWorkerStats) {
	w.mu.Lock()
	w.lastRunTime = stats.Timestamp
	w.lastStats = stats
	w.mu.Unlock()

	// Log summary if there was any activity
	if stats.RunsProcessed > 0 {
		log.Printf("[recommendation-worker] cycle: processed=%d success=%d retried=%d failed=%d errors=%d",
			stats.RunsProcessed, stats.ExtractionsSuccess, stats.ExtractionsRetried,
			stats.ExtractionsFailed, len(stats.Errors))
	}
}

// recoverStaleExtractions finds runs stuck in "extracting" status and resets them to pending.
// This handles cases where a worker crashed mid-extraction.
func (w *RecommendationWorker) recoverStaleExtractions(ctx context.Context) int {
	if w.config.StaleTimeout <= 0 {
		return 0
	}

	stale, err := w.runs.ListStaleExtractions(ctx, w.config.StaleTimeout, 50)
	if err != nil {
		log.Printf("[recommendation-worker] Failed to list stale extractions: %v", err)
		return 0
	}

	recovered := 0
	now := time.Now()
	for _, run := range stale {
		// Reset to pending for retry
		run.RecommendationStatus = domain.RecommendationStatusPending
		run.RecommendationQueuedAt = &now
		run.RecommendationError = "recovered from stale extraction (worker crash or timeout)"
		run.UpdatedAt = now

		if err := w.runs.Update(ctx, run); err != nil {
			log.Printf("[recommendation-worker] Failed to recover stale run %s: %v", run.ID, err)
			continue
		}

		recovered++
		log.Printf("[recommendation-worker] Recovered stale extraction for run %s (was extracting for >%v)",
			run.ID, w.config.StaleTimeout)

		// Broadcast status update
		if w.broadcaster != nil {
			w.broadcaster.BroadcastRunStatus(run)
		}
	}

	return recovered
}

// processQueue processes pending extraction requests.
func (w *RecommendationWorker) processQueue(ctx context.Context) RecommendationWorkerStats {
	start := time.Now()
	stats := RecommendationWorkerStats{Timestamp: start}

	// First, recover any stale extractions from crashed workers
	recovered := w.recoverStaleExtractions(ctx)
	if recovered > 0 {
		log.Printf("[recommendation-worker] Recovered %d stale extractions", recovered)
	}

	// Check if extractor is available
	if w.extractor == nil || !w.extractor.IsAvailable(ctx) {
		// Extractor not available, skip this cycle
		stats.Duration = time.Since(start)
		return stats
	}

	// Get pending extractions (database uses broad filter, we filter precisely here)
	// Request more than MaxConcurrent to account for filtering
	pending, err := w.runs.ListPendingRecommendationExtractions(ctx, w.config.MaxRetries, w.config.MaxConcurrent*10)
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to list pending: "+err.Error())
		stats.Duration = time.Since(start)
		return stats
	}

	// Filter runs using the configurable allowlist and limit to MaxConcurrent
	processed := 0
	for _, run := range pending {
		if processed >= w.config.MaxConcurrent {
			break
		}
		// Filter using the configurable allowlist (database query is intentionally broad)
		if !w.isEligibleForExtraction(ctx, run) {
			continue
		}
		if err := w.processRun(ctx, run, &stats); err != nil {
			stats.Errors = append(stats.Errors, err.Error())
		}
		processed++
	}

	stats.Duration = time.Since(start)
	return stats
}

// processRun processes a single run for recommendation extraction.
func (w *RecommendationWorker) processRun(ctx context.Context, run *domain.Run, stats *RecommendationWorkerStats) error {
	stats.RunsProcessed++

	// Atomically claim this run for extraction
	claimed, err := w.runs.ClaimRecommendationExtraction(ctx, run.ID)
	if err != nil {
		return err
	}
	if !claimed {
		// Another worker got it, skip
		stats.RunsProcessed--
		return nil
	}

	// Check retry backoff: skip if not enough time has passed since last attempt
	if run.RecommendationAttempts > 0 && run.RecommendationQueuedAt != nil {
		backoff := w.config.RetryBackoff * time.Duration(1<<(run.RecommendationAttempts-1))
		if time.Since(*run.RecommendationQueuedAt) < backoff {
			// Not enough time passed, reset to pending and skip
			run.RecommendationStatus = domain.RecommendationStatusPending
			run.UpdatedAt = time.Now()
			_ = w.runs.Update(ctx, run)
			return nil
		}
	}

	// Perform extraction
	result, extractErr := w.extractRecommendations(ctx, run)

	now := time.Now()
	run.RecommendationAttempts++
	run.UpdatedAt = now

	if extractErr != nil || (result != nil && !result.Success) {
		// Extraction failed
		if run.RecommendationAttempts >= w.config.MaxRetries {
			// Max retries exceeded - mark as permanently failed
			run.RecommendationStatus = domain.RecommendationStatusFailed
			if extractErr != nil {
				run.RecommendationError = extractErr.Error()
			} else if result != nil && result.Error != "" {
				run.RecommendationError = result.Error
			} else {
				run.RecommendationError = "extraction failed after max retries"
			}
			// Store partial result for fallback display
			if result != nil {
				run.RecommendationResult = result
			}
			stats.ExtractionsFailed++
		} else {
			// Will retry later
			run.RecommendationStatus = domain.RecommendationStatusPending
			run.RecommendationQueuedAt = &now // Reset backoff timer
			if extractErr != nil {
				run.RecommendationError = extractErr.Error()
			}
			stats.ExtractionsRetried++
		}
	} else {
		// Extraction succeeded
		run.RecommendationStatus = domain.RecommendationStatusComplete
		run.RecommendationResult = result
		run.RecommendationError = ""
		stats.ExtractionsSuccess++
	}

	// Save the updated run
	if err := w.runs.Update(ctx, run); err != nil {
		return err
	}

	// Broadcast status change for real-time UI updates
	if w.broadcaster != nil {
		w.broadcaster.BroadcastRunStatus(run)
	}

	return nil
}

// extractRecommendations extracts recommendations from an investigation run.
func (w *RecommendationWorker) extractRecommendations(ctx context.Context, run *domain.Run) (*domain.ExtractionResult, error) {
	// Get investigation text (prefer summary, fallback to events)
	text, source := w.getInvestigationText(ctx, run)

	if text == "" {
		return &domain.ExtractionResult{
			Success:       false,
			ExtractedFrom: source,
			Error:         "no investigation text found",
		}, nil
	}

	// Truncate if too long
	const maxLen = 10000
	if len(text) > maxLen {
		text = text[:maxLen]
	}

	// Create a timeout context for the extraction to prevent indefinite hangs
	extractCtx := ctx
	if w.config.ExtractionTimeout > 0 {
		var cancel context.CancelFunc
		extractCtx, cancel = context.WithTimeout(ctx, w.config.ExtractionTimeout)
		defer cancel()
	}

	// Call extractor adapter with timeout
	result, err := w.extractor.Extract(extractCtx, domain.ExtractionRequest{
		InvestigationText: text,
		MaxLength:         maxLen,
	})
	if err != nil {
		// Check if timeout occurred
		if extractCtx.Err() == context.DeadlineExceeded {
			return &domain.ExtractionResult{
				Success:       false,
				RawText:       text,
				ExtractedFrom: source,
				Error:         "extraction timed out after " + w.config.ExtractionTimeout.String(),
			}, nil
		}
		return nil, err
	}

	result.ExtractedFrom = source
	result.RawText = text
	return result, nil
}

// getInvestigationText extracts text from an investigation run.
// Returns (text, source) where source is "summary" or "events".
func (w *RecommendationWorker) getInvestigationText(ctx context.Context, run *domain.Run) (string, string) {
	// Try to get summary description from run metadata first
	if run.Summary != nil && run.Summary.Description != "" {
		return run.Summary.Description, "summary"
	}

	// Fallback: get events and concatenate message content
	if w.events == nil {
		return "", "events"
	}

	events, err := w.events.Get(ctx, run.ID, event.GetOptions{
		AfterSequence: -1,
		EventTypes:    []domain.RunEventType{domain.EventTypeMessage},
		Limit:         100,
	})
	if err != nil || len(events) == 0 {
		return "", "events"
	}

	// Concatenate assistant messages
	var builder strings.Builder
	for _, evt := range events {
		if evt.EventType == domain.EventTypeMessage {
			if msgData, ok := evt.Data.(*domain.MessageEventData); ok {
				if msgData.Role == "assistant" {
					if builder.Len() > 0 {
						builder.WriteString("\n\n")
					}
					builder.WriteString(msgData.Content)
				}
			}
		}
	}

	return builder.String(), "events"
}

// seedExistingRuns queues existing investigation runs that haven't had recommendations extracted.
// This is called on startup to ensure existing runs get processed.
// Limited to the most recent 100 runs to avoid overwhelming the queue on first startup.
func (w *RecommendationWorker) seedExistingRuns(ctx context.Context) int {
	const maxSeedLimit = 100

	// Find unextracted investigation runs (uses broad database filter)
	// We pass empty string to get all runs with "investigation" in the tag
	runs, err := w.runs.ListUnextractedInvestigationRuns(ctx, "", maxSeedLimit*2)
	if err != nil {
		log.Printf("[recommendation-worker] Failed to list unextracted runs: %v", err)
		return 0
	}

	if len(runs) == 0 {
		return 0
	}

	// Queue each eligible run for extraction by setting status to pending
	// Filter using the configurable allowlist (database query is intentionally broad)
	now := time.Now()
	seeded := 0
	for _, run := range runs {
		if seeded >= maxSeedLimit {
			break
		}
		// Filter using the configurable allowlist
		if !w.isEligibleForExtraction(ctx, run) {
			continue
		}

		run.RecommendationStatus = domain.RecommendationStatusPending
		run.RecommendationQueuedAt = &now
		run.UpdatedAt = now

		if err := w.runs.Update(ctx, run); err != nil {
			log.Printf("[recommendation-worker] Failed to queue run %s for extraction: %v", run.ID, err)
			continue
		}
		seeded++

		// Broadcast the status update so UI knows extraction is queued
		if w.broadcaster != nil {
			w.broadcaster.BroadcastRunStatus(run)
		}
	}

	return seeded
}
