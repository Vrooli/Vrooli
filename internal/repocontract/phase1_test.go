package repocontract

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractcheck"
	testkitgo "github.com/vrooli/vrooli/packages/testkit-go"
)

func TestRepoContractJSONParsesAndMatchesPhase1Semantics(t *testing.T) {
	contract := loadContract(t, repoRoot(t))

	if contract.Schema() != "schemas/repo-contract.schema.json" {
		t.Fatalf("schema = %q", contract.Schema())
	}
	if contract.Version() != "1.0.0" {
		t.Fatalf("version = %q", contract.Version())
	}
	platform := contract.Platform()
	if platform.Mode != "cross_platform_go_native" {
		t.Fatalf("platform.mode = %q", platform.Mode)
	}
	if platform.LegacyProjectBashSupported {
		t.Fatal("legacy_project_bash_supported must be false")
	}

	resource := contract.Resource()
	if resource.Manifest != "resource.json" {
		t.Fatalf("resource.manifest = %q, want resource.json", resource.Manifest)
	}
	if got, want := contract.EnvironmentVariables(), map[string]string{
		"repo_root":      "VROOLI_ROOT",
		"source_root":    "VROOLI_SOURCE_ROOT",
		"sandbox_id":     "VROOLI_SANDBOX_ID",
		"sandbox_merged": "VROOLI_SANDBOX_MERGED",
		"sandbox_scope":  "VROOLI_SANDBOX_SCOPE",
	}; !mapsEqual(got, want) {
		t.Fatalf("environment.variables = %#v, want %#v", got, want)
	}

	globs := contract.Globs()
	if globs.Syntax != "doublestar" {
		t.Fatalf("globs.syntax = %q", globs.Syntax)
	}
	if !globs.RootRelative || !globs.CaseSensitive || globs.AllowAbsolute {
		t.Fatalf("unexpected glob policy: %+v", globs)
	}
	if globs.PathFormat != "slash_normalized" {
		t.Fatalf("globs.path_format = %q", globs.PathFormat)
	}
	if got := contract.SandboxScenarioScopePrefix(); got != "scenarios/" {
		t.Fatalf("sandbox.scenario_scope_prefix = %q", got)
	}

	mini, ok := contract.Profiles()["mini_vrooli_bundle"]
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
	contract := loadContract(t, repoRoot(t))
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
		t.Fatalf("root.markers.required_dirs = %v, want %v", got, want)
	}
	if got, want := rootMarkers.RequiredFiles, []string{"go.mod"}; !slices.Equal(got, want) {
		t.Fatalf("root.markers.required_files = %v, want %v", got, want)
	}

	switch {
	case layout.ProjectConfigDir != ".vrooli":
		t.Fatalf("layout.project_config_dir = %q", layout.ProjectConfigDir)
	case layout.ScenarioDir != "scenarios":
		t.Fatalf("layout.scenario_dir = %q", layout.ScenarioDir)
	case layout.ResourceDir != "resources":
		t.Fatalf("layout.resource_dir = %q", layout.ResourceDir)
	case layout.TemplateDir != "templates":
		t.Fatalf("layout.template_dir = %q", layout.TemplateDir)
	case layout.PackageDir != "packages":
		t.Fatalf("layout.package_dir = %q", layout.PackageDir)
	case layout.CommandDir != "cmd":
		t.Fatalf("layout.command_dir = %q", layout.CommandDir)
	case layout.InternalDir != "internal":
		t.Fatalf("layout.internal_dir = %q", layout.InternalDir)
	case layout.DocsDir != "docs":
		t.Fatalf("layout.docs_dir = %q", layout.DocsDir)
	}

	if got, want := scenario.RequiredFiles, []string{".vrooli/service.json"}; !slices.Equal(got, want) {
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
	if !mapsEqual(scenario.WellKnownPaths, expectedScenarioPaths) {
		t.Fatalf("scenario.well_known_paths = %#v, want %#v", scenario.WellKnownPaths, expectedScenarioPaths)
	}

	expectedResourcePaths := map[string]string{
		"docs":           "docs",
		"initialization": "initialization",
	}
	if resource.Manifest != "resource.json" {
		t.Fatalf("resource.manifest = %q", resource.Manifest)
	}
	if !mapsEqual(resource.WellKnownPaths, expectedResourcePaths) {
		t.Fatalf("resource.well_known_paths = %#v, want %#v", resource.WellKnownPaths, expectedResourcePaths)
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
	contract := loadContract(t, root)
	rootMarkers := contract.RootMarkers()
	layout := contract.Layout()
	scenario := contract.Scenario()
	resource := contract.Resource()

	for _, dir := range rootMarkers.RequiredDirs {
		path := filepath.Join(root, filepath.FromSlash(dir))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("required dir %s missing: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("required dir %s is not a directory", dir)
		}
	}
	for _, file := range rootMarkers.RequiredFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("required file %s missing: %v", file, err)
		}
		if info.IsDir() {
			t.Fatalf("required file %s is a directory", file)
		}
	}

	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(layout.ScenarioDir)), filepath.FromSlash(scenario.WellKnownPaths["service"])); err != nil {
		t.Fatalf("count scenario manifests: %v", err)
	} else if count == 0 {
		t.Fatal("expected at least one scenario manifest matching the repo contract")
	}
	if count, err := manifestCount(filepath.Join(root, filepath.FromSlash(layout.ResourceDir)), filepath.FromSlash(resource.Manifest)); err != nil {
		t.Fatalf("count resource manifests: %v", err)
	} else if count == 0 {
		t.Fatal("expected at least one resource manifest matching the repo contract")
	}
}

