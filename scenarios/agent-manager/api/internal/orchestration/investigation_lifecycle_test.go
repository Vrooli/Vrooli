package orchestration

import (
	"encoding/json"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/investigation"
	"agent-manager/internal/runreport"

	"github.com/google/uuid"
)

func TestRunReportAvailabilityUsesInvocationFactsForInvocationPlane(t *testing.T) {
	report := &runreport.RunReport{
		InvocationValidity:     runreport.EvidenceValidity{Availability: runreport.Availability{State: runreport.AvailabilityAvailable}},
		ProjectionAvailability: runreport.Availability{State: runreport.AvailabilityUnavailable, Reason: "receipt reader is not configured"},
	}
	got, ok := runReportAvailability(report, "invocations")
	if !ok || got.State != runreport.AvailabilityAvailable {
		t.Fatalf("runReportAvailability(invocations) = (%+v, %v), want available invocation facts", got, ok)
	}
}

func TestTypedInvestigationResultPreservesSubjectSpecificAttribution(t *testing.T) {
	request := investigation.Request{
		SchemaVersion:   investigation.RequestSchemaVersion,
		RequestKey:      "subject-attribution-test",
		CallerAuthority: "test",
		Subject: investigation.Subject{
			Owner:    "agent-manager",
			Kind:     "run-set",
			Ref:      "subject-1",
			Revision: "revision-1",
			RunIDs:   []string{"run-a", "run-b"},
		},
		Question: "Which run needs attention?",
		EvidencePolicy: investigation.EvidencePolicy{
			Mode:               "bounded_current",
			MaxEvents:          1,
			MaxEvidenceBytes:   1,
			MaxReconciliations: 0,
		},
		Budget: investigation.Budget{
			MaxDelegatedRuns: 1,
			MaxTurns:         1,
			WallSeconds:      1,
		},
		RecommendationPolicy: investigation.RecommendationPolicy{AllowedKinds: []string{"observe"}},
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("request.Validate() error = %v", err)
	}

	item := &investigation.Lifecycle{ID: "investigation-1", Request: request}
	execution := &domain.WorkflowExecution{
		ID:               uuid.New(),
		WorkflowKey:      investigation.WorkflowKey,
		DefinitionDigest: "sha256:workflow-definition",
		Output:           json.RawMessage(`{"findings":{"condition":"stalled","disposition":"recommend_action","rootCause":"unknown","unprovenPredicates":["whether the provider remains unavailable"],"summary":"run-a needs attention","primaryCategory":"run-a is stalled","confidence":"High","categories":[{"name":"run-a stalled","subjectRunIds":["run-a"],"recommendations":[{"text":"observe run-a","subjectRunIds":["run-a"]}]}]}}`),
	}

	result := typedInvestigationResult(item, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if err := result.Validate(request); err != nil {
		t.Fatalf("result.Validate() error = %v", err)
	}
	if len(result.Findings) != 2 {
		t.Fatalf("findings = %d, want primary plus category", len(result.Findings))
	}
	category := result.Findings[1]
	if result.Findings[0].ID == category.ID {
		t.Fatalf("findings reused ID %q", category.ID)
	}
	if category.Relation != investigation.FindingImplicated {
		t.Fatalf("category relation = %q, want %q", category.Relation, investigation.FindingImplicated)
	}
	if len(category.SubjectRunIDs) != 1 || category.SubjectRunIDs[0] != "run-a" {
		t.Fatalf("category subjectRunIds = %#v, want [run-a]", category.SubjectRunIDs)
	}
	if len(result.Recommendations) != 1 || len(result.Recommendations[0].SubjectRunIDs) != 1 || result.Recommendations[0].SubjectRunIDs[0] != "run-a" {
		t.Fatalf("recommendation subjectRunIds = %#v, want [run-a]", result.Recommendations)
	}
	if result.Diagnosis.Condition != investigation.ConditionStalled || result.Diagnosis.Disposition != investigation.DispositionRecommendAction || result.Diagnosis.RootCause != investigation.RootCauseUnknown || len(result.Diagnosis.UnprovenPredicates) != 1 {
		t.Fatalf("diagnosis fields = %#v, want mapped typed outcome", result.Diagnosis)
	}
}

func TestTypedInvestigationResultCarriesStableLearningAttemptIdentity(t *testing.T) {
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "learning-attempt-test", CallerAuthority: "test",
		Subject:        investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:       "Classify the run.",
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"workflow_result"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
	execution := &domain.WorkflowExecution{ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition", Output: json.RawMessage(`{"findings":{"summary":"The run completed."}}`)}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "learning-investigation", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if got, want := result.Learning.AttemptID, "learning-investigation/attempt-1"; got != want {
		t.Fatalf("learning attempt id = %q, want %q", got, want)
	}
	if err := result.Validate(request); err != nil {
		t.Fatalf("result.Validate() error = %v", err)
	}
}

func TestTypedInvestigationResultDeclaresRequiredPlanesWithoutOverclaimingCompleteness(t *testing.T) {
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "coverage-test", CallerAuthority: "test",
		Subject:        investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:       "Classify the run.",
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"run_state", "events", "invocations"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
	execution := &domain.WorkflowExecution{ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition", Output: json.RawMessage(`{"findings":{"summary":"The cause remains unknown.","primaryCategory":"Unknown","categories":[]}}`)}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "coverage-investigation", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if err := result.Validate(request); err != nil {
		t.Fatalf("result.Validate() error = %v", err)
	}
	for _, required := range request.EvidencePolicy.RequiredPlanes {
		found := false
		for _, plane := range result.Coverage {
			if plane.Plane == required {
				found = true
				if plane.State != investigation.CoveragePartial || plane.Reason == "" {
					t.Fatalf("coverage[%s] = %+v, want explained partial coverage", required, plane)
				}
			}
		}
		if !found {
			t.Fatalf("required coverage plane %q missing from %+v", required, result.Coverage)
		}
	}
}

func TestTypedInvestigationResultDowngradesUnsupportedNoIntervention(t *testing.T) {
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "normalize-unsupported", CallerAuthority: "test",
		Subject:        investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:       "Classify the run.",
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"run_state", "events"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition",
		Output: json.RawMessage(`{"findings":{"condition":"progressing","disposition":"no_intervention","rootCause":"not_applicable","summary":"The run is progressing.","primaryCategory":"progressing","confidence":"High","categories":[]}}`),
	}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "normalize-unsupported", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if result.Diagnosis.Disposition != investigation.DispositionInconclusive || result.Diagnosis.Confidence != "Low" {
		t.Fatalf("diagnosis = %#v, want low-confidence inconclusive", result.Diagnosis)
	}
	if len(result.Diagnosis.UnprovenPredicates) != 1 {
		t.Fatalf("unproven predicates = %#v, want one coverage predicate", result.Diagnosis.UnprovenPredicates)
	}
	if err := result.Validate(request); err != nil {
		t.Fatalf("normalized result rejected: %v", err)
	}
}

