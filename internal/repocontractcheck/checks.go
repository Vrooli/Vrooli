package repocontractcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
)

type CheckResult struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type Report struct {
	Root         string        `json:"root"`
	ContractPath string        `json:"contract_path"`
	Success      bool          `json:"success"`
	Checks       []CheckResult `json:"checks"`
}

type adoptionRule struct {
	Name        string
	Description string
	Pattern     *regexp.Regexp
}

func Run(root string) (Report, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Report{}, fmt.Errorf("repo root is required")
	}
	root = filepath.Clean(root)

	contractPath := repocontractmeta.ContractPath(root)
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return Report{}, fmt.Errorf("read repo contract: %w", err)
	}

	contract, err := repocontract.Load(contractPath)
	if err != nil {
		return Report{}, fmt.Errorf("load repo contract: %w", err)
	}

	checks := []struct {
		name string
		fn   func(*repocontract.Contract, string, string) error
	}{
		{name: "phase1_semantics", fn: checkPhase1Semantics},
		{name: "canonical_markers_and_paths", fn: checkCanonicalMarkersAndPaths},
		{name: "live_repo_structure", fn: checkLiveRepoStructure},
		{name: "excluded_legacy_rules_and_paths", fn: checkExcludedLegacyRulesAndPaths},
		{name: "profile_roots_within_canonical_layout", fn: checkProfileRootsWithinCanonicalLayout},
		{name: "bundle_profile_policy", fn: checkBundleProfilePolicy},
		{name: "docs_alignment", fn: checkDocsAlignment},
		{name: "adoption_rules_alignment", fn: checkAdoptionRulesAlignment},
	}

	report := Report{
		Root:         root,
		ContractPath: contractPath,
		Success:      true,
		Checks:       make([]CheckResult, 0, len(checks)),
	}
	for _, check := range checks {
		err := check.fn(contract, root, string(data))
		result := CheckResult{Name: check.name, Passed: err == nil}
		if err != nil {
			report.Success = false
			result.Message = err.Error()
		} else {
			result.Message = "ok"
		}
		report.Checks = append(report.Checks, result)
	}

	return report, nil
}

func checkPhase1Semantics(contract *repocontract.Contract, root string, raw string) error {
	if contract.Schema() != repocontractmeta.ContractSchemaRef {
		return fmt.Errorf("schema = %q", contract.Schema())
	}
	if contract.Version() != repocontractmeta.DefaultContractVersion {
		return fmt.Errorf("version = %q", contract.Version())
	}
	platform := contract.Platform()
	if platform.Mode != "cross_platform_go_native" {
		return fmt.Errorf("platform.mode = %q", platform.Mode)
	}
	if platform.LegacyProjectBashSupported {
		return fmt.Errorf("legacy_project_bash_supported must be false")
	}
	resource := contract.Resource()
	if resource.Manifest != "resource.json" {
		return fmt.Errorf("resource.manifest = %q", resource.Manifest)
	}

	expectedEnv := map[string]string{
		"repo_root":      "VROOLI_ROOT",
		"source_root":    "VROOLI_SOURCE_ROOT",
		"sandbox_id":     "VROOLI_SANDBOX_ID",
		"sandbox_merged": "VROOLI_SANDBOX_MERGED",
		"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
	}
	if env := contract.EnvironmentVariables(); !mapsEqual(env, expectedEnv) {
		return fmt.Errorf("environment.variables = %#v, want %#v", env, expectedEnv)
	}
	globs := contract.Globs()
	if globs.Syntax != "doublestar" {
		return fmt.Errorf("globs.syntax = %q", globs.Syntax)
	}
	if !globs.RootRelative || !globs.CaseSensitive || globs.AllowAbsolute {
		return fmt.Errorf("unexpected glob policy: %+v", globs)
	}
	if globs.PathFormat != "slash_normalized" {
		return fmt.Errorf("globs.path_format = %q", globs.PathFormat)
	}
	if scopePrefix := contract.SandboxScenarioScopePrefix(); scopePrefix != "scenarios/" {
		return fmt.Errorf("sandbox.scenario_scope_prefix = %q", scopePrefix)
	}
	mini, ok := contract.Profiles()[repocontractmeta.MiniBundleProfile]
	if !ok {
		return fmt.Errorf("missing %s profile", repocontractmeta.MiniBundleProfile)
	}
	if len(mini.Include) == 0 || len(mini.Exclude) == 0 {
		return fmt.Errorf("%s must define include and exclude paths", repocontractmeta.MiniBundleProfile)
	}
	if !contains(mini.Parameters, "scenario") || !contains(mini.Parameters, "resources[*]") {
		return fmt.Errorf("unexpected profile parameters: %v", mini.Parameters)
	}
	return nil
}

