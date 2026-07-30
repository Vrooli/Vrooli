package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"unit-health/internal/discovery"
)

type unitPolicyDocument struct {
	Unit struct {
		PolicyProfile unitPolicyProfile `json:"policy_profile"`
	} `json:"unit"`
}

type unitPolicyProfile struct {
	Version       string                     `json:"version"`
	Template      unitPolicyTemplate         `json:"template"`
	RequiredRoles []unitPolicyRequiredRole   `json:"required_roles"`
	PolicyClasses map[string]unitPolicyClass `json:"policy_classes"`
	Customization unitPolicyCustomization    `json:"customization"`
}

type unitPolicyTemplate struct {
	ID            string `json:"id"`
	ScenarioClass string `json:"scenario_class"`
}

type unitPolicyRequiredRole struct {
	Role        string          `json:"role"`
	PolicyClass string          `json:"policy_class"`
	Match       unitPolicyMatch `json:"match"`
}

type unitPolicyMatch struct {
	SurfaceID string `json:"surface_id"`
	Kind      string `json:"kind"`
	Path      string `json:"path"`
	Language  string `json:"language"`
	Framework string `json:"framework"`
}

type unitPolicyClass struct {
	Language       string               `json:"language"`
	Framework      string               `json:"framework"`
	PackageManager string               `json:"package_manager,omitempty"`
	Coverage       unitCoveragePolicy   `json:"coverage"`
	TestUtils      unitTestUtilsPolicy  `json:"test_utils,omitempty"`
	Projection     unitProjectionPolicy `json:"projection,omitempty"`
}

type unitCoveragePolicy struct {
	MinimumPercent float64  `json:"minimum_percent"`
	Mode           string   `json:"mode,omitempty"`
	Provider       string   `json:"provider,omitempty"`
	Reporters      []string `json:"reporters,omitempty"`
}

type unitTestUtilsPolicy struct {
	RequiredRoots         []string `json:"required_roots,omitempty"`
	ProductionImportBan   bool     `json:"production_import_ban,omitempty"`
	CanonicalRenderHelper string   `json:"canonical_render_helper,omitempty"`
}

type unitProjectionPolicy struct {
	RequiredFiles   []string             `json:"required_files,omitempty"`
	RequiredScripts []string             `json:"required_scripts,omitempty"`
	Vitest          unitVitestProjection `json:"vitest,omitempty"`
}

type unitVitestProjection struct {
	Environment      string   `json:"environment,omitempty"`
	SetupFiles       []string `json:"setup_files,omitempty"`
	CoverageProvider string   `json:"coverage_provider,omitempty"`
}

type unitPolicyCustomization struct {
	Mode    string             `json:"mode"`
	Waivers []unitPolicyWaiver `json:"waivers"`
}

type unitPolicyWaiver struct {
	Finding   string `json:"finding"`
	Reason    string `json:"reason"`
	Owner     string `json:"owner"`
	ExpiresAt string `json:"expires_at"`
	Revisit   string `json:"revisit"`
	Evidence  string `json:"evidence"`
}

