package autosteer

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/effectiveness"
	"github.com/vrooli/maturity-go/dimensions"
)

type stubProfileService struct {
	listErr error
	getErr  error
}

func (s *stubProfileService) CreateProfile(profile *AutoSteerProfile) error { return nil }
func (s *stubProfileService) ListProfiles(tags []string) ([]*AutoSteerProfile, error) {
	return nil, s.listErr
}
func (s *stubProfileService) GetProfile(id string) (*AutoSteerProfile, error)          { return nil, s.getErr }
func (s *stubProfileService) UpdateProfile(id string, updates *AutoSteerProfile) error { return nil }
func (s *stubProfileService) DeleteProfile(id string) error                            { return nil }
func (s *stubProfileService) GetTemplates() []*AutoSteerProfile                        { return nil }

type stubExecutionEngine struct {
	state    *ProfileExecutionState
	stateErr error
	effStats []effectiveness.Stat
	effErr   error
}

func (s *stubExecutionEngine) StartExecution(taskID, profileID, scenarioName string) (*ProfileExecutionState, error) {
	return nil, nil
}

func (s *stubExecutionEngine) EvaluateIteration(taskID, scenarioName string) (*IterationEvaluation, error) {
	return nil, nil
}
func (s *stubExecutionEngine) DeleteExecutionState(taskID string) error { return nil }

func (s *stubExecutionEngine) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return s.state, s.stateErr
}

func (s *stubExecutionEngine) GetCurrentSet(taskID string) ([]string, error) {
	return []string{string(ModeProgress)}, nil
}

func (s *stubExecutionEngine) GetDecisionTrace(taskID string) ([]DecisionTraceEntry, error) {
	return nil, nil
}

func (s *stubExecutionEngine) Effectiveness(skillID string, dim dimensions.Dimension) ([]effectiveness.Stat, error) {
	return s.effStats, s.effErr
}

func (s *stubExecutionEngine) EffectivenessPrior() (float64, float64) {
	return 0, effectiveness.DefaultShrinkageK
}

type stubHistoryService struct{}

func (s *stubHistoryService) GetHistory(filters HistoryFilters) ([]ProfilePerformance, error) {
	return nil, nil
}

func (s *stubHistoryService) GetExecution(executionID string) (*ProfilePerformance, error) {
	return nil, nil
}

func (s *stubHistoryService) SubmitFeedback(executionID string, rating int, comments string) error {
	return nil
}

func (s *stubHistoryService) SubmitFeedbackEntry(executionID string, req ExecutionFeedbackRequest) (*ExecutionFeedbackEntry, error) {
	return &ExecutionFeedbackEntry{
		ID:              executionID,
		Category:        req.Category,
		Severity:        req.Severity,
		SuggestedAction: req.SuggestedAction,
		Comments:        req.Comments,
		Metadata:        req.Metadata,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (s *stubHistoryService) GetProfileAnalytics(profileID string) (*ProfileAnalytics, error) {
	return nil, nil
}

func TestGetEffectiveness_ReturnsLedgerWithDerivedEfficacy(t *testing.T) {
	engine := &stubExecutionEngine{effStats: []effectiveness.Stat{
		{SkillID: "lint-fix", Dimension: "standards", ClosedCount: 20, IntroducedCount: 0, TotalRuns: 5, TotalTokens: 5000},
	}}
	handlers := NewAutoSteerHandlers(&stubProfileService{}, engine, &stubHistoryService{})

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/effectiveness?dimension=standards", nil)
	w := httptest.NewRecorder()
	handlers.GetEffectiveness(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		Effectiveness []effectivenessRow `json:"effectiveness"`
		Count         int                `json:"count"`
		ShrinkageK    float64            `json:"shrinkage_k"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Count != 1 || len(resp.Effectiveness) != 1 {
		t.Fatalf("expected 1 row, got %d", resp.Count)
	}
	row := resp.Effectiveness[0]
	if row.SkillID != "lint-fix" || row.NetClosed != 20 || row.AvgTokensPerRun != 1000 {
		t.Fatalf("unexpected row: %+v", row)
	}
	if row.ExpectedEfficacyPerKtok <= 0 {
		t.Fatalf("expected positive efficacy estimate, got %v", row.ExpectedEfficacyPerKtok)
	}
}

func TestGetEffectiveness_PropagatesError(t *testing.T) {
	handlers := NewAutoSteerHandlers(&stubProfileService{}, &stubExecutionEngine{effErr: errors.New("boom")}, &stubHistoryService{})
	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/effectiveness", nil)
	w := httptest.NewRecorder()
	handlers.GetEffectiveness(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestListProfiles_ReturnsStructuredError(t *testing.T) {
	handlers := NewAutoSteerHandlers(&stubProfileService{listErr: errors.New("boom")}, &stubExecutionEngine{}, &stubHistoryService{})

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/profiles", nil)
	w := httptest.NewRecorder()

	handlers.ListProfiles(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Error != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("expected error %q, got %q", http.StatusText(http.StatusInternalServerError), resp.Error)
	}
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected code %d, got %d", http.StatusInternalServerError, resp.Code)
	}
	if resp.Message == "" {
		t.Fatalf("expected message to be set")
	}
}

func TestGetExecutionState_NotFoundStructured(t *testing.T) {
	handlers := NewAutoSteerHandlers(&stubProfileService{}, &stubExecutionEngine{}, &stubHistoryService{})

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/execution/123", nil)
	w := httptest.NewRecorder()

	handlers.GetExecutionState(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "No execution state found" {
		t.Fatalf("expected message 'No execution state found', got %q", resp.Message)
	}
}

func TestSubmitFeedback_InvalidRatingStructured(t *testing.T) {
	handlers := NewAutoSteerHandlers(&stubProfileService{}, &stubExecutionEngine{}, &stubHistoryService{})

	body := bytes.NewBufferString(`{"rating": 6}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auto-steer/history/exec-1/feedback", body)
	w := httptest.NewRecorder()

	handlers.SubmitFeedback(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Message != "Rating must be between 1 and 5" {
		t.Fatalf("unexpected message: %q", resp.Message)
	}
}
