package baseline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"git-control-tower/internal/git"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

// PinOwner returns the canonical test-genie pin owner for a baseline. Deleting
// the baseline unpins exactly this owner, releasing the run to normal GC.
func PinOwner(name string) string { return "gct:baseline:" + name }

// Service orchestrates baseline capture/diff/delete. A baseline is one
// comprehensive, durable Test Genie run pinned once. The service owns no run
// history or provider registry.
type Service struct {
	storage    *Storage
	exec       Executor
	runs       RunsClient
	probe      StalenessProbe
	reachable  Reachability
	captureGit func(ctx context.Context, repoDir string) (git.State, error)
	now        func() time.Time
	// reuseTTL bounds clean-tree run reuse: a completed run at the current sha is
	// reused only when it finished within this window (0 = no reuse). Lever
	// GCT_DIFF_RUN_REUSE_TTL.
	reuseTTL  time.Duration
	captureMu sync.Mutex
	// migrationMu serializes legacy retention reconciliation in this process.
	// Test Genie's pin owner is itself idempotent, so concurrent processes and
	// crash recovery still converge on one durable retention claim.
	migrationMu sync.Mutex
	// collectionDiffMu protects whole-operation member checkpoint rewrites while
	// independent child waits run concurrently outside the lock.
	collectionDiffMu sync.Mutex
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
	ReuseTTL   time.Duration
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
		reuseTTL:   d.ReuseTTL,
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

// CreateRequest captures a comprehensive baseline.
type CreateRequest struct {
	RepoID    int64
	RepoDir   string
	Scenario  string
	Name      string
	Branch    string // optional override; default derived from git state
	CreatedBy string
	Reason    string
}

// CreateResult is the outcome of Create.
type CreateResult struct {
	Manifest     BaselineManifest
	DirtyWarning string // non-empty when captured against a dirty tree
}

// LegacyOrphanReport is read-only migration evidence. It intentionally does
// not repair or delete anything: an operator must choose an explicit repair
// after inspecting the durable identity and run handle.
type LegacyOrphanReport struct {
	Scenario string
	Branch   string
	Name     string
	RunID    string
	Reason   string
}

type RepairRequest struct {
	RepoID   int64
	Scenario string
	Branch   string
	Name     string
	DryRun   bool
}

type RepairResult struct {
	Generation int64
	Actions    []string
	Applied    bool
}

// RepairBaseline plans and, when requested, applies only deterministic
// lifecycle cleanup. It never manufactures a missing manifest or replaces an
// anchor; those cases require an explicit recapture.
func (s *Service) RepairBaseline(ctx context.Context, req RepairRequest) (RepairResult, error) {
	lifecycle, err := s.storage.LoadLifecycle(req.RepoID, req.Scenario, req.Branch, req.Name)
	if err != nil {
		return RepairResult{}, err
	}
	result := RepairResult{Generation: lifecycle.Generation}
	manifest, manifestErr := s.storage.Load(req.RepoID, req.Scenario, req.Branch, req.Name)
	if manifestErr != nil && !errors.Is(manifestErr, ErrNotFound) {
		return RepairResult{}, manifestErr
	}
	switch lifecycle.Status {
	case LifecycleDeleted:
		if manifestErr == nil {
			result.Actions = append(result.Actions, "unpin retained manifest run", "remove tombstoned manifest")
			if !req.DryRun {
				if err := s.unpinRun(ctx, manifest); err != nil {
					return RepairResult{}, fmt.Errorf("repair tombstoned baseline retention pin: %w", err)
				}
				if err := s.storage.Delete(req.RepoID, req.Scenario, req.Branch, req.Name); err != nil {
					return RepairResult{}, err
				}
			}
		} else {
			result.Actions = append(result.Actions, "tombstone already converged")
		}
	case LifecycleReady:
		if errors.Is(manifestErr, ErrNotFound) {
			result.Actions = append(result.Actions, "missing ready manifest requires explicit recapture")
		} else {
			result.Actions = append(result.Actions, "ready lifecycle already converged")
		}
	case LifecycleCapturing:
		result.Actions = append(result.Actions, "leave capture for durable status reconciliation")
	case LifecycleFailed:
		result.Actions = append(result.Actions, "failed capture requires explicit recapture")
	}
	if !req.DryRun {
		entry := LifecycleAuditEntry{Scenario: req.Scenario, Branch: req.Branch, Name: req.Name, Generation: lifecycle.Generation, Action: "repair", Detail: strings.Join(result.Actions, "; "), CreatedAt: s.now().UTC()}
		if err := s.storage.SaveLifecycleAudit(req.RepoID, entry); err != nil {
			return RepairResult{}, err
		}
		result.Applied = true
	}
	return result, nil
}

// ScanLegacyOrphans reports completed snapshot intents that have no manifest
// and no deletion tombstone. These are the historical split-state records that
// predate lifecycle ownership. The scan is deterministic and non-mutating.
func (s *Service) ScanLegacyOrphans(repoID int64) ([]LegacyOrphanReport, error) {
	intents, err := s.storage.ListAllSnapshotIntents(repoID)
	if err != nil {
		return nil, err
	}
	reports := make([]LegacyOrphanReport, 0)
	for _, intent := range intents {
		if intent.Status != "ready" {
			continue
		}
		if lifecycle, lifecycleErr := s.storage.LoadLifecycle(repoID, intent.Scenario, intent.Branch, intent.Name); lifecycleErr == nil {
			// Lifecycle-managed records are never legacy orphans. A tombstone is
			// deliberate deletion evidence, while ready/capturing are governed by
			// the transition coordinator.
			_ = lifecycle
			continue
		} else if !errors.Is(lifecycleErr, ErrNotFound) {
			return nil, lifecycleErr
		}
		if _, manifestErr := s.storage.Load(repoID, intent.Scenario, intent.Branch, intent.Name); manifestErr == nil {
			continue
		} else if !errors.Is(manifestErr, ErrNotFound) {
			return nil, manifestErr
		}
		reports = append(reports, LegacyOrphanReport{Scenario: intent.Scenario, Branch: intent.Branch, Name: intent.Name, RunID: intent.Run.RunID, Reason: "ready snapshot intent has no manifest or lifecycle record"})
	}
	sort.Slice(reports, func(i, j int) bool {
		if reports[i].Scenario != reports[j].Scenario {
			return reports[i].Scenario < reports[j].Scenario
		}
		if reports[i].Branch != reports[j].Branch {
			return reports[i].Branch < reports[j].Branch
		}
		return reports[i].Name < reports[j].Name
	})
	return reports, nil
}

// Create is the blocking capture form used by focused tests and internal
// callers. The public SnapshotForBaseline path uses StartCapture/FinalizeCapture.
func (s *Service) Create(ctx context.Context, req CreateRequest) (CreateResult, error) {
	manifest, err := s.buildManifestSkeleton(ctx, req)
	if err != nil {
		return CreateResult{}, err
	}
	h, err := s.startCaptureRun(ctx, req.Scenario)
	if err != nil {
		return CreateResult{}, err
	}
	if err := s.beginLifecycle(req.RepoID, manifest, h.RunID); err != nil {
		return CreateResult{}, fmt.Errorf("begin baseline lifecycle: %w", err)
	}
	terminal, _, err := s.awaitStableCapture(ctx, req.Scenario, h, nil)
	if err != nil {
		_ = s.setLifecycle(req.RepoID, manifest, LifecycleFailed, h.RunID, err.Error())
		return CreateResult{}, err
	}
	if err := s.validateBaselineEvidence(ctx, req.Scenario, terminal.RunID); err != nil {
		_ = s.setLifecycle(req.RepoID, manifest, LifecycleFailed, h.RunID, err.Error())
		return CreateResult{}, err
	}
	if err := s.attachRun(ctx, &manifest, req, terminal); err != nil {
		_ = s.setLifecycle(req.RepoID, manifest, LifecycleFailed, h.RunID, err.Error())
		return CreateResult{}, err
	}

	if err := s.storage.Save(req.RepoID, manifest, CreateOnly); err != nil {
		_ = s.unpinRun(ctx, manifest)
		return CreateResult{}, err
	}
	if err := s.setLifecycle(req.RepoID, manifest, LifecycleReady, manifest.RunID(), ""); err != nil {
		return CreateResult{}, s.rollbackUncommittedReady(ctx, req.RepoID, manifest, err)
	}

	res := CreateResult{Manifest: manifest}
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

	pending := PendingCapture{Manifest: manifest, Req: req, Run: h}
	if err := s.beginLifecycle(req.RepoID, manifest, h.RunID); err != nil {
		return PendingCapture{}, err
	}
	if manifest.Git.Dirty {
		pending.DirtyWarning = dirtyCaptureWarning(manifest.Git.DirtySummary)
	}
	if err := s.saveSnapshotIntent(ctx, pending, "pending", ""); err != nil {
		return PendingCapture{}, err
	}
	return pending, nil
}

// FinalizeCapture blocks until the started run is terminal, then pins it once,
// records its immutable run anchor, and writes the manifest. It runs on a
// server-owned context so a disconnected client never abandons a half-pinned
// baseline. captureMu deliberately protects only the terminal commit: Test
// Genie waits may be long-lived, and must not prevent an unrelated terminal
// capture from being committed.
func (s *Service) FinalizeCapture(ctx context.Context, pending PendingCapture) (CreateResult, error) {
	res, _, err := s.awaitStableCapture(ctx, pending.Req.Scenario, pending.Run, func(retry RunHandle) error {
		// Persist the replacement run before waiting on it. If this process is
		// interrupted mid-retry, normal recovery resumes the new server-owned run
		// instead of treating the raced attempt as a terminal baseline result.
		pending.Run = retry
		return s.saveSnapshotIntent(ctx, pending, "pending", "retrying after relevant source inputs changed during the prior attempt")
	})
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			// This attachment ended; the Test Genie run remains server-owned.
			// Keep the durable intent pending so status/restart recovery can resume.
			return CreateResult{}, err
		}
		msg := "comprehensive run failed: " + err.Error()
		_ = s.setLifecycle(pending.Req.RepoID, pending.Manifest, LifecycleFailed, pending.Run.RunID, msg)
		_ = s.saveSnapshotIntent(ctx, pending, "failed", msg)
		return CreateResult{}, errors.New(msg)
	}
	if err := s.validateBaselineEvidence(ctx, pending.Req.Scenario, res.RunID); err != nil {
		_ = s.setLifecycle(pending.Req.RepoID, pending.Manifest, LifecycleFailed, pending.Run.RunID, err.Error())
		_ = s.saveSnapshotIntent(ctx, pending, "failed", err.Error())
		return CreateResult{}, err
	}

	// The only shared critical section begins after terminal Test Genie truth is
	// available. Rechecking storage here makes duplicate finalizers converge:
	// they may both await the same durable run, but only one pins and writes.
	s.captureMu.Lock()
	defer s.captureMu.Unlock()
	if existing, loadErr := s.storage.Load(pending.Req.RepoID, pending.Manifest.Scenario, pending.Manifest.Branch, pending.Manifest.Name); loadErr == nil {
		if existing.RunID() == pending.Run.RunID {
			_ = s.saveSnapshotIntent(ctx, pending, "ready", "")
			return CreateResult{Manifest: existing, DirtyWarning: pending.DirtyWarning}, nil
		}
		return CreateResult{}, ErrAlreadyExists
	} else if !errors.Is(loadErr, ErrNotFound) {
		return CreateResult{}, loadErr
	}

	if lifecycle, lifecycleErr := s.storage.LoadLifecycle(pending.Req.RepoID, pending.Manifest.Scenario, pending.Manifest.Branch, pending.Manifest.Name); lifecycleErr == nil && lifecycle.Status == LifecycleDeleted {
		// attachRun has already acquired the idempotent retention pin. A delete
		// that won the race must not leave the newly completed run retained.
		_ = s.unpinRun(ctx, manifestForRun(pending.Manifest, res))
		return CreateResult{}, fmt.Errorf("baseline %q was deleted while capture was in progress", pending.Manifest.Name)
	} else if lifecycleErr != nil && !errors.Is(lifecycleErr, ErrNotFound) {
		return CreateResult{}, lifecycleErr
	}
	manifest := pending.Manifest
	if err := s.attachRun(ctx, &manifest, pending.Req, res); err != nil {
		_ = s.saveSnapshotIntent(ctx, pending, "failed", err.Error())
		return CreateResult{}, err
	}
	if err := s.storage.Save(pending.Req.RepoID, manifest, CreateOnly); err != nil {
		_ = s.unpinRun(ctx, manifest)
		_ = s.saveSnapshotIntent(ctx, pending, "failed", err.Error())
		return CreateResult{}, err
	}
	if err := s.setLifecycle(pending.Req.RepoID, manifest, LifecycleReady, manifest.RunID(), ""); err != nil {
		commitErr := s.rollbackUncommittedReady(ctx, pending.Req.RepoID, manifest, err)
		_ = s.saveSnapshotIntent(ctx, pending, "failed", commitErr.Error())
		return CreateResult{}, commitErr
	}
	out := CreateResult{Manifest: manifest, DirtyWarning: pending.DirtyWarning}
	_ = s.saveSnapshotIntent(ctx, pending, "ready", "")
	return out, nil
}

