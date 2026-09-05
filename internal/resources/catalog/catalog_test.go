package catalog

import (
	"path/filepath"
	"testing"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
	testresource "github.com/vrooli/vrooli/internal/resources/resourcestest"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/internal/scenario/scenariotest"
	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func TestDiscoverReportContinuesAfterInvalidResourceManifest(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "redis", true)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "broken", true)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "redis", shelltest.BashShebang()+"exit 0\n")
	testresource.WriteMalformedResourceManifest(t, fixture.Root, "broken", `{"name":"broken","driver":`)

	report, err := New(fixture.Root).DiscoverReport(DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverReport: %v", err)
	}
	if len(report.Items) != 1 || report.Items[0].Name != "redis" {
		t.Fatalf("items = %#v", report.Items)
	}
	if len(report.Failures) != 1 {
		t.Fatalf("failures = %#v", report.Failures)
	}
	if report.Failures[0].Name != "broken" {
		t.Fatalf("failure = %#v", report.Failures[0])
	}
	if report.Failures[0].Path != filepath.Join(fixture.Root, "resources", "broken", "resource.json") {
		t.Fatalf("path = %q", report.Failures[0].Path)
	}
}

func TestDiscoverOneDoesNotDependOnGlobalResourceDiscovery(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "redis", true)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "broken", true)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "redis", shelltest.BashShebang()+"exit 0\n")
	testresource.WriteMalformedResourceManifest(t, fixture.Root, "broken", `{"name":"broken","driver":`)

	item, err := New(fixture.Root).DiscoverOne("redis", DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverOne(redis): %v", err)
	}
	if item == nil || item.Name != "redis" {
		t.Fatalf("item = %#v", item)
	}
}

func TestOperatorStateOverridesProjectResourceEnabledDefault(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "redis", true)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "redis", shelltest.BashShebang()+"exit 0\n")
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, ".vrooli", "operator-state.json"), `{
  "$schema": ".vrooli/schemas/operator-state.schema.json",
  "version": "1.0.0",
  "updated_at": "2026-08-12T00:00:00Z",
  "resources": {"redis": {"enabled": false}}
}`)

	item, err := New(fixture.Root).DiscoverOne("redis", DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverOne(redis): %v", err)
	}
	if item == nil {
		t.Fatal("DiscoverOne(redis) returned nil")
	}
	if item.Enabled {
		t.Fatalf("resource remained enabled despite operator-state override: %#v", item)
	}
	if item.Config.Enabled {
		t.Fatalf("config remained enabled despite operator-state override: %#v", item.Config)
	}
}

func TestDiscoverReportsScenarioConsumerForOperatorDisabledResource(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testscenario.WriteProjectResourceConfig(t, fixture.Root, "sherpa-onnx", false)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, ".vrooli", "operator-state.json"), `{"version":"1.0.0","resources":{"sherpa-onnx":{"enabled":false}}}`)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "sherpa-onnx", shelltest.BashShebang()+"exit 0\n")
	testscenario.WriteScenarioService(t, fixture.Root, "audio-tools", testscenario.ScenarioServiceManifest(
		"audio-tools",
		testscenario.WithDependencies(scenario.Dependencies{Resources: map[string]scenario.Dependency{
			"sherpa-onnx": {Enabled: true, StartupPolicy: scenario.DependencyStartupPolicyTryStart},
		}}),
	))

	item, err := New(fixture.Root).DiscoverOne("sherpa-onnx", DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverOne: %v", err)
	}
	if item == nil || item.Enabled {
		t.Fatalf("item = %#v, want disabled", item)
	}
	if len(item.DisabledDependencyConsumers) != 1 || item.DisabledDependencyConsumers[0].Scenario != "audio-tools" || item.DisabledDependencyConsumers[0].StartupPolicy != scenario.DependencyStartupPolicyTryStart {
		t.Fatalf("disabled consumers = %#v", item.DisabledDependencyConsumers)
	}
}

func TestScenarioDependencyEnablesResourceWithoutProjectEntry(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testresource.WriteExternalCLIResourceFixture(t, fixture.Root, "whisper", shelltest.BashShebang()+"exit 0\n")
	testscenario.WriteScenarioService(t, fixture.Root, "audio-tools", testscenario.ScenarioServiceManifest(
		"audio-tools",
		testscenario.WithDependencies(scenario.Dependencies{Resources: map[string]scenario.Dependency{
			"whisper": {StartupPolicy: scenario.DependencyStartupPolicyTryStart},
		}}),
	))

	item, err := New(fixture.Root).DiscoverOne("whisper", DiscoverOptions{})
	if err != nil {
		t.Fatalf("DiscoverOne: %v", err)
	}
	if item == nil || !item.Enabled {
		t.Fatalf("item = %#v, want enabled by scenario declaration", item)
	}
}

func TestReadConfigEntriesToleratesUnknownServiceManifestFields(t *testing.T) {
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	testkitgo.WriteFile(t, filepath.Join(fixture.Root, ".vrooli", "service.json"), `{
  "dependencies": {"resources": {"redis": {
    "enabled": true,
    "required": true,
    "description": "Cache responses",
    "future_dependency_field": {"mode": "new"}
  }}},
  "future_manifest_field": {"version": 2}
}`)

	entries, err := New(fixture.Root).ReadConfigEntries()
	if err != nil {
		t.Fatalf("ReadConfigEntries: %v", err)
	}
	if got := entries["redis"]; !got.Enabled || !got.Required || got.Description != "Cache responses" {
		t.Fatalf("redis config = %+v", got)
	}
}
