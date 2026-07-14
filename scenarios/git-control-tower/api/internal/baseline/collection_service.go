package baseline

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
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
	PathSelections []string
	PathPolicy     PathSnapshotPolicy
	CreatedBy      string
	Reason         string
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
		started, err := s.StartCapture(ctx, CreateRequest{
			RepoID: req.RepoID, RepoDir: req.RepoDir, Scenario: member.Scenario,
			Name: member.BaselineName, Branch: branch, CreatedBy: req.CreatedBy, Reason: req.Reason,
		})
		if err != nil {
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
	return CollectionManifest{Name: strings.TrimSpace(req.Name), Branch: branch, CreatedAt: now, UpdatedAt: now, SchemaVersion: CollectionSchemaVersion, Members: members, PathSnapshots: paths}.Normalized()
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
			return s.storage.LoadCollection(repoID, pending.Branch, pending.CollectionName)
		}
		_, updateErr := s.storage.UpdateCollectionMember(repoID, pending.Branch, pending.CollectionName, pending.Scenario, func(member *CollectionMember) error {
			member.Status, member.Error, member.UpdatedAt = CollectionMemberFailed, err.Error(), s.now().UTC()
			return nil
		})
		if updateErr != nil {
			return CollectionManifest{}, updateErr
		}
		return CollectionManifest{}, err
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
	for _, member := range collection.Members {
		if member.Status != CollectionMemberPending || member.RunID == "" {
			continue
		}
		intent, found, err := s.storage.LoadSnapshotIntent(repoID, member.Scenario, branch, member.BaselineName, member.RunID)
		if err != nil {
			return CollectionManifest{}, err
		}
		if !found {
			collection, err = s.storage.UpdateCollectionMember(repoID, branch, name, member.Scenario, func(target *CollectionMember) error {
				target.Status, target.Error, target.UpdatedAt = CollectionMemberFailed, "durable snapshot intent is missing", s.now().UTC()
				return nil
			})
			if err != nil {
				return CollectionManifest{}, err
			}
			continue
		}
		collection, err = s.FinalizeCollectionCapture(ctx, repoID, PendingCollectionCapture{CollectionName: name, Branch: branch, Scenario: member.Scenario, Pending: intent.PendingCapture()})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return collection, nil
			}
			return CollectionManifest{}, err
		}
	}
	return collection, nil
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
	if existing, err := s.storage.LoadCollectionDiffOperation(req.RepoID, req.Branch, req.Name, req.OperationID); err == nil {
		if !sameCollectionDiffSelection(existing.Members, selected) {
			return StartCollectionDiffResult{}, fmt.Errorf("collection diff operation %q already exists with a different member selection", req.OperationID)
		}
		return StartCollectionDiffResult{Collection: collection, Operation: existing, Members: append([]CollectionDiffMember(nil), existing.Members...)}, nil
	} else if !errors.Is(err, ErrNotFound) {
		return StartCollectionDiffResult{}, err
	}
	operation := CollectionDiffOperation{ID: req.OperationID, Collection: req.Name, Branch: req.Branch, CreatedAt: s.now().UTC(), UpdatedAt: s.now().UTC()}
	for _, member := range selected {
		operation.Members = append(operation.Members, CollectionDiffMember{Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required, Status: string(member.Status), RunID: member.RunID, Detail: member.Error})
	}
	if err := s.storage.SaveCollectionDiffOperation(req.RepoID, operation, CreateOnly); err != nil {
		return StartCollectionDiffResult{}, err
	}
	result := StartCollectionDiffResult{Collection: collection, Operation: operation}
	for _, member := range selected {
		state := CollectionDiffMember{Scenario: member.Scenario, BaselineName: member.BaselineName, Required: member.Required, Status: string(member.Status), RunID: member.RunID, Detail: member.Error}
		if member.Status != CollectionMemberReady {
			result.Members = append(result.Members, state)
			continue
		}
		started, err := s.StartDiff(ctx, StartDiffRequest{RepoID: req.RepoID, RepoDir: req.RepoDir, Branch: req.Branch, Scenario: member.Scenario, Name: member.BaselineName})
		if err != nil {
			state.Status, state.Detail = "failed", err.Error()
			result.Members = append(result.Members, state)
			continue
		}
		state.Status, state.RunID = "pending", started.RunID
		result.Members = append(result.Members, state)
		result.Pending = append(result.Pending, PendingCollectionDiff{CollectionName: req.Name, OperationID: req.OperationID, Scenario: member.Scenario, Pending: started.Pending})
	}
	result.Operation.Members = append([]CollectionDiffMember(nil), result.Members...)
	result.Operation.UpdatedAt = s.now().UTC()
	if err := s.storage.SaveCollectionDiffOperation(req.RepoID, result.Operation, Overwrite); err != nil {
		return StartCollectionDiffResult{}, err
	}
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
	result, err := s.FinalizeDiff(ctx, pending.Pending)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return operation, nil // detached wait; leave durable member pending
		}
		for i := range operation.Members {
			if operation.Members[i].Scenario == pending.Scenario {
				operation.Members[i].Status, operation.Members[i].Detail = "failed", err.Error()
			}
		}
	} else {
		for i := range operation.Members {
			if operation.Members[i].Scenario == pending.Scenario {
				operation.Members[i].Status, operation.Members[i].Verdict, operation.Members[i].Detail = "ready", result.Result.Verdict, result.Error
			}
		}
	}
	operation.UpdatedAt = s.now().UTC()
	if saveErr := s.storage.SaveCollectionDiffOperation(repoID, operation, Overwrite); saveErr != nil {
		return CollectionDiffOperation{}, saveErr
	}
	return operation, err
}

func (s *Service) GetCollectionDiff(ctx context.Context, repoID int64, branch, name, operationID string, wait bool) (CollectionManifest, CollectionDiffOperation, error) {
	collection, err := s.storage.LoadCollection(repoID, branch, name)
	if err != nil {
		return CollectionManifest{}, CollectionDiffOperation{}, err
	}
	operation, err := s.storage.LoadCollectionDiffOperation(repoID, branch, name, operationID)
	if err != nil || !wait {
		return collection, operation, err
	}
	for _, member := range operation.Members {
		if member.Status != "pending" || member.RunID == "" {
			continue
		}
		intent, found, err := s.storage.LoadDiffIntent(repoID, member.Scenario, branch, member.BaselineName, member.RunID)
		if err != nil {
			return collection, operation, err
		}
		if !found {
			member.Status, member.Detail = "failed", "durable child diff intent is missing"
			for i := range operation.Members {
				if operation.Members[i].Scenario == member.Scenario {
					operation.Members[i] = member
				}
			}
			operation.UpdatedAt = s.now().UTC()
			if err := s.storage.SaveCollectionDiffOperation(repoID, operation, Overwrite); err != nil {
				return collection, operation, err
			}
			continue
		}
		operation, err = s.FinalizeCollectionDiff(ctx, repoID, PendingCollectionDiff{CollectionName: name, OperationID: operationID, Scenario: member.Scenario, Pending: intent.PendingDiff()})
		if err != nil {
			return collection, operation, err
		}
	}
	return collection, operation, nil
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
	return s.storage.LoadCollection(repoID, branch, name)
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
