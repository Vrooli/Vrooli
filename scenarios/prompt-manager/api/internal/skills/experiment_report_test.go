package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"prompt-manager/internal/store"

	"github.com/gorilla/mux"
)

func TestBuildExperimentReport(t *testing.T) {
	exp := &store.Experiment{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Test",
		Status:  store.ExperimentStatusRunning,
		Arms: []store.ExperimentArm{
			{VariantID: "control", Weight: 0.4},
			{VariantID: "v1", Weight: 0.4},
			{VariantID: "v2", Weight: 0.2},
		},
	}

	serves := []store.ExperimentServe{
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "rogue"},
	}

	outcomes := []store.ExperimentOutcome{
		{VariantID: "control", Source: "agent-manager", Data: json.RawMessage(`{"runId":"r1","status":"complete","tokensUsed":100}`)},
		{VariantID: "control", Source: "agent-manager", Data: json.RawMessage(`{"runId":"r2","status":"failed","tokensUsed":300}`)},
		{VariantID: "control", Source: "agent-manager", Data: json.RawMessage(`{}`)},
		{VariantID: "v1", Source: "agent-manager", Data: json.RawMessage(`{"runId":"r3","status":"complete","tokensUsed":50}`)},
		{VariantID: "v1", Source: "agent-manager", Data: json.RawMessage(`{"runId":"r4","status":"complete"}`)},
	}

	report := buildExperimentReport(exp, serves, outcomes)

	if report.ExperimentID != "exp-1" || report.SkillID != "s1" || report.Status != store.ExperimentStatusRunning {
		t.Errorf("unexpected report header: %+v", report)
	}
	if report.TotalServes != 6 {
		t.Errorf("expected 6 total serves, got %d", report.TotalServes)
	}
	if report.TotalOutcomes != 5 {
		t.Errorf("expected 5 total outcomes, got %d", report.TotalOutcomes)
	}

	// Declared arms in order, undeclared "rogue" appended.
	if len(report.Arms) != 4 {
		t.Fatalf("expected 4 arms, got %d", len(report.Arms))
	}
	for i, want := range []string{"control", "v1", "v2", "rogue"} {
		if report.Arms[i].VariantID != want {
			t.Errorf("arm %d: expected %q, got %q", i, want, report.Arms[i].VariantID)
		}
	}

	control := report.Arms[0]
	if control.Serves != 3 || control.Outcomes != 3 {
		t.Errorf("control: expected serves=3 outcomes=3, got serves=%d outcomes=%d", control.Serves, control.Outcomes)
	}
	if control.StatusCounts["complete"] != 1 || control.StatusCounts["failed"] != 1 || control.StatusCounts["unknown"] != 1 {
		t.Errorf("control: unexpected status counts: %v", control.StatusCounts)
	}
	if control.SuccessRate != nil {
		t.Errorf("control: terminal state must not produce a primary success rate, got %v", control.SuccessRate)
	}
	if control.MeanTokensUsed == nil || *control.MeanTokensUsed != 200 {
		t.Errorf("control: expected mean tokens 200, got %v", control.MeanTokensUsed)
	}

	v1 := report.Arms[1]
	if v1.Serves != 2 || v1.Outcomes != 2 {
		t.Errorf("v1: expected serves=2 outcomes=2, got serves=%d outcomes=%d", v1.Serves, v1.Outcomes)
	}
	if v1.SuccessRate != nil {
		t.Errorf("v1: terminal state must not produce a primary success rate, got %v", v1.SuccessRate)
	}
	// Only one outcome carried tokensUsed
	if v1.MeanTokensUsed == nil || *v1.MeanTokensUsed != 50 {
		t.Errorf("v1: expected mean tokens 50, got %v", v1.MeanTokensUsed)
	}

	v2 := report.Arms[2]
	if v2.Serves != 0 || v2.Outcomes != 0 {
		t.Errorf("v2: expected zero data, got serves=%d outcomes=%d", v2.Serves, v2.Outcomes)
	}
	if v2.SuccessRate != nil || v2.MeanTokensUsed != nil {
		t.Errorf("v2: expected nil rate/tokens, got %v/%v", v2.SuccessRate, v2.MeanTokensUsed)
	}
	if len(report.ZeroDataArms) != 1 || report.ZeroDataArms[0] != "v2" {
		t.Errorf("expected zeroDataArms=[v2], got %v", report.ZeroDataArms)
	}

	rogue := report.Arms[3]
	if rogue.Serves != 1 || rogue.Outcomes != 0 {
		t.Errorf("rogue: expected serves=1 outcomes=0, got serves=%d outcomes=%d", rogue.Serves, rogue.Outcomes)
	}
}

