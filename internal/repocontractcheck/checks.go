package repocontractcheck

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
)

type contractDoc struct {
	Schema   string `json:"$schema"`
	Version  string `json:"version"`
	Platform struct {
		Mode                       string `json:"mode"`
		LegacyProjectBashSupported bool   `json:"legacy_project_bash_supported"`
	} `json:"platform"`
	Root struct {
		Markers struct {
			RequiredDirs  []string `json:"required_dirs"`
			RequiredFiles []string `json:"required_files"`
		} `json:"markers"`
	} `json:"root"`
	Layout struct {
		ProjectConfigDir string `json:"project_config_dir"`
		ScenarioDir      string `json:"scenario_dir"`
		ResourceDir      string `json:"resource_dir"`
		TemplateDir      string `json:"template_dir"`
		PackageDir       string `json:"package_dir"`
		CommandDir       string `json:"command_dir"`
		InternalDir      string `json:"internal_dir"`
		DocsDir          string `json:"docs_dir"`
	} `json:"layout"`
	Scenario struct {
		RequiredFiles  []string          `json:"required_files"`
		WellKnownPaths map[string]string `json:"well_known_paths"`
	} `json:"scenario"`
	Resource struct {
		Manifest       string            `json:"manifest"`
		WellKnownPaths map[string]string `json:"well_known_paths"`
	} `json:"resource"`
	Globs struct {
		Syntax        string `json:"syntax"`
		RootRelative  bool   `json:"root_relative"`
		CaseSensitive bool   `json:"case_sensitive"`
		AllowAbsolute bool   `json:"allow_absolute"`
		PathFormat    string `json:"path_format"`
	} `json:"globs"`
	Environment struct {
		Variables map[string]string `json:"variables"`
	} `json:"environment"`
	Sandbox struct {
		FullRepoScopes      []string `json:"full_repo_scopes"`
		ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
	} `json:"sandbox"`
	Profiles map[string]struct {
		Description     string   `json:"description"`
		Parameters      []string `json:"parameters"`
		Include         []string `json:"include"`
		OptionalInclude []string `json:"optional_include"`
		Exclude         []string `json:"exclude"`
	} `json:"profiles"`
}

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

type adoptionExceptionDoc struct {
	Version    string              `json:"version"`
	Exceptions []adoptionException `json:"exceptions"`
}

