package readiness

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryReviewRepository struct {
	review       Review
	evidence     []EvidenceItem
	findings     []ReviewFinding
	observations map[string]EvidenceItem
}

func (m *memoryReviewRepository) CreateOrGet(_ context.Context, review *Review) (bool, error) {
	key, err := review.Identity.Key()
	if err != nil {
		return false, err
	}
	review.Key = key
	review.CreatedAt = time.Now().UTC()
	review.UpdatedAt = review.CreatedAt
	if m.review.Key == key {
		*review = m.review
		return true, nil
	}
	m.review = *review
	return false, nil
}

func (m *memoryReviewRepository) Get(_ context.Context, key string) (*Review, error) {
	if m.review.Key != key {
		return nil, errors.New("missing")
	}
	value := m.review
	return &value, nil
}

func (m *memoryReviewRepository) ListReviews(context.Context, ReviewStatus, int) ([]Review, error) {
	return []Review{m.review}, nil
}

func (m *memoryReviewRepository) ListEvaluation(_ context.Context, key string) ([]EvidenceItem, []ReviewFinding, error) {
	if m.review.Key != key {
		return nil, nil, errors.New("missing")
	}
	return append([]EvidenceItem(nil), m.evidence...), append([]ReviewFinding(nil), m.findings...), nil
}

func (m *memoryReviewRepository) ListActiveWaivers(context.Context, string, time.Time) ([]ReviewWaiver, error) {
	return nil, nil
}

func (m *memoryReviewRepository) ListWaivers(context.Context, string, int) ([]ReviewWaiver, error) {
	return nil, nil
}

func (m *memoryReviewRepository) ReplaceEvaluation(_ context.Context, key string, evidence []EvidenceItem, findings []ReviewFinding, status ReviewStatus) error {
	m.review.Status = status
	m.review.GoalClosedAt = nil
	m.review.ApprovedAt = nil
	m.evidence = evidence
	m.findings = findings
	return nil
}

func (m *memoryReviewRepository) SetGoal(_ context.Context, _ string, goal string) error {
	m.review.GoalRef = goal
	return nil
}

func (m *memoryReviewRepository) RecordGoalClosure(context.Context, string, time.Time) error {
	return nil
}

func (m *memoryReviewRepository) Approve(context.Context, string, ReviewIdentity, string, time.Time) error {
	return nil
}
func (m *memoryReviewRepository) SaveWaiver(context.Context, ReviewWaiver) error { return nil }

func (m *memoryReviewRepository) SaveObservation(_ context.Context, observation EvidenceObservation) error {
	if m.observations == nil {
		m.observations = map[string]EvidenceItem{}
	}
	m.observations[observation.CriterionID+"|"+observation.ProducerBinding] = observation.Evidence
	return nil
}

func (m *memoryReviewRepository) FindObservation(_ context.Context, _ ReviewIdentity, criterion, binding string) (*EvidenceItem, error) {
	item, ok := m.observations[criterion+"|"+binding]
	if !ok {
		return nil, errors.New("missing observation")
	}
	return &item, nil
}
func (m *memoryReviewRepository) MarkPromoted(context.Context, string, time.Time) error { return nil }
func (m *memoryReviewRepository) SaveHumanCheck(context.Context, HumanCheck) error      { return nil }
func (m *memoryReviewRepository) ListHumanChecks(context.Context, string) ([]HumanCheck, error) {
	return nil, nil
}

type memoryGoals struct{ specs []GoalSpec }

func (m *memoryGoals) Open(_ context.Context, spec GoalSpec) (string, bool, error) {
	m.specs = append(m.specs, spec)
	return canonicalSwarmGoalName(spec.Name), len(m.specs) > 1, nil
}

type fixedPredecessor struct {
	value *Predecessor
	err   error
}

func (f fixedPredecessor) LatestDeployed(context.Context, ReviewIdentity) (*Predecessor, error) {
	return f.value, f.err
}

