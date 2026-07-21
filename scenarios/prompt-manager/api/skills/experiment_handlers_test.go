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
	experiments  map[string]*store.Experiment
	outcomes     map[string][]store.ExperimentOutcome
	serves       []store.ExperimentServe
	assignments  []store.ExperimentAssignment
	exposures    []store.ExperimentExposure
	auditReceipt *store.ExperimentAuditReceipt
}

func (m *mockExperimentStore) RecordExposure(_ context.Context, exposure store.ExperimentExposure) error {
	m.exposures = append(m.exposures, exposure)
	return nil
}
func (m *mockExperimentStore) ListExposures(_ context.Context, _ string) ([]store.ExperimentExposure, error) {
	return append([]store.ExperimentExposure(nil), m.exposures...), nil
}
func (m *mockExperimentStore) ListAssignments(_ context.Context, _ string) ([]store.ExperimentAssignment, error) {
	return append([]store.ExperimentAssignment(nil), m.assignments...), nil
}
func (m *mockExperimentStore) GetAssignment(_ context.Context, experimentID, key string) (*store.ExperimentAssignment, error) {
	for i := range m.assignments {
		if m.assignments[i].ExperimentID == experimentID && m.assignments[i].IdempotencyKey == key {
			return &m.assignments[i], nil
		}
	}
	return nil, errors.New("assignment not found")
}
func (m *mockExperimentStore) CreateAssignment(_ context.Context, assignment store.ExperimentAssignment) error {
	m.assignments = append(m.assignments, assignment)
	return nil
}
func (m *mockExperimentStore) RecordAuditReceipt(_ context.Context, receipt store.ExperimentAuditReceipt) error {
	m.auditReceipt = &receipt
	return nil
}
func (m *mockExperimentStore) GetAuditReceipt(_ context.Context, _ string) (*store.ExperimentAuditReceipt, error) {
	if m.auditReceipt == nil {
		return nil, errors.New("audit receipt not found")
	}
	return m.auditReceipt, nil
}

type mockDecisionPublisher struct{ entries []*store.DecisionEntry }

func (m *mockDecisionPublisher) AppendDecision(_ context.Context, _ string, entry *store.DecisionEntry) error {
	m.entries = append(m.entries, entry)
	return nil
}
func (m *mockDecisionPublisher) GetDecisions(_ context.Context, teamID, contextTag, status string, _ int) ([]store.DecisionEntry, int, error) {
	if teamID != "meta-optimization" {
		return nil, 0, errors.New("unexpected team")
	}
	result := make([]store.DecisionEntry, 0, len(m.entries))
	for _, entry := range m.entries {
		if (contextTag == "" || entry.Context == contextTag) && (status == "" || entry.Status == status) {
			result = append(result, *entry)
		}
	}
	return result, len(result), nil
}

func newMockExperimentStore() *mockExperimentStore {
	return &mockExperimentStore{
		experiments: make(map[string]*store.Experiment),
		outcomes:    make(map[string][]store.ExperimentOutcome),
	}
}

