package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/coreset"
	"github.com/vrooli/api-core/demand"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	checksvrooli "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks/vrooli"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/userconfig"
)

type mutableSupervisionExecutor struct {
	mu     sync.Mutex
	output []byte
	err    error
}

type coreDemandFixture struct {
	mu       sync.Mutex
	acquires []demand.AcquireRequest
	renews   []string
	releases []string
	err      error
}

func (f *coreDemandFixture) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.acquires = append(f.acquires, req)
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: req.LeaseID, Scenario: req.Scenario, ConsumerID: req.ConsumerID, Kind: req.Kind, ExpiresAt: time.Now().Add(req.TTL)}, nil
}

func (f *coreDemandFixture) Renew(_ context.Context, leaseID string, _ time.Duration) (demand.Lease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.renews = append(f.renews, leaseID)
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: leaseID}, nil
}

func (f *coreDemandFixture) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.releases = append(f.releases, leaseID)
	if f.err != nil {
		return demand.Lease{}, f.err
	}
	return demand.Lease{LeaseID: leaseID, Status: "released"}, nil
}

func (e *mutableSupervisionExecutor) set(report coreset.Report, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.err = err
	if report.Members == nil {
		e.output = nil
		return
	}
	e.output, _ = json.Marshal(report)
}

func (e *mutableSupervisionExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]byte(nil), e.output...), e.err
}

func (e *mutableSupervisionExecutor) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	return e.Output(ctx, name, args...)
}

func (e *mutableSupervisionExecutor) Run(context.Context, string, ...string) error { return e.err }

func supervisionReport(members ...coreset.Member) coreset.Report {
	return coreset.Report{Source: "control-plane", Members: members}
}

func supervisedMember(name, kind, intent string) coreset.Member {
	return coreset.Member{
		Name:              name,
		Kind:              kind,
		SupervisionIntent: intent,
		AttributionChain: []coreset.AttributionStep{{
			Name: name, Kind: kind, SupervisionIntent: intent, Source: "core.seed",
		}},
	}
}

func TestSupervisionSourceRetainsLastKnownGoodOnFailure(t *testing.T) {
	executor := &mutableSupervisionExecutor{}
	report := supervisionReport(supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentMustStart))
	executor.set(report, nil)
	source := NewSupervisionSource(executor)

	first, err := source.Load(context.Background())
	if err != nil || first.Degraded {
		t.Fatalf("initial load = %+v, %v", first, err)
	}
	executor.set(coreset.Report{}, errors.New("control plane unavailable"))
	fallback, err := source.Load(context.Background())
	if err != nil {
		t.Fatalf("last-known-good load returned error: %v", err)
	}
	if !fallback.Degraded || len(fallback.Report.Members) != 1 || fallback.SourceErr == "" {
		t.Fatalf("fallback did not preserve and report last-known-good state: %+v", fallback)
	}
}

func TestSupervisionSourceFailsClosedWithoutLastKnownGood(t *testing.T) {
	executor := &mutableSupervisionExecutor{err: errors.New("control plane unavailable")}
	_, err := NewSupervisionSource(executor).Load(context.Background())
	if err == nil {
		t.Fatal("source must fail closed instead of returning an empty supervision set")
	}
}

func TestSupervisionControllerReconcilesCanonicalSetAndAdditiveOverrides(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Clean("../../../../.."))
	tmp := t.TempDir()
	mgr := userconfig.NewManager(filepath.Join(tmp, "config.json"), filepath.Join(tmp, "schema.json"))
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddScenario("operator-extra", false); err != nil {
		t.Fatal(err)
	}
	if err := mgr.AddResource("redis"); err != nil {
		t.Fatal(err)
	}
	if err := mgr.SetCheckEnabled("scenario-search-hub", false); err != nil {
		t.Fatal(err)
	}

	executor := &mutableSupervisionExecutor{}
	executor.set(supervisionReport(
		supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentMustStart),
		supervisedMember("qdrant", coreset.MemberKindResource, coreset.IntentTryStart),
	), nil)
	registry := checks.NewRegistry(&platform.Capabilities{Platform: platform.Linux})
	controller := NewSupervisionController(registry, mgr, NewSupervisionSource(executor))
	if _, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"scenario-search-hub", "resource-qdrant"} {
		if _, ok := registry.GetCheck(id); !ok {
			t.Errorf("active checks missing %q", id)
		}
	}
	canonical, _ := registry.GetCheck("scenario-search-hub")
	if scenario, ok := canonical.(*checksvrooli.ScenarioCheck); !ok || !scenario.IsCritical() {
		t.Fatalf("must_start scenario was not critical: %T", canonical)
	}
	if !mgr.IsCheckEnabled("scenario-search-hub") || !mgr.IsAutoHealEnabled("scenario-search-hub") {
		t.Fatal("stale operator check config disabled the canonical supervision floor")
	}

	executor.set(supervisionReport(supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentTryStart)), nil)
	if _, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, ok := registry.GetCheck("resource-qdrant"); ok {
		t.Fatal("removed canonical member remained registered after reload")
	}
	if _, ok := registry.GetCheck("resource-redis"); ok {
		t.Fatal("stale operator override remained registered after canonical reload")
	}
	canonical, _ = registry.GetCheck("scenario-search-hub")
	if scenario := canonical.(*checksvrooli.ScenarioCheck); scenario.IsCritical() {
		t.Fatal("try_start scenario must use warning severity")
	}
}

