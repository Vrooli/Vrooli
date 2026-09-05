package orchestrator

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/eventbus"
	"storage-manager/internal/cleanup"
	"storage-manager/internal/providers"
)

// tierProvider is a provider whose safety tier and reclaimed bytes are chosen
// by the test. It records whether Apply ran, which is the only thing the tier
// boundary tests actually need to observe.
type tierProvider struct {
	id          string
	tier        cleanup.SafetyTier
	approval    cleanup.ApprovalMode
	applyErr    error
	ownerBudget bool
	zeroReclaim bool
	oneShot     bool

	mu          sync.Mutex
	applied     bool
	estimateErr error
}

type fakeEventPublisher struct {
	mu    sync.Mutex
	types []string
}

type fakeJournalAppender struct {
	mu   sync.Mutex
	runs []RecoveryRun
}

type planRecordingStore struct {
	*MemoryStore
	mu            sync.Mutex
	planSaveCount int
}

func (s *planRecordingStore) SavePlan(ctx context.Context, plan Plan) error {
	s.mu.Lock()
	s.planSaveCount++
	s.mu.Unlock()
	return s.MemoryStore.SavePlan(ctx, plan)
}

func (s *planRecordingStore) savedPlans() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.planSaveCount
}

func (j *fakeJournalAppender) AppendRecovery(_ context.Context, run RecoveryRun) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.runs = append(j.runs, run)
	return nil
}

func (p *fakeEventPublisher) PublishDomainEvent(_ context.Context, event eventbus.DomainEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.types = append(p.types, event.EventType)
	return nil
}

func (p *tierProvider) Metadata() cleanup.ProviderMetadata {
	return cleanup.ProviderMetadata{
		ID:                  p.id,
		Name:                p.id,
		Version:             "v1",
		OwnerScenario:       "storage-manager",
		SafetyTier:          p.tier,
		DefaultMode:         cleanup.ProviderModeEnabled,
		DefaultApproval:     p.approval,
		OwnerBudget:         p.ownerBudget,
		SupportedPlatforms:  []string{"linux"},
		IrreversibleEffects: []string{"removes files"},
		TestSubstitute:      "tier-provider",
		RegenerableProof:    cleanup.RegenerableProof{Derived: true, ToolRecreates: true, ExactRoot: true, NoLease: p.tier == cleanup.SafetyTierRegenerable},
		NoLease:             p.tier == cleanup.SafetyTierRegenerable,
	}
}

func (p *tierProvider) Estimate(ctx context.Context, _ cleanup.EstimateRequest) (cleanup.Estimate, error) {
	p.mu.Lock()
	p.estimateErr = ctx.Err()
	p.mu.Unlock()
	if p.oneShot {
		p.mu.Lock()
		applied := p.applied
		p.mu.Unlock()
		if applied {
			return cleanup.Estimate{ProviderID: p.id, ProviderVersion: "v1"}, nil
		}
	}
	return cleanup.Estimate{ProviderID: p.id, ProviderVersion: "v1", EstimatedBytes: 1000, ItemCount: 1}, nil
}

func TestRecoveryUsesServiceLifetimeInsteadOfCallerContext(t *testing.T) {
	serviceCtx, cancelService := context.WithCancel(context.Background())
	defer cancelService()
	provider := &tierProvider{id: "service-context", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	registry, err := providers.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	svc := NewServiceWithContext(serviceCtx, registry, NewMemoryStore(), nil)
	callerCtx, cancelCaller := context.WithCancel(context.Background())
	cancelCaller()

	run, err := svc.StartRecovery(callerCtx, "PRESSURE_TRIGGER_MANUAL", "/", 96, 1024, 0, false)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	if _, err := svc.WaitRecovery(context.Background(), run.ID); err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	provider.mu.Lock()
	estimateErr := provider.estimateErr
	provider.mu.Unlock()
	if estimateErr != nil {
		t.Fatalf("provider received caller cancellation: %v", estimateErr)
	}
}

func (p *tierProvider) Preview(context.Context, cleanup.PreviewRequest) (cleanup.Preview, error) {
	return cleanup.Preview{
		ProviderID:      p.id,
		ProviderVersion: "v1",
		Items: []cleanup.PreviewItem{{
			ID: p.id + "/item", Path: "/tmp/item", Bytes: 1000, Action: "remove", SafetyTier: p.tier,
		}},
	}, nil
}

func (p *tierProvider) Apply(context.Context, cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	p.mu.Lock()
	p.applied = true
	p.mu.Unlock()
	if p.applyErr != nil {
		return cleanup.ApplyResult{}, p.applyErr
	}
	if p.zeroReclaim {
		return cleanup.ApplyResult{ProviderID: p.id, Applied: true}, nil
	}
	return cleanup.ApplyResult{ProviderID: p.id, Applied: true, ReclaimedBytes: 1000}, nil
}

func (p *tierProvider) Verify(context.Context, cleanup.VerifyRequest) (cleanup.VerifyResult, error) {
	return cleanup.VerifyResult{Verified: true}, nil
}

func (p *tierProvider) didApply() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.applied
}

