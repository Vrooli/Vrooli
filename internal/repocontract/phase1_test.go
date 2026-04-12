package repocontract

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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

func TestRepoContractJSONParsesAndMatchesPhase1Semantics(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".vrooli", "repo-contract.json")
	data := mustReadFile(t, path)

	var doc contractDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal repo contract: %v", err)
	}

	if doc.Schema != "schemas/repo-contract.schema.json" {
		t.Fatalf("schema = %q", doc.Schema)
	}
	if doc.Version != "1.0.0" {
		t.Fatalf("version = %q", doc.Version)
	}
	if doc.Platform.Mode != "cross_platform_go_native" {
		t.Fatalf("platform.mode = %q", doc.Platform.Mode)
	}
	if doc.Platform.LegacyProjectBashSupported {
		t.Fatal("legacy_project_bash_supported must be false")
	}
	if got := doc.Resource.Manifest; got != "resource.json" {
		t.Fatalf("resource.manifest = %q, want resource.json", got)
	}
	if got := doc.Environment.Variables["repo_root"]; got != "VROOLI_ROOT" {
		t.Fatalf("repo_root env = %q", got)
	}
	if got := doc.Environment.Variables["source_root"]; got != "VROOLI_SOURCE_ROOT" {
		t.Fatalf("source_root env = %q", got)
	}
	if got, want := doc.Environment.Variables, map[string]string{
		"repo_root":      "VROOLI_ROOT",
		"source_root":    "VROOLI_SOURCE_ROOT",
		"sandbox_id":     "VROOLI_SANDBOX_ID",
		"sandbox_merged": "VROOLI_SANDBOX_MERGED",
		"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
	}; !mapsEqual(got, want) {
		t.Fatalf("environment.variables = %#v, want %#v", got, want)
	}
	if got := doc.Globs.Syntax; got != "doublestar" {
		t.Fatalf("globs.syntax = %q", got)
	}
	if !doc.Globs.RootRelative || !doc.Globs.CaseSensitive || doc.Globs.AllowAbsolute {
		t.Fatalf("unexpected glob policy: %+v", doc.Globs)
	}
	if got := doc.Globs.PathFormat; got != "slash_normalized" {
		t.Fatalf("globs.path_format = %q", got)
	}
	if got := doc.Sandbox.ScenarioScopePrefix; got != "scenarios/" {
		t.Fatalf("sandbox.scenario_scope_prefix = %q", got)
	}

	mini, ok := doc.Profiles["mini_vrooli_bundle"]
	if !ok {
		t.Fatal("missing mini_vrooli_bundle profile")
	}
	if len(mini.Include) == 0 || len(mini.Exclude) == 0 {
		t.Fatal("mini_vrooli_bundle must define include and exclude paths")
	}
	if !contains(mini.Parameters, "scenario") || !contains(mini.Parameters, "resources[*]") {
		t.Fatalf("unexpected profile parameters: %v", mini.Parameters)
	}
}

