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
