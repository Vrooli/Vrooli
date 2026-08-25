package orchestrator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"storage-manager/internal/cleanup"
	"storage-manager/internal/providers"
)

// tierProvider is a provider whose safety tier and reclaimed bytes are chosen
// by the test. It records whether Apply ran, which is the only thing the tier
// boundary tests actually need to observe.
type tierProvider struct {
	id       string
	tier     cleanup.SafetyTier
	approval cleanup.ApprovalMode

	mu      sync.Mutex
	applied bool
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
		SupportedPlatforms:  []string{"linux"},
		IrreversibleEffects: []string{"removes files"},
		TestSubstitute:      "tier-provider",
	}
}

func (p *tierProvider) Estimate(context.Context, cleanup.EstimateRequest) (cleanup.Estimate, error) {
	return cleanup.Estimate{ProviderID: p.id, ProviderVersion: "v1", EstimatedBytes: 1000, ItemCount: 1}, nil
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

// TestReportPressure_CriticalAppliesSafeTier is the core capability: pressure
// reclaims space with nobody watching.
func TestReportPressure_CriticalAppliesSafeTier(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	outcome, err := svc.ReportPressure(context.Background(), criticalSignal())
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}

	if outcome.Action != ActionApplied {
		t.Errorf("action = %s, want applied", outcome.Action)
	}
	if !safe.didApply() {
		t.Error("a safe-tier provider did not run at the critical band; detection still does not reach action")
	}
	if outcome.ReclaimedBytes != 1000 {
		t.Errorf("reclaimed = %d, want 1000", outcome.ReclaimedBytes)
	}
	if len(outcome.ProvidersApplied) != 1 || outcome.ProvidersApplied[0] != "tmp" {
		t.Errorf("providers applied = %v, want [tmp]", outcome.ProvidersApplied)
	}

	// The run must be attributable after the fact.
	events, err := svc.Audit(context.Background())
	if err != nil {
		t.Fatalf("Audit: %v", err)
	}
	var sawApplied bool
	for _, e := range events {
		if e.Type == "pressure.applied" {
			sawApplied = true
		}
	}
	if !sawApplied {
		t.Error("an autonomous apply left no pressure.applied audit event")
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

			if blocked.didApply() {
				t.Fatalf("a %s provider executed through the autonomous path; the tier boundary does not hold", tc.tier)
			}
			if outcome.ReclaimedBytes != 0 {
				t.Errorf("reclaimed = %d, want 0", outcome.ReclaimedBytes)
			}
			if len(outcome.ProvidersWithheld) != 1 {
				t.Errorf("withheld = %v, want the blocked provider reported rather than silently dropped", outcome.ProvidersWithheld)
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

	if !safe.didApply() {
		t.Error("the safe-tier provider did not run")
	}
	if owner.didApply() || conditional.didApply() {
		t.Error("a provider above safe tier ran unattended")
	}
	if outcome.ReclaimedBytes != 1000 {
		t.Errorf("reclaimed = %d, want only the safe provider's 1000", outcome.ReclaimedBytes)
	}
	if len(outcome.ProvidersWithheld) != 2 {
		t.Errorf("withheld = %v, want both above-safe providers", outcome.ProvidersWithheld)
	}
}

// TestAutonomousTierAllowed pins the gate directly, so a future edit that
// widens it fails here with an explicit message rather than only showing up as
// unexpected deletion.
func TestAutonomousTierAllowed(t *testing.T) {
	allowed := map[cleanup.SafetyTier]bool{
		cleanup.SafetyTierSafe:          true,
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

// TestReportPressure_HighPreviewsWithoutDeleting asserts the high band looks
// but does not touch.
func TestReportPressure_HighPreviewsWithoutDeleting(t *testing.T) {
	safe := &tierProvider{id: "tmp", tier: cleanup.SafetyTierSafe, approval: cleanup.ApprovalModeNone}
	svc := newPressureService(t, safe)

	signal := criticalSignal()
	signal.Band = BandHigh
	signal.UsedPercent = 91

	outcome, err := svc.ReportPressure(context.Background(), signal)
	if err != nil {
		t.Fatalf("ReportPressure: %v", err)
	}

	if outcome.Action != ActionPreviewed {
		t.Errorf("action = %s, want previewed", outcome.Action)
	}
	if safe.didApply() {
		t.Error("the high band deleted something; it must only preview")
	}
	if outcome.EstimatedBytes == 0 {
		t.Error("a high-band preview reported no estimate, so an operator learns nothing")
	}
	if outcome.PlanID == "" {
		t.Error("a high-band preview produced no plan id to inspect")
	}
}

// TestReportPressure_WarningRunsSafeTier asserts warning pressure starts the
// bounded safe-tier action and leaves owner/conditional providers withheld.
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

	if outcome.Action != ActionApplied {
		t.Errorf("action = %s, want applied", outcome.Action)
	}
	if !safe.didApply() {
		t.Error("the warning band did not run the safe tier")
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
	if outcome.Action != ActionSuppressed || filed != 1 {
		t.Fatalf("warning with kill switch = action %q, filed %d; want suppressed, 1", outcome.Action, filed)
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

	if got := counter.applyCount(); got != 1 {
		t.Errorf("%d concurrent reports produced %d applies, want exactly 1", callers, got)
	}

	applied, deduped := 0, 0
	totalReclaimed := int64(0)
	for _, o := range outcomes {
		switch o.Action {
		case ActionApplied:
			applied++
			totalReclaimed += o.ReclaimedBytes
		case ActionDeduplicated:
			deduped++
		}
	}
	if applied != 1 {
		t.Errorf("%d callers reported an apply, want 1", applied)
	}
	if deduped != callers-1 {
		t.Errorf("%d callers were deduplicated, want %d", deduped, callers-1)
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

	// Observation must continue: the report is still planned and audited.
	if outcome.PlanID == "" {
		t.Error("the kill switch also stopped planning; observation must survive it")
	}
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