// newPressureService builds a service whose registry contains exactly the
// providers given, with every provider enabled and requiring no approval.
//
// Enabling everything is deliberate: it means the tier tests below prove the
// tier filter blocks a provider that policy would otherwise happily run,
// rather than passing because policy blocked it first.
func newPressureService(t *testing.T, provs ...cleanup.Provider) *Service {
	t.Helper()

	registry, err := providers.NewRegistry(provs...)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	svc := NewService(registry, NewMemoryStore(), nil)

	policies := make(map[string]cleanup.ProviderPolicy, len(provs))
	for _, p := range provs {
		policies[p.Metadata().ID] = cleanup.ProviderPolicy{
			Enabled:      true,
			ApprovalMode: cleanup.ApprovalModeNone,
		}
	}
	if err := svc.store.SavePolicy(context.Background(), Policy{
		Version:   "policy-test",
		Providers: policies,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	return svc
}

func criticalSignal() PressureSignal {
	return PressureSignal{
		SourceScenario: "system-monitor",
		Partition:      "/",
		UsedPercent:    96,
		Band:           BandCritical,
		AvailableBytes: 1024,
	}
}

func waitPressureRun(t *testing.T, svc *Service, outcome PressureOutcome) RecoveryRun {
	t.Helper()
	if outcome.RunID == "" {
		t.Fatal("pressure report returned no recovery run id")
	}
	run, err := svc.WaitRecovery(context.Background(), outcome.RunID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	return run
}

func TestStartRecoveryReturnsImmediatelyAndWaitsForTerminalResult(t *testing.T) {
	provider := &tierProvider{id: "recovery-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, provider)
	events := &fakeEventPublisher{}
	svc.SetEventPublisher(events)
	run, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", 96, 1024, 0, false)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	if run.ID == "" || run.Status != RecoveryRunning {
		t.Fatalf("initial recovery run = %#v, want id and running status", run)
	}
	completed, err := svc.WaitRecovery(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if completed.Status != RecoveryComplete || completed.Action != ActionApplied || completed.ReclaimedBytes != 1000 {
		t.Fatalf("completed recovery run = %#v, want applied reclaim", completed)
	}
	if len(svc.ListRecovery(10)) != 1 {
		t.Fatalf("recovery history length = %d, want 1", len(svc.ListRecovery(10)))
	}
	if len(events.types) != 3 || events.types[0] != "storage.recovery.started" || events.types[1] != "storage.recovery.action" || events.types[2] != "storage.recovery.completed" {
		t.Fatalf("recovery events = %v, want started then completed", events.types)
	}
}

func TestDryRunWithoutEligibleProvidersIsReportedAsPreviewed(t *testing.T) {
	svc := newPressureService(t)
	run, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", 76, 1<<40, 15, true)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	completed, err := svc.WaitRecovery(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if completed.Status != RecoveryComplete || completed.Action != ActionPreviewed || completed.ReclaimedBytes != 0 {
		t.Fatalf("dry-run recovery = %#v, want complete preview with no reclaim", completed)
	}
}

func TestRecoveryApplyFailureIsReportedAsFailed(t *testing.T) {
	provider := &tierProvider{id: "failing-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone, applyErr: errors.New("provider apply failed")}
	svc := newPressureService(t, provider)
	run, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", 96, 1024, 0, false)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	completed, err := svc.WaitRecovery(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if completed.Status != RecoveryFailed || completed.StoppedBecause != "error" {
		t.Fatalf("failed recovery = %#v, want failed/error", completed)
	}
	if !strings.Contains(completed.Reason, "provider apply failed") {
		t.Fatalf("failed recovery reason = %q, want provider error", completed.Reason)
	}
}

func TestRecoveryAdvancesPastZeroByteProvider(t *testing.T) {
	zero := &tierProvider{id: "a-zero-owner", tier: cleanup.SafetyTierSafeWithOwner, approval: cleanup.ApprovalModeOwner, ownerBudget: true, zeroReclaim: true, oneShot: true}
	safe := &tierProvider{id: "z-reclaim-owner", tier: cleanup.SafetyTierSafeWithOwner, approval: cleanup.ApprovalModeOwner, ownerBudget: true, oneShot: true}
	svc := newPressureService(t, zero, safe)
	outcome, err := svc.executeRecovery(context.Background(), "run-zero-progress", "PRESSURE_TRIGGER_MANUAL", "/", 0, math.MaxInt64, false)
	if err != nil {
		t.Fatalf("executeRecovery: %v", err)
	}
	if !zero.didApply() || !safe.didApply() {
		t.Fatalf("providers applied: zero=%t safe=%t; want both", zero.didApply(), safe.didApply())
	}
	if outcome.ReclaimedBytes != 1000 {
		t.Fatalf("reclaimed bytes = %d, want 1000 after advancing past zero-byte provider", outcome.ReclaimedBytes)
	}
}

func TestRecoveryWritesOneJournalWorkRecordAfterReclaim(t *testing.T) {
	provider := &tierProvider{id: "journal-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, provider)
	journal := &fakeJournalAppender{}
	svc.SetJournalAppender(journal)
	run, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", 96, 1024, 0, false)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	completed, err := svc.WaitRecovery(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if completed.ReclaimedBytes <= 0 || len(journal.runs) != 1 || journal.runs[0].ID != completed.ID {
		t.Fatalf("journal runs = %#v, completed = %#v", journal.runs, completed)
	}
}

func TestRecoveryWritesJournalWorkRecordWhenTargetAlreadyMet(t *testing.T) {
	svc := newPressureService(t)
	journal := &fakeJournalAppender{}
	svc.SetJournalAppender(journal)
	run, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", 1, 1<<40, 15, false)
	if err != nil {
		t.Fatalf("StartRecovery: %v", err)
	}
	completed, err := svc.WaitRecovery(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if completed.StoppedBecause != "target_met" || len(journal.runs) != 1 || journal.runs[0].ID != completed.ID {
		t.Fatalf("journal runs = %#v, completed = %#v", journal.runs, completed)
	}
}

func TestAutonomousRecoveryOverridesDefaultDisabledPolicy(t *testing.T) {
	for _, tier := range []cleanup.SafetyTier{cleanup.SafetyTierSafe, cleanup.SafetyTierRegenerable} {
		t.Run(string(tier), func(t *testing.T) {
			provider := &tierProvider{id: "disabled-" + string(tier), tier: tier, approval: cleanup.ApprovalModeDisabled}
			svc := newPressureService(t, provider)
			if err := svc.store.SavePolicy(context.Background(), Policy{
				Version: "disabled-test", Providers: map[string]cleanup.ProviderPolicy{
					provider.id: {Enabled: false, ApprovalMode: cleanup.ApprovalModeDisabled},
				}, CreatedAt: time.Now().UTC(),
			}); err != nil {
				t.Fatal(err)
			}
			outcome, err := svc.executeRecovery(context.Background(), "disabled-"+string(tier), "PRESSURE_TRIGGER_MANUAL", "/", 0, math.MaxInt64, false)
			if err != nil {
				t.Fatalf("executeRecovery: %v", err)
			}
			if !provider.didApply() || outcome.ReclaimedBytes <= 0 {
				t.Fatalf("provider applied=%t outcome=%#v; autonomous tier did not override disabled policy", provider.didApply(), outcome)
			}
		})
	}
}

func TestReportPressureWithTriggerReturnsRecoveryRunIDBeforeApply(t *testing.T) {
	provider := &tierProvider{id: "trigger-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, provider)
	started := time.Now()
	outcome, err := svc.ReportPressure(context.Background(), PressureSignal{
		SourceScenario: "system-monitor", Partition: "/", UsedPercent: 96,
		AvailableBytes: 1024, Band: BandCritical, Trigger: "PRESSURE_TRIGGER_FLOOR",
	})
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	if outcome.RunID == "" || time.Since(started) > time.Second {
		t.Fatalf("outcome = %#v, expected prompt run id", outcome)
	}
	if outcome.ReclaimedBytes != 0 {
		t.Fatalf("pressure response reclaimed bytes = %d before wait", outcome.ReclaimedBytes)
	}
	run, err := svc.WaitRecovery(context.Background(), outcome.RunID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if run.Status != RecoveryComplete || run.ReclaimedBytes != 1000 {
		t.Fatalf("run = %#v", run)
	}
}

func TestBoundedRecoveryPreviewCapsOneBatch(t *testing.T) {
	capBytes := int64(2 * 1024 * 1024 * 1024)
	preview := cleanup.Preview{Items: []cleanup.PreviewItem{
		{ID: "old", Bytes: capBytes - 1},
		{ID: "would-overrun", Bytes: 2},
		{ID: "too-large", Bytes: capBytes + 1},
	}}

	bounded := boundedRecoveryPreview(preview, capBytes)
	if len(bounded.Items) != 1 || bounded.Items[0].ID != "old" {
		t.Fatalf("bounded preview = %#v, want only the first fitting item", bounded.Items)
	}
	if got := sumRecoveryPreviewBytes(bounded.Items); got > capBytes {
		t.Fatalf("bounded preview bytes = %d, exceeds cap %d", got, capBytes)
	}
}

func TestBoundedRecoveryPreviewAllowsExplicitAtomicOvershoot(t *testing.T) {
	preview := cleanup.Preview{
		AllowSingleOvershoot: true,
		Items:                []cleanup.PreviewItem{{ID: "atomic", Bytes: 3 * 1024 * 1024 * 1024}},
	}
	bounded := boundedRecoveryPreview(preview, 2*1024*1024*1024)
	if len(bounded.Items) != 1 || bounded.Items[0].ID != "atomic" {
		t.Fatalf("bounded atomic preview = %#v, want one explicit oversize item", bounded.Items)
	}

	preview.AllowSingleOvershoot = false
	bounded = boundedRecoveryPreview(preview, 2*1024*1024*1024)
	if len(bounded.Items) != 0 {
		t.Fatalf("ordinary oversize preview = %#v, want no items", bounded.Items)
	}
}

func TestRecoveryTargetUsesUsableCapacityAndFloor(t *testing.T) {
	const gib = int64(1024 * 1024 * 1024)
	if got, want := recoveryTargetBytes(80, 200*gib, 0), int64(150)*gib; got != want {
		t.Fatalf("target at 80%% = %d, want %d", got, want)
	}
	if got := recoveryTargetBytes(99, 1, 0); got != 10*gib {
		t.Fatalf("target floor = %d, want 10 GiB", got)
	}
	if got, want := recoveryTargetBytes(80, 200*gib, 50), int64(500)*gib; got != want {
		t.Fatalf("requested 50%% target = %d, want %d", got, want)
	}
}

func TestStartRecoveryRejectsNonFinitePressureInputs(t *testing.T) {
	svc := NewService(nil, NewMemoryStore(), nil)
	for _, input := range []struct {
		name   string
		used   float64
		target float64
	}{
		{name: "nan used", used: math.NaN()},
		{name: "infinite used", used: math.Inf(1)},
		{name: "nan target", target: math.NaN()},
		{name: "infinite target", target: math.Inf(1)},
	} {
		t.Run(input.name, func(t *testing.T) {
			if _, err := svc.StartRecovery(context.Background(), "PRESSURE_TRIGGER_MANUAL", "/", input.used, 1, input.target, true); err == nil {
				t.Fatal("StartRecovery accepted a non-finite pressure input")
			}
		})
	}
}

// TestReportPressure_CriticalAppliesSafeTier is the core capability: pressure
// reclaims space with nobody watching.
func TestReportPressure_CriticalAppliesSafeTier(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	outcome, err := svc.ReportPressure(context.Background(), criticalSignal())
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	if outcome.RunID == "" {
		t.Fatal("ReportPressure returned no server-owned recovery run id")
	}
	run := waitPressureRun(t, svc, outcome)

	if outcome.Action != ActionApplied {
		t.Errorf("action = %s, want applied", outcome.Action)
	}
	if !safe.didApply() {
		t.Error("a safe-tier provider did not run at the critical band; detection still does not reach action")
	}
	if run.ReclaimedBytes != 1000 {
		t.Errorf("reclaimed = %d, want 1000", run.ReclaimedBytes)
	}
}

func TestReportPressureHighWithoutTriggerSkipsCensus(t *testing.T) {
	provider := &tierProvider{id: "safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	registry, err := providers.NewRegistry(provider)
	if err != nil {
		t.Fatalf("NewRegistry: %v", err)
	}
	base := NewMemoryStore()
	svc := NewService(registry, base, nil)
	if err := base.SavePolicy(context.Background(), Policy{
		Version:   "policy-test",
		Providers: map[string]cleanup.ProviderPolicy{"safe": {Enabled: true, ApprovalMode: cleanup.ApprovalModeNone}},
	}); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}
	recording := &planRecordingStore{MemoryStore: base}
	svc.store = recording

	outcome, err := svc.ReportPressure(context.Background(), PressureSignal{
		SourceScenario: "legacy-sender", Partition: "/", UsedPercent: 96,
		AvailableBytes: 1024, Band: BandHigh,
	})
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	run := waitPressureRun(t, svc, outcome)
	if run.Status != RecoveryComplete || run.ReclaimedBytes != 1000 {
		t.Fatalf("recovery run = %#v, want completed run reclaiming 1000 bytes", run)
	}
	if got := recording.savedPlans(); got != 0 {
		t.Fatalf("pressure path saved %d census plans; want zero", got)
	}
}

// TestReportPressure_RefusesTiersAboveSafe is the correctness-critical test the
// plan calls out by name.
//
// At the critical band the system deletes with no operator present, so the
// safety-tier classification is the only thing preventing silent data loss.
// Every provider here is enabled with no approval required, so policy would
// gladly run them: the tier filter must be what stops them.
func TestReportPressure_RefusesTiersAboveSafe(t *testing.T) {
	tests := []struct {
		name string
		tier cleanup.SafetyTier
		// declaredApproval is the provider's own metadata. Conditional
		// providers are required by ValidateProviderMetadata to declare
		// operator approval, so it cannot be None here.
		declaredApproval cleanup.ApprovalMode
	}{
		{"safe_with_owner delegates deletion to another scenario", cleanup.SafetyTierSafeWithOwner, cleanup.ApprovalModeNone},
		{"conditional is not provably reversible", cleanup.SafetyTierConditional, cleanup.ApprovalModeOperator},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The installed policy grants this provider ApprovalModeNone and
			// enables it, so nothing except the tier filter can stop it.
			blocked := &tierProvider{id: "blocked", tier: tc.tier, approval: tc.declaredApproval}
			svc := newPressureService(t, blocked)

			outcome, err := svc.ReportPressure(context.Background(), criticalSignal())
			if err != nil {
				t.Fatalf("ReportPressure: %v", err)
			}
			waitPressureRun(t, svc, outcome)

			if blocked.didApply() {
				t.Fatalf("a %s provider executed through the autonomous path; the tier boundary does not hold", tc.tier)
			}
		})
	}
}

// TestReportPressure_MixedTiersRunsOnlySafe asserts the filter is per-provider,
// so one blocked provider neither blocks nor authorises the others.
func TestReportPressure_MixedTiersRunsOnlySafe(t *testing.T) {
	safe := &tierProvider{id: "safe-one", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	owner := &tierProvider{id: "owner-one", tier: cleanup.SafetyTierSafeWithOwner, approval: cleanup.ApprovalModeNone}
	conditional := &tierProvider{id: "conditional-one", tier: cleanup.SafetyTierConditional, approval: cleanup.ApprovalModeOperator}

	svc := newPressureService(t, safe, owner, conditional)

	outcome, err := svc.ReportPressure(context.Background(), criticalSignal())
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	run := waitPressureRun(t, svc, outcome)

	if !safe.didApply() {
		t.Error("the safe-tier provider did not run")
	}
	if owner.didApply() || conditional.didApply() {
		t.Error("a provider above safe tier ran unattended")
	}
	if run.ReclaimedBytes != 1000 {
		t.Errorf("reclaimed = %d, want only the safe provider's 1000", run.ReclaimedBytes)
	}
}

// TestAutonomousTierAllowed pins the gate directly, so a future edit that
// widens it fails here with an explicit message rather than only showing up as
// unexpected deletion.
func TestAutonomousTierAllowed(t *testing.T) {
	allowed := map[cleanup.SafetyTier]bool{
		cleanup.SafetyTierSafe:          true,
		cleanup.SafetyTierRegenerable:   true,
		cleanup.SafetyTierSafeWithOwner: false,
		cleanup.SafetyTierConditional:   false,
		cleanup.SafetyTierForbidden:     false,
	}
	for tier, want := range allowed {
		if got := autonomousTierAllowed(tier); got != want {
			t.Errorf("autonomousTierAllowed(%s) = %v, want %v — unattended deletion scope changed", tier, got, want)
		}
	}
}

// TestReportPressure_HighAppliesSafeTier asserts high pressure automatically
// reclaims only the provably safe tier.
func TestReportPressure_HighAppliesSafeTier(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	signal := criticalSignal()
	signal.Band = BandHigh
	signal.UsedPercent = 91

	outcome, err := svc.ReportPressure(context.Background(), signal)
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	run := waitPressureRun(t, svc, outcome)

	if outcome.Action != ActionApplied {
		t.Errorf("action = %s, want applied", outcome.Action)
	}
	if !safe.didApply() {
		t.Error("the high band did not apply safe-tier cleanup")
	}
	if run.ReclaimedBytes == 0 {
		t.Error("a high-band apply reported no reclaimed bytes")
	}
	if run.PlanID == "" {
		t.Error("a high-band apply produced no plan id to inspect")
	}
}

// TestReportPressure_WarningObservesWithoutCleanup asserts warning pressure
// records the signal and leaves deletion to an explicit recovery trigger.
func TestReportPressure_WarningRunsSafeTier(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	signal := criticalSignal()
	signal.Band = BandWarning
	signal.UsedPercent = 82

	outcome, err := svc.ReportPressure(context.Background(), signal)
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}

	if outcome.Action != ActionObserved {
		t.Errorf("action = %s, want observed", outcome.Action)
	}
	if safe.didApply() {
		t.Error("warning pressure unexpectedly ran the safe tier")
	}
	if outcome.RunID == "" || len(svc.ListRecovery(1)) != 1 || svc.ListRecovery(1)[0].Action != ActionObserved {
		t.Fatalf("warning recovery ledger = %#v, want one observed run", svc.ListRecovery(1))
	}
}

func TestReportPressure_WarningRateTriggerStartsRecovery(t *testing.T) {
	safe := &tierProvider{id: "rate-safe", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)
	governedHome := t.TempDir()
	t.Setenv("VROOLI_HOME", governedHome)
	governedRoot := filepath.Join(governedHome, "tmp", "go-work", "governed-hot-root")
	if err := os.MkdirAll(governedRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll governed root: %v", err)
	}
	signal := criticalSignal()
	signal.Band = BandWarning
	signal.Trigger = "PRESSURE_TRIGGER_RATE"
	signal.HotWriters = []HotWriter{{Root: governedRoot, BytesPerHour: 2 * 1024 * 1024 * 1024, WindowSeconds: 60}}

	outcome, err := svc.ReportPressure(context.Background(), signal)
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}
	if outcome.Action != ActionApplied || outcome.RunID == "" {
		t.Fatalf("rate warning outcome = %#v, want an applied recovery run", outcome)
	}
	if got := svc.ListRecovery(1)[0].Partition; got != governedRoot {
		t.Fatalf("rate recovery partition = %q, want named hot root", got)
	}
	run, err := svc.WaitRecovery(context.Background(), outcome.RunID)
	if err != nil {
		t.Fatalf("WaitRecovery: %v", err)
	}
	if run.Status != RecoveryComplete || !safe.didApply() {
		t.Fatalf("rate warning run = %#v, provider applied = %v", run, safe.didApply())
	}
}

func TestRecoveryPartitionForSignalFallsBackSafely(t *testing.T) {
	cases := []struct {
		name   string
		signal PressureSignal
		want   string
	}{
		{name: "one absolute root", signal: PressureSignal{Partition: "/", Trigger: "PRESSURE_TRIGGER_RATE", HotWriters: []HotWriter{{Root: "/tmp/hot"}}}, want: "/tmp/hot"},
		{name: "multiple roots", signal: PressureSignal{Partition: "/", Trigger: "PRESSURE_TRIGGER_RATE", HotWriters: []HotWriter{{Root: "/tmp/a"}, {Root: "/tmp/b"}}}, want: "/"},
		{name: "relative root", signal: PressureSignal{Partition: "/", Trigger: "PRESSURE_TRIGGER_RATE", HotWriters: []HotWriter{{Root: "tmp/hot"}}}, want: "/"},
		{name: "root mount", signal: PressureSignal{Partition: "/", Trigger: "PRESSURE_TRIGGER_RATE", HotWriters: []HotWriter{{Root: "/"}}}, want: "/"},
		{name: "non-rate trigger", signal: PressureSignal{Partition: "/", Trigger: "PRESSURE_TRIGGER_BAND", HotWriters: []HotWriter{{Root: "/tmp/hot"}}}, want: "/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := recoveryPartitionForSignal(tc.signal); got != tc.want {
				t.Fatalf("partition = %q, want %q", got, tc.want)
			}
		})
	}
}

type warningTestClock struct{ now time.Time }

func (c warningTestClock) Now() time.Time { return c.now }

func TestReportPressure_WarningFilesFastestUnboundedOwnerOncePerDay(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	svc.clock = warningTestClock{now: now}
	var reports []WarningBugReport
	svc.SetWarningDependencies(WarningDependencies{
		FastestUnbounded: func(context.Context) (WarningGrowthTarget, bool, error) {
			return WarningGrowthTarget{OwnerKind: "scenario", OwnerID: "fixture", EntryName: "records", CurrentBytes: 100, SlopeBytesPerHour: 25}, true, nil
		},
		FileBug: func(_ context.Context, report WarningBugReport) (string, error) {
			reports = append(reports, report)
			return "knw-growth-1", nil
		},
	})

	for _, partition := range []string{"/", "/var"} {
		signal := criticalSignal()
		signal.Band = BandWarning
		signal.Partition = partition
		outcome, err := svc.ReportPressure(context.Background(), signal)
		if err != nil {
			t.Fatalf("ReportPressure(%s): %v", partition, err)
		}
		if outcome.BugReference != "knw-growth-1" && partition == "/" {
			t.Fatalf("first warning bug reference = %q", outcome.BugReference)
		}
	}
	if len(reports) != 1 {
		t.Fatalf("bug reports = %d, want one within the daily owner limit", len(reports))
	}
	if reports[0].IdempotencyKey == "" || reports[0].Actual == "" {
		t.Fatalf("warning bug omitted idempotency or measured actual: %+v", reports[0])
	}

	svc.clock = warningTestClock{now: now.Add(25 * time.Hour)}
	signal := criticalSignal()
	signal.Band = BandWarning
	signal.Partition = "/home"
	if _, err := svc.ReportPressure(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("bug reports after daily limit = %d, want two", len(reports))
	}
}

func TestReportPressure_WarningBugSurvivesAutonomousKillSwitch(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)
	var filed int
	svc.SetWarningDependencies(WarningDependencies{
		FastestUnbounded: func(context.Context) (WarningGrowthTarget, bool, error) {
			return WarningGrowthTarget{OwnerID: "fixture", EntryName: "records", SlopeBytesPerHour: 1}, true, nil
		},
		FileBug: func(context.Context, WarningBugReport) (string, error) { filed++; return "knw-growth-2", nil },
	})
	svc.SetAutonomousApplyEnabled(false)
	signal := criticalSignal()
	signal.Band = BandWarning
	outcome, err := svc.ReportPressure(context.Background(), signal)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Action != ActionObserved || filed != 1 {
		t.Fatalf("warning with kill switch = action %q, filed %d; want observed, 1", outcome.Action, filed)
	}
}

