package targetpack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"structure-health/internal/rules"

	repocontract "github.com/vrooli/repo-contract-go"
)

// projectContract is deliberately a small read model. The project pack reads
// the repository contract as data; it does not become a second contract
// loader. Keeping this model here also lets the project target report a
// malformed contract as a finding with a location and remediation.
type projectContract struct {
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
		ProjectConfigDir       string   `json:"project_config_dir"`
		ScenarioDir            string   `json:"scenario_dir"`
		ResourceDir            string   `json:"resource_dir"`
		TemplateDir            string   `json:"template_dir"`
		PackageDir             string   `json:"package_dir"`
		CommandDir             string   `json:"command_dir"`
		InternalDir            string   `json:"internal_dir"`
		DocsDir                string   `json:"docs_dir"`
		ProjectConfigAllowlist []string `json:"project_config_allowlist"`
	} `json:"layout"`
	RuntimeHome struct {
		DirName      string `json:"dir_name"`
		EnvOverrides []any  `json:"env_overrides"`
		Entries      map[string]struct {
			Path        string `json:"path"`
			Kind        string `json:"kind"`
			Regenerable bool   `json:"regenerable"`
			Sensitive   bool   `json:"sensitive"`
			Owner       string `json:"owner"`
			Protected   bool   `json:"protected"`
			Cleanup     string `json:"cleanup"`
			Retention   *struct {
				MaxAge        string `json:"max_age"`
				MaxBytes      string `json:"max_bytes"`
				KeepCount     int    `json:"keep_count"`
				ProtectActive bool   `json:"protect_active"`
			} `json:"retention"`
		} `json:"entries"`
		Scoped map[string]string `json:"scoped"`
	} `json:"runtime_home"`
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
	Targets struct {
		Kinds map[string]struct {
			Roots []string `json:"roots"`
		} `json:"kinds"`
	} `json:"targets"`
	Sandbox struct {
		FullRepoScopes      []string `json:"full_repo_scopes"`
		ScenarioScopePrefix string   `json:"scenario_scope_prefix"`
	} `json:"sandbox"`
	Profiles map[string]struct {
		Parameters      []string `json:"parameters"`
		Include         []string `json:"include"`
		OptionalInclude []string `json:"optional_include"`
		Exclude         []string `json:"exclude"`
	} `json:"profiles"`
}

func evaluateProject(root string) []rules.Finding {
	contractPath := filepath.Join(root, ".vrooli", "repo-contract.json")
	raw, err := os.ReadFile(contractPath)
	if err != nil {
		return []rules.Finding{finding("PROJECT_CONTRACT_INVALID", "error", "repository contract is unreadable", ".vrooli/repo-contract.json", "Restore a readable .vrooli/repo-contract.json.")}
	}
	var contract projectContract
	if err := json.Unmarshal(raw, &contract); err != nil {
		return []rules.Finding{finding("PROJECT_CONTRACT_INVALID", "error", "repository contract is invalid JSON", ".vrooli/repo-contract.json", "Fix the repository contract JSON syntax.")}
	}
	var out []rules.Finding
	appendProjectCheck(&out, projectPhase1Semantics(contract), "PROJECT_PHASE1_SEMANTICS", ".vrooli/repo-contract.json", "Restore the phase-1 repository contract semantics.")
	appendProjectCheck(&out, projectCanonicalMarkers(contract), "PROJECT_CANONICAL_LAYOUT", ".vrooli/repo-contract.json", "Restore the canonical repository markers and paths.")
	appendProjectCheck(&out, projectRuntimeHome(contract), "PROJECT_RUNTIME_HOME", ".vrooli/repo-contract.json", "Restore the runtime-home structural authority.")
	appendProjectCheck(&out, projectLiveStructure(root, contract), "PROJECT_LIVE_STRUCTURE", ".", "Restore the required repository directories, files, and manifests.")
	out = append(out, projectConfigSurface(root, contract)...)
	out = append(out, projectWorkspaceSmells(root)...)
	out = append(out, projectClaimResolution(root)...)
	appendProjectCheck(&out, projectExcludedLegacy(contract, string(raw), root), "PROJECT_EXCLUDED_LEGACY", ".vrooli/repo-contract.json", "Remove retired paths and legacy contract entries.")
	appendProjectCheck(&out, projectProfileRoots(contract), "PROJECT_PROFILE_ROOTS", ".vrooli/repo-contract.json", "Keep profile includes inside canonical repository roots.")
	appendProjectCheck(&out, projectBundleProfile(contract), "PROJECT_BUNDLE_PROFILE", ".vrooli/repo-contract.json", "Restore the mini_vrooli_bundle include, exclude, and parameter policy.")
	appendProjectCheck(&out, projectResourceArtifacts(root), "PROJECT_RESOURCE_ARTIFACTS", ".vrooli/schemas/resource-definitions.json", "Regenerate resource schema artifacts and repair missing resource references.")
	out = append(out, projectCredentialDescriptorRules(root, contract)...)
	out = append(out, projectSchemaIDRules(root)...)
	out = append(out, projectCLIManifestSchemaRules(root)...)
	out = append(out, projectScopeVocabularyRules(root)...)
	out = append(out, projectConfigInvariantRules(root, contract)...)
	return out
}

