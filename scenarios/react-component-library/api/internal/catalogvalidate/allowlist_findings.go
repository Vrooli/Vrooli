package catalogvalidate

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type vacuousAllowlistEntry struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

type vacuousAllowlistFile struct {
	SchemaVersion int                     `json:"schemaVersion"`
	Entries       []vacuousAllowlistEntry `json:"entries"`
}

const (
	vacuousAllowlistRelativePath       = "scenarios/react-component-library/library/vacuous-allowlist.json"
	tautologicalAllowlistRelativePath  = "scenarios/react-component-library/library/tautological-story-allowlist.json"
	storyCoverageAllowlistRelativePath = "scenarios/react-component-library/library/story-coverage-allowlist.json"
)

func storyCoverageAllowlistFindings(repoRoot string) []Finding {
	path := filepath.Join(repoRoot, storyCoverageAllowlistRelativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: "catalog.story_coverage_allowlist_unreadable", Severity: "error", Location: storyCoverageAllowlistRelativePath, Message: err.Error()}}
	}
	current, findings := parseVacuousAllowlist(storyCoverageAllowlistRelativePath, data)
	if len(findings) > 0 {
		return findings
	}
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	for _, entry := range current.Entries {
		if _, statErr := os.Stat(filepath.Join(scenarioRoot, filepath.FromSlash(entry.Path))); statErr != nil {
			findings = append(findings, Finding{Code: "catalog.story_coverage_allowlist_unknown_entry", Severity: "error", Location: entry.Path, Message: "allowlisted story contract does not exist"})
		}
	}
	baselineData, baselineErr := gitShow(repoRoot, "HEAD:"+storyCoverageAllowlistRelativePath)
	if baselineErr != nil {
		return findings
	}
	baseline, baselineFindings := parseVacuousAllowlist(storyCoverageAllowlistRelativePath, baselineData)
	findings = append(findings, baselineFindings...)
	if len(baselineFindings) > 0 {
		return findings
	}
	for _, entryPath := range allowlistGrowth(current.Entries, baseline.Entries) {
		findings = append(findings, Finding{Code: "catalog.story_coverage_allowlist_growth", Severity: "error", Location: entryPath, Message: "story coverage allowlist may shrink but may not gain entries"})
	}
	for _, entry := range current.Entries {
		if allowlistPaths(baseline.Entries)[entry.Path] && allowlistSourceChanged(repoRoot, filepath.ToSlash(filepath.Join("scenarios/react-component-library", entry.Path))) {
			findings = append(findings, Finding{Code: "catalog.story_coverage_allowlist_source_changed", Severity: "error", Location: entry.Path, Message: "source changed while its story coverage debt remains allowlisted; remove the entry after adding the required stories"})
		}
	}
	return findings
}

func tautologicalAllowlistFindings(repoRoot string) []Finding {
	path := filepath.Join(repoRoot, tautologicalAllowlistRelativePath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: "catalog.tautological_allowlist_unreadable", Severity: "error", Location: tautologicalAllowlistRelativePath, Message: err.Error()}}
	}
	current, findings := parseVacuousAllowlist(tautologicalAllowlistRelativePath, data)
	if len(findings) > 0 {
		return findings
	}
	for _, entry := range current.Entries {
		contractPath := filepath.Join(repoRoot, "scenarios", "react-component-library", filepath.FromSlash(entry.Path))
		if _, statErr := os.Stat(contractPath); statErr != nil {
			findings = append(findings, Finding{Code: "catalog.tautological_allowlist_unknown_entry", Severity: "error", Location: entry.Path, Message: "allowlisted contract does not exist"})
		}
	}
	baselineData, baselineErr := gitShow(repoRoot, "HEAD:"+tautologicalAllowlistRelativePath)
	if baselineErr != nil {
		return findings
	}
	baseline, baselineFindings := parseVacuousAllowlist(tautologicalAllowlistRelativePath, baselineData)
	findings = append(findings, baselineFindings...)
	for _, entryPath := range allowlistGrowth(current.Entries, baseline.Entries) {
		findings = append(findings, Finding{Code: "catalog.tautological_allowlist_growth", Severity: "error", Location: entryPath, Message: "tautological story allowlist may shrink but may not gain entries"})
	}
	for _, entry := range current.Entries {
		if allowlistPaths(baseline.Entries)[entry.Path] && allowlistSourceChanged(repoRoot, filepath.ToSlash(filepath.Join("scenarios/react-component-library", entry.Path))) {
			findings = append(findings, Finding{Code: "catalog.tautological_allowlist_source_changed", Severity: "error", Location: entry.Path, Message: "source changed while its tautological story remains allowlisted; remove the entry and add a real expectation"})
		}
	}
	return findings
}

