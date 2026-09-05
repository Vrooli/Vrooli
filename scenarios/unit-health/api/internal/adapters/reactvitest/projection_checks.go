package reactvitest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"unit-health/internal/adapters"
)

// ValidatePolicySettings rejects unknown or malformed native projection keys
// before execution. The generic policy schema intentionally does not encode
// these adapter-owned details.
func ValidatePolicySettings(settings map[string]json.RawMessage) error {
	for key, raw := range settings {
		switch key {
		case "environment", "coverage_provider":
			var value string
			if err := json.Unmarshal(raw, &value); err != nil || strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s must be a non-empty string", key)
			}
		case "setup_files":
			var values []string
			if err := json.Unmarshal(raw, &values); err != nil || len(values) == 0 {
				return fmt.Errorf("%s must be a non-empty string array", key)
			}
			for _, value := range values {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("%s must contain non-empty strings", key)
				}
			}
		default:
			return fmt.Errorf("unsupported React/Vitest projection setting %q", key)
		}
	}
	return nil
}

// ProjectionPolicy is the adapter-neutral input projection for a React/Vitest
// policy class. The adapter owns how native files express these values.
type ProjectionPolicy = adapters.ProjectionPolicy

// DefaultPolicy is the canonical React/Vite projection contract used by the
// template. Scenario policy settings may only strengthen or specialize it.
func DefaultPolicy() ProjectionPolicy {
	return ProjectionPolicy{
		CoverageFloor:     80,
		Environment:       "jsdom",
		SetupFiles:        []string{"./src/test-setup.ts"},
		CoverageProvider:  "v8",
		CoverageReporters: []string{"json-summary", "json"},
		CoverageInclude:   []string{"src/**/*.{ts,tsx}"},
		CoverageExclude: []string{
			"src/**/*.test.{ts,tsx}", "src/**/*.spec.{ts,tsx}", "src/**/*.d.ts",
			"src/main.tsx", "src/test-setup.ts", "src/test-utils/**",
			"src/consts/strings.generated.ts", "src/i18n/locales/**", "src/**/generated/**",
		},
		ReportOnFailure: true,
	}
}

// PolicyFromSettings decodes the adapter-owned projection settings carried by
// the framework-neutral policy contract. The kernel stores these values as
// opaque JSON; only this adapter interprets the native keys.
func PolicyFromSettings(settings map[string]json.RawMessage) ProjectionPolicy {
	var policy ProjectionPolicy
	decode := func(key string, target any) {
		if raw, ok := settings[key]; ok {
			_ = json.Unmarshal(raw, target)
		}
	}
	decode("environment", &policy.Environment)
	decode("setup_files", &policy.SetupFiles)
	decode("coverage_provider", &policy.CoverageProvider)
	return policy
}

type (
	ProjectionInput = adapters.ProjectionInput
	ProjectionDrift = adapters.ProjectionDrift
)

