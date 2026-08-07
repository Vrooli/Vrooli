package baseline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// CollectionTarget is a caller-selected per-scenario member. Selection is
// explicit; collections do not discover scenarios from repository paths.
type CollectionTarget struct {
	Scenario     string
	BaselineName string
	Required     bool
}

type StartCollectionCaptureRequest struct {
	RepoID  int64
	RepoDir string
	Branch  string
	Name    string
	Targets []CollectionTarget
	// PathSelections are optional scoped source evidence. They are captured
	// once at collection creation and remain informational, never behavioral.
	PathSelections      []string
	PathPolicy          PathSnapshotPolicy
	CreatedBy           string
	Reason              string
	AcknowledgeReanchor bool
}

type CapturePathSnapshotRequest struct {
	RepoID     int64
	RepoDir    string
	Branch     string
	Name       string
	Selections []string
	Retention  time.Duration
	Policy     PathSnapshotPolicy
}

type CapturePathSnapshotResult struct {
	Snapshot PathSnapshot
	Resumed  bool
}

// PendingCollectionCapture is handed to a server-owned finalizer. Persisting
// the member run id before returning makes attachment loss recoverable.
type PendingCollectionCapture struct {
	CollectionName string
	Branch         string
	Scenario       string
	Pending        PendingCapture
}

type StartCollectionCaptureResult struct {
	Collection CollectionManifest
	Pending    []PendingCollectionCapture
	Resumed    bool
}

// CollectionIncompleteError is the typed detached outcome for a bounded wait.
// Durable member state remains authoritative and PendingRunIDs are exact
// recovery handles; callers must not translate this outcome to success.
type CollectionIncompleteError struct {
	PendingRunIDs []string
	Cause         error
}

func (e *CollectionIncompleteError) Error() string {
	return fmt.Sprintf("collection wait incomplete; pending run ids: %s: %v", strings.Join(e.PendingRunIDs, ", "), e.Cause)
}

func (e *CollectionIncompleteError) Unwrap() error { return e.Cause }

// ExtendCollectionRequest only permits append-only expansion. A new member's
// immutable before-state must be captured before that scenario is edited.
type ExtendCollectionRequest struct {
	RepoID    int64
	RepoDir   string
	Branch    string
	Name      string
	Targets   []CollectionTarget
	CreatedBy string
	Reason    string
}

type ExtendCollectionResult struct {
	Collection     CollectionManifest
	Pending        []PendingCollectionCapture
	AddedScenarios []string
	Resumed        bool
}

// StartCollectionDiffRequest starts durable child diffs for an explicit subset
// of a captured collection. Empty Scenarios means all members. Callers must
// pass their policy explicitly; GCT validates membership but never infers a
// phase scope from changed files.
type StartCollectionDiffRequest struct {
	RepoID      int64
	RepoDir     string
	Branch      string
	Name        string
	OperationID string
	Scenarios   []string
}

type PendingCollectionDiff struct {
	CollectionName string
	OperationID    string
	Scenario       string
	Pending        PendingDiff
}

const (
	collectionDiffDispatchLease       = 30 * time.Second
	maxCollectionDiffDispatchAttempts = 3
)

// Test Genie can reject a caller temporarily when its durable-run admission
// queue is full. That is backpressure, not a child-dispatch defect: terminally
// failing the collection after three fast retries turns a busy test service
// into a false regression. Keep the member pending and let the durable lease
// retry it after capacity advances.
func isTransientAdmissionSaturation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "resource_exhausted") &&
		(strings.Contains(message, "admission") || strings.Contains(message, "queued run capacity")) {
		return true
	}
	return strings.Contains(message, "wait for test-genie admission") &&
		strings.Contains(message, "context deadline exceeded")
}

type StartCollectionDiffResult struct {
	Collection CollectionManifest
	Operation  CollectionDiffOperation
	Members    []CollectionDiffMember
	Pending    []PendingCollectionDiff
}

// StartCollectionCapture creates (or resumes) one durable collection, then
// starts at most one existing baseline snapshot per pending member. It never
// waits for Test Genie, and a member failure is recorded on that member rather
// than hiding the rest of the collection behind a single operation error.
func (s *Service) StartCollectionCapture(ctx context.Context, req StartCollectionCaptureRequest) (StartCollectionCaptureResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return StartCollectionCaptureResult{}, fmt.Errorf("collection name is required")
	}
	if len(req.Targets) == 0 {
		return StartCollectionCaptureResult{}, fmt.Errorf("collection requires at least one target")
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		state, err := s.captureGit(ctx, req.RepoDir)
		if err != nil {
			return StartCollectionCaptureResult{}, fmt.Errorf("read git state: %w", err)
		}
		branch = ResolveStorageBranch(state)
	}
	collection := newCollectionManifest(req, branch, s.now().UTC())
	resumed := false
	if existing, err := s.storage.LoadCollection(req.RepoID, branch, req.Name); err == nil {
		if req.AcknowledgeReanchor {
			if existing.Coverage().Failed == 0 {
				return StartCollectionCaptureResult{}, fmt.Errorf("collection %q has no failed member to re-anchor", req.Name)
			}
			collection = newCollectionManifest(req, branch, s.now().UTC())
			collection.CreatedAt, collection.Generation, collection.Reanchored = existing.CreatedAt, existing.Generation+1, true
			if collection.Generation < 2 {
				collection.Generation = 2
			}
			collection.ReanchorDetail = "re-anchored at current source state after prior immutable capture failed"
			for i := range collection.Members {
				collection.Members[i].BaselineName = fmt.Sprintf("%s__g%d", collection.Name, collection.Generation)
			}
			if err := s.storage.SaveCollection(req.RepoID, collection, Overwrite); err != nil {
				return StartCollectionCaptureResult{}, err
			}
		} else {
			if err := compatibleCollectionSelection(existing, collection); err != nil {
				return StartCollectionCaptureResult{}, err
			}
			if len(req.PathSelections) > 0 {
				snapshot, err := s.storage.LoadPathSnapshot(req.RepoID, branch, req.Name)
				if err != nil {
					return StartCollectionCaptureResult{}, fmt.Errorf("load collection source evidence: %w", err)
				}
				requested, err := normalizeSnapshotSelections(req.PathSelections)
				if err != nil {
					return StartCollectionCaptureResult{}, err
				}
				if !sameStrings(snapshot.Selections, requested) {
					return StartCollectionCaptureResult{}, fmt.Errorf("collection %q already exists with different source evidence selection", req.Name)
				}
			}
			collection, resumed = existing, true
		}
	} else if errors.Is(err, ErrNotFound) {
		if len(req.PathSelections) > 0 {
			captured, err := s.CapturePathSnapshot(ctx, CapturePathSnapshotRequest{RepoID: req.RepoID, RepoDir: req.RepoDir, Branch: branch, Name: req.Name, Selections: req.PathSelections, Policy: req.PathPolicy})
			if err != nil {
				return StartCollectionCaptureResult{}, fmt.Errorf("capture collection source evidence: %w", err)
			}
			collection.PathSnapshots = []string{captured.Snapshot.Name}
		}
		if err := s.storage.SaveCollection(req.RepoID, collection, CreateOnly); err != nil {
			return StartCollectionCaptureResult{}, err
		}
	} else {
		return StartCollectionCaptureResult{}, err
	}

	var pending []PendingCollectionCapture
	for _, member := range collection.Members {
		if member.Status != CollectionMemberPending || member.RunID != "" {
			continue
		}
		// A collection is only an immutable index over per-scenario baselines.
		// If a previous collection capture was discarded after a partial start,
		// its successfully captured child baseline remains the authoritative
		// before-state and must be reusable. Starting another run here would fail
		// with ErrAlreadyExists and turn valid evidence into a collection failure.
		if existing, getErr := s.Get(ctx, req.RepoID, member.Scenario, branch, member.BaselineName); getErr == nil {
			updated, updateErr := s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(target *CollectionMember) error {
				target.Status = CollectionMemberReady
				target.RunID = existing.RunID()
				target.GitSHA = existing.Git.Sha
				target.Error = "reused immutable baseline"
				target.UpdatedAt = s.now().UTC()
				return nil
			})
			if updateErr != nil {
				return StartCollectionCaptureResult{}, updateErr
			}
			collection = updated
			continue
		} else if !errors.Is(getErr, ErrNotFound) {
			return StartCollectionCaptureResult{}, fmt.Errorf("load existing baseline for collection member %s: %w", member.Scenario, getErr)
		}
		started, err := s.StartCapture(ctx, CreateRequest{
			RepoID: req.RepoID, RepoDir: req.RepoDir, Scenario: member.Scenario,
			Name: member.BaselineName, Branch: branch, CreatedBy: req.CreatedBy, Reason: req.Reason,
		})
		if err != nil {
			if isTransientAdmissionSaturation(err) {
				// Keep the member pending. A collection capture can fill the
				// caller's Test Genie queue before the final member is admitted;
				// that is durable backpressure, not a terminal baseline failure.
				collection, err = s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(m *CollectionMember) error {
					m.Error, m.UpdatedAt = "deferred: "+err.Error(), s.now().UTC()
					return nil
				})
				if err != nil {
					return StartCollectionCaptureResult{}, err
				}
				continue
			}
			collection, err = s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(m *CollectionMember) error {
				m.Status, m.Error, m.UpdatedAt = CollectionMemberFailed, err.Error(), s.now().UTC()
				return nil
			})
			if err != nil {
				return StartCollectionCaptureResult{}, err
			}
			continue
		}
		collection, err = s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(m *CollectionMember) error {
			m.RunID, m.UpdatedAt = started.Run.RunID, s.now().UTC()
			return nil
		})
		if err != nil {
			return StartCollectionCaptureResult{}, err
		}
		pending = append(pending, PendingCollectionCapture{CollectionName: req.Name, Branch: branch, Scenario: member.Scenario, Pending: started})
	}
	return StartCollectionCaptureResult{Collection: collection, Pending: pending, Resumed: resumed}, nil
}

