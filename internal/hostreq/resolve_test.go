package hostreq

import (
	"path/filepath"
	"strings"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
)

func TestResolveMergesRootScenarioAndResourceDeclarations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		Dependencies: scenario.Dependencies{
			Resources: map[string]scenario.Dependency{
				"alpha-resource": {Enabled: true},
			},
		},
		HostTools: []hostreqspec.Declaration{
			{Name: "docker", Required: true, Reason: "root docker", When: []string{"setup", "develop"}, Environments: []string{"development"}},
			{Name: "node", Required: true, Reason: "root node", Environments: []string{"development"}},
		},
		HostSafeguards: []hostreqspec.Declaration{
			{Name: "remote_session_protection", Required: false, Reason: "root safeguard", Platforms: []string{"linux"}},
		},
	})
	testscenario.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{
			{Name: "node", Required: false, Reason: "scenario node", When: []string{"develop"}, Notes: "scenario note"},
			{Name: "ffmpeg", Required: true, Reason: "scenario ffmpeg", Platforms: []string{"linux"}},
		},
	})
	testresource.WriteResourceManifest(t, root, "alpha-resource", manifestpkg.ResourceManifest{
		Name:   "alpha-resource",
		Driver: "external-cli",
		Binary: "alpha",
		HostTools: []hostreqspec.Declaration{
			{Name: "docker", Required: false, Reason: "resource docker", When: []string{"develop"}},
			{Name: "sqlite", Required: true, Reason: "resource sqlite", Manual: true},
		},
	})

	resolution, err := Resolve(root, home, ResolveOptions{
		Environment: "development",
		When:        "develop",
		Resources:   "enabled",
		Scenarios:   "all",
		Platform:    "linux",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(resolution.Tools) != 4 {
		t.Fatalf("tool count = %d, want 4", len(resolution.Tools))
	}

	docker := findRequirement(t, resolution.Tools, "docker")
	if !docker.Required {
		t.Fatal("docker should remain required after merge")
	}
	if len(docker.Provenance) != 2 {
		t.Fatalf("docker provenance count = %d, want 2", len(docker.Provenance))
	}
	if got := strings.Join(docker.Reasons, ","); got != "resource docker,root docker" {
		t.Fatalf("docker reasons = %q", got)
	}

	node := findRequirement(t, resolution.Tools, "node")
	if got := strings.Join(node.When, ","); got != "develop" {
		t.Fatalf("node when = %q", got)
	}
	if got := strings.Join(node.Notes, ","); got != "scenario note" {
		t.Fatalf("node notes = %q", got)
	}

	sqlite := findRequirement(t, resolution.Tools, "sqlite")
	if !sqlite.Manual {
		t.Fatal("sqlite manual should be true")
	}

	safeguard := findRequirement(t, resolution.Safeguards, "remote_session_protection")
	if safeguard.Kind != KindSafeguard {
		t.Fatalf("safeguard kind = %q", safeguard.Kind)
	}
}

func TestResolveKeepsRootAutohealProtectionWhenScenarioSelectionIsNone(t *testing.T) {
	root, home := t.TempDir(), t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service:        scenario.ServiceMetadata{Name: "vrooli"},
		HostSafeguards: []hostreqspec.Declaration{{Name: "autoheal_watchdog", Required: true, Reason: "protect autoheal across reboot", When: []string{"setup"}}},
	})
	resolution, err := Resolve(root, home, ResolveOptions{Environment: "development", When: "setup", Resources: "none", Scenarios: "none", Platform: "linux"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	item := findRequirement(t, resolution.Safeguards, "autoheal_watchdog")
	if !item.Required {
		t.Fatal("root autoheal watchdog must remain required")
	}
	if len(item.Provenance) != 1 || item.Provenance[0].Kind != "root" {
		t.Fatalf("provenance=%+v, want root-only", item.Provenance)
	}
}

func TestResolveHonorsSelectorsAndFilters(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostTools: []hostreqspec.Declaration{
			{Name: "docker", Required: true, Reason: "root docker", Environments: []string{"development", "production"}},
			{Name: "python", Required: true, Reason: "root python", Environments: []string{"development"}},
			{Name: "openbox", Required: true, Reason: "linux only", Platforms: []string{"linux"}},
		},
	})
	testscenario.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service:   scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{{Name: "ffmpeg", Required: true, Reason: "alpha ffmpeg"}},
	})
	testscenario.WriteScenarioService(t, root, "beta", scenario.ServiceManifest{
		Service:   scenario.ServiceMetadata{Name: "beta"},
		HostTools: []hostreqspec.Declaration{{Name: "buf", Required: true, Reason: "beta buf"}},
	})
	testresource.WriteResourceManifest(t, root, "alpha-resource", manifestpkg.ResourceManifest{
		Name:      "alpha-resource",
		Driver:    "external-cli",
		Binary:    "alpha",
		HostTools: []hostreqspec.Declaration{{Name: "sqlite", Required: true, Reason: "alpha sqlite"}},
	})
	testresource.WriteResourceManifest(t, root, "beta-resource", manifestpkg.ResourceManifest{
		Name:      "beta-resource",
		Driver:    "external-cli",
		Binary:    "beta",
		HostTools: []hostreqspec.Declaration{{Name: "helm", Required: true, Reason: "beta helm"}},
	})

	resolution, err := Resolve(root, home, ResolveOptions{
		Environment: "production",
		Resources:   "beta-resource",
		Scenarios:   "beta",
		Platform:    "macos",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if findOptionalRequirement(resolution.Tools, "python") != nil {
		t.Fatal("python should be filtered out for production")
	}
	if findOptionalRequirement(resolution.Tools, "openbox") == nil {
		t.Fatal("openbox should remain visible for a macOS NotApplicable result")
	}
	if findOptionalRequirement(resolution.Tools, "ffmpeg") != nil {
		t.Fatal("alpha scenario should not be included")
	}
	if findOptionalRequirement(resolution.Tools, "sqlite") != nil {
		t.Fatal("alpha resource should not be included")
	}
	if findOptionalRequirement(resolution.Tools, "helm") == nil {
		t.Fatal("beta resource should be included")
	}
	if findOptionalRequirement(resolution.Tools, "buf") == nil {
		t.Fatal("beta scenario should be included")
	}
}

func TestResolveNormalizesLegacyDarwinPlatformToken(t *testing.T) {
	root := t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostSafeguards: []hostreqspec.Declaration{
			{Name: "vrooli_launcher", Required: false, Reason: "legacy launcher", Platforms: []string{"darwin"}},
			{Name: "not-on-macos", Required: false, Reason: "unknown platform", Platforms: []string{"windoze"}},
		},
	})

	resolution, err := Resolve(root, t.TempDir(), ResolveOptions{Platform: " macOS "})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if findOptionalRequirement(resolution.Safeguards, "vrooli_launcher") == nil {
		t.Fatal("legacy darwin declaration should resolve on macos")
	}
	if findOptionalRequirement(resolution.Safeguards, "not-on-macos") == nil {
		t.Fatal("unknown platform declaration should remain visible for a NotApplicable result")
	}
}

