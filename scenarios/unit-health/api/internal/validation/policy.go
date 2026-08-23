package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"unit-health/internal/adapterregistry"
	"unit-health/internal/adapters"
	"unit-health/internal/discovery"
	"unit-health/internal/executor"
)

type unitPolicyDocument struct {
	Unit struct {
		PolicyProfile unitPolicyProfile `json:"policy_profile"`
	} `json:"unit"`
}

type unitPolicyProfile struct {
	Version        string                       `json:"version"`
	Template       unitPolicyTemplate           `json:"template"`
	RequiredRoles  []unitPolicyRequiredRole     `json:"required_roles"`
	PolicyClasses  map[string]unitPolicyClass   `json:"policy_classes"`
	RunnerProfiles map[string]unitRunnerProfile `json:"runner_profiles,omitempty"`
	Customization  unitPolicyCustomization      `json:"customization"`
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
	Adapter        unitAdapterRef       `json:"adapter,omitempty"`
	RunnerProfile  string               `json:"runner_profile,omitempty"`
	TestKind       string               `json:"test_kind,omitempty"`
	Platforms      []string             `json:"platforms,omitempty"`
	Hermetic       unitHermeticity      `json:"hermetic,omitempty"`
	Language       string               `json:"language"`
	Framework      string               `json:"framework"`
	PackageManager string               `json:"package_manager,omitempty"`
	Coverage       unitCoveragePolicy   `json:"coverage"`
	TestUtils      unitTestUtilsPolicy  `json:"test_utils,omitempty"`
	Projection     unitProjectionPolicy `json:"projection,omitempty"`
}

type unitAdapterRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type unitRunnerProfile struct {
	CPUWeight              int      `json:"cpu_weight,omitempty"`
	MemoryBytes            int64    `json:"memory_bytes,omitempty"`
	MaxWorkers             int      `json:"max_workers,omitempty"`
	TimeoutSeconds         int      `json:"timeout_seconds,omitempty"`
	NoOutputTimeoutSeconds int      `json:"no_output_timeout_seconds,omitempty"`
	Sharding               string   `json:"sharding,omitempty"`
	Network                string   `json:"network,omitempty"`
	Filesystem             string   `json:"filesystem,omitempty"`
	Platforms              []string `json:"platforms,omitempty"`
}

type unitHermeticity struct {
	Network            string `json:"network,omitempty"`
	Filesystem         string `json:"filesystem,omitempty"`
	TemporaryRoot      bool   `json:"temporary_root,omitempty"`
	RestoreEnvironment bool   `json:"restore_environment,omitempty"`
	DetectOpenHandles  bool   `json:"detect_open_handles,omitempty"`
	DetectChildLeaks   bool   `json:"detect_child_leaks,omitempty"`
	OrderIndependent   bool   `json:"order_independent,omitempty"`
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
	RequiredFiles   []string                   `json:"required_files,omitempty"`
	RequiredScripts []string                   `json:"required_scripts,omitempty"`
	Settings        map[string]json.RawMessage `json:"settings,omitempty"`
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
			Remediation:  "Restore the missing surface or add a valid, time-bounded waiver with accountable ownership and evidence.",
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
		return unitPolicyProfile{}, path, false, []Finding{policyFileFinding(scenario, codeUnitPolicyInvalid, path, "unit.policy_profile missing", "unit.policy_profile.version 2.0.0 and versioned adapter/runner policy classes", "missing unit.policy_profile", now)}
	}
	return profile, path, true, nil
}

func validateUnitPolicyProfile(scenario, path string, profile unitPolicyProfile, now string) []Finding {
	var findings []Finding
	if profile.Version != "2.0.0" {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "version="+profile.Version, "unit.policy_profile.version 2.0.0", "unsupported policy-profile version", now))
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
		if class.Adapter.ID == "" || class.Adapter.Version == "" {
			findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+name+" adapter version missing", "adapter.id and adapter.version", "incomplete adapter identity", now))
		} else if len(class.Platforms) == 0 || containsPlatform(class.Platforms, runtime.GOOS) {
			if err := adapters.DefaultPlannerRegistry().ValidateIdentity(adapters.Identity{ID: class.Adapter.ID, Version: class.Adapter.Version}, adapters.Match{Language: class.Language, Framework: class.Framework, Platform: runtime.GOOS}); err != nil {
				findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+name+" adapter="+class.Adapter.ID+"@"+class.Adapter.Version, "registered adapter supports the declared language/framework/platform", err.Error(), now))
			}
		}
		if analyzer, ok := adapterregistry.Default().Resolve(class.Adapter.ID, class.Language, class.Framework); ok {
			if err := analyzer.ValidatePolicySettings(class.Projection.Settings); err != nil {
				findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+name+" projection.settings", "adapter-owned projection settings are valid", err.Error(), now))
			}
		}
		if class.RunnerProfile != "" {
			profileDef, ok := profile.RunnerProfiles[class.RunnerProfile]
			if !ok {
				findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+name+" runner_profile="+class.RunnerProfile, "runner_profile references a declared profile", "unknown runner profile", now))
			} else {
				findings = append(findings, validateRunnerProfile(scenario, path, name, profileDef, now)...)
			}
		}
		if class.TestKind != "" && !validUnitTestKind(class.TestKind) {
			findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+name+" test_kind="+class.TestKind, "unit, component, repository, integration, or workflow", "unknown test kind", now))
		}
		if min := minimumCoverageForPolicyClass(name, class); min > 0 && class.Coverage.MinimumPercent > 0 && class.Coverage.MinimumPercent < min {
			findings = append(findings, policyFileFinding(scenario, codeUnitPolicyWeakened, path, fmt.Sprintf("policy_class=%s minimum_percent=%.1f", name, class.Coverage.MinimumPercent), fmt.Sprintf("coverage minimum >= %.1f", min), "weaker coverage minimum", now))
		}
	}
	for i, waiver := range profile.Customization.Waivers {
		findings = append(findings, validateUnitPolicyWaiver(scenario, path, waiver, i, now)...)
	}
	findings = applyUnitPolicyWaivers(findings, profile.Customization.Waivers, scenario, path, now)
	return findings
}

