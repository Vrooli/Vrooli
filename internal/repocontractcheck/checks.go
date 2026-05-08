package repocontractcheck

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/resources"
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
		{name: "project_config_surface", fn: checkProjectConfigSurface},
		{name: "excluded_legacy_rules_and_paths", fn: checkExcludedLegacyRulesAndPaths},
		{name: "profile_roots_within_canonical_layout", fn: checkProfileRootsWithinCanonicalLayout},
		{name: "bundle_profile_policy", fn: checkBundleProfilePolicy},
		{name: "docs_alignment", fn: checkDocsAlignment},
		{name: "personal_absolute_paths", fn: checkNoPersonalAbsolutePaths},
		{name: "adoption_rules_alignment", fn: checkAdoptionRulesAlignment},
		{name: "resource_schema_artifacts", fn: checkResourceSchemaArtifacts},
		{name: "ollama_gateway_only", fn: checkOllamaGatewayOnly},
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
		"orientation":    ".vrooli/orientation.json",
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

func checkProjectConfigSurface(contract *repocontract.Contract, root string, raw string) error {
	projectConfigDir := filepath.Join(root, filepath.FromSlash(contract.Layout().ProjectConfigDir))
	entries, err := os.ReadDir(projectConfigDir)
	if err != nil {
		return fmt.Errorf("read project config dir: %w", err)
	}
	allowed := map[string]struct{}{
		"build":              {},
		"repo-contract.json": {},
		"resources":          {},
		"schemas":            {},
		"service.json":       {},
	}
	for _, entry := range entries {
		if _, ok := allowed[entry.Name()]; ok {
			continue
		}
		return fmt.Errorf("project config dir contains unapproved entry %q; keep only repo metadata plus local build output in .vrooli/", entry.Name())
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

func checkResourceSchemaArtifacts(contract *repocontract.Contract, root string, raw string) error {
	report, err := resources.ValidateSchemaArtifacts(root)
	if err != nil {
		return err
	}
	if report.Passed {
		return nil
	}
	parts := make([]string, 0, len(report.ArtifactIssues)+len(report.MissingReferences))
	for _, issue := range report.ArtifactIssues {
		message := issue.Message
		if strings.TrimSpace(issue.Path) != "" {
			message = fmt.Sprintf("%s: %s", issue.Path, issue.Message)
		}
		parts = append(parts, message)
	}
	for _, item := range report.MissingReferences {
		parts = append(parts, fmt.Sprintf("%s references missing resource %s", item.Scenario, item.Resource))
	}
	sort.Strings(parts)
	return errors.New(strings.Join(parts, "; "))
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
		"## Allowed `.vrooli/` Surface",
		"`~/.vrooli/secrets.json`",
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

// checkOllamaGatewayOnly enforces that scenarios reach Ollama through
// `resource-ollama gateway ...`, never via raw HTTP. The CLI fronts the daemon
// with a host-wide cross-process semaphore; any code that constructs
// /api/embeddings, /api/generate, or /api/chat directly bypasses that
// throttle and re-introduces the OOM/cascade risk this contract guards.
//
// Initial scope is the two scenarios that have been migrated. Additional
// scenarios are expected to fold in as their migrations land.
func checkOllamaGatewayOnly(contract *repocontract.Contract, root string, raw string) error {
	bannedPaths := []string{"/api/embeddings", "/api/generate", "/api/chat"}
	scopes := []string{
		"scenarios/swarm-manager/api",
		"scenarios/prompt-manager/api",
	}

	var violations []string
	for _, scope := range scopes {
		base := filepath.Join(root, filepath.FromSlash(scope))
		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == "node_modules" || name == "vendor" || name == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			text := string(data)
			for _, banned := range bannedPaths {
				literal := "\"" + banned + "\""
				if strings.Contains(text, literal) {
					violations = append(violations, fmt.Sprintf("%s contains banned literal %q (use resource-ollama gateway)", rel, banned))
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan ollama gateway compliance under %s: %w", scope, err)
		}
	}
	if len(violations) == 0 {
		return nil
	}
	sort.Strings(violations)
	return fmt.Errorf("ollama-gateway-only violations: %s", strings.Join(violations, "; "))
}

func checkAdoptionRulesAlignment(contract *repocontract.Contract, root string, raw string) error {
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

func checkNoPersonalAbsolutePaths(contract *repocontract.Contract, root string, raw string) error {
	var violations []string
	for _, topLevel := range []string{"cmd", "internal", "packages", "resources", "scenarios", "templates", ".vrooli"} {
		base := filepath.Join(root, filepath.FromSlash(topLevel))
		if _, err := os.Stat(base); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("stat personal path scan root %s: %w", topLevel, err)
		}

		err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if shouldSkipPersonalPathScan(rel, true) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			generatedStateTarget := isSwarmManagerGeneratedStatePersonalPathTarget(rel)
			promptTarget := isPersonalPathPromptMarkdown(rel)
			if !generatedStateTarget && !promptTarget && (shouldSkipPersonalPathScan(rel, false) || !isPersonalPathScannableFile(rel) || isPersonalPathAllowed(rel)) {
				return nil
			}
			if (generatedStateTarget || promptTarget) && isPersonalPathAllowed(rel) {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if isBinary(data) {
				return nil
			}
			for lineNo, line := range strings.Split(string(data), "\n") {
				allowFixtureUsers := !generatedStateTarget && !promptTarget
				if containsPersonalHomePath(line, allowFixtureUsers) || containsOperatorIdentity(line, generatedStateTarget || promptTarget) {
					violations = append(violations, fmt.Sprintf("%s:%d", rel, lineNo+1))
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("scan personal absolute paths: %w", err)
		}
	}

	if len(violations) == 0 {
		return nil
	}
	sort.Strings(violations)
	return fmt.Errorf("personal absolute paths found: %s", strings.Join(violations, "; "))
}

var (
	personalHomePathPattern = regexp.MustCompile("(?:^|[^A-Za-z0-9._-])((?:/home|/Users)/([A-Za-z0-9._-]+)(?:/|[[:space:]\"']|$))")
	operatorIdentityPattern = regexp.MustCompile(`(?i)\b(?:matthalloran8|matt(?:hew)?[[:space:]_-]*halloran)\b`)
)

func containsNonFixturePersonalHomePath(line string) bool {
	return containsPersonalHomePath(line, true)
}

func containsPersonalHomePath(line string, allowFixtureUsers bool) bool {
	for _, match := range personalHomePathPattern.FindAllStringSubmatch(line, -1) {
		if len(match) < 3 {
			continue
		}
		if !allowFixtureUsers || !isFixtureHomeUser(match[2]) {
			return true
		}
	}
	return false
}

func containsOperatorIdentity(line string, enabled bool) bool {
	return enabled && operatorIdentityPattern.MatchString(line)
}

func isFixtureHomeUser(username string) bool {
	switch strings.ToLower(username) {
	case ".", "..", "...", "alice", "bob", "charlie", "claude", "example", "me", "other", "sage", "test", "tester", "testuser", "u", "user", "username", "x", "you":
		return true
	default:
		return false
	}
}

func isPersonalPathPromptMarkdown(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.EqualFold(filepath.Ext(rel), ".md") {
		return false
	}
	return strings.Contains(rel, "/prompts/") ||
		strings.HasPrefix(rel, "scenarios/prompt-manager/store/skills/") ||
		strings.HasPrefix(rel, "templates/")
}

func shouldSkipPersonalPathScan(rel string, isDir bool) bool {
	rel = filepath.ToSlash(rel)
	base := pathBase(rel)
	if isDir {
		if isSwarmManagerGeneratedStateDir(rel) {
			return false
		}
		switch base {
		case ".git", "node_modules", ".venv", "vendor", "dist", "build", "coverage", ".cache", ".gocache", ".nyc_output", ".claude", "tmp", "temp", "logs", "data", "docs", "investigations", "report", ".swarm", "review", "evidence", "captures", "handoff":
			return true
		}
		if strings.HasSuffix(rel, "/test/artifacts") {
			return true
		}
	}
	if strings.HasSuffix(rel, ".log") || strings.HasSuffix(rel, "/acceptance-validation.json") {
		return true
	}
	if base == "test_output.txt" {
		return true
	}
	if strings.EqualFold(filepath.Ext(rel), ".md") {
		return true
	}
	return false
}

func isSwarmManagerGeneratedStateDir(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "scenarios/swarm-manager/") {
		return false
	}
	for _, marker := range []string{
		"/.swarm",
		"/workshop",
		"/review",
		"/captures",
		"/evidence",
		"/handoff",
		"/feedback",
		"/operating-mode",
	} {
		if strings.HasSuffix(rel, marker) || strings.Contains(rel, marker+"/") {
			return true
		}
	}
	return false
}

func isSwarmManagerGeneratedStatePersonalPathTarget(rel string) bool {
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, "scenarios/swarm-manager/") {
		return false
	}
	base := pathBase(rel)
	switch {
	case strings.HasSuffix(rel, "/.swarm/last-research-prompt-trace.json"):
		return true
	case base == "acceptance-validation.json":
		return true
	case strings.Contains(rel, "/workshop/") && strings.HasPrefix(base, "round-") && strings.HasSuffix(base, ".json"):
		return true
	case strings.Contains(rel, "/review/") && strings.HasPrefix(base, "round-") && strings.HasSuffix(base, ".json"):
		return true
	case strings.Contains(rel, "/review/captures/"):
		return true
	case strings.Contains(rel, "/review/decisions/") && strings.HasSuffix(base, ".json"):
		return true
	case strings.Contains(rel, "/evidence/"):
		return true
	case strings.Contains(rel, "/handoff/") && (base == "manifest.json" || base == "source-index.json" || base == "brief.md"):
		return true
	case strings.Contains(rel, "/feedback/") && base == "feedback.json":
		return true
	case strings.Contains(rel, "/operating-mode/") && strings.HasPrefix(base, "round-") && strings.HasSuffix(base, ".json"):
		return true
	default:
		return false
	}
}

func isPersonalPathScannableFile(rel string) bool {
	if strings.HasPrefix(pathBase(rel), ".env") {
		return false
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go", ".sh", ".bash", ".js", ".cjs", ".mjs", ".ts", ".tsx", ".json", ".jsonl", ".yaml", ".yml", ".toml":
		return true
	default:
		return false
	}
}

func pathBase(rel string) string {
	rel = strings.TrimSuffix(filepath.ToSlash(rel), "/")
	if rel == "" {
		return ""
	}
	if idx := strings.LastIndex(rel, "/"); idx >= 0 {
		return rel[idx+1:]
	}
	return rel
}

func isPersonalPathAllowed(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == "scenarios/code-smell/initialization/rules/vrooli-specific.yaml" {
		return true
	}
	if strings.HasPrefix(rel, "internal/repocontractcheck/") {
		return true
	}
	return false
}

func isBinary(data []byte) bool {
	return slices.Contains(data, 0)
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
