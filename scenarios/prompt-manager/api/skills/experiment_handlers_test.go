package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/store"

	"github.com/gorilla/mux"
)

// mockExperimentStore implements store.ExperimentStore for testing.
type mockExperimentStore struct {
	experiments map[string]*store.Experiment
	outcomes    map[string][]store.ExperimentOutcome
	serves      []store.ExperimentServe
}

func newMockExperimentStore() *mockExperimentStore {
	return &mockExperimentStore{
		experiments: make(map[string]*store.Experiment),
		outcomes:    make(map[string][]store.ExperimentOutcome),
	}
}

func (m *mockExperimentStore) List(_ context.Context) ([]store.Experiment, error) {
	var result []store.Experiment
	for _, e := range m.experiments {
		result = append(result, *e)
	}
	return result, nil
}

func (m *mockExperimentStore) ListBySkill(_ context.Context, skillID string) ([]store.Experiment, error) {
	var result []store.Experiment
	for _, e := range m.experiments {
		if e.SkillID == skillID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (m *mockExperimentStore) Get(_ context.Context, id string) (*store.Experiment, error) {
	if e, ok := m.experiments[id]; ok {
		return e, nil
	}
	return nil, errors.New("experiment not found")
}

func (m *mockExperimentStore) Create(_ context.Context, exp *store.Experiment) error {
	if _, ok := m.experiments[exp.ID]; ok {
		return errors.New("experiment already exists")
	}
	exp.Kind = store.KindExperiment
	exp.SchemaVersion = store.CurrentSchemaVersion
	if exp.Status == "" {
		exp.Status = store.ExperimentStatusDraft
	}
	exp.Timestamps = store.NewTimestamps()
	m.experiments[exp.ID] = exp
	return nil
}

func (m *mockExperimentStore) Update(_ context.Context, id string, exp *store.Experiment) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	exp.UpdateTimestamp()
	m.experiments[id] = exp
	return nil
}

func (m *mockExperimentStore) Delete(_ context.Context, id string) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	delete(m.experiments, id)
	delete(m.outcomes, id)
	return nil
}

func (m *mockExperimentStore) RecordOutcome(_ context.Context, id string, outcome store.ExperimentOutcome) error {
	if _, ok := m.experiments[id]; !ok {
		return errors.New("experiment not found")
	}
	m.outcomes[id] = append(m.outcomes[id], outcome)
	return nil
}

func (m *mockExperimentStore) RecordServe(_ context.Context, serve store.ExperimentServe) error {
	if _, ok := m.experiments[serve.ExperimentID]; !ok {
		return errors.New("experiment not found")
	}
	m.serves = append(m.serves, serve)
	return nil
}