// TestReportPressure_RejectsUnknownBand asserts an unrecognised band is refused
// rather than defaulted. Defaulting to critical would let a typo authorise
// unattended deletion.
func TestReportPressure_RejectsUnknownBand(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	for _, band := range []string{"", "urgent", "CRITICAL!", "normal"} {
		signal := criticalSignal()
		signal.Band = PressureBand(band)

		if _, err := svc.ReportPressure(context.Background(), signal); err == nil {
			t.Errorf("band %q was accepted; unknown bands must be rejected", band)
		}
		if safe.didApply() {
			t.Fatalf("band %q triggered an apply", band)
		}
	}
}

// TestReportPressure_ConcurrentReportsRunOnce asserts the two independent
// safeguards reporting the same event produce one execution.
//
// Two paths reach storage-manager by design so they do not share a point of
// failure; the cost is that duplicate concurrent reports are expected, and
// running cleanup twice would double-count reclaimed bytes.
func TestReportPressure_ConcurrentReportsRunOnce(t *testing.T) {
	counter := &countingProvider{tierProvider: tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}}
	svc := newPressureService(t, counter)

	const callers = 8
	var wg sync.WaitGroup
	outcomes := make([]PressureOutcome, callers)
	errs := make([]error, callers)

	start := make(chan struct{})
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			signal := criticalSignal()
			// Alternate the reporting scenario: the same event seen by both
			// safeguards must still collapse to one execution.
			if i%2 == 0 {
				signal.SourceScenario = "vrooli-autoheal"
			}
			<-start
			outcomes[i], errs[i] = svc.ReportPressure(context.Background(), signal)
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("caller %d: %v", i, err)
		}
	}
	// ReportPressure returns the server-owned run id before filesystem work
	// begins. Wait on that run before inspecting the apply count; otherwise the
	// assertion races the deliberately asynchronous recovery worker.
	for _, outcome := range outcomes {
		if outcome.RunID != "" {
			_ = waitPressureRun(t, svc, outcome)
		}
	}

	if got := counter.applyCount(); got != 1 {
		t.Errorf("%d concurrent reports produced %d applies, want exactly 1", callers, got)
	}

	completed, lockHeld, deduped := 0, 0, 0
	totalReclaimed := int64(0)
	for _, o := range outcomes {
		if o.RunID == "" {
			if o.Action != ActionDeduplicated {
				t.Errorf("duplicate pressure outcome without run id = %#v, want deduplicated", o)
			}
			deduped++
			continue
		}
		run := waitPressureRun(t, svc, o)
		switch run.StoppedBecause {
		case "lock_held":
			lockHeld++
		default:
			if run.Status == RecoveryComplete {
				completed++
				totalReclaimed += run.ReclaimedBytes
			}
		}
	}
	if completed != 1 {
		t.Errorf("%d concurrent runs completed recovery, want 1", completed)
	}
	if lockHeld+deduped != callers-1 {
		t.Errorf("%d concurrent reports were lock-held or deduplicated, want %d (lock-held=%d deduplicated=%d)", lockHeld+deduped, callers-1, lockHeld, deduped)
	}
	if totalReclaimed != 1000 {
		t.Errorf("reclaimed bytes summed to %d across callers, want 1000 — space freed once must not be counted twice", totalReclaimed)
	}
}

