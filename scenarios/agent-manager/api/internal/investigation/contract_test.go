package investigation

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func validRequest() Request {
	return Request{
		SchemaVersion:        RequestSchemaVersion,
		RequestKey:           "run-set-efficiency-1",
		CallerAuthority:      "service",
		Subject:              Subject{Owner: "agent-manager", Kind: "run-set", Ref: "set-1", Revision: "17", RunIDs: []string{"run-a", "run-b"}},
		Question:             "Are tool calls efficient?",
		EvidencePolicy:       EvidencePolicy{Mode: "bounded_current", RequiredPlanes: []string{"events"}, MaxEvents: 128, MaxEvidenceBytes: 65536, MaxReconciliations: 1},
		Budget:               Budget{MaxDelegatedRuns: 1, MaxTurns: 12, WallSeconds: 600, MaxChargeMicroUSD: 1000000},
		RecommendationPolicy: RecommendationPolicy{AllowedKinds: []string{"observe", "request_evidence", "nudge", "escalate"}},
	}
}

func validEvidence(ref string) EvidenceReference {
	return EvidenceReference{Owner: "agent-manager", Kind: "run-report", Ref: ref, Revision: "17", SchemaVersion: "run-report/v1"}
}

func validResult(req Request) Result {
	evidence := validEvidence("run-a-report")
	return Result{
		SchemaVersion: ResultSchemaVersion, InvestigationID: "inv-1", OperationStatus: OperationCompleted,
		Diagnosis:   Diagnosis{Condition: ConditionProgressing, Disposition: DispositionNoIntervention, Summary: "The run is making compatible progress.", RootCause: RootCauseNotApplicable, Confidence: "supported"},
		EvidenceCut: EvidenceCut{ID: "cut-1", SubjectRevision: req.Subject.Revision, RunEventHeads: map[string]string{"run-a": "event-1"}, CapturedAt: "2026-09-06T12:00:00Z"},
		Coverage:    []CoveragePlane{{Plane: "events", State: CoverageComplete, ThroughRef: "event-1"}},
		Findings:    []Finding{}, Recommendations: []Recommendation{}, Applicability: Applicability{State: ApplicabilityCurrent, CheckedRevision: req.Subject.Revision, CheckedAt: "2026-09-06T12:00:01Z"},
		EvidenceRefs: []EvidenceReference{evidence}, Learning: Learning{AttemptID: "inv-1/attempt-1", CaptureState: "recorded", AdviceVerdict: "unknown"},
	}
}

func TestCleanResultAcceptsZeroRecommendations(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	if err := result.Validate(req); err != nil {
		t.Fatalf("clean result rejected: %v", err)
	}
}

func TestUnknownCoverageIsNotHealthyZero(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	result.Coverage = []CoveragePlane{{Plane: "events", State: CoverageComplete}, {Plane: "receipts", State: CoverageUnavailable}}
	if err := result.Validate(req); err == nil || !errors.Is(err, ErrInvalidResult) {
		t.Fatal("unexplained unavailable coverage was accepted")
	}
	result.Coverage[1].Reason = "receipt producer is unavailable"
	if err := result.Validate(req); err != nil {
		t.Fatalf("explicit unavailable coverage rejected: %v", err)
	}
}

func TestSupportedHealthyDiagnosisRequiresCompleteRequiredCoverage(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	result.Coverage = []CoveragePlane{{Plane: "events", State: CoverageLagging, Reason: "projection is behind the requested cut"}}
	if err := result.Validate(req); err == nil || !errors.Is(err, ErrInvalidResult) {
		t.Fatal("supported healthy diagnosis accepted lagging required coverage")
	}
	result.Coverage = []CoveragePlane{{Plane: "events", State: CoverageComplete}}
	if err := result.Validate(req); err != nil {
		t.Fatalf("complete required coverage rejected: %v", err)
	}
}

func TestResultRequiresEveryRequestedEvidencePlane(t *testing.T) {
	req := validRequest()
	req.EvidencePolicy.RequiredPlanes = []string{"events", "receipts"}
	if err := validResult(req).Validate(req); err == nil || !errors.Is(err, ErrInvalidResult) {
		t.Fatal("result without every requested evidence plane was accepted")
	}
}

func TestResultRequiresDeclaredConfidence(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	result.Diagnosis.Confidence = "certain"
	if err := result.Validate(req); err == nil || !errors.Is(err, ErrInvalidResult) {
		t.Fatal("undeclared confidence was accepted")
	}
}

func TestInvalidAuthorityAndSchemaRefuseBeforeDispatch(t *testing.T) {
	req := validRequest()
	req.CallerAuthority = "untrusted-client"
	if err := req.Validate(); err == nil || !errors.Is(err, ErrInvalidRequest) {
		t.Fatal("invalid authority was accepted")
	}
	req = validRequest()
	req.SchemaVersion = "investigation-request/v0"
	if err := req.Validate(); err == nil || !errors.Is(err, ErrInvalidRequest) {
		t.Fatal("incompatible schema was accepted")
	}
}

func TestGenericRunSetSerializesWithoutPlanSpecificFields(t *testing.T) {
	req := validRequest()
	raw, err := marshalRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || containsString(string(raw), "plan-manager") {
		t.Fatalf("generic request unexpectedly contains plan-specific branch: %s", raw)
	}
	if _, err := DecodeRequest(raw); err != nil {
		t.Fatalf("serialized generic request did not round-trip: %v", err)
	}
}