// rollbackUncommittedReady compensates a failed lifecycle commit after a
// manifest write. A ready manifest without its lifecycle authority would
// recreate the split state this service is designed to eliminate, so it is
// never exposed as a successful baseline. Best-effort cleanup errors are kept
// in the returned failure for an operator/audit trail.
func (s *Service) rollbackUncommittedReady(ctx context.Context, repoID int64, manifest BaselineManifest, lifecycleErr error) error {
	cleanup := make([]string, 0, 2)
	if err := s.storage.Delete(repoID, manifest.Scenario, manifest.Branch, manifest.Name); err != nil {
		cleanup = append(cleanup, "remove manifest: "+err.Error())
	}
	if err := s.unpinRun(ctx, manifest); err != nil {
		cleanup = append(cleanup, "unpin run: "+err.Error())
	}
	if len(cleanup) == 0 {
		return fmt.Errorf("commit ready lifecycle: %w", lifecycleErr)
	}
	return fmt.Errorf("commit ready lifecycle: %w (compensation failed: %s)", lifecycleErr, strings.Join(cleanup, "; "))
}

// manifestForRun creates the minimal valid ownership record required to undo a
// pin when a terminal capture loses to a deletion tombstone.
func manifestForRun(seed BaselineManifest, result ExecResult) BaselineManifest {
	seed.Run.RunID = result.RunID
	return seed
}