// TestReportPressure_KillSwitchBlocksApplyButRecords asserts an operator can
// stop unattended deletion without going blind.
func TestReportPressure_KillSwitchBlocksApplyButRecords(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)
	svc.SetAutonomousApplyEnabled(false)

	outcome, err := svc.ReportPressure(context.Background(), criticalSignal())
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}

	if safe.didApply() {
		t.Fatal("the kill switch did not stop unattended deletion")
	}
	if outcome.Action != ActionSuppressed {
		t.Errorf("action = %s, want suppressed", outcome.Action)
	}
	if outcome.AutonomousApplyEnabled {
		t.Error("the response claims autonomous apply is enabled while the kill switch is off")
	}
	if outcome.Reason == "" {
		t.Error("suppression gave no reason")
	}

	// Observation must continue: the report is still audited even though
	// autonomous recovery was suppressed.
	events, err := svc.Audit(context.Background())
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	var sawReport, sawSuppressed bool
	for _, e := range events {
		switch e.Type {
		case "pressure.reported":
			sawReport = true
		case "pressure.suppressed":
			sawSuppressed = true
		}
	}
	if !sawReport || !sawSuppressed {
		t.Error("a suppressed report left an incomplete audit trail")
	}

	// Re-enabling restores remediation.
	svc.SetAutonomousApplyEnabled(true)
	if !svc.AutonomousApplyEnabled() {
		t.Error("the kill switch could not be turned back on")
	}
}

