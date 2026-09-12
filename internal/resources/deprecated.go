package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
)

type DeprecatedResource struct {
	Name                string `json:"name"`
	DeprecatedAt        string `json:"deprecated_at"`
	Reason              string `json:"reason"`
	Replacement         string `json:"replacement,omitempty"`
	ArchivePath         string `json:"archive_path,omitempty"`
	ArchiveHash         string `json:"archive_hash,omitempty"`
	RetentionPolicyDays int    `json:"retention_policy_days"`
	RestoreSupported    bool   `json:"restore_supported"`
	PurgeAfter          string `json:"purge_after"`
	PurgedAt            string `json:"purged_at,omitempty"`
}

type DeprecatedResourceList struct {
	Resources []DeprecatedResource `json:"resources"`
}

type DeprecationReport struct {
	Resource   DeprecatedResource `json:"resource"`
	Archived   bool               `json:"archived"`
	ArchiveDir string             `json:"archive_dir,omitempty"`
}

type RestoreReport struct {
	Resource     DeprecatedResource `json:"resource"`
	Restored     bool               `json:"restored"`
	RestoredPath string             `json:"restored_path,omitempty"`
}

type ArchiveGCItem struct {
	Name        string `json:"name"`
	ArchivePath string `json:"archive_path,omitempty"`
	Removed     bool   `json:"removed"`
}

type ArchiveGCReport struct {
	Removed []ArchiveGCItem `json:"removed"`
	Skipped []ArchiveGCItem `json:"skipped"`
}

type archiveSource struct {
	kind       string
	sourcePath string
	targetPath string
	bytes      []byte
}

type ArchiveSkippedPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type archiveCollection struct {
	Sources []archiveSource      `json:"sources"`
	Skipped []ArchiveSkippedPath `json:"skipped,omitempty"`
}

func (c *Controller) ListDeprecatedResources() ([]DeprecatedResource, error) {
	items, err := c.loadDeprecatedResources()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (c *Controller) DeprecatedResource(name string) (DeprecatedResource, bool, error) {
	items, err := c.loadDeprecatedResources()
	if err != nil {
		return DeprecatedResource{}, false, err
	}
	for _, item := range items {
		if item.Name == strings.TrimSpace(name) {
			return item, true, nil
		}
	}
	return DeprecatedResource{}, false, nil
}

func (c *Controller) IsDeprecated(name string) (bool, error) {
	_, ok, err := c.DeprecatedResource(name)
	return ok, err
}

func (c *Controller) DeprecateResource(name string) (DeprecationReport, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return DeprecationReport{}, fmt.Errorf("resource name cannot be empty")
	}

	if !c.resourceKnown(name) {
		return DeprecationReport{}, fmt.Errorf("resource %q not found in resources or config", name)
	}

	if item, ok, err := c.DeprecatedResource(name); err != nil {
		return DeprecationReport{}, err
	} else if ok {
		return DeprecationReport{Resource: item, Archived: item.ArchivePath != "", ArchiveDir: item.ArchivePath}, nil
	}

	now := time.Now().UTC()
	collection, err := c.collectArchiveSources(name)
	if err != nil {
		return DeprecationReport{}, err
	}
	archiveBase, err := c.archiveRoot()
	if err != nil {
		return DeprecationReport{}, err
	}
	archiveDir := filepath.Join(archiveBase, fmt.Sprintf("%s-%s", now.Format("20060102-150405"), name))
	if _, err := config.EnsureOwnedDir(archiveDir); err != nil {
		return DeprecationReport{}, fmt.Errorf("create archive dir %s: %w", archiveDir, err)
	}
	archiveHash, err := writeArchive(archiveDir, collection.Sources)
	if err != nil {
		return DeprecationReport{}, err
	}
	if len(collection.Skipped) > 0 {
		if err := writeJSONMetadata(filepath.Join(archiveDir, "archive-skipped-paths.json"), map[string]any{"paths": collection.Skipped}); err != nil {
			return DeprecationReport{}, err
		}
	}

	record := DeprecatedResource{
		Name:                name,
		DeprecatedAt:        now.Format(time.DateOnly),
		Reason:              "Deprecated during Phase 2 resource cross-platform migration triage.",
		Replacement:         c.deprecationReplacement(name),
		ArchivePath:         archiveDir,
		ArchiveHash:         archiveHash,
		RetentionPolicyDays: defaultRetentionDays,
		RestoreSupported:    true,
		PurgeAfter:          now.AddDate(0, 0, defaultRetentionDays).Format(time.DateOnly),
	}
	if err := writeArchiveMetadata(filepath.Join(archiveDir, "deprecated-resource.json"), record); err != nil {
		return DeprecationReport{}, err
	}
	if err := c.removeActiveResourceState(name); err != nil {
		return DeprecationReport{}, err
	}
	if err := c.appendDeprecatedResource(record); err != nil {
		return DeprecationReport{}, err
	}
	return DeprecationReport{Resource: record, Archived: true, ArchiveDir: archiveDir}, nil
}