func (s *Service) beginLifecycle(repoID int64, manifest BaselineManifest, runID string) error {
	generation := int64(1)
	if prior, err := s.storage.LoadLifecycle(repoID, manifest.Scenario, manifest.Branch, manifest.Name); err == nil {
		generation = prior.Generation + 1
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	return s.storage.SaveLifecycle(repoID, LifecycleRecord{Scenario: manifest.Scenario, Branch: manifest.Branch, Name: manifest.Name, Generation: generation, Status: LifecycleCapturing, RunID: runID, UpdatedAt: s.now().UTC()})
}

func (s *Service) setLifecycle(repoID int64, manifest BaselineManifest, status LifecycleStatus, runID, message string) error {
	record, err := s.storage.LoadLifecycle(repoID, manifest.Scenario, manifest.Branch, manifest.Name)
	if errors.Is(err, ErrNotFound) {
		record = LifecycleRecord{Scenario: manifest.Scenario, Branch: manifest.Branch, Name: manifest.Name, Generation: 1}
	} else if err != nil {
		return err
	}
	record.Status, record.RunID, record.Error, record.UpdatedAt = status, runID, message, s.now().UTC()
	return s.storage.SaveLifecycle(repoID, record)
}

// awaitStableCapture owns one bounded retry for a raced source snapshot. A
// relevant edit invalidates only the current attempt; it never invalidates an
// already-stored baseline before-result.
func (s *Service) awaitStableCapture(ctx context.Context, scenario string, handle RunHandle, onRetry func(RunHandle) error) (ExecResult, RunHandle, error) {
	current := handle
	for attempt := 0; attempt < 2; attempt++ {
		res, err := s.exec.AwaitResult(ctx, scenario, current.RunID)
		if err != nil {
			return ExecResult{}, current, fmt.Errorf("comprehensive run failed: %w", err)
		}
		if isStableScopedEvidence(res) {
			return res, current, nil
		}
		if attempt == 1 {
			return ExecResult{}, current, fmt.Errorf("relevant source inputs changed during two capture attempts; rerun to capture the current declared inputs")
		}
		started, err := s.startCaptureRun(ctx, scenario)
		if err != nil {
			return ExecResult{}, current, fmt.Errorf("retry comprehensive run after relevant source edit: %w", err)
		}
		if onRetry != nil {
			if err := onRetry(started); err != nil {
				return ExecResult{}, current, fmt.Errorf("persist capture retry intent: %w", err)
			}
		}
		current = started
	}
	return ExecResult{}, current, fmt.Errorf("capture retry state exhausted")
}

func isStableScopedEvidence(res ExecResult) bool {
	// Historical Test Genie terminal records predate explicit tier/stability
	// fields. A digest-stamped legacy record is readable; new raced attempts are
	// unambiguous because they carry the explicit degraded tier.
	if res.EvidenceTier == "" && res.TreeDigest != "" {
		return true
	}
	return res.SourceStable && (res.EvidenceTier == "strict" || res.EvidenceTier == "shared-scoped")
}

// validateBaselineEvidence verifies that Test Genie can read the terminal
// evidence. It intentionally does not reject a before-result just because one
// phase was unavailable or incomparable: that condition belongs to the later
// comparison's coverage axis. Retaining the baseline lets a later rerun measure
// that phase without losing the durable before behavior for every other phase.
func (s *Service) validateBaselineEvidence(ctx context.Context, scenario, runID string) error {
	if s.runs == nil {
		return fmt.Errorf("validate baseline evidence: Test Genie runs client is unavailable")
	}
	comparison, err := s.runs.CompareRuns(ctx, scenario, runID, runID, "")
	if err != nil {
		return fmt.Errorf("validate baseline evidence for run %s: %w", runID, err)
	}
	_ = comparison
	return nil
}

func (s *Service) saveSnapshotIntent(_ context.Context, pending PendingCapture, status, errMsg string) error {
	now := s.now().UTC()
	intent := SnapshotIntent{
		Status:       status,
		Error:        errMsg,
		RepoID:       pending.Req.RepoID,
		RepoDir:      pending.Req.RepoDir,
		Scenario:     pending.Req.Scenario,
		Branch:       pending.Manifest.Branch,
		Name:         pending.Req.Name,
		CreatedBy:    pending.Req.CreatedBy,
		Reason:       pending.Req.Reason,
		Manifest:     pending.Manifest,
		Run:          pending.Run,
		DirtyWarning: pending.DirtyWarning,
		CreatedAt:    pending.Manifest.CreatedAt,
		UpdatedAt:    now,
	}
	if intent.CreatedAt.IsZero() {
		intent.CreatedAt = now
	}
	return s.storage.SaveSnapshotIntent(pending.Req.RepoID, intent)
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
		SchemaVersion: SchemaVersion,
	}, nil
}