type adoptionException struct {
	Path   string `json:"path"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
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

	contractPath := filepath.Join(root, ".vrooli", "repo-contract.json")
	data, err := os.ReadFile(contractPath)
	if err != nil {
		return Report{}, fmt.Errorf("read repo contract: %w", err)
	}

	var doc contractDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return Report{}, fmt.Errorf("decode repo contract JSON: %w", err)
	}

	checks := []struct {
		name string
		fn   func(contractDoc, string, string) error
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
		err := check.fn(doc, root, string(data))
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

func checkPhase1Semantics(doc contractDoc, root string, raw string) error {
	if doc.Schema != "schemas/repo-contract.schema.json" {
		return fmt.Errorf("schema = %q", doc.Schema)
	}
	if doc.Version != "1.0.0" {
		return fmt.Errorf("version = %q", doc.Version)
	}
	if doc.Platform.Mode != "cross_platform_go_native" {
		return fmt.Errorf("platform.mode = %q", doc.Platform.Mode)
	}
	if doc.Platform.LegacyProjectBashSupported {
		return fmt.Errorf("legacy_project_bash_supported must be false")
	}
	if doc.Resource.Manifest != "resource.json" {
		return fmt.Errorf("resource.manifest = %q", doc.Resource.Manifest)
	}

	expectedEnv := map[string]string{
		"repo_root":      "VROOLI_ROOT",
		"source_root":    "VROOLI_SOURCE_ROOT",
		"sandbox_id":     "VROOLI_SANDBOX_ID",
		"sandbox_merged": "VROOLI_SANDBOX_MERGED",
		"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
	}
	if !mapsEqual(doc.Environment.Variables, expectedEnv) {
		return fmt.Errorf("environment.variables = %#v, want %#v", doc.Environment.Variables, expectedEnv)
	}
	if doc.Globs.Syntax != "doublestar" {
		return fmt.Errorf("globs.syntax = %q", doc.Globs.Syntax)
	}
	if !doc.Globs.RootRelative || !doc.Globs.CaseSensitive || doc.Globs.AllowAbsolute {
		return fmt.Errorf("unexpected glob policy: %+v", doc.Globs)
	}
	if doc.Globs.PathFormat != "slash_normalized" {
		return fmt.Errorf("globs.path_format = %q", doc.Globs.PathFormat)
	}
	if doc.Sandbox.ScenarioScopePrefix != "scenarios/" {
		return fmt.Errorf("sandbox.scenario_scope_prefix = %q", doc.Sandbox.ScenarioScopePrefix)
	}
	mini, ok := doc.Profiles["mini_vrooli_bundle"]
	if !ok {
		return fmt.Errorf("missing mini_vrooli_bundle profile")
	}
	if len(mini.Include) == 0 || len(mini.Exclude) == 0 {
		return fmt.Errorf("mini_vrooli_bundle must define include and exclude paths")
	}
	if !contains(mini.Parameters, "scenario") || !contains(mini.Parameters, "resources[*]") {
		return fmt.Errorf("unexpected profile parameters: %v", mini.Parameters)
	}
	return nil
}

func checkCanonicalMarkersAndPaths(doc contractDoc, root string, raw string) error {
	if got, want := doc.Root.Markers.RequiredDirs, []string{
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
	if got, want := doc.Root.Markers.RequiredFiles, []string{"go.mod"}; !slices.Equal(got, want) {
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
	case doc.Layout.ProjectConfigDir != ".vrooli":
		return fmt.Errorf("layout.project_config_dir = %q", doc.Layout.ProjectConfigDir)
	case doc.Layout.ScenarioDir != "scenarios":
		return fmt.Errorf("layout.scenario_dir = %q", doc.Layout.ScenarioDir)
	case doc.Layout.ResourceDir != "resources":
		return fmt.Errorf("layout.resource_dir = %q", doc.Layout.ResourceDir)
	case doc.Layout.TemplateDir != "templates":
		return fmt.Errorf("layout.template_dir = %q", doc.Layout.TemplateDir)
	case doc.Layout.PackageDir != "packages":
		return fmt.Errorf("layout.package_dir = %q", doc.Layout.PackageDir)
	case doc.Layout.CommandDir != "cmd":
		return fmt.Errorf("layout.command_dir = %q", doc.Layout.CommandDir)
	case doc.Layout.InternalDir != "internal":
		return fmt.Errorf("layout.internal_dir = %q", doc.Layout.InternalDir)
	case doc.Layout.DocsDir != "docs":
		return fmt.Errorf("layout.docs_dir = %q", doc.Layout.DocsDir)
	case !slices.Equal(doc.Scenario.RequiredFiles, []string{".vrooli/service.json"}):
		return fmt.Errorf("scenario.required_files = %v", doc.Scenario.RequiredFiles)
	case !mapsEqual(doc.Scenario.WellKnownPaths, expectedScenarioPaths):
		return fmt.Errorf("scenario.well_known_paths = %#v", doc.Scenario.WellKnownPaths)
	case doc.Resource.Manifest != "resource.json":
		return fmt.Errorf("resource.manifest = %q", doc.Resource.Manifest)
	case !mapsEqual(doc.Resource.WellKnownPaths, expectedResourcePaths):
		return fmt.Errorf("resource.well_known_paths = %#v", doc.Resource.WellKnownPaths)
	}
	return nil
}

func checkLiveRepoStructure(doc contractDoc, root string, raw string) error {
	for _, dir := range doc.Root.Markers.RequiredDirs {
		path := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required dir %s missing: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("required dir %s is not a directory", dir)
		}
	}
	for _, file := range doc.Root.Markers.RequiredFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required file %s missing: %w", file, err)
		}
		if info.IsDir() {
			return fmt.Errorf("required file %s is a directory", file)
		}
	}

	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(doc.Layout.ScenarioDir)), filepath.FromSlash(doc.Scenario.WellKnownPaths["service"])); err != nil {
		return fmt.Errorf("count scenario manifests: %w", err)
	} else if count == 0 {
		return fmt.Errorf("expected at least one scenario manifest matching the repo contract")
	}

	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(doc.Layout.ResourceDir)), filepath.FromSlash(doc.Resource.Manifest)); err != nil {
		return fmt.Errorf("count resource manifests: %w", err)
	} else if count == 0 {
		return fmt.Errorf("expected at least one resource manifest matching the repo contract")
	}

	return nil
}

func checkExcludedLegacyRulesAndPaths(doc contractDoc, root string, raw string) error {
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
	for _, path := range collectContractPaths(doc) {
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

func checkProfileRootsWithinCanonicalLayout(doc contractDoc, root string, raw string) error {
	allowedPrefixes := []string{
		doc.Layout.ProjectConfigDir,
		doc.Layout.CommandDir,
		doc.Layout.InternalDir,
		doc.Layout.PackageDir,
		doc.Layout.ScenarioDir + "/",
		doc.Layout.ResourceDir + "/",
		doc.Layout.DocsDir,
		"go.mod",
		"go.sum",
		"go.work",
		"go.work.sum",
		"Makefile",
		"README.md",
		"LICENSE",
	}
	for profileName, profile := range doc.Profiles {
		for _, include := range append(append([]string{}, profile.Include...), profile.OptionalInclude...) {
			if !hasAllowedPrefix(include, allowedPrefixes) {
				return fmt.Errorf("profile %s contains non-canonical include root %q", profileName, include)
			}
		}
	}
	if got, want := doc.Sandbox.FullRepoScopes, []string{"", ".", "/"}; !slices.Equal(got, want) {
		return fmt.Errorf("sandbox.full_repo_scopes = %v, want %v", got, want)
	}
	return nil
}

func checkBundleProfilePolicy(doc contractDoc, root string, raw string) error {
	profile, ok := doc.Profiles["mini_vrooli_bundle"]
	if !ok {
		return fmt.Errorf("missing mini_vrooli_bundle profile")
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

func checkDocsAlignment(doc contractDoc, root string, raw string) error {
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

func checkAdoptionRulesAlignment(doc contractDoc, root string, raw string) error {
	if err := checkGuidanceAlignment(root); err != nil {
		return err
	}

	exceptions, err := loadAdoptionExceptions(root)
	if err != nil {
		return err
	}

	violations, err := scanAdoptionViolations(root, exceptions)
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
				"## Grandfathered Debt and Exceptions",
				"`.vrooli/repo-contract-adoption-exceptions.json`",
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

func loadAdoptionExceptions(root string) (map[string]map[string]string, error) {
	path := filepath.Join(root, ".vrooli", "repo-contract-adoption-exceptions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read adoption exceptions: %w", err)
	}

	var doc adoptionExceptionDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("decode adoption exceptions: %w", err)
	}
	if strings.TrimSpace(doc.Version) == "" {
		return nil, fmt.Errorf("adoption exceptions missing version")
	}

	allowed := make(map[string]map[string]string, len(doc.Exceptions))
	for _, item := range doc.Exceptions {
		pathKey := filepath.ToSlash(filepath.Clean(strings.TrimSpace(item.Path)))
		ruleKey := strings.TrimSpace(item.Rule)
		reason := strings.TrimSpace(item.Reason)
		switch {
		case pathKey == "" || pathKey == ".":
			return nil, fmt.Errorf("adoption exception has empty path")
		case ruleKey == "":
			return nil, fmt.Errorf("adoption exception %q missing rule", item.Path)
		case reason == "":
			return nil, fmt.Errorf("adoption exception %q/%q missing reason", item.Path, item.Rule)
		}
		if _, ok := allowed[pathKey]; !ok {
			allowed[pathKey] = map[string]string{}
		}
		allowed[pathKey][ruleKey] = reason
	}
	return allowed, nil
}

func scanAdoptionViolations(root string, exceptions map[string]map[string]string) ([]string, error) {
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
				if isAllowedAdoptionViolation(exceptions, rel, rule.Name) {
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

func isAllowedAdoptionViolation(exceptions map[string]map[string]string, path string, rule string) bool {
	rules, ok := exceptions[path]
	if !ok {
		return false
	}
	_, ok = rules[rule]
	return ok
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

func collectContractPaths(doc contractDoc) []string {
	paths := []string{
		doc.Schema,
		doc.Layout.ProjectConfigDir,
		doc.Layout.ScenarioDir,
		doc.Layout.ResourceDir,
		doc.Layout.PackageDir,
		doc.Layout.CommandDir,
		doc.Layout.InternalDir,
		doc.Layout.DocsDir,
		doc.Resource.Manifest,
		doc.Sandbox.ScenarioScopePrefix,
	}
	paths = append(paths, doc.Root.Markers.RequiredDirs...)
	paths = append(paths, doc.Root.Markers.RequiredFiles...)
	paths = append(paths, doc.Scenario.RequiredFiles...)
	for _, value := range doc.Scenario.WellKnownPaths {
		paths = append(paths, value)
	}
	for _, value := range doc.Resource.WellKnownPaths {
		paths = append(paths, value)
	}
	for _, profile := range doc.Profiles {
		paths = append(paths, profile.Include...)
		paths = append(paths, profile.OptionalInclude...)
		paths = append(paths, profile.Exclude...)
	}
	return paths
}
