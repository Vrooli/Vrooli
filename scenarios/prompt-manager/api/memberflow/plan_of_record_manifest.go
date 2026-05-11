package memberflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const teamPlanOfRecordSchema = "team-plan-of-record/v1"

type PlanOfRecordManifest struct {
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
	Kind   string `json:"kind,omitempty"`
	Schema string `json:"schema,omitempty"`
	Base   string `json:"base,omitempty"`
	Team   string `json:"team,omitempty"`
}

type PlanOfRecordSection struct {
	ID            string                 `json:"id"`
	Path          string                 `json:"path,omitempty"`
	Required      bool                   `json:"required,omitempty"`
	Documents     []PlanOfRecordDocument `json:"documents,omitempty"`
	Packages      []PlanOfRecordPackage  `json:"packages,omitempty"`
	PackageType   string                 `json:"packageType,omitempty"`
	RequiredFiles []string               `json:"requiredFiles,omitempty"`
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
	ID             string   `json:"id"`
	Path           string   `json:"path,omitempty"`
	PackageType    string   `json:"packageType,omitempty"`
	RequiredFiles  []string `json:"requiredFiles,omitempty"`
	EntryPattern   string   `json:"entryPattern,omitempty"`
	MinimumEntries int      `json:"minimumEntries,omitempty"`
}

func LoadPlanOfRecordManifest(path string) (PlanOfRecordManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PlanOfRecordManifest{}, err
	}
	var manifest PlanOfRecordManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return PlanOfRecordManifest{}, err
	}
	manifest.SourcePath = filepath.ToSlash(path)
	manifest.RootDir = filepath.Dir(path)
	return manifest, nil
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
		manifest, err := LoadPlanOfRecordManifest(manifestPath)
		if err != nil {
			findings = append(findings, planOfRecordFinding(model, "por_manifest_invalid", relPath(runtime.RepoRoot, manifestPath), 0, fmt.Sprintf("plan-of-record manifest is invalid: %v", err), SeverityError))
			continue
		}
		findings = append(findings, ValidatePlanOfRecordManifest(runtime.RepoRoot, manifest, model)...)
	}
	return findings
}

func ValidatePlanOfRecordManifest(repoRoot string, manifest PlanOfRecordManifest, model OperatingModelDocument) []OperatingGraphFinding {
	var findings []OperatingGraphFinding
	sourcePath := relPath(repoRoot, manifest.SourcePath)
	if manifest.Contract.Schema != teamPlanOfRecordSchema {
		findings = append(findings, planOfRecordFinding(model, "por_manifest_schema_unknown", sourcePath, 0, fmt.Sprintf("plan-of-record manifest schema must be %q", teamPlanOfRecordSchema), SeverityError))
	}
	if manifest.Contract.Team != "" && model.Team != "" && manifest.Contract.Team != model.Team {
		findings = append(findings, planOfRecordFinding(model, "por_manifest_team_mismatch", sourcePath, 0, fmt.Sprintf("plan-of-record manifest team %q does not match operating model team %q", manifest.Contract.Team, model.Team), SeverityError))
	}
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
