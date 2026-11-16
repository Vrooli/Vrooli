package validationcatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

type staticResolver string

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return string(r), nil
}

func TestResolveMapsProviderOwnedJourneyCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/validation/catalog" || r.URL.Query().Get("scenario") != "sample" {
			t.Fatalf("unexpected catalog request: %s", r.URL.String())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"journeys":[{"journey_id":"sample/bas/cases/login.json","display_name":"Login","source_path":"bas/cases/login.json","execution_mode":"observer","required":true,"category":"existing-bas-case","requirements":["auth"],"estimated_duration_seconds":12,"safety":{"mutating":false,"requires_isolation":false,"requires_confirmation":false}}]}`))
	}))
	defer server.Close()

	catalog, err := NewClient(staticResolver(server.URL), server.Client()).Resolve(context.Background(), "sample")
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Journeys) != 2 || catalog.Journeys[0].JourneyID != "sample/bas/cases/login.json" || catalog.Journeys[0].Safety.Mutating {
		t.Fatalf("unexpected catalog: %+v", catalog)
	}
	platform := catalog.Journeys[1]
	if platform.JourneyID != "desktop-smoke" || platform.ExecutionMode != "platform" || !platform.Required || platform.SourcePath != "" {
		t.Fatalf("desktop smoke must be a generic required platform journey without a provider source path: %+v", platform)
	}
	selection, err := (validationmatrix.MatrixSelection{
		ScenarioName: "sample",
		Journeys:     []validationmatrix.JourneySelection{{JourneyID: "desktop-smoke", Required: true}},
	}).WithCatalog(context.Background(), NewClient(staticResolver(server.URL), server.Client()))
	if err != nil {
		t.Fatalf("platform journey should be accepted by the shared catalog contract: %v", err)
	}
	if len(selection.Journeys) != 1 || selection.Journeys[0].ExecutionMode != "platform" {
		t.Fatalf("platform selection did not resolve to the generic catalog entry: %+v", selection.Journeys)
	}
}

func TestDesktopSmokeCatalogEntryDoesNotReplaceProviderEntry(t *testing.T) {
	journeys := []validationmatrix.JourneySelection{{
		JourneyID: "desktop-smoke", DisplayName: "Provider-owned custom smoke", SourcePath: "custom/smoke.json", ExecutionMode: "observer",
	}}
	merged := withDesktopSmokeJourney(journeys)
	if len(merged) != 1 || merged[0].DisplayName != "Provider-owned custom smoke" || merged[0].ExecutionMode != "observer" {
		t.Fatalf("provider-owned catalog entry must win over platform default: %+v", merged)
	}
	if len(journeys) != 1 {
		t.Fatalf("catalog merge mutated the provider snapshot: %+v", journeys)
	}
}

func TestPackagedBASConsentJourneyIsRegisteredOnlyForBAS(t *testing.T) {
	journeys := []validationmatrix.JourneySelection{{JourneyID: "bas/cases/example.json", DisplayName: "Existing BAS case"}}
	bas := withPackagedBASConsentJourney("browser-automation-studio", journeys)
	if len(bas) != 2 || bas[1].JourneyID != "browser-automation-studio:a15/packaged-bas-consent" || bas[1].ExecutionMode != "s2d-packaged-fixture" || bas[1].SourcePath == "" {
		t.Fatalf("BAS consent journey was not registered with its runnable fixture: %+v", bas)
	}
	if len(journeys) != 1 {
		t.Fatalf("BAS catalog registration mutated the provider snapshot: %+v", journeys)
	}
	other := withPackagedBASConsentJourney("web-console", journeys)
	if len(other) != 1 {
		t.Fatalf("BAS-specific fixture leaked into another scenario catalog: %+v", other)
	}
	selection, err := (validationmatrix.MatrixSelection{
		ScenarioName: "browser-automation-studio",
		Journeys:     []validationmatrix.JourneySelection{{JourneyID: "browser-automation-studio:a15/packaged-bas-consent", Required: true}},
	}).WithCatalog(context.Background(), staticCatalog{journeys: bas})
	if err != nil {
		t.Fatalf("registered packaged consent journey was rejected by matrix selection: %v", err)
	}
	if got := selection.Journeys[0].ExecutionMode; got != "s2d-packaged-fixture" {
		t.Fatalf("selected journey execution mode = %q", got)
	}
}

func TestPackagedWebConsoleMonetizationJourneyIsRegisteredOnlyForWebConsole(t *testing.T) {
	journeys := []validationmatrix.JourneySelection{{JourneyID: "existing", DisplayName: "Existing provider journey"}}
	webConsole := withPackagedWebConsoleMonetizationJourney("web-console", journeys)
	if len(webConsole) != 2 || webConsole[1].JourneyID != "web-console:sp1/packaged-metered-ai" || webConsole[1].ExecutionMode != "s2d-packaged-fixture" || webConsole[1].SourcePath == "" {
		t.Fatalf("Web Console packaged monetization journey was not registered with its runnable fixture: %+v", webConsole)
	}
	if len(journeys) != 1 {
		t.Fatalf("Web Console catalog registration mutated the provider snapshot: %+v", journeys)
	}
	other := withPackagedWebConsoleMonetizationJourney("browser-automation-studio", journeys)
	if len(other) != 1 {
		t.Fatalf("Web Console-specific fixture leaked into another scenario catalog: %+v", other)
	}
	selection, err := (validationmatrix.MatrixSelection{
		ScenarioName: "web-console",
		Journeys:     []validationmatrix.JourneySelection{{JourneyID: "web-console:sp1/packaged-metered-ai", Required: true}},
	}).WithCatalog(context.Background(), staticCatalog{journeys: webConsole})
	if err != nil {
		t.Fatalf("registered packaged Web Console journey was rejected by matrix selection: %v", err)
	}
	if len(selection.Journeys) != 1 || selection.Journeys[0].ExecutionMode != "s2d-packaged-fixture" {
		t.Fatalf("selected Web Console journey did not resolve to the packaged fixture: %+v", selection.Journeys)
	}
}

func TestPackagedWebConsoleJourneyReplacesStaleProviderPathForOwnedID(t *testing.T) {
	journeys := []validationmatrix.JourneySelection{{
		JourneyID: "web-console:sp1/packaged-metered-ai", DisplayName: "Stale provider copy",
		SourcePath: "scenarios/web-console/api:TestWebConsolePackagedMonetizationJourney", ExecutionMode: "provider",
	}}
	merged := withPackagedWebConsoleMonetizationJourney("web-console", journeys)
	if len(merged) != 1 || merged[0].SourcePath != "scenarios/web-console/api:TestWebConsolePackagedMeteringRefusal" || merged[0].ExecutionMode != "s2d-packaged-fixture" {
		t.Fatalf("S2D-owned packaged journey kept stale provider provenance: %+v", merged)
	}
	if journeys[0].SourcePath != "scenarios/web-console/api:TestWebConsolePackagedMonetizationJourney" {
		t.Fatalf("catalog merge mutated provider snapshot: %+v", journeys)
	}
}

type staticCatalog struct {
	journeys []validationmatrix.JourneySelection
}

func (c staticCatalog) Resolve(context.Context, string) (validationmatrix.CatalogSnapshot, error) {
	return validationmatrix.CatalogSnapshot{Journeys: c.journeys}, nil
}

func TestResolveFailsClosedOnProviderError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	_, err := NewClient(staticResolver(server.URL), server.Client()).Resolve(context.Background(), "sample")
	if err == nil {
		t.Fatal("expected catalog provider error")
	}
}

var _ validationmatrix.CatalogResolver = (*Client)(nil)
