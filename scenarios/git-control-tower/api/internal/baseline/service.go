package baseline

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/git"
)

// PinOwner returns the canonical test-genie pin owner for a baseline. Deleting
// the baseline unpins exactly this owner, releasing the run to normal GC
// (Decision 3).
func PinOwner(name string) string { return "gct:baseline:" + name }

// Service orchestrates baseline capture/diff/delete across surface adapters. It
// owns no surface logic — adapters do (Decision 3). test-genie owns run
// history; GCT owns baselines.
type Service struct {
	storage    *Storage
	snaps      *SnapshotStore
	adapters   map[string]SurfaceAdapter
	probe      StalenessProbe
	runs       RunsClient
	captureGit func(ctx context.Context, repoDir string) (git.State, error)
	now        func() time.Time
}

// Deps wires the Service. Adapters is keyed by surface ID.
type Deps struct {
	Storage    *Storage
	Snaps      *SnapshotStore
	Adapters   map[string]SurfaceAdapter
	Probe      StalenessProbe
	Runs       RunsClient
	CaptureGit func(ctx context.Context, repoDir string) (git.State, error)
	Now        func() time.Time
}

// NewService builds a Service, defaulting CaptureGit to the real git reader and
// Now to time.Now.
func NewService(d Deps) *Service {
	if d.CaptureGit == nil {
		d.CaptureGit = git.Capture
	}
	if d.Now == nil {
		d.Now = time.Now
	}
	if d.Adapters == nil {
		d.Adapters = map[string]SurfaceAdapter{}
	}
	return &Service{
		storage:    d.Storage,
		snaps:      d.Snaps,
		adapters:   d.Adapters,
		probe:      d.Probe,
		runs:       d.Runs,
		captureGit: d.CaptureGit,
		now:        d.Now,
	}
}

// CreateRequest captures (or assembles) a baseline.
type CreateRequest struct {
	RepoID    int64
	RepoDir   string
	Scenario  string
	Name      string
	Branch    string   // optional override; default derived from git state
	Include   []string // surface IDs; empty = all available
	Fast      bool
	Capture   bool // true = run adapters (snapshot); false = empty manifest (create)
	CreatedBy string
	Reason    string
}

// CreateResult is the outcome of Create.
type CreateResult struct {
	Manifest     BaselineManifest
	Skipped      map[string]string // surfaceID → why it was skipped
	DirtyWarning string            // non-empty when captured against a dirty tree
}

// Create captures a baseline. On Capture=false it writes an empty manifest
// (power-user/UI path). Surface capture is best-effort: unavailable or failing
// surfaces are recorded in Skipped, not fatal — except that a fully-empty
// capture (every requested surface skipped) is reported via Skipped for the
// caller to judge.
func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return CreateResult{}, fmt.Errorf("baseline name is required")
	}
	gitState, err := s.captureGit(ctx, req.RepoDir)
	if err != nil {
		return CreateResult{}, fmt.Errorf("read git state: %w", err)
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = ResolveStorageBranch(gitState)
	}

	// Fail fast if it already exists (Save also enforces this atomically).
	if _, err := s.storage.Load(req.RepoID, req.Scenario, branch, req.Name); err == nil {
		return CreateResult{}, ErrAlreadyExists
	}

	manifest := BaselineManifest{
		Name:          req.Name,
		Scenario:      req.Scenario,
		Branch:        branch,
		CreatedAt:     s.now().UTC(),
		CreatedBy:     req.CreatedBy,
		Git:           gitState,
		Surfaces:      map[string]SurfacePointer{},
		SchemaVersion: SchemaVersion,
	}

	skipped := map[string]string{}
	if req.Capture {
		target := Target{RepoID: req.RepoID, RepoDir: req.RepoDir, Scenario: req.Scenario}
		opts := CaptureOptions{Fast: req.Fast, PinnedBy: PinOwner(req.Name)}
		for _, id := range s.surfacesToCapture(ctx, target, req.Include) {
			adapter := s.adapters[id]
			if adapter == nil {
				skipped[id] = "no adapter registered"
				continue
			}
			ptr, capErr := adapter.Capture(ctx, target, opts)
			if capErr != nil {
				skipped[id] = capErr.Error()
				continue
			}
			manifest.Surfaces[id] = ptr
		}
	}

	if err := s.storage.Save(req.RepoID, manifest, CreateOnly); err != nil {
		// Roll back any pins we created so a failed create leaves no orphans.
		s.unpinSurfaces(ctx, manifest)
		return CreateResult{}, err
	}

	res := CreateResult{Manifest: manifest, Skipped: skipped}
	if gitState.Dirty {
		res.DirtyWarning = fmt.Sprintf("baseline captured against dirty tree (%s) — comparisons may be muddled by uncommitted changes", gitState.DirtySummary)
	}
	return res, nil
}

// surfacesToCapture resolves the requested surface set (or all) to those that
// report Available.
func (s *Service) surfacesToCapture(ctx context.Context, t Target, include []string) []string {
	want := include
	if len(want) == 0 {
		want = AllSurfaces
	}
	var out []string
	for _, id := range want {
		adapter := s.adapters[id]
		if adapter != nil && adapter.Available(ctx, t) {
			out = append(out, id)
		}
	}
	return out
}

// Get returns a single baseline manifest.
func (s *Service) Get(ctx context.Context, repoID int64, scenario, branch, name string) (BaselineManifest, error) {
	return s.storage.Load(repoID, scenario, branch, name)
}

// List returns baselines for a scenario; empty branch lists all branches.
func (s *Service) List(ctx context.Context, repoID int64, scenario, branch string) ([]BaselineManifest, error) {
	return s.storage.List(repoID, scenario, branch)
}

