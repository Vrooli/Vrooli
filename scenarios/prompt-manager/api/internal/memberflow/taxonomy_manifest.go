package memberflow

// Taxonomy loading reads the plan-of-record manifests under docs/. Only the
// manifest model, base-chain resolution and discovery remain here; the manifest
// shape and document validators were removed.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	teamPlanOfRecordSchema   = "team-plan-of-record/v1"
	teamPlanOfRecordKind     = "team-plan-of-record"
	maxPlanOfRecordBaseDepth = 8
)

type PlanOfRecordManifest struct {
	Schema      string                `json:"$schema,omitempty"`
	Version     string                `json:"version,omitempty"`
	Title       string                `json:"title,omitempty"`
	Description string                `json:"description,omitempty"`
	Contract    PlanOfRecordContract  `json:"contract"`
	Sections    []PlanOfRecordSection `json:"sections,omitempty"`
	// Instrument names the scenario that computes what this plan of record no
	// longer restates. A team whose durable numbers moved into a scenario needs
	// somewhere to say so: without this block the manifest can only list
	// documents, and the reader cannot tell a folder that never had an
	// instrument from one whose targets now live in a setpoint.
	Instrument    *PlanOfRecordInstrument `json:"instrument,omitempty"`
	ExtensionRule string                  `json:"extensionRule,omitempty"`
	SourcePath    string                  `json:"-"`
	RootDir       string                  `json:"-"`
}

// PlanOfRecordInstrument records the scenario that owns a team's computed
// surfaces, the verbs that read them, and the rule dividing what the instrument
// computes from what the plan of record still authors.
type PlanOfRecordInstrument struct {
	Scenario string `json:"scenario"`
	Role     string `json:"role,omitempty"`
	// ReadSurfaces are the CLI verbs a reader should use instead of a document.
	ReadSurfaces []string `json:"readSurfaces,omitempty"`
	// Rule states what belongs to the instrument rather than to this folder.
	Rule string `json:"rule,omitempty"`
}

type PlanOfRecordContract struct {
	Kind          string `json:"kind,omitempty"`
	Schema        string `json:"schema,omitempty"`
	Base          string `json:"base,omitempty"`
	Team          string `json:"team,omitempty"`
	ExtensionRule string `json:"extensionRule,omitempty"`
}

type PlanOfRecordSection struct {
	ID              string                 `json:"id"`
	Path            string                 `json:"path,omitempty"`
	Required        bool                   `json:"required,omitempty"`
	Documents       []PlanOfRecordDocument `json:"documents,omitempty"`
	Packages        []PlanOfRecordPackage  `json:"packages,omitempty"`
	PackageType     string                 `json:"packageType,omitempty"`
	RequiredFiles   []string               `json:"requiredFiles,omitempty"`
	OptionalFolders []string               `json:"optionalFolders,omitempty"`
}

type PlanOfRecordDocument struct {
	Path       string                    `json:"path"`
	DocType    string                    `json:"docType,omitempty"`
	Required   bool                      `json:"required,omitempty"`
	Validation PlanOfRecordDocumentRules `json:"validation,omitempty"`
}

type PlanOfRecordDocumentRules struct {
	RequiredHeadings []string `json:"requiredHeadings,omitempty"`
	RequiredLinks    []string `json:"requiredLinks,omitempty"`
}

type PlanOfRecordPackage struct {
	ID              string   `json:"id"`
	Path            string   `json:"path,omitempty"`
	PackageType     string   `json:"packageType,omitempty"`
	RequiredFiles   []string `json:"requiredFiles,omitempty"`
	OptionalFolders []string `json:"optionalFolders,omitempty"`
	EntryPattern    string   `json:"entryPattern,omitempty"`
	MinimumEntries  int      `json:"minimumEntries,omitempty"`
}

func (p PlanOfRecordPackage) requiredFilesIn(section PlanOfRecordSection) []string {
	if len(p.RequiredFiles) > 0 {
		return p.RequiredFiles
	}
	return section.RequiredFiles
}

