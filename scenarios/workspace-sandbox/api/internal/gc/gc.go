// Package gc provides garbage collection for sandboxes.
// [OT-P1-003] GC/Prune Operations
//
// # Overview
//
// The GC system removes sandboxes that are no longer needed based on configurable
// policies. It supports multiple criteria for determining what to collect:
//   - Age-based: sandboxes older than a threshold
//   - Idle-based: sandboxes not used recently
//   - Size-based: when total storage exceeds limits
//   - Terminal state: approved/rejected sandboxes after a delay
//
// # Safety
//
// The GC system is designed with safety in mind:
//   - Never collects active or creating sandboxes
//   - Supports dry-run mode to preview what would be collected
//   - Logs all operations for auditing
//   - Handles partial failures gracefully
//
// # Usage
//
//	svc := gc.NewService(repo, driver, config)
//	result, err := svc.Run(ctx, &types.GCRequest{
//	    DryRun: true,
//	    Policy: &types.GCPolicy{
//	        MaxAge: 24 * time.Hour,
//	    },
//	})
package gc

import (
	"context"
	"fmt"
	"time"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/types"
)

// Service provides garbage collection operations for sandboxes.
type Service struct {
	repo   repository.Repository
	driver driver.Driver
	config Config
}

// Config holds GC service configuration.
type Config struct {
	// DefaultMaxAge is the default maximum age for sandboxes.
	DefaultMaxAge time.Duration

	// DefaultIdleTimeout is the default idle timeout.
	DefaultIdleTimeout time.Duration

	// DefaultTerminalDelay is the default delay before collecting terminal sandboxes.
	DefaultTerminalDelay time.Duration

	// DefaultLimit is the default max sandboxes to collect per run.
	DefaultLimit int

	// MaxTotalSizeBytes is the maximum total storage allowed.
	MaxTotalSizeBytes int64
}

// DefaultConfig returns sensible defaults for GC configuration.
func DefaultConfig() Config {
	return Config{
		DefaultMaxAge:        24 * time.Hour,
		DefaultIdleTimeout:   4 * time.Hour,
		DefaultTerminalDelay: 1 * time.Hour,
		DefaultLimit:         100,
		MaxTotalSizeBytes:    100 * 1024 * 1024 * 1024, // 100 GB
	}
}

// NewService creates a new GC service.
func NewService(repo repository.Repository, drv driver.Driver, cfg Config) *Service {
	return &Service{
		repo:   repo,
		driver: drv,
		config: cfg,
	}
}

// Run executes a garbage collection cycle based on the request parameters.
// If DryRun is true, returns what would be collected without actually deleting.
func (s *Service) Run(ctx context.Context, req *types.GCRequest) (*types.GCResult, error) {
	startedAt := time.Now()

	result := &types.GCResult{
		Collected: []*types.GCCollectedSandbox{},
		DryRun:    req.DryRun,
		StartedAt: startedAt,
		Reasons:   make(map[string][]string),
	}

	// Use default policy if not specified
	policy := req.Policy
	if policy == nil {
		defaultPolicy := s.buildDefaultPolicy()
		policy = &defaultPolicy
	}

	// Get limit
	limit := req.Limit
	if limit <= 0 {
		limit = s.config.DefaultLimit
	}

	// Get candidates from repository
	candidates, err := s.repo.GetGCCandidates(ctx, policy, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get GC candidates: %w", err)
	}

	// Check if we need size-based collection
	if policy.MaxTotalSizeBytes > 0 {
		candidates, err = s.filterBySizeLimit(ctx, candidates, policy.MaxTotalSizeBytes, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to filter by size: %w", err)
		}
	}

	// Determine reasons for each candidate
	now := time.Now()
	for _, sandbox := range candidates {
		reasons := s.determineReasons(sandbox, policy, now)
		result.Reasons[sandbox.ID.String()] = reasons
	}

	// If dry run, just return the candidates without deleting
	if req.DryRun {
		for _, sandbox := range candidates {
			reasons := result.Reasons[sandbox.ID.String()]
			reason := "multiple criteria"
			if len(reasons) == 1 {
				reason = reasons[0]
			}

			result.Collected = append(result.Collected, &types.GCCollectedSandbox{
				ID:        sandbox.ID,
				ScopePath: sandbox.ScopePath,
				Status:    sandbox.Status,
				SizeBytes: sandbox.SizeBytes,
				CreatedAt: sandbox.CreatedAt,
				Reason:    reason,
			})
			result.TotalBytesReclaimed += sandbox.SizeBytes
		}
		result.TotalCollected = len(result.Collected)
		result.CompletedAt = time.Now()
		return result, nil
	}

	// Actually delete the sandboxes
	for _, sandbox := range candidates {
		reasons := result.Reasons[sandbox.ID.String()]
		reason := "multiple criteria"
		if len(reasons) == 1 {
			reason = reasons[0]
		}

		// Cleanup driver resources first
		if err := s.driver.Cleanup(ctx, sandbox); err != nil {
			// Log error but continue - driver cleanup failures shouldn't block deletion
			result.Errors = append(result.Errors, types.GCError{
				SandboxID: sandbox.ID,
				Error:     fmt.Sprintf("driver cleanup warning: %v", err),
			})
		}

		// Delete from repository
		if err := s.repo.Delete(ctx, sandbox.ID); err != nil {
			result.Errors = append(result.Errors, types.GCError{
				SandboxID: sandbox.ID,
				Error:     fmt.Sprintf("delete failed: %v", err),
			})
			continue
		}

		// Log audit event
		s.repo.LogAuditEvent(ctx, &types.AuditEvent{
			SandboxID: &sandbox.ID,
			EventType: "gc_collected",
			Actor:     req.Actor,
			ActorType: "gc",
			Details: map[string]interface{}{
				"reason":    reason,
				"sizeBytes": sandbox.SizeBytes,
				"dryRun":    false,
			},
		})

		result.Collected = append(result.Collected, &types.GCCollectedSandbox{
			ID:        sandbox.ID,
			ScopePath: sandbox.ScopePath,
			Status:    sandbox.Status,
			SizeBytes: sandbox.SizeBytes,
			CreatedAt: sandbox.CreatedAt,
			Reason:    reason,
		})
		result.TotalBytesReclaimed += sandbox.SizeBytes
	}

	result.TotalCollected = len(result.Collected)
	result.CompletedAt = time.Now()

	return result, nil
}