func resolveUnitPolicyFindings(scenario string, inv discovery.Inventory, now string) []Finding {
	profile, path, ok, findings := loadUnitPolicyProfile(scenario, inv.RootPath, now)
	if !ok {
		return findings
	}
	findings = append(findings, validateUnitPolicyProfile(scenario, path, profile, now)...)

	roleBySurface := map[string]string{}
	for _, role := range profile.RequiredRoles {
		if surface, found := findRoleSurface(role, inv); found {
			roleBySurface[surface.ID] = role.Role
			continue
		}
		findings = append(findings, Finding{
			ID:           codeUnitRequiredRoleMissing + "-" + role.Role,
			Scenario:     scenario,
			Code:         codeUnitRequiredRoleMissing,
			Category:     "policy",
			Severity:     codeSeverity[codeUnitRequiredRoleMissing],
			FilePath:     path,
			Message:      fmt.Sprintf("Required unit-test role %q was not observed by Code Facts.", role.Role),
			Evidence:     fmt.Sprintf("role=%s policy_class=%s", role.Role, role.PolicyClass),
			Expected:     "Code Facts observes every role required by the unit policy profile.",
			Observed:     "required role missing from observed surfaces",
			WhyItMatters: "Template policy cannot protect a surface that is absent from discovery.",
			Remediation:  "Restore the missing surface or add a formal waiver once waiver enforcement is available.",
			CreatedAt:    now,
		})
	}

	for _, surface := range inv.Surfaces {
		// Code Facts can retain a declared surface after its directory has been
		// removed. A missing path is not a runnable product surface, so it must
		// not require a policy role or lower maturity until discovery reports it
		// as present again.
		if strings.EqualFold(surface.Status, "missing") {
			continue
		}
		if _, ok := roleBySurface[surface.ID]; ok {
			continue
		}
		if hasSupportedUnitDefault(surface) {
			continue
		}
		findings = append(findings, Finding{
			ID:          codeUnitSurfaceUngoverned + "-" + surface.ID,
			Scenario:    scenario,
			SurfaceID:   surface.ID,
			Language:    normalizeLanguage(surface.Language, surface.RootPath),
			Framework:   surface.Framework,
			Code:        codeUnitSurfaceUngoverned,
			Category:    "policy",
			Severity:    codeSeverity[codeUnitSurfaceUngoverned],
			FilePath:    surface.RootPath,
			Message:     fmt.Sprintf("Observed surface %q has no unit policy class or supported Unit Health default.", surface.ID),
			Evidence:    fmt.Sprintf("kind=%s language=%s framework=%s", surface.Kind, surface.Language, surface.Framework),
			Expected:    "Every observed surface is governed by a required role, explicit profile rule, or Unit Health language default.",
			Observed:    "surface ungoverned",
			Remediation: "Add an additive policy class for this surface or extend Unit Health defaults for its language/framework.",
			CreatedAt:   now,
		})
	}
	return findings
}

func loadUnitPolicyProfile(scenario, root, now string) (unitPolicyProfile, string, bool, []Finding) {
	path := filepath.Join(root, ".vrooli", "testing.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return unitPolicyProfile{}, path, false, nil
	}
	var doc unitPolicyDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		return unitPolicyProfile{}, path, false, []Finding{policyFileFinding(scenario, codeUnitPolicyInvalid, path, err.Error(), "valid JSON testing configuration", "invalid JSON", now)}
	}
	profile := doc.Unit.PolicyProfile
	if strings.TrimSpace(profile.Version) == "" {
		return unitPolicyProfile{}, path, false, []Finding{policyFileFinding(scenario, codeUnitPolicyInvalid, path, "unit.policy_profile missing", "unit.policy_profile.version 1.0.0 and template policy classes", "missing unit.policy_profile", now)}
	}
	return profile, path, true, nil
}

func validateUnitPolicyProfile(scenario, path string, profile unitPolicyProfile, now string) []Finding {
	var findings []Finding
	if profile.Version != "1.0.0" {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "version="+profile.Version, "unit.policy_profile.version 1.0.0", "unsupported policy-profile version", now))
	}
	if profile.Template.ID == "" || profile.Template.ScenarioClass == "" {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "template identity incomplete", "template.id and template.scenario_class", "missing template identity", now))
	}
	if profile.Customization.Mode != "monotonic" {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "customization.mode="+profile.Customization.Mode, "customization.mode=monotonic", "unsupported customization mode", now))
	}
	for _, role := range profile.RequiredRoles {
		if _, ok := profile.PolicyClasses[role.PolicyClass]; !ok {
			findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "role="+role.Role+" policy_class="+role.PolicyClass, "required roles reference declared policy classes", "unknown policy class", now))
		}
	}
	for name, class := range profile.PolicyClasses {
		if min := minimumCoverageForPolicyClass(name, class); min > 0 && class.Coverage.MinimumPercent > 0 && class.Coverage.MinimumPercent < min {
			findings = append(findings, policyFileFinding(scenario, codeUnitPolicyWeakened, path, fmt.Sprintf("policy_class=%s minimum_percent=%.1f", name, class.Coverage.MinimumPercent), fmt.Sprintf("coverage minimum >= %.1f", min), "weaker coverage minimum", now))
		}
	}
	for i, waiver := range profile.Customization.Waivers {
		findings = append(findings, validateUnitPolicyWaiver(scenario, path, waiver, i, now)...)
	}
	return findings
}