// projectConfigInvariantRules owns repository-wide configuration invariants
// that used to be measured by the retired debt census.
func projectConfigInvariantRules(root string, contract projectContract) []rules.Finding {
	var out []rules.Finding
	for _, kind := range []string{"control-plane", "project"} {
		for _, pattern := range contract.Targets.Kinds[kind].Roots {
			matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
			if err != nil {
				out = append(out, finding("PROJECT_CONFIG_SURFACE", "error", fmt.Sprintf("invalid %s target root pattern: %v", kind, err), ".vrooli/repo-contract.json", "Repair the target root pattern."))
				continue
			}
			for _, match := range matches {
				if _, err := os.Stat(filepath.Join(match, ".vrooli", "testing.json")); os.IsNotExist(err) {
					location, _ := filepath.Rel(root, filepath.Join(match, ".vrooli", "testing.json"))
					out = append(out, finding("PROJECT_CONFIG_SURFACE", "error", fmt.Sprintf("%s target has no testing budget", kind), filepath.ToSlash(location), "Add .vrooli/testing.json with a budget for this target."))
				}
			}
		}
	}

	for _, scenario := range []string{"api-health", "architecture-cartographer", "measures-health", "performance-health"} {
		path := filepath.Join(root, "scenarios", scenario, ".vrooli", "test-genie.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			continue
		}
		var descriptor struct {
			Targets struct {
				Kinds              []string `json:"kinds"`
				NotApplicableKinds []struct {
					Kind string `json:"kind"`
				} `json:"notApplicableKinds"`
			} `json:"targets"`
		}
		if json.Unmarshal(data, &descriptor) != nil || slices.Contains(descriptor.Targets.Kinds, "control-plane") {
			continue
		}
		declaredNotApplicable := false
		for _, item := range descriptor.Targets.NotApplicableKinds {
			declaredNotApplicable = declaredNotApplicable || item.Kind == "control-plane"
		}
		if !declaredNotApplicable {
			out = append(out, finding("PROJECT_CONFIG_SURFACE", "error", "phase provider must declare control-plane coverage", filepath.ToSlash(filepath.Join("scenarios", scenario, ".vrooli", "test-genie.json")), "Declare control-plane in targets.kinds or explicitly document non-applicability."))
		}
	}
	return out
}

func appendProjectCheck(out *[]rules.Finding, err error, code, location, remediation string) {
	if err == nil {
		return
	}
	*out = append(*out, finding(code, "error", err.Error(), location, remediation))
}

