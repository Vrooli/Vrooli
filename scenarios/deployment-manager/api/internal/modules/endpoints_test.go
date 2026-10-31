package modules

import "testing"

func TestAllEndpointsExposeStableConnectContracts(t *testing.T) {
	endpoints := AllEndpoints()
	if len(endpoints) < 30 {
		t.Fatalf("endpoint count = %d", len(endpoints))
	}
	seen := map[string]bool{}
	for _, endpoint := range endpoints {
		if endpoint.ID == "" || endpoint.Path == "" || endpoint.Method == "" || endpoint.Category == "" {
			t.Errorf("incomplete endpoint: %#v", endpoint)
		}
		if seen[endpoint.ID] {
			t.Errorf("duplicate endpoint id %q", endpoint.ID)
		}
		seen[endpoint.ID] = true
	}
}