// StartDeferredCollectionCapture admits one pending member after a sibling
// child completes. Capture fanout can exhaust Test Genie's per-caller queue;
// the member remains pending until a finalizer advances capacity and invokes
// this server-owned handoff.
func (s *Service) StartDeferredCollectionCapture(ctx context.Context, repoID int64, pending PendingCollectionCapture) (PendingCollectionCapture, bool, error) {
	collection, err := s.storage.LoadCollection(repoID, pending.Branch, pending.CollectionName)
	if err != nil {
		return PendingCollectionCapture{}, false, err
	}
	// A required member failure terminalizes the collection. Do not keep
	// draining deferred work after that point: doing so turns a failed capture
	// into an unbounded stream of new Test Genie runs after callers have already
	// detached or aborted the parent operation.
	coverage := collection.Coverage()
	if coverage.Failed > 0 || coverage.Skipped > 0 || coverage.Stale > 0 {
		return PendingCollectionCapture{}, false, nil
	}
	for _, member := range collection.Members {
		if member.Status != CollectionMemberPending || member.RunID != "" {
			continue
		}
		started, startErr := s.StartCapture(ctx, CreateRequest{
			RepoID: repoID, RepoDir: pending.Pending.Req.RepoDir, Scenario: member.Scenario,
			Name: member.BaselineName, Branch: pending.Branch, CreatedBy: pending.Pending.Req.CreatedBy, Reason: pending.Pending.Req.Reason,
		})
		if startErr != nil {
			if isTransientAdmissionSaturation(startErr) {
				return PendingCollectionCapture{}, false, nil
			}
			_, updateErr := s.storage.UpdateCollectionMember(repoID, pending.Branch, pending.CollectionName, member.Scenario, func(target *CollectionMember) error {
				target.Status, target.Error, target.UpdatedAt = CollectionMemberFailed, startErr.Error(), s.now().UTC()
				return nil
			})
			if updateErr != nil {
				return PendingCollectionCapture{}, false, updateErr
			}
			return PendingCollectionCapture{}, false, startErr
		}
		_, updateErr := s.storage.UpdateCollectionMember(repoID, pending.Branch, pending.CollectionName, member.Scenario, func(target *CollectionMember) error {
			target.RunID, target.Error, target.UpdatedAt = started.Run.RunID, "", s.now().UTC()
			return nil
		})
		if updateErr != nil {
			return PendingCollectionCapture{}, false, updateErr
		}
		return PendingCollectionCapture{CollectionName: pending.CollectionName, Branch: pending.Branch, Scenario: member.Scenario, Pending: started}, true, nil
	}
	return PendingCollectionCapture{}, false, nil
}

// ExtendCollection adds only previously unknown scenarios to an existing
// collection. It never alters existing member identity, status, or source
// evidence; callers must use an explicit degraded/repair path if a scenario
// was already changed before its baseline could be captured.
func (s *Service) ExtendCollection(ctx context.Context, req ExtendCollectionRequest) (ExtendCollectionResult, error) {
	if strings.TrimSpace(req.Name) == "" || len(req.Targets) == 0 {
		return ExtendCollectionResult{}, fmt.Errorf("collection name and at least one new target are required")
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		state, err := s.captureGit(ctx, req.RepoDir)
		if err != nil {
			return ExtendCollectionResult{}, fmt.Errorf("read git state: %w", err)
		}
		branch = ResolveStorageBranch(state)
	}
	collection, err := s.storage.AppendCollectionMembers(req.RepoID, branch, req.Name, req.Targets, s.now().UTC())
	if err != nil {
		return ExtendCollectionResult{}, err
	}
	result := ExtendCollectionResult{Collection: collection}
	for _, target := range req.Targets {
		result.AddedScenarios = append(result.AddedScenarios, strings.TrimSpace(target.Scenario))
	}
	for _, member := range collection.Members {
		added := false
		for _, scenario := range result.AddedScenarios {
			if member.Scenario == scenario {
				added = true
				break
			}
		}
		if !added || member.Status != CollectionMemberPending || member.RunID != "" {
			continue
		}
		started, startErr := s.StartCapture(ctx, CreateRequest{RepoID: req.RepoID, RepoDir: req.RepoDir, Scenario: member.Scenario, Name: member.BaselineName, Branch: branch, CreatedBy: req.CreatedBy, Reason: req.Reason})
		if startErr != nil {
			collection, err = s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(m *CollectionMember) error {
				m.Status, m.Error, m.UpdatedAt = CollectionMemberFailed, startErr.Error(), s.now().UTC()
				return nil
			})
			if err != nil {
				return ExtendCollectionResult{}, err
			}
			continue
		}
		collection, err = s.storage.UpdateCollectionMember(req.RepoID, branch, req.Name, member.Scenario, func(m *CollectionMember) error { m.RunID, m.UpdatedAt = started.Run.RunID, s.now().UTC(); return nil })
		if err != nil {
			return ExtendCollectionResult{}, err
		}
		result.Pending = append(result.Pending, PendingCollectionCapture{CollectionName: req.Name, Branch: branch, Scenario: member.Scenario, Pending: started})
	}
	result.Collection = collection
	return result, nil
}