func TestSafeguardManifestOwnsPlatformGate(t *testing.T) {
	state := resolverState{
		platform: "macos",
		catalog: requirementCatalog{safeguards: map[string]hostreqkit.SafeguardManifest{
			"linux_only": {Name: "linux_only", Platforms: []string{"linux"}},
			"macos_ok":   {Name: "macos_ok", Platforms: []string{"macos"}},
		}},
	}
	if !state.matches(Declaration{Name: "linux_only"}, KindSafeguard) {
		t.Fatal("linux-only safeguard should remain visible so runtime can report NotApplicable")
	}
	if !state.matches(Declaration{Name: "macos_ok"}, KindSafeguard) {
		t.Fatal("macOS safeguard was rejected despite its manifest platform")
	}
}

func TestResolvePreservesPlatformMismatchForNotApplicableReporting(t *testing.T) {
	root := t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service:        scenario.ServiceMetadata{Name: "vrooli"},
		HostSafeguards: []hostreqspec.Declaration{{Name: "remote_session_protection", Required: false, Reason: "protect remote sessions"}},
	})

	resolution, err := Resolve(root, t.TempDir(), ResolveOptions{Platform: "macos", Resources: "none"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	item := findOptionalRequirement(resolution.Safeguards, "remote_session_protection")
	if item == nil {
		t.Fatal("platform-mismatched safeguard disappeared from resolution")
	}
	if !containsPlatform(item.Platforms, "linux") {
		t.Fatalf("Platforms = %v, want linux manifest platform", item.Platforms)
	}
}

func TestResolveRejectsUnknownExplicitSelections(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
	})

	if _, err := Resolve(root, home, ResolveOptions{Resources: "missing"}); err == nil || !strings.Contains(err.Error(), `resource "missing" not found`) {
		t.Fatalf("resource error = %v", err)
	}
	if _, err := Resolve(root, home, ResolveOptions{Scenarios: "missing"}); err == nil || !strings.Contains(err.Error(), `load scenario "missing"`) {
		t.Fatalf("scenario error = %v", err)
	}
}