func (m *mockExperimentStore) ListServes(_ context.Context, id string) ([]store.ExperimentServe, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	var result []store.ExperimentServe
	for _, s := range m.serves {
		if s.ExperimentID == id {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockExperimentStore) CountServesByVariant(_ context.Context, id string) (map[string]int, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	counts := make(map[string]int)
	for _, s := range m.serves {
		if s.ExperimentID == id {
			counts[s.VariantID]++
		}
	}
	return counts, nil
}

func (m *mockExperimentStore) ListOutcomes(_ context.Context, id string) ([]store.ExperimentOutcome, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	return m.outcomes[id], nil
}

func (m *mockExperimentStore) CountOutcomesByVariant(_ context.Context, id string) (map[string]int, error) {
	if _, ok := m.experiments[id]; !ok {
		return nil, errors.New("experiment not found")
	}
	counts := make(map[string]int)
	for _, o := range m.outcomes[id] {
		counts[o.VariantID]++
	}
	return counts, nil
}

func TestExperimentHandlers_CreateAndGet(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v1", Name: "V1"}, "content"); err != nil {
		t.Fatal(err)
	}
	h := NewExperimentHandlers(es, vs, ss)

	reqBody := CreateExperimentRequest{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Test Experiment",
		Arms: []ExperimentArmInput{
			{VariantID: "control", Weight: 0.5},
			{VariantID: "v1", Weight: 0.5},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/experiments", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateExperiment(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp ExperimentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ID != "exp-1" || resp.Status != "draft" {
		t.Errorf("unexpected response: %+v", resp)
	}
	if len(resp.Arms) != 2 {
		t.Errorf("expected 2 arms, got %d", len(resp.Arms))
	}

	// Get
	req2 := httptest.NewRequest("GET", "/experiments/exp-1", nil)
	req2 = mux.SetURLVars(req2, map[string]string{"eid": "exp-1"})
	w2 := httptest.NewRecorder()
	h.GetExperiment(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w2.Code)
	}
}

func TestExperimentHandlers_Lifecycle(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v1", Name: "V1"}, "winner content"); err != nil {
		t.Fatal(err)
	}
	h := NewExperimentHandlers(es, vs, ss)

	// Create
	if err := es.Create(context.Background(), &store.Experiment{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Test",
		Arms: []store.ExperimentArm{
			{VariantID: "control", Weight: 0.5},
			{VariantID: "v1", Weight: 0.5},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// Start
	req := httptest.NewRequest("POST", "/experiments/exp-1/start", nil)
	req = mux.SetURLVars(req, map[string]string{"eid": "exp-1"})
	w := httptest.NewRecorder()
	h.StartExperiment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("start: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var startResp ExperimentResponse
	if err := json.Unmarshal(w.Body.Bytes(), &startResp); err != nil {
		t.Fatal(err)
	}
	if startResp.Status != "running" {
		t.Errorf("expected status running, got %s", startResp.Status)
	}

	// Record outcome
	outcomeBody, _ := json.Marshal(RecordOutcomeRequest{
		VariantID:     "v1",
		Source:        "test",
		SchemaVersion: 1,
		Data:          json.RawMessage(`{"classification":"ready"}`),
	})
	req2 := httptest.NewRequest("POST", "/experiments/exp-1/outcomes", bytes.NewReader(outcomeBody))
	req2 = mux.SetURLVars(req2, map[string]string{"eid": "exp-1"})
	w2 := httptest.NewRecorder()
	h.RecordOutcome(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("outcome: expected 204, got %d: %s", w2.Code, w2.Body.String())
	}

	// Conclude
	concludeBody, _ := json.Marshal(ConcludeExperimentRequest{
		WinnerVariantID: "v1",
		Notes:           "V1 wins",
	})
	req3 := httptest.NewRequest("POST", "/experiments/exp-1/conclude", bytes.NewReader(concludeBody))
	req3 = mux.SetURLVars(req3, map[string]string{"eid": "exp-1"})
	w3 := httptest.NewRecorder()
	h.ConcludeExperiment(w3, req3)

	if w3.Code != http.StatusOK {
		t.Fatalf("conclude: expected 200, got %d: %s", w3.Code, w3.Body.String())
	}

	var concludeResp ExperimentResponse
	if err := json.Unmarshal(w3.Body.Bytes(), &concludeResp); err != nil {
		t.Fatal(err)
	}
	if concludeResp.Status != "concluded" {
		t.Errorf("expected status concluded, got %s", concludeResp.Status)
	}
	if concludeResp.WinnerVariantID == nil || *concludeResp.WinnerVariantID != "v1" {
		t.Errorf("expected winner v1")
	}
}

func TestExperimentHandlers_InvalidWeights(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	h := NewExperimentHandlers(es, vs, ss)

	reqBody := CreateExperimentRequest{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Bad Weights",
		Arms: []ExperimentArmInput{
			{VariantID: "control", Weight: 0.3},
			{VariantID: "v1", Weight: 0.3},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/experiments", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateExperiment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExperimentHandlers_StartNonDraft(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	h := NewExperimentHandlers(es, vs, ss)

	es.experiments["exp-1"] = &store.Experiment{
		ID:     "exp-1",
		Status: store.ExperimentStatusRunning,
	}

	req := httptest.NewRequest("POST", "/experiments/exp-1/start", nil)
	req = mux.SetURLVars(req, map[string]string{"eid": "exp-1"})
	w := httptest.NewRecorder()
	h.StartExperiment(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestExperimentHandlers_ConcludeInvalidWinner(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	h := NewExperimentHandlers(es, vs, ss)

	es.experiments["exp-1"] = &store.Experiment{
		ID:      "exp-1",
		SkillID: "s1",
		Status:  store.ExperimentStatusRunning,
		Arms: []store.ExperimentArm{
			{VariantID: "control", Weight: 0.5},
			{VariantID: "v1", Weight: 0.5},
		},
	}

	body, _ := json.Marshal(ConcludeExperimentRequest{WinnerVariantID: "nonexistent"})
	req := httptest.NewRequest("POST", "/experiments/exp-1/conclude", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"eid": "exp-1"})
	w := httptest.NewRecorder()
	h.ConcludeExperiment(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestExperimentHandlers_ListOutcomes(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	h := NewExperimentHandlers(es, vs, ss)

	es.experiments["exp-1"] = &store.Experiment{
		ID:     "exp-1",
		Status: store.ExperimentStatusRunning,
	}
	es.outcomes["exp-1"] = []store.ExperimentOutcome{
		{VariantID: "control", Source: "test", SchemaVersion: 1, RecordedAt: "t1", Data: json.RawMessage(`{}`)},
		{VariantID: "v1", Source: "test", SchemaVersion: 1, RecordedAt: "t2", Data: json.RawMessage(`{}`)},
	}

	req := httptest.NewRequest("GET", "/experiments/exp-1/outcomes", nil)
	req = mux.SetURLVars(req, map[string]string{"eid": "exp-1"})
	w := httptest.NewRecorder()
	h.ListOutcomes(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp []ExperimentOutcomeResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Errorf("expected 2 outcomes, got %d", len(resp))
	}
}
