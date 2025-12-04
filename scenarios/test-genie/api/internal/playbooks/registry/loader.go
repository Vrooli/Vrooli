package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"test-genie/internal/playbooks/types"
)

const (
	// FolderName is the name of the playbooks folder within the test directory.
	FolderName = "playbooks"
	// RegistryFileName is the name of the registry file.
	RegistryFileName = "registry.json"
)

var (
	// ErrNotFound indicates the registry file was not found.
	ErrNotFound = errors.New("registry not found")
	// ErrParse indicates the registry file could not be parsed.
	ErrParse = errors.New("registry parse failed")
)

// Loader defines the interface for loading playbook registries.
type Loader interface {
	// Load loads and parses the playbook registry.
	Load() (types.Registry, error)
}

// FileLoader loads registries from the filesystem.
type FileLoader struct {
	testDir string
}

// NewLoader creates a new registry loader for the given test directory.
func NewLoader(testDir string) *FileLoader {
	return &FileLoader{testDir: testDir}
}

// Load loads and parses the playbook registry from disk.
// It handles the deprecated_playbooks fallback and sorts entries by order.
func (l *FileLoader) Load() (types.Registry, error) {
	var registry types.Registry

	registryPath := l.RegistryPath()
	data, err := os.ReadFile(registryPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return registry, fmt.Errorf("%w: %s", ErrNotFound, registryPath)
		}
		return registry, fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, &registry); err != nil {
		return registry, fmt.Errorf("%w: %s: %v", ErrParse, registryPath, err)
	}

	// Fallback to deprecated playbooks if no playbooks defined
	if len(registry.Playbooks) == 0 && len(registry.Deprecated) > 0 {
		registry.Playbooks = registry.Deprecated
	}

	// Sort by order, then by file name for stable ordering
	sortEntries(registry.Playbooks)

	return registry, nil
}

// RegistryPath returns the full path to the registry file.
func (l *FileLoader) RegistryPath() string {
	return filepath.Join(l.testDir, FolderName, RegistryFileName)
}

// sortEntries sorts playbook entries by order, then by file name.
func sortEntries(entries []types.Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Order == entries[j].Order {
			return entries[i].File < entries[j].File
		}
		return entries[i].Order < entries[j].Order
	})
}