// buildDefaultPolicy creates a policy from the service config.
func (s *Service) buildDefaultPolicy() types.GCPolicy {
	return types.GCPolicy{
		MaxAge:            s.config.DefaultMaxAge,
		IdleTimeout:       s.config.DefaultIdleTimeout,
		IncludeTerminal:   true,
		TerminalDelay:     s.config.DefaultTerminalDelay,
		MaxTotalSizeBytes: s.config.MaxTotalSizeBytes,
		Statuses:          []types.Status{types.StatusStopped, types.StatusError, types.StatusApproved, types.StatusRejected},
	}
}

// filterBySizeLimit returns candidates needed to get under the size limit.
// If current total size is under the limit, returns empty slice.
// Otherwise, returns oldest sandboxes needed to reduce to target.
func (s *Service) filterBySizeLimit(ctx context.Context, candidates []*types.Sandbox, maxSize int64, limit int) ([]*types.Sandbox, error) {
	stats, err := s.repo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	// If under limit, no size-based collection needed
	if stats.TotalSizeBytes <= maxSize {
		return candidates, nil
	}

	// Need to free: current - target
	bytesToFree := stats.TotalSizeBytes - maxSize
	var freedBytes int64
	var result []*types.Sandbox

	// Candidates are already sorted oldest-first from the repository
	for _, sandbox := range candidates {
		if freedBytes >= bytesToFree || len(result) >= limit {
			break
		}
		result = append(result, sandbox)
		freedBytes += sandbox.SizeBytes
	}

	return result, nil
}

// determineReasons returns the reasons why a sandbox is eligible for GC.
func (s *Service) determineReasons(sandbox *types.Sandbox, policy *types.GCPolicy, now time.Time) []string {
	var reasons []string

	// Check MaxAge
	if policy.MaxAge > 0 {
		cutoff := now.Add(-policy.MaxAge)
		if sandbox.CreatedAt.Before(cutoff) {
			reasons = append(reasons, fmt.Sprintf("exceeded max age (%v)", policy.MaxAge))
		}
	}

	// Check IdleTimeout
	if policy.IdleTimeout > 0 {
		idleCutoff := now.Add(-policy.IdleTimeout)
		if sandbox.LastUsedAt.Before(idleCutoff) {
			reasons = append(reasons, fmt.Sprintf("idle for %v", now.Sub(sandbox.LastUsedAt).Round(time.Minute)))
		}
	}

	// Check TerminalDelay for approved/rejected sandboxes
	if policy.IncludeTerminal && policy.TerminalDelay > 0 {
		if sandbox.Status == types.StatusApproved || sandbox.Status == types.StatusRejected {
			terminalCutoff := now.Add(-policy.TerminalDelay)
			terminalTime := sandbox.StoppedAt
			if sandbox.ApprovedAt != nil {
				terminalTime = sandbox.ApprovedAt
			}
			if terminalTime != nil && terminalTime.Before(terminalCutoff) {
				reasons = append(reasons, fmt.Sprintf("terminal state (%s) exceeded delay (%v)", sandbox.Status, policy.TerminalDelay))
			}
		}
	}

	// Check size-based (handled separately but mark as reason)
	if policy.MaxTotalSizeBytes > 0 {
		// Size-based is determined at the service level, not per-sandbox
		// This reason is added by the caller if applicable
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "matched policy criteria")
	}

	return reasons
}

// Preview returns what would be collected without actually deleting.
// This is a convenience wrapper around Run with DryRun=true.
func (s *Service) Preview(ctx context.Context, policy *types.GCPolicy, limit int) (*types.GCResult, error) {
	return s.Run(ctx, &types.GCRequest{
		Policy: policy,
		DryRun: true,
		Limit:  limit,
	})
}
