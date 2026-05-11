package memberflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const teamPlanOfRecordSchema = "team-plan-of-record/v1"
const teamPlanOfRecordKind = "team-plan-of-record"
const maxPlanOfRecordBaseDepth = 8

var planOfRecordBannedPhrases = []string{
	"PLAN_OF_RECORD_STRUCTURE",
	"NOTEBOOK_DEBT_TAXONOMY",
	"docs/marketing/notebook",
	"store/teams/meta-optimization/notebook",
	"topic[old]:friction/",
	"friction/<",
	"Migration Notes",
}

type PlanOfRecordManifest struct {
	Schema        string                `json:"$schema,omitempty"`
	Version       string                `json:"version,omitempty"`
	Title         string                `json:"title,omitempty"`
	Description   string                `json:"description,omitempty"`
	Contract      PlanOfRecordContract  `json:"contract"`
	Sections      []PlanOfRecordSection `json:"sections,omitempty"`
	ExtensionRule string                `json:"extensionRule,omitempty"`
	SourcePath    string                `json:"-"`
	RootDir       string                `json:"-"`
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

func ValidatePlanOfRecordManifestsForModels(models []OperatingModelDocument, runtime OperatingGraphRuntime) []OperatingGraphFinding {
	if strings.TrimSpace(runtime.RepoRoot) == "" {
		return nil
	}
	seen := map[string]bool{}
	var findings []OperatingGraphFinding
	for _, model := range models {
		rootRel := planOfRecordRootFromModel(model)
		if rootRel == "" || seen[rootRel] {
			continue
		}
		seen[rootRel] = true
		manifestPath := filepath.Join(runtime.RepoRoot, filepath.FromSlash(rootRel), "manifest.json")
		if _, err := os.Stat(manifestPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			findings = append(findings, planOfRecordFinding(model, "por_manifest_unreadable", filepath.ToSlash(manifestPath), 0, fmt.Sprintf("plan-of-record manifest cannot be read: %v", err), SeverityError))
			continue
		}
		manifest, err := LoadResolvedPlanOfRecordManifest(runtime.RepoRoot, manifestPath)
		if err != nil {
			findings = append(findings, planOfRecordFinding(model, "por_manifest_invalid", relPath(runtime.RepoRoot, manifestPath), 0, fmt.Sprintf("plan-of-record manifest is invalid: %v", err), SeverityError))
			continue
		}
		findings = append(findings, ValidatePlanOfRecordManifest(runtime.RepoRoot, manifest, model)...)
	}
	return findings
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

func ValidateAllPlanOfRecords(repoRoot string) []OperatingGraphFinding {
	paths, err := DiscoverPlanOfRecordManifestPaths(repoRoot)
	if err != nil {
		return []OperatingGraphFinding{{
			Rule:       "por_discovery_failed",
			Severity:   string(SeverityError),
			SourcePath: filepath.ToSlash(filepath.Join(repoRoot, "docs")),
			Path:       filepath.ToSlash(filepath.Join(repoRoot, "docs")),
			Detail:     fmt.Sprintf("plan-of-record discovery failed: %v", err),
		}}
	}
	var findings []OperatingGraphFinding
	for _, path := range paths {
		manifest, err := LoadResolvedPlanOfRecordManifest(repoRoot, path)
		model := planOfRecordModelFromManifest(repoRoot, path, manifest)
		if err != nil {
			findings = append(findings, planOfRecordFinding(model, "por_manifest_invalid", relPath(repoRoot, path), 0, fmt.Sprintf("plan-of-record manifest is invalid: %v", err), SeverityError))
			continue
		}
		model = planOfRecordModelFromManifest(repoRoot, path, manifest)
		findings = append(findings, ValidatePlanOfRecordManifest(repoRoot, manifest, model)...)
	}
	sortOperatingFindings(findings)
	return findings
}

func ValidatePlanOfRecordManifest(repoRoot string, manifest PlanOfRecordManifest, model OperatingModelDocument) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	sourcePath := relPath(repoRoot, manifest.SourcePath)
	if manifest.Contract.Kind != teamPlanOfRecordKind {
		findings = append(findings, planOfRecordFinding(model, "por_manifest_kind_unknown", sourcePath, 0, fmt.Sprintf("plan-of-record manifest kind must be %q", teamPlanOfRecordKind), SeverityError))
	}
	if manifest.Contract.Schema != teamPlanOfRecordSchema {
		findings = append(findings, planOfRecordFinding(model, "por_manifest_schema_unknown", sourcePath, 0, fmt.Sprintf("plan-of-record manifest schema must be %q", teamPlanOfRecordSchema), SeverityError))
	}
	if manifest.Contract.Team != "" && model.Team != "" && manifest.Contract.Team != model.Team {
		findings = append(findings, planOfRecordFinding(model, "por_manifest_team_mismatch", sourcePath, 0, fmt.Sprintf("plan-of-record manifest team %q does not match operating model team %q", manifest.Contract.Team, model.Team), SeverityError))
	}
	findings = append(findings, validatePlanOfRecordManifestShape(manifest, model)...)
	registered := map[string]bool{
		"manifest.json": true,
	}
	for _, section := range manifest.Sections {
		sectionDir := filepath.Join(manifest.RootDir, filepath.FromSlash(section.Path))
		if section.Required {
			if _, err := os.Stat(sectionDir); err != nil {
				if os.IsNotExist(err) {
					findings = append(findings, planOfRecordFinding(model, "por_required_section_missing", sourcePath, 0, fmt.Sprintf("required plan-of-record section %q is missing at %s", section.ID, cleanSectionPath(section.Path)), SeverityError))
				}
				continue
			}
		}
		for _, doc := range section.Documents {
			docRel := cleanJoin(section.Path, doc.Path)
			registered[docRel] = true
			docPath := filepath.Join(manifest.RootDir, filepath.FromSlash(docRel))
			if _, err := os.Stat(docPath); err != nil {
				if doc.Required || section.Required {
					findings = append(findings, planOfRecordFinding(model, "por_required_document_missing", docRel, 0, fmt.Sprintf("required plan-of-record document %s is missing", docRel), SeverityError))
				}
				continue
			}
			findings = append(findings, validatePlanOfRecordDocumentRules(model, docPath, docRel, doc.Validation)...)
		}
		for _, pkg := range section.Packages {
			pkgRel := cleanJoin(section.Path, pkg.Path)
			requiredFiles := pkg.RequiredFiles
			if len(requiredFiles) == 0 {
				requiredFiles = section.RequiredFiles
			}
			for _, required := range requiredFiles {
				fileRel := cleanJoin(pkgRel, required)
				registered[fileRel] = true
				if _, err := os.Stat(filepath.Join(manifest.RootDir, filepath.FromSlash(fileRel))); err != nil && os.IsNotExist(err) {
					findings = append(findings, planOfRecordFinding(model, "por_package_required_file_missing", fileRel, 0, fmt.Sprintf("plan-of-record package %q is missing required file %s", pkg.ID, required), SeverityError))
				}
			}
			if pkg.EntryPattern != "" {
				count, err := countPlanOfRecordPackageEntries(filepath.Join(manifest.RootDir, filepath.FromSlash(pkgRel)), pkg.EntryPattern)
				if err == nil && count < pkg.MinimumEntries {
					findings = append(findings, planOfRecordFinding(model, "por_package_entries_missing", pkgRel, 0, fmt.Sprintf("plan-of-record package %q has %d entries, want at least %d", pkg.ID, count, pkg.MinimumEntries), SeverityError))
				}
			}
		}
	}
	findings = append(findings, validatePlanOfRecordRegisteredFiles(repoRoot, manifest, model, registered)...)
	findings = append(findings, validatePlanOfRecordHardCutover(repoRoot, manifest, model)...)
	return findings
}

func validatePlanOfRecordManifestShape(manifest PlanOfRecordManifest, model OperatingModelDocument) []OperatingGraphFinding {
	sourcePath := manifest.SourcePath
	seenSections := map[string]bool{}
	var findings []OperatingGraphFinding
	for _, section := range manifest.Sections {
		if seenSections[section.ID] {
			findings = append(findings, planOfRecordFinding(model, "por_manifest_duplicate_section", sourcePath, 0, fmt.Sprintf("plan-of-record manifest declares duplicate section %q", section.ID), SeverityError))
		}
		seenSections[section.ID] = true
		if pathEscapesRoot(section.Path) {
			findings = append(findings, planOfRecordFinding(model, "por_manifest_path_invalid", sourcePath, 0, fmt.Sprintf("plan-of-record section %q has invalid path %q", section.ID, section.Path), SeverityError))
		}
		seenDocs := map[string]bool{}
		for _, doc := range section.Documents {
			if seenDocs[doc.Path] {
				findings = append(findings, planOfRecordFinding(model, "por_manifest_duplicate_document", sourcePath, 0, fmt.Sprintf("plan-of-record section %q declares duplicate document %q", section.ID, doc.Path), SeverityError))
			}
			seenDocs[doc.Path] = true
			if pathEscapesRoot(doc.Path) {
				findings = append(findings, planOfRecordFinding(model, "por_manifest_path_invalid", sourcePath, 0, fmt.Sprintf("plan-of-record section %q has invalid document path %q", section.ID, doc.Path), SeverityError))
			}
		}
		seenPackages := map[string]bool{}
		for _, pkg := range section.Packages {
			if seenPackages[pkg.ID] {
				findings = append(findings, planOfRecordFinding(model, "por_manifest_duplicate_package", sourcePath, 0, fmt.Sprintf("plan-of-record section %q declares duplicate package %q", section.ID, pkg.ID), SeverityError))
			}
			seenPackages[pkg.ID] = true
			if pathEscapesRoot(pkg.Path) {
				findings = append(findings, planOfRecordFinding(model, "por_manifest_path_invalid", sourcePath, 0, fmt.Sprintf("plan-of-record package %q has invalid path %q", pkg.ID, pkg.Path), SeverityError))
			}
			for _, required := range pkg.RequiredFiles {
				if pathEscapesRoot(required) {
					findings = append(findings, planOfRecordFinding(model, "por_manifest_path_invalid", sourcePath, 0, fmt.Sprintf("plan-of-record package %q has invalid required file %q", pkg.ID, required), SeverityError))
				}
			}
		}
	}
	return findings
}

func validatePlanOfRecordDocumentRules(model OperatingModelDocument, path, rel string, rules PlanOfRecordDocumentRules) []OperatingGraphFinding {
	if len(rules.RequiredHeadings) == 0 && len(rules.RequiredLinks) == 0 {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return []OperatingGraphFinding{planOfRecordFinding(model, "por_document_unreadable", rel, 0, fmt.Sprintf("plan-of-record document %s cannot be read: %v", rel, err), SeverityError)}
	}
	text := string(data)
	headings := markdownHeadingSet(text)
	var findings []OperatingGraphFinding
	for _, heading := range rules.RequiredHeadings {
		if !headings[strings.ToLower(strings.TrimSpace(heading))] {
			findings = append(findings, planOfRecordFinding(model, "por_required_heading_missing", rel, 0, fmt.Sprintf("plan-of-record document %s is missing heading %q", rel, heading), SeverityError))
		}
	}
	for _, link := range rules.RequiredLinks {
		if !strings.Contains(text, link) {
			findings = append(findings, planOfRecordFinding(model, "por_required_link_missing", rel, 0, fmt.Sprintf("plan-of-record document %s is missing required link %q", rel, link), SeverityError))
		}
	}
	return findings
}

func validatePlanOfRecordRegisteredFiles(repoRoot string, manifest PlanOfRecordManifest, model OperatingModelDocument, registered map[string]bool) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	allowedDirs := map[string]bool{}
	for _, section := range manifest.Sections {
		if section.Path != "" && section.Path != "." {
			allowedDirs[strings.Trim(cleanSectionPath(section.Path), "/")] = true
		}
		for _, pkg := range section.Packages {
			pkgRel := cleanJoin(section.Path, pkg.Path)
			allowedDirs[strings.Trim(pkgRel, "/")] = true
		}
	}
	_ = filepath.WalkDir(manifest.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(manifest.RootDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if registered[rel] || planOfRecordFileInAllowedPackage(rel, allowedDirs) {
			return nil
		}
		findings = append(findings, planOfRecordFinding(model, "por_unregistered_document", relPath(repoRoot, path), 0, fmt.Sprintf("durable plan-of-record document %s is not registered in manifest.json", rel), SeverityWarning))
		return nil
	})
	sortOperatingFindings(findings)
	return findings
}