// DiffResult is the cross-surface comparison of a baseline against the current
// working tree.
type DiffResult struct {
	Manifest   BaselineManifest
	CurrentGit git.State
	Staleness  Staleness
	Surfaces   []SurfaceDiff
	Verdict    Verdict
}

// Diff compares a baseline against the current working tree. surface, when
// non-empty, restricts the diff to that one surface.
func (s *Service) Diff(ctx context.Context, repoID int64, repoDir, scenario, branch, name, surface string) (DiffResult, error) {
	manifest, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return DiffResult{}, err
	}
	cur, gerr := s.captureGit(ctx, repoDir)
	if gerr != nil {
		// Non-fatal: diff can still run; staleness is best-effort.
		cur = git.State{}
	}
	stale, _ := ComputeStaleness(ctx, s.probe, repoDir, manifest.Git.Sha)

	target := Target{RepoID: repoID, RepoDir: repoDir, Scenario: scenario}
	res := DiffResult{Manifest: manifest, CurrentGit: cur, Staleness: stale, Verdict: VerdictClean}

	ids := s.sortedSurfaceIDs(manifest)
	for _, id := range ids {
		if surface != "" && id != surface {
			continue
		}
		ptr := manifest.Surfaces[id]
		adapter := s.adapters[id]
		if adapter == nil {
			res.Surfaces = append(res.Surfaces, notComparable(id, "no adapter registered"))
			res.Verdict = WorseVerdict(res.Verdict, VerdictNotComparable)
			continue
		}
		d, derr := adapter.Diff(ctx, target, ptr)
		if derr != nil {
			d = notComparable(id, derr.Error())
		}
		res.Surfaces = append(res.Surfaces, d)
		res.Verdict = WorseVerdict(res.Verdict, d.Verdict)
	}
	if surface != "" && len(res.Surfaces) == 0 {
		return DiffResult{}, fmt.Errorf("surface %q not present in baseline %q", surface, name)
	}
	return res, nil
}

// Delete removes a baseline and releases the test-genie runs it pinned plus any
// GCT-local issue snapshots it owned.
func (s *Service) Delete(ctx context.Context, repoID int64, scenario, branch, name string) error {
	manifest, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return err
	}
	s.unpinSurfaces(ctx, manifest)
	s.deleteLocalSnapshots(repoID, manifest)
	return s.storage.Delete(repoID, scenario, branch, name)
}

// EditRequest swaps the pointer for one surface to a different test-genie run.
type EditRequest struct {
	RepoID   int64
	Scenario string
	Branch   string
	Name     string
	Surface  string
	PinRunID string
	Reason   string
}

// Edit re-points a surface at a different (pinned) test-genie run.
func (s *Service) Edit(ctx context.Context, req EditRequest) (BaselineManifest, error) {
	manifest, err := s.storage.Load(req.RepoID, req.Scenario, req.Branch, req.Name)
	if err != nil {
		return BaselineManifest{}, err
	}
	if manifest.Surfaces == nil {
		manifest.Surfaces = map[string]SurfacePointer{}
	}
	if s.runs != nil && req.PinRunID != "" {
		if err := s.runs.PinRun(ctx, req.Scenario, req.PinRunID, PinOwner(req.Name), req.Reason); err != nil {
			return BaselineManifest{}, fmt.Errorf("pin run %s: %w", req.PinRunID, err)
		}
	}
	manifest.Surfaces[req.Surface] = SurfacePointer{
		SurfaceID:  req.Surface,
		Kind:       KindTestGenieRun,
		Ref:        req.PinRunID,
		CapturedAt: s.now().UTC(),
	}
	if err := s.storage.Save(req.RepoID, manifest, Overwrite); err != nil {
		return BaselineManifest{}, err
	}
	return manifest, nil
}

// unpinSurfaces releases every test-genie run a manifest pinned.
func (s *Service) unpinSurfaces(ctx context.Context, m BaselineManifest) {
	if s.runs == nil {
		return
	}
	owner := PinOwner(m.Name)
	for _, ptr := range m.Surfaces {
		if ptr.Kind == KindTestGenieRun && ptr.Ref != "" {
			_ = s.runs.UnpinRun(ctx, m.Scenario, ptr.Ref, owner)
		}
	}
}

// deleteLocalSnapshots removes GCT-local issue snapshots (structure/rules).
// Visual snapshots live in VisualCaptureStorage under its own retention and are
// left alone.
func (s *Service) deleteLocalSnapshots(repoID int64, m BaselineManifest) {
	if s.snaps == nil {
		return
	}
	for id, ptr := range m.Surfaces {
		if ptr.Kind == KindGCTLocalSnapshot && (id == SurfaceStructure || id == SurfaceRules) {
			_ = s.snaps.Delete(repoID, m.Scenario, ptr.Ref)
		}
	}
}

func (s *Service) sortedSurfaceIDs(m BaselineManifest) []string {
	ids := make([]string, 0, len(m.Surfaces))
	for id := range m.Surfaces {
		ids = append(ids, id)
	}
	// Stable canonical order: AllSurfaces order first, then any extras.
	rank := map[string]int{}
	for i, id := range AllSurfaces {
		rank[id] = i
	}
	sort.SliceStable(ids, func(a, b int) bool {
		ra, oka := rank[ids[a]]
		rb, okb := rank[ids[b]]
		if oka && okb {
			return ra < rb
		}
		if oka != okb {
			return oka
		}
		return ids[a] < ids[b]
	})
	return ids
}
