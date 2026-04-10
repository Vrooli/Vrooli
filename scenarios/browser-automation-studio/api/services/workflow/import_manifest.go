package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

const manifestFileName = "import-manifest.json"

// ImportManifest tracks which external workflow files have been imported.
// Stored at {project}/.bas/import-manifest.json
type ImportManifest struct {
	// Entries maps source relative path to import information.
	Entries map[string]ImportedEntry `json:"entries"`
}

// ImportedEntry records information about an imported workflow.
type ImportedEntry struct {
	// WorkflowID is the UUID assigned to the converted workflow.
	WorkflowID uuid.UUID `json:"workflow_id"`
	// SourceHash is the SHA-256 hash of the source file content when imported.
	SourceHash string `json:"source_hash"`
	// NativeRelPath is the relative path to the converted workflow file in workflows/.
	NativeRelPath string `json:"native_rel_path"`
}

// manifestCache holds loaded manifests to avoid repeated disk reads.
var manifestCache sync.Map // map[uuid.UUID]*ImportManifest

// LoadImportManifest loads the import manifest for a project.
// Returns an empty manifest if none exists.
func LoadImportManifest(project *database.ProjectIndex) (*ImportManifest, error) {
	if project == nil {
		return newManifest(), nil
	}

	// Check cache first
	if cached, ok := manifestCache.Load(project.ID); ok {
		return cached.(*ImportManifest), nil
	}

	manifestPath := getManifestPath(project)
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return newManifest(), nil
		}
		return nil, err
	}

	var manifest ImportManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		// If manifest is corrupted, start fresh
		return newManifest(), nil
	}

	if manifest.Entries == nil {
		manifest.Entries = make(map[string]ImportedEntry)
	}

	manifestCache.Store(project.ID, &manifest)
	return &manifest, nil
}

// SaveImportManifest saves the import manifest for a project.
func SaveImportManifest(project *database.ProjectIndex, manifest *ImportManifest) error {
	if project == nil || manifest == nil {
		return nil
	}

	manifestPath := getManifestPath(project)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return err
	}

	manifestCache.Store(project.ID, manifest)
	return nil
}

// IsImported checks if a source file has already been imported.
func (m *ImportManifest) IsImported(sourceRelPath string) bool {
	if m == nil || m.Entries == nil {
		return false
	}
	_, exists := m.Entries[normalizeSourcePath(sourceRelPath)]
	return exists
}

// HasChanged checks if the source file content has changed since import.
func (m *ImportManifest) HasChanged(sourceRelPath string, currentContent []byte) bool {
	if m == nil || m.Entries == nil {
		return true
	}

	entry, exists := m.Entries[normalizeSourcePath(sourceRelPath)]
	if !exists {
		return true
	}

	currentHash := hashContent(currentContent)
	return entry.SourceHash != currentHash
}

// GetEntry retrieves the import entry for a source path.
func (m *ImportManifest) GetEntry(sourceRelPath string) (ImportedEntry, bool) {
	if m == nil || m.Entries == nil {
		return ImportedEntry{}, false
	}
	entry, exists := m.Entries[normalizeSourcePath(sourceRelPath)]
	return entry, exists
}

// RecordImport records that a source file has been imported.
func (m *ImportManifest) RecordImport(sourceRelPath string, workflowID uuid.UUID, nativeRelPath string, sourceContent []byte) {
	if m == nil {
		return
	}
	if m.Entries == nil {
		m.Entries = make(map[string]ImportedEntry)
	}

	m.Entries[normalizeSourcePath(sourceRelPath)] = ImportedEntry{
		WorkflowID:    workflowID,
		SourceHash:    hashContent(sourceContent),
		NativeRelPath: nativeRelPath,
	}
}

// RemoveEntry removes an import entry.
func (m *ImportManifest) RemoveEntry(sourceRelPath string) {
	if m == nil || m.Entries == nil {
		return
	}
	delete(m.Entries, normalizeSourcePath(sourceRelPath))
}

// GetWorkflowIDBySource returns the workflow ID for a given source path.
func (m *ImportManifest) GetWorkflowIDBySource(sourceRelPath string) (uuid.UUID, bool) {
	entry, exists := m.GetEntry(sourceRelPath)
	if !exists {
		return uuid.Nil, false
	}
	return entry.WorkflowID, true
}

// FindSourceByWorkflowID finds the source path for a given workflow ID.
func (m *ImportManifest) FindSourceByWorkflowID(workflowID uuid.UUID) (string, bool) {
	if m == nil || m.Entries == nil {
		return "", false
	}
	for sourcePath, entry := range m.Entries {
		if entry.WorkflowID == workflowID {
			return sourcePath, true
		}
	}
	return "", false
}

// ClearCache clears the manifest cache for a project.
func ClearManifestCache(projectID uuid.UUID) {
	manifestCache.Delete(projectID)
}

func newManifest() *ImportManifest {
	return &ImportManifest{
		Entries: make(map[string]ImportedEntry),
	}
}

func getManifestPath(project *database.ProjectIndex) string {
	return filepath.Join(project.FolderPath, ".bas", manifestFileName)
}

func normalizeSourcePath(path string) string {
	normalized := strings.ReplaceAll(path, "\\", "/")
	normalized = strings.TrimPrefix(normalized, "./")
	return normalized
}

func hashContent(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}
