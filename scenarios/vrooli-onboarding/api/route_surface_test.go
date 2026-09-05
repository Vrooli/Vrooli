package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"testing"
)

type routeSurfaceEntry struct {
	Method         string   `json:"method"`
	Path           string   `json:"path"`
	Surfaces       []string `json:"surfaces"`
	InternalReason string   `json:"internal_reason"`
}

func TestEveryRegisteredRouteIsInTheSurfaceContract(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "route-surface.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Routes []routeSurfaceEntry `json:"routes"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	contractRoutes := map[string]routeSurfaceEntry{}
	for _, route := range contract.Routes {
		key := route.Method + " " + route.Path
		if _, exists := contractRoutes[key]; exists {
			t.Fatalf("duplicate route contract entry %s", key)
		}
		contractRoutes[key] = route
		if len(route.Surfaces) == 0 && route.InternalReason == "" {
			t.Errorf("%s has no surface and no internal_reason", key)
		}
	}

	source, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatal(err)
	}
	pattern := regexp.MustCompile(`Handle(?:Func)?\("([^"]+)"[^\n]*\)\.Methods\("([A-Z]+)"\)`)
	registered := map[string]bool{}
	for _, match := range pattern.FindAllStringSubmatch(string(source), -1) {
		registered[match[2]+" "+match[1]] = true
	}
	if len(registered) != len(contractRoutes) {
		t.Fatalf("registered route count = %d, contract count = %d; registered=%v contract=%v", len(registered), len(contractRoutes), sortedKeys(registered), sortedEntryKeys(contractRoutes))
	}
	for key := range registered {
		if _, ok := contractRoutes[key]; !ok {
			t.Errorf("registered route missing from contract: %s", key)
		}
	}
	for key := range contractRoutes {
		if !registered[key] {
			t.Errorf("contract route is not registered: %s", key)
		}
	}
}

func sortedKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedEntryKeys(values map[string]routeSurfaceEntry) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