func checkCanonicalMarkersAndPaths(contract *repocontract.Contract, root string, raw string) error {
	rootMarkers := contract.RootMarkers()
	layout := contract.Layout()
	scenario := contract.Scenario()
	resource := contract.Resource()
	if got, want := rootMarkers.RequiredDirs, []string{
		".vrooli",
		"templates",
		"scenarios",
		"resources",
		"packages",
		"cmd",
		"internal",
	}; !slices.Equal(got, want) {
		return fmt.Errorf("root.markers.required_dirs = %v, want %v", got, want)
	}
	if got, want := rootMarkers.RequiredFiles, []string{"go.mod"}; !slices.Equal(got, want) {
		return fmt.Errorf("root.markers.required_files = %v, want %v", got, want)
	}

	expectedScenarioPaths := map[string]string{
		"service":        ".vrooli/service.json",
		"docs":           "docs",
		"requirements":   "requirements",
		"api":            "api",
		"ui":             "ui",
		"cli":            "cli",
		"initialization": "initialization",
	}
	expectedResourcePaths := map[string]string{
		"docs":           "docs",
		"initialization": "initialization",
	}
	switch {
	case layout.ProjectConfigDir != ".vrooli":
		return fmt.Errorf("layout.project_config_dir = %q", layout.ProjectConfigDir)
	case layout.ScenarioDir != "scenarios":
		return fmt.Errorf("layout.scenario_dir = %q", layout.ScenarioDir)
	case layout.ResourceDir != "resources":
		return fmt.Errorf("layout.resource_dir = %q", layout.ResourceDir)
	case layout.TemplateDir != "templates":
		return fmt.Errorf("layout.template_dir = %q", layout.TemplateDir)
	case layout.PackageDir != "packages":
		return fmt.Errorf("layout.package_dir = %q", layout.PackageDir)
	case layout.CommandDir != "cmd":
		return fmt.Errorf("layout.command_dir = %q", layout.CommandDir)
	case layout.InternalDir != "internal":
		return fmt.Errorf("layout.internal_dir = %q", layout.InternalDir)
	case layout.DocsDir != "docs":
		return fmt.Errorf("layout.docs_dir = %q", layout.DocsDir)
	case !slices.Equal(scenario.RequiredFiles, []string{".vrooli/service.json"}):
		return fmt.Errorf("scenario.required_files = %v", scenario.RequiredFiles)
	case !mapsEqual(scenario.WellKnownPaths, expectedScenarioPaths):
		return fmt.Errorf("scenario.well_known_paths = %#v", scenario.WellKnownPaths)
	case resource.Manifest != "resource.json":
		return fmt.Errorf("resource.manifest = %q", resource.Manifest)
	case !mapsEqual(resource.WellKnownPaths, expectedResourcePaths):
		return fmt.Errorf("resource.well_known_paths = %#v", resource.WellKnownPaths)
	}
	return nil
}

func checkLiveRepoStructure(contract *repocontract.Contract, root string, raw string) error {
	rootMarkers := contract.RootMarkers()
	layout := contract.Layout()
	scenario := contract.Scenario()
	resource := contract.Resource()
	for _, dir := range rootMarkers.RequiredDirs {
		path := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required dir %s missing: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("required dir %s is not a directory", dir)
		}
	}
	for _, file := range rootMarkers.RequiredFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required file %s missing: %w", file, err)
		}
		if info.IsDir() {
			return fmt.Errorf("required file %s is a directory", file)
		}
	}

	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(layout.ScenarioDir)), filepath.FromSlash(scenario.WellKnownPaths["service"])); err != nil {
		return fmt.Errorf("count scenario manifests: %w", err)
	} else if count == 0 {
		return fmt.Errorf("expected at least one scenario manifest matching the repo contract")
	}

	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(layout.ResourceDir)), filepath.FromSlash(resource.Manifest)); err != nil {
		return fmt.Errorf("count resource manifests: %w", err)
	} else if count == 0 {
		return fmt.Errorf("expected at least one resource manifest matching the repo contract")
	}

	return nil
}