func validProtocol() store.ExperimentProtocol {
	return store.ExperimentProtocol{
		Population: "reference workflow", RandomizationUnit: "workflow-node-per-execution",
		PrimaryMetric: "evaluator verdict", EffectThreshold: 0.01,
		ExposurePolicy: "exclude-contaminated", OutcomeCompletenessThreshold: 0.9,
		Budget: "100 runs", StoppingRule: "fixed sample", HoldoutRequired: true, HoldoutPopulationHash: "sha256:holdout-population",
		PromotionAuthority: "operator", EvaluatorRubricHash: "sha256:rubric",
		EvaluatorAuthor: "independent-evaluator",
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
		ID:         "exp-1",
		SkillID:    "s1",
		Name:       "Test Experiment",
		Hypothesis: "variant improves evaluator verdict",
		Protocol:   validProtocol(),
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
	publisher := &mockDecisionPublisher{}
	h.SetDecisionPublisher(publisher)
	h.SetAuditSecret([]byte("test-audit-secret"))

	// Create
	if err := es.Create(context.Background(), &store.Experiment{
		ID:       "exp-1",
		SkillID:  "s1",
		Name:     "Test",
		Protocol: validProtocol(),
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
	es.assignments = append(es.assignments, store.ExperimentAssignment{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1", ExecutionID: "execution-1", NodeID: "treatment", IdempotencyKey: "assignment-1"})
	success := true
	outcomeBody, _ := json.Marshal(RecordOutcomeRequest{
		VariantID:     "v1",
		Source:        "test",
		SchemaVersion: 1,
		Data:          json.RawMessage(`{"classification":"ready"}`),
		Controlled:    &store.ControlledExperimentOutcome{AssignmentID: "assignment-1", ExecutionID: "execution-1", EvaluatorAttemptID: "eval-attempt", EvaluatorRunID: "eval-run", Verdict: "pass", Success: &success, OutcomeStatus: "complete", RubricHash: "sha256:rubric", EvaluatorPromptHash: "sha256:evaluator", StructuredSchemaHash: "sha256:schema"},
	})
	req2 := httptest.NewRequest("POST", "/experiments/exp-1/outcomes", bytes.NewReader(outcomeBody))
	req2 = mux.SetURLVars(req2, map[string]string{"eid": "exp-1"})
	w2 := httptest.NewRecorder()
	h.RecordOutcome(w2, req2)

	if w2.Code != http.StatusNoContent {
		t.Fatalf("outcome: expected 204, got %d: %s", w2.Code, w2.Body.String())
	}
	controlSuccess := false
	es.assignments = append(es.assignments, store.ExperimentAssignment{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control", ExecutionID: "execution-2", NodeID: "treatment", IdempotencyKey: "assignment-control"})
	es.outcomes["exp-1"] = append(es.outcomes["exp-1"], store.ExperimentOutcome{VariantID: "control", Controlled: &store.ControlledExperimentOutcome{AssignmentID: "assignment-control", ExecutionID: "execution-2", EvaluatorAttemptID: "eval-control", EvaluatorRunID: "eval-run-control", Verdict: "fail", Success: &controlSuccess, OutcomeStatus: "complete", RubricHash: "sha256:rubric", EvaluatorPromptHash: "sha256:evaluator", StructuredSchemaHash: "sha256:schema"}})

	// Conclude
	audit := &store.ExperimentAuditReceipt{ExperimentID: "exp-1", ProtocolHash: es.experiments["exp-1"].Protocol.ProtocolHash, SampledAssignmentIDs: []string{"attempt-1"}, FindingsHash: "sha256:findings", ChallengeState: "clear", CompletedAt: "2026-01-01T00:00:00Z", IdempotencyKey: "audit/exp-1"}
	audit.Signature = h.signAuditReceipt(audit)
	es.auditReceipt = audit
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
	if concludeResp.PromotionDecisionID == "" || len(publisher.entries) != 1 || publisher.entries[0].Status != store.DecisionStatusPending {
		t.Fatalf("expected one pending promotion decision, got %#v", publisher.entries)
	}
	if ss.updates != 0 {
		t.Fatalf("conclude must create a recommendation only; skill update calls = %d", ss.updates)
	}

	// A signed holdout still cannot promote without the exact decision accepted
	// in prompt-manager's meta-optimization team.
	holdoutBody, _ := json.Marshal(RecordHoldoutReceiptRequest{FindingsHash: "sha256:holdout", IdempotencyKey: "holdout/exp-1"})
	holdoutReq := httptest.NewRequest("POST", "/experiments/exp-1/holdout-receipt", bytes.NewReader(holdoutBody))
	holdoutReq = mux.SetURLVars(holdoutReq, map[string]string{"eid": "exp-1"})
	holdoutW := httptest.NewRecorder()
	h.RecordHoldoutReceipt(holdoutW, holdoutReq)
	if holdoutW.Code != http.StatusOK {
		t.Fatalf("holdout: expected 200, got %d: %s", holdoutW.Code, holdoutW.Body.String())
	}
	promoteBody, _ := json.Marshal(PromoteExperimentRequest{DecisionID: concludeResp.PromotionDecisionID})
	promoteReq := httptest.NewRequest("POST", "/experiments/exp-1/promote", bytes.NewReader(promoteBody))
	promoteReq = mux.SetURLVars(promoteReq, map[string]string{"eid": "exp-1"})
	promoteW := httptest.NewRecorder()
	h.PromoteExperiment(promoteW, promoteReq)
	if promoteW.Code != http.StatusForbidden {
		t.Fatalf("promotion without acceptance: expected 403, got %d: %s", promoteW.Code, promoteW.Body.String())
	}
	publisher.entries[0].Status = store.DecisionStatusAccepted
	promoteReq = httptest.NewRequest("POST", "/experiments/exp-1/promote", bytes.NewReader(promoteBody))
	promoteReq = mux.SetURLVars(promoteReq, map[string]string{"eid": "exp-1"})
	promoteW = httptest.NewRecorder()
	h.PromoteExperiment(promoteW, promoteReq)
	if promoteW.Code != http.StatusOK {
		t.Fatalf("accepted promotion: expected 200, got %d: %s", promoteW.Code, promoteW.Body.String())
	}
	if ss.updates != 1 {
		t.Fatalf("accepted variant must update skill exactly once; got %d", ss.updates)
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
	success := true
	es.outcomes["exp-1"] = []store.ExperimentOutcome{
		{VariantID: "control", Source: "test", SchemaVersion: 1, RecordedAt: "t1", Data: json.RawMessage(`{}`), Controlled: &store.ControlledExperimentOutcome{AssignmentID: "assignment-1", OutcomeStatus: "complete", Success: &success}},
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
	if resp[0].Controlled == nil || resp[0].Controlled.AssignmentID != "assignment-1" {
		t.Fatalf("expected controlled evaluator provenance in response, got %#v", resp[0].Controlled)
	}
}