// TestParsePressureBand covers the parser directly, including the casing and
// whitespace a hand-written caller might send.
func TestParsePressureBand(t *testing.T) {
	valid := map[string]PressureBand{
		"warning":    BandWarning,
		"HIGH":       BandHigh,
		" critical ": BandCritical,
	}
	for raw, want := range valid {
		got, err := ParsePressureBand(raw)
		if err != nil {
			t.Errorf("ParsePressureBand(%q) errored: %v", raw, err)
			continue
		}
		if got != want {
			t.Errorf("ParsePressureBand(%q) = %s, want %s", raw, got, want)
		}
	}

	for _, raw := range []string{"", "normal", "urgent", "3"} {
		if _, err := ParsePressureBand(raw); err == nil {
			t.Errorf("ParsePressureBand(%q) accepted an unknown band", raw)
		}
	}
}

// countingProvider counts how many times Apply ran, for the concurrency test.
type countingProvider struct {
	tierProvider
	mu    sync.Mutex
	count int
}

func (p *countingProvider) Apply(ctx context.Context, req cleanup.ApplyRequest) (cleanup.ApplyResult, error) {
	p.mu.Lock()
	p.count++
	p.mu.Unlock()
	// Hold long enough that a missing guard would let a second caller in.
	time.Sleep(20 * time.Millisecond)
	return p.tierProvider.Apply(ctx, req)
}

