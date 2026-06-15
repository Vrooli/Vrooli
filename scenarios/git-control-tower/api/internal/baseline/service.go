package baseline

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"git-control-tower/internal/git"
)

// PinOwner returns the canonical test-genie pin owner for a baseline. Deleting
// the baseline unpins exactly this owner, releasing the run to normal GC.
func PinOwner(name string) string { return "gct:baseline:" + name }

// Service orchestrates baseline capture/diff/delete. A baseline is ONE
// comprehensive, durable test-genie run pinned once; each surface is a
// phase-set view over that single run (option-c). The service owns no run
// history — test-genie does; GCT owns the baseline manifest of pointers.
type Service struct {
	storage    *Storage
	exec       Executor
	runs       RunsClient
	probe      StalenessProbe
	reachable  Reachability
	captureGit func(ctx context.Context, repoDir string) (git.State, error)
	now        func() time.Time
}

// Deps wires the Service.
type Deps struct {
	Storage    *Storage
	Exec       Executor
	Runs       RunsClient
	Probe      StalenessProbe
	Reachable  Reachability
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
	return &Service{
		storage:    d.Storage,
		exec:       d.Exec,
		runs:       d.Runs,
		probe:      d.Probe,
		reachable:  d.Reachable,
		captureGit: d.CaptureGit,
		now:        d.Now,
	}
}

// reachabilityError returns a non-empty skip reason when the test-genie backend
// is not usable for a comprehensive run: the seams aren't wired, or a bounded
// reachability probe fails fast. An empty string means "proceed". Centralizing
// this here is what turns an unreachable backend from a multi-minute silent
// hang into an immediate, clearly-explained skip (the reported bug).
func (s *Service) reachabilityError(ctx context.Context) string {
	if s.exec == nil || s.runs == nil {
		return "test-genie unavailable at capture time (owning subsystem not wired)"
	}
	if s.reachable != nil {
		if err := s.reachable.Probe(ctx); err != nil {
			return "test-genie unreachable (fast-skip, not blocked): " + err.Error()
		}
	}
	return ""
}

// CreateRequest captures (or assembles) a baseline.
type CreateRequest struct {
	RepoID    int64
	RepoDir   string
	Scenario  string
	Name      string
	Branch    string   // optional override; default derived from git state
	Include   []string // surface IDs to view; empty = all surfaces
	Fast      bool
	Capture   bool // true = run the comprehensive suite (snapshot); false = empty manifest (create)
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
// (power-user/UI path). On Capture=true it triggers ONE comprehensive test-genie
// run with the baseline capture profile, pins it once, and points every
// requested surface at that single run — surfaces are views, not separate runs.
// If the comprehensive run cannot be triggered every requested surface is
// recorded in Skipped (so a partial baseline never masquerades as complete).
func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	manifest, err := s.buildManifestSkeleton(ctx, req)
	if err != nil {
		return CreateResult{}, err
	}

	want := req.Include
	if len(want) == 0 {
		want = AllSurfaces
	}

	skipped := map[string]string{}
	if req.Capture {
		// Synchronous capture (start → await → fill). The durable, return-fast
		// path is StartCapture/FinalizeCapture; Create stays the simple blocking
		// form used by CreateBaseline (Capture=false) and tests.
		if h, startErr := s.startCaptureRun(ctx, req.Scenario, want, skipped); startErr == nil {
			res, awaitErr := s.exec.AwaitResult(ctx, req.Scenario, h.RunID)
			if awaitErr != nil {
				for _, id := range want {
					skipped[id] = "comprehensive run failed: " + awaitErr.Error()
				}
			} else {
				s.fillSurfaces(ctx, &manifest, req, want, skipped, res)
			}
		}
	}
	if len(skipped) > 0 {
		manifest.Skipped = skipped
	}

	if err := s.storage.Save(req.RepoID, manifest, CreateOnly); err != nil {
		s.unpinRun(ctx, manifest)
		return CreateResult{}, err
	}

	res := CreateResult{Manifest: manifest, Skipped: skipped}
	if manifest.Git.Dirty {
		res.DirtyWarning = dirtyCaptureWarning(manifest.Git.DirtySummary)
	}
	return res, nil
}