func LoadPlanOfRecordManifest(path string) (PlanOfRecordManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	var manifest PlanOfRecordManifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&manifest); err != nil {
		return PlanOfRecordManifest{}, err
	}
	if err := ensureJSONEOF(dec); err != nil {
		return PlanOfRecordManifest{}, err
	}
	manifest.SourcePath = filepath.ToSlash(path)
	manifest.RootDir = filepath.Dir(path)
	return manifest, nil
}

func LoadResolvedPlanOfRecordManifest(repoRoot, path string) (PlanOfRecordManifest, error) {
	return loadResolvedPlanOfRecordManifest(repoRoot, path, map[string]bool{}, 0)
}

func loadResolvedPlanOfRecordManifest(repoRoot, path string, stack map[string]bool, depth int) (PlanOfRecordManifest, error) {
	if depth > maxPlanOfRecordBaseDepth {
		return PlanOfRecordManifest{}, fmt.Errorf("plan-of-record base chain exceeds depth %d", maxPlanOfRecordBaseDepth)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	if stack[absPath] {
		return PlanOfRecordManifest{}, fmt.Errorf("plan-of-record manifest base cycle includes %s", filepath.ToSlash(path))
	}
	stack[absPath] = true
	defer delete(stack, absPath)

	child, err := LoadPlanOfRecordManifest(path)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	if strings.TrimSpace(child.Contract.Base) == "" {
		return child, nil
	}
	basePath, err := resolvePlanOfRecordBasePath(repoRoot, child.Contract.Base)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	base, err := loadResolvedPlanOfRecordManifest(repoRoot, basePath, stack, depth+1)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	if base.Contract.Schema != "" && child.Contract.Schema != "" && base.Contract.Schema != child.Contract.Schema {
		return PlanOfRecordManifest{}, fmt.Errorf("plan-of-record manifest schema %q does not match base schema %q", child.Contract.Schema, base.Contract.Schema)
	}
	merged, err := mergePlanOfRecordManifests(base, child)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	return merged, nil
}

func resolvePlanOfRecordBasePath(repoRoot, base string) (string, error) {
	if filepath.IsAbs(base) || pathEscapesRoot(base) {
		return "", fmt.Errorf("plan-of-record base path must be repo-relative: %s", base)
	}
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("repo root is required to resolve plan-of-record base %s", base)
	}
	return filepath.Join(repoRoot, filepath.FromSlash(base)), nil
}

func mergePlanOfRecordManifests(base, child PlanOfRecordManifest) (PlanOfRecordManifest, error) {
	merged := base
	if child.Schema != "" {
		merged.Schema = child.Schema
	}
	if child.Version != "" {
		merged.Version = child.Version
	}
	if child.Title != "" {
		merged.Title = child.Title
	}
	if child.Description != "" {
		merged.Description = child.Description
	}
	merged.Contract = mergePlanOfRecordContracts(base.Contract, child.Contract)
	if child.ExtensionRule != "" {
		merged.ExtensionRule = child.ExtensionRule
	}
	sections, err := mergePlanOfRecordSections(base.Sections, child.Sections)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	merged.Sections = sections
	merged.SourcePath = child.SourcePath
	merged.RootDir = child.RootDir
	return merged, nil
}

func mergePlanOfRecordContracts(base, child PlanOfRecordContract) PlanOfRecordContract {
	merged := base
	if child.Kind != "" {
		merged.Kind = child.Kind
	}
	if child.Schema != "" {
		merged.Schema = child.Schema
	}
	if child.Base != "" {
		merged.Base = child.Base
	}
	if child.Team != "" {
		merged.Team = child.Team
	}
	if child.ExtensionRule != "" {
		merged.ExtensionRule = child.ExtensionRule
	}
	return merged
}

func mergePlanOfRecordSections(base, child []PlanOfRecordSection) ([]PlanOfRecordSection, error) {
	out := append([]PlanOfRecordSection(nil), base...)
	index := map[string]int{}
	for i, section := range out {
		if section.ID == "" {
			continue
		}
		index[section.ID] = i
	}
	for _, childSection := range child {
		if i, ok := index[childSection.ID]; ok {
			section, err := mergePlanOfRecordSection(out[i], childSection)
			if err != nil {
				return nil, err
			}
			out[i] = section
			continue
		}
		index[childSection.ID] = len(out)
		out = append(out, childSection)
	}
	return out, nil
}