func prepareIdentity() ReviewIdentity {
	return ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: ChecklistVersion}
}

func passedEvidence(id string) EvidenceItem {
	return EvidenceItem{CriterionID: id, Status: SignalPassed, Applicability: "applicable", Producer: "test-genie", ProducerVersion: "1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Target: "linux", Environment: "stable", PolicyVersion: ChecklistVersion, ObservedAt: time.Now().UTC(), Reference: "run:one"}
}

func TestPreparePersistsFirstReleaseWithoutFabricatingApproval(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("suite", Required, SafetyBlocker, "Given a candidate, when tests run, then they pass")}}
	decision, err := (&Preparer{Policy: policy, Repository: repo}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity(), ProvidedEvidence: []EvidenceItem{passedEvidence("suite")}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.ComparisonMode != ComparisonFirstRelease || decision.Review.Status != ReviewAgentReview || decision.Review.ApprovedAt != nil || len(decision.Verdict.Findings) != 0 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestPrepareReadsTypedObservationFromDeclaredProducer(t *testing.T) {
	repo := &memoryReviewRepository{observations: map[string]EvidenceItem{}}
	criterion := validTestItem("suite", Required, SafetyBlocker, "Given a candidate, when tests run, then they pass")
	binding := criterion.ProducerBinding()
	repo.observations[criterion.ID+"|"+binding] = passedEvidence(criterion.ID)
	policy := Checklist{Version: ChecklistVersion, Items: []Item{criterion}}
	decision, err := (&Preparer{Policy: policy, Repository: repo, Producers: ObservationProducers(policy, repo)}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("typed producer observation was not consumed: decision=%+v evidence=%+v", decision, repo.evidence)
	}
}

func TestPrepareFailsClosedAndCreatesOnlyUnresolvedMilestone(t *testing.T) {
	repo := &memoryReviewRepository{}
	goals := &memoryGoals{}
	policy := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("passed", Required, SafetyBlocker, "passed"), validTestItem("missing", Required, SafetyBlocker, "missing fixed")}}
	decision, err := (&Preparer{Policy: policy, Repository: repo, Goals: goals}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity(), ProvidedEvidence: []EvidenceItem{passedEvidence("passed")}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewBlocked || decision.Verdict.Approved || len(goals.specs) != 1 || len(goals.specs[0].Milestones) != 1 || goals.specs[0].Milestones[0].Name != "missing" {
		t.Fatalf("unexpected blocked decision: %+v specs=%+v", decision, goals.specs)
	}
	if repo.evidence[1].Status != SignalUnavailable || repo.evidence[1].Reference == "" {
		t.Fatalf("missing producer was not attributable: %+v", repo.evidence[1])
	}
}

func TestPrepareClassifiesPredecessorEvidenceHonestly(t *testing.T) {
	policy := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("suite", Required, SafetyBlocker, "pass")}}
	for _, tc := range []struct {
		name        string
		predecessor *Predecessor
		err         error
		want        ComparisonMode
	}{
		{"first release", nil, nil, ComparisonFirstRelease},
		{"comparable", &Predecessor{ReleaseID: "r1", Commit: "old", ArtifactDigest: "sha256:old", PolicyVersion: 1}, nil, ComparisonComparable},
		{"missing history", &Predecessor{ReleaseID: "r1", Commit: "old"}, nil, ComparisonUnavailable},
		{"owner unavailable", nil, errors.New("offline"), ComparisonUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &memoryReviewRepository{}
			decision, err := (&Preparer{Policy: policy, Repository: repo, Predecessor: fixedPredecessor{value: tc.predecessor, err: tc.err}}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity(), ProvidedEvidence: []EvidenceItem{passedEvidence("suite")}})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Review.ComparisonMode != tc.want {
				t.Fatalf("mode=%s want=%s", decision.Review.ComparisonMode, tc.want)
			}
		})
	}
}