func validatePlanOfRecordHardCutover(repoRoot string, manifest PlanOfRecordManifest, model OperatingModelDocument) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	_ = filepath.WalkDir(manifest.RootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if strings.EqualFold(d.Name(), "notebook") || strings.EqualFold(d.Name(), "notebooks") {
				findings = append(findings, planOfRecordFinding(model, "por_notebook_surface", relPath(repoRoot, path), 0, "plan-of-record folders must not contain notebook surfaces; agents write working observations to typed knowledge topics", SeverityError))
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".md" && ext != ".json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(data)
		for _, phrase := range planOfRecordBannedPhrases {
			if idx := strings.Index(text, phrase); idx >= 0 {
				findings = append(findings, planOfRecordFinding(model, "por_hard_cutover_drift", relPath(repoRoot, path), lineNumberAt(text, idx), fmt.Sprintf("plan-of-record file contains hard-cutover drift phrase %q", phrase), SeverityError))
			}
		}
		return nil
	})
	sortOperatingFindings(findings)
	return findings
}

func countPlanOfRecordPackageEntries(dir, pattern string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return 0, err
	}
	var count int
	for _, match := range matches {
		if strings.EqualFold(filepath.Base(match), "README.md") {
			continue
		}
		info, err := os.Stat(match)
		if err == nil && !info.IsDir() {
			count++
		}
	}
	return count, nil
}

