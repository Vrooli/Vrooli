package dependencyhealth

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func sectionWithFindingIDs(id, title, status, summary string, ids []string) *healthv1.DependencyHealthSection {
	out := section(id, title, status, summary)
	out.FindingIds = ids
	return out
}

func statusFromFindings(findings []*healthv1.DependencyHealthFinding, domain string) string {
	status := "pass"
	for _, finding := range findings {
		if domain != "" && finding.GetSourceDomain() != domain {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(finding.GetSeverity())) {
		case "ERROR", "CRITICAL", "HIGH":
			return "fail"
		case "WARNING", "WARN", "MEDIUM":
			status = "warn"
		}
	}
	return status
}

func summarizeReadiness(findings []*healthv1.DependencyHealthFinding, commandResults []*healthv1.DependencyHealthCommandResult) string {
	status := statusFromFindings(findings, "readiness")
	if status == "pass" {
		return fmt.Sprintf("Host commands, runtimes, modules, and packages passed readiness checks (%d command probe(s)).", len(commandResults))
	}
	return fmt.Sprintf("%d readiness finding(s) across host commands, runtimes, modules, and packages.", len(findingIDs(findings, "readiness")))
}

func findingIDs(findings []*healthv1.DependencyHealthFinding, domain string) []string {
	var ids []string
	for _, finding := range findings {
		if domain == "" || finding.GetSourceDomain() == domain {
			ids = append(ids, finding.GetId())
		}
	}
	sort.Strings(ids)
	return ids
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedSet(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func scenarioDirFromSurfaces(surfaces []*healthv1.DependencyHealthSurface) string {
	for _, surface := range surfaces {
		root := strings.TrimSpace(packageRoot(surface))
		if root != "" {
			return root
		}
	}
	return ""
}

func packageRoot(surface *healthv1.DependencyHealthSurface) string {
	return firstNonEmpty(surface.GetParseUnitRoot(), surface.GetRootPath())
}

func packageConfigPath(surface *healthv1.DependencyHealthSurface, fallbackName string) string {
	if config := strings.TrimSpace(surface.GetConfigPath()); config != "" {
		return config
	}
	root := packageRoot(surface)
	if root == "" || fallbackName == "" {
		return root
	}
	return filepath.Join(root, fallbackName)
}

func surfaceID(surface *healthv1.DependencyHealthSurface) string {
	return firstNonEmpty(surface.GetId(), filepath.Base(surface.GetRootPath()), "surface")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func relScenarioPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/scenarios/")
	if len(parts) < 2 {
		return filepath.ToSlash(path)
	}
	return "scenarios/" + parts[len(parts)-1]
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func section(id, title, status, summary string) *healthv1.DependencyHealthSection {
	return &healthv1.DependencyHealthSection{
		Id:      id,
		Title:   title,
		Status:  status,
		Summary: summary,
	}
}

func finalize(resp *healthv1.DependencyHealthResponse) {
	summary := &healthv1.DependencyHealthSummary{
		Sections:             count32(len(resp.Sections)),
		Surfaces:             count32(len(resp.Surfaces)),
		Findings:             count32(len(resp.Findings)),
		DegradedDependencies: count32(len(resp.DegradedDependencies)),
	}
	passed := true
	for _, section := range resp.Sections {
		if strings.EqualFold(strings.TrimSpace(section.GetStatus()), "pending") {
			passed = false
		}
	}
	for _, finding := range resp.Findings {
		switch strings.ToUpper(strings.TrimSpace(finding.GetSeverity())) {
		case "ERROR", "CRITICAL", "HIGH":
			summary.Errors++
			passed = false
		case "WARNING", "WARN", "MEDIUM":
			summary.Warnings++
		default:
			summary.Infos++
		}
	}
	resp.Summary = summary
	resp.Passed = passed && len(resp.DegradedDependencies) == 0
}

func count32(count int) int32 {
	if count > 1<<31-1 {
		return 1<<31 - 1
	}
	if count < 0 {
		return 0
	}
	return int32(count)
}
