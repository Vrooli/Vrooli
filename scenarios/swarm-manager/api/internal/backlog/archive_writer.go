package backlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"swarm-manager/internal/storage"
)

// CreateModuleInput holds the fields for creating a new requirements module.
type CreateModuleInput struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// resolveRequirementsDir finds or creates the requirements directory for an item.
// It checks archive/requirements/ first, then requirements/ at item root.
// If neither exists, it creates requirements/ at the item root.
func resolveRequirementsDir(itemDir string) (string, error) {
	archiveReq := filepath.Join(itemDir, "archive", "requirements")
	if info, err := os.Stat(archiveReq); err == nil && info.IsDir() {
		return archiveReq, nil
	}

	rootReq := filepath.Join(itemDir, "requirements")
	if info, err := os.Stat(rootReq); err == nil && info.IsDir() {
		return rootReq, nil
	}

	// Create at item root.
	if err := os.MkdirAll(rootReq, 0o755); err != nil {
		return "", fmt.Errorf("create requirements dir: %w", err)
	}
	return rootReq, nil
}

// indexFile represents the top-level requirements index.json.
type indexFile struct {
	Metadata     map[string]any `json:"_metadata,omitempty"`
	Imports      []string       `json:"imports"`
	Requirements []any          `json:"requirements"`
}

// readIndex reads an index.json from the requirements directory.
// Returns a zero-value indexFile if the file does not exist.
func readIndex(reqDir string) (indexFile, error) {
	path := filepath.Join(reqDir, "index.json")
	var idx indexFile
	found, err := storage.ReadJSON(path, &idx)
	if err != nil {
		return indexFile{}, err
	}
	if !found {
		return indexFile{
			Imports:      []string{},
			Requirements: []any{},
		}, nil
	}
	if idx.Imports == nil {
		idx.Imports = []string{}
	}
	if idx.Requirements == nil {
		idx.Requirements = []any{}
	}
	return idx, nil
}

// writeIndex writes the index.json back to disk atomically.
func writeIndex(reqDir string, idx indexFile) error {
	return storage.WriteJSONAtomic(filepath.Join(reqDir, "index.json"), idx)
}

// resolveModulePath scans index.json imports to find the module file path
// whose directory base name (without numeric prefix) matches moduleId.
func resolveModulePath(reqDir, moduleID string) (string, error) {
	idx, err := readIndex(reqDir)
	if err != nil {
		return "", fmt.Errorf("read index: %w", err)
	}

	for _, imp := range idx.Imports {
		dir := filepath.Dir(imp)
		base := filepath.Base(dir)
		cleaned := stripNumericPrefix(base)
		if cleaned == moduleID {
			return filepath.Join(reqDir, imp), nil
		}
	}
	return "", fmt.Errorf("module %q not found in index imports", moduleID)
}

// stripNumericPrefix removes a leading "NN-" prefix from a directory name.
func stripNumericPrefix(name string) string {
	if idx := strings.Index(name, "-"); idx > 0 && isNumeric(name[:idx]) {
		return name[idx+1:]
	}
	return name
}

// WriteModuleRequirements reads an existing module file, replaces its
// requirements array while preserving all other fields, and writes it back.
func WriteModuleRequirements(itemDir, moduleID string, reqs []json.RawMessage) error {
	reqDir, err := resolveRequirementsDir(itemDir)
	if err != nil {
		return err
	}

	modPath, err := resolveModulePath(reqDir, moduleID)
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(modPath)
	if err != nil {
		return fmt.Errorf("read module file: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse module file: %w", err)
	}

	// Convert []json.RawMessage to []any so JSON marshal produces the right output.
	anyReqs := make([]any, len(reqs))
	for i, r := range reqs {
		var v any
		if err := json.Unmarshal(r, &v); err != nil {
			return fmt.Errorf("parse requirement %d: %w", i, err)
		}
		anyReqs[i] = v
	}
	doc["requirements"] = anyReqs

	return storage.WriteJSONAtomic(modPath, doc)
}