func TestRepoContractCanonicalMarkersAndPathsStayExact(t *testing.T) {
	doc := loadContract(t, repoRoot(t))

	if got, want := doc.Root.Markers.RequiredDirs, []string{
		".vrooli",
		"scenarios",
		"resources",
		"packages",
		"cmd",
		"internal",
	}; !slices.Equal(got, want) {
		t.Fatalf("root.markers.required_dirs = %v, want %v", got, want)
	}

	if got, want := doc.Root.Markers.RequiredFiles, []string{"go.mod"}; !slices.Equal(got, want) {
		t.Fatalf("root.markers.required_files = %v, want %v", got, want)
	}

	if got := doc.Layout.ProjectConfigDir; got != ".vrooli" {
		t.Fatalf("layout.project_config_dir = %q", got)
	}
	if got := doc.Layout.ScenarioDir; got != "scenarios" {
		t.Fatalf("layout.scenario_dir = %q", got)
	}
	if got := doc.Layout.ResourceDir; got != "resources" {
		t.Fatalf("layout.resource_dir = %q", got)
	}
	if got := doc.Layout.PackageDir; got != "packages" {
		t.Fatalf("layout.package_dir = %q", got)
	}
	if got := doc.Layout.CommandDir; got != "cmd" {
		t.Fatalf("layout.command_dir = %q", got)
	}
	if got := doc.Layout.InternalDir; got != "internal" {
		t.Fatalf("layout.internal_dir = %q", got)
	}
	if got := doc.Layout.DocsDir; got != "docs" {
		t.Fatalf("layout.docs_dir = %q", got)
	}

	if got, want := doc.Scenario.RequiredFiles, []string{".vrooli/service.json"}; !slices.Equal(got, want) {
		t.Fatalf("scenario.required_files = %v, want %v", got, want)
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
	if !mapsEqual(doc.Scenario.WellKnownPaths, expectedScenarioPaths) {
		t.Fatalf("scenario.well_known_paths = %#v, want %#v", doc.Scenario.WellKnownPaths, expectedScenarioPaths)
	}

	expectedResourcePaths := map[string]string{
		"docs":           "docs",
		"initialization": "initialization",
	}
	if doc.Resource.Manifest != "resource.json" {
		t.Fatalf("resource.manifest = %q", doc.Resource.Manifest)
	}
	if !mapsEqual(doc.Resource.WellKnownPaths, expectedResourcePaths) {
		t.Fatalf("resource.well_known_paths = %#v, want %#v", doc.Resource.WellKnownPaths, expectedResourcePaths)
	}
}

func TestRepoContractSchemaParsesAndReferencesSharedSemver(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".vrooli", "schemas", "repo-contract.schema.json")
	data := mustReadFile(t, path)

	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	if got := stringValue(schema["$schema"]); got != "http://json-schema.org/draft-07/schema#" {
		t.Fatalf("$schema = %q", got)
	}
	if got := stringValue(schema["$id"]); got != "https://vrooli.com/schemas/repo-contract.schema.json" {
		t.Fatalf("$id = %q", got)
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties object")
	}
	version, ok := properties["version"].(map[string]any)
	if !ok {
		t.Fatal("schema missing version property")
	}
	if got := stringValue(version["$ref"]); got != "common.schema.json#/definitions/semver" {
		t.Fatalf("version.$ref = %q", got)
	}
}

func TestRepoContractValidatesAgainstSchema(t *testing.T) {
	root := repoRoot(t)
	cmd := exec.Command("python3", filepath.Join(root, ".vrooli", "schemas", "validate-repo-contract.py"))
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("validate repo contract via schema: %v\n%s", err, output)
	}
}

func TestRepoContractMatchesLiveRepoStructure(t *testing.T) {
	root := repoRoot(t)
	doc := loadContract(t, root)

	for _, dir := range doc.Root.Markers.RequiredDirs {
		path := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("required dir %s missing: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("required dir %s is not a directory", dir)
		}
	}

	for _, file := range doc.Root.Markers.RequiredFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("required file %s missing: %v", file, err)
		}
		if info.IsDir() {
			t.Fatalf("required file %s is a directory", file)
		}
	}

	scenarioManifestCount := 0
	scenariosDir := filepath.Join(root, filepath.FromSlash(doc.Layout.ScenarioDir))
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		t.Fatalf("read scenarios dir: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(scenariosDir, entry.Name(), filepath.FromSlash(doc.Scenario.WellKnownPaths["service"]))
		if _, err := os.Stat(manifestPath); err == nil {
			scenarioManifestCount++
		}
	}
	if scenarioManifestCount == 0 {
		t.Fatal("expected at least one scenario manifest matching the repo contract")
	}

	resourceManifestCount := 0
	resourcesDir := filepath.Join(root, filepath.FromSlash(doc.Layout.ResourceDir))
	resourceEntries, err := os.ReadDir(resourcesDir)
	if err != nil {
		t.Fatalf("read resources dir: %v", err)
	}
	for _, entry := range resourceEntries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(resourcesDir, entry.Name(), filepath.FromSlash(doc.Resource.Manifest))
		if _, err := os.Stat(manifestPath); err == nil {
			resourceManifestCount++
		}
	}
	if resourceManifestCount == 0 {
		t.Fatal("expected at least one resource manifest matching the repo contract")
	}
}