func checkExcludedLegacyRulesAndPaths(contract *repocontract.Contract, root string, raw string) error {
	disallowed := []string{
		".vrooli/resource.json",
		".git\"",
		"pnpm-workspace.yaml",
		"$HOME/Vrooli",
		"APP_ROOT",
		".vrooli/metadata.json",
	}
	for _, item := range disallowed {
		if strings.Contains(raw, item) {
			return fmt.Errorf("repo contract unexpectedly contains legacy or deferred item %q", item)
		}
	}
	for _, path := range collectContractPaths(contract) {
		switch {
		case strings.Contains(path, "\\"):
			return fmt.Errorf("contract path %q must be slash-normalized", path)
		case strings.HasPrefix(path, "/"):
			return fmt.Errorf("contract path %q must be repo-relative", path)
		case strings.Contains(path, ".."):
			return fmt.Errorf("contract path %q must not contain parent traversal", path)
		}
	}
	return nil
}

func checkProfileRootsWithinCanonicalLayout(contract *repocontract.Contract, root string, raw string) error {
	layout := contract.Layout()
	allowedPrefixes := []string{
		layout.ProjectConfigDir,
		layout.CommandDir,
		layout.InternalDir,
		layout.PackageDir,
		layout.ScenarioDir + "/",
		layout.ResourceDir + "/",
		layout.DocsDir,
		"go.mod",
		"go.sum",
		"go.work",
		"go.work.sum",
		"Makefile",
		"README.md",
		"LICENSE",
	}
	for profileName, profile := range contract.Profiles() {
		for _, include := range append(append([]string{}, profile.Include...), profile.OptionalInclude...) {
			if !hasAllowedPrefix(include, allowedPrefixes) {
				return fmt.Errorf("profile %s contains non-canonical include root %q", profileName, include)
			}
		}
	}
	if got, want := contract.SandboxFullRepoScopes(), []string{"", ".", "/"}; !slices.Equal(got, want) {
		return fmt.Errorf("sandbox.full_repo_scopes = %v, want %v", got, want)
	}
	return nil
}

func checkBundleProfilePolicy(contract *repocontract.Contract, root string, raw string) error {
	profile, ok := contract.Profiles()[repocontractmeta.MiniBundleProfile]
	if !ok {
		return fmt.Errorf("missing %s profile", repocontractmeta.MiniBundleProfile)
	}
	if got, want := profile.Parameters, []string{"scenario", "resources[*]"}; !slices.Equal(got, want) {
		return fmt.Errorf("profile.parameters = %v, want %v", got, want)
	}
	requiredIncludes := []string{
		".vrooli",
		"cmd",
		"internal",
		"packages",
		"scenarios/{scenario}",
		"resources/{resources[*]}",
	}
	for _, include := range requiredIncludes {
		if !contains(profile.Include, include) {
			return fmt.Errorf("profile.include missing %q in %v", include, profile.Include)
		}
	}
	for _, forbidden := range []string{
		"api",
		"src",
		"scripts",
		"platforms",
		"assets",
		"package.json",
		"pnpm-lock.yaml",
		"pnpm-workspace.yaml",
		".npmrc",
		".env-example",
	} {
		if contains(profile.Include, forbidden) || contains(profile.OptionalInclude, forbidden) {
			return fmt.Errorf("bundle profile unexpectedly treats legacy/transitional root %q as canonical", forbidden)
		}
	}
	for _, exclude := range []string{
		".git/**",
		"**/.git/**",
		"**/node_modules/**",
		"**/coverage/**",
		"**/data/**",
		".vrooli/secrets.json",
		"**/.vrooli/secrets.json",
		"cli/**",
		"scripts/lib/**",
		"scripts/manage.sh",
	} {
		if !contains(profile.Exclude, exclude) {
			return fmt.Errorf("profile.exclude missing %q in %v", exclude, profile.Exclude)
		}
	}
	return nil
}

