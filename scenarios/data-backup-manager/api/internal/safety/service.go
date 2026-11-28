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

// ErrNoSafetyBackup is returned by PopulateShadow when there is no completed
// safety backup to restore from — the safety destination is missing, or the
// scenario's ephemeral safety plan has no terminal run yet. The caller runs
// `safety backup-now` first.
var ErrNoSafetyBackup = errors.New("no completed safety backup for scenario")

// ErrRunNotTerminal is returned by PopulateShadow when an explicitly-requested
// run has not finished yet — its snapshots are not safe to restore from.
var ErrRunNotTerminal = errors.New("safety run is not finished")

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

	// PopulateShadow restores a scenario's already-captured safety snapshots into
	// the caller-chosen shadow namespaces (the data half of `baseline start` in
	// shadow mode). It resolves the safety run (the given runID, or the latest
	// terminal run of the ephemeral safety plan when empty), and for each mapping
	// restores the target's latest successful snapshot into its shadow location.
	// The restores run asynchronously; poll the restores domain for each. A
	// mapping with no registered target or no successful snapshot is reported in
	// Skipped rather than failing the whole call. ErrNoSafetyBackup when there is
	// nothing to restore from; ErrRunNotTerminal when an explicit run is unfinished.
	PopulateShadow(ctx context.Context, scenario, runID string, mappings []ShadowMapping) (PopulateShadowResult, error)
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

// ShadowMapping maps one registered target to the fresh shadow namespace its
// latest safety snapshot is restored into. The caller owns the namespace's
// uniqueness + teardown (the restore overwrites it).
type ShadowMapping struct {
	TargetName string
	Location   string
}

// PopulateShadowResult is the outcome of a PopulateShadow call: the restores
// enqueued (one per populated target, each non-terminal — poll the restores
// domain) and the mappings skipped with the reason.
type PopulateShadowResult struct {
	Scenario string
	RunID    string
	Restores []ShadowRestoreRef
	Skipped  []ShadowSkip
}

// ShadowRestoreRef is one enqueued restore of a target's snapshot into its shadow.
type ShadowRestoreRef struct {
	TargetName string
	TargetID   string
	SnapshotID string
	RestoreID  string
	Location   string
	Status     string
}

// ShadowSkip records a mapping that was not populated, with the reason.
type ShadowSkip struct {
	TargetName string
	Reason     string
}