func TestRepoContractExcludesLegacyRulesAndPaths(t *testing.T) {
	root := repoRoot(t)
	data := string(mustReadFile(t, filepath.Join(root, ".vrooli", "repo-contract.json")))

	disallowed := []string{
		".vrooli/resource.json",
		".git\"",
		"pnpm-workspace.yaml",
		"$HOME/Vrooli",
		"APP_ROOT",
		".vrooli/metadata.json",
	}
	for _, item := range disallowed {
		if strings.Contains(data, item) {
			t.Fatalf("repo contract unexpectedly contains legacy or deferred item %q", item)
		}
	}

	doc := loadContract(t, root)
	for _, path := range collectContractPaths(doc) {
		if strings.Contains(path, "\\") {
			t.Fatalf("contract path %q must be slash-normalized", path)
		}
		if strings.HasPrefix(path, "/") {
			t.Fatalf("contract path %q must be repo-relative", path)
		}
		if strings.Contains(path, "..") {
			t.Fatalf("contract path %q must not contain parent traversal", path)
		}
	}
}

func TestRepoContractProfileRootsStayWithinCanonicalLayout(t *testing.T) {
	doc := loadContract(t, repoRoot(t))

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
				t.Fatalf("profile %s contains non-canonical include root %q", profileName, include)
			}
		}
	}

	if got, want := doc.Sandbox.FullRepoScopes, []string{"", ".", "/"}; !slices.Equal(got, want) {
		t.Fatalf("sandbox.full_repo_scopes = %v, want %v", got, want)
	}
}

func TestRepoContractBundleProfileStaysWithinPhase1Policy(t *testing.T) {
	doc := loadContract(t, repoRoot(t))
	profile, ok := doc.Profiles["mini_vrooli_bundle"]
	if !ok {
		t.Fatal("missing mini_vrooli_bundle profile")
	}

	if got, want := profile.Parameters, []string{"scenario", "resources[*]"}; !slices.Equal(got, want) {
		t.Fatalf("profile.parameters = %v, want %v", got, want)
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
			t.Fatalf("profile.include missing %q in %v", include, profile.Include)
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
			t.Fatalf("bundle profile unexpectedly treats legacy/transitional root %q as canonical", forbidden)
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
			t.Fatalf("profile.exclude missing %q in %v", exclude, profile.Exclude)
		}
	}
}

func TestRepoContractDocsStayAlignedWithPhase1Contract(t *testing.T) {
	root := repoRoot(t)
	docs := string(mustReadFile(t, filepath.Join(root, "docs", "repo-contract.md")))

	requiredSnippets := []string{
		"Phase 1 implementation status:",
		"make validate-repo-contract",
		"canonical scenario manifest path: `scenarios/<name>/.vrooli/service.json`",
		"canonical resource manifest path: `resources/<name>/resource.json`",
		"`packages/repo-contract-go`",
		"`swarm-manager` backlog glob validation and counting",
	}
	for _, snippet := range requiredSnippets {
		if !strings.Contains(docs, snippet) {
			t.Fatalf("docs/repo-contract.md missing required snippet %q", snippet)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Clean(filepath.Join(dir, "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func loadContract(t *testing.T, root string) contractDoc {
	t.Helper()
	var doc contractDoc
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(root, ".vrooli", "repo-contract.json")), &doc); err != nil {
		t.Fatalf("unmarshal repo contract: %v", err)
	}
	return doc
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
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

func stringValue(value any) string {
	str, _ := value.(string)
	return str
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