func newCollectionManifest(req StartCollectionCaptureRequest, branch string, now time.Time) CollectionManifest {
	members := make([]CollectionMember, 0, len(req.Targets))
	for _, target := range req.Targets {
		members = append(members, CollectionMember{Scenario: strings.TrimSpace(target.Scenario), BaselineName: strings.TrimSpace(target.BaselineName), Required: target.Required, Status: CollectionMemberPending, UpdatedAt: now})
	}
	paths := []string(nil)
	if len(req.PathSelections) > 0 {
		paths = []string{strings.TrimSpace(req.Name)}
	}
	return CollectionManifest{Name: strings.TrimSpace(req.Name), Branch: branch, CreatedAt: now, UpdatedAt: now, SchemaVersion: CollectionSchemaVersion, Generation: 1, Members: members, PathSnapshots: paths}.Normalized()
}

// CapturePathSnapshot creates or resumes one immutable, branch-scoped source
// evidence manifest. It reads the working tree directly, so uncommitted dirty
// content is represented exactly when policy permits it.
func (s *Service) CapturePathSnapshot(_ context.Context, req CapturePathSnapshotRequest) (CapturePathSnapshotResult, error) {
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		return CapturePathSnapshotResult{}, fmt.Errorf("path snapshot branch is required")
	}
	if _, err := s.storage.SweepExpiredPathSnapshots(req.RepoID, s.now().UTC()); err != nil {
		return CapturePathSnapshotResult{}, fmt.Errorf("sweep expired path snapshots: %w", err)
	}
	if existing, err := s.storage.LoadPathSnapshot(req.RepoID, branch, req.Name); err == nil {
		return CapturePathSnapshotResult{Snapshot: existing, Resumed: true}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return CapturePathSnapshotResult{}, err
	}
	lease := req.Retention
	if lease <= 0 {
		lease = defaultPathSnapshotLease
	}
	snapshot, objects, err := CapturePathSnapshotWithPolicyAndLease(req.RepoDir, req.Name, branch, req.Selections, req.Policy, s.now().UTC(), lease)
	if err != nil {
		return CapturePathSnapshotResult{}, err
	}
	if err := s.storage.SavePathSnapshot(req.RepoID, snapshot, objects, CreateOnly); err != nil {
		if errors.Is(err, ErrAlreadyExists) {
			return s.CapturePathSnapshot(context.Background(), req)
		}
		return CapturePathSnapshotResult{}, err
	}
	return CapturePathSnapshotResult{Snapshot: snapshot}, nil
}

// EstimatePathSnapshot is intentionally a thin service seam so capture and
// callers share the resolver rather than duplicating Git or glob policy.
func (s *Service) EstimatePathSnapshot(_ context.Context, repoDir string, selections []string, policy PathSnapshotPolicy) (PathSnapshotEstimate, error) {
	return EstimatePathSnapshot(repoDir, selections, policy)
}

func (s *Service) StorageLoadPathSnapshot(repoID int64, branch, name string) (PathSnapshot, error) {
	return s.storage.LoadPathSnapshot(repoID, branch, name)
}

func (s *Service) DeletePathSnapshot(_ context.Context, repoID int64, branch, name string) error {
	return s.storage.DeletePathSnapshot(repoID, branch, name)
}

func compatibleCollectionSelection(existing, proposed CollectionManifest) error {
	if existing.Name != proposed.Name || existing.Branch != proposed.Branch || len(existing.Members) != len(proposed.Members) || !sameStrings(existing.PathSnapshots, proposed.PathSnapshots) {
		return fmt.Errorf("collection %q already exists with a different target selection", proposed.Name)
	}
	for i, member := range existing.Normalized().Members {
		want := proposed.Normalized().Members[i]
		if member.Scenario != want.Scenario || member.BaselineName != want.BaselineName || member.Required != want.Required {
			return fmt.Errorf("collection %q already exists with a different target selection", proposed.Name)
		}
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// FinalizeCollectionCapture converts one durable child run into its immutable
// baseline then transitions only that member. Cancellation leaves it pending so
// the caller can safely reattach; terminal errors are visible as failed member
// coverage and cannot be mistaken for clean.
func (s *Service) FinalizeCollectionCapture(ctx context.Context, repoID int64, pending PendingCollectionCapture) (CollectionManifest, error) {
	result, err := s.FinalizeCapture(ctx, pending.Pending)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			collection, loadErr := s.storage.LoadCollection(repoID, pending.Branch, pending.CollectionName)
			if loadErr != nil {
				return CollectionManifest{}, loadErr
			}
			return collection, err
		}
		collection, updateErr := s.storage.UpdateCollectionMember(repoID, pending.Branch, pending.CollectionName, pending.Scenario, func(member *CollectionMember) error {
			member.Status, member.Error, member.UpdatedAt = CollectionMemberFailed, err.Error(), s.now().UTC()
			return nil
		})
		if updateErr != nil {
			return CollectionManifest{}, updateErr
		}
		// The member transition is durable evidence of this terminal failure.
		// Return it with the error so aggregate callers can keep processing
		// siblings instead of leaving the collection pending forever.
		return collection, err
	}
	return s.storage.UpdateCollectionMember(repoID, pending.Branch, pending.CollectionName, pending.Scenario, func(member *CollectionMember) error {
		member.Status, member.RunID, member.GitSHA, member.Error, member.UpdatedAt = CollectionMemberReady, result.Manifest.RunID(), result.Manifest.Git.Sha, "", s.now().UTC()
		return nil
	})
}

// ResumeCollectionCapture reattaches to durable child snapshot intents. It is
// intentionally server-owned: callers can make one bounded wait request rather
// than polling or reconstructing individual Test Genie handles themselves.
func (s *Service) ResumeCollectionCapture(ctx context.Context, repoID int64, branch, name string) (CollectionManifest, error) {
	collection, err := s.storage.LoadCollection(repoID, branch, name)
	if err != nil {
		return CollectionManifest{}, err
	}
	type pendingJob struct {
		member CollectionMember
		intent SnapshotIntent
	}
	var jobs []pendingJob
	seen := make(map[string]struct{})
	for _, member := range collection.Members {
		if member.Status != CollectionMemberPending || member.RunID == "" {
			continue
		}
		intent, found, err := s.storage.LoadSnapshotIntent(repoID, member.Scenario, branch, member.BaselineName, member.RunID)
		if err != nil {
			return CollectionManifest{}, err
		}
		if !found {
			// A crash can occur after the child manifest commits but before the
			// collection member projection commits. The child lifecycle is
			// authoritative: repair this eligible projection instead of turning a
			// recoverable duplicate/orphan into terminal collection failure.
			manifest, getErr := s.Get(ctx, repoID, member.Scenario, branch, member.BaselineName)
			if getErr == nil {
				collection, err = s.storage.UpdateCollectionMember(repoID, branch, name, member.Scenario, func(target *CollectionMember) error {
					target.Status, target.RunID, target.GitSHA, target.Error, target.UpdatedAt = CollectionMemberReady, manifest.RunID(), manifest.Git.Sha, "reconciled from durable child lifecycle", s.now().UTC()
					return nil
				})
				if err != nil {
					return CollectionManifest{}, err
				}
				continue
			}
			if getErr != nil && !errors.Is(getErr, ErrNotFound) {
				return CollectionManifest{}, getErr
			}
			collection, err = s.storage.UpdateCollectionMember(repoID, branch, name, member.Scenario, func(target *CollectionMember) error {
				target.Status, target.Error, target.UpdatedAt = CollectionMemberFailed, "durable snapshot intent is missing", s.now().UTC()
				return nil
			})
			if err != nil {
				return CollectionManifest{}, err
			}
			continue
		}
		key := member.Scenario + "\x00" + member.RunID
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		jobs = append(jobs, pendingJob{member: member, intent: intent})
	}

	const maxConcurrentCollectionWaits = 8
	sem := make(chan struct{}, maxConcurrentCollectionWaits)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			updated, finalizeErr := s.FinalizeCollectionCapture(ctx, repoID, PendingCollectionCapture{CollectionName: name, Branch: branch, Scenario: job.member.Scenario, Pending: job.intent.PendingCapture()})
			mu.Lock()
			defer mu.Unlock()
			if updated.Name != "" {
				collection = updated
			}
			if finalizeErr != nil && firstErr == nil {
				firstErr = finalizeErr
			}
		}()
	}
	wg.Wait()
	collection, loadErr := s.storage.LoadCollection(repoID, branch, name)
	if loadErr != nil {
		return CollectionManifest{}, loadErr
	}
	if errors.Is(firstErr, context.Canceled) || errors.Is(firstErr, context.DeadlineExceeded) || ctx.Err() != nil {
		pending := make([]string, 0)
		for _, member := range collection.Members {
			if member.Status == CollectionMemberPending && member.RunID != "" {
				pending = append(pending, member.RunID)
			}
		}
		cause := firstErr
		if cause == nil {
			cause = ctx.Err()
		}
		return collection, &CollectionIncompleteError{PendingRunIDs: pending, Cause: cause}
	}
	// Terminal child failures are already projected durably on their members;
	// aggregate coverage is the truthful response, not an opaque parent error.
	return collection, nil
}