func lineNumberAt(text string, idx int) int {
	if idx < 0 {
		return 0
	}
	line := 1
	for i, r := range text {
		if i >= idx {
			break
		}
		if r == '\n' {
			line++
		}
	}
	return line
}

func markdownHeadingSet(text string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading != "" {
			out[strings.ToLower(heading)] = true
		}
	}
	return out
}

func planOfRecordRootFromModel(model OperatingModelDocument) string {
	parts := strings.Split(filepath.ToSlash(model.Source.Path), "/")
	if len(parts) < 2 || parts[0] != "docs" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(parts[0], parts[1]))
}

func planOfRecordModelFromManifest(repoRoot, path string, manifest PlanOfRecordManifest) OperatingModelDocument {
	team := manifest.Contract.Team
	if team == "" {
		parts := strings.Split(relPath(repoRoot, path), "/")
		if len(parts) >= 2 && parts[0] == "docs" {
			team = parts[1]
		}
	}
	return OperatingModelDocument{
		ID:   team + "-plan-of-record",
		Team: team,
		Source: OperatingModelSource{
			Path: relPath(repoRoot, path),
		},
	}
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

func planOfRecordFileInAllowedPackage(rel string, dirs map[string]bool) bool {
	dir := strings.Trim(filepath.ToSlash(filepath.Dir(rel)), "/")
	for dir != "." && dir != "" {
		if dirs[dir] {
			return true
		}
		next := filepath.ToSlash(filepath.Dir(dir))
		if next == dir {
			break
		}
		dir = strings.Trim(next, "/")
	}
	return false
}

func cleanJoin(base, child string) string {
	if base == "" || base == "." {
		return filepath.ToSlash(filepath.Clean(child))
	}
	return filepath.ToSlash(filepath.Clean(filepath.Join(base, child)))
}

func cleanSectionPath(path string) string {
	if path == "" || path == "." {
		return "."
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func relPath(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func planOfRecordFinding(model OperatingModelDocument, rule, path string, line int, detail string, severity Severity) OperatingGraphFinding {
	return OperatingGraphFinding{
		Rule:       rule,
		Severity:   string(severity),
		GraphID:    model.ID,
		Team:       model.Team,
		Path:       filepath.ToSlash(path),
		SourcePath: filepath.ToSlash(path),
		Line:       line,
		Detail:     detail,
	}
}