func validateUnitPolicyWaiver(scenario, path string, waiver unitPolicyWaiver, index int, now string) []Finding {
	var problems []string
	if strings.TrimSpace(waiver.Finding) == "" {
		problems = append(problems, "missing finding")
	} else if _, ok := codeSeverity[waiver.Finding]; !ok {
		problems = append(problems, "unknown finding="+waiver.Finding)
	}
	if strings.TrimSpace(waiver.Reason) == "" {
		problems = append(problems, "missing reason")
	}
	if weakFreeText(waiver.Owner) {
		problems = append(problems, "missing accountable owner")
	}
	if weakFreeText(waiver.Evidence) {
		problems = append(problems, "missing evidence reference")
	}
	if strings.TrimSpace(waiver.ExpiresAt) == "" && strings.TrimSpace(waiver.Revisit) == "" {
		problems = append(problems, "missing expires_at or revisit")
	}
	if strings.TrimSpace(waiver.ExpiresAt) != "" {
		expires, err := parseWaiverTime(waiver.ExpiresAt)
		if err != nil {
			problems = append(problems, "invalid expires_at="+waiver.ExpiresAt)
		} else if current, ok := parseCurrentTime(now); ok && !expires.After(current) {
			problems = append(problems, "expired at "+waiver.ExpiresAt)
		}
	}
	if len(problems) == 0 {
		return nil
	}
	evidence := fmt.Sprintf("waiver[%d] %s", index, strings.Join(problems, "; "))
	return []Finding{policyFileFinding(scenario, codeUnitWaiverInvalid, path, evidence, "waiver with known finding, reason, owner, evidence, and future expires_at or concrete revisit trigger", "invalid waiver", now)}
}

func weakFreeText(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "todo", "tbd", "unknown", "none", "n/a":
		return true
	default:
		return false
	}
}

func parseCurrentTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, value)
	return t, err == nil
}

func parseWaiverTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("expected RFC3339 timestamp or YYYY-MM-DD date")
}

func findRoleSurface(role unitPolicyRequiredRole, inv discovery.Inventory) (discovery.Surface, bool) {
	for _, surface := range inv.Surfaces {
		if role.Match.SurfaceID != "" && !strings.EqualFold(role.Match.SurfaceID, surface.ID) {
			continue
		}
		if role.Match.Kind != "" && !strings.EqualFold(role.Match.Kind, surface.Kind) {
			continue
		}
		if role.Match.Language != "" && normalizeLanguage(surface.Language, surface.RootPath) != normalizeLanguage(role.Match.Language, surface.RootPath) {
			continue
		}
		if role.Match.Framework != "" && !frameworkMatches(role.Match.Framework, surface.Framework) {
			continue
		}
		if role.Match.Path != "" && !pathMatches(role.Match.Path, surface.RootPath, inv.RootPath) {
			continue
		}
		return surface, true
	}
	return discovery.Surface{}, false
}

func frameworkMatches(want, got string) bool {
	want = normalizePolicyFramework(want)
	got = normalizePolicyFramework(got)
	return want != "" && want == got
}

func normalizePolicyFramework(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "react-vite":
		return "vite"
	default:
		return value
	}
}

func pathMatches(want, got, root string) bool {
	want = filepath.Clean(want)
	got = filepath.Clean(got)
	if filepath.IsAbs(want) {
		return want == got
	}
	if root != "" && filepath.Clean(filepath.Join(root, want)) == got {
		return true
	}
	return filepath.Base(got) == want
}

func hasSupportedUnitDefault(surface discovery.Surface) bool {
	switch normalizeLanguage(surface.Language, surface.RootPath) {
	case "go", "typescript", "python", "bash":
		return true
	default:
		return false
	}
}

func minimumCoverageForPolicyClass(name string, class unitPolicyClass) float64 {
	lang := normalizeLanguage(class.Language, "")
	switch {
	case strings.Contains(name, "react_vite") || class.Framework == "vitest":
		return 85
	case lang == "go":
		return 75
	default:
		return 0
	}
}

func policyFileFinding(scenario, code, path, evidence, expected, observed, now string) Finding {
	return Finding{
		ID:           code,
		Scenario:     scenario,
		Code:         code,
		Category:     "policy",
		Severity:     codeSeverity[code],
		FilePath:     path,
		Message:      policyMessage(code),
		Evidence:     evidence,
		Expected:     expected,
		Observed:     observed,
		WhyItMatters: "Unit-test policy has to be explicit before Unit Health can enforce template-derived testing contracts.",
		Remediation:  "Update .vrooli/testing.json unit.policy_profile so it matches the schema and is equal-or-stricter than the template baseline.",
		CreatedAt:    now,
	}
}

func policyMessage(code string) string {
	switch code {
	case codeUnitPolicyWeakened:
		return "Unit policy profile weakens the template baseline."
	case codeUnitWaiverInvalid:
		return "Unit policy waiver is incomplete or invalid."
	default:
		return "Unit policy profile is invalid."
	}
}
