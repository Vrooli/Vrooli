package dochealing

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	StatusPending     = "pending"
	StatusRunning     = "running"
	StatusNeedsReview = "needs_review"
	StatusApproved    = "approved"
	StatusRejected    = "rejected"
	StatusFailed      = "failed"
)

const (
	defaultTimeoutSeconds = 600
	maxTimeoutSeconds     = 1800
)

var (
	ErrScenarioRequired    = errors.New("scenario is required")
	ErrScenarioRootEmpty   = errors.New("scenarios root is not configured")
	ErrScenarioNotFound    = errors.New("scenario not found")
	ErrAgentUnavailable    = errors.New("agent manager is not configured")
	ErrJobStoreUnavailable = errors.New("doc healing job store is not configured")
	ErrJobIDRequired       = errors.New("job_id is required")
	ErrJobNotFound         = errors.New("job not found")
	ErrJobNotReady         = errors.New("job is not ready for review")
	ErrJobNotApprovable    = errors.New("job is not ready for approval")
	ErrJobNotRejectable    = errors.New("job is not ready for rejection")
	ErrHealthUnavailable   = errors.New("doc health service is not configured")
)

// HealRequest defines the input for a documentation healing job.
type HealRequest struct {
	ScenarioName   string
	Issues         []string
	AutoApprove    bool
	DryRun         bool
	TimeoutSeconds int
}

// DiffPreview captures a unified diff preview for review.
type DiffPreview struct {
	Files   []FileDiff
	Summary string
}

// FileDiff represents a single file change.
type FileDiff struct {
	Path      string
	Operation string
	OldPath   string
	Diff      string
}

// HealJob tracks the status of a healing job.
type HealJob struct {
	JobID        string
	ScenarioName string
	Status       string
	Progress     string
	StartedAt    *time.Time
	CompletedAt  *time.Time
	Diff         *DiffPreview
	HealthBefore *float64
	HealthAfter  *float64
	Error        string
	AgentRunID   string
	AutoApprove  bool
	DryRun       bool
}

// AutoFixResult describes the outcome of a deterministic auto-fix operation.
type AutoFixResult struct {
	ScenarioName string
	Moved        []MovedDoc
	Skipped      []SkippedDoc
	HealthBefore float64
	HealthAfter  float64
}

// MovedDoc records a successfully moved file.
type MovedDoc struct {
	FromPath string
	ToPath   string
	DocType  string
}

// SkippedDoc records a file that could not be moved.
type SkippedDoc struct {
	FromPath string
	ToPath   string
	DocType  string
	Reason   string
}

// API exposes the healing service surface.
type API interface {
	StartHealing(ctx context.Context, req HealRequest) (*HealJob, error)
	GetJob(ctx context.Context, jobID string) (*HealJob, error)
	ApproveJob(ctx context.Context, jobID, actor string) (*HealJob, error)
	RejectJob(ctx context.Context, jobID, actor, reason string) (*HealJob, error)
	AutoFix(ctx context.Context, scenarioName string, dryRun bool) (*AutoFixResult, error)
}

func (r *HealRequest) normalize() error {
	r.ScenarioName = strings.TrimSpace(r.ScenarioName)
	if r.ScenarioName == "" {
		return ErrScenarioRequired
	}
	issues := make([]string, 0, len(r.Issues))
	for _, issue := range r.Issues {
		issue = strings.TrimSpace(issue)
		if issue != "" {
			issues = append(issues, issue)
		}
	}
	r.Issues = issues
	if r.TimeoutSeconds <= 0 {
		r.TimeoutSeconds = defaultTimeoutSeconds
	}
	if r.TimeoutSeconds > maxTimeoutSeconds {
		r.TimeoutSeconds = maxTimeoutSeconds
	}
	return nil
}
