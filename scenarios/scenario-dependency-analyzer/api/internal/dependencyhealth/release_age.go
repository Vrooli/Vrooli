package dependencyhealth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_health"
)

type releaseAgePolicy struct {
	Path                     string
	MinimumReleaseAge        int
	HasMinimumReleaseAge     bool
	MinimumReleaseAgeExclude []string
}

type releaseAgeException struct {
	PackageName   string `json:"package_name"`
	Spec          string `json:"spec"`
	Rationale     string `json:"rationale"`
	ApprovedBy    string `json:"approved_by"`
	ApprovedDate  string `json:"approved_date"`
	ReviewExpires string `json:"review_expires"`
	State         string `json:"state"`
}

type releaseAgeExceptionFile struct {
	ReleaseAgeExceptions []releaseAgeException `json:"release_age_exceptions"`
}

func (h *connectHandler) evaluateReleaseAge(scenario string, surfaces []*healthv1.DependencyHealthSurface) (*healthv1.DependencyHealthSection, []*healthv1.DependencyHealthFinding, *healthv1.DependencyPolicySummary) {
	scenarioDir := filepath.Join(h.resolveScenariosDir(), scenario)
	exceptions := loadReleaseAgeExceptions(filepath.Join(filepath.Dir(h.resolveScenariosDir()), ".vrooli", "dependencies", "approved-dependencies.json"))
	var findings []*healthv1.DependencyHealthFinding
	policies := map[string]struct{}{}
	exceptionCount := 0
	pnpmSurfaces := 0

	for _, surface := range surfaces {
		if !isJavaScriptSurface(surface) || packageManagerForSurface(surface) != "pnpm" || !fileExists(filepath.Join(surface.GetRootPath(), "package.json")) {
			continue
		}
		pnpmSurfaces++
		policy, found, err := readReleaseAgePolicy(surface.GetRootPath(), scenarioDir)
		if err != nil {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".unreadable", "ERROR", "pnpm release-age policy is unreadable", "SDA could not read the pnpm workspace policy for this surface.", "Fix the reported pnpm-workspace.yaml file, then rerun dependency health.", surface, "dependency.release_age.policy_readable", err.Error(), "readable pnpm release-age policy"))
			continue
		}
		if !found {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".missing", "ERROR", "pnpm release-age policy is missing", "This pnpm-managed dependency surface does not define a pnpm-workspace.yaml release-age gate.", "Add `minimumReleaseAge: 10080` to the surface's pnpm-workspace.yaml, or document an operator-approved exception.", surface, "dependency.release_age.minimum_configured", "missing minimumReleaseAge", fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
			continue
		}
		policies[filepath.ToSlash(policy.Path)] = struct{}{}
		if !policy.HasMinimumReleaseAge {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".minimum-missing", "ERROR", "pnpm release-age minimum is missing", "This pnpm workspace policy does not set minimumReleaseAge.", "Add `minimumReleaseAge: 10080` to the reported pnpm-workspace.yaml.", surface, "dependency.release_age.minimum_configured", "minimumReleaseAge unset", fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
		} else if policy.MinimumReleaseAge < releaseAgeMinimumMinutes {
			findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".minimum-too-low", "ERROR", "pnpm release-age minimum is too low", "This pnpm workspace allows dependency versions newer than the Vrooli default cooldown.", "Raise minimumReleaseAge to at least 10080 minutes, or file an explicit exception for the package that must bypass the cooldown.", surface, "dependency.release_age.minimum_value", fmt.Sprintf("minimumReleaseAge=%d", policy.MinimumReleaseAge), fmt.Sprintf("minimumReleaseAge >= %d", releaseAgeMinimumMinutes)))
		}
		for _, excluded := range policy.MinimumReleaseAgeExclude {
			exceptionCount++
			if !hasApprovedReleaseAgeException(exceptions, excluded) {
				findings = append(findings, releaseAgeFinding("policy."+surfaceID(surface)+".exclude."+slug(excluded), "ERROR", "pnpm release-age exclusion lacks governance approval", "A package bypasses the release-age gate but no approved dependency governance exception with rationale and expiry was found.", "Record the exception in .vrooli/dependencies/approved-dependencies.json with package/spec, rationale, approver, and review expiry, or remove the exclusion.", surface, "dependency.release_age.exclude_governed", excluded, "approved release-age exception with rationale and expiry"))
			}
		}
	}

	policyNames := sortedSet(policies)
	summary := &healthv1.DependencyPolicySummary{
		Status:                   statusFromFindings(findings, "release-age"),
		ReleaseAgeMinimumMinutes: releaseAgeMinimumMinutes,
		ReleaseAgeExceptions:     count32(exceptionCount),
		Policies:                 policyNames,
	}
	if pnpmSurfaces == 0 {
		summary.Status = "not_applicable"
		return section("release-age", "Package release-age policy", "not_applicable", "No pnpm-managed JavaScript/TypeScript dependency surfaces were discovered."), findings, summary
	}
	text := fmt.Sprintf("%d pnpm-managed dependency surface(s) checked for minimumReleaseAge >= %d minutes.", pnpmSurfaces, releaseAgeMinimumMinutes)
	return sectionWithFindingIDs("release-age", "Package release-age policy", summary.GetStatus(), text, findingIDs(findings, "release-age")), findings, summary
}