func TestSupervisionControllerReconcilesCoreDemandLeases(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Clean("../../../../.."))
	tmp := t.TempDir()
	mgr := userconfig.NewManager(filepath.Join(tmp, "config.json"), filepath.Join(tmp, "schema.json"))
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}

	executor := &mutableSupervisionExecutor{}
	executor.set(supervisionReport(
		supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentMustStart),
		supervisedMember("qdrant", coreset.MemberKindResource, coreset.IntentTryStart),
	), nil)
	demandFixture := &coreDemandFixture{}
	controller := NewSupervisionController(
		checks.NewRegistry(&platform.Capabilities{Platform: platform.Linux}),
		mgr,
		NewSupervisionSource(executor),
		demandFixture,
	)
	if snapshot, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	} else if len(snapshot.DemandErrors) != 0 {
		t.Fatalf("initial demand reconciliation reported errors: %v", snapshot.DemandErrors)
	}
	if len(demandFixture.acquires) != 1 {
		t.Fatalf("acquires = %d, want one scenario core hold", len(demandFixture.acquires))
	}
	acquired := demandFixture.acquires[0]
	if acquired.Scenario != "search-hub" || acquired.Kind != demand.KindCore || acquired.ConsumerID != coreSupervisionConsumerID {
		t.Fatalf("unexpected core lease request: %+v", acquired)
	}
	if acquired.TTL != coreSupervisionLeaseTTL {
		t.Fatalf("core lease ttl = %s, want %s", acquired.TTL, coreSupervisionLeaseTTL)
	}

	if _, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(demandFixture.renews) != 1 || len(demandFixture.acquires) != 1 {
		t.Fatalf("second refresh should renew existing hold (acquires=%d renews=%d)", len(demandFixture.acquires), len(demandFixture.renews))
	}

	executor.set(supervisionReport(supervisedMember("qdrant", coreset.MemberKindResource, coreset.IntentTryStart)), nil)
	if _, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(demandFixture.releases) != 1 || demandFixture.releases[0] != acquired.LeaseID {
		t.Fatalf("removed scenario core hold was not released: %+v", demandFixture.releases)
	}
}

func TestSupervisionControllerKeepsCoreDemandLeasesOnDegradedSource(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Clean("../../../../.."))
	tmp := t.TempDir()
	mgr := userconfig.NewManager(filepath.Join(tmp, "config.json"), filepath.Join(tmp, "schema.json"))
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}
	executor := &mutableSupervisionExecutor{}
	executor.set(supervisionReport(supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentMustStart)), nil)
	demandFixture := &coreDemandFixture{}
	controller := NewSupervisionController(
		checks.NewRegistry(&platform.Capabilities{Platform: platform.Linux}),
		mgr,
		NewSupervisionSource(executor),
		demandFixture,
	)
	if _, err := controller.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	executor.set(coreset.Report{}, errors.New("control plane unavailable"))
	snapshot, err := controller.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Degraded {
		t.Fatal("expected last-known-good supervision snapshot")
	}
	if len(demandFixture.releases) != 0 {
		t.Fatalf("degraded refresh released core holds: %+v", demandFixture.releases)
	}
	if len(demandFixture.renews) != 1 {
		t.Fatalf("degraded refresh should renew retained core hold, renews=%v", demandFixture.renews)
	}
}

func TestSupervisionSourceCheckReportsDemandLeaseFailure(t *testing.T) {
	t.Setenv("VROOLI_ROOT", filepath.Clean("../../../../.."))
	tmp := t.TempDir()
	mgr := userconfig.NewManager(filepath.Join(tmp, "config.json"), filepath.Join(tmp, "schema.json"))
	if err := mgr.Load(); err != nil {
		t.Fatal(err)
	}
	executor := &mutableSupervisionExecutor{}
	executor.set(supervisionReport(supervisedMember("search-hub", coreset.MemberKindScenario, coreset.IntentMustStart)), nil)
	demandFixture := &coreDemandFixture{err: errors.New("runtime unavailable")}
	controller := NewSupervisionController(
		checks.NewRegistry(&platform.Capabilities{Platform: platform.Linux}),
		mgr,
		NewSupervisionSource(executor),
		demandFixture,
	)
	result := (&supervisionSourceCheck{controller: controller}).Run(context.Background())
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %s, want warning: %+v", result.Status, result)
	}
	if result.Details["demandLeaseErrors"] == nil {
		t.Fatalf("demand lease failure was not exposed: %+v", result.Details)
	}
}