func TestTypedInvestigationResultDowngradesUnknownFailedNoIntervention(t *testing.T) {
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "normalize-failed", CallerAuthority: "test",
		Subject:        investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:       "Classify the run.",
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"workflow_result"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition",
		Output: json.RawMessage(`{"findings":{"condition":"failed","disposition":"no_intervention","rootCause":"unknown","summary":"The run failed.","primaryCategory":"failed","confidence":"High","categories":[]}}`),
	}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "normalize-failed", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if result.Diagnosis.Disposition != investigation.DispositionInconclusive || result.Diagnosis.Confidence != "Low" {
		t.Fatalf("diagnosis = %#v, want low-confidence inconclusive", result.Diagnosis)
	}
	if len(result.Diagnosis.UnprovenPredicates) != 1 || result.Diagnosis.UnprovenPredicates[0] != "which causal predicate explains the terminal failure" {
		t.Fatalf("unproven predicates = %#v", result.Diagnosis.UnprovenPredicates)
	}
	if err := result.Validate(request); err != nil {
		t.Fatalf("normalized result rejected: %v", err)
	}
}

func TestTypedInvestigationResultDowngradesUnprovenRecommendation(t *testing.T) {
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "normalize-recommendation", CallerAuthority: "test",
		Subject:              investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:             "Classify the run.",
		EvidencePolicy:       investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"workflow_result"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:               investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
		RecommendationPolicy: investigation.RecommendationPolicy{AllowedKinds: []string{"observe"}},
	}
	execution := &domain.WorkflowExecution{
		ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition",
		Output: json.RawMessage(`{"findings":{"condition":"failed","disposition":"recommend_action","rootCause":"unknown","summary":"The run failed.","primaryCategory":"failed","confidence":"Low","unprovenPredicates":["the failure cause"],"categories":[{"name":"failed","recommendations":[{"text":"repair the run"}]}]}}`),
	}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "normalize-recommendation", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if result.Diagnosis.Disposition != investigation.DispositionInconclusive || result.Diagnosis.Confidence != "Low" {
		t.Fatalf("diagnosis = %#v, want inconclusive", result.Diagnosis)
	}
	if len(result.Recommendations) != 0 {
		t.Fatalf("unsupported recommendation survived projection: %#v", result.Recommendations)
	}
	if err := result.Validate(request); err != nil {
		t.Fatalf("normalized result rejected: %v", err)
	}
}

