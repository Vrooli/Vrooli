package initiatives

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
)

// initiativeFileName is the metadata file inside each initiative folder.
const initiativeFileName = "initiative.json"

// Store manages initiative persistence on the local filesystem.
// Each initiative is stored as a folder at {baseDir}/.vrooli/initiatives/{name}/
// containing an initiative.json metadata file and any additional context files.
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

// InitDir returns the absolute path for an initiative's folder.
func (s *Store) InitDir(name string) string {
	return filepath.Join(s.dir, name)
}

// filePath returns the absolute path for an initiative's metadata file.
func (s *Store) filePath(name string) string {
	return filepath.Join(s.InitDir(name), initiativeFileName)
}

// Exists reports whether an initiative with the given name exists on disk.
func (s *Store) Exists(name string) bool {
	info, err := os.Stat(s.InitDir(name))
	return err == nil && info.IsDir()
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
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		init, loadErr := s.Load(name)
		if loadErr != nil {
			continue // skip malformed entries
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

// Save writes an initiative to disk atomically. Creates the initiative folder
// if it does not already exist.
func (s *Store) Save(init *Initiative) error {
	if strings.TrimSpace(init.Name) == "" {
		return fmt.Errorf("initiative name is required")
	}
	// WriteJSONAtomic handles MkdirAll for the parent directory.
	return storage.WriteJSONAtomic(s.filePath(init.Name), init)
}

// Delete removes an initiative and all its files from disk.
// Returns nil if already absent.
func (s *Store) Delete(name string) error {
	dir := s.InitDir(name)
	err := os.RemoveAll(dir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete initiative %q: %w", name, err)
	}
	return nil
}

// Migrate moves old-style flat {name}.json files into {name}/initiative.json
// folder layout. This is idempotent — it only processes .json files at the
// root of the initiatives directory.
func (s *Store) Migrate() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read initiatives directory for migration: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		oldPath := filepath.Join(s.dir, entry.Name())
		newDir := s.InitDir(name)
		newPath := filepath.Join(newDir, initiativeFileName)

		if err := os.MkdirAll(newDir, 0o755); err != nil {
			log.Printf("[initiatives] migrate: failed to create directory for %q: %v", name, err)
			continue
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			log.Printf("[initiatives] migrate: failed to move %q: %v", name, err)
			continue
		}
		log.Printf("[initiatives] migrated %q to folder layout", name)
	}
	return nil
}
