package opsrunner

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"swarm-manager/internal/agentops"
)

const workflowSchemaVersion = "1.0.0"

// DomainLocator maps a domain entity (kind + id) to the on-disk directory that
// owns its agent-operations coordination state. Workflow instances and binding
// overrides live beside the item/initiative they coordinate, never in an
// unbounded central store. Implementations MUST reject an id that would escape
// its domain root (path traversal), returning an error rather than a resolved
// path.
type DomainLocator interface {
	// AgentOpsDir returns "<domain entity dir>/agentops", the coordination-state
	// directory for one entity. It does not create the directory.
	AgentOpsDir(kind agentops.TargetKind, id string) (string, error)
	// Scan returns every existing agentops directory across all domain roots, so
	// a scheduler or lister can reload persisted workflows on boot without an
	// index. Bounded by the number of items/initiatives.
	Scan() ([]string, error)
}

// WorkflowRepo persists domain workflow instances with atomic, traversal-safe
// writes and optimistic concurrency. One JSON document per domain entity lives
// at "<agentops dir>/workflow.json".
type WorkflowRepo struct {
	loc DomainLocator
	mu  sync.Map // path -> *sync.Mutex, serializing writes within this process
}

// NewWorkflowRepo constructs a repository over a domain locator.
func NewWorkflowRepo(loc DomainLocator) *WorkflowRepo { return &WorkflowRepo{loc: loc} }

const workflowFile = "workflow.json"

func (r *WorkflowRepo) pathFor(kind agentops.TargetKind, id string) (string, error) {
	dir, err := r.loc.AgentOpsDir(kind, id)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, workflowFile), nil
}

func (r *WorkflowRepo) lockFor(path string) *sync.Mutex {
	m, _ := r.mu.LoadOrStore(path, &sync.Mutex{})
	return m.(*sync.Mutex)
}

// Load returns the persisted workflow instance for a domain entity. found=false
// when none exists yet.
func (r *WorkflowRepo) Load(kind agentops.TargetKind, id string) (agentops.WorkflowInstance, bool, error) {
	path, err := r.pathFor(kind, id)
	if err != nil {
		return agentops.WorkflowInstance{}, false, err
	}
	return loadWorkflowFile(path)
}

func loadWorkflowFile(path string) (agentops.WorkflowInstance, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return agentops.WorkflowInstance{}, false, nil
		}
		return agentops.WorkflowInstance{}, false, err
	}
	if err := agentops.ValidateWorkflowInstance(raw); err != nil {
		return agentops.WorkflowInstance{}, false, fmt.Errorf("persisted workflow %s is invalid: %w", path, err)
	}
	w, err := agentops.DecodeWorkflowInstance(raw)
	if err != nil {
		return agentops.WorkflowInstance{}, false, err
	}
	return w, true, nil
}

// CreateOrLoad returns the existing workflow for a domain entity or a fresh
// open one (not yet persisted; version 0). The caller mutates and Commits it.
func (r *WorkflowRepo) CreateOrLoad(kind agentops.TargetKind, id string) (agentops.WorkflowInstance, error) {
	w, found, err := r.Load(kind, id)
	if err != nil {
		return agentops.WorkflowInstance{}, err
	}
	if found {
		return w, nil
	}
	return agentops.WorkflowInstance{
		Kind:          "agentops-workflow-instance",
		SchemaVersion: workflowSchemaVersion,
		InstanceID:    instanceID(kind, id),
		Domain:        agentops.WorkflowDomain{Kind: domainKindFor(kind), ID: id},
		State:         agentops.WorkflowOpen,
		Version:       0,
	}, nil
}

// Commit persists next, enforcing optimistic concurrency: the on-disk version
// must equal prevVersion, and next.Version must be prevVersion+1. A concurrent
// writer that advanced the on-disk version first makes this return
// ErrWorkflowConflict, so no update is silently lost. The write is atomic
// (temp file + rename in the same directory), so a crash never leaves a
// half-written document — a restart between prepare and commit sees either the
// old committed doc or the new one, never a torn state.
func (r *WorkflowRepo) Commit(prevVersion int, next agentops.WorkflowInstance) error {
	if next.Version != prevVersion+1 {
		return fmt.Errorf("%w: next.Version=%d must be prevVersion+1=%d", ErrWorkflowConflict, next.Version, prevVersion+1)
	}
	path, err := r.pathFor(agentops.TargetKind(next.Domain.Kind), next.Domain.ID)
	if err != nil {
		return err
	}
	m := r.lockFor(path)
	m.Lock()
	defer m.Unlock()

	// Re-read under the lock to confirm the on-disk version still matches.
	onDisk, found, err := loadWorkflowFile(path)
	if err != nil {
		return err
	}
	switch {
	case !found && prevVersion != 0:
		return fmt.Errorf("%w: expected version %d but no document exists", ErrWorkflowConflict, prevVersion)
	case found && onDisk.Version != prevVersion:
		return fmt.Errorf("%w: on-disk version %d != expected %d", ErrWorkflowConflict, onDisk.Version, prevVersion)
	}

	raw, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := agentops.ValidateWorkflowInstance(raw); err != nil {
		return fmt.Errorf("refusing to persist invalid workflow: %w", err)
	}
	return atomicWrite(path, raw)
}

