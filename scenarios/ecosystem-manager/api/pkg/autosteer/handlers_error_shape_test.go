package autosteer

import (
	"bytes"
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
func (s *stubExecutionEngine) SeekExecution(taskID, profileID, scenarioName string, phaseIndex, phaseIteration int) (*ProfileExecutionState, error) {
	return nil, nil
}
func (s *stubExecutionEngine) AdvancePhase(taskID, scenarioName string) (*PhaseAdvanceResult, error) {
	return nil, nil
}
func (s *stubExecutionEngine) GetExecutionState(taskID string) (*ProfileExecutionState, error) {
	return s.state, s.stateErr
}
func (s *stubExecutionEngine) GetCurrentMode(taskID string) (SteerMode, error) {
	return ModeProgress, nil
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
func (s *stubHistoryService) GetProfileAnalytics(profileID string) (*ProfileAnalytics, error) {
	return nil, nil
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