// CreateModule creates a new module directory and module.json, then adds
// the import to index.json. If the requirements directory and index.json
// don't exist yet, they are created.
func CreateModule(itemDir string, mod CreateModuleInput, position int) error {
	reqDir, err := resolveRequirementsDir(itemDir)
	if err != nil {
		return err
	}

	idx, err := readIndex(reqDir)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	// Determine numeric prefix based on position.
	prefix := position
	if prefix <= 0 {
		prefix = len(idx.Imports) + 1
	}
	dirName := fmt.Sprintf("%02d-%s", prefix, mod.ID)
	modDir := filepath.Join(reqDir, dirName)

	if err := os.MkdirAll(modDir, 0o755); err != nil {
		return fmt.Errorf("create module dir: %w", err)
	}

	// Build the module.json content.
	moduleDoc := map[string]any{
		"module":       mod.ID,
		"description":  mod.Description,
		"prd_refs":     []any{},
		"requirements": []any{},
	}

	modFilePath := filepath.Join(modDir, "module.json")
	if err := storage.WriteJSONAtomic(modFilePath, moduleDoc); err != nil {
		return fmt.Errorf("write module.json: %w", err)
	}

	// Add import to index.
	importPath := filepath.Join(dirName, "module.json")
	idx.Imports = append(idx.Imports, importPath)

	return writeIndex(reqDir, idx)
}

// UpdateModuleMeta updates a module file's module (title) and description fields.
func UpdateModuleMeta(itemDir, moduleID, title, description string) error {
	reqDir, err := resolveRequirementsDir(itemDir)
	if err != nil {
		return err
	}

	modPath, err := resolveModulePath(reqDir, moduleID)
	if err != nil {
		return err
	}

	raw, err := os.ReadFile(modPath)
	if err != nil {
		return fmt.Errorf("read module file: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse module file: %w", err)
	}

	if title != "" {
		doc["module"] = title
	}
	if description != "" {
		doc["description"] = description
	}

	return storage.WriteJSONAtomic(modPath, doc)
}

// DeleteModule removes a module directory and its import from index.json.
func DeleteModule(itemDir, moduleID string) error {
	reqDir, err := resolveRequirementsDir(itemDir)
	if err != nil {
		return err
	}

	idx, err := readIndex(reqDir)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	// Find and remove the matching import.
	found := false
	newImports := make([]string, 0, len(idx.Imports))
	var matchedDir string
	for _, imp := range idx.Imports {
		dir := filepath.Dir(imp)
		base := filepath.Base(dir)
		cleaned := stripNumericPrefix(base)
		if cleaned == moduleID {
			found = true
			matchedDir = dir
			continue
		}
		newImports = append(newImports, imp)
	}

	if !found {
		return fmt.Errorf("module %q not found in index imports", moduleID)
	}

	// Remove the directory.
	modDir := filepath.Join(reqDir, matchedDir)
	if err := os.RemoveAll(modDir); err != nil {
		return fmt.Errorf("remove module dir: %w", err)
	}

	idx.Imports = newImports
	return writeIndex(reqDir, idx)
}

// RequirementReviewUpdate holds review field values for a single requirement.
type RequirementReviewUpdate struct {
	ReviewStatus  string
	ReviewComment string
	ReviewedAt    string
}

// PatchModuleReviewState reads a module's requirements, patches review fields,
// and writes back using the json.RawMessage pass-through pattern.
func PatchModuleReviewState(itemDir, moduleID string, updates map[string]RequirementReviewUpdate) error {
	reqDir, err := resolveRequirementsDir(itemDir)
	if err != nil {
		return err
	}

	// The root index.json is parsed as group "index" but doesn't appear in
	// its own imports list. Resolve it directly instead of via resolveModulePath.
	var modPath string
	if moduleID == "index" {
		modPath = filepath.Join(reqDir, "index.json")
	} else {
		modPath, err = resolveModulePath(reqDir, moduleID)
		if err != nil {
			return err
		}
	}

	raw, err := os.ReadFile(modPath)
	if err != nil {
		return fmt.Errorf("read module file: %w", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse module file: %w", err)
	}

	reqsRaw, ok := doc["requirements"]
	if !ok {
		return nil
	}

	reqs, ok := reqsRaw.([]any)
	if !ok {
		return nil
	}

	for i, reqAny := range reqs {
		reqMap, ok := reqAny.(map[string]any)
		if !ok {
			continue
		}
		id, _ := reqMap["id"].(string)
		upd, found := updates[id]
		if !found {
			continue
		}

		if upd.ReviewStatus == "unreviewed" {
			delete(reqMap, "reviewed_at")
			delete(reqMap, "review_comment")
			delete(reqMap, "review_status")
		} else {
			reqMap["review_status"] = upd.ReviewStatus
			reqMap["review_comment"] = upd.ReviewComment
			reqMap["reviewed_at"] = upd.ReviewedAt
		}
		reqs[i] = reqMap
	}

	doc["requirements"] = reqs
	return storage.WriteJSONAtomic(modPath, doc)
}