func TestRepoContractExcludesLegacyRulesAndPaths(t *testing.T) {
	root := repoRoot(t)
	data := string(mustReadFile(t, filepath.Join(root, ".vrooli", "repo-contract.json")))

	for _, item := range []string{
		".vrooli/resource.json",
		".git\"",
		"pnpm-workspace.yaml",
		"$HOME/Vrooli",
		"APP_ROOT",
		".vrooli/metadata.json",
	} {
		if strings.Contains(data, item) {
			t.Fatalf("repo contract unexpectedly contains legacy or deferred item %q", item)
		}
	}

	for _, path := range collectContractPaths(loadContract(t, root)) {
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
	contract := loadContract(t, repoRoot(t))
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
				t.Fatalf("profile %s contains non-canonical include root %q", profileName, include)
			}
		}
	}

	if got, want := contract.SandboxFullRepoScopes(), []string{"", ".", "/"}; !slices.Equal(got, want) {
		t.Fatalf("sandbox.full_repo_scopes = %v, want %v", got, want)
	}
}

func TestRepoContractBundleProfileStaysWithinPhase1Policy(t *testing.T) {
	profile, ok := loadContract(t, repoRoot(t)).Profiles()["mini_vrooli_bundle"]
	if !ok {
		t.Fatal("missing mini_vrooli_bundle profile")
	}

	if got, want := profile.Parameters, []string{"scenario", "resources[*]"}; !slices.Equal(got, want) {
		t.Fatalf("profile.parameters = %v, want %v", got, want)
	}

	for _, include := range []string{
		".vrooli",
		"cmd",
		"internal",
		"packages",
		"scenarios/{scenario}",
		"resources/{resources[*]}",
	} {
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

func TestRepoContractChecksPackagePasses(t *testing.T) {
	report, err := repocontractcheck.Run(repoRoot(t))
	if err != nil {
		t.Fatalf("repocontractcheck.Run(): %v", err)
	}
	if !report.Success {
		t.Fatalf("repocontractcheck report = %+v", report)
	}
}

func TestRepoContractDocsStayAlignedWithPhase1Contract(t *testing.T) {
	root := repoRoot(t)
	docs := string(mustReadFile(t, filepath.Join(root, "docs", "repo-contract.md")))

	for _, snippet := range []string{
		"`vrooli contract validate`",
		"`vrooli contract show`",
		"`vrooli contract resolve scenario <name> --file service`",
		"`vrooli contract match-glob <pattern> <path>`",
		"make validate-repo-contract",
		"CI/automation entrypoint",
		"`packages/repo-contract-go`",
		"## Landed Consumer Migrations",
		"`swarm-manager`",
	} {
		if !strings.Contains(docs, snippet) {
			t.Fatalf("docs/repo-contract.md missing required snippet %q", snippet)
		}
	}
}

func repoRoot(t *testing.T) string {
	return testkitgo.ProjectRoot(t)
}

func loadContract(t *testing.T, root string) *repocontract.Contract {
	t.Helper()
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		t.Fatalf("LoadDefault(%s): %v", root, err)
	}
	return contract
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
