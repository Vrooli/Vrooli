package deepsearch

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	ScopeGlobal   = "global"
	ScopeScenario = "scenario"
	ScopePath     = "path"
)

const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

const (
	defaultMaxResults     = 10
	maxMaxResults         = 50
	defaultTimeoutSeconds = 60
	maxTimeoutSeconds     = 600
)

var (
	ErrQueryRequired       = errors.New("query is required")
	ErrScopeInvalid        = errors.New("scope must be global, scenario, or path")
	ErrScenarioRequired    = errors.New("scenario is required for scenario scope")
	ErrBasePathRequired    = errors.New("base_path is required for path scope")
	ErrBasePathInvalid     = errors.New("base_path must be within repo root")
	ErrScenarioRootEmpty   = errors.New("scenarios root is not configured")
	ErrAgentUnavailable    = errors.New("agent manager is not configured")
	ErrJobStoreUnavailable = errors.New("deep search job store is not configured")
	ErrJobIDRequired       = errors.New("job_id is required")
	ErrJobNotFound         = errors.New("job not found")
)

// DeepSearchRequest defines the input for agent-powered doc search.
type DeepSearchRequest struct {
	Query          string
	Scope          string
	Scenario       string
	BasePath       string
	MaxResults     int
	FollowRefs     bool
	TimeoutSeconds int
}

// DeepSearchResult represents a ranked documentation match.
type DeepSearchResult struct {
	Path        string   `json:"path"`
	Relevance   float64  `json:"relevance"`
	Summary     string   `json:"summary"`
	MatchReason string   `json:"match_reason"`
	References  []string `json:"references,omitempty"`
	Snippet     string   `json:"snippet,omitempty"`
}

// DeepSearchJob tracks async search status.
type DeepSearchJob struct {
	JobID       string
	Status      string
	Progress    string
	StartedAt   *time.Time
	CompletedAt *time.Time
	Results     []DeepSearchResult
	Error       string
	AgentRunID  string
	MaxResults  int
}

// API exposes the deep search service surface.
type API interface {
	StartSearch(ctx context.Context, req DeepSearchRequest) (*DeepSearchJob, error)
	GetJob(ctx context.Context, jobID string) (*DeepSearchJob, error)
}

func (r *DeepSearchRequest) normalize() error {
	r.Query = strings.TrimSpace(r.Query)
	r.Scope = strings.ToLower(strings.TrimSpace(r.Scope))
	r.Scenario = strings.TrimSpace(r.Scenario)
	r.BasePath = strings.TrimSpace(r.BasePath)
	if r.Query == "" {
		return ErrQueryRequired
	}
	if r.Scope == "" {
		r.Scope = ScopeGlobal
	}
	switch r.Scope {
	case ScopeGlobal:
	case ScopeScenario:
		if r.Scenario == "" {
			return ErrScenarioRequired
		}
	case ScopePath:
		if r.BasePath == "" {
			return ErrBasePathRequired
		}
	default:
		return ErrScopeInvalid
	}
	if r.MaxResults <= 0 {
		r.MaxResults = defaultMaxResults
	}
	if r.MaxResults > maxMaxResults {
		r.MaxResults = maxMaxResults
	}
	if r.TimeoutSeconds <= 0 {
		r.TimeoutSeconds = defaultTimeoutSeconds
	}
	if r.TimeoutSeconds > maxTimeoutSeconds {
		r.TimeoutSeconds = maxTimeoutSeconds
	}
	return nil
}