// GetCollectionCaptureStatus is the non-blocking read/reconciliation path for
// collection capture. A Test Genie run is the execution authority, so a
// terminal child must be projected on every status read even when the original
// asynchronous finalizer was lost to a restart. Unlike ResumeCollectionCapture
// this never waits for an active child.
func (s *Service) GetCollectionCaptureStatus(ctx context.Context, repoID int64, branch, name string) (CollectionManifest, error) {
	collection, err := s.storage.LoadCollection(repoID, branch, name)
	if err != nil {
		return CollectionManifest{}, err
	}
	for _, member := range collection.Members {
		if member.Status != CollectionMemberPending || member.RunID == "" {
			continue
		}
		status, statusErr := s.GetSnapshotStatus(ctx, SnapshotStatusRequest{
			RepoID: repoID, Scenario: member.Scenario, Branch: branch, Name: member.BaselineName, RunID: member.RunID,
		})
		if statusErr != nil || status.Status == "pending" {
			// Status transport failures are recoverable. Preserve the durable
			// pending member; the periodic reconciler and later reads retry.
			continue
		}
		switch status.Status {
		case "ready":
			if status.Baseline == nil {
				continue
			}
			collection, err = s.storage.UpdateCollectionMember(repoID, branch, name, member.Scenario, func(target *CollectionMember) error {
				target.Status, target.RunID, target.GitSHA, target.Error, target.UpdatedAt = CollectionMemberReady, status.Baseline.RunID(), status.Baseline.Git.Sha, "reconciled from terminal durable child", s.now().UTC()
				return nil
			})
		case "failed", "missing":
			detail := strings.TrimSpace(status.Error)
			if detail == "" {
				detail = "terminal snapshot child has no baseline result"
			}
			collection, err = s.storage.UpdateCollectionMember(repoID, branch, name, member.Scenario, func(target *CollectionMember) error {
				target.Status, target.Error, target.UpdatedAt = CollectionMemberFailed, detail, s.now().UTC()
				return nil
			})
		}
		if err != nil {
			return CollectionManifest{}, err
		}
	}
	return s.storage.LoadCollection(repoID, branch, name)
}

// ReconcileCollectionCaptures is the detached-client/restart backstop for
// capture parents. It complements collection-diff reconciliation: both parent
// types must project terminal Test Genie truth without a particular client
// remembering to issue a wait command.
func (s *Service) ReconcileCollectionCaptures(ctx context.Context, repoID int64) error {
	collections, err := s.storage.ListCollections(repoID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, collection := range collections {
		if collection.Coverage().Pending == 0 {
			continue
		}
		if _, reconcileErr := s.GetCollectionCaptureStatus(ctx, repoID, collection.Branch, collection.Name); reconcileErr != nil && firstErr == nil {
			firstErr = reconcileErr
		}
	}
	return firstErr
}

// StartCollectionDiff dispatches one existing per-scenario diff for every
// selected ready member. Individual StartDiff intents remain the durable child
// operation records; the returned handles make an interrupted aggregate wait
// resumable without any client-side orchestration guesswork.
func (s *Service) StartCollectionDiff(ctx context.Context, req StartCollectionDiffRequest) (StartCollectionDiffResult, error) {
	if strings.TrimSpace(req.OperationID) == "" {
		return StartCollectionDiffResult{}, fmt.Errorf("collection diff operation id is required")
	}
	collection, err := s.storage.LoadCollection(req.RepoID, req.Branch, req.Name)
	if err != nil {
		return StartCollectionDiffResult{}, err
	}
	selected, err := selectCollectionMembers(collection, req.Scenarios)
	if err != nil {
		return StartCollectionDiffResult{}, err
	}
	// Serialize local dispatch decisions. Cross-process callers still converge
	// through Test Genie's one-run-per-scenario coalescing and the durable
	// operation record below; this mutex eliminates duplicate starts within the
	// normal server process without holding the finalization mutex while waiting.
	s.collectionDiffMu.Lock()
	defer s.collectionDiffMu.Unlock()
	if existing, err := s.storage.LoadCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID); err == nil {
		if !sameCollectionDiffSelection(existing.Members, selected) {
			return StartCollectionDiffResult{}, fmt.Errorf("collection diff operation %q already exists with a different member selection", req.OperationID)
		}
		return s.dispatchCollectionDiff(ctx, req, collection, existing)
	} else if !errors.Is(err, ErrNotFound) {
		return StartCollectionDiffResult{}, err
	}
	operation := CollectionDiffOperation{
		ID:                 req.OperationID,
		Collection:         req.Name,
		Branch:             req.Branch,
		RepoDir:            req.RepoDir,
		CollectionSnapshot: collection,
		CreatedAt:          s.now().UTC(),
		UpdatedAt:          s.now().UTC(),
		Lifecycle:          "dispatching",
		LastProgressAt:     s.now().UTC(),
	}
	for _, member := range selected {
		state := CollectionDiffMember{Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required, Status: string(member.Status), Detail: member.Error}
		if member.Status == CollectionMemberReady {
			// The parent graph is committed before dispatch. A ready baseline has
			// no current diff run yet, so it is pending dispatch rather than
			// incorrectly carrying its immutable baseline run id as a child handle.
			state.Status = "pending"
			state.Lifecycle = CollectionDiffChildDispatching
		}
		operation.Members = append(operation.Members, state)
	}
	if err := s.storage.SaveCollectionDiffOperation(req.RepoID, operation, CreateOnly); err != nil {
		if !errors.Is(err, ErrAlreadyExists) {
			return StartCollectionDiffResult{}, err
		}
		// Another agent won creation between our initial read and write. Reload
		// the authoritative graph and resume it instead of surfacing a spurious
		// conflict to a retrying caller.
		existing, loadErr := s.storage.LoadCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID)
		if loadErr != nil {
			return StartCollectionDiffResult{}, loadErr
		}
		if !sameCollectionDiffSelection(existing.Members, selected) {
			return StartCollectionDiffResult{}, fmt.Errorf("collection diff operation %q already exists with a different member selection", req.OperationID)
		}
		return s.dispatchCollectionDiff(ctx, req, collection, existing)
	}
	return s.dispatchCollectionDiff(ctx, req, collection, operation)
}

