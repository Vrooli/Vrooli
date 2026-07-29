package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisteredRoutesMatchesCurrentRegistry(t *testing.T) {
	routes, err := registeredRoutes(filepath.Join("..", "..", "routes.go"))
	if err != nil {
		t.Fatalf("registeredRoutes() error = %v", err)
	}
	if len(routes) < 100 {
		t.Fatalf("registeredRoutes() returned %d routes; expected the current API registry to contain at least 100", len(routes))
	}

	allRoutes, err := inventoryRoutes(filepath.Join("..", "..", "routes.go"))
	if err != nil {
		t.Fatalf("inventoryRoutes() error = %v", err)
	}
	if len(allRoutes) <= len(routes) {
		t.Fatalf("inventoryRoutes() returned %d routes; expected generated Measures RPCs in addition to %d mux routes", len(allRoutes), len(routes))
	}

	endpoints := endpointsFor(allRoutes)
	if len(endpoints) != len(allRoutes) {
		t.Fatalf("endpointsFor() returned %d endpoints for %d routes", len(endpoints), len(allRoutes))
	}
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.ID == "" || endpoint.Path == "" || endpoint.Method == "" {
			t.Fatalf("generated incomplete endpoint: %#v", endpoint)
		}
		if _, duplicate := seen[endpoint.ID]; duplicate {
			t.Fatalf("duplicate generated endpoint id %q", endpoint.ID)
		}
		seen[endpoint.ID] = struct{}{}
	}
}

func TestEndpointIDDistinguishesMethods(t *testing.T) {
	get := endpointID(route{Path: "/api/v1/admin/profile", Method: "GET"})
	put := endpointID(route{Path: "/api/v1/admin/profile", Method: "PUT"})
	if get == put {
		t.Fatalf("endpoint IDs must distinguish methods: %q", get)
	}
}

func TestCommittedManifestMatchesRouteRegistry(t *testing.T) {
	temporaryOutput := filepath.Join(t.TempDir(), "endpoints.json")
	if err := generate(filepath.Join("..", "..", "routes.go"), temporaryOutput); err != nil {
		t.Fatalf("generate() error = %v", err)
	}
	generated, err := os.ReadFile(temporaryOutput)
	if err != nil {
		t.Fatalf("read generated endpoint manifest: %v", err)
	}
	committed, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "endpoints.json"))
	if err != nil {
		t.Fatalf("read committed endpoint manifest: %v", err)
	}
	if !bytes.Equal(generated, committed) {
		t.Fatal(".vrooli/endpoints.json is stale; run make endpoints from the scenario root and commit the result")
	}
	info, err := os.Stat(temporaryOutput)
	if err != nil {
		t.Fatalf("stat generated endpoint manifest: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("generated endpoint manifest mode = %o, want 600", got)
	}
}
