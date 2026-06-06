// Package safety is the Baseline Modes data substrate: the orchestration the
// platform recovery floor shells out to for pre-promote safety snapshots. It
// owns no storage of its own — every verb composes the existing
// destinations / plans / runs domains through the narrow seams below.
//
// Each seam is owned here (the consumer) and satisfied by a thin adapter over a
// sibling service, wired in main.go. Keeping them as safety-local interfaces
// means the orchestration is unit-testable against fakes without standing up
// the whole domain tree, and avoids importing the sibling domain packages (no
// import cycles) — the same pattern restores/runs already use.
package safety

import (
	"context"

	"data-backup-manager/internal/sources"
)

const (
	// SafetyDestinationName is the reserved name of the auto-provisioned Baseline
	// Modes safety destination. It is kept distinct from every operator-configured
	// destination and is used only for pre-promote safety snapshots.
	SafetyDestinationName = "baseline-safety"

	// ephemeralPlanPrefix names the per-scenario ephemeral plan BackupScenarioNow
	// builds-or-reuses (one plan per scenario, reused across calls). The plan is
	// manual-only (empty schedule) so the scheduler never fires it.
	ephemeralPlanPrefix = "baseline-safety-"
)

// DestinationRef is the slice of a destination the safety orchestration needs.
type DestinationRef struct {
	ID                 string
	Name               string
	Location           string
	RepositoryLocation string
}

// TargetRef is the slice of a target the safety orchestration needs.
type TargetRef struct {
	ID    string
	Owner string
	Name  string
}

// PlanRef is the slice of a plan the safety orchestration needs.
type PlanRef struct {
	ID   string
	Name string
}

// RunRef is the slice of a run BackupScenarioNow returns.
type RunRef struct {
	ID     string
	PlanID string
	Status string
}

// Destinations is the destinations-domain seam: list existing destinations and
// create the reserved safety destination. Implemented by an adapter over
// destinations.Service in main.go.
type Destinations interface {
	// List returns all destinations (the safety verb only needs id/name/location).
	List(ctx context.Context) ([]DestinationRef, error)
	// CreateSafety creates the reserved filesystem safety destination at location
	// with the given cap (0 = no cap), alert+block cap policy.
	CreateSafety(ctx context.Context, name, location string, capBytes int64) (DestinationRef, error)
}

// Targets is the targets-domain seam: list and register a scenario's targets
// (targets carry owner=scenario). Implemented by an adapter over targets.Service
// in main.go.
type Targets interface {
	ListByOwner(ctx context.Context, owner string) ([]TargetRef, error)
	// Register idempotently upserts a target keyed by (owner, name). An identical
	// re-register is a no-op in the targets domain.
	Register(ctx context.Context, owner, name string, kind sources.SourceKind, locator string) error
}

// ScenarioInspector is the seam that resolves a scenario's derivable storage
// facts from its .vrooli/service.json + on-disk layout. Implemented by an
// adapter over internal/scenariospec in main.go; stubbed in tests.
type ScenarioInspector interface {
	Inspect(ctx context.Context, scenario string) (ScenarioFacts, error)
}

// ScenarioFacts are the storage facts RegisterScenarioTargets derives targets
// from. Mirrors scenariospec.Facts across the seam so the orchestration is
// unit-testable without the filesystem.
type ScenarioFacts struct {
	// UsesPostgres is true when the scenario declares an enabled Postgres
	// resource — its lifecycle DB (vrooli_<scenario>) is a target.
	UsesPostgres bool
	// DataDir is the conventional absolute durable-data directory.
	DataDir string
	// DataDirPresent is true when DataDir exists on disk and is non-empty.
	DataDirPresent bool
}

// Plans is the plans-domain seam: build-or-reuse the ephemeral per-scenario
// plan. Implemented by an adapter over plans.Service in main.go.
type Plans interface {
	List(ctx context.Context) ([]PlanRef, error)
	Create(ctx context.Context, name string, targetIDs, destinationIDs []string, keepLatest int32) (PlanRef, error)
	Update(ctx context.Context, id, name string, targetIDs, destinationIDs []string, keepLatest int32) (PlanRef, error)
}

// Runs is the runs-domain seam: trigger a manual run. Implemented by an adapter
// over runs.Service in main.go.
type Runs interface {
	TriggerManual(ctx context.Context, planID string) (RunRef, error)
}

// RuntimeRootFunc resolves the Vrooli runtime root (~/.vrooli) under which the
// safety destination is placed. Injected so tests can stub it; satisfied by
// discovery.RuntimeRoot in main.go.
type RuntimeRootFunc func() string
