package opsrunner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/agentops"
)

// FSLocator resolves domain agentops directories on the filesystem. It is
// injected with the domain stores' own directory functions so it never
// hard-codes the backlog kind-directory layout or the initiative root: the
// backlog/initiative packages remain the SSOT for where their entities live.
type FSLocator struct {
	// BacklogItemDir returns the on-disk directory of a backlog item given its
	// kind and name (e.g. FileStore.ItemDir). Required to locate item workflows.
	BacklogItemDir func(kind, name string) (string, error)
	// InitiativeDir returns the on-disk directory of an initiative by name.
	InitiativeDir func(name string) (string, error)
	// PlanExecutionDir returns the on-disk directory for a plan-execution target
	// workflow given its handle (plan id / slug). plan-execution is a transient
	// unit of work with no owning domain entity, so its workflow lives under a
	// dedicated scope root rather than an entity folder. Required to locate the
	// primary backlog execution path's workflows.
	PlanExecutionDir func(id string) (string, error)
	// ScenarioDir returns the on-disk directory for a scenario target workflow
	// given its scenario name. Like plan-execution, a scenario spec-sync run has no
	// owning domain entity, so its workflow lives under a dedicated scope root
	// rather than an entity folder. Required to locate spec-sync workflows.
	ScenarioDir func(name string) (string, error)
	// ScanRoots are the base directories walked by Scan to reload persisted
	// workflows on boot (typically the backlog root and the initiative root).
	ScanRoots []string
}

const agentOpsSubdir = "agentops"

// AgentOpsDir returns "<entity dir>/agentops" for a domain entity, rejecting a
// traversal-unsafe id.
func (l FSLocator) AgentOpsDir(kind agentops.TargetKind, id string) (string, error) {
	if err := validateDomainToken(id); err != nil {
		return "", err
	}
	switch kind {
	case agentops.TargetBacklogItem:
		itemKind, name, ok := strings.Cut(id, "/")
		if !ok || strings.TrimSpace(itemKind) == "" || strings.TrimSpace(name) == "" {
			return "", fmt.Errorf("backlog-item id %q must be kind/name", id)
		}
		if err := validateDomainToken(name); err != nil {
			return "", err
		}
		if l.BacklogItemDir == nil {
			return "", fmt.Errorf("FSLocator has no BacklogItemDir resolver")
		}
		dir, err := l.BacklogItemDir(itemKind, name)
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, agentOpsSubdir), nil
	case agentops.TargetInitiative:
		if l.InitiativeDir == nil {
			return "", fmt.Errorf("FSLocator has no InitiativeDir resolver")
		}
		dir, err := l.InitiativeDir(id)
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, agentOpsSubdir), nil
	case agentops.TargetPlanExecution:
		if l.PlanExecutionDir == nil {
			return "", fmt.Errorf("FSLocator has no PlanExecutionDir resolver")
		}
		dir, err := l.PlanExecutionDir(id)
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, agentOpsSubdir), nil
	case agentops.TargetScenario:
		if l.ScenarioDir == nil {
			return "", fmt.Errorf("FSLocator has no ScenarioDir resolver")
		}
		dir, err := l.ScenarioDir(id)
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, agentOpsSubdir), nil
	default:
		return "", fmt.Errorf("no domain workflow storage for target kind %q", kind)
	}
}

// Scan walks the configured roots and returns every agentops directory found.
func (l FSLocator) Scan() ([]string, error) {
	var out []string
	for _, root := range l.ScanRoots {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if d.IsDir() && d.Name() == agentOpsSubdir {
				out = append(out, path)
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// FSOverrideStore reads layered binding overrides from domain storage. Each
// override document lives at "<entity agentops dir>/binding-overrides/*.json".
type FSOverrideStore struct {
	loc DomainLocator
	// InitiativeOfItem, when set, returns the initiative name a backlog item
	// belongs to (empty when none), so an item target also picks up its
	// initiative's overrides. Optional; nil means item targets see only their own
	// overrides.
	InitiativeOfItem func(itemRef string) (string, error)
}

// NewFSOverrideStore constructs an override store over a domain locator.
func NewFSOverrideStore(loc DomainLocator) *FSOverrideStore {
	return &FSOverrideStore{loc: loc}
}

const overridesSubdir = "binding-overrides"

// OverridesFor returns the in-scope override bindings for a target and operation.
func (s *FSOverrideStore) OverridesFor(ctx context.Context, target TargetRef, operation agentops.OperationID) ([]agentops.OperationBinding, error) {
	var out []agentops.OperationBinding
	own, err := s.readOverrides(target.Kind, target.ID, operation)
	if err != nil {
		return nil, err
	}
	out = append(out, own...)

	if target.Kind == agentops.TargetBacklogItem && s.InitiativeOfItem != nil {
		initName, err := s.InitiativeOfItem(target.ID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(initName) != "" {
			initOverrides, err := s.readOverrides(agentops.TargetInitiative, initName, operation)
			if err != nil {
				return nil, err
			}
			out = append(out, initOverrides...)
		}
	}
	return out, nil
}

func (s *FSOverrideStore) readOverrides(kind agentops.TargetKind, id string, operation agentops.OperationID) ([]agentops.OperationBinding, error) {
	dir, err := s.loc.AgentOpsDir(kind, id)
	if err != nil {
		return nil, err
	}
	overrideDir := filepath.Join(dir, overridesSubdir)
	entries, err := os.ReadDir(overrideDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []agentops.OperationBinding
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(overrideDir, e.Name()))
		if err != nil {
			return nil, err
		}
		// A malformed override is fail-closed: return the error rather than
		// dropping it, so a lower layer never silently wins against the operator.
		if err := agentops.ValidateBinding(raw); err != nil {
			return nil, fmt.Errorf("invalid binding override %s: %w", filepath.Join(overrideDir, e.Name()), err)
		}
		b, err := agentops.DecodeBinding(raw)
		if err != nil {
			return nil, err
		}
		if b.Operation != operation {
			continue
		}
		out = append(out, b)
	}
	return out, nil
}