func (s *Service) startCaptureRun(ctx context.Context, scenario string) (RunHandle, error) {
	if reason := s.reachabilityError(ctx); reason != "" {
		return RunHandle{}, fmt.Errorf("test-genie is not reachable: %s", reason)
	}
	h, err := s.exec.StartRun(ctx, scenario)
	if err != nil || h.RunID == "" {
		if err != nil {
			return RunHandle{}, fmt.Errorf("start comprehensive run: %w", err)
		}
		return RunHandle{}, fmt.Errorf("test-genie returned no run id")
	}
	return h, nil
}

// attachRun pins and records the one canonical run. PinRun is idempotent for the
// owner key, so recovery after a crash between pin and save cannot create a
// duplicate retention claim.
func (s *Service) attachRun(ctx context.Context, manifest *BaselineManifest, req CreateRequest, res ExecResult) error {
	if res.RunID == "" {
		return fmt.Errorf("test-genie returned no run id")
	}
	// V2 Test Genie evidence did not serialize scoped provenance. It remains a
	// readable baseline format; when it has a content digest, project it into
	// the normal shared-scoped tier rather than manufacturing the old
	// dirty-workspace rejection. New Test Genie writers always set these fields.
	if res.EvidenceTier == "" && res.TreeDigest != "" {
		res.EvidenceTier, res.SourceScope, res.SourceStable = "shared-scoped", "scenario:"+req.Scenario, true
	}
	if !res.SourceStable || (res.EvidenceTier != "strict" && res.EvidenceTier != "shared-scoped") {
		return fmt.Errorf("run %s did not retain stable scoped source evidence; rerun to capture the current declared inputs", res.RunID)
	}
	if err := s.runs.PinRun(ctx, req.Scenario, res.RunID, PinOwner(req.Name), "baseline:"+req.Name); err != nil {
		return fmt.Errorf("pin run %s: %w", res.RunID, err)
	}
	capturedAt := res.CompletedAt.UTC()
	if capturedAt.IsZero() {
		capturedAt = s.now().UTC()
	}
	profile := res.CaptureProfile
	if profile == "" {
		profile = CaptureProfile
	}
	manifest.Run = RunAnchor{
		RunID:                           res.RunID,
		CapturedAt:                      capturedAt,
		CaptureProfile:                  profile,
		TreeDigest:                      res.TreeDigest,
		PhaseSetDigest:                  res.PhaseSetDigest,
		DescriptorSnapshotRef:           "test-genie-run:" + res.RunID + "#descriptor-snapshot",
		DescriptorSnapshotDigest:        res.DescriptorSnapshotDigest,
		DescriptorSnapshotSchemaVersion: res.DescriptorSnapshotSchemaVersion,
		EvidenceTier:                    res.EvidenceTier,
		SourceScope:                     res.SourceScope,
		SourceStable:                    res.SourceStable,
	}
	return nil
}

// dirtyCaptureWarning is advisory only: dirtiness outside the declared source
// scope does not make a baseline invalid.
func dirtyCaptureWarning(summary string) string {
	return fmt.Sprintf("baseline captured in a shared dirty workspace (%s); scoped provenance records the tested inputs", summary)
}

// Get returns a single baseline manifest.
func (s *Service) Get(ctx context.Context, repoID int64, scenario, branch, name string) (BaselineManifest, error) {
	if lifecycle, lifecycleErr := s.storage.LoadLifecycle(repoID, scenario, branch, name); lifecycleErr == nil && lifecycle.Status == LifecycleDeleted {
		return BaselineManifest{}, ErrNotFound
	} else if lifecycleErr != nil && !errors.Is(lifecycleErr, ErrNotFound) {
		return BaselineManifest{}, lifecycleErr
	}
	m, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return BaselineManifest{}, err
	}
	return s.reconcileMigrationPin(ctx, repoID, m)
}

// List returns baselines for a scenario; empty branch lists all branches.
func (s *Service) List(ctx context.Context, repoID int64, scenario, branch string) ([]BaselineManifest, error) {
	manifests, err := s.storage.List(repoID, scenario, branch)
	if err != nil {
		return nil, err
	}
	active := manifests[:0]
	for i := range manifests {
		lifecycle, lifecycleErr := s.storage.LoadLifecycle(repoID, manifests[i].Scenario, manifests[i].Branch, manifests[i].Name)
		if lifecycleErr == nil && lifecycle.Status == LifecycleDeleted {
			continue
		}
		if lifecycleErr != nil && !errors.Is(lifecycleErr, ErrNotFound) {
			return nil, lifecycleErr
		}
		manifests[i], err = s.reconcileMigrationPin(ctx, repoID, manifests[i])
		if err != nil {
			return nil, err
		}
		active = append(active, manifests[i])
	}
	return active, nil
}

// reconcileMigrationPin completes the retention half of a V1-to-V2 migration.
// Storage can safely rewrite the manifest without network access, but only the
// service owns the Test Genie seam. The persisted checkpoint is deliberately
// written after PinRun: a crash may repeat an idempotent pin, but can never
// publish a checkpoint for a retention claim that was not accepted.
func (s *Service) reconcileMigrationPin(ctx context.Context, repoID int64, manifest BaselineManifest) (BaselineManifest, error) {
	if manifest.Migration == nil || !manifest.Migration.PinReconciledAt.IsZero() {
		return manifest, nil
	}
	if s.runs == nil {
		return BaselineManifest{}, fmt.Errorf("reconcile migrated baseline %q retention pin: Test Genie runs client is unavailable", manifest.Name)
	}

	s.migrationMu.Lock()
	defer s.migrationMu.Unlock()

	// Another reader may have reconciled and persisted the checkpoint while we
	// waited. Reload under the storage lock before issuing the external call.
	current, err := s.storage.Load(repoID, manifest.Scenario, manifest.Branch, manifest.Name)
	if err != nil {
		return BaselineManifest{}, err
	}
	if current.Migration == nil || !current.Migration.PinReconciledAt.IsZero() {
		return current, nil
	}
	if err := s.runs.PinRun(ctx, current.Scenario, current.RunID(), PinOwner(current.Name), "baseline-migration:"+current.Name); err != nil {
		return BaselineManifest{}, fmt.Errorf("reconcile migrated baseline %q retention pin: %w", current.Name, err)
	}
	current.Migration.PinReconciledAt = s.now().UTC()
	if err := s.storage.Save(repoID, current, Overwrite); err != nil {
		return BaselineManifest{}, fmt.Errorf("checkpoint migrated baseline %q retention pin: %w", current.Name, err)
	}
	return current, nil
}

