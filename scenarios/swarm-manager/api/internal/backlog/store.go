// Store provides filesystem CRUD operations for backlog items.
// It encapsulates all direct disk access (reading/writing spec.json files,
// walking kind directories) behind a clean interface, keeping HTTP concerns
// out of the data layer.
package backlog

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

)

// Store abstracts persistence for backlog items. The primary implementation
// is FileStore (filesystem-backed); tests may inject alternatives for fault
// injection or isolation.
type Store interface {
	KindDir(kind BacklogKind) string
	ItemDir(kind BacklogKind, name string) string
	LoadAll(kinds []BacklogKind) ([]BacklogItem, error)
	LoadItem(kind BacklogKind, name string) (BacklogItem, error)
	LoadItemFromPath(kind BacklogKind, specPath string) (BacklogItem, error)
	SaveItem(item BacklogItem) error
	ValidateDependencies(dependsOn []string) error
	CheckDependencies(dependsOn []string) ([]string, error)
}

// FileStore is the filesystem-backed Store implementation. It reads and writes
// backlog items as spec.json files in kind-specific directories.
type FileStore struct {
	rootDir string
}

// NewFileStore creates a FileStore rooted at the given directory.
func NewFileStore(rootDir string) *FileStore {
	return &FileStore{rootDir: rootDir}
}

// KindDir returns the absolute path for a given backlog kind directory.
func (s *FileStore) KindDir(kind BacklogKind) string {
	return filepath.Join(s.rootDir, backlogKindDirs[kind])
}

// ItemDir returns the absolute path for a specific backlog item.
func (s *FileStore) ItemDir(kind BacklogKind, name string) string {
	return filepath.Join(s.KindDir(kind), name)
}

// LoadAll reads all backlog items from the specified kinds. If kinds is empty,
// all known kinds are loaded.
func (s *FileStore) LoadAll(kinds []BacklogKind) ([]BacklogItem, error) {
	var items []BacklogItem

	if len(kinds) == 0 {
		kinds = []BacklogKind{KindIdea, KindResearch, KindFix, KindExecute, KindChore}
	}

	for _, kind := range kinds {
		kindDir := s.KindDir(kind)
		err := filepath.WalkDir(kindDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) && path == kindDir {
					return nil
				}
				return err
			}

			if d.IsDir() && path != kindDir {
				specPath := filepath.Join(path, "spec.json")
				if _, err := os.Stat(specPath); err == nil {
					item, err := s.LoadItemFromPath(kind, specPath)
					if err == nil {
						items = append(items, item)
					}
				}
				return fs.SkipDir
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	if items == nil {
		items = []BacklogItem{}
	}
	return items, nil
}

// LoadItem reads a single backlog item by kind and name.
// Returns ErrNotFound if the item does not exist.
func (s *FileStore) LoadItem(kind BacklogKind, name string) (BacklogItem, error) {
	specPath := filepath.Join(s.ItemDir(kind, name), "spec.json")
	return s.LoadItemFromPath(kind, specPath)
}

// LoadItemFromPath reads a backlog item from the given spec.json path.
// It normalizes legacy status values, backfills missing timestamps, and
// clamps priority to the valid 1-10 range.
// Returns ErrNotFound if the spec.json file does not exist.
func (s *FileStore) LoadItemFromPath(kind BacklogKind, specPath string) (BacklogItem, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			return BacklogItem{}, fmt.Errorf("%w: %s", ErrNotFound, specPath)
		}
		return BacklogItem{}, err
	}

	var item BacklogItem
	if err := json.Unmarshal(data, &item); err != nil {
		return BacklogItem{}, err
	}

	item.Name = filepath.Base(filepath.Dir(specPath))
	item.Kind = kind
	if item.Tags == nil {
		item.Tags = []string{}
	}
	// Normalize status to valid proto values. On-disk data may contain
	// legacy values (e.g. "done") that are not in the proto enum.
	if !validateBacklogStatus(string(item.Status)) {
		switch string(item.Status) {
		case "done", "complete", "finished":
			item.Status = StatusCompleted
		default:
			item.Status = StatusBacklog
		}
	}
	// Backfill missing created timestamp from updated or file mtime.
	if strings.TrimSpace(item.Created) == "" {
		if strings.TrimSpace(item.Updated) != "" {
			item.Created = item.Updated
		} else if info, statErr := os.Stat(specPath); statErr == nil {
			item.Created = info.ModTime().UTC().Format(time.RFC3339)
		} else {
			item.Created = time.Now().UTC().Format(time.RFC3339)
		}
	}
	// Normalize effort to uppercase if present.
	if item.Effort != "" {
		item.Effort = strings.ToUpper(strings.TrimSpace(item.Effort))
	}
	// Ensure priority is within valid range (1-10).
	if item.Priority < 1 {
		item.Priority = 5
	} else if item.Priority > 10 {
		item.Priority = 10
	}
	return item, nil
}