func projectPhase1Semantics(c projectContract) error {
	if c.Schema != "schemas/repo-contract.schema.json" || c.Version != "1.2.0" {
		return fmt.Errorf("phase-1 contract schema/version is invalid")
	}
	if c.Platform.Mode != "cross_platform_go_native" || c.Platform.LegacyProjectBashSupported {
		return fmt.Errorf("platform is not cross-platform Go-native")
	}
	wantEnv := map[string]string{"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}
	if !stringMapEqual(c.Environment.Variables, wantEnv) {
		return fmt.Errorf("environment.variables does not match the contract")
	}
	if c.Globs.Syntax != "doublestar" || !c.Globs.RootRelative || !c.Globs.CaseSensitive || c.Globs.AllowAbsolute || c.Globs.PathFormat != "slash_normalized" {
		return fmt.Errorf("glob policy is invalid")
	}
	mini, ok := c.Profiles["mini_vrooli_bundle"]
	if c.Sandbox.ScenarioScopePrefix != "scenarios/" || !ok || !stringSliceEqual(mini.Parameters, []string{"scenario", "resources[*]"}) || len(mini.Include) == 0 || len(mini.Exclude) == 0 {
		return fmt.Errorf("sandbox or bundle parameter policy is invalid")
	}
	return nil
}

func projectCanonicalMarkers(c projectContract) error {
	wantDirs := []string{".vrooli", "templates", "scenarios", "resources", "packages", "cmd", "internal"}
	if !stringSliceEqual(c.Root.Markers.RequiredDirs, wantDirs) || !stringSliceEqual(c.Root.Markers.RequiredFiles, []string{"go.mod"}) {
		return fmt.Errorf("root markers are not canonical")
	}
	wantLayout := []string{c.Layout.ProjectConfigDir, c.Layout.ScenarioDir, c.Layout.ResourceDir, c.Layout.TemplateDir, c.Layout.PackageDir, c.Layout.CommandDir, c.Layout.InternalDir, c.Layout.DocsDir}
	if !stringSliceEqual(wantLayout, []string{".vrooli", "scenarios", "resources", "templates", "packages", "cmd", "internal", "docs"}) {
		return fmt.Errorf("layout directories are not canonical")
	}
	wantScenario := map[string]string{"service": ".vrooli/service.json", "orientation": ".vrooli/orientation.json", "docs": "docs", "docs_manifest": "docs/manifest.json", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "cli_manifest": "cli/manifest.json"}
	wantResource := map[string]string{
		"manifest": "resource.json",
		"readme":   "README.md",
		"docs":     "docs",
		"cli":      "cli",
		"config":   "config",
		"test":     "test",
	}
	if !stringSliceEqual(c.Scenario.RequiredFiles, []string{".vrooli/service.json"}) || !stringMapEqual(c.Scenario.WellKnownPaths, wantScenario) || c.Resource.Manifest != "resource.json" || !stringMapEqual(c.Resource.WellKnownPaths, wantResource) {
		return fmt.Errorf("scenario or resource well-known paths are not canonical")
	}
	return nil
}

func projectRuntimeHome(c projectContract) error {
	if c.RuntimeHome.DirName != ".vrooli" || len(c.RuntimeHome.EnvOverrides) != 0 {
		return fmt.Errorf("runtime_home dir or overrides are invalid")
	}
	want := map[string]homeEntryExpectation{
		"plans":       {"plans", "dir", false, false, postureProtected},
		"state":       {"state", "dir", false, false, postureProtected},
		"config":      {"config", "dir", false, false, postureProtected},
		"data":        {"data", "dir", false, false, postureProtected},
		"runtime_db":  {"state/runtime.db", "file", false, false, postureProtected},
		"secrets":     {"secrets.json", "file", false, true, postureProtected},
		"secrets_enc": {"secrets.enc.json", "file", false, true, postureProtected},
		"bin":         {"bin", "dir", true, false, postureProtected},
		"shims":       {"shims", "dir", true, false, postureSelfManaged},
		"backups":     {"backups", "dir", false, false, postureProtected},
		"cache":       {"cache", "dir", true, false, postureManaged},
		"logs":        {"logs", "dir", true, false, postureManaged},
		"metrics":     {"metrics", "dir", true, false, postureManaged},
		"processes":   {"processes", "dir", true, false, postureManaged},
		"build":       {"build", "dir", true, false, postureManaged},
		"test_runs":   {"test-runs", "dir", true, false, postureManaged},
		"artifacts":   {"artifacts", "dir", true, false, postureManaged},
	}
	if len(c.RuntimeHome.Entries) != len(want) {
		return fmt.Errorf("runtime_home entry count is invalid")
	}
	seen := map[string]string{}
	for key, expected := range want {
		got, ok := c.RuntimeHome.Entries[key]
		if !ok || got.Path != expected.path || got.Kind != expected.kind || got.Regenerable != expected.regenerable || (expected.sensitive && !got.Sensitive) {
			return fmt.Errorf("runtime_home entry %q is invalid", key)
		}
		if err := checkHomeEntryPosture(key, expected.posture, got.Owner, got.Cleanup, got.Protected, got.Retention != nil, retentionProtectsActive(got.Retention)); err != nil {
			return err
		}
		if prior, duplicate := seen[got.Path]; duplicate {
			return fmt.Errorf("runtime_home entries %q and %q share path %q", prior, key, got.Path)
		}
		seen[got.Path] = key
	}
	if !stringMapEqual(c.RuntimeHome.Scoped, map[string]string{
		"scenario_secrets":   "scenarios/{scenario}/secrets.json",
		"project_state":      "state/projects/{project_key}",
		"test_runs_scenario": "test-runs/{scenario}",
	}) {
		return fmt.Errorf("runtime_home scoped paths are invalid")
	}
	return nil
}

func projectLiveStructure(root string, c projectContract) error {
	for _, rel := range c.Root.Markers.RequiredDirs {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || !info.IsDir() {
			return fmt.Errorf("required directory %q is missing", rel)
		}
	}
	for _, rel := range c.Root.Markers.RequiredFiles {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || info.IsDir() {
			return fmt.Errorf("required file %q is missing", rel)
		}
	}
	if countFilesNamed(filepath.Join(root, c.Layout.ScenarioDir), ".vrooli/service.json") == 0 || countFilesNamed(filepath.Join(root, c.Layout.ResourceDir), c.Resource.Manifest) == 0 {
		return fmt.Errorf("required scenario and resource manifests are missing")
	}
	return nil
}

func projectConfigSurface(root string, c projectContract) []rules.Finding {
	dir := filepath.Join(root, c.Layout.ProjectConfigDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []rules.Finding{finding("PROJECT_CONFIG_SURFACE", "error", "read project config dir: "+err.Error(), c.Layout.ProjectConfigDir, "Restore the project config directory.")}
	}
	allowed := make(map[string]bool, len(c.Layout.ProjectConfigAllowlist))
	for _, name := range c.Layout.ProjectConfigAllowlist {
		allowed[name] = true
	}
	var out []rules.Finding
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			out = append(out, finding("PROJECT_CONFIG_SURFACE", "error", fmt.Sprintf("project config contains unapproved entry %q", entry.Name()), filepath.ToSlash(filepath.Join(c.Layout.ProjectConfigDir, entry.Name())), "Remove the entry or add it deliberately to layout.project_config_allowlist."))
		}
	}
	return out
}