type packageManifest struct {
	Scripts         map[string]string `json:"scripts"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

// Manifest is the adapter-owned package manifest projection used by the
// validation reporting bridge.
type Manifest = packageManifest

func LoadManifest(root string) (Manifest, error) {
	return loadManifest(filepath.Join(root, "package.json"))
}

func (m packageManifest) HasDependency(name string) bool { return m.hasDependency(name) }
func (m packageManifest) HasScript(name string) bool     { return m.hasScript(name) }

func (m packageManifest) hasDependency(name string) bool {
	if _, ok := m.Dependencies[name]; ok {
		return true
	}
	_, ok := m.DevDependencies[name]
	return ok
}

func (m packageManifest) hasScript(name string) bool {
	_, ok := m.Scripts[name]
	return ok
}

func loadManifest(path string) (packageManifest, error) {
	var manifest packageManifest
	if err := readJSON(path, &manifest); err != nil {
		return packageManifest{}, err
	}
	return manifest, nil
}

// AnalyzeProjection checks the native React/Vitest projection without
// constructing Unit Health findings or importing validation policy types.
func AnalyzeProjection(input ProjectionInput) []ProjectionDrift {
	root := input.RootPath
	vitePath := filepath.Join(root, "vite.config.ts")
	config := readText(vitePath)
	if config == "" {
		vitePath = filepath.Join(root, "vite.config.js")
		config = readText(vitePath)
	}
	if config == "" {
		return nil
	}
	manifest, _ := LoadManifest(root)
	projection := Parse(config, readText(filepath.Join(root, "eslint.config.js")))
	policy := input.Policy
	var drifts []ProjectionDrift
	add := func(file, message, evidence, expected, observed, remediation string) {
		drifts = append(drifts, ProjectionDrift{File: file, Message: message, Evidence: evidence, Expected: expected, Observed: observed, Remediation: remediation})
	}

	pkgPath := filepath.Join(root, "package.json")
	if !manifest.hasDependency("vitest") {
		add(pkgPath, "UI policy projection is missing the Vitest dependency.", "vitest dependency not found", "package.json declares vitest as the React/Vite unit-test runner.", "missing vitest dependency", "Add vitest through Scenario Dependency Analyzer and keep the test scripts on Vitest.")
	}
	if !manifest.hasScript("test") || !strings.Contains(manifest.Scripts["test"], "vitest") {
		add(pkgPath, "UI policy projection is missing a Vitest test script.", fmt.Sprintf("test=%q", manifest.Scripts["test"]), "package.json scripts.test runs vitest.", "missing or non-Vitest test script", "Set scripts.test to a Vitest command such as \"vitest run\".")
	}
	if !manifest.hasScript("test:coverage") || !strings.Contains(manifest.Scripts["test:coverage"], "vitest") || !strings.Contains(manifest.Scripts["test:coverage"], "coverage") {
		add(pkgPath, "UI policy projection is missing a coverage test script.", fmt.Sprintf("test:coverage=%q", manifest.Scripts["test:coverage"]), "package.json scripts.test:coverage runs Vitest coverage.", "missing or non-Vitest coverage script", "Set scripts.test:coverage to a Vitest coverage command such as \"vitest run --coverage\".")
	}
	if !projection.HasVitestConfig {
		add(vitePath, "UI policy projection is missing the Vite test block.", "no test: block detected", "vite.config declares a test block.", "missing Vitest config", "Add test configuration to vite.config.")
	}
	if projection.Environment != policy.Environment {
		add(vitePath, "UI policy projection is missing jsdom test environment.", "environment="+policy.Environment+" not detected", "Vitest test.environment is "+policy.Environment+".", "missing jsdom environment", "Set test.environment to "+fmt.Sprintf("%q", policy.Environment)+".")
	}
	if !ContainsAllSetupFiles(projection.SetupFiles, policy.SetupFiles) {
		add(vitePath, "UI policy projection is missing setupFiles registration.", "setupFiles="+strings.Join(policy.SetupFiles, ",")+" not detected", "Vitest setupFiles includes "+strings.Join(policy.SetupFiles, ", ")+".", "missing setupFiles", "Register the policy-declared setup file(s) in test.setupFiles.")
	}
	if projection.CoverageProvider != policy.CoverageProvider {
		add(vitePath, "UI policy projection is missing V8 coverage provider.", "coverage.provider="+policy.CoverageProvider+" not detected", "Vitest coverage.provider is "+policy.CoverageProvider+".", "missing V8 coverage provider", "Set coverage.provider to "+fmt.Sprintf("%q", policy.CoverageProvider)+".")
	}
	if !ContainsAllStrings(projection.CoverageReporters, policy.CoverageReporters) {
		add(vitePath, "UI policy projection is missing coverage reporters.", strings.Join(policy.CoverageReporters, "/")+" reporters not all detected", "Coverage reporters include "+strings.Join(policy.CoverageReporters, ", ")+".", "missing coverage reporters", "Include the policy-declared reporters in coverage.reporter.")
	}
	if !ContainsAllStrings(projection.CoverageInclude, policy.CoverageInclude) {
		add(vitePath, "UI policy projection is missing an explicit source coverage include set.", "coverage.include="+strings.Join(projection.CoverageInclude, ", "), "Coverage include contains "+strings.Join(policy.CoverageInclude, ", ")+".", "missing source coverage include", "Set coverage.include so coverage denominators stay scoped to production source files.")
	}
	if !ContainsAllStrings(projection.CoverageExclude, policy.CoverageExclude) {
		add(vitePath, "UI policy projection is missing canonical coverage exclusions.", "coverage.exclude="+strings.Join(projection.CoverageExclude, ", "), "Coverage exclude contains test scaffolding, generated files, boot files, and locale catalogs.", "missing coverage exclusions", "Restore the canonical coverage.exclude entries without weakening production-source coverage.")
	}
	if policy.ReportOnFailure && (!projection.HasReportOnFailure || !projection.ReportOnFailure) {
		add(vitePath, "UI policy projection is missing coverage reporting on test failure.", "coverage.reportOnFailure=true not detected", "Vitest coverage.reportOnFailure is true.", "missing coverage reportOnFailure", "Set coverage.reportOnFailure to true so failed coverage runs remain interpretable.")
	}
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		value, ok := projection.Thresholds[key]
		if !ok || value < policy.CoverageFloor {
			add(vitePath, "UI policy projection weakens Vitest coverage thresholds.", fmt.Sprintf("%s=%.1f", key, value), fmt.Sprintf("Vitest coverage thresholds are at least %.1f for lines, functions, branches, and statements.", policy.CoverageFloor), "coverage threshold below policy", fmt.Sprintf("Restore the threshold to %.1f or higher.", policy.CoverageFloor))
		}
	}
	if !projection.HasImportBanRule {
		add(filepath.Join(root, "eslint.config.js"), "UI policy projection is missing the production import ban for test helpers.", "no-restricted-imports/test-utils pattern not detected", "ESLint forbids production imports from src/test-utils and feature mocks.", "missing production import ban", "Restore the no-restricted-imports patterns for test-utils and feature mocks.")
	}
	return drifts
}

// AnalyzeProjectionChecks exposes the adapter's detailed native projection
// evidence without leaking manifest or Vite parsing into the validation
// kernel. The keys are stable adapter evidence identifiers, not framework
// selection logic.
func AnalyzeProjectionChecks(input ProjectionInput) []adapters.ProjectionCheck {
	root := input.RootPath
	configPath := filepath.Join(root, "vite.config.ts")
	config := readText(configPath)
	if config == "" {
		configPath = filepath.Join(root, "vite.config.js")
		config = readText(configPath)
	}
	eslintPath := filepath.Join(root, "eslint.config.js")
	manifest, _ := LoadManifest(root)
	projection := Parse(config, readText(eslintPath))
	policy := input.Policy
	checks := make([]adapters.ProjectionCheck, 0, 15)
	add := func(key, owner, file, policyValue, nativeValue string, pass bool, remediation string) {
		checks = append(checks, adapters.ProjectionCheck{Key: key, Owner: owner, File: file, PolicyValue: policyValue, NativeValue: nativeValue, Pass: pass, Remediation: remediation})
	}
	add("runner.dependency", "package.json dependencies", filepath.Join(root, "package.json"), "vitest", boolText(manifest.HasDependency("vitest")), manifest.HasDependency("vitest"), "Add vitest through Scenario Dependency Analyzer and keep the test scripts on Vitest.")
	add("script.test", "package.json scripts", filepath.Join(root, "package.json"), "contains vitest", manifest.Scripts["test"], strings.Contains(manifest.Scripts["test"], "vitest"), "Set scripts.test to a Vitest command such as \"vitest run\".")
	add("script.coverage", "package.json scripts", filepath.Join(root, "package.json"), "contains vitest and coverage", manifest.Scripts["test:coverage"], strings.Contains(manifest.Scripts["test:coverage"], "vitest") && strings.Contains(manifest.Scripts["test:coverage"], "coverage"), "Set scripts.test:coverage to a Vitest coverage command such as \"vitest run --coverage\".")
	add("vitest.environment", "vite.config.ts test", configPath, policy.Environment, projection.Environment, projection.Environment == policy.Environment, "Set test.environment to the policy-declared environment.")
	add("vitest.setup_files", "vite.config.ts test", configPath, strings.Join(policy.SetupFiles, ","), strings.Join(projection.SetupFiles, ","), ContainsAllSetupFiles(projection.SetupFiles, policy.SetupFiles), "Register the policy-declared setup file(s) in test.setupFiles.")
	add("coverage.provider", "vite.config.ts test.coverage", configPath, policy.CoverageProvider, projection.CoverageProvider, projection.CoverageProvider == policy.CoverageProvider, "Set coverage.provider to the policy-declared provider.")
	add("coverage.reporters", "vite.config.ts test.coverage", configPath, strings.Join(policy.CoverageReporters, ","), strings.Join(projection.CoverageReporters, ","), ContainsAllStrings(projection.CoverageReporters, policy.CoverageReporters), "Include every policy-declared reporter in coverage.reporter.")
	add("coverage.include", "vite.config.ts test.coverage", configPath, strings.Join(policy.CoverageInclude, ","), strings.Join(projection.CoverageInclude, ","), ContainsAllStrings(projection.CoverageInclude, policy.CoverageInclude), "Set coverage.include so coverage denominators stay scoped to production source files.")
	add("coverage.exclude", "vite.config.ts test.coverage", configPath, strings.Join(policy.CoverageExclude, ","), strings.Join(projection.CoverageExclude, ","), ContainsAllStrings(projection.CoverageExclude, policy.CoverageExclude), "Restore the canonical coverage exclusions.")
	add("coverage.report_on_failure", "vite.config.ts test.coverage", configPath, "true", boolText(projection.HasReportOnFailure && projection.ReportOnFailure), projection.HasReportOnFailure && projection.ReportOnFailure, "Set coverage.reportOnFailure to true.")
	add("eslint.production_import_ban", "eslint.config.js no-restricted-imports", eslintPath, "test-utils and feature mocks banned from production", boolText(projection.HasImportBanRule), projection.HasImportBanRule, "Restore the no-restricted-imports patterns for test helpers and feature mocks.")
	for _, key := range []string{"lines", "functions", "branches", "statements"} {
		value, ok := projection.Thresholds[key]
		add("coverage.threshold."+key, "vite.config.ts test.coverage.thresholds", configPath, formatCoverageFloor(policy.CoverageFloor), thresholdValue(value, ok), ok && value >= policy.CoverageFloor, "Restore the "+key+" threshold to the policy floor or higher.")
	}
	return checks
}

func boolText(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func thresholdValue(value float64, ok bool) string {
	if !ok {
		return ""
	}
	return formatCoverageFloor(value)
}

func formatCoverageFloor(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", value), "0"), ".")
}

func readText(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(raw)
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
