package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
)

func TestTemporaryConfigurationGrantsOnlyTmpAndRejectsFreshCleanup(t *testing.T) {
	tmp := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeOperator}
	other := &tierProvider{id: "owner-data", tier: cleanup.SafetyTierConditional, approval: cleanup.ApprovalModeOperator}
	svc := newPressureService(t, tmp, other)
	before, err := svc.CurrentPolicy(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ConfigureTemporaryRetention(context.Background(), true, time.Hour)
	if err == nil {
		t.Fatal("accepted automatic cleanup of fresh entries")
	}
	after, err := svc.ConfigureTemporaryRetention(context.Background(), true, 72*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if after.Providers["tmp"].ApprovalMode != cleanup.ApprovalModeNone || after.Providers["tmp"].MinAge != 72*time.Hour || !reflect.DeepEqual(before.Providers["owner-data"], after.Providers["owner-data"]) {
		t.Fatalf("configuration granted unrelated authority: %#v", after)
	}
	disabled, err := svc.ConfigureTemporaryRetention(context.Background(), false, 0)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Providers["tmp"].ApprovalMode != cleanup.ApprovalModeOperator || disabled.Providers["tmp"].MinAge != 72*time.Hour {
		t.Fatalf("revocation lost age or retained automatic authority: %#v", disabled)
	}
	run, err := svc.StartTemporaryRetention(context.Background())
	if err != nil || run.ID != "" {
		t.Fatalf("revoked authority started cleanup: %#v, %v", run, err)
	}
}

func TestScheduledTemporaryRetentionSelectsOnlyTmpAndHonorsPolicy(t *testing.T) {
	for _, blocked := range []string{"", "disabled", "operator approval", "kill switch", "no age"} {
		t.Run(blocked, func(t *testing.T) {
			tmp := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone, oneShot: true}
			other := &tierProvider{id: "other-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
			svc := newPressureService(t, tmp, other)
			pol, err := svc.CurrentPolicy(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			p := pol.Providers["tmp"]
			p.MinAge = 7 * 24 * time.Hour
			if blocked == "disabled" {
				p.Enabled = false
			}
			if blocked == "operator approval" {
				p.ApprovalMode = cleanup.ApprovalModeOperator
			}
			if blocked == "no age" {
				p.MinAge = 0
			}
			pol.Providers["tmp"] = p
			if err := svc.store.SavePolicy(context.Background(), pol); err != nil {
				t.Fatal(err)
			}
			if blocked == "kill switch" {
				svc.SetAutonomousApplyEnabled(false)
			}
			run, err := svc.StartTemporaryRetention(context.Background())
			if blocked == "no age" {
				if err == nil || run.ID != "" {
					t.Fatalf("age-free scheduling allowed: %#v, %v", run, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if blocked != "" {
				if run.ID != "" || tmp.didApply() {
					t.Fatalf("disabled maintenance started: %#v", run)
				}
				return
			}
			terminal, err := svc.WaitRecovery(context.Background(), run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if terminal.Status != RecoveryComplete || !tmp.didApply() || other.didApply() {
				t.Fatalf("terminal=%#v tmp=%t other=%t", terminal, tmp.didApply(), other.didApply())
			}
		})
	}
}

type pendingMeasurementProvider struct{ tierProvider }

func (p *pendingMeasurementProvider) Estimate(context.Context, cleanup.EstimateRequest) (cleanup.Estimate, error) {
	return cleanup.Estimate{ProviderID: p.id, BlockedReason: "temporary measurement pending; traversal position retained"}, nil
}

func TestRecoveryReportsIncompleteMeasurementWithoutClaimingApply(t *testing.T) {
	tmp := &pendingMeasurementProvider{tierProvider: tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}}
	svc := newPressureService(t, tmp)
	outcome, err := svc.executeRecovery(context.Background(), "run-1", "PRESSURE_TRIGGER_BAND", "/", 0, 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Action != ActionObserved || outcome.Reason != "measurement_pending" || len(outcome.ProvidersApplied) != 0 || tmp.didApply() {
		t.Fatalf("pending outcome=%#v", outcome)
	}
}

func TestRateRecoveryChildRequiresDirectGovernedGoWorkChild(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VROOLI_HOME", home)
	base := filepath.Join(home, "tmp", "go-work")

	tests := []struct {
		name      string
		trigger   string
		partition string
		want      bool
	}{
		{"direct child", "PRESSURE_TRIGGER_RATE", filepath.Join(base, "chaos"), true},
		{"parent", "PRESSURE_TRIGGER_RATE", base, false},
		{"nested child", "PRESSURE_TRIGGER_RATE", filepath.Join(base, "chaos", "nested"), false},
		{"band trigger", "PRESSURE_TRIGGER_BAND", filepath.Join(base, "chaos"), false},
		{"outside", "PRESSURE_TRIGGER_RATE", filepath.Join(home, "tmp", "other"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := rateRecoveryChild(test.trigger, test.partition); got != test.want {
				t.Fatalf("rateRecoveryChild(%q, %q) = %t, want %t", test.trigger, test.partition, got, test.want)
			}
		})
	}
}

func TestRecoveryFilesRemovedUsesAppliedIDsOrBoundedBatch(t *testing.T) {
	batch := cleanup.Preview{Items: []cleanup.PreviewItem{{ID: "a"}, {ID: "b"}, {ID: "c"}}}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{AppliedItems: []string{"a"}, ReclaimedBytes: 1}, batch); got != 1 {
		t.Fatalf("explicit applied count = %d, want 1", got)
	}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{ReclaimedBytes: 1, SkippedItems: []string{"c"}}, batch); got != 2 {
		t.Fatalf("bounded inferred count = %d, want 2", got)
	}
	if got := recoveryFilesRemoved(cleanup.ApplyResult{ReclaimedBytes: 0}, batch); got != 0 {
		t.Fatalf("zero-progress count = %d, want 0", got)
	}
}

func TestRecoveryTargetForRunServicesRateRootEvenWhenDeviceTargetMet(t *testing.T) {
	if got := recoveryTargetForRun("PRESSURE_TRIGGER_RATE", "/tmp/hot", 73, 500, 0); got != 501 {
		t.Fatalf("rate root target = %d, want one byte above observed free space", got)
	}
	if got := recoveryTargetForRun("PRESSURE_TRIGGER_BAND", "/", 73, 500, 0); got <= 500 {
		t.Fatalf("ordinary pressure target = %d, want device target above current free space", got)
	}
}

// scriptedRecoveryProvider previews whatever it has not removed yet, in a
// fixed order, and fails every Apply of an item named in failing. pending
// estimates report a paused measurement before any item is ready.
type scriptedRecoveryProvider struct {
	tierProvider
	items     []cleanup.PreviewItem
	failing   map[string]bool
	pending   int
	removed   map[string]bool
	attempted []string
}

func (p *scriptedRecoveryProvider) remaining() []cleanup.PreviewItem {
	var out []cleanup.PreviewItem
	for _, item := range p.items {
		if !p.removed[item.ID] {
			out = append(out, item)
		}
	}
	return out
}

func (p *scriptedRecoveryProvider) Estimate(context.Context, cleanup.EstimateRequest) (cleanup.Estimate, error) {
	if p.pending > 0 {
		p.pending--
		return cleanup.Estimate{ProviderID: p.id, BlockedReason: "temporary measurement pending; traversal position retained"}, nil
	}
	var total int64
	for _, item := range p.remaining() {
		total += item.Bytes
	}
	return cleanup.Estimate{ProviderID: p.id, ProviderVersion: "v1", EstimatedBytes: total, ItemCount: len(p.remaining())}, nil
}

func (p *scriptedRecoveryProvider) Preview(context.Context, cleanup.PreviewRequest) (cleanup.Preview, error) {
	return cleanup.Preview{ProviderID: p.id, ProviderVersion: "v1", Items: p.remaining()}, nil
}

func (p *scriptedRecoveryProvider) Apply(_ context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	out := cleanup.ApplyResult{ProviderID: p.id}
	for _, item := range req.Preview.Items {
		p.attempted = append(p.attempted, item.ID)
		if p.failing[item.ID] {
			out.SkippedItems = append(out.SkippedItems, item.ID)
			continue
		}
		p.removed[item.ID] = true
		out.AppliedItems = append(out.AppliedItems, item.ID)
		out.ReclaimedBytes += item.Bytes
	}
	out.Applied = out.ReclaimedBytes > 0
	return out, nil
}

func scriptedItems(ids ...string) []cleanup.PreviewItem {
	items := make([]cleanup.PreviewItem, 0, len(ids))
	for _, id := range ids {
		// Each item fills most of a batch, so a batch holds exactly one.
		items = append(items, cleanup.PreviewItem{ID: id, Path: "/tmp/" + id, Bytes: recoveryBatchCap * 3 / 4, Action: "remove", SafetyTier: cleanup.SafetyTierSafe})
	}
	return items
}

func newScriptedRecoveryProvider(failing ...string) *scriptedRecoveryProvider {
	p := &scriptedRecoveryProvider{tierProvider: tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}, failing: map[string]bool{}, removed: map[string]bool{}}
	for _, id := range failing {
		p.failing[id] = true
	}
	return p
}

// One unremovable entry at the head of a provider must not end the
// provider's turn: the same run retries without it and reaches the entries
// behind it.
func TestRecoveryAdvancesPastFailedItemWithinRun(t *testing.T) {
	provider := newScriptedRecoveryProvider("read-only-fixture")
	provider.items = scriptedItems("read-only-fixture", "removable")
	svc := newPressureService(t, provider)
	outcome, err := svc.executeRecovery(context.Background(), "run-1", "PRESSURE_TRIGGER_MANUAL", t.TempDir(), 0, 1<<62, false)
	if err != nil {
		t.Fatal(err)
	}
	if !provider.removed["removable"] || outcome.ReclaimedBytes != recoveryBatchCap*3/4 {
		t.Fatalf("outcome=%#v attempted=%v, want the item behind the failure reclaimed", outcome, provider.attempted)
	}
	if !reflect.DeepEqual(provider.attempted, []string{"read-only-fixture", "removable"}) {
		t.Fatalf("attempted = %v, want the failed item tried once per run", provider.attempted)
	}
}

// Zero-progress retries are bounded, so a provider whose every item fails
// cannot spin the run.
func TestRecoveryBoundsConsecutiveZeroProgressBatches(t *testing.T) {
	ids := make([]string, 0, 3*recoveryMaxStalledBatches)
	for i := 0; i < 3*recoveryMaxStalledBatches; i++ {
		ids = append(ids, fmt.Sprintf("failing-%d", i))
	}
	provider := newScriptedRecoveryProvider(ids...)
	provider.items = scriptedItems(ids...)
	svc := newPressureService(t, provider)
	if _, err := svc.executeRecovery(context.Background(), "run-1", "PRESSURE_TRIGGER_MANUAL", t.TempDir(), 0, 1<<62, false); err != nil {
		t.Fatal(err)
	}
	if len(provider.attempted) != recoveryMaxStalledBatches {
		t.Fatalf("attempted %d batches, want %d before advancing", len(provider.attempted), recoveryMaxStalledBatches)
	}
}

// A paused resumable measurement is resumed within the run, a bounded number
// of times, instead of waiting for the next pressure trigger.
func TestRecoveryResumesPendingMeasurementWithinRun(t *testing.T) {
	provider := newScriptedRecoveryProvider()
	provider.items = scriptedItems("expired-tree")
	provider.pending = 2
	svc := newPressureService(t, provider)
	outcome, err := svc.executeRecovery(context.Background(), "run-1", "PRESSURE_TRIGGER_MANUAL", t.TempDir(), 0, 1<<62, false)
	if err != nil {
		t.Fatal(err)
	}
	if !provider.removed["expired-tree"] || outcome.Action != ActionApplied {
		t.Fatalf("outcome=%#v, want the tree measured after two pending passes reclaimed", outcome)
	}
}