func projectExcludedLegacy(c projectContract, raw, root string) error {
	for _, item := range []string{".vrooli/resource.json", ".git\"", "pnpm-workspace.yaml", "$HOME/Vrooli", "APP_ROOT", ".vrooli/metadata.json"} {
		if strings.Contains(raw, item) {
			return fmt.Errorf("contract contains retired item %q", item)
		}
	}
	for _, rel := range []string{"scripts/lib", "scripts/resources"} {
		if info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err == nil && info.IsDir() {
			return fmt.Errorf("retired path %q exists", rel)
		}
	}
	for _, value := range contractPaths(c) {
		if strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || hasParentSegment(value) {
			return fmt.Errorf("contract path %q is not a safe repo-relative path", value)
		}
	}
	return nil
}

func projectProfileRoots(c projectContract) error {
	allowed := []string{c.Layout.ProjectConfigDir, c.Layout.CommandDir, c.Layout.InternalDir, c.Layout.PackageDir, c.Layout.ScenarioDir + "/", c.Layout.ResourceDir + "/", c.Layout.DocsDir, "go.mod", "go.sum", "go.work", "go.work.sum", "Makefile", "README.md", "LICENSE"}
	for name, profile := range c.Profiles {
		for _, include := range append(append([]string{}, profile.Include...), profile.OptionalInclude...) {
			if !hasAllowedPrefix(include, allowed) {
				return fmt.Errorf("profile %s contains non-canonical include root %q", name, include)
			}
		}
	}
	if !stringSliceEqual(c.Sandbox.FullRepoScopes, []string{"", ".", "/"}) {
		return fmt.Errorf("sandbox.full_repo_scopes is not canonical")
	}
	return nil
}