// PendingCapture is the handle StartCapture hands to FinalizeCapture: everything
// needed to pin the run and write the manifest once the durable run completes.
type PendingCapture struct {
	Manifest     BaselineManifest
	Req          CreateRequest
	Want         []string
	Run          RunHandle
	DirtyWarning string
}

// StartCapture validates the request, captures git state, starts ONE
// comprehensive durable run, and returns its handle WITHOUT waiting for it to
// finish. The caller (the Connect handler) returns the handle to the client
// immediately and hands the PendingCapture to FinalizeCapture on a server-owned
// context so the pin + manifest survive client disconnect.
func (s *Service) StartCapture(ctx context.Context, req CreateRequest) (PendingCapture, error) {
	manifest, err := s.buildManifestSkeleton(ctx, req)
	if err != nil {
		return PendingCapture{}, err
	}

	want := req.Include
	if len(want) == 0 {
		want = AllSurfaces
	}

	if reason := s.reachabilityError(ctx); reason != "" {
		return PendingCapture{}, fmt.Errorf("test-genie is not reachable: %s", reason)
	}
	h, err := s.exec.StartRun(ctx, req.Scenario)
	if err != nil {
		return PendingCapture{}, fmt.Errorf("start comprehensive run: %w", err)
	}
	if h.RunID == "" {
		return PendingCapture{}, fmt.Errorf("test-genie returned no run id")
	}

	pending := PendingCapture{Manifest: manifest, Req: req, Want: want, Run: h}
	if manifest.Git.Dirty {
		pending.DirtyWarning = dirtyCaptureWarning(manifest.Git.DirtySummary)
	}
	return pending, nil
}

// FinalizeCapture blocks until the started run is terminal, pins it once, fills
// the surface pointers, and writes the manifest. It runs on a server-owned
// context so a disconnected client never abandons a half-pinned baseline.
func (s *Service) FinalizeCapture(ctx context.Context, pending PendingCapture) (CreateResult, error) {
	manifest := pending.Manifest
	skipped := map[string]string{}

	res, err := s.exec.AwaitResult(ctx, pending.Req.Scenario, pending.Run.RunID)
	if err != nil {
		for _, id := range pending.Want {
			skipped[id] = "comprehensive run failed: " + err.Error()
		}
	} else {
		s.fillSurfaces(ctx, &manifest, pending.Req, pending.Want, skipped, res)
	}
	if len(skipped) > 0 {
		manifest.Skipped = skipped
	}
	if err := s.storage.Save(pending.Req.RepoID, manifest, CreateOnly); err != nil {
		s.unpinRun(ctx, manifest)
		return CreateResult{}, err
	}
	out := CreateResult{Manifest: manifest, Skipped: skipped, DirtyWarning: pending.DirtyWarning}
	return out, nil
}

// buildManifestSkeleton validates the request, reads git state, resolves the
// branch, and fails fast if the baseline already exists. It returns the
// not-yet-captured manifest skeleton (its resolved branch is manifest.Branch).
func (s *Service) buildManifestSkeleton(ctx context.Context, req CreateRequest) (BaselineManifest, error) {
	if strings.TrimSpace(req.Name) == "" {
		return BaselineManifest{}, fmt.Errorf("baseline name is required")
	}
	gitState, err := s.captureGit(ctx, req.RepoDir)
	if err != nil {
		return BaselineManifest{}, fmt.Errorf("read git state: %w", err)
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = ResolveStorageBranch(gitState)
	}
	if _, err := s.storage.Load(req.RepoID, req.Scenario, branch, req.Name); err == nil {
		return BaselineManifest{}, ErrAlreadyExists
	}
	return BaselineManifest{
		Name:          req.Name,
		Scenario:      req.Scenario,
		Branch:        branch,
		CreatedAt:     s.now().UTC(),
		CreatedBy:     req.CreatedBy,
		Git:           gitState,
		Surfaces:      map[string]SurfacePointer{},
		SchemaVersion: SchemaVersion,
	}, nil
}

