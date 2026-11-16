package validationcatalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

type URLResolver interface {
	ResolveScenarioURLDefault(context.Context, string) (string, error)
}

type discoveryResolver struct{}

func (discoveryResolver) ResolveScenarioURLDefault(ctx context.Context, scenario string) (string, error) {
	return discovery.ResolveScenarioURLDefault(ctx, scenario)
}

type Client struct {
	resolver URLResolver
	http     *http.Client
}

func NewWorkflowHealthResolver() *Client {
	return NewClient(discoveryResolver{}, &http.Client{Timeout: 10 * time.Second})
}

func NewClient(resolver URLResolver, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{resolver: resolver, http: httpClient}
}

type response struct {
	Journeys []validationmatrix.JourneySelection `json:"journeys"`
}

// desktopSmokeJourney is owned by scenario-to-desktop: it validates the
// generated application shell and native launch path without claiming a
// provider-owned semantic workflow. Keeping it in the catalog adapter lets
// shared validation-matrix policy continue to reject unknown semantic
// journeys while making the platform-only smoke journey selectable for every
// scenario.
var desktopSmokeJourney = validationmatrix.JourneySelection{
	JourneyID:     "desktop-smoke",
	DisplayName:   "Desktop platform smoke",
	Category:      "desktop-platform-smoke",
	ExecutionMode: "platform",
	Required:      true,
}

// packagedBASConsentJourney binds the already exercised native PKCE, signed
// BAS entitlement, consent lifecycle, and fake Gateway dispatch fixture to
// the scenario-to-desktop execution owner. It is deliberately registered
// only for BAS; other scenarios continue to use the provider catalog.
var packagedBASConsentJourney = validationmatrix.JourneySelection{
	JourneyID:            "browser-automation-studio:a15/packaged-bas-consent",
	DisplayName:          "Packaged BAS consent and Gateway",
	SourcePath:           "scenarios/ai-gateway/api/internal/identity:TestGatewayConsentBrowserJourney",
	ExecutionMode:        "s2d-packaged-fixture",
	Required:             true,
	Category:             "packaged-consent-gateway",
	Requirements:         []string{"native-pkce", "app-scoped-signed-entitlement", "consent-grant-reload-revoke", "fake-gateway-dispatch-refusal"},
	EstimatedDurationSec: 120,
	Safety:               validationmatrix.JourneySafety{Mutating: true, RequiresIsolation: true},
}

// packagedWebConsoleMonetizationJourney binds the Web Console paid-surface
// fixture to the exact selected AppImage. The fixture is scenario-owned; S2D
// only runs it and persists its evidence through the matrix contract.
var packagedWebConsoleMonetizationJourney = validationmatrix.JourneySelection{
	JourneyID:            "web-console:sp1/packaged-metered-ai",
	DisplayName:          "Packaged Web Console metered AI refusal",
	SourcePath:           "scenarios/web-console/api:TestWebConsolePackagedMeteringRefusal",
	ExecutionMode:        "s2d-packaged-fixture",
	Required:             true,
	Category:             "packaged-monetization",
	Requirements:         []string{"root-api-artifact-binding", "metered-ai-no-session-refusal", "offline-api"},
	EstimatedDurationSec: 120,
	Safety:               validationmatrix.JourneySafety{Mutating: true, RequiresIsolation: true},
}

func withDesktopSmokeJourney(journeys []validationmatrix.JourneySelection) []validationmatrix.JourneySelection {
	for _, journey := range journeys {
		if journey.JourneyID == desktopSmokeJourney.JourneyID {
			return journeys
		}
	}
	result := append([]validationmatrix.JourneySelection(nil), journeys...)
	return append(result, desktopSmokeJourney)
}

func withPackagedBASConsentJourney(scenario string, journeys []validationmatrix.JourneySelection) []validationmatrix.JourneySelection {
	if strings.TrimSpace(scenario) != "browser-automation-studio" {
		return journeys
	}
	for _, journey := range journeys {
		if journey.JourneyID == packagedBASConsentJourney.JourneyID {
			return journeys
		}
	}
	result := append([]validationmatrix.JourneySelection(nil), journeys...)
	return append(result, packagedBASConsentJourney)
}

func withPackagedWebConsoleMonetizationJourney(scenario string, journeys []validationmatrix.JourneySelection) []validationmatrix.JourneySelection {
	if strings.TrimSpace(scenario) != "web-console" {
		return journeys
	}
	for index, journey := range journeys {
		if journey.JourneyID == packagedWebConsoleMonetizationJourney.JourneyID {
			// This stable ID is owned by the S2D packaged fixture. Replace an older
			// provider snapshot for the same ID so persisted selections point at the
			// test the executor actually runs.
			result := append([]validationmatrix.JourneySelection(nil), journeys...)
			result[index] = packagedWebConsoleMonetizationJourney
			return result
		}
	}
	result := append([]validationmatrix.JourneySelection(nil), journeys...)
	return append(result, packagedWebConsoleMonetizationJourney)
}

func (c *Client) Resolve(ctx context.Context, scenario string) (validationmatrix.CatalogSnapshot, error) {
	if c == nil || c.resolver == nil || c.http == nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("workflow validation catalog client is unavailable")
	}
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("scenario is required")
	}
	base, err := c.resolver.ResolveScenarioURLDefault(ctx, "workflow-health")
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("resolve workflow-health URL: %w", err)
	}
	endpoint := strings.TrimRight(base, "/") + "/api/v1/validation/catalog?scenario=" + url.QueryEscape(scenario)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("build catalog request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("read workflow-health catalog: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("workflow-health catalog returned HTTP %d", resp.StatusCode)
	}
	var payload response
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("decode workflow-health catalog: %w", err)
	}
	journeys := withDesktopSmokeJourney(payload.Journeys)
	journeys = withPackagedBASConsentJourney(scenario, journeys)
	journeys = withPackagedWebConsoleMonetizationJourney(scenario, journeys)
	return validationmatrix.CatalogSnapshot{Journeys: journeys}, nil
}