func projectBundleProfile(c projectContract) error {
	profile, ok := c.Profiles["mini_vrooli_bundle"]
	if !ok || !stringSliceEqual(profile.Parameters, []string{"scenario", "resources[*]"}) {
		return fmt.Errorf("mini_vrooli_bundle parameters are invalid")
	}
	for _, include := range []string{".vrooli", "cmd", "internal", "packages", "scenarios/{scenario}", "resources/{resources[*]}"} {
		if !containsString(profile.Include, include) {
			return fmt.Errorf("mini_vrooli_bundle include missing %q", include)
		}
	}
	for _, excluded := range []string{"api", "src", "scripts", "platforms", "assets", "package.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc", ".env-example"} {
		if containsString(profile.Include, excluded) || containsString(profile.OptionalInclude, excluded) {
			return fmt.Errorf("mini_vrooli_bundle includes retired root %q", excluded)
		}
	}
	for _, exclude := range []string{".git/**", "**/.git/**", "**/node_modules/**", "**/coverage/**", "**/data/**", ".vrooli/secrets.json", "**/.vrooli/secrets.json", "cli/**"} {
		if !containsString(profile.Exclude, exclude) {
			return fmt.Errorf("mini_vrooli_bundle exclude missing %q", exclude)
		}
	}
	return nil
}

func projectWorkspaceSmells(root string) []rules.Finding {
	var out []rules.Finding
	add := func(code, title, location, remediation string) {
		out = append(out, finding(code, "error", title, location, remediation))
	}
	if fileExists(filepath.Join(root, "pnpm-lock.yaml")) {
		add("PROJECT_ROOT_PNPM_LOCK", "root pnpm-lock.yaml is not allowed", "pnpm-lock.yaml", "Remove the root pnpm-lock.yaml; scenario UIs own their lockfiles.")
	}
	if fileExists(filepath.Join(root, "go.work.sum")) && !fileExists(filepath.Join(root, "go.work")) {
		add("PROJECT_ORPHAN_GO_WORK_SUM", "go.work.sum has no go.work owner", "go.work.sum", "Remove the orphaned go.work.sum or restore its intentional go.work owner.")
	}
	if fileExists(filepath.Join(root, ".npmrc")) {
		add("PROJECT_ROOT_NPMRC", "root .npmrc leaks package-manager configuration", ".npmrc", "Remove the root .npmrc or move configuration to its owning boundary.")
	}
	workspacePath := filepath.Join(root, "pnpm-workspace.yaml")
	if raw, err := os.ReadFile(workspacePath); err == nil {
		// Comments commonly document the forbidden scenario workspace shape;
		// they must not make a valid packages-only workspace fail the rule.
		lines := strings.Split(string(raw), "\n")
		for i, line := range lines {
			if comment := strings.IndexByte(line, '#'); comment >= 0 {
				lines[i] = line[:comment]
			}
		}
		text := strings.Join(lines, "\n")
		if !strings.Contains(text, "packages:") || !strings.Contains(text, "packages/*") || strings.Contains(text, "scenarios/") || !strings.Contains(text, "autoInstallPeers: false") || !strings.Contains(text, "link-workspace-packages: false") {
			add("PROJECT_PNPM_WORKSPACE_INVALID", "root pnpm workspace settings are invalid", "pnpm-workspace.yaml", "Keep the root workspace scoped to packages/* with isolated package settings.")
		}
	}
	scenariosRoot := filepath.Join(root, "scenarios")
	entries, _ := os.ReadDir(scenariosRoot)
	for _, scenario := range entries {
		if !scenario.IsDir() {
			continue
		}
		uiRoot := filepath.Join(scenariosRoot, scenario.Name(), "ui")
		if !fileExists(filepath.Join(uiRoot, "package.json")) {
			continue
		}
		base := filepath.ToSlash(filepath.Join("scenarios", scenario.Name(), "ui"))
		if !fileExists(filepath.Join(uiRoot, "pnpm-workspace.yaml")) {
			add("SCENARIO_UI_BOUNDARY_MISSING", "scenario UI has no pnpm workspace boundary", base+"/pnpm-workspace.yaml", "Add a ui/pnpm-workspace.yaml boundary file.")
		}
		if !fileExists(filepath.Join(uiRoot, "pnpm-lock.yaml")) {
			add("SCENARIO_UI_LOCKFILE_MISSING", "scenario UI has no pnpm lockfile", base+"/pnpm-lock.yaml", "Generate and commit the UI lockfile through Scenario Dependency Analyzer.")
		}
		if raw, err := os.ReadFile(filepath.Join(uiRoot, "package.json")); err == nil {
			var pkg map[string]any
			if json.Unmarshal(raw, &pkg) == nil {
				if deps, ok := pkg["dependencies"].(map[string]any); ok {
					for name, value := range deps {
						if strings.HasPrefix(stringValue(value), "workspace:") {
							add("SCENARIO_WORKSPACE_DEPENDENCY", "scenario UI uses workspace protocol dependency", base+"/package.json#/dependencies/"+name, "Use an explicit package version or governed local dependency declaration.")
						}
					}
				}
			}
		}
	}
	return out
}

