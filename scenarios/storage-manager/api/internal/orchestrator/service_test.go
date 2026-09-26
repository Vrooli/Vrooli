package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	"storage-manager/internal/policy"
	"storage-manager/internal/providers"
	cleanupfakes "storage-manager/internal/testutil/cleanup"
)

func TestPlanIDStableForSamePolicyProviderVersionsAndPreview(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t, cleanup.ProviderModeEnabled, cleanup.ApprovalModeNone)
	ctx := context.Background()

	first, err := svc.Plan(ctx, cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("Plan() first error = %v", err)
	}
	second, err := svc.Plan(ctx, cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("Plan() second error = %v", err)
	}
	if first.ID == "" || first.ID != second.ID {
		t.Fatalf("plan IDs not stable: %q vs %q", first.ID, second.ID)
	}
	if first.PolicyVersion != second.PolicyVersion {
		t.Fatalf("policy versions differ: %q vs %q", first.PolicyVersion, second.PolicyVersion)
	}
}

func TestSetStandingApprovalRejectsApprovalFromAnotherHost(t *testing.T) {
	provider := &tierProvider{id: "conditional", tier: cleanup.SafetyTierConditional, approval: cleanup.ApprovalModeOperator}
	registry, err := providers.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	svc := NewService(registry, NewMemoryStore(), cleanupfakes.Clock{Time: time.Now()})
	svc.hostID = func() string { return "current-host" }
	if err := svc.store.SavePolicy(context.Background(), Policy{
		Version:   "policy-test",
		Profile:   policy.ProfileConservative,
		Providers: map[string]cleanup.ProviderPolicy{"conditional": {Enabled: true, ApprovalMode: cleanup.ApprovalModeOperator}},
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	_, err = svc.SetStandingApproval(context.Background(), "conditional", StandingApproval{
		ApprovedAt: time.Now(), ApprovedBy: "operator", HostID: "other-host",
	})
	if err == nil || !strings.Contains(err.Error(), "does not match current host") {
		t.Fatalf("SetStandingApproval error = %v, want host mismatch", err)
	}
}

func TestCensusContinuesAfterCallerStopsWaiting(t *testing.T) {
	t.Parallel()

	provider := &blockingProvider{
		started: make(chan struct{}),
		release: make(chan struct{}),
	}
	registry, err := providers.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	observedAt := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	svc := NewService(registry, NewMemoryStore(), cleanupfakes.Clock{Time: observedAt})
	if err := svc.store.SavePolicy(context.Background(), Policy{
		Version:   "policy-test",
		Profile:   policy.ProfileConservative,
		Providers: map[string]cleanup.ProviderPolicy{"blocking": {Enabled: true, ApprovalMode: cleanup.ApprovalModeNone}},
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	id, err := svc.StartCensus(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("StartCensus: %v", err)
	}
	<-provider.started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.WaitCensus(ctx, id); err == nil {
		t.Fatal("WaitCensus with canceled caller context succeeded")
	}
	close(provider.release)
	plan, err := svc.WaitCensus(context.Background(), id)
	if err != nil {
		t.Fatalf("WaitCensus after release: %v", err)
	}
	if plan.CensusID != id || plan.CensusStatus != CensusStatusComplete {
		t.Fatalf("census plan = %#v, want completed tracked census %q", plan, id)
	}
	reused, err := svc.Plan(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("follow-up Plan() error = %v", err)
	}
	if reused.CensusID != id {
		t.Fatalf("follow-up Plan() started a new census %q, want completed census %q", reused.CensusID, id)
	}
}

type blockingProvider struct {
	started chan struct{}
	release chan struct{}
}

func (*blockingProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID: "blocking", Name: "Blocking", Version: "v1", OwnerScenario: "storage-manager",
		SafetyTier: cleanup.SafetyTierSafe, DefaultMode: cleanup.ProviderModeEnabled,
		DefaultApproval: cleanup.ApprovalModeNone, SupportedPlatforms: []string{"linux"},
		IrreversibleEffects: []string{"none"}, TestSubstitute: "blocking-provider",
	}
}

func (p *blockingProvider) Estimate(context.Context, cleanup.EstimateRequest) (cleanup.Estimate, error) {
	select {
	case <-p.started:
	default:
		close(p.started)
	}
	<-p.release
	return cleanup.Estimate{ProviderID: "blocking", ProviderVersion: "v1"}, nil
}

func (*blockingProvider) Preview(context.Context, cleanup.PreviewRequest) (cleanup.Preview, error) {
	return cleanup.Preview{ProviderID: "blocking", ProviderVersion: "v1"}, nil
}

func (*blockingProvider) Apply(context.Context, cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	return cleanup.ApplyResult{ProviderID: "blocking"}, nil
}

func (*blockingProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true}, nil
}

// [REQ:CLN-P0-004]
func TestApplyRequiresPolicyVersionApprovalAndIdempotency(t *testing.T) {
	t.Parallel()

	svc, fsys := newTestService(t, cleanup.ProviderModeEnabled, cleanup.ApprovalModeOperator)
	ctx := context.Background()
	plan, err := svc.Plan(ctx, cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err := svc.Apply(ctx, ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  "stale-policy",
		ApprovalMode:   cleanup.ApprovalModeOperator,
		ApprovalToken:  "operator-ok",
		IdempotencyKey: "apply-1",
	}); err == nil || !strings.Contains(err.Error(), "policy version mismatch") {
		t.Fatalf("Apply() stale policy error = %v, want mismatch", err)
	}
	if _, err := svc.Apply(ctx, ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  plan.PolicyVersion,
		ApprovalMode:   cleanup.ApprovalModeNone,
		IdempotencyKey: "apply-1",
	}); err == nil || !strings.Contains(err.Error(), "requires operator approval") {
		t.Fatalf("Apply() missing approval error = %v, want approval gate", err)
	}
	applied, err := svc.Apply(ctx, ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  plan.PolicyVersion,
		ApprovalMode:   cleanup.ApprovalModeOperator,
		ApprovalToken:  "operator-ok",
		IdempotencyKey: "apply-1",
	})
	if err != nil {
		t.Fatalf("Apply() approved error = %v", err)
	}
	if applied.AlreadyApplied || applied.ReclaimedBytes != 64 {
		t.Fatalf("Apply() approved = %#v, want first reclaim", applied)
	}
	if len(fsys.Removed) != 1 {
		t.Fatalf("fake removals = %d, want 1", len(fsys.Removed))
	}
	replayed, err := svc.Apply(ctx, ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  plan.PolicyVersion,
		ApprovalMode:   cleanup.ApprovalModeOperator,
		ApprovalToken:  "operator-ok",
		IdempotencyKey: "apply-1",
	})
	if err != nil {
		t.Fatalf("Apply() replay error = %v", err)
	}
	if !replayed.AlreadyApplied || replayed.ReclaimedBytes != 64 {
		t.Fatalf("Apply() replay = %#v, want already-applied same result", replayed)
	}
	if len(fsys.Removed) != 1 {
		t.Fatalf("fake removals after replay = %d, want still 1", len(fsys.Removed))
	}
}