// SnapshotStatusRequest parameterizes GetSnapshotStatus.
type SnapshotStatusRequest struct {
	RepoID   int64
	RepoDir  string
	Scenario string
	Branch   string
	Name     string
	RunID    string
	Wait     bool
}

// SnapshotStatus reports the lifecycle of a return-fast snapshot capture.
type SnapshotStatus struct {
	Status                      string // pending | ready | failed | missing
	Scenario                    string
	Name                        string
	Branch                      string
	RunID                       string
	RunStatus                   string
	Baseline                    *BaselineManifest
	Error                       string
	SimilarBaselines            []string
	RecommendedNextCheckSeconds int
}

// GetSnapshotStatus reconciles a durable snapshot intent with test-genie's run
// state. It is intentionally able to finish a pending snapshot after a GCT API
// restart by replaying FinalizeCapture from the persisted intent.
func (s *Service) GetSnapshotStatus(ctx context.Context, req SnapshotStatusRequest) (SnapshotStatus, error) {
	out := SnapshotStatus{Status: "missing", Scenario: req.Scenario, Name: req.Name, Branch: req.Branch, RunID: req.RunID}
	if manifest, err := s.Get(ctx, req.RepoID, req.Scenario, req.Branch, req.Name); err == nil {
		m := manifest
		out.Status = "ready"
		out.Baseline = &m
		out.RunID = m.RunID()
		return out, nil
	} else if !errors.Is(err, ErrNotFound) {
		return SnapshotStatus{}, err
	}

	intents, err := s.storage.ListSnapshotIntents(req.RepoID, req.Scenario, req.Branch, req.Name)
	if err != nil {
		return SnapshotStatus{}, err
	}
	intent, ok := selectSnapshotIntent(intents, req.RunID)
	if !ok {
		out.SimilarBaselines = similarBaselineNames(s.storage, req.RepoID, req.Scenario, req.Branch, req.Name)
		out.Error = "baseline manifest and snapshot intent not found"
		return out, nil
	}

	out.Status = intent.Status
	out.RunID = intent.Run.RunID
	out.RunStatus = intent.Status
	out.Error = intent.Error
	pending := intent.PendingCapture()

	st, serr := s.exec.RunStatus(ctx, intent.Scenario, intent.Run.RunID)
	if serr == nil {
		out.RunStatus = st.Status
		out.RecommendedNextCheckSeconds = st.RecommendedNextCheckSeconds
		if !st.Terminal && !req.Wait {
			out.Status = "pending"
			return out, nil
		}
	} else if req.Wait || intent.Status == "pending" {
		return SnapshotStatus{}, serr
	}

	if intent.Status == "ready" {
		if manifest, err := s.Get(ctx, req.RepoID, req.Scenario, req.Branch, req.Name); err == nil {
			m := manifest
			out.Baseline = &m
			out.RunID = m.RunID()
			return out, nil
		}
		out.Status = "failed"
		out.Error = "snapshot intent says ready but baseline manifest is missing"
		return out, nil
	}
	if intent.Status == "failed" {
		return out, nil
	}

	res, ferr := s.FinalizeCapture(ctx, pending)
	if ferr != nil {
		out.Status = "failed"
		out.Error = ferr.Error()
		return out, nil
	}
	out.Status = "ready"
	out.Error = ""
	out.RunStatus = "passed"
	out.Baseline = &res.Manifest
	out.RunID = res.Manifest.RunID()
	return out, nil
}

// PendingSnapshotCaptures reconstructs pending snapshot finalizers after an API
// restart. The handler reattaches these to server-owned goroutines so
// return-fast snapshots remain durable across process lifetimes, not just client
// disconnects.
func (s *Service) PendingSnapshotCaptures(repoID int64, repoDir string) ([]PendingCapture, error) {
	intents, err := s.storage.ListAllSnapshotIntents(repoID)
	if err != nil {
		return nil, err
	}
	out := make([]PendingCapture, 0, len(intents))
	for _, intent := range intents {
		if intent.Status != "pending" {
			continue
		}
		if _, err := s.storage.Load(repoID, intent.Scenario, intent.Branch, intent.Name); err == nil {
			continue
		} else if !errors.Is(err, ErrNotFound) {
			return nil, err
		}
		intent.RepoID = repoID
		if intent.RepoDir == "" {
			intent.RepoDir = repoDir
		}
		out = append(out, intent.PendingCapture())
	}
	return out, nil
}

func selectSnapshotIntent(intents []SnapshotIntent, runID string) (SnapshotIntent, bool) {
	for _, intent := range intents {
		if runID == "" || intent.Run.RunID == runID {
			return intent, true
		}
	}
	return SnapshotIntent{}, false
}

func similarBaselineNames(st *Storage, repoID int64, scenario, branch, want string) []string {
	manifests, err := st.List(repoID, scenario, "")
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var names []string
	for _, m := range manifests {
		if m.Name == want {
			continue
		}
		if branch != "" && m.Branch != branch {
			// Still keep cross-branch names when there are no same-branch names.
			continue
		}
		if !seen[m.Name] {
			seen[m.Name] = true
			names = append(names, m.Name)
		}
	}
	if len(names) == 0 && branch != "" {
		for _, m := range manifests {
			if m.Name == want || seen[m.Name] {
				continue
			}
			seen[m.Name] = true
			names = append(names, m.Name)
		}
	}
	sort.Strings(names)
	if len(names) > 5 {
		names = names[:5]
	}
	return names
}