func projectClaimResolution(root string) []rules.Finding {
	claims, err := catalogClaims()
	if err != nil {
		return []rules.Finding{finding("PROJECT_CLAIM_UNRESOLVED", "error", "structure rule catalog is unavailable: "+err.Error(), "scenarios/structure-health/api/internal/rules/catalog.go", "Restore the Structure Health rule catalog before validating enforcement claims.")}
	}
	var out []rules.Finding
	docsRoot := filepath.Join(root, "docs")
	_ = filepath.WalkDir(docsRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return walkErr
		}
		raw, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		for _, token := range claimTokens(string(raw)) {
			if claims[token] {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			out = append(out, finding("PROJECT_CLAIM_UNRESOLVED", "error", fmt.Sprintf("documentation claim %q does not resolve to a catalog claim", token), filepath.ToSlash(rel), "Use a claim:<catalog-claim> marker or add the claim to Structure Health's catalog."))
		}
		return nil
	})
	return out
}

func catalogClaims() (map[string]bool, error) {
	claims := map[string]bool{}
	for _, entry := range rules.Catalog() {
		if entry.Claim != "" {
			claims[entry.Claim] = true
		}
	}
	return claims, nil
}

var claimMarkerPattern = regexp.MustCompile(`claim:([A-Za-z0-9._-]+)`)

func claimTokens(raw string) []string {
	matches := claimMarkerPattern.FindAllStringSubmatch(raw, -1)
	out := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, match := range matches {
		if len(match) < 2 || seen[match[1]] {
			continue
		}
		seen[match[1]] = true
		out = append(out, match[1])
	}
	return out
}

func projectResourceArtifacts(root string) error {
	artifact := filepath.Join(root, ".vrooli", "schemas", "resource-definitions.json")
	if !fileExists(artifact) {
		return fmt.Errorf("resource schema artifact is missing")
	}
	if _, _, ok := loadObject(artifact); !ok {
		return fmt.Errorf("resource schema artifact is invalid JSON")
	}
	return nil
}

// projectCredentialDescriptorRules validates every repository-owned manifest
// with the same generic descriptor walk. A credential declaration is scoped to
// its manifest: the same logical credential may intentionally be declared by
// different consumers, but one manifest cannot declare the same store key
// twice.
func projectCredentialDescriptorRules(root string, c projectContract) []rules.Finding {
	var out []rules.Finding
	check := func(path string) {
		raw, err := os.ReadFile(path)
		if err != nil {
			return
		}
		duplicates, err := repocontract.FindCredentialDescriptorDuplicates(raw)
		if err != nil {
			// The owning manifest rule reports malformed JSON. Keep this check
			// focused on duplicate declarations rather than double-reporting it.
			return
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return
		}
		location := filepath.ToSlash(rel)
		for _, duplicate := range duplicates {
			out = append(out, finding(
				"PROJECT_CREDENTIAL_DESCRIPTOR_DUPLICATE",
				"error",
				fmt.Sprintf("credential %s:%s is declared more than once (first at %s, again at %s)", duplicate.LogicalID, duplicate.Field, duplicate.FirstPath, duplicate.DuplicatePath),
				location+duplicate.DuplicatePath,
				"Remove the duplicate logical_id/field declaration from this manifest; declarations in separate manifests remain independent.",
			))
		}
	}
	scanTargets := func(base, manifestRel string) {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(base)))
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				check(filepath.Join(root, filepath.FromSlash(base), entry.Name(), filepath.FromSlash(manifestRel)))
			}
		}
	}
	scanTargets(c.Layout.ScenarioDir, c.Scenario.WellKnownPaths["service"])
	scanTargets(c.Layout.ResourceDir, c.Resource.Manifest)
	return out
}

func countFilesNamed(root, name string) int {
	count := 0
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		cleanPath := filepath.ToSlash(path)
		if !entry.IsDir() && (cleanPath == filepath.ToSlash(name) || strings.HasSuffix(cleanPath, "/"+filepath.ToSlash(name))) {
			count++
		}
		return nil
	})
	return count
}