func (c *Controller) RestoreDeprecatedResource(name string) (RestoreReport, error) {
	item, ok, err := c.DeprecatedResource(name)
	if err != nil {
		return RestoreReport{}, err
	}
	if !ok {
		return RestoreReport{}, fmt.Errorf("deprecated resource %q not found", name)
	}
	if !item.RestoreSupported {
		return RestoreReport{}, fmt.Errorf("deprecated resource %q is not restoreable", name)
	}
	if strings.TrimSpace(item.PurgedAt) != "" {
		return RestoreReport{}, fmt.Errorf("deprecated resource %q has already been purged", name)
	}
	if strings.TrimSpace(item.ArchivePath) == "" {
		return RestoreReport{}, fmt.Errorf("deprecated resource %q does not have an archive path", name)
	}

	destRoot := filepath.Join(c.Root, filepath.FromSlash(restoredResourcesDirPath), name)
	if err := os.RemoveAll(destRoot); err != nil {
		return RestoreReport{}, fmt.Errorf("clear restored path %s: %w", destRoot, err)
	}
	if err := os.MkdirAll(destRoot, tuning.PermDir); err != nil {
		return RestoreReport{}, fmt.Errorf("create restored path %s: %w", destRoot, err)
	}

	sourceRoot := filepath.Join(item.ArchivePath, "files")
	if _, err := os.Stat(sourceRoot); err != nil {
		return RestoreReport{}, fmt.Errorf("archive files for %q are unavailable: %w", name, err)
	}
	if err := copyDir(sourceRoot, destRoot); err != nil {
		return RestoreReport{}, err
	}
	return RestoreReport{
		Resource:     item,
		Restored:     true,
		RestoredPath: destRoot,
	}, nil
}

func (c *Controller) GarbageCollectDeprecatedArchives(now time.Time) (ArchiveGCReport, error) {
	items, err := c.loadDeprecatedResources()
	if err != nil {
		return ArchiveGCReport{}, err
	}
	report := ArchiveGCReport{
		Removed: make([]ArchiveGCItem, 0),
		Skipped: make([]ArchiveGCItem, 0),
	}
	changed := false
	for i := range items {
		item := &items[i]
		if strings.TrimSpace(item.PurgedAt) != "" {
			report.Skipped = append(report.Skipped, ArchiveGCItem{Name: item.Name, ArchivePath: item.ArchivePath, Removed: false})
			continue
		}
		purgeAfter, err := time.Parse(time.DateOnly, item.PurgeAfter)
		if err != nil || purgeAfter.After(now.UTC()) {
			report.Skipped = append(report.Skipped, ArchiveGCItem{Name: item.Name, ArchivePath: item.ArchivePath, Removed: false})
			continue
		}
		if strings.TrimSpace(item.ArchivePath) != "" {
			if err := os.RemoveAll(item.ArchivePath); err != nil {
				return ArchiveGCReport{}, fmt.Errorf("remove archive %s: %w", item.ArchivePath, err)
			}
		}
		item.PurgedAt = now.UTC().Format(time.DateOnly)
		changed = true
		report.Removed = append(report.Removed, ArchiveGCItem{Name: item.Name, ArchivePath: item.ArchivePath, Removed: true})
	}
	if changed {
		if err := c.writeDeprecatedResources(items); err != nil {
			return ArchiveGCReport{}, err
		}
	}
	return report, nil
}

