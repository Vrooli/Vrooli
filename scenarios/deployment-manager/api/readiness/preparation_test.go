package readiness

import (
	"context"
	"errors"
	"strings"
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

type candidateResolverFake struct{ value CandidateProvenance }

func (f candidateResolverFake) ResolveCandidate(context.Context, string) (CandidateProvenance, error) {
	return f.value, nil
}

type rampEvidenceResolverFake struct{ rows []RampEvidence }

func (f rampEvidenceResolverFake) ListRampEvidence(context.Context, string, string) ([]RampEvidence, error) {
	return f.rows, nil
}

type recoveryReadinessResolverFake struct{ value RecoveryReadiness }

func (f recoveryReadinessResolverFake) CheckRecovery(context.Context, ReviewIdentity) (RecoveryReadiness, error) {
	return f.value, nil
}

type observabilityResolverFake struct{ value ObservabilityReadiness }

func (f observabilityResolverFake) CheckObservability(context.Context, ReviewIdentity) (ObservabilityReadiness, error) {
	return f.value, nil
}

type fixedEvidenceProducer struct{ value EvidenceItem }

func (f fixedEvidenceProducer) Collect(_ context.Context, _ ReviewIdentity, criterion Item) (EvidenceItem, error) {
	if f.value.CriterionID != criterion.ID {
		return EvidenceItem{}, errors.New("fixed evidence producer does not own criterion")
	}
	return f.value, nil
}

func prepareIdentity() ReviewIdentity {
	return ReviewIdentity{Scenario: "demo", ProfileID: "p1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Targets: []string{"linux"}, Channel: "stable", PolicyVersion: ChecklistVersion}
}

func passedEvidence(id string) EvidenceItem {
	return EvidenceItem{CriterionID: id, Status: SignalPassed, Applicability: "applicable", Producer: "test-genie", ProducerVersion: "1", CandidateCommit: "abc", ArtifactDigest: "sha256:candidate", Target: "linux", Environment: "stable", PolicyVersion: ChecklistVersion, ObservedAt: time.Now().UTC(), Reference: "run:one"}
}

func TestUnavailableEvidenceUsesStableOwnerReferenceWithoutBinding(t *testing.T) {
	criterion := Item{ID: "customer-contact", Owner: "offer-desk"}
	evidence := unavailableEvidence("rr-test", prepareIdentity(), criterion, "linux", time.Now().UTC(), "owner route is not configured")
	if evidence.Reference != "owner:offer-desk" {
		t.Fatalf("reference = %q, want owner:offer-desk", evidence.Reference)
	}
	if evidence.Status != SignalUnavailable || !strings.Contains(evidence.Detail, "next action:") {
		t.Fatalf("unavailable evidence lost refusal detail: %+v", evidence)
	}
}

func TestPreparePersistsFirstReleaseWithoutFabricatingApproval(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("suite", Required, SafetyBlocker, "Given a candidate, when tests run, then they pass")}}
	decision, err := (&Preparer{Policy: policy, Repository: repo, Producers: map[string]EvidenceProducer{
		"deployment-manager.test.read": fixedEvidenceProducer{value: passedEvidence("suite")},
	}}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.ComparisonMode != ComparisonFirstRelease || decision.Review.Status != ReviewAgentReview || decision.Review.ApprovedAt != nil || len(decision.Verdict.Findings) != 0 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestPrepareReadsOwnerObservationFromRepositoryWithoutProducerRegistration(t *testing.T) {
	repo := &memoryReviewRepository{observations: map[string]EvidenceItem{}}
	criterion := validTestItem("suite", Required, SafetyBlocker, "Given a candidate, when tests run, then they pass")
	binding := criterion.ProducerBinding()
	repo.observations[criterion.ID+"|"+binding] = passedEvidence(criterion.ID)
	policy := Checklist{Version: ChecklistVersion, Items: []Item{criterion}}
	decision, err := (&Preparer{Policy: policy, Repository: repo}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("typed producer observation was not consumed: decision=%+v evidence=%+v", decision, repo.evidence)
	}
}

func TestPrepareDoesNotUseStoredObservationWhenDeclaredOwnerIsUnavailable(t *testing.T) {
	repo := &memoryReviewRepository{observations: map[string]EvidenceItem{}}
	criterion := Item{
		ID: "compatibility", Title: "Compatibility", Category: "compatibility", Owner: "deployment-manager",
		Applicability: "all", CleanRequirement: Required, GlobalImpact: SafetyBlocker,
		Freshness:   FreshnessPolicy{Basis: "candidate_identity"},
		Producer:    &ProducerRoute{Binding: "deployment-manager.compatibility.compare"},
		Acceptance:  GherkinAcceptance{Given: "a candidate", When: "compatibility is compared", Then: "incompatibilities are dispositioned"},
		Remediation: Remediation{Skill: "deployment-manager", Topic: "compatibility"},
	}
	// A stale passing row must not satisfy the criterion once the production
	// server explicitly knows that its owner execution seam is unavailable.
	repo.observations[criterion.ID+"|"+criterion.ProducerBinding()] = passedEvidence(criterion.ID)
	policy := Checklist{Version: ChecklistVersion, Items: []Item{criterion}}
	decision, err := (&Preparer{
		Policy:     policy,
		Repository: repo,
		Goals:      &memoryGoals{},
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): UnavailableProducer{Binding: criterion.ProducerBinding(), Reason: "comparison owner is not configured"},
		},
	}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity(), ProvidedEvidence: []EvidenceItem{passedEvidence(criterion.ID)}})
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.evidence) != 1 || repo.evidence[0].Status != SignalUnavailable || repo.evidence[0].Producer == "test-genie" {
		t.Fatalf("stale owner observation was accepted: decision=%+v evidence=%+v", decision, repo.evidence)
	}
	if !strings.Contains(repo.evidence[0].Detail, "comparison owner is not configured") {
		t.Fatalf("unavailability reason was lost: %+v", repo.evidence[0])
	}
}