// EvidenceComparison associates Test Genie's typed, opaque artifacts with the
// two compared runs. Visual deltas are advisory metadata over those references,
// never a synthetic pass/fail surface.
type EvidenceComparison struct {
	BaseRunID        string
	CurrentRunID     string
	BaseCatalog      ArtifactCatalog
	CurrentCatalog   ArtifactCatalog
	VisualDeltas     []VisualDelta
	BlockingReasons  []string
	AdvisoryWarnings []string
	EvidenceStatus   string
	// DegradedReasons is retained as the aggregate read projection for older
	// stored results. New classification logic never derives verdict from it.
	DegradedReasons []string
}

// DiffResult is the descriptor-driven comparison of a baseline against the
// current working tree.
type DiffResult struct {
	Manifest     BaselineManifest
	CurrentGit   git.State
	Staleness    Staleness
	Phases       []*PhaseDiff
	Comparison   *runspb.CompareRunsResponse
	Evidence     EvidenceComparison
	Verdict      Verdict
	DirtyWarning string // non-empty when the current tree is dirty (verdicts are most suspect then)
}

// StartDiffRequest parameterizes StartDiff.
type StartDiffRequest struct {
	RepoID   int64
	RepoDir  string
	Scenario string
	Branch   string
	Name     string
}

// StartDiffOutcome is returned immediately by StartDiff: the comprehensive run
// the diff will compare against (fresh / coalesced / reused) plus the
// PendingDiff handed to FinalizeDiff on a server-owned context.
type StartDiffOutcome struct {
	RunID                 string
	Scenario              string
	Name                  string
	Branch                string
	EstimatedTotalSeconds int
	EtaKnown              bool
	Coalesced             bool
	ReusedRun             bool
	ReusedSha             string
	DirtyWarning          string
	Pending               PendingDiff
}

// PendingDiff is everything FinalizeDiff needs to compute + cache the verdict
// once the current run is terminal. Mirrors PendingCapture's role for snapshots.
type PendingDiff struct {
	RepoID     int64
	Scenario   string
	Branch     string
	Name       string
	Manifest   BaselineManifest
	CurrentGit git.State
	Staleness  Staleness
	BaseRunID  string
	CurRunID   string
}

// StartDiff resolves the current comprehensive run a baseline will be diffed
// against and returns its handle WITHOUT waiting for it. The run is reused (clean
// tree, same sha), coalesced onto an in-flight run, or freshly started (always
// under the one-run-per-scenario guard, so a snapshot/diff already running is
// ridden, never stacked). FinalizeDiff computes + caches the verdict on a
// server-owned context. Mirrors StartCapture's durable, return-fast contract.
func (s *Service) StartDiff(ctx context.Context, req StartDiffRequest) (StartDiffOutcome, error) {
	manifest, err := s.Get(ctx, req.RepoID, req.Scenario, req.Branch, req.Name)
	if err != nil {
		return StartDiffOutcome{}, err
	}
	baseRunID := manifest.RunID()
	if baseRunID == "" {
		return StartDiffOutcome{}, fmt.Errorf("baseline %q has no captured run; recapture it", manifest.Name)
	}

	cur, gerr := s.captureGit(ctx, req.RepoDir)
	if gerr != nil {
		cur = git.State{}
	}
	stale, _ := ComputeStaleness(ctx, s.probe, req.RepoDir, manifest.Git.Sha)

	if reason := s.reachabilityError(ctx); reason != "" {
		return StartDiffOutcome{}, fmt.Errorf("test-genie is not reachable: %s", reason)
	}
	runID, coalesced, reused, sha, handle, err := s.resolveCurrentRun(ctx, req.Scenario, cur)
	if err != nil {
		return StartDiffOutcome{}, err
	}

	out := StartDiffOutcome{
		RunID:                 runID,
		Scenario:              req.Scenario,
		Name:                  req.Name,
		Branch:                req.Branch,
		EstimatedTotalSeconds: handle.EstimatedTotalSeconds,
		EtaKnown:              handle.EtaKnown,
		Coalesced:             coalesced,
		ReusedRun:             reused,
		ReusedSha:             sha,
		Pending: PendingDiff{
			RepoID:     req.RepoID,
			Scenario:   req.Scenario,
			Branch:     req.Branch,
			Name:       req.Name,
			Manifest:   manifest,
			CurrentGit: cur,
			Staleness:  stale,
			BaseRunID:  baseRunID,
			CurRunID:   runID,
		},
	}
	if cur.Dirty {
		out.DirtyWarning = fmt.Sprintf("working tree is dirty (%s) — failures may be caused by uncommitted changes rather than the diff itself", cur.DirtySummary)
	}
	if err := s.saveDiffIntent(ctx, out.Pending, "pending", ""); err != nil {
		return StartDiffOutcome{}, err
	}
	return out, nil
}

