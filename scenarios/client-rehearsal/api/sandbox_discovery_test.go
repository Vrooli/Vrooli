package main

import (
	"context"
	"errors"
	"testing"
)

func TestResolveWorkspaceSandboxURLUsesConfiguredAddress(t *testing.T) {
	called := false
	got, err := resolveWorkspaceSandboxURL(context.Background(), " http://localhost:15120 ", func(context.Context, string, string) (int, error) {
		called = true
		return 1, nil
	})
	if err != nil || got != "http://localhost:15120" || called {
		t.Fatalf("URL = %q, called lookup = %t, err = %v", got, called, err)
	}
}

func TestResolveWorkspaceSandboxURLDiscoversDynamicPort(t *testing.T) {
	called := false
	got, err := resolveWorkspaceSandboxURL(context.Background(), "", func(_ context.Context, scenario, portName string) (int, error) {
		called = true
		if scenario != "workspace-sandbox" || portName != "API_PORT" {
			t.Fatalf("lookup target = %s/%s", scenario, portName)
		}
		return 15123, nil
	})
	if err != nil || got != "http://localhost:15123" || !called {
		t.Fatalf("URL = %q, called lookup = %t, err = %v", got, called, err)
	}
}

func TestResolveWorkspaceSandboxURLReportsMissingOrInvalidDiscovery(t *testing.T) {
	for name, lookup := range map[string]scenarioPortLookup{
		"not configured":   nil,
		"discovery failed": func(context.Context, string, string) (int, error) { return 0, errors.New("scenario stopped") },
		"invalid port":     func(context.Context, string, string) (int, error) { return 70000, nil },
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := resolveWorkspaceSandboxURL(context.Background(), "", lookup); err == nil || got != "" {
				t.Fatalf("URL = %q, err = %v; want empty URL and an error", got, err)
			}
		})
	}
}

func TestResolveWorkspaceSandboxURLRejectsInvalidConfiguredAddress(t *testing.T) {
	if got, err := resolveWorkspaceSandboxURL(context.Background(), "not a URL", nil); err == nil || got != "" {
		t.Fatalf("URL = %q, err = %v; want invalid configuration error", got, err)
	}
}