func TestPrepareUsesDeploymentManagerPolicyProducer(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := DefaultChecklist()
	var criterion Item
	for _, item := range policy.Items {
		if item.ID == "update-policy-set" {
			criterion = item
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("update-policy-set criterion is missing")
	}
	policy.Items = []Item{criterion}
	observedAt := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	decision, err := (&Preparer{
		Policy: policy, Repository: repo,
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): PolicyProducer{Policy: policy, Now: func() time.Time { return observedAt }},
		},
		Now: func() time.Time { return observedAt },
	}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range repo.evidence {
		if item.CriterionID != criterion.ID {
			continue
		}
		if item.Status != SignalPassed || item.Producer != "deployment-manager" || !strings.HasPrefix(item.Reference, "readiness-policy:sha256:") {
			t.Fatalf("policy evidence is not attributable: %+v", item)
		}
		if !item.ObservedAt.Equal(observedAt) {
			t.Fatalf("observation time = %s, want %s", item.ObservedAt, observedAt)
		}
		return
	}
	t.Fatalf("policy producer evidence missing from evaluation: %+v", decision)
}

func TestPrepareUsesImmutableCandidateForArtifactProvenance(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := DefaultChecklist()
	var criterion Item
	for _, item := range policy.Items {
		if item.ID == "artifact-provenance-complete" {
			criterion = item
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("artifact-provenance-complete criterion is missing")
	}
	policy.Items = []Item{criterion}
	identity := prepareIdentity()
	identity.CandidateID = "candidate-1"
	identity.DestinationRevisionID = "destination-1"
	identity.AuthorizationEpoch = 1
	observedAt := time.Date(2026, 9, 9, 12, 1, 0, 0, time.UTC)
	decision, err := (&Preparer{
		Policy: policy, Repository: repo,
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): CandidateProvenanceProducer{
				Resolver: candidateResolverFake{value: CandidateProvenance{
					ID: identity.CandidateID, SourceRevision: identity.CandidateCommit,
					ArtifactManifestDigest: identity.ArtifactDigest, DependencyLockDigest: "sha256:lock",
					PolicyDigest: "sha256:policy", BuildInputsPresent: true, ArtifactCount: 1, SignedArtifactCount: 1,
				}},
				Now: func() time.Time { return observedAt },
			},
		},
		Now: func() time.Time { return observedAt },
	}).Prepare(context.Background(), PrepareRequest{Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("candidate provenance was not accepted: decision=%+v evidence=%+v", decision, repo.evidence)
	}
	if repo.evidence[0].Reference != "candidate:candidate-1:sha256:candidate" || !repo.evidence[0].ObservedAt.Equal(observedAt) {
		t.Fatalf("candidate provenance attribution = %+v", repo.evidence[0])
	}
}

func TestPrepareUsesExactTargetRampEvidence(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := DefaultChecklist()
	var criterion Item
	for _, item := range policy.Items {
		if item.ID == "ramp-evidence-complete" {
			criterion = item
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("ramp-evidence-complete criterion is missing")
	}
	policy.Items = []Item{criterion}
	identity := prepareIdentity()
	identity.Targets = []string{"linux", "windows"}
	observedAt := time.Date(2026, 9, 9, 12, 2, 0, 0, time.UTC)
	decision, err := (&Preparer{
		Policy: policy, Repository: repo,
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): RampEvidenceProducer{
				Resolver: rampEvidenceResolverFake{rows: []RampEvidence{
					{Target: "linux", Disposition: "passed", RunID: "run-linux"},
					{Target: "windows", Disposition: "passed", RunID: "run-windows"},
				}},
				Now: func() time.Time { return observedAt },
			},
		},
		Now: func() time.Time { return observedAt },
	}).Prepare(context.Background(), PrepareRequest{Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("ramp evidence was not accepted: decision=%+v evidence=%+v", decision, repo.evidence)
	}
	if repo.evidence[0].Reference != "evidence:p1:abc:run-linux,run-windows" || !repo.evidence[0].ObservedAt.Equal(observedAt) {
		t.Fatalf("ramp evidence attribution = %+v", repo.evidence[0])
	}
}

