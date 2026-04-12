package resources

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const blueprintArchivedResourcesPath = ".vrooli/blueprint-archived-resources.json"

type BlueprintArchivedResource struct {
	Name                string `json:"name"`
	ArchivedAt          string `json:"archived_at"`
	Reason              string `json:"reason"`
	BlueprintName       string `json:"blueprint_name"`
	ArchivePath         string `json:"archive_path,omitempty"`
	ArchiveHash         string `json:"archive_hash,omitempty"`
	RetentionPolicyDays int    `json:"retention_policy_days"`
	RestoreSupported    bool   `json:"restore_supported"`
	PurgeAfter          string `json:"purge_after"`
	PurgedAt            string `json:"purged_at,omitempty"`
}

type BlueprintArchivedResourceList struct {
	Resources []BlueprintArchivedResource `json:"resources"`
}

type BlueprintArchiveReport struct {
	Resource   BlueprintArchivedResource `json:"resource"`
	Archived   bool                      `json:"archived"`
	ArchiveDir string                    `json:"archive_dir,omitempty"`
}

type BlueprintRestoreReport struct {
	Resource     BlueprintArchivedResource `json:"resource"`
	Restored     bool                      `json:"restored"`
	RestoredPath string                    `json:"restored_path,omitempty"`
}

func (c *Controller) ListBlueprintArchivedResources() ([]BlueprintArchivedResource, error) {
	items, err := c.loadBlueprintArchivedResources()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (c *Controller) BlueprintArchivedResource(name string) (BlueprintArchivedResource, bool, error) {
	items, err := c.loadBlueprintArchivedResources()
	if err != nil {
		return BlueprintArchivedResource{}, false, err
	}
	for _, item := range items {
		if item.Name == strings.TrimSpace(name) {
			return item, true, nil
		}
	}
	return BlueprintArchivedResource{}, false, nil
}

func (c *Controller) IsBlueprintArchived(name string) (bool, error) {
	_, ok, err := c.BlueprintArchivedResource(name)
	return ok, err
}

func (c *Controller) ArchiveResourceToBlueprint(name string) (BlueprintArchiveReport, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return BlueprintArchiveReport{}, fmt.Errorf("resource name cannot be empty")
	}
	if deprecated, err := c.IsDeprecated(name); err != nil {
		return BlueprintArchiveReport{}, err
	} else if deprecated {
		return BlueprintArchiveReport{}, fmt.Errorf("resource %q is already deprecated; use the deprecated resource workflow", name)
	}
	if item, ok, err := c.BlueprintArchivedResource(name); err != nil {
		return BlueprintArchiveReport{}, err
	} else if ok {
		return BlueprintArchiveReport{Resource: item, Archived: item.ArchivePath != "", ArchiveDir: item.ArchivePath}, nil
	}
	if err := c.validateBlueprintArchiveCandidate(name); err != nil {
		return BlueprintArchiveReport{}, err
	}
	now := time.Now().UTC()
	sources, err := c.collectArchiveSources(name)
	if err != nil {
		return BlueprintArchiveReport{}, err
	}
	archiveDir := filepath.Join(c.archiveRoot(), fmt.Sprintf("%s-%s", now.Format("20060102-150405"), name))
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return BlueprintArchiveReport{}, fmt.Errorf("create archive dir %s: %w", archiveDir, err)
	}
	archiveHash, err := writeArchive(archiveDir, sources)
	if err != nil {
		return BlueprintArchiveReport{}, err
	}

	record := BlueprintArchivedResource{
		Name:                name,
		ArchivedAt:          now.Format(time.DateOnly),
		Reason:              "Archived from active repo implementation after blueprint preservation.",
		BlueprintName:       name,
		ArchivePath:         archiveDir,
		ArchiveHash:         archiveHash,
		RetentionPolicyDays: defaultRetentionDays,
		RestoreSupported:    true,
		PurgeAfter:          now.AddDate(0, 0, defaultRetentionDays).Format(time.DateOnly),
	}
	if err := writeJSONMetadata(filepath.Join(archiveDir, "blueprint-archived-resource.json"), record); err != nil {
		return BlueprintArchiveReport{}, err
	}
	if err := c.removeActiveResourceState(name); err != nil {
		return BlueprintArchiveReport{}, err
	}
	if err := c.appendBlueprintArchivedResource(record); err != nil {
		return BlueprintArchiveReport{}, err
	}
	return BlueprintArchiveReport{Resource: record, Archived: true, ArchiveDir: archiveDir}, nil
}