func (s *Service) saveDiffIntent(_ context.Context, pending PendingDiff, status, errMsg string) error {
	now := s.now().UTC()
	intent := DiffIntent{
		Status:     status,
		Error:      errMsg,
		RepoID:     pending.RepoID,
		Scenario:   pending.Scenario,
		Branch:     pending.Branch,
		Name:       pending.Name,
		Manifest:   pending.Manifest,
		CurrentGit: pending.CurrentGit,
		Staleness:  pending.Staleness,
		BaseRunID:  pending.BaseRunID,
		CurRunID:   pending.CurRunID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	return s.storage.SaveDiffIntent(pending.RepoID, intent)
}

// resolveCurrentRun decides the comprehensive run a diff compares against:
//   - a completed comprehensive+baseline run whose declared source scope and
//     validation configuration still match → reuse it (no suite re-run);
//   - otherwise StartRun (which itself coalesces onto an in-flight run of the
//     scenario, or starts a fresh one, under the one-run-per-scenario guard).
func (s *Service) resolveCurrentRun(ctx context.Context, scenario string, cur git.State) (runID string, coalesced, reused bool, sha string, handle RunHandle, err error) {
	if rr, found, ferr := s.exec.FindReusableRun(ctx, scenario); ferr == nil && found {
		if s.reuseTTL <= 0 || s.now().UTC().Sub(rr.CompletedAt) <= s.reuseTTL {
			return rr.RunID, false, true, "scoped", RunHandle{RunID: rr.RunID}, nil
		}
	}
	h, serr := s.exec.StartRun(ctx, scenario)
	if serr != nil {
		return "", false, false, "", RunHandle{}, serr
	}
	if h.RunID == "" {
		return "", false, false, "", RunHandle{}, fmt.Errorf("test-genie returned no run id")
	}
	return h.RunID, h.Coalesced, false, "", h, nil
}

// FinalizeDiff blocks until the current run is terminal, computes the diff
// verdict, and persists it under (repoID, scenario, branch, name, runID). It runs
// on a server-owned context so a disconnected client never abandons the result.
// Multiple baseline names sharing one current run each persist their own result.
func (s *Service) FinalizeDiff(ctx context.Context, pending PendingDiff) (CachedDiff, error) {
	currentRun, awaitErr := s.exec.AwaitResult(ctx, pending.Scenario, pending.CurRunID)
	if errors.Is(awaitErr, context.DeadlineExceeded) || errors.Is(awaitErr, context.Canceled) {
		// Queue/execution ownership lives with Test Genie. A transport/server-tail
		// attachment ending cannot publish a terminal not-comparable verdict.
		return CachedDiff{Status: "in_progress", RunID: pending.CurRunID}, awaitErr
	}
	if awaitErr == nil && (!currentRun.SourceStable || (currentRun.EvidenceTier != "strict" && currentRun.EvidenceTier != "shared-scoped")) {
		awaitErr = fmt.Errorf("relevant source inputs changed during this current test attempt; rerun the current evidence")
	}
	diff := s.computeDiff(ctx, pending.Manifest, pending.CurrentGit, pending.Staleness, pending.BaseRunID, pending.CurRunID, awaitErr)
	cd := CachedDiff{Status: "ready", Result: &diff, RunID: pending.CurRunID, ComputedAt: s.now().UTC()}
	if err := s.storage.SaveDiffResult(pending.RepoID, pending.Scenario, pending.Branch, pending.Name, pending.CurRunID, cd); err != nil {
		_ = s.saveDiffIntent(ctx, pending, "failed", err.Error())
		return cd, err
	}
	_ = s.saveDiffIntent(ctx, pending, "ready", "")
	return cd, nil
}

// GetDiffResultRequest parameterizes GetDiffResult.
type GetDiffResultRequest struct {
	RepoID   int64
	RepoDir  string
	Scenario string
	Branch   string
	Name     string
	RunID    string
	Latest   bool
	// Wait blocks SERVER-SIDE until the run is terminal before computing (no
	// client polling). When false, an in-flight run returns status=in_progress.
	Wait bool
}

// GetDiffResult returns the cached diff for (baseline, run). When no cache exists
// it inspects the run: still in-flight → status=in_progress + a recommended
// next-check backoff (unless Wait, which blocks server-side until terminal);
// terminal but uncached (finalize lost to a crash) → recompute once on demand
// and cache it. The returned CachedDiff.Status is one of in_progress | ready.
func (s *Service) GetDiffResult(ctx context.Context, req GetDiffResultRequest) (CachedDiff, int, error) {
	if req.RunID == "" && req.Latest {
		intent, ok, err := s.storage.LatestDiffIntent(req.RepoID, req.Scenario, req.Branch, req.Name)
		if err != nil {
			return CachedDiff{}, 0, err
		}
		if !ok {
			return CachedDiff{}, 0, fmt.Errorf("no diff run found for baseline %q (start one with `baseline diff --scenario %s --name %s`)", req.Name, req.Scenario, req.Name)
		}
		req.RunID = intent.CurRunID
	}
	if strings.TrimSpace(req.RunID) == "" {
		return CachedDiff{}, 0, fmt.Errorf("run id is required")
	}
	cached, ok, err := s.storage.LoadDiffResult(req.RepoID, req.Scenario, req.Branch, req.Name, req.RunID)
	if err != nil {
		return CachedDiff{}, 0, err
	}
	if ok {
		return cached, 0, nil
	}

	st, serr := s.exec.RunStatus(ctx, req.Scenario, req.RunID)
	if serr != nil {
		return CachedDiff{}, 0, serr
	}
	if !st.Terminal && !req.Wait {
		return CachedDiff{Status: "in_progress", RunID: req.RunID}, st.RecommendedNextCheckSeconds, nil
	}

	// Either the run is terminal but uncached (the finalize tail was lost to a
	// crash/restart), or Wait was requested and we block server-side via
	// AwaitResult until terminal. Recompute once on demand; the runs are durable.
	manifest, err := s.Get(ctx, req.RepoID, req.Scenario, req.Branch, req.Name)
	if err != nil {
		return CachedDiff{}, 0, err
	}
	cur, gerr := s.captureGit(ctx, req.RepoDir)
	if gerr != nil {
		cur = git.State{}
	}
	stale, _ := ComputeStaleness(ctx, s.probe, req.RepoDir, manifest.Git.Sha)
	_, awaitErr := s.exec.AwaitResult(ctx, req.Scenario, req.RunID)
	diff := s.computeDiff(ctx, manifest, cur, stale, manifest.RunID(), req.RunID, awaitErr)
	cd := CachedDiff{Status: "ready", Result: &diff, RunID: req.RunID, ComputedAt: s.now().UTC()}
	_ = s.storage.SaveDiffResult(req.RepoID, req.Scenario, req.Branch, req.Name, req.RunID, cd)
	return cd, 0, nil
}

// computeDiff issues one unfiltered CompareRuns and preserves Test Genie's
// complete phase sequence verbatim. Artifact catalogs are fetched by run ID and
// remain opaque; GCT never reconstructs provider paths.
func (s *Service) computeDiff(ctx context.Context, manifest BaselineManifest, cur git.State, stale Staleness, baseRunID, curRunID string, runErr error) DiffResult {
	res := DiffResult{
		Manifest: manifest, CurrentGit: cur, Staleness: stale, Verdict: VerdictClean,
		Evidence: EvidenceComparison{BaseRunID: baseRunID, CurrentRunID: curRunID},
	}
	if cur.Dirty {
		res.DirtyWarning = fmt.Sprintf("working tree is dirty (%s) — failures may be caused by uncommitted changes rather than the diff itself", cur.DirtySummary)
	}
	if manifest.Migration != nil {
		res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, manifest.Migration.DegradedReasons...)
	}
	switch {
	case baseRunID == "":
		res.Verdict = VerdictNotComparable
		res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "baseline has no comparable run")
	case runErr != nil:
		res.Verdict = VerdictNotComparable
		res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "could not run current comprehensive suite: "+runErr.Error())
	case curRunID == "":
		res.Verdict = VerdictNotComparable
		res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "current run produced no run id")
	default:
		cmp, cmpErr := s.runs.CompareRuns(ctx, manifest.Scenario, baseRunID, curRunID, "")
		if cmpErr != nil {
			res.Verdict = VerdictNotComparable
			res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "compare failed: "+cmpErr.Error())
		} else {
			comparison := cmp.Comparison
			if comparison == nil {
				comparison = &runspb.CompareRunsResponse{Verdict: cmp.Verdict, Phases: cmp.Phases, SchemaVersion: 1, Behavior: "unknown", Coverage: "legacy", Compatibility: "legacy", Provenance: "legacy"}
			}
			res.Comparison = comparison
			res.Phases = append(res.Phases, comparison.GetPhases()...)
			res.Verdict = GateDecisionForComparison(comparison).LegacyVerdict
		}
		res.Evidence.BaseCatalog = s.loadArtifactCatalog(ctx, manifest.Scenario, baseRunID, "baseline", &res)
		res.Evidence.CurrentCatalog = s.loadArtifactCatalog(ctx, manifest.Scenario, curRunID, "current", &res)
		baseHasVisual := catalogHasVisualEvidence(res.Evidence.BaseCatalog)
		currentHasVisual := catalogHasVisualEvidence(res.Evidence.CurrentCatalog)
		// The artifact catalog, rather than the capture-profile label, is the
		// evidence authority here. This permits an ordinary reused run to satisfy
		// content comparison while still failing closed whenever the immutable
		// baseline actually contains visual evidence that must be compared.
		visualRunID := curRunID
		if baseHasVisual && !currentHasVisual {
			// A normal run may be content-complete but lack the optional visual
			// capture depth required by an older baseline. Give the owning
			// provider one explicit capture-only opportunity. The provider must
			// return a separate evidence run; it may not silently re-execute the
			// comprehensive suite. If that seam is unavailable or cannot produce
			// the requested evidence, fail closed as not-comparable.
			if capturer, ok := s.runs.(MissingEvidenceCapturer); ok {
				capturedRunID, captureErr := capturer.CaptureMissingEvidence(ctx, manifest.Scenario, curRunID, []string{"visual"})
				if captureErr != nil {
					res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "required current visual evidence capture failed: "+captureErr.Error())
				} else if strings.TrimSpace(capturedRunID) == "" {
					res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "required current visual evidence capture returned no run id")
				} else {
					visualRunID = capturedRunID
					res.Evidence.CurrentCatalog = s.loadArtifactCatalog(ctx, manifest.Scenario, visualRunID, "current visual capture", &res)
					currentHasVisual = catalogHasVisualEvidence(res.Evidence.CurrentCatalog)
				}
			} else {
				res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "required current visual evidence is unavailable")
			}
		}
		if baseHasVisual && currentHasVisual {
			if deltas, err := s.runs.CompareRunVisuals(ctx, manifest.Scenario, baseRunID, visualRunID); err != nil {
				res.Evidence.BlockingReasons = append(res.Evidence.BlockingReasons, "required visual evidence comparison is unavailable: "+err.Error())
			} else {
				res.Evidence.VisualDeltas = append(res.Evidence.VisualDeltas, deltas...)
			}
		}
	}
	if len(res.Evidence.BlockingReasons) > 0 {
		res.Verdict = VerdictNotComparable
	}
	res.Evidence.DegradedReasons = append(res.Evidence.DegradedReasons, res.Evidence.BlockingReasons...)
	res.Evidence.DegradedReasons = append(res.Evidence.DegradedReasons, res.Evidence.AdvisoryWarnings...)
	if len(res.Evidence.BlockingReasons) > 0 || len(res.Evidence.AdvisoryWarnings) > 0 {
		res.Evidence.EvidenceStatus = "incomplete"
	} else {
		res.Evidence.EvidenceStatus = "complete"
	}
	return res
}