// vacuousAllowlistFindings enforces the two-way ratchet around legacy
// contracts. The committed file is the comparison authority; a missing HEAD
// copy is allowed only while the allowlist is first being introduced.
func vacuousAllowlistFindings(repoRoot string) []Finding {
	path := filepath.Join(repoRoot, vacuousAllowlistRelativePath)
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "react-component-library")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Finding{{Code: "catalog.vacuous_allowlist_unreadable", Severity: "error", Location: vacuousAllowlistRelativePath, Message: err.Error()}}
	}
	current, findings := parseVacuousAllowlist(vacuousAllowlistRelativePath, data)
	if len(findings) > 0 {
		return findings
	}
	for _, entry := range current.Entries {
		contractPath := filepath.Join(scenarioRoot, filepath.FromSlash(entry.Path))
		if _, err := os.Stat(contractPath); err != nil {
			if isEvictedContract(scenarioRoot, entry.Path) {
				// Eviction removes the working-tree projection but preserves
				// the version identity and contract mirror in the registry.
				// The allowlist entry is still valid; it is not an unknown
				// contract merely because its bytes are cold.
				continue
			}
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_unknown_entry", Severity: "error", Location: entry.Path, Message: "allowlisted contract does not exist"})
		}
	}
	baselineData, err := gitShow(repoRoot, "HEAD:"+vacuousAllowlistRelativePath)
	if err != nil {
		return findings
	}
	baseline, baselineFindings := parseVacuousAllowlist(vacuousAllowlistRelativePath, baselineData)
	findings = append(findings, baselineFindings...)
	if len(baselineFindings) > 0 {
		return findings
	}
	for _, path := range allowlistGrowth(current.Entries, baseline.Entries) {
		findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_growth", Severity: "error", Location: path, Message: "legacy vacuous allowlist may shrink but may not gain entries"})
	}
	for _, entry := range current.Entries {
		if !allowlistPaths(baseline.Entries)[entry.Path] {
			continue
		}
		if allowlistSourceChanged(repoRoot, filepath.ToSlash(filepath.Join("scenarios/react-component-library", entry.Path))) {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_source_changed", Severity: "error", Location: entry.Path, Message: "source changed while its vacuous contract remains allowlisted; remove the entry and add an implemented machine claim"})
		}
	}
	return findings
}

func isEvictedContract(scenarioRoot, relativePath string) bool {
	clean := filepath.Clean(filepath.FromSlash(relativePath))
	if filepath.Base(clean) != "experience-contract.json" {
		return false
	}
	versionDir := filepath.Dir(clean)
	version := filepath.Base(versionDir)
	componentDir := filepath.Dir(filepath.Dir(versionDir))
	manifestPath := filepath.Join(scenarioRoot, componentDir, "component.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return false
	}
	var manifest struct {
		EvictedVersions []string `json:"evictedVersions"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return false
	}
	for _, evicted := range manifest.EvictedVersions {
		if strings.TrimSpace(evicted) == version {
			return true
		}
	}
	return false
}

func parseVacuousAllowlist(location string, data []byte) (vacuousAllowlistFile, []Finding) {
	var document vacuousAllowlistFile
	if err := json.Unmarshal(data, &document); err != nil {
		return document, []Finding{{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: err.Error()}}
	}
	if document.SchemaVersion != 1 {
		return document, []Finding{{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: "schemaVersion must be 1"}}
	}
	var findings []Finding
	previous := ""
	seen := map[string]bool{}
	for index := range document.Entries {
		entry := &document.Entries[index]
		entry.Path = filepath.ToSlash(strings.TrimSpace(entry.Path))
		entry.Reason = strings.TrimSpace(entry.Reason)
		if entry.Path == "" || entry.Reason == "" {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_invalid", Severity: "error", Location: location, Message: "every entry requires a path and written reason"})
		}
		if seen[entry.Path] {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_duplicate", Severity: "error", Location: entry.Path, Message: "allowlist entry is duplicated"})
		}
		seen[entry.Path] = true
		if previous != "" && entry.Path < previous {
			findings = append(findings, Finding{Code: "catalog.vacuous_allowlist_unsorted", Severity: "error", Location: location, Message: "allowlist entries must be sorted by path"})
		}
		previous = entry.Path
	}
	return document, findings
}

func allowlistGrowth(current, baseline []vacuousAllowlistEntry) []string {
	baselinePaths := allowlistPaths(baseline)
	var growth []string
	for _, entry := range current {
		if !baselinePaths[entry.Path] {
			growth = append(growth, entry.Path)
		}
	}
	sort.Strings(growth)
	return growth
}

func allowlistPaths(entries []vacuousAllowlistEntry) map[string]bool {
	out := make(map[string]bool, len(entries))
	for _, entry := range entries {
		out[filepath.ToSlash(strings.TrimSpace(entry.Path))] = true
	}
	return out
}

func gitShow(repoRoot, ref string) ([]byte, error) {
	command := exec.Command("git", "-C", repoRoot, "show", ref)
	return command.Output()
}

func allowlistSourceChanged(repoRoot, contractPath string) bool {
	versionDir := filepath.Dir(filepath.Join(repoRoot, filepath.FromSlash(contractPath)))
	paths, _ := filepath.Glob(filepath.Join(versionDir, "*.tsx"))
	paths2, _ := filepath.Glob(filepath.Join(versionDir, "*.ts"))
	paths = append(paths, paths2...)
	for _, path := range paths {
		relative := rel(repoRoot, path)
		command := exec.Command("git", "-C", repoRoot, "diff", "--quiet", "HEAD", "--", relative)
		if err := command.Run(); err != nil {
			return true
		}
	}
	return false
}