func TestResolveCarriesCapabilityRequires(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	gpu := true
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostTools: []hostreqspec.Declaration{
			{
				Name:     "sd",
				Required: false,
				Reason:   "gpu backend",
				Requires: &hostreqspec.CapabilityRequirement{GPU: &gpu, MinVRAMGb: 6},
			},
		},
	})

	resolution, err := Resolve(root, home, ResolveOptions{Environment: "development", Platform: "linux"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	sd := findRequirement(t, resolution.Tools, "sd")
	if sd.Requires == nil || sd.Requires.GPU == nil || !*sd.Requires.GPU {
		t.Fatalf("capability requires not carried through: %+v", sd.Requires)
	}
	if sd.Requires.MinVRAMGb != 6 {
		t.Fatalf("min vram = %v, want 6", sd.Requires.MinVRAMGb)
	}
}

func TestResolvePromotesRegisteredResourceTargetRequirements(t *testing.T) {
	root := testkitgo.ProjectRoot(t)
	resolution, err := Resolve(root, t.TempDir(), ResolveOptions{Resources: "vault", Platform: "linux"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	item := findRequirement(t, resolution.Tools, "secret-tool")
	if !item.Required || len(item.Provenance) < 1 || item.Provenance[0].Kind != "resource" || !strings.Contains(strings.Join(item.Reasons, " "), "desktop target requires secret-tool") {
		t.Fatalf("target requirement was not promoted with resource provenance: %#v", item)
	}
}

func TestSchemaFilesDeclareHostRequirementProperties(t *testing.T) {
	root := testkitgo.ProjectRoot(t)

	serviceSchema := testkitgo.ReadJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "service.schema.json"))
	resourceSchema := testkitgo.ReadJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "resource.schema.json"))

	assertSchemaHasHostRequirements(t, serviceSchema)
	assertSchemaHasHostRequirements(t, resourceSchema)
}

func assertSchemaHasHostRequirements(t *testing.T, schema map[string]any) {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema properties missing")
	}
	for _, field := range []string{"hostTools", "hostSafeguards"} {
		value, ok := properties[field].(map[string]any)
		if !ok {
			t.Fatalf("%s property missing", field)
		}
		if value["type"] != "array" {
			t.Fatalf("%s type = %v", field, value["type"])
		}
	}
	definitions, ok := schema["definitions"].(map[string]any)
	if !ok {
		t.Fatal("schema definitions missing")
	}
	hostRequirement, ok := definitions["hostRequirement"].(map[string]any)
	if !ok {
		t.Fatal("hostRequirement definition missing")
	}
	required, ok := hostRequirement["required"].([]any)
	if !ok {
		t.Fatal("hostRequirement required missing")
	}
	got := make([]string, 0, len(required))
	for _, value := range required {
		got = append(got, value.(string))
	}
	if strings.Join(got, ",") != "name,required,reason" {
		t.Fatalf("hostRequirement required = %q", strings.Join(got, ","))
	}
}

func findRequirement(t *testing.T, items []ResolvedRequirement, name string) ResolvedRequirement {
	t.Helper()
	item := findOptionalRequirement(items, name)
	if item == nil {
		t.Fatalf("requirement %q not found", name)
	}
	return *item
}

func findOptionalRequirement(items []ResolvedRequirement, name string) *ResolvedRequirement {
	for i := range items {
		if items[i].Name == name {
			return &items[i]
		}
	}
	return nil
}
