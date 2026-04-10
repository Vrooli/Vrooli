// Package deploy provides deploy target management for desktop artifact deployment.
//
// Deploy targets are stored in .vrooli/deploy-targets.json and describe which
// LPBS instance and remote profile to deploy through.
package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// DeployTarget describes a landing page deployment endpoint.
type DeployTarget struct {
	Label                      string `json:"label"`
	ScenarioName               string `json:"scenario_name"`
	RemoteProfile              string `json:"remote_profile"`
	DeploymentManagerProfileID string `json:"deployment_manager_profile_id,omitempty"`
}

// deployTargetsFile is the on-disk schema.
type deployTargetsFile struct {
	SchemaVersion string                   `json:"schema_version"`
	Targets       map[string]*DeployTarget `json:"targets"`
}

// TargetRepository reads and writes .vrooli/deploy-targets.json.
type TargetRepository struct {
	vrooliRoot string
	mu         sync.Mutex
}

// NewTargetRepository creates a repository rooted at the given Vrooli root.
func NewTargetRepository(vrooliRoot string) *TargetRepository {
	return &TargetRepository{vrooliRoot: vrooliRoot}
}

func (r *TargetRepository) filePath() string {
	return filepath.Join(r.vrooliRoot, ".vrooli", "deploy-targets.json")
}

func (r *TargetRepository) load() (*deployTargetsFile, error) {
	data, err := os.ReadFile(r.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &deployTargetsFile{
				SchemaVersion: "1.0",
				Targets:       make(map[string]*DeployTarget),
			}, nil
		}
		return nil, fmt.Errorf("read deploy targets: %w", err)
	}
	var f deployTargetsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse deploy targets: %w", err)
	}
	if f.Targets == nil {
		f.Targets = make(map[string]*DeployTarget)
	}
	return &f, nil
}

func (r *TargetRepository) save(f *deployTargetsFile) error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode deploy targets: %w", err)
	}
	dir := filepath.Dir(r.filePath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create deploy targets dir: %w", err)
	}
	return os.WriteFile(r.filePath(), data, 0o644)
}

// Get returns a single deploy target by name.
func (r *TargetRepository) Get(name string) (*DeployTarget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := r.load()
	if err != nil {
		return nil, err
	}
	t, ok := f.Targets[name]
	if !ok {
		return nil, fmt.Errorf("deploy target %q not found", name)
	}
	return t, nil
}

// List returns all deploy targets.
func (r *TargetRepository) List() (map[string]*DeployTarget, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := r.load()
	if err != nil {
		return nil, err
	}
	return f.Targets, nil
}

// Save creates or updates a deploy target.
func (r *TargetRepository) Save(name string, target *DeployTarget) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := r.load()
	if err != nil {
		return err
	}
	f.Targets[name] = target
	return r.save(f)
}

// Delete removes a deploy target by name.
func (r *TargetRepository) Delete(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, err := r.load()
	if err != nil {
		return err
	}
	if _, ok := f.Targets[name]; !ok {
		return fmt.Errorf("deploy target %q not found", name)
	}
	delete(f.Targets, name)
	return r.save(f)
}
