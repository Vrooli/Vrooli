package safety

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"data-backup-manager/internal/sources"
)

// ErrNoTargets is returned by BackupScenarioNow when the scenario has no
// registered targets to back up. The caller registers the scenario's targets
// first (e.g. via `targets register`) before requesting a safety snapshot.
var ErrNoTargets = errors.New("scenario has no registered targets")

// Service is the application surface the safety handlers depend on. It owns the
// Baseline Modes substrate orchestration; persistence lives in the composed
// domains, never here.
type Service interface {
	// EnsureSafetyDestination idempotently provisions the reserved
	// baseline-safety filesystem destination and returns it. created is true only
	// when this call created it.
	EnsureSafetyDestination(ctx context.Context, capBytes int64) (dest DestinationRef, created bool, err error)

	// BackupScenarioNow backs up every target registered under owner=scenario to
	// the safety destination now, by building-or-reusing an ephemeral per-scenario
	// plan and triggering a run. ErrNoTargets when the scenario has none.
	BackupScenarioNow(ctx context.Context, scenario string, keepLatest int32) (BackupResult, error)

	// RegisterScenarioTargets derives a scenario's reliably-conventional backup
	// targets (Postgres DB name + filesystem data dir) from its service.json and
	// on-disk layout, and idempotently registers them under owner=scenario — so
	// BackupScenarioNow works without a hand-run `targets register`. Kinds that
	// aren't derivable (Redis/Qdrant/SQLite) are returned in Skipped.
	RegisterScenarioTargets(ctx context.Context, scenario string) (RegisterTargetsResult, error)
}

// BackupResult is the outcome of a BackupScenarioNow call. The run executes
// asynchronously — poll RunsService.GetRun with Run.ID for the terminal status.
type BackupResult struct {
	Run           RunRef
	DestinationID string
	TargetCount   int
}

// RegisterTargetsResult is the outcome of a RegisterScenarioTargets call: the
// targets registered (or refreshed) and the kinds skipped with the reason.
type RegisterTargetsResult struct {
	Scenario   string
	Registered []ScenarioTargetSpec
	Skipped    []SkippedNote
}

// ScenarioTargetSpec is one derived, registered target.
type ScenarioTargetSpec struct {
	Name    string
	Kind    sources.SourceKind
	Locator string
}

// SkippedNote records a kind that was considered but not registered.
type SkippedNote struct {
	Kind   string
	Reason string
}

// Deps are the seams the service composes, injected from main.go.
type Deps struct {
	Destinations Destinations
	Targets      Targets
	Plans        Plans
	Runs         Runs
	Inspector    ScenarioInspector
	RuntimeRoot  RuntimeRootFunc
}

type service struct {
	deps Deps
}

// NewService returns the safety orchestration service.
func NewService(d Deps) Service {
	return &service{deps: d}
}

func (s *service) EnsureSafetyDestination(ctx context.Context, capBytes int64) (DestinationRef, bool, error) {
	existing, err := s.deps.Destinations.List(ctx)
	if err != nil {
		return DestinationRef{}, false, fmt.Errorf("list destinations: %w", err)
	}
	for _, d := range existing {
		if d.Name == SafetyDestinationName {
			// Idempotent: the safety destination already exists; cap is applied
			// only at creation time, so an existing destination is returned as-is.
			return d, false, nil
		}
	}
	location, err := s.safetyLocation()
	if err != nil {
		return DestinationRef{}, false, err
	}
	created, err := s.deps.Destinations.CreateSafety(ctx, SafetyDestinationName, location, capBytes)
	if err != nil {
		return DestinationRef{}, false, fmt.Errorf("create safety destination: %w", err)
	}
	return created, true, nil
}

// safetyLocation resolves the safety destination's bundle root to a sibling of
// the protected data dir under the Vrooli runtime root, so it satisfies the
// destinations separate-root rule (a filesystem destination must not point
// inside the storage root it protects).
func (s *service) safetyLocation() (string, error) {
	root := ""
	if s.deps.RuntimeRoot != nil {
		root = strings.TrimSpace(s.deps.RuntimeRoot())
	}
	if root == "" {
		return "", errors.New("cannot resolve Vrooli runtime root for the safety destination")
	}
	return filepath.Join(root, SafetyDestinationName), nil
}

// Target names RegisterScenarioTargets registers under owner=scenario. Kept
// stable so a re-run upserts the same rows rather than creating duplicates.
const (
	postgresTargetName = "postgres"
	dataDirTargetName  = "data"
	nonDerivableKinds  = "redis, qdrant, sqlite"
	nonDerivableReason = "not derivable from service.json (per-scenario prefix/collection/path); register explicitly via `targets register`"
	postgresSkipReason = "scenario does not declare an enabled Postgres resource"
	dataDirSkipReason  = "no durable data found at the conventional data directory"
	postgresLocatorPfx = "vrooli_"
)