// dispatchCollectionDiff resumes an already-persisted child graph. Every child
// run id is checkpointed immediately after StartDiff, so a crash can lose only
// the current attachment, never the work identity needed by another agent.
// The caller holds collectionDiffMu while choosing unassigned children.
func (s *Service) dispatchCollectionDiff(ctx context.Context, req StartCollectionDiffRequest, collection CollectionManifest, operation CollectionDiffOperation) (StartCollectionDiffResult, error) {
	// Older operation records predate RepoDir. A resumed explicit start has the
	// authoritative path, so checkpoint it before any handoff; later status
	// recovery then remains server-owned even if this caller detaches.
	if operation.RepoDir == "" && req.RepoDir != "" {
		var pathErr error
		operation, pathErr = s.storage.UpdateCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID, func(current *CollectionDiffOperation) error {
			if current.RepoDir == "" {
				current.RepoDir, current.UpdatedAt = req.RepoDir, s.now().UTC()
			}
			return nil
		})
		if pathErr != nil {
			return StartCollectionDiffResult{}, pathErr
		}
	}
	// Repair the short-lived pre-graph representation written by older servers:
	// its ready members were baseline references, not dispatched current runs.
	if operation.Lifecycle == "preparing" {
		var repairErr error
		operation, repairErr = s.storage.UpdateCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID, func(current *CollectionDiffOperation) error {
			if current.Lifecycle != "preparing" {
				return nil
			}
			for i := range current.Members {
				if current.Members[i].Status == "ready" {
					current.Members[i].Status, current.Members[i].RunID, current.Members[i].Lifecycle = "pending", "", CollectionDiffChildDispatching
				}
			}
			current.Lifecycle, current.UpdatedAt = "dispatching", s.now().UTC()
			return nil
		})
		if repairErr != nil {
			return StartCollectionDiffResult{}, repairErr
		}
	}
	result := StartCollectionDiffResult{Collection: collection}
	for i := range operation.Members {
		member := &operation.Members[i]
		if member.Status != "pending" {
			continue
		}
		if member.RunID == "" {
			now := s.now().UTC()
			if !member.DispatchLeaseExpiresAt.IsZero() && now.Before(member.DispatchLeaseExpiresAt) {
				continue
			}
			claimed := false
			claimedOperation, claimErr := s.storage.UpdateCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID, func(current *CollectionDiffOperation) error {
				for j := range current.Members {
					candidate := &current.Members[j]
					if candidate.Scenario != member.Scenario || candidate.Status != "pending" || candidate.RunID != "" {
						continue
					}
					if !candidate.DispatchLeaseExpiresAt.IsZero() && now.Before(candidate.DispatchLeaseExpiresAt) {
						return nil
					}
					candidate.DispatchLeaseExpiresAt, candidate.Lifecycle = now.Add(collectionDiffDispatchLease), CollectionDiffChildDispatching
					claimed = true
					break
				}
				return nil
			})
			if claimErr != nil {
				return StartCollectionDiffResult{}, claimErr
			}
			operation = claimedOperation
			if !claimed {
				continue
			}
			member = &operation.Members[i]
			started, err := s.StartDiff(ctx, StartDiffRequest{RepoID: req.RepoID, RepoDir: req.RepoDir, Branch: req.Branch, Scenario: member.Scenario, Name: member.BaselineName})
			outcomeStatus, outcomeDetail, outcomeRunID := member.Status, member.Detail, member.RunID
			outcomeAttempts, outcomeLease := member.DispatchAttempts, member.DispatchLeaseExpiresAt
			if err != nil {
				if isTransientAdmissionSaturation(err) {
					outcomeDetail = "dispatch deferred until Test Genie admission capacity is available: " + err.Error()
					outcomeLease = now.Add(collectionDiffDispatchLease)
				} else {
					outcomeAttempts++
					outcomeDetail = "dispatch attempt " + fmt.Sprint(outcomeAttempts) + ": " + err.Error()
					if outcomeAttempts >= maxCollectionDiffDispatchAttempts {
						outcomeStatus = "failed"
					} else {
						outcomeLease = now.Add(collectionDiffDispatchLease)
					}
				}
			} else {
				outcomeRunID, outcomeDetail, outcomeLease = started.RunID, "", time.Time{}
				result.Pending = append(result.Pending, PendingCollectionDiff{CollectionName: req.Name, OperationID: req.OperationID, Scenario: member.Scenario, Pending: started.Pending})
			}
			operation, err = s.storage.UpdateCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID, func(current *CollectionDiffOperation) error {
				for j := range current.Members {
					if current.Members[j].Scenario == member.Scenario {
						current.Members[j].Status, current.Members[j].Detail, current.Members[j].RunID, current.Members[j].DispatchAttempts, current.Members[j].DispatchLeaseExpiresAt = outcomeStatus, outcomeDetail, outcomeRunID, outcomeAttempts, outcomeLease
						if outcomeStatus == "failed" {
							current.Members[j].Lifecycle = CollectionDiffChildFailed
						} else if outcomeRunID != "" {
							current.Members[j].Lifecycle = CollectionDiffChildAwaiting
						} else {
							current.Members[j].Lifecycle = CollectionDiffChildDispatching
						}
					}
				}
				current.UpdatedAt, current.LastProgressAt = s.now().UTC(), s.now().UTC()
				current.Lifecycle = collectionDiffLifecycle(*current)
				return nil
			})
			if err != nil {
				return StartCollectionDiffResult{}, err
			}
			continue
		}
		intent, found, err := s.storage.LoadDiffIntent(req.RepoID, member.Scenario, req.Branch, member.BaselineName, member.RunID)
		if err != nil {
			return StartCollectionDiffResult{}, err
		}
		if !found {
			if err := s.failCollectionDiffMember(req.RepoID, req.Branch, req.Name, req.OperationID, member.Scenario, "durable child diff intent is missing"); err != nil {
				return StartCollectionDiffResult{}, err
			}
			operation, err = s.storage.LoadCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID)
			if err != nil {
				return StartCollectionDiffResult{}, err
			}
			continue
		}
		result.Pending = append(result.Pending, PendingCollectionDiff{CollectionName: req.Name, OperationID: req.OperationID, Scenario: member.Scenario, Pending: intent.PendingDiff()})
	}
	result.Operation = operation
	result.Members = append([]CollectionDiffMember(nil), operation.Members...)
	return result, nil
}

func sameCollectionDiffSelection(existing []CollectionDiffMember, selected []CollectionMember) bool {
	if len(existing) != len(selected) {
		return false
	}
	byScenario := make(map[string]CollectionMember, len(selected))
	for _, member := range selected {
		byScenario[member.Scenario] = member
	}
	for _, member := range existing {
		want, ok := byScenario[member.Scenario]
		if !ok || member.BaselineName != want.BaselineName || member.Required != want.Required {
			return false
		}
	}
	return true
}