func readReleaseAgePolicy(surfaceRoot, scenarioDir string) (releaseAgePolicy, bool, error) {
	for _, path := range candidatePNPMWorkspacePaths(surfaceRoot, scenarioDir) {
		if !fileExists(path) {
			continue
		}
		policy, err := parseReleaseAgePolicy(path)
		return policy, true, err
	}
	return releaseAgePolicy{}, false, nil
}

func candidatePNPMWorkspacePaths(surfaceRoot, scenarioDir string) []string {
	surfaceRoot = filepath.Clean(surfaceRoot)
	scenarioDir = filepath.Clean(scenarioDir)
	var paths []string
	for dir := surfaceRoot; ; dir = filepath.Dir(dir) {
		paths = append(paths, filepath.Join(dir, "pnpm-workspace.yaml"))
		if dir == scenarioDir || dir == filepath.Dir(dir) {
			break
		}
	}
	return paths
}

func parseReleaseAgePolicy(path string) (releaseAgePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return releaseAgePolicy{}, err
	}
	policy := releaseAgePolicy{Path: path}
	lines := strings.Split(string(data), "\n")
	for i := 0; i < len(lines); i++ {
		line := stripYAMLComment(lines[i])
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "minimumReleaseAge":
			var minutes int
			if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &minutes); err != nil {
				return releaseAgePolicy{}, fmt.Errorf("parse minimumReleaseAge in %s: %w", path, err)
			}
			policy.MinimumReleaseAge = minutes
			policy.HasMinimumReleaseAge = true
		case "minimumReleaseAgeExclude":
			inline := strings.TrimSpace(value)
			if inline == "[]" {
				continue
			}
			for j := i + 1; j < len(lines); j++ {
				next := stripYAMLComment(lines[j])
				if strings.TrimSpace(next) == "" {
					continue
				}
				if !strings.HasPrefix(next, " ") && !strings.HasPrefix(next, "\t") {
					break
				}
				item := strings.TrimSpace(next)
				if !strings.HasPrefix(item, "-") {
					continue
				}
				excluded := strings.TrimSpace(strings.TrimPrefix(item, "-"))
				excluded = strings.Trim(excluded, `"'`)
				if excluded != "" {
					policy.MinimumReleaseAgeExclude = append(policy.MinimumReleaseAgeExclude, excluded)
				}
			}
		}
	}
	return policy, nil
}

func stripYAMLComment(line string) string {
	inSingle := false
	inDouble := false
	for i, r := range line {
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble {
				return line[:i]
			}
		}
	}
	return line
}

func loadReleaseAgeExceptions(path string) []releaseAgeException {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var file releaseAgeExceptionFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil
	}
	return file.ReleaseAgeExceptions
}

func hasApprovedReleaseAgeException(exceptions []releaseAgeException, excluded string) bool {
	for _, exception := range exceptions {
		if !strings.EqualFold(strings.TrimSpace(exception.State), "approved") {
			continue
		}
		if strings.TrimSpace(exception.Rationale) == "" || strings.TrimSpace(exception.ReviewExpires) == "" {
			continue
		}
		if sameReleaseAgeException(exception, excluded) {
			return true
		}
	}
	return false
}

func sameReleaseAgeException(exception releaseAgeException, excluded string) bool {
	excluded = strings.TrimSpace(excluded)
	spec := strings.TrimSpace(exception.Spec)
	if spec != "" && strings.EqualFold(spec, excluded) {
		return true
	}
	name := strings.TrimSpace(exception.PackageName)
	if name == "" {
		return false
	}
	if strings.EqualFold(name, excluded) {
		return true
	}
	excludedName, _, hasVersion := strings.Cut(excluded, "@")
	if strings.HasPrefix(excluded, "@") {
		parts := strings.SplitN(strings.TrimPrefix(excluded, "@"), "@", 2)
		excludedName = "@" + parts[0]
		hasVersion = len(parts) == 2
	}
	return !hasVersion && strings.EqualFold(name, excludedName)
}

func releaseAgeFinding(id, severity, title, description, remediation string, surface *healthv1.DependencyHealthSurface, ruleID, observed, expected string) *healthv1.DependencyHealthFinding {
	return &healthv1.DependencyHealthFinding{
		Id:           "release-age." + slug(id),
		Severity:     severity,
		SourceDomain: "release-age",
		Title:        title,
		Description:  description,
		Remediation:  remediation,
		FilePath:     releaseAgeFilePath(surface),
		SurfaceId:    surface.GetId(),
		RuleId:       ruleID,
		Observed:     observed,
		Expected:     expected,
	}
}

func releaseAgeFilePath(surface *healthv1.DependencyHealthSurface) string {
	for _, path := range candidatePNPMWorkspacePaths(surface.GetRootPath(), filepath.Dir(surface.GetRootPath())) {
		if fileExists(path) {
			return relScenarioPath(path)
		}
	}
	return relScenarioPath(filepath.Join(surface.GetRootPath(), "pnpm-workspace.yaml"))
}

func isJavaScriptSurface(surface *healthv1.DependencyHealthSurface) bool {
	switch normalizedLanguage(surface.GetLanguage()) {
	case "javascript", "typescript":
		return true
	default:
		return false
	}
}