func (s *service) RegisterScenarioTargets(ctx context.Context, scenario string) (RegisterTargetsResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return RegisterTargetsResult{}, errors.New("scenario is required")
	}

	facts, err := s.deps.Inspector.Inspect(ctx, scenario)
	if err != nil {
		return RegisterTargetsResult{}, fmt.Errorf("inspect scenario %q: %w", scenario, err)
	}

	res := RegisterTargetsResult{Scenario: scenario}

	// Postgres — the universal, override-free lifecycle convention: a scenario's
	// database is always named vrooli_<scenario>. Registered when the scenario
	// declares an enabled Postgres resource.
	if facts.UsesPostgres {
		spec := ScenarioTargetSpec{Name: postgresTargetName, Kind: sources.KindPostgres, Locator: postgresLocatorPfx + scenario}
		if err := s.deps.Targets.Register(ctx, scenario, spec.Name, spec.Kind, spec.Locator); err != nil {
			return RegisterTargetsResult{}, fmt.Errorf("register postgres target: %w", err)
		}
		res.Registered = append(res.Registered, spec)
	} else {
		res.Skipped = append(res.Skipped, SkippedNote{Kind: string(sources.KindPostgres), Reason: postgresSkipReason})
	}

	// Filesystem durable-data directory — registered only when it exists on disk
	// and is non-empty (self-validating: confirms the conventional layout holds
	// and there is genuinely state worth snapshotting), so we never register a
	// target whose later backup would fail on a missing path.
	if facts.DataDirPresent {
		spec := ScenarioTargetSpec{Name: dataDirTargetName, Kind: sources.KindFilesystem, Locator: facts.DataDir}
		if err := s.deps.Targets.Register(ctx, scenario, spec.Name, spec.Kind, spec.Locator); err != nil {
			return RegisterTargetsResult{}, fmt.Errorf("register data dir target: %w", err)
		}
		res.Registered = append(res.Registered, spec)
	} else {
		res.Skipped = append(res.Skipped, SkippedNote{Kind: string(sources.KindFilesystem), Reason: dataDirSkipReason})
	}

	// Redis/Qdrant/SQLite carry per-scenario prefixes/collections/paths that are
	// not reliably derivable from service.json — the scenario self-registers them.
	res.Skipped = append(res.Skipped, SkippedNote{Kind: nonDerivableKinds, Reason: nonDerivableReason})

	return res, nil
}

func (s *service) BackupScenarioNow(ctx context.Context, scenario string, keepLatest int32) (BackupResult, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return BackupResult{}, errors.New("scenario is required")
	}

	targets, err := s.deps.Targets.ListByOwner(ctx, scenario)
	if err != nil {
		return BackupResult{}, fmt.Errorf("list targets for %q: %w", scenario, err)
	}
	if len(targets) == 0 {
		return BackupResult{}, fmt.Errorf("%w: %s", ErrNoTargets, scenario)
	}

	dest, _, err := s.EnsureSafetyDestination(ctx, 0)
	if err != nil {
		return BackupResult{}, err
	}

	targetIDs := make([]string, len(targets))
	for i, t := range targets {
		targetIDs[i] = t.ID
	}

	planName := ephemeralPlanPrefix + scenario
	plan, err := s.ensureEphemeralPlan(ctx, planName, targetIDs, []string{dest.ID}, keepLatest)
	if err != nil {
		return BackupResult{}, err
	}

	run, err := s.deps.Runs.TriggerManual(ctx, plan.ID)
	if err != nil {
		return BackupResult{}, fmt.Errorf("trigger run: %w", err)
	}
	return BackupResult{Run: run, DestinationID: dest.ID, TargetCount: len(targets)}, nil
}

// ensureEphemeralPlan builds-or-reuses the named per-scenario ephemeral plan,
// refreshing its membership to the current target set so a newly-registered
// target is picked up on the next backup. The plan is manual-only (empty
// schedule); the scheduler never fires it.
func (s *service) ensureEphemeralPlan(ctx context.Context, name string, targetIDs, destIDs []string, keepLatest int32) (PlanRef, error) {
	plans, err := s.deps.Plans.List(ctx)
	if err != nil {
		return PlanRef{}, fmt.Errorf("list plans: %w", err)
	}
	for _, p := range plans {
		if p.Name == name {
			updated, err := s.deps.Plans.Update(ctx, p.ID, name, targetIDs, destIDs, keepLatest)
			if err != nil {
				return PlanRef{}, fmt.Errorf("refresh ephemeral plan: %w", err)
			}
			return updated, nil
		}
	}
	created, err := s.deps.Plans.Create(ctx, name, targetIDs, destIDs, keepLatest)
	if err != nil {
		return PlanRef{}, fmt.Errorf("create ephemeral plan: %w", err)
	}
	return created, nil
}