func (s *Service) FinalizeCollectionDiff(ctx context.Context, repoID int64, pending PendingCollectionDiff) (CollectionDiffOperation, error) {
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, pending.Pending.Manifest.Branch, pending.CollectionName, pending.OperationID)
	if err != nil {
		return CollectionDiffOperation{}, err
	}
	_ = s.markCollectionDiffLifecycle(repoID, pending.Pending.Manifest.Branch, pending.CollectionName, pending.OperationID, pending.Scenario, CollectionDiffChildReconciling)
	result, err := s.FinalizeDiff(ctx, pending.Pending)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return operation, err // detached wait; leave durable member pending
		}
	}
	operation, saveErr := s.storage.UpdateCollectionDiffOperation(repoID, pending.Pending.Manifest.Branch, pending.CollectionName, pending.OperationID, func(operation *CollectionDiffOperation) error {
		for i := range operation.Members {
			if operation.Members[i].Scenario != pending.Scenario || operation.Members[i].Status != "pending" {
				continue
			}
			if err != nil {
				operation.Members[i].Status, operation.Members[i].Detail, operation.Members[i].Lifecycle = "failed", err.Error(), CollectionDiffChildFailed
				continue
			}
			operation.Members[i].Status, operation.Members[i].Verdict, operation.Members[i].Detail = "ready", result.Result.Verdict, collectionDiffDetail(result)
			switch result.Result.Verdict {
			case VerdictClean:
				operation.Members[i].Lifecycle = CollectionDiffChildPassed
			case VerdictNotComparable:
				operation.Members[i].Lifecycle = CollectionDiffChildNotComparable
			default:
				operation.Members[i].Lifecycle = CollectionDiffChildFailed
			}
		}
		operation.UpdatedAt, operation.LastProgressAt = s.now().UTC(), s.now().UTC()
		operation.Lifecycle = collectionDiffLifecycle(*operation)
		if err == nil {
			operation.LastReconciliationError = ""
		}
		return nil
	})
	if saveErr != nil {
		return CollectionDiffOperation{}, saveErr
	}
	return operation, err
}

// collectionDiffDetail retains the actionable comparison reason on the durable
// aggregate member. A collection-level not-comparable verdict without its
// provider diagnostic strands callers: the underlying Test Genie comparison
// already knows whether recovery means restoring a provider, capturing a
// baseline, or fixing incompatible provenance.
func collectionDiffDetail(result CachedDiff) string {
	if detail := strings.TrimSpace(result.Error); detail != "" {
		return detail
	}
	if result.Result == nil || result.Result.Comparison == nil {
		return ""
	}
	comparison := result.Result.Comparison
	for _, diagnostic := range comparison.GetDiagnostics() {
		if diagnostic == nil {
			continue
		}
		detail := strings.TrimSpace(diagnostic.GetDetail())
		if detail == "" {
			detail = strings.TrimSpace(diagnostic.GetCode())
		}
		if detail == "" {
			continue
		}
		if remediation := strings.TrimSpace(diagnostic.GetRemediation()); remediation != "" {
			return detail + " (recovery: " + remediation + ")"
		}
		return detail
	}
	return ""
}

func (s *Service) markCollectionDiffLifecycle(repoID int64, branch, name, operationID, scenario string, lifecycle CollectionDiffChildLifecycle) error {
	_, err := s.storage.UpdateCollectionDiffOperation(repoID, branch, name, operationID, func(operation *CollectionDiffOperation) error {
		for i := range operation.Members {
			if operation.Members[i].Scenario == scenario && operation.Members[i].Status == "pending" {
				operation.Members[i].Lifecycle = lifecycle
			}
		}
		operation.UpdatedAt, operation.LastProgressAt = s.now().UTC(), s.now().UTC()
		return nil
	})
	return err
}

// collectionDiffLifecycle is the aggregate owner state. A terminal child that
// still needs comparison/fan-in is finalizing, never generic pending.
func collectionDiffLifecycle(operation CollectionDiffOperation) string {
	if len(operation.Members) == 0 {
		return "preparing"
	}
	for _, member := range operation.Members {
		if member.Status == "pending" {
			return "executing"
		}
	}
	return "terminal"
}

// GetCollectionDiffStatus reconciles already-terminal Test Genie children
// before projecting the durable aggregate. Test Genie owns execution; GCT owns
// this durable projection. This pull path is the correctness backstop when the
// asynchronous finalizer was lost during a restart or client detach.
func (s *Service) GetCollectionDiffStatus(ctx context.Context, repoID int64, branch, name, operationID string) (CollectionManifest, CollectionDiffOperation, *commonv1.OperationStanding, error) {
	if _, _, err := s.reconcileCollectionDiff(ctx, repoID, branch, name, operationID); err != nil {
		return CollectionManifest{}, CollectionDiffOperation{}, nil, err
	}
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil {
		return CollectionManifest{}, CollectionDiffOperation{}, nil, err
	}
	collection, err := s.collectionForOperation(repoID, branch, name, operation)
	if err != nil {
		return CollectionManifest{}, operation, nil, err
	}
	standing := &commonv1.OperationStanding{
		Owner: "git-control-tower", OperationId: operation.ID,
		StartedAt:       operation.CreatedAt.UTC().Format(time.RFC3339),
		LastProgressAt:  operation.LastProgressAt.UTC().Format(time.RFC3339),
		Directive:       "wait",
		ReattachCommand: collectionDiffReattachCommand(collection, operation),
	}
	for _, member := range operation.Members {
		if member.Status != "pending" || member.RunID == "" {
			continue
		}
		status, statusErr := s.exec.RunStatus(ctx, member.Scenario, member.RunID)
		if statusErr != nil {
			standing.Children = append(standing.Children, &commonv1.OperationStanding{Owner: "test-genie", OperationId: member.RunID, Lifecycle: "executing", Directive: "recover", Detail: statusErr.Error()})
			continue
		}
		child := status.Standing
		if child == nil {
			child = &commonv1.OperationStanding{Owner: "test-genie", OperationId: member.RunID, Directive: "wait", RecommendedWaitSeconds: int32(status.RecommendedNextCheckSeconds)}
			if status.Terminal {
				child.Lifecycle = "terminal"
			} else {
				child.Lifecycle = "executing"
			}
		} else if !status.Terminal && child.Lifecycle == "queued" {
			// Once the parent has durably recorded a Test Genie run ID, it owns an
			// attached execution. Preserve Test Genie's phase detail but never
			// expose the parent child as a fresh queued handoff.
			child.Lifecycle = "executing"
		}
		standing.Children = append(standing.Children, child)
	}
	standing.Lifecycle = collectionDiffStandingLifecycle(operation, standing.Children)
	if standing.Lifecycle == "terminal" {
		standing.Directive = collectionDiffTerminalDirective(operation)
		standing.ReattachCommand = ""
		standing.TerminalOutcome = collectionDiffTerminalOutcome(operation, collection)
	}
	return collection, operation, standing, nil
}