// startCaptureRun checks reachability and starts the run for the synchronous
// Create path, recording a skip reason per surface on failure.
func (s *Service) startCaptureRun(ctx context.Context, scenario string, want []string, skipped map[string]string) (RunHandle, error) {
	if reason := s.reachabilityError(ctx); reason != "" {
		for _, id := range want {
			skipped[id] = reason
		}
		return RunHandle{}, fmt.Errorf("unreachable")
	}
	h, err := s.exec.StartRun(ctx, scenario)
	if err != nil || h.RunID == "" {
		reason := "test-genie returned no run id"
		if err != nil {
			reason = "comprehensive run failed: " + err.Error()
		}
		for _, id := range want {
			skipped[id] = reason
		}
		return RunHandle{}, fmt.Errorf("start run")
	}
	return h, nil
}

// fillSurfaces pins the shared run once and fills one pointer per requested
// surface (all referencing the run). On pin failure it records every surface as
// skipped.
func (s *Service) fillSurfaces(ctx context.Context, manifest *BaselineManifest, req CreateRequest, want []string, skipped map[string]string, res ExecResult) {
	if res.RunID == "" {
		for _, id := range want {
			skipped[id] = "test-genie returned no runID"
		}
		return
	}
	if err := s.runs.PinRun(ctx, req.Scenario, res.RunID, PinOwner(req.Name), "baseline:"+req.Name); err != nil {
		for _, id := range want {
			skipped[id] = "pin run " + res.RunID + ": " + err.Error()
		}
		return
	}

	now := s.now().UTC()
	phaseStatus := map[string]string{}
	for _, p := range res.Phases {
		phaseStatus[p.Name] = p.Status
	}

	for _, id := range want {
		switch id {
		case SurfaceVisuals:
			manifest.Surfaces[id] = s.visualPointer(ctx, req.Scenario, res.RunID, now)
		default:
			phases, ok := surfacePhases[id]
			if !ok {
				skipped[id] = "unknown surface"
				continue
			}
			manifest.Surfaces[id] = phaseSurfacePointer(id, res.RunID, phases, phaseStatus, now)
		}
	}
}

// runToCompletion starts a comprehensive run and blocks for its result. Used by
// Diff, which needs the current comparison run synchronously (the diff is an
// inline operation, not a durable pin).
func (s *Service) runToCompletion(ctx context.Context, scenario string) (ExecResult, error) {
	h, err := s.exec.StartRun(ctx, scenario)
	if err != nil {
		return ExecResult{}, err
	}
	return s.exec.AwaitResult(ctx, scenario, h.RunID)
}

// dirtyCaptureWarning is the standard warning when a baseline is captured
// against a dirty working tree.
func dirtyCaptureWarning(summary string) string {
	return fmt.Sprintf("baseline captured against dirty tree (%s) — comparisons may be muddled by uncommitted changes", summary)
}

// phaseSurfacePointer builds a surface's pointer (referencing the shared run)
// with a compact per-surface summary of its phases' terminal statuses.
func phaseSurfacePointer(surfaceID, runID string, phases []string, phaseStatus map[string]string, now time.Time) SurfacePointer {
	sum := surfaceSummary{RunID: runID}
	for _, p := range phases {
		st, ran := phaseStatus[p]
		if !ran {
			continue
		}
		sum.Phases = append(sum.Phases, PhaseStatus{Name: p, Status: st})
		switch st {
		case "passed":
			sum.Passed++
		case "failed":
			sum.Failed++
		}
	}
	raw, _ := json.Marshal(sum)
	return SurfacePointer{
		SurfaceID:  surfaceID,
		Kind:       KindTestGenieRun,
		Ref:        runID,
		CapturedAt: now,
		Summary:    raw,
	}
}

