package hostreq

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqspec"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/testfixture"
	"github.com/vrooli/vrooli/internal/testutil"
)

func TestResolveMergesRootScenarioAndResourceDeclarations(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
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
	testfixture.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{
			{Name: "node", Required: false, Reason: "scenario node", When: []string{"develop"}, Notes: "scenario note"},
			{Name: "ffmpeg", Required: true, Reason: "scenario ffmpeg", Platforms: []string{"linux"}},
		},
	})
	testfixture.WriteResourceManifest(t, root, "alpha-resource", manifestpkg.ResourceManifest{
		Name:            "alpha-resource",
		Driver:          "external-cli",
		Binary:          "alpha",
		PortabilityTier: "full",
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

func TestResolveHonorsSelectorsAndFilters(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()

	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostTools: []hostreqspec.Declaration{
			{Name: "docker", Required: true, Reason: "root docker", Environments: []string{"development", "production"}},
			{Name: "python", Required: true, Reason: "root python", Environments: []string{"development"}},
			{Name: "openbox", Required: true, Reason: "linux only", Platforms: []string{"linux"}},
		},
	})
	testfixture.WriteScenarioService(t, root, "alpha", scenario.ServiceManifest{
		Service:   scenario.ServiceMetadata{Name: "alpha"},
		HostTools: []hostreqspec.Declaration{{Name: "ffmpeg", Required: true, Reason: "alpha ffmpeg"}},
	})
	testfixture.WriteScenarioService(t, root, "beta", scenario.ServiceManifest{
		Service:   scenario.ServiceMetadata{Name: "beta"},
		HostTools: []hostreqspec.Declaration{{Name: "buf", Required: true, Reason: "beta buf"}},
	})
	testfixture.WriteResourceManifest(t, root, "alpha-resource", manifestpkg.ResourceManifest{
		Name:            "alpha-resource",
		Driver:          "external-cli",
		Binary:          "alpha",
		PortabilityTier: "full",
		HostTools:       []hostreqspec.Declaration{{Name: "sqlite", Required: true, Reason: "alpha sqlite"}},
	})
	testfixture.WriteResourceManifest(t, root, "beta-resource", manifestpkg.ResourceManifest{
		Name:            "beta-resource",
		Driver:          "external-cli",
		Binary:          "beta",
		PortabilityTier: "full",
		HostTools:       []hostreqspec.Declaration{{Name: "helm", Required: true, Reason: "beta helm"}},
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
	if findOptionalRequirement(resolution.Tools, "openbox") != nil {
		t.Fatal("openbox should be filtered out for macos")
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

func TestResolveRejectsUnknownExplicitSelections(t *testing.T) {
	root := t.TempDir()
	home := t.TempDir()
	testfixture.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
	})

	if _, err := Resolve(root, home, ResolveOptions{Resources: "missing"}); err == nil || !strings.Contains(err.Error(), `resource "missing" not found`) {
		t.Fatalf("resource error = %v", err)
	}
	if _, err := Resolve(root, home, ResolveOptions{Scenarios: "missing"}); err == nil || !strings.Contains(err.Error(), `load scenario "missing"`) {
		t.Fatalf("scenario error = %v", err)
	}
}

func TestSchemaFilesDeclareHostRequirementProperties(t *testing.T) {
	root := testutil.ProjectRoot(t)

	serviceSchema := testutil.ReadJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "service.schema.json"))
	resourceSchema := testutil.ReadJSONFile(t, filepath.Join(root, ".vrooli", "schemas", "resource.schema.json"))

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
