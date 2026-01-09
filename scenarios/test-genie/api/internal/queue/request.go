package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"test-genie/internal/shared"
)

// Status constants for suite request lifecycle.
const (
	StatusQueued    = "queued"
	StatusDelegated = "delegated"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Priority constants for queue ordering.
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

const MaxSuiteListPage = 50

var (
	allowedSuiteTypes = map[string]struct{}{
		"unit":        {},
		"integration": {},
		"performance": {},
		"vault":       {},
		"regression":  {},
	}
	defaultSuiteTypes = []string{"unit", "integration"}
	allowedPriorities = map[string]struct{}{
		PriorityLow:    {},
		PriorityNormal: {},
		PriorityHigh:   {},
		PriorityUrgent: {},
	}
	allowedStatuses = map[string]struct{}{
		StatusQueued:    {},
		StatusDelegated: {},
		StatusRunning:   {},
		StatusCompleted: {},
		StatusFailed:    {},
	}
)

// SuiteRequest represents a queued generation job that may be delegated to downstream agents.
type SuiteRequest struct {
	ID                 uuid.UUID `json:"id"`
	ScenarioName       string    `json:"scenarioName"`
	RequestedTypes     []string  `json:"requestedTypes"`
	CoverageTarget     int       `json:"coverageTarget"`
	Priority           string    `json:"priority"`
	Status             string    `json:"status"`
	Notes              string    `json:"notes,omitempty"`
	DelegationIssueID  *string   `json:"delegationIssueId,omitempty"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
	EstimatedQueueTime int       `json:"estimatedQueueTimeSeconds,omitempty"`
}

// SuiteRequestSnapshot summarizes queue health for telemetry surfaces.
type SuiteRequestSnapshot struct {
	Total          int        `json:"total"`
	Queued         int        `json:"queued"`
	Delegated      int        `json:"delegated"`
	Running        int        `json:"running"`
	Completed      int        `json:"completed"`
	Failed         int        `json:"failed"`
	OldestQueuedAt *time.Time `json:"oldestQueuedAt,omitempty"`
}

// QueueSuiteRequestInput captures the payload that queues a new suite request.
type QueueSuiteRequestInput struct {
	ScenarioName   string     `json:"scenarioName"`
	RequestedTypes stringList `json:"requestedTypes"`
	CoverageTarget int        `json:"coverageTarget"`
	Priority       string     `json:"priority"`
	Notes          string     `json:"notes"`
}

type stringList []string

func (sl *stringList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*sl = nil
		return nil
	}

	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*sl = nil
		return nil
	}

	var arr []string
	if err := json.Unmarshal(trimmed, &arr); err == nil {
		*sl = arr
		return nil
	}

	var single string
	if err := json.Unmarshal(trimmed, &single); err == nil {
		*sl = []string{single}
		return nil
	}

	return fmt.Errorf("expected string or array of strings for requestedTypes")
}

// SuiteRequestRepository exposes persistence operations for the domain service.
type SuiteRequestRepository interface {
	Create(ctx context.Context, req *SuiteRequest) error
	List(ctx context.Context, limit int) ([]SuiteRequest, error)
	GetByID(ctx context.Context, id uuid.UUID) (*SuiteRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	StatusSnapshot(ctx context.Context) (SuiteRequestSnapshot, error)
}

// SuiteRequestService owns the orchestration rules for queueing and retrieving suites.
type SuiteRequestService struct {
	repo SuiteRequestRepository
}

func NewSuiteRequestService(repo SuiteRequestRepository) *SuiteRequestService {
	return &SuiteRequestService{repo: repo}
}

// Queue creates a new suite request instance and persists it through the repository.
func (s *SuiteRequestService) Queue(ctx context.Context, payload QueueSuiteRequestInput) (*SuiteRequest, error) {
	req, err := buildSuiteRequest(payload)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// List returns recent suite requests, capping the limit to a safe page size.
func (s *SuiteRequestService) List(ctx context.Context, limit int) ([]SuiteRequest, error) {
	if limit <= 0 || limit > MaxSuiteListPage {
		limit = MaxSuiteListPage
	}
	return s.repo.List(ctx, limit)
}

// Get loads a suite request by identifier.
func (s *SuiteRequestService) Get(ctx context.Context, id uuid.UUID) (*SuiteRequest, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateStatus transitions a request to a new state when orchestration progresses.
func (s *SuiteRequestService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	if _, ok := allowedStatuses[status]; !ok {
		return shared.NewValidationError(fmt.Sprintf("status '%s' is not supported", status))
	}
	return s.repo.UpdateStatus(ctx, id, status)
}

// StatusSnapshot reports queue state for telemetry/health surfaces.
func (s *SuiteRequestService) StatusSnapshot(ctx context.Context) (SuiteRequestSnapshot, error) {
	return s.repo.StatusSnapshot(ctx)
}

func buildSuiteRequest(payload QueueSuiteRequestInput) (*SuiteRequest, error) {
	scenario := strings.TrimSpace(payload.ScenarioName)
	if scenario == "" {
		return nil, shared.NewValidationError("scenarioName is required")
	}

	types, err := normalizeSuiteTypes([]string(payload.RequestedTypes))
	if err != nil {
		return nil, err
	}

	coverage := payload.CoverageTarget
	if coverage == 0 {
		coverage = 95
	}
	if coverage < 1 || coverage > 100 {
		return nil, shared.NewValidationError("coverageTarget must be between 1 and 100")
	}

	priority, err := normalizePriority(payload.Priority)
	if err != nil {
		return nil, err
	}

	notes := strings.TrimSpace(payload.Notes)

	req := &SuiteRequest{
		ID:             uuid.New(),
		ScenarioName:   scenario,
		RequestedTypes: types,
		CoverageTarget: coverage,
		Priority:       priority,
		Status:         StatusQueued,
		Notes:          notes,
	}
	req.EstimatedQueueTime = estimateQueueSeconds(len(req.RequestedTypes), req.CoverageTarget)
	return req, nil
}

func normalizeSuiteTypes(values []string) ([]string, error) {
	if len(values) == 0 {
		return append([]string(nil), defaultSuiteTypes...), nil
	}

	seen := make(map[string]struct{}, len(values))
	var sanitized []string
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			continue
		}
		if _, ok := allowedSuiteTypes[trimmed]; !ok {
			return nil, shared.NewValidationError(fmt.Sprintf("requested type '%s' is not supported", value))
		}
		if _, dup := seen[trimmed]; dup {
			continue
		}
		seen[trimmed] = struct{}{}
		sanitized = append(sanitized, trimmed)
	}

	if len(sanitized) == 0 {
		return append([]string(nil), defaultSuiteTypes...), nil
	}

	return sanitized, nil
}

func normalizePriority(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return PriorityNormal, nil
	}
	if _, ok := allowedPriorities[trimmed]; !ok {
		return "", shared.NewValidationError(fmt.Sprintf("priority '%s' is not supported", value))
	}
	return trimmed, nil
}

func estimateQueueSeconds(typeCount, coverage int) int {
	if typeCount <= 0 {
		typeCount = len(defaultSuiteTypes)
	}
	if coverage <= 0 {
		coverage = 95
	}
	// Deterministic heuristic: weight coverage target plus workload per type
	return (typeCount * 30) + coverage
}