// visualPointer builds the visuals surface pointer: the same shared run, with a
// summary of its visual artifacts (page set + screenshot count) read from
// test-genie's ListRunVisuals.
func (s *Service) visualPointer(ctx context.Context, scenario, runID string, now time.Time) SurfacePointer {
	sum := visualSummary{RunID: runID}
	if vis, err := s.runs.ListRunVisuals(ctx, scenario, runID); err == nil {
		for _, v := range vis {
			sum.Pages = append(sum.Pages, v.Page)
			if v.ScreenshotRelPath != "" {
				sum.Screenshots++
			}
		}
		sort.Strings(sum.Pages)
	}
	raw, _ := json.Marshal(sum)
	return SurfacePointer{
		SurfaceID:  SurfaceVisuals,
		Kind:       KindTestGenieRun,
		Ref:        runID,
		CapturedAt: now,
		Summary:    raw,
	}
}

// surfaceSummary is the compact per-phase-surface summary stored on a pointer.
type surfaceSummary struct {
	RunID  string        `json:"run_id"`
	Passed int           `json:"passed"`
	Failed int           `json:"failed"`
	Phases []PhaseStatus `json:"phases"`
}

// visualSummary is the compact visuals-surface summary stored on a pointer.
type visualSummary struct {
	RunID       string   `json:"run_id"`
	Screenshots int      `json:"screenshots"`
	Pages       []string `json:"pages"`
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
	Manifest     BaselineManifest
	CurrentGit   git.State
	Staleness    Staleness
	Surfaces     []SurfaceDiff
	Verdict      Verdict
	DirtyWarning string // non-empty when the current tree is dirty (verdicts are most suspect then)
}

// Diff compares a baseline against the current working tree. It triggers ONE
// comprehensive run now, issues ONE empty-phase CompareRuns over (baseline run,
// current run), and buckets the returned PhaseDiff[] into surfaces locally
// (option-c). The visuals surface is diffed at the metadata level over the two
// runs' visual artifacts. surface, when non-empty, restricts the result to that
// one surface.
func (s *Service) Diff(ctx context.Context, repoID int64, repoDir, scenario, branch, name, surface string) (DiffResult, error) {
	manifest, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return DiffResult{}, err
	}
	cur, gerr := s.captureGit(ctx, repoDir)
	if gerr != nil {
		cur = git.State{}
	}
	stale, _ := ComputeStaleness(ctx, s.probe, repoDir, manifest.Git.Sha)

	res := DiffResult{Manifest: manifest, CurrentGit: cur, Staleness: stale, Verdict: VerdictClean}
	if cur.Dirty {
		res.DirtyWarning = fmt.Sprintf("working tree is dirty (%s) — failures may be caused by uncommitted changes rather than the diff itself", cur.DirtySummary)
	}

	baseRunID := manifest.RunID()
	s.diffSurfaces(ctx, scenario, baseRunID, manifest, surface, &res)

	// Surfaces requested but never captured (recorded in manifest.Skipped) are
	// reported as not-comparable so a partial baseline cannot masquerade as a
	// clean one.
	for _, id := range sortedSkippedIDs(manifest) {
		if surface != "" && id != surface {
			continue
		}
		if _, captured := manifest.Surfaces[id]; captured {
			continue
		}
		appendSurface(&res, notComparable(id, "surface was not captured in this baseline: "+manifest.Skipped[id]))
	}

	if surface != "" && len(res.Surfaces) == 0 {
		return DiffResult{}, fmt.Errorf("surface %q not present in baseline %q", surface, name)
	}
	return res, nil
}

