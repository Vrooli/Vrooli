package aisearch

import (
	"context"
	"testing"
)

type routeObserverDiscoveryFake struct{}

func (routeObserverDiscoveryFake) ListScenarios(context.Context) ([]string, error) {
	return []string{"react-component-library"}, nil
}

func (routeObserverDiscoveryFake) Discover(context.Context, string) ([]SurfaceRecord, error) {
	return []SurfaceRecord{
		{Scenario: "react-component-library", Slot: "page", Kind: "SURFACE_KIND_PAGE", DisplayName: "CoveragePage", FilePath: "ui/src/pages/CoveragePage.tsx"},
	}, nil
}

func TestObservedDiscoveryEnrichesPageSurfaceFromRoute(t *testing.T) {
	d := newObservedDiscovery(routeObserverDiscoveryFake{})
	d.byRoute["react-component-library"] = map[string]RouteObservation{
		"/coverage": {Scenario: "react-component-library", Route: "/coverage", LinkText: "Catalog coverage", PageTitle: "UI", ObservedAt: "2026-09-07T19:49:45Z", Reachable: true, HTTPStatus: 200},
	}

	records, err := d.Discover(context.Background(), "react-component-library")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ObservedRoute != "/coverage" || records[0].ObservedLinkText != "Catalog coverage" {
		t.Fatalf("route observation did not enrich surface: %+v", records)
	}
}
