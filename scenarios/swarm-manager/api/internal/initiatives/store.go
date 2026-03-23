package initiatives

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
)

// Store manages initiative persistence on the local filesystem.
// Each initiative is stored as a separate JSON file at
// {baseDir}/.vrooli/initiatives/{name}.json.
type Store struct {
	dir string // absolute path to the initiatives directory
}

// NewStore creates a Store. baseDir is the scenario root; initiatives are
// stored under {baseDir}/.vrooli/initiatives/.
func NewStore(baseDir string) *Store {
	return &Store{
		dir: filepath.Join(baseDir, ".vrooli", "initiatives"),
	}
}

// filePath returns the absolute path for an initiative's JSON file.
func (s *Store) filePath(name string) string {
	return filepath.Join(s.dir, name+".json")
}

// Exists reports whether an initiative with the given name exists on disk.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.filePath(name))
	return err == nil
}

// Load reads a single initiative from disk.
func (s *Store) Load(name string) (*Initiative, error) {
	path := s.filePath(name)
	var init Initiative
	found, err := storage.ReadJSON(path, &init)
	if err != nil {
		return nil, fmt.Errorf("load initiative %q: %w", name, err)
	}
	if !found {
		return nil, fmt.Errorf("initiative %q not found", name)
	}
	return &init, nil
}

// LoadAll reads all initiatives from disk, sorted by name.
func (s *Store) LoadAll() ([]Initiative, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Initiative{}, nil
		}
		return nil, fmt.Errorf("read initiatives directory: %w", err)
	}

	var initiatives []Initiative
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		init, loadErr := s.Load(name)
		if loadErr != nil {
			continue // skip malformed files
		}
		initiatives = append(initiatives, *init)
	}

	sort.Slice(initiatives, func(i, j int) bool {
		return initiatives[i].Name < initiatives[j].Name
	})

	if initiatives == nil {
		initiatives = []Initiative{}
	}
	return initiatives, nil
}

// Save writes an initiative to disk atomically.
func (s *Store) Save(init *Initiative) error {
	if strings.TrimSpace(init.Name) == "" {
		return fmt.Errorf("initiative name is required")
	}
	return storage.WriteJSONAtomic(s.filePath(init.Name), init)
}

// Delete removes an initiative from disk. Returns nil if already absent.
func (s *Store) Delete(name string) error {
	path := s.filePath(name)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete initiative %q: %w", name, err)
	}
	return nil
}