func TestRampEvidenceDoesNotReuseOnePlatformRowAcrossTargets(t *testing.T) {
	var criterion Item
	for _, item := range DefaultChecklist().Items {
		if item.ID == "ramp-evidence-complete" {
			criterion = item
			break
		}
	}
	identity := prepareIdentity()
	identity.Targets = []string{"linux-x64", "linux-arm64"}
	evidence, err := (RampEvidenceProducer{Resolver: rampEvidenceResolverFake{rows: []RampEvidence{{Platform: "linux", Disposition: "passed", RunID: "run-linux"}}}}).Collect(context.Background(), identity, criterion)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != SignalFailed || !strings.Contains(evidence.Detail, "missing=linux-arm64") {
		t.Fatalf("one broad platform row was reused: %+v", evidence)
	}
}

func TestPrepareUsesCandidateAssetsForDeclaredTargets(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := DefaultChecklist()
	var criterion Item
	for _, item := range policy.Items {
		if item.ID == "platform-assets-set" {
			criterion = item
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("platform-assets-set criterion is missing")
	}
	policy.Items = []Item{criterion}
	identity := prepareIdentity()
	identity.CandidateID = "candidate-1"
	identity.DestinationRevisionID = "destination-1"
	identity.AuthorizationEpoch = 1
	observedAt := time.Date(2026, 9, 9, 12, 3, 0, 0, time.UTC)
	decision, err := (&Preparer{
		Policy: policy, Repository: repo,
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): PlatformAssetsProducer{
				Resolver: candidateResolverFake{value: CandidateProvenance{
					ID: identity.CandidateID, ArtifactManifestDigest: identity.ArtifactDigest,
					ArtifactTargetIDs: []string{"linux"}, ArtifactPlatforms: []string{"linux"}, ArtifactRefsComplete: true,
				}},
				Now: func() time.Time { return observedAt },
			},
		},
		Now: func() time.Time { return observedAt },
	}).Prepare(context.Background(), PrepareRequest{Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("candidate assets were not accepted: decision=%+v evidence=%+v", decision, repo.evidence)
	}
	if repo.evidence[0].Reference != "candidate-assets:candidate-1:sha256:candidate" || !repo.evidence[0].ObservedAt.Equal(observedAt) {
		t.Fatalf("candidate assets attribution = %+v", repo.evidence[0])
	}
}

func TestPrepareUsesRecoveryPreflightCapability(t *testing.T) {
	repo := &memoryReviewRepository{}
	policy := DefaultChecklist()
	var criterion Item
	for _, item := range policy.Items {
		if item.ID == "recovery-proven" {
			criterion = item
			break
		}
	}
	if criterion.ID == "" {
		t.Fatal("recovery-proven criterion is missing")
	}
	policy.Items = []Item{criterion}
	identity := prepareIdentity()
	identity.CandidateID = "candidate-1"
	identity.DestinationRevisionID = "destination-1"
	identity.AuthorizationEpoch = 1
	observedAt := time.Date(2026, 9, 9, 12, 4, 0, 0, time.UTC)
	decision, err := (&Preparer{
		Policy: policy, Repository: repo,
		Producers: map[string]EvidenceProducer{
			criterion.ProducerBinding(): RecoveryProducer{
				Resolver: recoveryReadinessResolverFake{value: RecoveryReadiness{ExecutorConfigured: true, AuthorizationConfigured: true, TargetIdentityBound: true}},
				Now:      func() time.Time { return observedAt },
			},
		},
		Now: func() time.Time { return observedAt },
	}).Prepare(context.Background(), PrepareRequest{Identity: identity})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewAgentReview || len(repo.evidence) != 1 || repo.evidence[0].Status != SignalPassed {
		t.Fatalf("recovery preflight was not accepted: decision=%+v evidence=%+v", decision, repo.evidence)
	}
	if repo.evidence[0].Reference != "recovery-preflight:p1:abc:destination-1" || !repo.evidence[0].ObservedAt.Equal(observedAt) {
		t.Fatalf("recovery preflight attribution = %+v", repo.evidence[0])
	}
}

