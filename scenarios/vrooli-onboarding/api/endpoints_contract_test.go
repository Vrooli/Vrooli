package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/gorilla/mux"
)

// TestEndpointContract keeps the checked-in public declaration honest in both
// directions. It is intentionally router-derived rather than a second list of
// hand-authored paths, so adding a route without updating the contract fails.
func TestEndpointContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", ".vrooli", "endpoints.json"))
	if err != nil {
		t.Fatal(err)
	}
	var declared struct {
		Endpoints []struct {
			Path   string `json:"path"`
			Method string `json:"method"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(data, &declared); err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{}
	for _, endpoint := range declared.Endpoints {
		want[endpoint.Method+" "+endpoint.Path] = true
	}
	got := map[string]bool{}
	err = NewServer().router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, pathErr := route.GetPathTemplate()
		methods, methodErr := route.GetMethods()
		if pathErr != nil || methodErr != nil {
			return nil
		}
		for _, method := range methods {
			got[method+" "+path] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var missing, extra []string
	for key := range want {
		if !got[key] {
			missing = append(missing, key)
		}
	}
	for key := range got {
		if !want[key] {
			extra = append(extra, key)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		t.Fatalf("endpoint contract drift: missing=%v extra=%v", missing, extra)
	}
}
