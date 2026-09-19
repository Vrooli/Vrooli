package autofiler

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/settings"
)

type fakeAutoFilerSettings struct {
	cfg settings.AutoFilerSettings
	err error
}

func (p fakeAutoFilerSettings) LoadAutoFiler() (settings.AutoFilerSettings, error) {
	return p.cfg, p.err
}

type staticTargets struct {
	targets []Target
}

func (s staticTargets) Candidates(_ context.Context, _ int) ([]Target, error) {
	return append([]Target(nil), s.targets...), nil
}

type staticFindings struct {
	findings []Finding
	calls    int
}

func (s *staticFindings) Findings(_ context.Context, _ Target) ([]Finding, error) {
	s.calls++
	return append([]Finding(nil), s.findings...), nil
}

func TestSweeperDisabledNoops(t *testing.T) {
	source := &staticFindings{findings: []Finding{{ID: "gct:alpha:tests", Scenario: "alpha"}}}
	sw := NewSweeper(
		fakeAutoFilerSettings{cfg: settings.DefaultSettings().AutoFiler},
		fakeBacklogReader{},
		fakeTransitionCounter{},
		nil,
		source,
	)

	result, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if result.Enabled || source.calls != 0 {
		t.Fatalf("result=%+v source.calls=%d, want disabled no-op", result, source.calls)
	}
}

func TestSweeperFilesWithinPolicy(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	source := &staticFindings{findings: []Finding{{ID: "gct:alpha:tests", Scenario: "alpha", Dimension: "tests"}}}
	cfg := settings.DefaultSettings().AutoFiler
	cfg.Enabled = true
	cfg.MinVelocityTransitions = 0

	sw := NewSweeper(
		fakeAutoFilerSettings{cfg: cfg},
		store,
		fakeTransitionCounter{},
		NewFiler(svc, &fakeGoals{}),
		source,
	)
	sw.Feature = staticTargets{targets: []Target{{Scenario: "alpha"}}}

	result, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if result.Created != 1 || result.Findings != 1 || result.Candidates != 1 {
		t.Fatalf("result = %+v, want one created finding", result)
	}
	item, err := store.LoadItem(backlog.KindFix, "auto-filer-alpha-gct-alpha-tests")
	if err != nil {
		t.Fatalf("LoadItem: %v", err)
	}
	if item.Status != backlog.StatusSuggested {
		t.Fatalf("status = %q, want suggested", item.Status)
	}
}

func TestSweeperHonorsDismissalsAndBrake(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	source := &staticFindings{findings: []Finding{{ID: "gct:alpha:tests", Scenario: "alpha", Dimension: "tests"}}}
	cfg := settings.DefaultSettings().AutoFiler
	cfg.Enabled = true
	cfg.MinVelocityTransitions = 2

	sw := NewSweeper(
		fakeAutoFilerSettings{cfg: cfg},
		store,
		fakeTransitionCounter{},
		NewFiler(svc, &fakeGoals{}),
		source,
	)
	sw.Feature = staticTargets{targets: []Target{{Scenario: "alpha"}}}

	braked, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce braked: %v", err)
	}
	if !braked.Brake.Braked || braked.Created != 0 || source.calls != 1 {
		t.Fatalf("braked result=%+v source.calls=%d, want source read without filing", braked, source.calls)
	}

	cfg.MinVelocityTransitions = 0
	dismissals := NewDismissalStorePath(filepath.Join(t.TempDir(), "dismissed_findings.json"))
	if err := dismissals.Remember("gct:alpha:tests", "fix/old", "operator dismissed"); err != nil {
		t.Fatalf("Remember: %v", err)
	}
	sw.Settings = fakeAutoFilerSettings{cfg: cfg}
	sw.Dismissals = dismissals

	result, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce dismissed: %v", err)
	}
	if result.Created != 0 || result.SkippedDismissed != 1 || source.calls != 2 {
		t.Fatalf("result = %+v, want dismissed skip", result)
	}
}

func TestSweeperReconcilesResolvedFindingsEvenWhenBraked(t *testing.T) {
	store := backlog.NewFileStore(t.TempDir())
	svc, err := backlog.NewService(backlog.ServiceConfig{Store: store})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	for _, item := range []backlog.BacklogItem{
		autoFiledFindingItem("stale-suggested", backlog.StatusSuggested, "gct:alpha:stale"),
		autoFiledFindingItem("accepted", backlog.StatusBacklog, "gct:alpha:accepted"),
	} {
		if err := svc.Create(item, backlog.CreationContext{Source: backlog.SourceAutoFiler}); err != nil {
			t.Fatalf("seed %s: %v", item.Name, err)
		}
	}
	source := &staticFindings{findings: []Finding{{ID: "gct:alpha:active", Scenario: "alpha", Dimension: "tests"}}}
	cfg := settings.DefaultSettings().AutoFiler
	cfg.Enabled = true
	cfg.MinVelocityTransitions = 99

	sw := NewSweeper(
		fakeAutoFilerSettings{cfg: cfg},
		store,
		fakeTransitionCounter{},
		NewFiler(svc, &fakeGoals{}),
		source,
	)
	sw.Reconciler = svc
	sw.Feature = staticTargets{targets: []Target{{Scenario: "alpha"}}}

	result, err := sw.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if !result.Brake.Braked {
		t.Fatalf("result.Brake = %+v, want braked", result.Brake)
	}
	if result.Created != 0 || result.ReconciledClosed != 1 || result.ReconciledNoted != 1 {
		t.Fatalf("result = %+v, want closed=1 noted=1 created=0", result)
	}

	stale, err := store.LoadItem(backlog.KindFix, "stale-suggested")
	if err != nil {
		t.Fatalf("LoadItem stale: %v", err)
	}
	if stale.ArchivedAt == nil {
		t.Fatalf("stale suggestion was not archived")
	}
	if !strings.Contains(stale.Note, "no longer appears") {
		t.Fatalf("stale note = %q, want reconcile reason", stale.Note)
	}
	accepted, err := store.LoadItem(backlog.KindFix, "accepted")
	if err != nil {
		t.Fatalf("LoadItem accepted: %v", err)
	}
	if accepted.ArchivedAt != nil {
		t.Fatalf("accepted item should not be archived")
	}
	if !strings.Contains(accepted.Note, "already accepted") {
		t.Fatalf("accepted note = %q, want accepted annotation", accepted.Note)
	}
}

func autoFiledFindingItem(name string, status backlog.BacklogStatus, findingRef string) backlog.BacklogItem {
	return backlog.BacklogItem{
		Name:       name,
		Title:      "Auto-filed " + name,
		Kind:       backlog.KindFix,
		Status:     status,
		Priority:   4,
		Created:    "2026-07-09T00:00:00Z",
		Updated:    "2026-07-09T00:00:00Z",
		FindingRef: findingRef,
		CreatedBy:  &identity.Provenance{Actor: identity.TypeAgent, Source: Origin(StrategyFeaturePending, findingRef)},
	}
}