func (c *Controller) loadDeprecatedResources() ([]DeprecatedResource, error) {
	path := filepath.Join(c.Root, filepath.FromSlash(deprecatedResourcesPath))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []DeprecatedResource{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var payload DeprecatedResourceList
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if payload.Resources == nil {
		payload.Resources = []DeprecatedResource{}
	}
	for _, item := range payload.Resources {
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("parse %s: deprecated resource name cannot be empty", path)
		}
	}
	return payload.Resources, nil
}

func (c *Controller) appendDeprecatedResource(item DeprecatedResource) error {
	items, err := c.loadDeprecatedResources()
	if err != nil {
		return err
	}
	items = append(items, item)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return c.writeDeprecatedResources(items)
}

func (c *Controller) writeDeprecatedResources(items []DeprecatedResource) error {
	path := filepath.Join(c.Root, filepath.FromSlash(deprecatedResourcesPath))
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return fmt.Errorf("create deprecated resource metadata dir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(DeprecatedResourceList{Resources: items}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, tuning.PermFile)
}

func (c *Controller) collectArchiveSources(name string) (archiveCollection, error) {
	collection := archiveCollection{
		Sources: make([]archiveSource, 0),
		Skipped: make([]ArchiveSkippedPath, 0),
	}

	resourcePath := filepath.Join(c.Root, "resources", name)
	if info, err := os.Stat(resourcePath); err == nil && info.IsDir() {
		dirCollection, err := archiveSourcesFromDir(resourcePath, filepath.Join("resource", name))
		if err != nil {
			return archiveCollection{}, err
		}
		collection.Sources = append(collection.Sources, dirCollection.Sources...)
		collection.Skipped = append(collection.Skipped, dirCollection.Skipped...)
	}

	if entry, ok, err := c.serviceConfigEntry(name); err != nil {
		return archiveCollection{}, err
	} else if ok {
		data, marshalErr := json.MarshalIndent(entry, "", "  ")
		if marshalErr != nil {
			return archiveCollection{}, marshalErr
		}
		data = append(data, '\n')
		collection.Sources = append(collection.Sources, archiveSource{
			kind:       "service-config",
			targetPath: filepath.Join("config", "service-entry.json"),
			bytes:      data,
		})
	}

	if len(collection.Sources) == 0 {
		return archiveCollection{}, fmt.Errorf("resource %q has no archiveable implementation or config state", name)
	}
	return collection, nil
}

func (c *Controller) removeActiveResourceState(name string) error {
	if err := c.removeResourceDirectory(name); err != nil {
		return err
	}
	if err := c.removeResourceConfigEntry(name); err != nil {
		return err
	}
	return nil
}

func (c *Controller) removeResourceDirectory(name string) error {
	path := filepath.Join(c.Root, "resources", name)
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			archiveBase, archiveErr := c.archiveRoot()
			if archiveErr != nil {
				return fmt.Errorf("remove resource directory %s: %w", path, err)
			}
			remnantsRoot := filepath.Join(archiveBase, "remnants")
			if _, mkErr := config.EnsureOwnedDir(remnantsRoot); mkErr != nil {
				return fmt.Errorf("remove resource directory %s: %w", path, err)
			}
			target := filepath.Join(remnantsRoot, fmt.Sprintf("%s-%s", time.Now().UTC().Format("20060102-150405"), name))
			if renameErr := os.Rename(path, target); renameErr == nil { //nolint:forbidigo // intentional directory archival
				return nil
			}
			return fmt.Errorf("remove resource directory %s: %w", path, err)
		}
	}
	return nil
}

