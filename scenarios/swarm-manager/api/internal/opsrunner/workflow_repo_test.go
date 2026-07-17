package opsrunner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/agentops"
)

func baseWorkflow(kind agentops.TargetKind, id string) agentops.WorkflowInstance {
	return agentops.WorkflowInstance{
		Kind: "agentops-workflow-instance", SchemaVersion: workflowSchemaVersion,
		InstanceID: instanceID(kind, id), Domain: agentops.WorkflowDomain{Kind: domainKindFor(kind), ID: id},
		State: agentops.WorkflowOpen, Version: 0,
	}
}

func TestWorkflowRepoCommitAndLoad(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	kind, id := agentops.TargetInitiative, "init-a"
	w, err := repo.CreateOrLoad(kind, id)
	if err != nil {
		t.Fatal(err)
	}
	next := cloneWorkflow(w)
	next.State = agentops.WorkflowRunning
	next.Version = w.Version + 1
	if err := repo.Commit(w.Version, next); err != nil {
		t.Fatalf("commit: %v", err)
	}
	got, found, err := repo.Load(kind, id)
	if err != nil || !found {
		t.Fatalf("load: found=%v err=%v", found, err)
	}
	if got.State != agentops.WorkflowRunning || got.Version != 1 {
		t.Fatalf("loaded %+v", got)
	}
}

// TestWorkflowRepoCompareAndSwapConflict proves a stale writer loses: after one
// writer advances the version, a second commit against the old version fails.
// [REQ:REQ-P0-011-WORKFLOW-DURABILITY]
func TestWorkflowRepoCompareAndSwapConflict(t *testing.T) {
	repo := NewWorkflowRepo(memLocator{root: t.TempDir()})
	kind, id := agentops.TargetInitiative, "init-a"
	w, _ := repo.CreateOrLoad(kind, id)

	first := cloneWorkflow(w)
	first.State = agentops.WorkflowRunning
	first.Version = 1
	if err := repo.Commit(0, first); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	// A second writer that still holds version 0 must be rejected.
	stale := cloneWorkflow(w)
	stale.State = agentops.WorkflowBlocked
	stale.Version = 1
	if err := repo.Commit(0, stale); !errors.Is(err, ErrWorkflowConflict) {
		t.Fatalf("stale commit must be ErrWorkflowConflict, got %v", err)
	}
}

// TestWorkflowRepoAtomicWriteNoTornState proves writes are atomic: no temp file
// leaks and the persisted document is always complete + valid.
// [REQ:REQ-P0-011-WORKFLOW-DURABILITY]
func TestWorkflowRepoAtomicWriteNoTornState(t *testing.T) {
	root := t.TempDir()
	repo := NewWorkflowRepo(memLocator{root: root})
	kind, id := agentops.TargetBacklogItem, "fix/y"
	w, _ := repo.CreateOrLoad(kind, id)
	next := cloneWorkflow(w)
	next.State = agentops.WorkflowRunning
	next.Version = 1
	if err := repo.Commit(0, next); err != nil {
		t.Fatal(err)
	}
	dir, _ := memLocator{root: root}.AgentOpsDir(kind, id)
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp file leaked: %s", e.Name())
		}
	}
	// The persisted document validates.
	raw, err := os.ReadFile(filepath.Join(dir, workflowFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := agentops.ValidateWorkflowInstance(raw); err != nil {
		t.Fatalf("persisted workflow invalid: %v", err)
	}
}

// TestWorkflowRepoListReloadsFromDisk proves List reconstructs persisted
// workflows across domains (the boot-time reload substrate).
func TestWorkflowRepoListReloadsFromDisk(t *testing.T) {
	root := t.TempDir()
	repo := NewWorkflowRepo(memLocator{root: root})
	for _, tc := range []struct {
		kind agentops.TargetKind
		id   string
	}{{agentops.TargetInitiative, "a"}, {agentops.TargetBacklogItem, "fix/b"}} {
		w, _ := repo.CreateOrLoad(tc.kind, tc.id)
		n := cloneWorkflow(w)
		n.State = agentops.WorkflowRunning
		n.Version = 1
		if err := repo.Commit(0, n); err != nil {
			t.Fatal(err)
		}
	}
	// A fresh repo (simulating a restart) reloads both from disk.
	fresh := NewWorkflowRepo(memLocator{root: root})
	all, err := fresh.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("List returned %d workflows, want 2", len(all))
	}
}

func TestAgentOpsDirRejectsTraversal(t *testing.T) {
	loc := memLocator{root: t.TempDir()}
	if _, err := loc.AgentOpsDir(agentops.TargetInitiative, "../escape"); err == nil {
		t.Fatalf("traversal id must be rejected")
	}
}

// TestFSLocatorPlanExecutionDir proves the FS locator resolves plan-execution
// workflow storage (the primary backlog execution path) via its dedicated
// resolver, and fails typed when the resolver is not wired.
func TestFSLocatorPlanExecutionDir(t *testing.T) {
	loc := FSLocator{
		PlanExecutionDir: func(id string) (string, error) {
			return filepath.Join("/data", "plan-executions", id), nil
		},
	}
	dir, err := loc.AgentOpsDir(agentops.TargetPlanExecution, "plan-abc")
	if err != nil {
		t.Fatalf("AgentOpsDir plan-execution: %v", err)
	}
	if want := filepath.Join("/data", "plan-executions", "plan-abc", "agentops"); dir != want {
		t.Fatalf("plan-execution agentops dir = %q, want %q", dir, want)
	}
	if _, err := (FSLocator{}).AgentOpsDir(agentops.TargetPlanExecution, "plan-abc"); err == nil {
		t.Fatal("expected a typed error when PlanExecutionDir resolver is missing")
	}
}