func (c *Controller) RestoreBlueprintArchivedResource(name string) (BlueprintRestoreReport, error) {
	item, ok, err := c.BlueprintArchivedResource(name)
	if err != nil {
		return BlueprintRestoreReport{}, err
	}
	if !ok {
		return BlueprintRestoreReport{}, fmt.Errorf("blueprint-archived resource %q not found", name)
	}
	if !item.RestoreSupported {
		return BlueprintRestoreReport{}, fmt.Errorf("blueprint-archived resource %q is not restorable", name)
	}
	if strings.TrimSpace(item.PurgedAt) != "" {
		return BlueprintRestoreReport{}, fmt.Errorf("blueprint-archived resource %q has already been purged", name)
	}
	if strings.TrimSpace(item.ArchivePath) == "" {
		return BlueprintRestoreReport{}, fmt.Errorf("blueprint-archived resource %q does not have an archive path", name)
	}

	destRoot := filepath.Join(c.Root, filepath.FromSlash(restoredResourcesDirPath), name)
	if err := os.RemoveAll(destRoot); err != nil {
		return BlueprintRestoreReport{}, fmt.Errorf("clear restored path %s: %w", destRoot, err)
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return BlueprintRestoreReport{}, fmt.Errorf("create restored path %s: %w", destRoot, err)
	}

	sourceRoot := filepath.Join(item.ArchivePath, "files")
	if _, err := os.Stat(sourceRoot); err != nil {
		return BlueprintRestoreReport{}, fmt.Errorf("archive files for %q are unavailable: %w", name, err)
	}
	if err := copyDir(sourceRoot, destRoot); err != nil {
		return BlueprintRestoreReport{}, err
	}
	return BlueprintRestoreReport{
		Resource:     item,
		Restored:     true,
		RestoredPath: destRoot,
	}, nil
}

func (c *Controller) GarbageCollectBlueprintArchives(now time.Time) (ArchiveGCReport, error) {
	items, err := c.loadBlueprintArchivedResources()
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
		if err := c.writeBlueprintArchivedResources(items); err != nil {
			return ArchiveGCReport{}, err
		}
	}
	return report, nil
}

func (c *Controller) loadBlueprintArchivedResources() ([]BlueprintArchivedResource, error) {
	path := filepath.Join(c.Root, filepath.FromSlash(blueprintArchivedResourcesPath))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []BlueprintArchivedResource{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var payload BlueprintArchivedResourceList
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if payload.Resources == nil {
		payload.Resources = []BlueprintArchivedResource{}
	}
	for _, item := range payload.Resources {
		if strings.TrimSpace(item.Name) == "" {
			return nil, fmt.Errorf("parse %s: blueprint archived resource name cannot be empty", path)
		}
	}
	return payload.Resources, nil
}

func (c *Controller) appendBlueprintArchivedResource(item BlueprintArchivedResource) error {
	items, err := c.loadBlueprintArchivedResources()
	if err != nil {
		return err
	}
	items = append(items, item)
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})
	return c.writeBlueprintArchivedResources(items)
}

func (c *Controller) writeBlueprintArchivedResources(items []BlueprintArchivedResource) error {
	path := filepath.Join(c.Root, filepath.FromSlash(blueprintArchivedResourcesPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create blueprint archived metadata dir %s: %w", filepath.Dir(path), err)
	}
	data, err := json.MarshalIndent(BlueprintArchivedResourceList{Resources: items}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func (c *Controller) validateBlueprintArchiveCandidate(name string) error {
	if _, err := c.Blueprint(name); err != nil {
		return fmt.Errorf("resource %q does not have a blueprint record; create or restore the blueprint before archiving", name)
	}
	if enabled, required, err := c.projectResourceFlags(name); err != nil {
		return err
	} else if enabled || required {
		return fmt.Errorf("resource %q is still active in .vrooli/service.json (enabled=%t required=%t)", name, enabled, required)
	}
	if refs, err := c.scenarioResourceReferenceCount(name); err != nil {
		return err
	} else if refs > 0 {
		return fmt.Errorf("resource %q is still referenced by %d scenario manifest(s)", name, refs)
	}
	resourcePath := filepath.Join(c.Root, "resources", name)
	if info, err := os.Stat(resourcePath); err != nil || !info.IsDir() {
		if err == nil {
			return fmt.Errorf("resource %q does not have an implementation directory under resources/", name)
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("resource %q does not have an implementation directory under resources/", name)
		}
		return err
	}
	return nil
}

func (c *Controller) projectResourceFlags(name string) (enabled bool, required bool, err error) {
	entries, err := c.readConfigEntries()
	if err != nil {
		return false, false, err
	}
	entry, ok := entries[name]
	if !ok {
		return false, false, nil
	}
	return entry.Enabled, entry.Required, nil
}

func (c *Controller) scenarioResourceReferenceCount(name string) (int, error) {
	scenariosRoot := filepath.Join(c.Root, "scenarios")
	total := 0
	err := filepath.WalkDir(scenariosRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "service.json" || filepath.Base(filepath.Dir(path)) != ".vrooli" {
			return nil
		}
		used, err := scenarioManifestUsesResource(path, name)
		if err != nil {
			return err
		}
		if used {
			total++
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}

func scenarioManifestUsesResource(path, name string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	var payload struct {
		Dependencies struct {
			Resources map[string]ConfigEntry `json:"resources"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return false, fmt.Errorf("parse scenario manifest %s: %w", path, err)
	}
	_, ok := payload.Dependencies.Resources[name]
	return ok, nil
}