func TestTypedInvestigationResultDoesNotTrustFabricatedEvidenceIDs(t *testing.T) {
	request := investigation.Request{
		SchemaVersion:   investigation.RequestSchemaVersion,
		RequestKey:      "fabricated-evidence-test",
		CallerAuthority: "test",
		Subject:         investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a"}},
		Question:        "Is the run healthy?",
		EvidencePolicy:  investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"workflow_result"}, MaxEvents: 1, MaxEvidenceBytes: 1024, MaxReconciliations: 0},
		Budget:          investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1},
	}
	execution := &domain.WorkflowExecution{
		ID:               uuid.New(),
		WorkflowKey:      investigation.WorkflowKey,
		DefinitionDigest: "sha256:workflow-definition",
		Output:           json.RawMessage(`{"findings":{"summary":"The run is progressing.","evidenceRefs":[{"owner":"model","kind":"event","ref":"does-not-exist","revision":"spoofed","schemaVersion":"event/v1"}]}}`),
	}

	result := typedInvestigationResult(&investigation.Lifecycle{ID: "investigation-fabricated", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if err := result.Validate(request); err != nil {
		t.Fatalf("result.Validate() error = %v", err)
	}
	if len(result.EvidenceRefs) != 1 || result.EvidenceRefs[0].Ref != execution.ID.String() {
		t.Fatalf("evidence refs = %#v, want only the durable workflow execution reference", result.EvidenceRefs)
	}
	for _, finding := range result.Findings {
		for _, ref := range finding.EvidenceRefs {
			if ref.Ref == "does-not-exist" {
				t.Fatal("fabricated model evidence reference survived typed projection")
			}
		}
	}
}

func TestTypedInvestigationResultRetainsOnlyKnownSubjectEvidence(t *testing.T) {
	evidence := investigation.EvidenceReference{Owner: "plan-manager", Kind: "brief", Ref: "brief-1", Revision: "7", SchemaVersion: "brief/v1", SubjectRunIDs: []string{"run-a"}}
	request := investigation.Request{
		SchemaVersion: investigation.RequestSchemaVersion, RequestKey: "known-evidence-test", CallerAuthority: "test",
		Subject:  investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1", Revision: "revision-1", RunIDs: []string{"run-a", "run-b"}},
		Question: "Which run needs attention?", DomainEvidence: []investigation.EvidenceReference{evidence},
		EvidencePolicy: investigation.EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"workflow_result"}, MaxEvents: 1, MaxEvidenceBytes: 1024},
		Budget:         investigation.Budget{MaxDelegatedRuns: 1, MaxTurns: 1, WallSeconds: 1}, RecommendationPolicy: investigation.RecommendationPolicy{AllowedKinds: []string{"observe"}},
	}
	if err := request.Validate(); err != nil {
		t.Fatal(err)
	}
	execution := &domain.WorkflowExecution{ID: uuid.New(), WorkflowKey: investigation.WorkflowKey, DefinitionDigest: "sha256:workflow-definition", Output: json.RawMessage(`{"findings":{"condition":"stalled","disposition":"recommend_action","rootCause":"unknown","confidence":"High","summary":"run-a is stalled","primaryCategory":"run-a stalled","categories":[{"name":"run-a stalled","subjectRunIds":["run-a"],"evidenceRefs":[{"owner":"plan-manager","kind":"brief","ref":"brief-1","revision":"7","schemaVersion":"brief/v1","subjectRunIds":["run-a"]},{"owner":"model","kind":"event","ref":"fabricated","revision":"x","schemaVersion":"event/v1"}],"recommendations":[{"text":"observe run-a","subjectRunIds":["run-a"],"evidenceRefs":[{"owner":"plan-manager","kind":"brief","ref":"brief-1","revision":"7","schemaVersion":"brief/v1","subjectRunIds":["run-a"]}]}]}]}}`)}
	result := typedInvestigationResult(&investigation.Lifecycle{ID: "known-evidence-investigation", Request: request}, execution, time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC))
	if err := result.Validate(request); err != nil {
		t.Fatalf("result.Validate() error = %v", err)
	}
	if len(result.EvidenceRefs) != 2 {
		t.Fatalf("result evidence refs=%#v, want workflow plus caller evidence", result.EvidenceRefs)
	}
	if len(result.Findings) < 2 || len(result.Findings[1].EvidenceRefs) != 2 || result.Findings[1].EvidenceRefs[1].Ref != "brief-1" {
		t.Fatalf("finding evidence refs=%#v, want known brief citation", result.Findings)
	}
	if len(result.Recommendations) != 1 || len(result.Recommendations[0].EvidenceRefs) != 2 || result.Recommendations[0].EvidenceRefs[1].Ref != "brief-1" {
		t.Fatalf("recommendation evidence refs=%#v, want known brief citation", result.Recommendations)
	}
}