// diffSurfaces runs the current comprehensive run + one CompareRuns, then
// buckets the per-phase deltas into the captured phase-set surfaces and diffs
// the visuals surface at the metadata level.
func (s *Service) diffSurfaces(ctx context.Context, scenario, baseRunID string, manifest BaselineManifest, surface string, res *DiffResult) {
	captured := s.sortedSurfaceIDs(manifest)
	markAll := func(reason string) {
		for _, id := range captured {
			if surface != "" && id != surface {
				continue
			}
			appendSurface(res, notComparable(id, reason))
		}
	}

	if baseRunID == "" {
		markAll("baseline has no comparable run")
		return
	}
	if reason := s.reachabilityError(ctx); reason != "" {
		markAll(reason)
		return
	}

	cur, err := s.runToCompletion(ctx, scenario)
	if err != nil || cur.RunID == "" {
		reason := "could not run current comprehensive suite"
		if err != nil {
			reason += ": " + err.Error()
		}
		markAll(reason)
		return
	}
	curRunID := cur.RunID

	// ONE empty-phase compare returns every phase's delta; bucket locally.
	cmp, cmpErr := s.runs.CompareRuns(ctx, scenario, baseRunID, curRunID, "")
	bucketed := bucketPhaseDiffs(cmp)

	for _, id := range captured {
		if surface != "" && id != surface {
			continue
		}
		if id == SurfaceVisuals {
			appendSurface(res, s.visualsDiff(ctx, scenario, baseRunID, curRunID))
			continue
		}
		if cmpErr != nil {
			appendSurface(res, notComparable(id, "compare failed: "+cmpErr.Error()))
			continue
		}
		d, ok := bucketed[id]
		if !ok {
			// No phase for this surface appeared in the compare (e.g. the phase
			// was skipped in one run) — report not-comparable rather than
			// silently clean.
			appendSurface(res, notComparable(id, "no comparable phase results for surface"))
			continue
		}
		appendSurface(res, d)
	}
}

// bucketPhaseDiffs folds the flat PhaseDiff[] from one empty-phase CompareRuns
// into one SurfaceDiff per phase-set surface (option-c). A multi-phase surface
// like `tests` aggregates unit+integration+smoke. Phases with no owning surface
// are dropped.
func bucketPhaseDiffs(cmp CompareResult) map[string]SurfaceDiff {
	acc := map[string]*SurfaceDiff{}
	for _, p := range cmp.Phases {
		surfaceID, ok := phaseSurface[p.Phase]
		if !ok {
			continue
		}
		d := acc[surfaceID]
		if d == nil {
			d = &SurfaceDiff{SurfaceID: surfaceID, Verdict: VerdictClean}
			acc[surfaceID] = d
		}
		d.Verdict = WorseVerdict(d.Verdict, Verdict(p.Verdict))
		d.Regressions = append(d.Regressions, p.Regressions...)
		d.NewFailures = append(d.NewFailures, p.NewFailures...)
		d.Preexisting = append(d.Preexisting, p.Preexisting...)
		d.Cleared = append(d.Cleared, p.Cleared...)
	}
	out := make(map[string]SurfaceDiff, len(acc))
	for id, d := range acc {
		d.Summary = summarizeVerdict(*d)
		out[id] = *d
	}
	return out
}

// visualsDiff compares the two runs' visual artifacts at the pixel level via
// test-genie's CompareRunVisuals (test-genie owns the analyzer). The visuals
// surface is purely advisory: every per-page difference is reported as a neutral
// "review before/after" signal, never a failure. A clearly-broken render is
// caught earlier — at smoke time, where it fails its phase — so it surfaces on
// the test/smoke surface, not here. This keeps one concept per surface: smoke is
// the pass/fail authority; the visuals diff answers "did the UI move?".
func (s *Service) visualsDiff(ctx context.Context, scenario, baseRunID, curRunID string) SurfaceDiff {
	deltas, err := s.runs.CompareRunVisuals(ctx, scenario, baseRunID, curRunID)
	if err != nil {
		return notComparable(SurfaceVisuals, "compare visuals: "+err.Error())
	}
	return diffVisuals(deltas)
}