func validUnitTestKind(kind string) bool {
	switch kind {
	case "unit", "component", "repository", "integration", "workflow":
		return true
	}
	return false
}

func containsPlatform(platforms []string, platform string) bool {
	for _, candidate := range platforms {
		if strings.EqualFold(strings.TrimSpace(candidate), platform) {
			return true
		}
	}
	return false
}

func validateRunnerProfile(scenario, path, class string, profile unitRunnerProfile, now string) []Finding {
	var findings []Finding
	if profile.CPUWeight < 0 || profile.MemoryBytes < 0 || profile.MaxWorkers < 0 || profile.TimeoutSeconds < 0 || profile.NoOutputTimeoutSeconds < 0 {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+class+" runner profile has negative budget", "non-negative runner budgets", "negative runner budget", now))
	}
	if profile.MaxWorkers == 0 && profile.CPUWeight > 0 {
		findings = append(findings, policyFileFinding(scenario, codeUnitPolicyInvalid, path, "policy_class="+class+" runner profile has no worker bound", "max_workers >= 1", "unbounded worker profile", now))
	}
	return findings
}

// applyRunnerProfiles projects only bounded execution controls into the
// neutral plan. Adapter-specific correctness remains in policy validation;
// this projection carries resource and timeout budgets to the executor.
func applyRunnerProfiles(root string, workspaces []Workspace) {
	profile, _, ok, _ := loadUnitPolicyProfile("", root, "")
	if !ok {
		return
	}
	bySurface := make(map[string]unitPolicyClass, len(profile.RequiredRoles))
	for _, role := range profile.RequiredRoles {
		if class, exists := profile.PolicyClasses[role.PolicyClass]; exists && role.Match.SurfaceID != "" {
			bySurface[role.Match.SurfaceID] = class
		}
	}
	for index := range workspaces {
		class, exists := bySurface[workspaces[index].ID]
		if !exists || class.RunnerProfile == "" {
			continue
		}
		runner, exists := profile.RunnerProfiles[class.RunnerProfile]
		if !exists {
			continue
		}
		workspaces[index].RunnerProfile = class.RunnerProfile
		workspaces[index].Resource = ResourceLimits{CPUWeight: runner.CPUWeight, MemoryBytes: runner.MemoryBytes, MaxWorkers: runner.MaxWorkers}
		workspaces[index].TimeoutSeconds = runner.TimeoutSeconds
		workspaces[index].NoOutputTimeoutSeconds = runner.NoOutputTimeoutSeconds
		workspaces[index].Hermetic = executor.HermeticPolicy{
			Network: class.Hermetic.Network, Filesystem: class.Hermetic.Filesystem,
			TemporaryRoot: class.Hermetic.TemporaryRoot, RestoreEnvironment: class.Hermetic.RestoreEnvironment,
			DetectChildLeaks: class.Hermetic.DetectChildLeaks, DetectOpenHandles: class.Hermetic.DetectOpenHandles,
			OrderIndependent: class.Hermetic.OrderIndependent,
		}
	}
}

// applyUnitPolicyWaivers marks matching findings for transparent suppression.
// Validation deliberately uses the same validator that emits
// UNIT_POLICY_WAIVER_INVALID: malformed and expired waivers can never suppress
// a real finding.
func applyUnitPolicyWaivers(findings []Finding, waivers []unitPolicyWaiver, scenario, path, now string) []Finding {
	for _, waiver := range waivers {
		if len(validateUnitPolicyWaiver(scenario, path, waiver, 0, now)) != 0 {
			continue
		}
		for i := range findings {
			if findings[i].Code == waiver.Finding {
				findings[i].Suppressed = true
			}
		}
	}
	return findings
}

// applyConfiguredUnitPolicyWaivers applies the validated profile waivers to
// findings from every analyzer, not only policy-profile findings. Keeping this
// at the response boundary makes waivers transparent and ensures newly added
// architecture rules cannot accidentally bypass the Phase 4 suppression
// contract.
func applyConfiguredUnitPolicyWaivers(findings []Finding, scenario, root, now string) []Finding {
	profile, path, ok, _ := loadUnitPolicyProfile(scenario, root, now)
	if !ok {
		return findings
	}
	return applyUnitPolicyWaivers(findings, profile.Customization.Waivers, scenario, path, now)
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
	want = adapterregistry.NormalizeFramework(want)
	got = adapterregistry.NormalizeFramework(got)
	return want != "" && want == got
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
	return adapterregistry.MinimumCoverageFloor(class.Adapter.ID, class.Framework, lang, name)
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