func TestOperatorApprovalCanApplyOwnerProvider(t *testing.T) {
	t.Parallel()

	svc, _ := newTestService(t, cleanup.ProviderModeEnabled, cleanup.ApprovalModeOwner)
	plan, err := svc.Plan(context.Background(), cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	result, err := svc.Apply(context.Background(), ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  plan.PolicyVersion,
		ApprovalMode:   cleanup.ApprovalModeOperator,
		ApprovalToken:  "operator-ok",
		IdempotencyKey: "owner-provider-operator-approval",
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if result.ReclaimedBytes != 64 {
		t.Fatalf("reclaimed bytes = %d, want 64", result.ReclaimedBytes)
	}
}

// TestAuditRedactsPathsFromFailureMessages asserts that when individual items
// cannot be removed, the failure is recorded in the audit log with the path
// stripped out.
//
// The run itself must SUCCEED. A single unremovable entry — another user's file
// in a shared /tmp, or one that vanished mid-run — is routine, and aborting
// would abandon every remaining entry at exactly the moment the host needs
// space. The failure is therefore surfaced as an apply.partial audit event
// rather than as an error, and it still must not leak the path.
func TestAuditRedactsPathsFromFailureMessages(t *testing.T) {
	t.Parallel()

	svc, fsys := newTestService(t, cleanup.ProviderModeEnabled, cleanup.ApprovalModeOperator)
	fsys.AllowRemove = false
	ctx := context.Background()
	plan, err := svc.Plan(ctx, cleanup.ObservationScope{})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if _, err = svc.Apply(ctx, ApplyInput{
		PlanID:         plan.ID,
		PolicyVersion:  plan.PolicyVersion,
		ApprovalMode:   cleanup.ApprovalModeOperator,
		ApprovalToken:  "operator-ok",
		IdempotencyKey: "apply-2",
	}); err != nil {
		t.Fatalf("Apply() should tolerate unremovable items, got error = %v", err)
	}
	events, err := svc.Audit(ctx)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}
	var partial AuditEvent
	for _, event := range events {
		if event.Type == "apply.partial" {
			partial = event
			break
		}
	}
	if partial.Type == "" {
		t.Fatalf("audit events missing apply.partial: %#v", events)
	}
	if strings.Contains(partial.Message, "/fake/tmp") || !partial.Redacted {
		t.Fatalf("partial-failure audit not redacted: %#v", partial)
	}
}

func newTestService(t *testing.T, mode cleanup.ProviderMode, approval cleanup.ApprovalMode) (*Service, *cleanupfakes.FileSystem) {
	t.Helper()
	now := time.Date(2026, 7, 7, 12, 0, 0, 0, time.UTC)
	fsys := &cleanupfakes.FileSystem{
		Root:        "/fake",
		AllowRemove: true,
		Files: map[string]cleanup.FileInfo{
			"/fake/tmp/old.log": {Path: "/fake/tmp/old.log", Size: 64, ModTime: now.Add(-48 * time.Hour)},
		},
	}
	provider := providers.NewTmpProvider(fsys, cleanupfakes.Clock{Time: now}, providers.FileProviderConfig{
		ID:    "tmp",
		Name:  "Temporary files",
		Roots: []string{"/fake/tmp"},
	})
	registry, err := providers.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	svc := NewService(registry, NewMemoryStore(), cleanupfakes.Clock{Time: now})
	pol, err := policy.BuildProfile(policy.ProfileConservative, registry.List())
	if err != nil {
		t.Fatalf("BuildProfile() error = %v", err)
	}
	pol.Defaults["tmp"] = cleanup.ProviderPolicy{Enabled: mode == cleanup.ProviderModeEnabled, MinAge: 24 * time.Hour, ApprovalMode: approval}
	if err := svc.store.SavePolicy(context.Background(), Policy{
		Version:   stablePolicyVersion(pol.Name, pol.Defaults),
		Profile:   pol.Name,
		Providers: pol.Defaults,
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("SavePolicy() error = %v", err)
	}
	return svc, fsys
}