func checkDocsAlignment(contract *repocontract.Contract, root string, raw string) error {
	docsBytes, err := os.ReadFile(filepath.Join(root, "docs", "repo-contract.md"))
	if err != nil {
		return fmt.Errorf("read docs/repo-contract.md: %w", err)
	}
	docs := string(docsBytes)
	requiredSnippets := []string{
		"`vrooli contract validate`",
		"`vrooli contract show`",
		"`vrooli contract resolve scenario <name> --file service`",
		"`vrooli contract match-glob <pattern> <path>`",
		"`make validate-repo-contract` remains the CI/automation entrypoint",
		"## Landed Consumer Migrations",
		"`swarm-manager`",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(docs, snippet) {
			return fmt.Errorf("docs/repo-contract.md missing required snippet %q", snippet)
		}
	}
	if strings.Contains(docs, "CLI/operator tooling such as `vrooli contract ...`") && strings.Contains(docs, "deferred") {
		return fmt.Errorf("docs/repo-contract.md still marks CLI tooling as deferred")
	}
	return nil
}

func checkAdoptionRulesAlignment(contract *repocontract.Contract, root string, raw string) error {
	if err := checkGuidanceAlignment(root); err != nil {
		return err
	}

	violations, err := scanAdoptionViolations(root)
	if err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}

	sort.Strings(violations)
	return fmt.Errorf("repo-contract adoption violations: %s", strings.Join(violations, "; "))
}

func checkGuidanceAlignment(root string) error {
	required := []struct {
		path     string
		snippets []string
	}{
		{
			path: "docs/repo-contract.md",
			snippets: []string{
				"## Adoption Rules",
				"future repo-aware work",
				"`packages/repo-contract-go` directly",
			},
		},
		{
			path: "docs/CONTRIBUTING.md",
			snippets: []string{
				"**Repo Contract**",
				"Do not add new repo-root heuristics",
				"`make validate-repo-contract`",
				"`vrooli contract show`",
			},
		},
		{
			path: "AGENTS.md",
			snippets: []string{
				"## Repo Contract Adoption",
				"Do not add new independent repo-root detection logic",
				"Do not add new hard-coded canonical scenario path assembly",
				"Run `make validate-repo-contract`",
			},
		},
		{
			path: "scenarios/prompt-manager/store/skills/packs/core/cross-platform-readiness/SKILL.md",
			snippets: []string{
				"repo-contract-backed helpers",
				"`packages/repo-contract-go`",
			},
		},
	}

	for _, file := range required {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.path)))
		if err != nil {
			return fmt.Errorf("read %s: %w", file.path, err)
		}
		text := string(data)
		for _, snippet := range file.snippets {
			if !strings.Contains(text, snippet) {
				return fmt.Errorf("%s missing required snippet %q", file.path, snippet)
			}
		}
	}

	skillBytes, err := os.ReadFile(filepath.Join(root, "scenarios", "prompt-manager", "store", "skills", "packs", "core", "cross-platform-readiness", "SKILL.md"))
	if err != nil {
		return fmt.Errorf("read cross-platform-readiness skill: %w", err)
	}
	if strings.Contains(string(skillBytes), `return filepath.Join(os.Getenv("VROOLI_ROOT"), "scenarios", "my-scenario", "config")`) {
		return fmt.Errorf("cross-platform-readiness skill still teaches direct VROOLI_ROOT/scenarios joins")
	}
	return nil
}