func TestBuildExperimentReport_NoData(t *testing.T) {
	exp := &store.Experiment{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Test",
		Status:  store.ExperimentStatusDraft,
		Arms: []store.ExperimentArm{
			{VariantID: "control", Weight: 0.5},
			{VariantID: "v1", Weight: 0.5},
		},
	}

	report := buildExperimentReport(exp, nil, nil)

	if report.TotalServes != 0 || report.TotalOutcomes != 0 {
		t.Errorf("expected zero totals, got serves=%d outcomes=%d", report.TotalServes, report.TotalOutcomes)
	}
	if len(report.Arms) != 2 {
		t.Fatalf("expected 2 arms, got %d", len(report.Arms))
	}
	if len(report.ZeroDataArms) != 2 {
		t.Errorf("expected both arms in zeroDataArms, got %v", report.ZeroDataArms)
	}
}

func TestBuildControlledReport_ExcludesContaminatedAndIncompleteEvidence(t *testing.T) {
	exp := &store.Experiment{ID: "exp-1", SkillID: "s1", Arms: []store.ExperimentArm{{VariantID: "control"}, {VariantID: "v1"}}}
	assignments := []store.ExperimentAssignment{
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "control", ExecutionID: "e1", NodeID: "treat", IdempotencyKey: "a1"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1", ExecutionID: "e2", NodeID: "treat", IdempotencyKey: "a2"},
		{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1", ExecutionID: "e3", NodeID: "treat", IdempotencyKey: "a3"},
	}
	ok := true
	outcomes := []store.ExperimentOutcome{
		{VariantID: "control", Controlled: &store.ControlledExperimentOutcome{AssignmentID: "a1", OutcomeStatus: "complete", Success: &ok}},
		{VariantID: "v1", Controlled: &store.ControlledExperimentOutcome{AssignmentID: "a2", OutcomeStatus: "incomplete"}},
		{VariantID: "v1", Controlled: &store.ControlledExperimentOutcome{AssignmentID: "a3", OutcomeStatus: "complete", Success: &ok}},
	}
	exposures := []store.ExperimentExposure{{ExperimentID: "exp-1", VariantID: "control", ExecutionID: "e3", NodeID: "other", ReadSkillID: "s1"}}
	report := buildControlledReport(exp, assignments, exposures, outcomes)
	if report.Assignments != 3 || report.EligibleAssignments != 1 || report.IncompleteAssignments != 1 || report.ExcludedAssignments != 1 {
		t.Fatalf("unexpected evidence accounting: %+v", report)
	}
	if report.ExclusionReasons["contaminated"] != 1 || report.ExclusionReasons["incomplete-outcome"] != 1 {
		t.Fatalf("expected explicit exclusions, got %+v", report.ExclusionReasons)
	}
	if report.Arms[0].PosteriorMean == nil || report.Arms[1].PosteriorMean != nil {
		t.Fatalf("only clean complete evaluator evidence may enter primary estimates: %+v", report.Arms)
	}
}

func TestBuildControlledReport_JoinsAgentAttemptProvenanceToDispatchAssignment(t *testing.T) {
	exp := &store.Experiment{ID: "exp-1", SkillID: "s1", Arms: []store.ExperimentArm{{VariantID: "v1"}}}
	assignment := store.ExperimentAssignment{ExperimentID: "exp-1", SkillID: "s1", VariantID: "v1", ExecutionID: "execution-1", NodeID: "treatment", IdempotencyKey: "workflow-assignment/execution-1/node/treatment"}
	success := true
	outcome := store.ExperimentOutcome{VariantID: "v1", Controlled: &store.ControlledExperimentOutcome{AssignmentID: "agent-attempt-uuid", ExecutionID: "execution-1", OutcomeStatus: "complete", Success: &success}}
	report := buildControlledReport(exp, []store.ExperimentAssignment{assignment}, nil, []store.ExperimentOutcome{outcome})
	if report.EligibleAssignments != 1 || report.Arms[0].Successes != 1 {
		t.Fatalf("agent attempt provenance must join its unique durable assignment: %+v", report)
	}
}

func TestExperimentHandlers_GetExperimentReport(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	ss.skills["s1"] = &store.Skill{ID: "s1", Name: "S1", Pack: "local"}
	if err := vs.Create(context.Background(), "s1", &store.Variant{ID: "v1", Name: "V1"}, "content"); err != nil {
		t.Fatal(err)
	}
	h := NewExperimentHandlers(es, vs, ss)

	if err := es.Create(context.Background(), &store.Experiment{
		ID:      "exp-1",
		SkillID: "s1",
		Name:    "Test",
		Status:  store.ExperimentStatusRunning,
		Arms: []store.ExperimentArm{
			{VariantID: "control", Weight: 0.5},
			{VariantID: "v1", Weight: 0.5},
		},
	}); err != nil {
		t.Fatal(err)
	}

	for _, vid := range []string{"control", "v1", "v1"} {
		if err := es.RecordServe(context.Background(), store.ExperimentServe{ExperimentID: "exp-1", SkillID: "s1", VariantID: vid}); err != nil {
			t.Fatal(err)
		}
	}
	if err := es.RecordOutcome(context.Background(), "exp-1", store.ExperimentOutcome{
		VariantID: "v1",
		Source:    "agent-manager",
		Data:      json.RawMessage(`{"runId":"r1","status":"complete","tokensUsed":42}`),
	}); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/experiments/exp-1/report", nil)
	req = mux.SetURLVars(req, map[string]string{"eid": "exp-1"})
	w := httptest.NewRecorder()
	h.GetExperimentReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("report: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var report ExperimentReportResponse
	if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.TotalServes != 3 || report.TotalOutcomes != 1 {
		t.Errorf("expected serves=3 outcomes=1, got serves=%d outcomes=%d", report.TotalServes, report.TotalOutcomes)
	}
	if len(report.Arms) != 2 {
		t.Fatalf("expected 2 arms, got %d", len(report.Arms))
	}
	if report.Arms[0].VariantName != "control (original)" {
		t.Errorf("expected control arm name %q, got %q", "control (original)", report.Arms[0].VariantName)
	}
	if report.Arms[1].VariantName != "V1" {
		t.Errorf("expected v1 arm name %q, got %q", "V1", report.Arms[1].VariantName)
	}
	if report.Arms[1].SuccessRate != nil {
		t.Errorf("expected no terminal-status success rate, got %v", report.Arms[1].SuccessRate)
	}
	if len(report.ZeroDataArms) != 0 {
		t.Errorf("expected no zero-data arms, got %v", report.ZeroDataArms)
	}
}

func TestExperimentHandlers_GetExperimentReportNotFound(t *testing.T) {
	es := newMockExperimentStore()
	vs := newMockVariantStore()
	ss := newMockPackSkillStore()
	h := NewExperimentHandlers(es, vs, ss)

	req := httptest.NewRequest("GET", "/experiments/nonexistent/report", nil)
	req = mux.SetURLVars(req, map[string]string{"eid": "nonexistent"})
	w := httptest.NewRecorder()
	h.GetExperimentReport(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
