package autosteer

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
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

type stubHistoryService struct{}

func (s *stubHistoryService) GetHistory(filters HistoryFilters) ([]ProfilePerformance, error) {
	return nil, nil
}

func (s *stubHistoryService) GetExecution(executionID string) (*ProfilePerformance, error) {
	return nil, nil
}

func (s *stubHistoryService) GetProfileAnalytics(profileID string) (*ProfileAnalytics, error) {
	return nil, nil
}

type stubCoverageReporter struct {
	report CoverageReport
	err    error
}

func (s stubCoverageReporter) Report(profileID, scenario string) (CoverageReport, error) {
	if s.err != nil {
		return CoverageReport{}, s.err
	}
	s.report.ProfileID = profileID
	s.report.Scenario = scenario
	return s.report, nil
}

func TestGetCoverage_ReturnsReport(t *testing.T) {
	handlers := NewAutoSteerHandlers(
		&stubProfileService{},
		&stubExecutionEngine{},
		&stubHistoryService{},
		WithCoverageReporter(stubCoverageReporter{report: CoverageReport{
			ProfileName:       "Production Ready",
			EffectiveAllowSet: []string{"lint-fix"},
		}}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/coverage?profile=production-ready&scenario=demo", nil)
	w := httptest.NewRecorder()
	handlers.GetCoverage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	var resp CoverageReport
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ProfileID != "production-ready" || resp.Scenario != "demo" {
		t.Fatalf("unexpected report identity: %+v", resp)
	}
	if len(resp.EffectiveAllowSet) != 1 || resp.EffectiveAllowSet[0] != "lint-fix" {
		t.Fatalf("unexpected allow set: %+v", resp.EffectiveAllowSet)
	}
}

func TestGetCoverage_RequiresProfile(t *testing.T) {
	handlers := NewAutoSteerHandlers(
		&stubProfileService{},
		&stubExecutionEngine{},
		&stubHistoryService{},
		WithCoverageReporter(stubCoverageReporter{}),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/auto-steer/coverage", nil)
	w := httptest.NewRecorder()
	handlers.GetCoverage(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
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