func contractPaths(c projectContract) []string {
	paths := append([]string{}, c.Root.Markers.RequiredDirs...)
	paths = append(paths, c.Root.Markers.RequiredFiles...)
	paths = append(paths, c.Schema, c.Layout.ProjectConfigDir, c.Layout.ScenarioDir, c.Layout.ResourceDir, c.Layout.PackageDir, c.Layout.CommandDir, c.Layout.InternalDir, c.Layout.DocsDir, c.Sandbox.ScenarioScopePrefix)
	paths = append(paths, c.Scenario.RequiredFiles...)
	for _, value := range c.Scenario.WellKnownPaths {
		paths = append(paths, value)
	}
	paths = append(paths, c.Resource.Manifest)
	for _, value := range c.Resource.WellKnownPaths {
		paths = append(paths, value)
	}
	for _, profile := range c.Profiles {
		paths = append(paths, profile.Include...)
		paths = append(paths, profile.OptionalInclude...)
		paths = append(paths, profile.Exclude...)
	}
	return paths
}

func hasParentSegment(value string) bool {
	for _, segment := range strings.Split(filepath.ToSlash(value), "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func hasAllowedPrefix(value string, prefixes []string) bool {
	value = filepath.ToSlash(value)
	for _, prefix := range prefixes {
		prefix = filepath.ToSlash(prefix)
		if value == prefix || strings.HasPrefix(value, strings.TrimSuffix(prefix, "/")+"/") {
			return true
		}
	}
	return false
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func stringMapEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range b {
		if a[key] != value {
			return false
		}
	}
	return true
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// homeEntryPosture is who may reclaim a runtime-home entry.
//
// This is deliberately independent of `regenerable`. The contract defines
// regenerable as a backup predicate -- "can be reconstructed, need not be
// backed up" -- and this validator used to read it as a cleanup predicate too,
// requiring every regenerable entry to carry a storage-manager retention
// policy. That conflation is what put a bulk age-and-size budget on ~/.vrooli/bin,
// a shared install root that is regenerable in the backup sense and must still
// never be walked by a reaper. The two questions are now asked separately.
type homeEntryPosture int

const (
	// postureProtected: never a bulk cleanup or retention target. Reclaiming a
	// single artifact, where that is meaningful at all, belongs to a component
	// that can prove the specific artifact is dead.
	postureProtected homeEntryPosture = iota
	// postureManaged: storage-manager reclaims by declared age and size.
	postureManaged
	// postureSelfManaged: the declaring component owns the whole lifecycle and
	// re-asserts its own contents. Not protected -- losing it is survivable --
	// but no age or size rule applies, because nothing here is stale merely by
	// being old.
	postureSelfManaged
)

// homeEntryExpectation is one expected runtime-home entry.
type homeEntryExpectation struct {
	path, kind             string
	regenerable, sensitive bool
	posture                homeEntryPosture
}

func retentionProtectsActive(r *struct {
	MaxAge        string `json:"max_age"`
	MaxBytes      string `json:"max_bytes"`
	KeepCount     int    `json:"keep_count"`
	ProtectActive bool   `json:"protect_active"`
},
) bool {
	return r != nil && r.ProtectActive
}

// checkHomeEntryPosture verifies the declared cleanup authority matches the
// posture the contract is required to express for that entry.
func checkHomeEntryPosture(key string, posture homeEntryPosture, owner, cleanup string, protected, hasRetention, protectsActive bool) error {
	switch posture {
	case postureProtected:
		if !protected || (cleanup != "" && cleanup != "never") || hasRetention {
			return fmt.Errorf("runtime_home entry %q must be protected with cleanup=never and no retention policy", key)
		}
	case postureManaged:
		if protected || owner != "control_plane" || cleanup != "storage_manager" || !hasRetention || !protectsActive {
			return fmt.Errorf("runtime_home entry %q lacks a storage-manager retention policy protecting active entries", key)
		}
	case postureSelfManaged:
		if protected || owner != "control_plane" || cleanup != "never" || hasRetention {
			return fmt.Errorf("runtime_home entry %q must declare cleanup=never with no retention policy; its owner re-asserts it", key)
		}
	default:
		return fmt.Errorf("runtime_home entry %q has an unknown posture", key)
	}
	return nil
}