// List returns every persisted workflow instance across all domain roots, for
// boot-time scheduler reload and diagnostics.
func (r *WorkflowRepo) List() ([]agentops.WorkflowInstance, error) {
	dirs, err := r.loc.Scan()
	if err != nil {
		return nil, err
	}
	sort.Strings(dirs)
	var out []agentops.WorkflowInstance
	for _, dir := range dirs {
		w, found, err := loadWorkflowFile(filepath.Join(dir, workflowFile))
		if err != nil {
			return nil, err
		}
		if found {
			out = append(out, w)
		}
	}
	return out, nil
}

// FindByRunID scans persisted workflows for the one whose operation was
// dispatched under a live agent run id, returning that workflow. It is the
// completion seam's correlation path for a target whose delivered round is keyed
// by a scope id that differs from the workflow key: a plan-execution round is
// keyed by the operating-mode engine's resolved plan execution id, while its
// workflow is keyed by the plan handle the runner targeted, so loading the
// workflow by the round's scope id would miss. The globally-unique agent run id
// is the invariant linkage the runner persists on the operation record, so a scan
// by run id correlates the round back to its owning workflow regardless of target
// kind. Bounded by the number of items/initiatives/plan-execution scopes; an
// empty runID never matches.
func (r *WorkflowRepo) FindByRunID(runID string) (agentops.WorkflowInstance, bool, error) {
	if strings.TrimSpace(runID) == "" {
		return agentops.WorkflowInstance{}, false, nil
	}
	workflows, err := r.List()
	if err != nil {
		return agentops.WorkflowInstance{}, false, err
	}
	for _, w := range workflows {
		if _, ok := FindOperationByRunID(w, runID); ok {
			return w, true, nil
		}
	}
	return agentops.WorkflowInstance{}, false, nil
}

// FindOperationByIdempotencyKey returns the correlated execution record for a
// key, if the workflow already consumed it (a replay).
func FindOperationByIdempotencyKey(w agentops.WorkflowInstance, key string) (agentops.OperationExecutionRecord, bool) {
	for _, op := range w.Operations {
		if op.IdempotencyKey == key {
			return op, true
		}
	}
	return agentops.OperationExecutionRecord{}, false
}

// FindOperationByRunID returns the operation record dispatched under a live agent
// run id, if the workflow correlates one. It is how the completion seam maps a
// delivered round (which carries only its run id) back to the execution to
// finalize via CommitResult. An empty runID never matches.
func FindOperationByRunID(w agentops.WorkflowInstance, runID string) (agentops.OperationExecutionRecord, bool) {
	if runID == "" {
		return agentops.OperationExecutionRecord{}, false
	}
	for _, op := range w.Operations {
		if op.RunID == runID {
			return op, true
		}
	}
	return agentops.OperationExecutionRecord{}, false
}

// HasIdempotencyKey reports whether a key was already consumed by any correlated
// action or operation on the workflow.
func HasIdempotencyKey(w agentops.WorkflowInstance, key string) bool {
	for _, k := range w.IdempotencyKeys {
		if k == key {
			return true
		}
	}
	_, ok := FindOperationByIdempotencyKey(w, key)
	return ok
}

// atomicWrite writes data to path via a same-directory temp file + rename, so a
// reader never observes a partial file and a crash leaves either the old file or
// the new one.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".wf-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after a successful rename
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// instanceID builds a stable durable identity for a domain entity's workflow.
func instanceID(kind agentops.TargetKind, id string) string {
	return "wf-" + domainKindFor(kind) + "-" + sanitizeToken(id)
}

// sanitizeToken maps an arbitrary domain id onto a stable slug used in the
// instance id (not a filesystem path — the locator owns path safety).
func sanitizeToken(id string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(id) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}

// validateDomainToken rejects an id that would escape its domain root. It is the
// shared traversal guard concrete locators use.
func validateDomainToken(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("empty domain id")
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("domain id %q contains a parent-directory traversal", id)
	}
	if filepath.IsAbs(id) {
		return fmt.Errorf("domain id %q must be relative", id)
	}
	return nil
}