// reconcileCollectionDiff projects terminal child evidence without waiting for
// active work. It is safe to call from every status read and from a resumed
// wait: the child diff cache and parent member transition are both idempotent.
// A missing durable handoff is terminal infrastructure failure, never a
// forever-pending operation.
func (s *Service) reconcileCollectionDiff(ctx context.Context, repoID int64, branch, name, operationID string) (CollectionManifest, CollectionDiffOperation, error) {
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil {
		return CollectionManifest{}, CollectionDiffOperation{}, err
	}
	collection, err := s.collectionForOperation(repoID, branch, name, operation)
	if err != nil {
		return CollectionManifest{}, operation, err
	}
	// A pre-run dispatch has no Test Genie child to query. Its lease makes that
	// window recoverable after a process crash or transient outage without
	// requiring an agent to remember to issue StartCollectionDiff again.
	if operationNeedsDispatchRetry(operation, s.now().UTC()) && operation.RepoDir != "" {
		s.collectionDiffMu.Lock()
		latest, loadErr := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
		if loadErr == nil {
			resumed, resumeErr := s.dispatchCollectionDiff(ctx, StartCollectionDiffRequest{RepoID: repoID, RepoDir: latest.RepoDir, Branch: branch, Name: name, OperationID: operationID}, collection, latest)
			if resumeErr != nil {
				s.collectionDiffMu.Unlock()
				return collection, latest, resumeErr
			}
			operation = resumed.Operation
		}
		s.collectionDiffMu.Unlock()
		if loadErr != nil {
			return collection, operation, loadErr
		}
	} else if operationNeedsDispatchRetry(operation, s.now().UTC()) {
		if err := s.failUnrecoverableCollectionDispatch(repoID, branch, name, operationID, "cannot recover collection diff dispatch: repository path is missing from the durable operation"); err != nil {
			return collection, operation, err
		}
	}
	for _, member := range operation.Members {
		if member.Status != "pending" || member.RunID == "" {
			continue
		}
		intent, found, loadErr := s.storage.LoadDiffIntent(repoID, member.Scenario, branch, member.BaselineName, member.RunID)
		if loadErr != nil {
			return collection, operation, loadErr
		}
		if !found {
			if err := s.failCollectionDiffMember(repoID, branch, name, operationID, member.Scenario, "durable child diff intent is missing"); err != nil {
				return collection, operation, err
			}
			continue
		}
		status, statusErr := s.exec.RunStatus(ctx, member.Scenario, member.RunID)
		if statusErr != nil {
			// A transport error is recoverable; retain it for diagnostics while a
			// later status read retries the durable source of truth.
			_ = s.noteCollectionDiffReconciliationError(repoID, branch, name, operationID, statusErr)
			continue
		}
		if status.Missing {
			detail := fmt.Sprintf("test-genie run %s is missing from its durable run record", member.RunID)
			if err := s.failCollectionDiffMember(repoID, branch, name, operationID, member.Scenario, detail); err != nil {
				return collection, operation, err
			}
			continue
		}
		if !status.Terminal {
			if member.Lifecycle != CollectionDiffChildAwaiting {
				if markErr := s.markCollectionDiffLifecycle(repoID, branch, name, operationID, member.Scenario, CollectionDiffChildAwaiting); markErr != nil {
					return collection, operation, markErr
				}
			}
			continue
		}
		if _, finalizeErr := s.FinalizeCollectionDiff(ctx, repoID, PendingCollectionDiff{CollectionName: name, OperationID: operationID, Scenario: member.Scenario, Pending: intent.PendingDiff()}); finalizeErr != nil {
			if errors.Is(finalizeErr, context.Canceled) || errors.Is(finalizeErr, context.DeadlineExceeded) {
				return collection, operation, finalizeErr
			}
			return collection, operation, finalizeErr
		}
	}
	operation, err = s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	return collection, operation, err
}