func scanAdoptionViolations(root string) ([]string, error) {
	rules := []adoptionRule{
		{
			Name:        "ad_hoc_repo_root_detector",
			Description: "local repo-root detector instead of shared contract helpers",
			Pattern:     regexp.MustCompile(`(?m)^func (?:\([^)]*\) )?(?:findRepoRoot|getVrooliRoot|FindRepoRoot|DetectVrooliRoot)\(`),
		},
		{
			Name:        "legacy_vrooli_home_fallback",
			Description: "historical $HOME/Vrooli fallback in runtime Go code",
			Pattern:     regexp.MustCompile(`\$HOME/Vrooli|filepath\.Join\([^,\n]+,\s*"Vrooli"(?:,\s*"scenarios")?`),
		},
		{
			Name:        "canonical_service_manifest_join",
			Description: "direct canonical service manifest join instead of shared contract helpers",
			Pattern:     regexp.MustCompile(`filepath\.Join\([^,\n]+,\s*"scenarios",\s*[^,\n]+,\s*"\.vrooli",\s*"service\.json"\)`),
		},
		{
			Name:        "app_root_repo_env",
			Description: "APP_ROOT used as a repo-aware canonical input in Go code",
			Pattern:     regexp.MustCompile(`os\.Getenv\("APP_ROOT"\)`),
		},
		{
			Name:        "git_marker_repo_root",
			Description: "direct .git marker probing used for repo-root detection",
			Pattern:     regexp.MustCompile(`(?s)func (?:\([^)]*\) )?(?:findRepoRoot|FindRepoRoot|resolveRepoRoot|ResolveRepoRoot|detectRepoRoot)\([^)]*\).*?filepath\.Join\([^,\n]+,\s*"\.git"\)`),
		},
		{
			Name:        "pnpm_workspace_repo_root",
			Description: "direct pnpm workspace probing used for repo-root detection",
			Pattern:     regexp.MustCompile(`(?s)func (?:\([^)]*\) )?(?:findRepoRoot|FindRepoRoot|resolveRepoRoot|ResolveRepoRoot|detectRepoRoot)\([^)]*\).*?filepath\.Join\([^,\n]+,\s*"pnpm-workspace\.yaml"\)`),
		},
	}

	var violations []string
	for _, topLevel := range []string{"cmd", "internal", "packages", "scenarios"} {
		base := filepath.Join(root, topLevel)
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" {
					return filepath.SkipDir
				}
				return nil
			}

			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}

			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if shouldSkipAdoptionScan(rel) {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			for _, rule := range rules {
				if !rule.Pattern.MatchString(text) {
					continue
				}
				if rule.Name == "ad_hoc_repo_root_detector" && strings.Contains(text, "repocontract.") {
					continue
				}
				violations = append(violations, fmt.Sprintf("%s (%s)", rel, rule.Name))
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("scan adoption violations: %w", err)
		}
	}
	return violations, nil
}

func shouldSkipAdoptionScan(rel string) bool {
	if strings.HasPrefix(rel, "internal/repocontract/") || strings.HasPrefix(rel, "internal/repocontractcheck/") {
		return true
	}
	if strings.HasPrefix(rel, "packages/repo-contract-go/") {
		return true
	}
	return false
}

func manifestCount(root string, relManifest string) (int, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, entry.Name(), relManifest)); err == nil {
			count++
		}
	}
	return count, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func mapsEqual(got, want map[string]string) bool {
	if len(got) != len(want) {
		return false
	}
	for key, wantValue := range want {
		if gotValue, ok := got[key]; !ok || gotValue != wantValue {
			return false
		}
	}
	return true
}

func hasAllowedPrefix(value string, allowed []string) bool {
	for _, prefix := range allowed {
		if value == prefix || strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func collectContractPaths(contract *repocontract.Contract) []string {
	layout := contract.Layout()
	rootMarkers := contract.RootMarkers()
	scenario := contract.Scenario()
	resource := contract.Resource()
	paths := []string{
		contract.Schema(),
		layout.ProjectConfigDir,
		layout.ScenarioDir,
		layout.ResourceDir,
		layout.PackageDir,
		layout.CommandDir,
		layout.InternalDir,
		layout.DocsDir,
		resource.Manifest,
		contract.SandboxScenarioScopePrefix(),
	}
	paths = append(paths, rootMarkers.RequiredDirs...)
	paths = append(paths, rootMarkers.RequiredFiles...)
	paths = append(paths, scenario.RequiredFiles...)
	for _, value := range scenario.WellKnownPaths {
		paths = append(paths, value)
	}
	for _, value := range resource.WellKnownPaths {
		paths = append(paths, value)
	}
	for _, profile := range contract.Profiles() {
		paths = append(paths, profile.Include...)
		paths = append(paths, profile.OptionalInclude...)
		paths = append(paths, profile.Exclude...)
	}
	return paths
}