// diffVisuals folds per-page visual deltas into the advisory `changed` tier.
// Any non-identical page (changed pixels, added, or removed) is a neutral review
// item; the surface verdict is `changed` when there is at least one, else
// `clean`. It never emits a failing verdict — visuals are advisory by contract.
func diffVisuals(deltas []VisualDelta) SurfaceDiff {
	d := SurfaceDiff{SurfaceID: SurfaceVisuals, Verdict: VerdictClean}
	for _, delta := range deltas {
		switch delta.Status {
		case "changed":
			d.Changed = append(d.Changed, fmt.Sprintf("%s changed (%.0f%% of frame)", delta.Page, delta.ChangedFraction*100))
		case "added":
			d.Changed = append(d.Changed, fmt.Sprintf("%s captured now, absent in baseline (review)", delta.Page))
		case "removed":
			d.Changed = append(d.Changed, fmt.Sprintf("%s captured in baseline, absent now (review)", delta.Page))
		}
	}
	sort.Strings(d.Changed)
	if len(d.Changed) > 0 {
		d.Verdict = VerdictChanged
	}
	d.Summary = summarizeVerdict(d)
	return d
}

// appendSurface adds a surface diff and rolls its verdict into the overall one.
func appendSurface(res *DiffResult, d SurfaceDiff) {
	res.Surfaces = append(res.Surfaces, d)
	res.Verdict = WorseVerdict(res.Verdict, d.Verdict)
}

// notComparable builds a not-comparable SurfaceDiff with a reason.
func notComparable(surfaceID, reason string) SurfaceDiff {
	return SurfaceDiff{SurfaceID: surfaceID, Verdict: VerdictNotComparable, Summary: reason}
}

// summarizeVerdict renders a one-line human summary for a surface diff.
func summarizeVerdict(d SurfaceDiff) string {
	switch d.Verdict {
	case VerdictClean:
		if len(d.Cleared) > 0 {
			return fmt.Sprintf("no regressions (%d cleared)", len(d.Cleared))
		}
		return "no change"
	case VerdictChanged:
		return fmt.Sprintf("%d change(s) to review (not a failure)", len(d.Changed))
	case VerdictRegression:
		return fmt.Sprintf("%d regression(s)", len(d.Regressions))
	case VerdictNewFailure:
		return fmt.Sprintf("%d new failure(s) (added by your changes)", len(d.NewFailures))
	case VerdictPreexisting:
		return fmt.Sprintf("%d preexisting failure(s) (inherited)", len(d.Preexisting))
	case VerdictNotComparable:
		return "not comparable"
	}
	return string(d.Verdict)
}

// Delete removes a baseline and unpins the shared comprehensive run it pinned.
func (s *Service) Delete(ctx context.Context, repoID int64, scenario, branch, name string) error {
	manifest, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return err
	}
	s.unpinRun(ctx, manifest)
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

// unpinRun releases the single shared run a baseline pinned. Every surface
// references the same run, so this unpins once regardless of surface count.
func (s *Service) unpinRun(ctx context.Context, m BaselineManifest) {
	if s.runs == nil {
		return
	}
	runID := m.RunID()
	if runID == "" {
		return
	}
	_ = s.runs.UnpinRun(ctx, m.Scenario, runID, PinOwner(m.Name))
}

// sortedSkippedIDs returns the manifest's skipped surface IDs in canonical
// order (same ordering rule as sortedSurfaceIDs).
func sortedSkippedIDs(m BaselineManifest) []string {
	ids := make([]string, 0, len(m.Skipped))
	for id := range m.Skipped {
		ids = append(ids, id)
	}
	sortSurfaceIDs(ids)
	return ids
}

func (s *Service) sortedSurfaceIDs(m BaselineManifest) []string {
	ids := make([]string, 0, len(m.Surfaces))
	for id := range m.Surfaces {
		ids = append(ids, id)
	}
	sortSurfaceIDs(ids)
	return ids
}

// sortSurfaceIDs orders surface IDs by the canonical AllSurfaces order first,
// then any extras alphabetically.
func sortSurfaceIDs(ids []string) {
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
}