// Deps are the seams the service composes, injected from main.go.
type Deps struct {
	Destinations Destinations
	Targets      Targets
	Plans        Plans
	Runs         Runs
	Restores     Restores
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

func (s *service) PopulateShadow(ctx context.Context, scenario, runID string, mappings []ShadowMapping) (PopulateShadowResult, error) {
	scenario = strings.TrimSpace(scenario)
	runID = strings.TrimSpace(runID)
	if scenario == "" {
		return PopulateShadowResult{}, errors.New("scenario is required")
	}
	if len(mappings) == 0 {
		return PopulateShadowResult{}, errors.New("at least one target mapping is required")
	}

	// The snapshots live in the reserved safety destination. It must already
	// exist — populate-shadow restores from a prior `safety backup-now`, it never
	// creates the backup.
	safetyDest, err := s.findSafetyDestination(ctx)
	if err != nil {
		return PopulateShadowResult{}, err
	}

	// Map registered target name -> id so a mapping addresses a target by its
	// stable name (e.g. "postgres", "data") rather than a generated id.
	targets, err := s.deps.Targets.ListByOwner(ctx, scenario)
	if err != nil {
		return PopulateShadowResult{}, fmt.Errorf("list targets for %q: %w", scenario, err)
	}
	targetByName := make(map[string]TargetRef, len(targets))
	for _, t := range targets {
		targetByName[t.Name] = t
	}

	run, err := s.resolveSafetyRun(ctx, scenario, runID)
	if err != nil {
		return PopulateShadowResult{}, err
	}

	// Index the run's successful snapshots in the safety destination by target.
	snapshotByTarget := make(map[string]string, len(run.Outcomes))
	for _, o := range run.Outcomes {
		if o.DestinationID == safetyDest.ID && o.Succeeded && o.SnapshotID != "" {
			snapshotByTarget[o.TargetID] = o.SnapshotID
		}
	}

	res := PopulateShadowResult{Scenario: scenario, RunID: run.ID}
	for _, m := range mappings {
		name := strings.TrimSpace(m.TargetName)
		location := strings.TrimSpace(m.Location)
		if name == "" || location == "" {
			res.Skipped = append(res.Skipped, ShadowSkip{TargetName: name, Reason: "mapping requires both target_name and location"})
			continue
		}
		target, ok := targetByName[name]
		if !ok {
			res.Skipped = append(res.Skipped, ShadowSkip{TargetName: name, Reason: fmt.Sprintf("no target named %q registered under %q", name, scenario)})
			continue
		}
		snapshotID, ok := snapshotByTarget[target.ID]
		if !ok {
			res.Skipped = append(res.Skipped, ShadowSkip{TargetName: name, Reason: fmt.Sprintf("target %q has no successful snapshot in safety run %s", name, run.ID)})
			continue
		}
		// The restore overwrites its destination, so the caller must hand us a
		// fresh shadow namespace; a non-empty existing filesystem dir is refused
		// by the restores domain and surfaces here as a per-mapping skip (the
		// other mappings still proceed).
		restore, err := s.deps.Restores.RestoreTarget(ctx, target.ID, safetyDest.ID, snapshotID, location)
		if err != nil {
			res.Skipped = append(res.Skipped, ShadowSkip{TargetName: name, Reason: fmt.Sprintf("restore into %q failed: %v", location, err)})
			continue
		}
		res.Restores = append(res.Restores, ShadowRestoreRef{
			TargetName: name,
			TargetID:   target.ID,
			SnapshotID: snapshotID,
			RestoreID:  restore.ID,
			Location:   location,
			Status:     restore.Status,
		})
	}
	return res, nil
}

// findSafetyDestination returns the reserved baseline-safety destination, or
// ErrNoSafetyBackup when it has never been provisioned (so no snapshot exists).
func (s *service) findSafetyDestination(ctx context.Context) (DestinationRef, error) {
	existing, err := s.deps.Destinations.List(ctx)
	if err != nil {
		return DestinationRef{}, fmt.Errorf("list destinations: %w", err)
	}
	for _, d := range existing {
		if d.Name == SafetyDestinationName {
			return d, nil
		}
	}
	return DestinationRef{}, fmt.Errorf("%w: safety destination not provisioned; run `safety backup-now` first", ErrNoSafetyBackup)
}

// resolveSafetyRun resolves the safety run whose snapshots populate the shadow.
// An explicit runID must be terminal; an empty runID resolves the latest
// terminal run of the scenario's ephemeral safety plan.
func (s *service) resolveSafetyRun(ctx context.Context, scenario, runID string) (RunDetail, error) {
	if runID != "" {
		run, err := s.deps.Runs.GetRun(ctx, runID)
		if err != nil {
			return RunDetail{}, fmt.Errorf("get safety run %q: %w", runID, err)
		}
		if !run.Terminal {
			return RunDetail{}, fmt.Errorf("%w: run %s is %s", ErrRunNotTerminal, runID, run.Status)
		}
		return run, nil
	}

	planName := ephemeralPlanPrefix + scenario
	planID, err := s.findPlanID(ctx, planName)
	if err != nil {
		return RunDetail{}, err
	}
	run, ok, err := s.deps.Runs.LatestTerminalRun(ctx, planID)
	if err != nil {
		return RunDetail{}, fmt.Errorf("resolve latest safety run: %w", err)
	}
	if !ok {
		return RunDetail{}, fmt.Errorf("%w: ephemeral safety plan has no finished run; run `safety backup-now --scenario %s` first", ErrNoSafetyBackup, scenario)
	}
	return run, nil
}

// findPlanID resolves the ephemeral safety plan id by name, or ErrNoSafetyBackup
// when it was never built (so no backup has run).
func (s *service) findPlanID(ctx context.Context, name string) (string, error) {
	plans, err := s.deps.Plans.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list plans: %w", err)
	}
	for _, p := range plans {
		if p.Name == name {
			return p.ID, nil
		}
	}
	return "", fmt.Errorf("%w: no ephemeral safety plan %q", ErrNoSafetyBackup, name)
}