func mergePlanOfRecordSection(base, child PlanOfRecordSection) (PlanOfRecordSection, error) {
	merged := base
	if child.ID != "" {
		merged.ID = child.ID
	}
	if child.Path != "" {
		merged.Path = child.Path
	}
	if child.Required && !base.Required {
		merged.Required = true
	}
	if !child.Required && base.Required {
		merged.Required = true
	}
	if child.PackageType != "" {
		merged.PackageType = child.PackageType
	}
	if child.RequiredFiles != nil {
		merged.RequiredFiles = child.RequiredFiles
	}
	if child.OptionalFolders != nil {
		merged.OptionalFolders = child.OptionalFolders
	}
	merged.Documents = mergePlanOfRecordDocuments(base.Documents, child.Documents)
	merged.Packages = mergePlanOfRecordPackages(base.Packages, child.Packages)
	return merged, nil
}

func mergePlanOfRecordDocuments(base, child []PlanOfRecordDocument) []PlanOfRecordDocument {
	out := append([]PlanOfRecordDocument(nil), base...)
	index := map[string]int{}
	for i, doc := range out {
		index[doc.Path] = i
	}
	for _, childDoc := range child {
		if i, ok := index[childDoc.Path]; ok {
			out[i] = mergePlanOfRecordDocument(out[i], childDoc)
			continue
		}
		index[childDoc.Path] = len(out)
		out = append(out, childDoc)
	}
	return out
}

func mergePlanOfRecordDocument(base, child PlanOfRecordDocument) PlanOfRecordDocument {
	merged := base
	if child.Path != "" {
		merged.Path = child.Path
	}
	if child.DocType != "" {
		merged.DocType = child.DocType
	}
	if child.Required {
		merged.Required = true
	}
	if child.Validation.RequiredHeadings != nil || child.Validation.RequiredLinks != nil {
		merged.Validation = child.Validation
	}
	return merged
}

func mergePlanOfRecordPackages(base, child []PlanOfRecordPackage) []PlanOfRecordPackage {
	out := append([]PlanOfRecordPackage(nil), base...)
	index := map[string]int{}
	for i, pkg := range out {
		index[pkg.ID] = i
	}
	for _, childPkg := range child {
		if i, ok := index[childPkg.ID]; ok {
			out[i] = mergePlanOfRecordPackage(out[i], childPkg)
			continue
		}
		index[childPkg.ID] = len(out)
		out = append(out, childPkg)
	}
	return out
}

func mergePlanOfRecordPackage(base, child PlanOfRecordPackage) PlanOfRecordPackage {
	merged := base
	if child.ID != "" {
		merged.ID = child.ID
	}
	if child.Path != "" {
		merged.Path = child.Path
	}
	if child.PackageType != "" {
		merged.PackageType = child.PackageType
	}
	if child.RequiredFiles != nil {
		merged.RequiredFiles = child.RequiredFiles
	}
	if child.OptionalFolders != nil {
		merged.OptionalFolders = child.OptionalFolders
	}
	if child.EntryPattern != "" {
		merged.EntryPattern = child.EntryPattern
	}
	if child.MinimumEntries != 0 {
		merged.MinimumEntries = child.MinimumEntries
	}
	return merged
}

func ensureJSONEOF(dec *json.Decoder) error {
	var extra any
	if err := dec.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("manifest contains multiple JSON values")
}

func DiscoverPlanOfRecordManifestPaths(repoRoot string) ([]string, error) {
	docsDir := filepath.Join(repoRoot, "docs")
	entries, err := os.ReadDir(docsDir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(docsDir, entry.Name(), "manifest.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var header struct {
			Contract PlanOfRecordContract `json:"contract"`
		}
		if err := json.Unmarshal(data, &header); err != nil {
			paths = append(paths, path)
			continue
		}
		if header.Contract.Kind == teamPlanOfRecordKind || header.Contract.Schema == teamPlanOfRecordSchema {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

func pathEscapesRoot(path string) bool {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" {
		return false
	}
	if strings.HasPrefix(path, "/") || filepath.IsAbs(path) {
		return true
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if part == ".." {
			return true
		}
	}
	return false
}