func TestImplicationMustNameOnlySubjectRunsAndUseTypedEvidence(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	evidence := validEvidence("run-a-report")
	evidence.SubjectRunIDs = []string{"run-a"}
	result.Diagnosis.Disposition = DispositionRecommendAction
	result.EvidenceRefs = []EvidenceReference{evidence}
	result.Findings = []Finding{{ID: "finding-a", SubjectRunIDs: []string{"run-a"}, Relation: FindingImplicated, Summary: "Run A has a bounded tool failure.", EvidenceRefs: []EvidenceReference{evidence}}}
	result.Recommendations = []Recommendation{{ID: "rec-a", Kind: "observe", SubjectRunIDs: []string{"run-a"}, Text: "Observe the next bounded attempt.", EvidenceRefs: []EvidenceReference{evidence}}}
	if err := result.Validate(req); err != nil {
		t.Fatalf("typed implicated result rejected: %v", err)
	}
	result.Findings[0].SubjectRunIDs = []string{"run-c"}
	if err := result.Validate(req); err == nil {
		t.Fatal("finding outside subject was accepted")
	}
}

func TestEvidenceBoundToAnotherSubjectCannotSupportFinding(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	evidence := validEvidence("run-b-report")
	evidence.SubjectRunIDs = []string{"run-b"}
	result.Diagnosis.Disposition = DispositionRecommendAction
	result.EvidenceRefs = []EvidenceReference{evidence}
	result.Findings = []Finding{{ID: "finding-a", SubjectRunIDs: []string{"run-a"}, Relation: FindingImplicated, Summary: "Run A has a bounded tool failure.", EvidenceRefs: []EvidenceReference{evidence}}}
	result.Recommendations = []Recommendation{{ID: "rec-a", Kind: "observe", SubjectRunIDs: []string{"run-a"}, Text: "Observe the next bounded attempt.", EvidenceRefs: []EvidenceReference{evidence}}}
	if err := result.Validate(req); err == nil || !strings.Contains(err.Error(), "different subject run") {
		t.Fatalf("evidence from an unrelated subject was accepted: %v", err)
	}
}

func TestRunScopedFindingRequiresBoundEvidence(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	evidence := validEvidence("run-a-report")
	result.Diagnosis.Disposition = DispositionRecommendAction
	result.Findings = []Finding{{ID: "finding-a", SubjectRunIDs: []string{"run-a"}, Relation: FindingImplicated, Summary: "Run A has a bounded tool failure.", EvidenceRefs: []EvidenceReference{evidence}}}
	result.Recommendations = []Recommendation{{ID: "rec-a", Kind: "observe", SubjectRunIDs: []string{"run-a"}, Text: "Observe the next bounded attempt.", EvidenceRefs: []EvidenceReference{evidence}}}
	if err := result.Validate(req); err == nil || !strings.Contains(err.Error(), "subject-scoped references must declare subject run ids") {
		t.Fatalf("unbound evidence supported a run-scoped claim: %v", err)
	}
}

func TestMixedCauseRequiresDistinctEvidenceChains(t *testing.T) {
	req := validRequest()
	result := validResult(req)
	result.Diagnosis.RootCause = RootCauseMixed
	first := validEvidence("run-a-report")
	first.SubjectRunIDs = []string{"run-a"}
	result.EvidenceRefs = []EvidenceReference{first}
	result.Findings = []Finding{{ID: "finding-a", SubjectRunIDs: []string{"run-a"}, Relation: FindingImplicated, Summary: "Mixed contribution.", EvidenceRefs: []EvidenceReference{first}}}
	if err := result.Validate(req); err == nil {
		t.Fatal("mixed cause with one evidence chain was accepted")
	}
	second := validEvidence("run-b-report")
	second.SubjectRunIDs = []string{"run-b"}
	result.EvidenceRefs = append(result.EvidenceRefs, second)
	result.Findings[0].SubjectRunIDs = []string{"run-a", "run-b"}
	result.Findings[0].EvidenceRefs = append(result.Findings[0].EvidenceRefs, second)
	if err := result.Validate(req); err != nil {
		t.Fatalf("mixed cause with distinct evidence chains rejected: %v", err)
	}
}

func TestCanonicalDigestIgnoresOrderButDetectsQuestionChange(t *testing.T) {
	left, right := validRequest(), validRequest()
	right.Subject.RunIDs = []string{"run-b", "run-a"}
	right.EvidencePolicy.RequiredPlanes = []string{"events"}
	first, err := left.CanonicalDigest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := right.CanonicalDigest()
	if err != nil || first != second {
		t.Fatalf("reordered request changed digest: %s != %s (err=%v)", first, second, err)
	}
	right.Question = "Is the run waiting?"
	third, err := right.CanonicalDigest()
	if err != nil || first == third {
		t.Fatalf("question change did not change digest: %s == %s (err=%v)", first, third, err)
	}
}

func marshalRequest(req Request) ([]byte, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	return jsonMarshal(req)
}

// Kept as a tiny seam so the tests exercise the same standard encoding path
// without exposing a second public serialization API.
var jsonMarshal = func(value any) ([]byte, error) { return json.Marshal(value) }
var containsString = func(value, needle string) bool { return strings.Contains(value, needle) }
