package goals

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"swarm-manager/internal/storage"
)

// goalFileName is the metadata file inside each goal folder.
const goalFileName = "goal.json"

// Store persists goals on the local filesystem, mirroring the initiatives
// store: each goal is a folder at {dataRoot}/goals/{name}/ containing goal.json.
type Store struct {
	dir string
}

// File is an entry in a goal's on-disk folder. It deliberately exposes a
// relative path only so callers never receive host filesystem paths.
type File struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

// NewStore creates a Store rooted at {dataRoot}/goals.
func NewStore(dataRoot string) *Store {
	return &Store{dir: filepath.Join(dataRoot, "goals")}
}

// GoalDir returns the absolute path for a goal's folder.
func (s *Store) GoalDir(name string) string {
	return filepath.Join(s.dir, name)
}

func (s *Store) filePath(name string) string {
	return filepath.Join(s.GoalDir(name), goalFileName)
}

// Exists reports whether a goal with the given name exists on disk.
func (s *Store) Exists(name string) bool {
	info, err := os.Stat(s.GoalDir(name))
	return err == nil && info.IsDir()
}

// Load reads a single goal from disk.
func (s *Store) Load(name string) (*Goal, error) {
	var g Goal
	found, err := storage.ReadJSON(s.filePath(name), &g)
	if err != nil {
		return nil, fmt.Errorf("load goal %q: %w", name, err)
	}
	if !found {
		return nil, fmt.Errorf("goal %q not found", name)
	}
	return &g, nil
}

// LoadAll reads all goals from disk, sorted by name.
func (s *Store) LoadAll() ([]Goal, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Goal{}, nil
		}
		return nil, fmt.Errorf("read goals directory: %w", err)
	}
	var out []Goal
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		g, loadErr := s.Load(entry.Name())
		if loadErr != nil {
			continue // skip malformed entries
		}
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	if out == nil {
		out = []Goal{}
	}
	return out, nil
}

// Save writes a goal to disk atomically.
func (s *Store) Save(g *Goal) error {
	if strings.TrimSpace(g.Name) == "" {
		return fmt.Errorf("goal name is required")
	}
	return storage.WriteJSONAtomic(s.filePath(g.Name), g)
}

// Delete removes a goal and all its files. Returns nil if already absent.
func (s *Store) Delete(name string) error {
	if err := os.RemoveAll(s.GoalDir(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete goal %q: %w", name, err)
	}
	return nil
}

// ListFiles returns all regular files held by a goal, including migrated
// initiative material.
func (s *Store) ListFiles(name string) ([]File, error) {
	root := s.GoalDir(name)
	if !s.Exists(name) {
		return nil, fmt.Errorf("goal %q not found", name)
	}
	files := make([]File, 0)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		files = append(files, File{Path: filepath.ToSlash(rel), Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list goal %q files: %w", name, err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
