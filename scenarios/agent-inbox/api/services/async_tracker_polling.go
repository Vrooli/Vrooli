package services

import (
	"context"
	"log"
	"time"
)

// snapshotOperation creates a read-only snapshot of immutable operation fields.
// Call this at the start of pollLoop to avoid repeated lock acquisitions.
func (s *AsyncTrackerService) snapshotOperation(toolCallID string) (*OperationSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	op, ok := s.operations[toolCallID]
	if !ok || op == nil {
		return nil, false
	}

	snap := &OperationSnapshot{
		ToolCallID:    op.ToolCallID,
		ChatID:        op.ChatID,
		ToolName:      op.ToolName,
		Scenario:      op.Scenario,
		ExternalRunID: op.ExternalRunID,
		AsyncBehavior: op.AsyncBehavior, // Pointer to immutable proto struct
		StartedAt:     op.StartedAt,
	}

	// Extract backoff configuration from proto (use defaults if not configured)
	snap.BackoffInitial = DefaultPollInterval
	snap.BackoffMax = DefaultPollInterval // No backoff by default
	snap.BackoffMultiplier = 1.0          // No backoff by default

	if op.AsyncBehavior != nil && op.AsyncBehavior.StatusPolling != nil {
		polling := op.AsyncBehavior.StatusPolling

		// Use configured base interval if valid
		if polling.PollIntervalSeconds > 0 {
			interval := time.Duration(polling.PollIntervalSeconds) * time.Second
			if interval >= MinPollInterval {
				snap.BackoffInitial = interval
				snap.BackoffMax = interval // Default max to initial if no backoff config
			}
		}

		// Apply backoff config if present
		if backoff := polling.GetBackoff(); backoff != nil {
			if backoff.InitialIntervalSeconds > 0 {
				initial := time.Duration(backoff.InitialIntervalSeconds) * time.Second
				if initial >= MinPollInterval {
					snap.BackoffInitial = initial
				}
			}
			if backoff.MaxIntervalSeconds > 0 {
				snap.BackoffMax = time.Duration(backoff.MaxIntervalSeconds) * time.Second
			}
			if backoff.Multiplier >= 1.0 {
				snap.BackoffMultiplier = float64(backoff.Multiplier)
			}
		}
	}

	return snap, true
}

// pollLoop runs the background polling for an operation.
// Uses OperationSnapshot for immutable config to avoid race conditions.
// Implements exponential backoff when configured via AsyncBehavior.StatusPolling.Backoff.
func (s *AsyncTrackerService) pollLoop(ctx context.Context, op *AsyncOperation) {
	// Snapshot immutable fields at the start to avoid repeated lock acquisitions
	// and potential race conditions when reading config.
	snap, ok := s.snapshotOperation(op.ToolCallID)
	if !ok {
		log.Printf("[ERROR] pollLoop: operation not found for %s", op.ToolCallID)
		return
	}

	polling := snap.AsyncBehavior.StatusPolling
	conditions := snap.AsyncBehavior.CompletionConditions

	// Use configured max duration, with reasonable default
	maxDuration := time.Duration(polling.MaxPollDurationSeconds) * time.Second
	if maxDuration <= 0 {
		maxDuration = DefaultMaxPollDuration
	}

	deadline := snap.StartedAt.Add(maxDuration)

	// Initialize dynamic interval for exponential backoff
	// Backoff config is pre-extracted in snapshotOperation for thread-safe access
	currentInterval := snap.BackoffInitial

	// Log backoff configuration if enabled
	if snap.BackoffMultiplier > 1.0 {
		log.Printf("[DEBUG] pollLoop: starting with backoff for %s (initial=%v, max=%v, multiplier=%.2f)",
			snap.ToolCallID, snap.BackoffInitial, snap.BackoffMax, snap.BackoffMultiplier)
	}

	// Use timer instead of ticker for dynamic intervals
	timer := time.NewTimer(currentInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Polling cancelled for %s", snap.ToolCallID)
			return
		case <-timer.C:
			if time.Now().After(deadline) {
				s.handleTimeout(op)
				return
			}

			// Call the status tool using snapshot for immutable config
			statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
			if err != nil {
				log.Printf("Error polling status for %s: %v", snap.ToolCallID, err)

				// Track consecutive errors
				s.mu.Lock()
				op.ConsecutiveErrors++
				op.LastPollError = err.Error()
				errorCount := op.ConsecutiveErrors
				s.mu.Unlock()

				// Push error update to UI after 2+ consecutive failures
				if errorCount >= 2 {
					s.pushPollErrorUpdate(snap.ChatID, snap.ToolCallID, err.Error(), errorCount)
				}

				// Continue polling despite error - reset timer with current interval
				timer.Reset(currentInterval)
				continue
			}

			// Reset error count on successful poll
			s.mu.Lock()
			op.ConsecutiveErrors = 0
			op.LastPollError = ""
			s.mu.Unlock()

			// Process the status result
			terminal, status := s.processStatusResult(op, statusResult, conditions)
			if terminal {
				log.Printf("Operation %s reached terminal status: %s", snap.ToolCallID, status)
				return
			}

			// Calculate next interval with exponential backoff
			if snap.BackoffMultiplier > 1.0 {
				nextInterval := time.Duration(float64(currentInterval) * snap.BackoffMultiplier)
				if nextInterval > snap.BackoffMax {
					nextInterval = snap.BackoffMax
				}
				if nextInterval != currentInterval {
					log.Printf("[DEBUG] pollLoop: backoff %s interval %v -> %v",
						snap.ToolCallID, currentInterval, nextInterval)
				}
				currentInterval = nextInterval
			}

			// Reset timer with (potentially new) interval
			timer.Reset(currentInterval)
		}
	}
}