func (p *countingProvider) applyCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.count
}

// TestReportPressure_ClassifiesCallerErrors asserts that a malformed signal is
// distinguishable from a storage-manager-side failure.
//
// The transport maps ErrInvalidPressureSignal to InvalidArgument and everything
// else to Internal. Getting this wrong is not cosmetic: a reporting safeguard
// told "invalid argument" should stop and fix its request, while one told
// "internal" should keep reporting. Before this distinction existed, a plan that
// blew the write timeout came back to the reporter as a 400 — telling the
// safeguard its perfectly valid signal was malformed.
func TestReportPressure_ClassifiesCallerErrors(t *testing.T) {
	t.Parallel()

	svc := newPressureService(t)
	ctx := context.Background()

	t.Run("unknown band is the caller's fault", func(t *testing.T) {
		_, err := svc.ReportPressure(ctx, PressureSignal{Partition: "/", Band: PressureBand("sideways")})
		if !errors.Is(err, ErrInvalidPressureSignal) {
			t.Errorf("err = %v, want it to wrap ErrInvalidPressureSignal", err)
		}
	})

	t.Run("missing partition is the caller's fault", func(t *testing.T) {
		_, err := svc.ReportPressure(ctx, PressureSignal{Partition: "  ", Band: BandHigh})
		if !errors.Is(err, ErrInvalidPressureSignal) {
			t.Errorf("err = %v, want it to wrap ErrInvalidPressureSignal", err)
		}
	})
}