// SaveItem writes a backlog item's spec.json to disk. It performs a
// read-modify-write to preserve unknown fields (such as archive metadata)
// that are not part of the BacklogItem struct.
func (s *FileStore) SaveItem(item BacklogItem) error {
	if item.Kind == "" {
		return fmt.Errorf("backlog kind is required")
	}
	specPath := filepath.Join(s.ItemDir(item.Kind, item.Name), "spec.json")

	// Preserve archive and other unknown metadata fields when rewriting spec.json.
	merged := map[string]any{}
	if existing, err := os.ReadFile(specPath); err == nil {
		if unmarshalErr := json.Unmarshal(existing, &merged); unmarshalErr != nil {
			slog.Warn("existing spec.json has malformed JSON, metadata may be lost", "kind", item.Kind, "name", item.Name, "err", unmarshalErr)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	merged["name"] = item.Name
	merged["title"] = item.Title
	merged["description"] = item.Description
	merged["status"] = item.Status
	merged["priority"] = item.Priority
	merged["tags"] = item.Tags
	merged["created"] = item.Created
	merged["updated"] = item.Updated
	merged["kind"] = item.Kind
	delete(merged, "research_target")
	if len(item.DependsOn) > 0 {
		merged["depends_on"] = item.DependsOn
	} else {
		delete(merged, "depends_on")
	}
	if strings.TrimSpace(item.Initiative) != "" {
		merged["initiative"] = item.Initiative
	} else {
		delete(merged, "initiative")
	}
	if strings.TrimSpace(item.Effort) != "" {
		merged["effort"] = item.Effort
	} else {
		delete(merged, "effort")
	}
	if len(item.AcceptanceAllow) > 0 {
		merged["acceptance_allow"] = item.AcceptanceAllow
	} else {
		delete(merged, "acceptance_allow")
	}
	if len(item.AcceptanceDeny) > 0 {
		merged["acceptance_deny"] = item.AcceptanceDeny
	} else {
		delete(merged, "acceptance_deny")
	}
	if strings.TrimSpace(item.SpawnedFrom) != "" {
		merged["spawned_from"] = item.SpawnedFrom
	} else {
		delete(merged, "spawned_from")
	}
	if strings.TrimSpace(item.Note) != "" {
		merged["note"] = item.Note
	} else {
		delete(merged, "note")
	}
	if len(item.SuggestedSkills) > 0 {
		merged["suggested_skills"] = item.SuggestedSkills
	} else {
		delete(merged, "suggested_skills")
	}

	if item.ArchivedAt != nil {
		merged["archived_at"] = *item.ArchivedAt
	} else {
		delete(merged, "archived_at")
	}

	// Preserve immutable created_by: once set on disk, never overwrite.
	if _, exists := merged["created_by"]; !exists && item.CreatedBy != nil {
		merged["created_by"] = item.CreatedBy
	}

	data, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(specPath, data, 0o644)
}

// parseDependencyRef splits a "kind/name" dependency reference into its
// components. Returns an error if the format is invalid.
func parseDependencyRef(ref string) (BacklogKind, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", fmt.Errorf("invalid dependency reference %q: expected format kind/name", ref)
	}
	kind, err := ParseBacklogKind(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid dependency reference %q: %w", ref, err)
	}
	return kind, parts[1], nil
}

// ValidateDependencies checks that all depends_on references exist on disk.
func (s *FileStore) ValidateDependencies(dependsOn []string) error {
	for _, ref := range dependsOn {
		kind, name, err := parseDependencyRef(ref)
		if err != nil {
			return err
		}
		itemDir := s.ItemDir(kind, name)
		if _, statErr := os.Stat(itemDir); os.IsNotExist(statErr) {
			return fmt.Errorf("dependency %q does not exist", ref)
		}
	}
	return nil
}

// CheckDependencies returns the subset of depends_on references that are still
// in an unplanned state (backlog or researching). A dependency whose spec no
// longer exists on disk is presumed completed/archived and treated as
// satisfied. Dependencies that have progressed past the planning phase (ready,
// queued, in_progress, completed, failed, archived) are not blocking.
func (s *FileStore) CheckDependencies(dependsOn []string) ([]string, error) {
	var unmet []string
	for _, ref := range dependsOn {
		kind, name, err := parseDependencyRef(ref)
		if err != nil {
			// Unparseable refs are skipped — they cannot be validated and
			// should not block execution.
			continue
		}
		item, loadErr := s.LoadItem(kind, name)
		if loadErr != nil {
			// Missing/unloadable specs are presumed completed & archived.
			// A dependency that no longer exists on disk should never block
			// execution — it is valid for completed work to be cleaned up.
			continue
		}
		if blockingDepStatuses[item.Status] {
			unmet = append(unmet, ref)
		}
	}
	return unmet, nil
}