func TestObservabilityProducerBindsOwnerObservation(t *testing.T) {
	criterion := Item{ID: "observability-reachable", Producer: &ProducerRoute{Binding: "deployment-manager.observability.readiness"}}
	now := time.Date(2026, 9, 9, 12, 5, 0, 0, time.UTC)
	evidence, err := (ObservabilityProducer{Resolver: observabilityResolverFake{value: ObservabilityReadiness{
		Healthy: true, Target: "cloud:deployment-1", Reference: "scenario-to-cloud:health:deployment-1:2026-09-09T12:05:00Z", ObservedAt: now, Detail: "healthy",
	}}}).Collect(context.Background(), prepareIdentity(), criterion)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != SignalPassed || evidence.Target != "cloud:deployment-1" || evidence.Reference == "" || !evidence.ObservedAt.Equal(now) {
		t.Fatalf("observability evidence = %+v", evidence)
	}
}

func TestOperationsOwnershipProducerBindsCandidateDeclaration(t *testing.T) {
	criterion := Item{ID: "operations-ownership-set", Producer: &ProducerRoute{Binding: "deployment-manager.operations.readiness"}}
	identity := prepareIdentity()
	identity.CandidateID = "candidate-ops"
	now := time.Date(2026, 9, 9, 12, 6, 0, 0, time.UTC)
	evidence, err := (OperationsOwnershipProducer{
		Resolver: candidateResolverFake{value: CandidateProvenance{
			ID: identity.CandidateID,
			CapabilityDeclaration: OperationsOwnership{
				SupportOwner: "team:support", IncidentOwner: "team:incident", CustomerContact: "mailto:support@example.test",
				ReleaseAuthority: "team:release", RollbackAuthority: "team:rollback", DegradedModeAuthority: "runbook:degraded-mode",
			},
		}},
		Now: func() time.Time { return now },
	}).Collect(context.Background(), identity, criterion)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != SignalPassed || evidence.Reference != "candidate-operations:candidate-ops" || !evidence.ObservedAt.Equal(now) {
		t.Fatalf("operations ownership evidence = %+v", evidence)
	}
}

func TestOperationsOwnershipProducerFailsWhenCandidateDeclarationIsIncomplete(t *testing.T) {
	criterion := Item{ID: "operations-ownership-set", Producer: &ProducerRoute{Binding: "deployment-manager.operations.readiness"}}
	identity := prepareIdentity()
	identity.CandidateID = "candidate-ops"
	evidence, err := (OperationsOwnershipProducer{Resolver: candidateResolverFake{value: CandidateProvenance{ID: identity.CandidateID}}}).Collect(context.Background(), identity, criterion)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.Status != SignalFailed || !strings.Contains(evidence.Detail, "missing") {
		t.Fatalf("incomplete operations ownership was not refused: %+v", evidence)
	}
}

func TestPrepareFailsClosedAndCreatesOnlyUnresolvedMilestone(t *testing.T) {
	repo := &memoryReviewRepository{}
	goals := &memoryGoals{}
	policy := Checklist{Version: ChecklistVersion, Items: []Item{validTestItem("passed", Required, SafetyBlocker, "passed"), validTestItem("missing", Required, SafetyBlocker, "missing fixed")}}
	decision, err := (&Preparer{Policy: policy, Repository: repo, Goals: goals, Producers: map[string]EvidenceProducer{
		"deployment-manager.test.read": fixedEvidenceProducer{value: passedEvidence("passed")},
	}}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Review.Status != ReviewBlocked || decision.Verdict.Approved || len(goals.specs) != 1 || len(goals.specs[0].Milestones) != 1 || goals.specs[0].Milestones[0].Name != "missing" {
		t.Fatalf("unexpected blocked decision: %+v specs=%+v", decision, goals.specs)
	}
	if repo.evidence[1].Status != SignalUnavailable || repo.evidence[1].Reference == "" || !strings.Contains(repo.evidence[1].Detail, "next action: run ") || !strings.Contains(repo.evidence[1].Detail, "owner \"deployment-manager\"") {
		t.Fatalf("missing producer was not attributable: %+v", repo.evidence[1])
	}
	if len(decision.Next) != 1 || !strings.Contains(decision.Next[0], "missing:") || !strings.Contains(decision.Next[0], "next action: run ") {
		t.Fatalf("next action was not operator actionable: %+v", decision.Next)
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
			decision, err := (&Preparer{Policy: policy, Repository: repo, Predecessor: fixedPredecessor{value: tc.predecessor, err: tc.err}, Producers: map[string]EvidenceProducer{
				"deployment-manager.test.read": fixedEvidenceProducer{value: passedEvidence("suite")},
			}}).Prepare(context.Background(), PrepareRequest{Identity: prepareIdentity()})
			if err != nil {
				t.Fatal(err)
			}
			if decision.Review.ComparisonMode != tc.want {
				t.Fatalf("mode=%s want=%s", decision.Review.ComparisonMode, tc.want)
			}
		})
	}
}
