package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	component "github.com/vrooli/vrooli/packages/desktop-component"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/components"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

type ac7OptionalCatalogRuntime struct {
	*mockRuntime
	declarations []capabilityregistry.Def
}

func (f *ac7OptionalCatalogRuntime) ComponentCatalog(context.Context) ([]component.SignedEntry, error) {
	return nil, nil
}
func (f *ac7OptionalCatalogRuntime) ComponentInstallations(context.Context) ([]components.Installation, error) {
	return nil, nil
}
func (f *ac7OptionalCatalogRuntime) InstallComponent(context.Context, string) (components.Installation, error) {
	return components.Installation{}, nil
}
func (f *ac7OptionalCatalogRuntime) RemoveComponent(context.Context, string, bool) error { return nil }
func (f *ac7OptionalCatalogRuntime) OptionalDependencyDeclarations(context.Context) ([]capabilityregistry.Def, error) {
	return f.declarations, nil
}

func TestComponentResponseJoinsStagedOptionalRowsToScenarioCatalogIdentity(t *testing.T) {
	type optionalCatalogRow struct {
		Name         string                   `json:"name"`
		Dependencies []capabilityregistry.Def `json:"optional_dependencies"`
	}
	catalogRow := optionalCatalogRow{Name: "web-console", Dependencies: []capabilityregistry.Def{
		{ID: "disabled-audio", DependencySlug: "audio-tools", DependencyKind: capabilityregistry.DependencyScenario, Enabled: false},
		{ID: "ollama", DependencySlug: "ollama", DependencyKind: capabilityregistry.DependencyResource, Enabled: true},
		{ID: "audio-tools", DependencySlug: "audio-tools", DependencyKind: capabilityregistry.DependencyScenario, Enabled: true},
		{ID: "control-plane", DependencySlug: "vrooli", DependencyKind: capabilityregistry.DependencyControlPlane, Enabled: true},
	}}
	// Round-trip the generated all-app row's wire fields before joining to the runtime response.
	catalogBytes, err := json.Marshal(catalogRow)
	if err != nil {
		t.Fatal(err)
	}
	var generatedCatalogRow optionalCatalogRow
	if err := json.Unmarshal(catalogBytes, &generatedCatalogRow); err != nil {
		t.Fatal(err)
	}
	runtime := &ac7OptionalCatalogRuntime{
		mockRuntime:  &mockRuntime{manifest: &manifest.Manifest{App: manifest.App{Scenario: generatedCatalogRow.Name, Version: "1.2.0"}}},
		declarations: append(append([]capabilityregistry.Def(nil), generatedCatalogRow.Dependencies...), capabilityregistry.Def{ID: "required-core", DependencySlug: "agent-manager", DependencyKind: capabilityregistry.DependencyScenario, Enabled: true, Required: true}),
	}
	response := httptest.NewRecorder()
	server := NewServer(runtime, "test-token")
	handler := server.AuthMiddleware(http.HandlerFunc(server.handleComponents))
	request := httptest.NewRequest(http.MethodGet, "/components", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /components = HTTP%d: %s", response.Code, response.Body.String())
	}
	var body struct {
		AppScenario  string                      `json:"app_scenario"`
		Dependencies []OptionalDependencyRelease `json:"dependencies"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode components response: %v", err)
	}
	if body.AppScenario != generatedCatalogRow.Name {
		t.Fatalf("app_scenario = %q; optional catalog row name is %q", body.AppScenario, generatedCatalogRow.Name)
	}
	byID := make(map[string]OptionalDependencyRelease, len(body.Dependencies))
	for _, row := range body.Dependencies {
		byID[row.DependencyID] = row
	}
	if len(byID) != len(generatedCatalogRow.Dependencies) {
		t.Fatalf("optional response rows = %d; generated catalog row has %d optional declarations", len(byID), len(generatedCatalogRow.Dependencies))
	}
	for _, declaration := range generatedCatalogRow.Dependencies {
		row, ok := byID[declaration.ID]
		if !ok || row.DependencySlug != declaration.DependencySlug || row.DependencyKind != string(declaration.DependencyKind) || row.Enabled != declaration.Enabled {
			t.Errorf("runtime row %q does not match generated optional declaration: found=%t row=%+v declaration=%+v", declaration.ID, ok, row, declaration)
		}
	}
	disabled := byID["disabled-audio"]
	if disabled.State != "disabled" || disabled.Enabled || disabled.CanInstallOrUpdate || disabled.CanRemove {
		t.Fatalf("disabled optional declaration acquired lifecycle actions: %+v", disabled)
	}
	resource := byID["ollama"]
	if resource.State != "resource_managed" || !resource.Enabled || resource.CanInstallOrUpdate || resource.CanRemove {
		t.Fatalf("resource declaration was routed to scenario release lifecycle: %+v", resource)
	}
	controlPlane := byID["control-plane"]
	if controlPlane.State != "control_plane_managed" || !controlPlane.Enabled || controlPlane.CanInstallOrUpdate || controlPlane.CanRemove {
		t.Fatalf("control-plane declaration was routed to scenario release lifecycle: %+v", controlPlane)
	}
	scenario := byID["audio-tools"]
	if scenario.State != "catalog_unavailable" || !scenario.Enabled || scenario.CanInstallOrUpdate || scenario.CanRemove {
		t.Fatalf("unconfigured signed release catalog was not fail-closed: %+v", scenario)
	}
	if _, ok := byID["required-core"]; ok {
		t.Fatal("required dependency leaked into the optional catalog response")
	}
}