// ReconcileCollectionDiffOperations is the server-owned pull backstop for
// detached clients and lost completion delivery. It never waits for active
// Test Genie work; each operation is independently reconciled from durable run
// state, so one unavailable child does not prevent the others from converging.
func (s *Service) ReconcileCollectionDiffOperations(ctx context.Context, repoID int64) error {
	operations, err := s.storage.ListCollectionDiffOperations(repoID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, operation := range operations {
		if operation.Lifecycle == "terminal" {
			continue
		}
		if _, _, reconcileErr := s.reconcileCollectionDiff(ctx, repoID, operation.Branch, operation.Collection, operation.ID); reconcileErr != nil && firstErr == nil {
			firstErr = reconcileErr
		}
	}
	return firstErr
}

func operationNeedsDispatchRetry(operation CollectionDiffOperation, now time.Time) bool {
	for _, member := range operation.Members {
		if member.Status != "pending" || member.RunID != "" {
			continue
		}
		if member.DispatchLeaseExpiresAt.IsZero() || !now.Before(member.DispatchLeaseExpiresAt) {
			return true
		}
	}
	return false
}

func (s *Service) failUnrecoverableCollectionDispatch(repoID int64, branch, name, operationID, detail string) error {
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil {
		return err
	}
	for _, member := range operation.Members {
		if member.Status == "pending" && member.RunID == "" {
			if err := s.failCollectionDiffMember(repoID, branch, name, operationID, member.Scenario, detail); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) failCollectionDiffMember(repoID int64, branch, name, operationID, scenario, detail string) error {
	_, err := s.storage.UpdateCollectionDiffOperation(repoID, branch, name, operationID, func(operation *CollectionDiffOperation) error {
		for i := range operation.Members {
			if operation.Members[i].Scenario == scenario && operation.Members[i].Status == "pending" {
				operation.Members[i].Status, operation.Members[i].Detail, operation.Members[i].Lifecycle = "failed", detail, CollectionDiffChildFailed
			}
		}
		operation.UpdatedAt, operation.LastProgressAt = s.now().UTC(), s.now().UTC()
		operation.Lifecycle, operation.LastReconciliationError = collectionDiffLifecycle(*operation), detail
		return nil
	})
	return err
}

func (s *Service) noteCollectionDiffReconciliationError(repoID int64, branch, name, operationID string, cause error) error {
	_, err := s.storage.UpdateCollectionDiffOperation(repoID, branch, name, operationID, func(operation *CollectionDiffOperation) error {
		operation.LastReconciliationError = cause.Error()
		operation.UpdatedAt = s.now().UTC()
		return nil
	})
	return err
}

// WaitCollectionDiff is the only blocking aggregate operation. Its durable
// children keep running when the caller's context expires.
func (s *Service) WaitCollectionDiff(ctx context.Context, repoID int64, branch, name, operationID string) (CollectionManifest, CollectionDiffOperation, error) {
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil {
		return CollectionManifest{}, CollectionDiffOperation{}, err
	}
	collection, err := s.collectionForOperation(repoID, branch, name, operation)
	if err != nil {
		return CollectionManifest{}, operation, err
	}
	// Recover only undispatched children here. Terminal fan-in remains below,
	// where it is deliberately concurrent; doing full reconciliation before the
	// fan-out can make one slow child serialize every sibling.
	if operationNeedsDispatchRetry(operation, s.now().UTC()) && operation.RepoDir != "" {
		s.collectionDiffMu.Lock()
		latest, loadErr := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
		if loadErr == nil {
			resumed, resumeErr := s.dispatchCollectionDiff(ctx, StartCollectionDiffRequest{RepoID: repoID, RepoDir: latest.RepoDir, Branch: branch, Name: name, OperationID: operationID}, collection, latest)
			if resumeErr != nil {
				s.collectionDiffMu.Unlock()
				return collection, latest, resumeErr
			}
			operation = resumed.Operation
		}
		s.collectionDiffMu.Unlock()
		if loadErr != nil {
			return collection, operation, loadErr
		}
	} else if operationNeedsDispatchRetry(operation, s.now().UTC()) {
		if err := s.failUnrecoverableCollectionDispatch(repoID, branch, name, operationID, "cannot recover collection diff dispatch: repository path is missing from the durable operation"); err != nil {
			return collection, operation, err
		}
		operation, err = s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
		if err != nil {
			return collection, CollectionDiffOperation{}, err
		}
	}
	type pendingDiffJob struct {
		member CollectionDiffMember
		intent DiffIntent
	}
	var jobs []pendingDiffJob
	seen := make(map[string]struct{})
	for _, member := range operation.Members {
		if member.Status != "pending" || member.RunID == "" {
			continue
		}
		intent, found, err := s.storage.LoadDiffIntent(repoID, member.Scenario, branch, member.BaselineName, member.RunID)
		if err != nil {
			return collection, operation, err
		}
		if !found {
			if err := s.failCollectionDiffMember(repoID, branch, name, operationID, member.Scenario, "durable child diff intent is missing"); err != nil {
				return collection, operation, err
			}
			continue
		}
		key := member.Scenario + "\x00" + member.RunID
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		jobs = append(jobs, pendingDiffJob{member: member, intent: intent})
	}
	const maxConcurrentCollectionDiffWaits = 8
	sem := make(chan struct{}, maxConcurrentCollectionDiffWaits)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, job := range jobs {
		job := job
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			_, finalizeErr := s.FinalizeCollectionDiff(ctx, repoID, PendingCollectionDiff{CollectionName: name, OperationID: operationID, Scenario: job.member.Scenario, Pending: job.intent.PendingDiff()})
			if finalizeErr != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = finalizeErr
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	operation, err = s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil {
		return collection, CollectionDiffOperation{}, err
	}
	if errors.Is(firstErr, context.Canceled) || errors.Is(firstErr, context.DeadlineExceeded) || ctx.Err() != nil {
		pendingIDs := make([]string, 0)
		for _, member := range operation.Members {
			if member.Status == "pending" && member.RunID != "" {
				pendingIDs = append(pendingIDs, member.RunID)
			}
		}
		cause := firstErr
		if cause == nil {
			cause = ctx.Err()
		}
		return collection, operation, &CollectionIncompleteError{PendingRunIDs: pendingIDs, Cause: cause}
	}
	return collection, operation, nil
}

func collectionDiffStandingLifecycle(operation CollectionDiffOperation, children []*commonv1.OperationStanding) string {
	if len(operation.Members) == 0 {
		return "preparing"
	}
	if operation.Lifecycle == "terminal" {
		return "terminal"
	}
	for _, child := range children {
		if child.GetLifecycle() != "terminal" {
			return "executing"
		}
	}
	for _, member := range operation.Members {
		if member.Status == "pending" && member.RunID == "" {
			return "dispatching"
		}
		if member.Status == "pending" {
			return "finalizing"
		}
	}
	if operation.Lifecycle != "" {
		return operation.Lifecycle
	}
	return "preparing"
}

func collectionDiffTerminalDirective(operation CollectionDiffOperation) string {
	for _, member := range operation.Members {
		if member.Required && (member.Status == "failed" || member.Status == "not-comparable") {
			return "inspect"
		}
	}
	return ""
}

func collectionDiffTerminalOutcome(operation CollectionDiffOperation, collection CollectionManifest) string {
	verdict := operation.Aggregate(collection).Verdict
	switch verdict {
	case CollectionDiffClean:
		return "passed"
	case CollectionDiffNotComparable:
		return "not-comparable"
	default:
		return "failed"
	}
}

func collectionDiffReattachCommand(collection CollectionManifest, operation CollectionDiffOperation) string {
	args := []string{"git-control-tower", "baseline", "collection", "diff", "wait", "--name", collection.Name}
	if collection.Branch != "" {
		args = append(args, "--branch", collection.Branch)
	}
	args = append(args, "--operation-id", operation.ID, "--json")
	return strings.Join(args, " ")
}

// collectionForOperation uses the operation's creation-time snapshot whenever
// it exists. That prevents later append-only collection expansion (or a
// same-named replacement) from changing an already-started operation's
// comparison universe. Legacy operations intentionally fall back only to the
// live collection; if it is absent they fail clearly rather than guessing.
func (s *Service) collectionForOperation(repoID int64, branch, name string, operation CollectionDiffOperation) (CollectionManifest, error) {
	snapshot := operation.CollectionSnapshot.Normalized()
	if snapshot.Name != "" {
		if snapshot.Name != strings.TrimSpace(name) || snapshot.Branch != strings.TrimSpace(branch) {
			return CollectionManifest{}, fmt.Errorf("collection diff operation %q snapshot identity does not match requested collection", operation.ID)
		}
		if err := snapshot.Validate(); err != nil {
			return CollectionManifest{}, fmt.Errorf("validate collection diff operation snapshot: %w", err)
		}
		return snapshot, nil
	}
	collection, err := s.storage.LoadCollection(repoID, branch, name)
	if err == nil {
		return collection, nil
	}
	if errors.Is(err, ErrNotFound) {
		return CollectionManifest{}, fmt.Errorf("collection %q is unavailable and legacy diff operation %q has no immutable collection snapshot; recapture the baseline before retrying", name, operation.ID)
	}
	return CollectionManifest{}, err
}

func selectCollectionMembers(collection CollectionManifest, scenarios []string) ([]CollectionMember, error) {
	if len(scenarios) == 0 {
		return append([]CollectionMember(nil), collection.Members...), nil
	}
	want := make(map[string]struct{}, len(scenarios))
	for _, scenario := range scenarios {
		scenario = strings.TrimSpace(scenario)
		if scenario == "" {
			return nil, fmt.Errorf("collection diff scenario is required")
		}
		if _, duplicate := want[scenario]; duplicate {
			return nil, fmt.Errorf("collection diff contains duplicate scenario %q", scenario)
		}
		want[scenario] = struct{}{}
	}
	out := make([]CollectionMember, 0, len(want))
	for _, member := range collection.Members {
		if _, selected := want[member.Scenario]; selected {
			out = append(out, member)
			delete(want, member.Scenario)
		}
	}
	if len(want) > 0 {
		unknown := make([]string, 0, len(want))
		for scenario := range want {
			unknown = append(unknown, scenario)
		}
		sort.Strings(unknown)
		return nil, fmt.Errorf("collection diff includes scenario outside collection: %s", strings.Join(unknown, ", "))
	}
	return out, nil
}

// StorageLoadCollection is the narrow read seam exposed to transport handlers.
// Collection policy and transitions remain owned by this package rather than
// letting handlers reach into storage directly.
func (s *Service) StorageLoadCollection(repoID int64, branch, name string) (CollectionManifest, error) {
	return s.GetCollectionCaptureStatus(context.Background(), repoID, branch, name)
}

// CollectionCaptureStanding projects durable capture state for every transport.
func CollectionCaptureStanding(collection CollectionManifest) *commonv1.OperationStanding {
	standing := &commonv1.OperationStanding{Owner: "git-control-tower", OperationId: "capture:" + collection.Name, Directive: "wait"}
	if collection.Coverage().Complete() {
		standing.Lifecycle, standing.TerminalOutcome, standing.Directive = "terminal", "passed", ""
		return standing
	}
	for _, member := range collection.Members {
		if member.Status == CollectionMemberFailed {
			// A failed member is terminal capture evidence, not active work. If
			// we report it as executing, callers correctly refuse to advance but
			// have no terminal state to synchronize or recover from.
			standing.Lifecycle, standing.TerminalOutcome, standing.Directive, standing.Detail = "terminal", "failed", "inspect", member.Error
			return standing
		}
	}
	standing.Lifecycle = "executing"
	args := []string{"git-control-tower", "baseline", "collection", "show", "--name", collection.Name}
	if collection.Branch != "" {
		args = append(args, "--branch", collection.Branch)
	}
	standing.ReattachCommand = strings.Join(append(args, "--wait", "--json"), " ")
	return standing
}

func (s *Service) DeleteCollection(_ context.Context, repoID int64, branch, name string) error {
	collection, err := s.storage.LoadCollection(repoID, branch, name)
	if err != nil {
		return err
	}
	if err := s.storage.DeleteCollection(repoID, branch, name); err != nil {
		return err
	}
	for _, snapshot := range collection.PathSnapshots {
		if err := s.storage.DeletePathSnapshot(repoID, branch, snapshot); err != nil && !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("collection deleted but source evidence cleanup %q failed: %w", snapshot, err)
		}
	}
	return nil
}

// SortedTargets is a small public helper for callers building stable
// idempotency keys before asking GCT to create/resume a collection.
func SortedTargets(targets []CollectionTarget) []CollectionTarget {
	out := append([]CollectionTarget(nil), targets...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}
