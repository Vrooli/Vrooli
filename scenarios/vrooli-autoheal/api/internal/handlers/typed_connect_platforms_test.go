package handlers

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// TestListChecksExposesDeclaredPlatforms pins the platform declaration onto
// the typed surface. Before it was carried there, the only way to learn that a
// check is Linux-only was to parse this scenario's Go source from another
// scenario — a reader that goes stale silently the moment a check changes its
// declaration, and that cannot run against a deployed binary at all.
//
// The empty case is the load-bearing half: a check declaring no platforms
// applies everywhere, and an empty list must therefore read as "all", never as
// "unknown".
func TestListChecksExposesDeclaredPlatforms(t *testing.T) {
	caps := &platform.Capabilities{Platform: platform.Linux}
	registry := checks.NewRegistry(caps)
	registry.Register(&mockCheck{id: "linux-only", status: checks.StatusOK, platform: []platform.Type{platform.Linux}})
	registry.Register(&mockCheck{id: "linux-and-macos", status: checks.StatusOK, platform: []platform.Type{platform.Linux, platform.MacOS}})
	registry.Register(&mockCheck{id: "every-platform", status: checks.StatusOK})

	handlers := NewWithInterface(registry, &mockStore{}, caps)
	handlers.hostCollector = fakeHostCollector{}
	service := &typedChecks{h: handlers}

	response, err := service.ListChecks(context.Background(), connect.NewRequest(&checksproto.ListChecksRequest{}))
	if err != nil {
		t.Fatalf("ListChecks: %v", err)
	}
	got := make(map[string][]string, len(response.Msg.GetChecks()))
	for _, info := range response.Msg.GetChecks() {
		got[info.GetId()] = info.GetPlatforms()
	}

	want := map[string][]string{
		"linux-only":      {"linux"},
		"linux-and-macos": {"linux", "macos"},
		"every-platform":  {},
	}
	for id, expected := range want {
		actual, ok := got[id]
		if !ok {
			t.Errorf("check %q is missing from ListChecks", id)
			continue
		}
		if len(actual) != len(expected) {
			t.Errorf("check %q reports platforms %v, want %v", id, actual, expected)
			continue
		}
		for i := range expected {
			if actual[i] != expected[i] {
				t.Errorf("check %q reports platforms %v, want %v", id, actual, expected)
				break
			}
		}
	}
}