func (c *Controller) removeResourceConfigEntry(name string) error {
	configPath := filepath.Join(c.Root, filepath.FromSlash(resourceConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	dependencies := ensureObject(payload, "dependencies")
	resources := ensureObject(dependencies, "resources")
	if _, ok := resources[name]; !ok {
		return nil
	}
	delete(resources, name)
	updated, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	updated = append(updated, '\n')
	return os.WriteFile(configPath, updated, tuning.PermFile)
}

func (c *Controller) serviceConfigEntry(name string) (map[string]any, bool, error) {
	configPath := filepath.Join(c.Root, filepath.FromSlash(resourceConfigPath))
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, false, err
	}
	dependencies, ok := payload["dependencies"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	resources, ok := dependencies["resources"].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	entry, ok := resources[name].(map[string]any)
	if !ok {
		return nil, false, nil
	}
	return entry, true, nil
}

func (c *Controller) resourceKnown(name string) bool {
	if _, ok, err := c.DeprecatedResource(name); err == nil && ok {
		return true
	}
	if _, err := os.Stat(filepath.Join(c.Root, "resources", name)); err == nil {
		return true
	}
	entries, err := c.readConfigEntries()
	if err == nil {
		if _, ok := entries[name]; ok {
			return true
		}
	}
	return false
}

func (c *Controller) deprecationReplacement(name string) string {
	if _, err := c.Blueprint(name); err == nil {
		return name
	}
	return ""
}

func (c *Controller) archiveRoot() (string, error) {
	home := strings.TrimSpace(c.Home)
	if home == "" {
		// Sudo-aware: never bare os.UserHomeDir (would archive under /root).
		resolved, err := config.HomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home for resource archive: %w", err)
		}
		home = resolved
	}
	root, err := repocontract.VrooliUserRoot(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "archive", "resources"), nil
}

func archiveSourcesFromDir(root, targetPrefix string) (archiveCollection, error) {
	items := make([]archiveSource, 0)
	skipped := make([]ArchiveSkippedPath, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if err != nil {
			if os.IsPermission(err) {
				skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: "permission-denied"})
				return filepath.SkipDir
			}
			return err
		}
		if rel != "." {
			if reason, skip := archiveSkipReason(rel); skip {
				skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: reason})
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			info, statErr := os.Stat(path)
			if statErr != nil {
				if os.IsPermission(statErr) || os.IsNotExist(statErr) {
					skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: "unreadable-symlink"})
					return nil
				}
				return statErr
			}
			if info.IsDir() {
				skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: "symlinked-directory"})
				return nil
			}
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			if os.IsPermission(readErr) {
				skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: "permission-denied"})
				return nil
			}
			if strings.Contains(readErr.Error(), "is a directory") {
				skipped = append(skipped, ArchiveSkippedPath{Path: rel, Reason: "directory-like-entry"})
				return nil
			}
			return readErr
		}
		items = append(items, archiveSource{
			kind:       "resource",
			sourcePath: path,
			targetPath: filepath.Join(targetPrefix, rel),
			bytes:      data,
		})
		return nil
	})
	if err != nil {
		return archiveCollection{}, fmt.Errorf("collect archive sources from %s: %w", root, err)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].targetPath < items[j].targetPath
	})
	sort.Slice(skipped, func(i, j int) bool {
		return skipped[i].Path < skipped[j].Path
	})
	return archiveCollection{Sources: items, Skipped: skipped}, nil
}

func archiveSkipReason(rel string) (string, bool) {
	for _, segment := range strings.Split(rel, "/") {
		switch segment {
		case ".venv":
			return "generated-virtualenv", true
		case "node_modules":
			return "generated-node-modules", true
		case "__pycache__":
			return "generated-python-cache", true
		case ".pytest_cache":
			return "generated-pytest-cache", true
		case ".mypy_cache":
			return "generated-mypy-cache", true
		}
	}
	return "", false
}

func writeArchive(dir string, sources []archiveSource) (string, error) {
	if err := os.MkdirAll(filepath.Join(dir, "files"), tuning.PermDir); err != nil {
		return "", fmt.Errorf("create archive files dir %s: %w", filepath.Join(dir, "files"), err)
	}
	hash := sha256.New()
	for _, source := range sources {
		target := filepath.Join(dir, "files", source.targetPath)
		if err := os.MkdirAll(filepath.Dir(target), tuning.PermDir); err != nil {
			return "", fmt.Errorf("create archive path %s: %w", filepath.Dir(target), err)
		}
		if err := os.WriteFile(target, source.bytes, tuning.PermFile); err != nil {
			return "", fmt.Errorf("write archive file %s: %w", target, err)
		}
		_, _ = io.WriteString(hash, filepath.ToSlash(source.targetPath))
		_, _ = hash.Write(source.bytes)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeArchiveMetadata(path string, item DeprecatedResource) error {
	return writeJSONMetadata(path, item)
}

func writeJSONMetadata(path string, item any) error {
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, tuning.PermFile)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, tuning.PermDir)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, tuning.PermFile)
	})
}