func catalogHasVisualEvidence(catalog ArtifactCatalog) bool {
	for _, artifact := range catalog.Artifacts {
		if artifact == nil {
			continue
		}
		kind := strings.ToLower(strings.TrimSpace(artifact.GetKind()))
		if strings.Contains(kind, "visual") || strings.Contains(kind, "screenshot") || strings.Contains(kind, "video") {
			return true
		}
	}
	return false
}

func (s *Service) loadArtifactCatalog(ctx context.Context, scenario, runID, label string, res *DiffResult) ArtifactCatalog {
	catalog, err := s.runs.ListRunArtifacts(ctx, scenario, runID)
	if err != nil {
		res.Evidence.AdvisoryWarnings = append(res.Evidence.AdvisoryWarnings, label+" artifact catalog unavailable: "+err.Error())
		return ArtifactCatalog{RunID: runID}
	}
	if len(catalog.DegradedReasons) > 0 {
		for _, reason := range catalog.DegradedReasons {
			res.Evidence.AdvisoryWarnings = append(res.Evidence.AdvisoryWarnings, label+" artifact catalog: "+reason)
		}
	}
	return catalog
}

// shortSha truncates a git sha to its 8-char display form.
func shortSha(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}

// Delete removes a baseline and unpins its comprehensive run.
func (s *Service) Delete(ctx context.Context, repoID int64, scenario, branch, name string) error {
	manifest, err := s.storage.Load(repoID, scenario, branch, name)
	if err != nil {
		return err
	}
	// Retire first: a concurrent finalizer must see the tombstone and never
	// recreate the just-deleted baseline if unpinning is interrupted.
	if err := s.setLifecycle(repoID, manifest, LifecycleDeleted, manifest.RunID(), "deleted by operator"); err != nil {
		return err
	}
	if err := s.unpinRun(ctx, manifest); err != nil {
		return fmt.Errorf("release baseline %q retention pin: %w", manifest.Name, err)
	}
	return s.storage.Delete(repoID, scenario, branch, name)
}

// unpinRun releases the baseline's single run claim.
func (s *Service) unpinRun(ctx context.Context, m BaselineManifest) error {
	if s.runs == nil {
		return fmt.Errorf("Test Genie runs client is unavailable")
	}
	runID := m.RunID()
	if runID == "" {
		return fmt.Errorf("baseline has no retained run")
	}
	return s.runs.UnpinRun(ctx, m.Scenario, runID, PinOwner(m.Name))
}
