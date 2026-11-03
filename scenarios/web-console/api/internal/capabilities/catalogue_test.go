package capabilities

import (
	"context"
	"testing"
)

func TestKnownCatalogueExplainsEveryIntegration(t *testing.T) {
	t.Setenv("WEB_CONSOLE_SERVICE_MANIFEST", "../../../.vrooli/service.json")
	defs := KnownForPlatform("linux")
	if len(defs) <= len(virtualCatalogue) {
		t.Fatal("production manifest integrations are missing")
	}
	for _, def := range defs {
		if len(def.Uses) == 0 {
			t.Errorf("integration %q has no consumer purpose", def.ID)
		}
		for _, use := range def.Uses {
			if use.ID == def.DependencySlug || use.Name == "" || len(use.Description) <= 15 || use.Group == "" {
				t.Errorf("integration %q has incomplete consumer purpose: %+v", def.ID, use)
			}
		}
	}
}

func TestKnownCatalogueHasLiveCheckerForEveryDeclaredIntegration(t *testing.T) {
	want := map[string]DependencyKind{
		"audio-tools":                DependencyScenario,
		"ollama":                     DependencyResource,
		"openrouter":                 DependencyResource,
		"session-backend-standard":   DependencyResource,
		"session-backend-persistent": DependencyResource,
		"vrooli-bridge":              DependencyScenario,
		"agent-manager":              DependencyScenario,
		"integration-hub":            DependencyScenario,
		"notification-hub":           DependencyScenario,
		"claude":                     DependencyResource,
		"codex":                      DependencyResource,
		"opencode":                   DependencyResource,
		"grok":                       DependencyResource,
		"agy":                        DependencyResource,
	}

	checkers := map[string]Checker{
		"audio-tools":                &ScenarioChecker{Slug: "audio-tools"},
		"ollama":                     &OllamaChecker{BaseURL: "http://127.0.0.1:1"},
		"openrouter":                 &OpenRouterChecker{},
		"session-backend-standard":   &StaticChecker{Available: func() (bool, string) { return true, "" }},
		"session-backend-persistent": &StaticChecker{Available: func() (bool, string) { return true, "" }},
		"vrooli-bridge":              &BridgeChecker{},
		"agent-manager":              &ScenarioChecker{Slug: "agent-manager"},
		"integration-hub":            &ScenarioChecker{Slug: "integration-hub"},
		"notification-hub":           &ScenarioChecker{Slug: "notification-hub"},
		"claude":                     HostCapabilityChecker{},
		"codex":                      HostCapabilityChecker{},
		"opencode":                   HostCapabilityChecker{},
		"grok":                       HostCapabilityChecker{},
		"agy":                        HostCapabilityChecker{},
	}

	seen := make(map[string]bool, len(Known))
	for _, def := range Known {
		seen[def.ID] = true
		if want[def.ID] != def.DependencyKind {
			t.Errorf("catalogue entry %q kind = %q, want %q", def.ID, def.DependencyKind, want[def.ID])
		}
		if _, ok := checkers[def.ID]; !ok {
			t.Errorf("catalogue entry %q has no checker", def.ID)
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("real catalogue is missing %q", id)
		}
	}

	registry := NewRegistry(Known, checkers, 0)
	if err := registry.Validate(); err != nil {
		t.Fatalf("real catalogue validation failed: %v", err)
	}
	if states := registry.Resolve(context.Background()); len(states) != len(want) {
		t.Fatalf("resolved states = %d, want %d", len(states), len(want))
	}
}

func TestBridgeCheckerReportsTypedConfigurationReasons(t *testing.T) {
	tests := []struct {
		name    string
		checker BridgeChecker
		code    string
	}{
		{name: "missing URL", checker: BridgeChecker{}, code: "bridge_url_missing"},
		{name: "invalid URL", checker: BridgeChecker{BaseURL: "bridge://bad"}, code: "bridge_url_invalid"},
		{name: "missing credentials", checker: BridgeChecker{BaseURL: "http://bridge.test"}, code: "bridge_credentials_missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.checker.CheckResult(context.Background())
			if result.ReasonCode != tt.code {
				t.Fatalf("reason = %q, want %q", result.ReasonCode, tt.code)
			}
			if result.ActionKind != ActionKindScenarioStart {
				t.Fatalf("action = %q, want scenario start", result.ActionKind)
			}
		})
	}
}

func TestKnownForPlatformMarksUnavailableBackends(t *testing.T) {
	defs := KnownForPlatform("windows")
	for _, def := range defs {
		switch def.ID {
		case "session-backend-standard":
			if def.Platform.Support == PlatformUnsupported {
				t.Fatalf("native ConPTY standard backend incorrectly unsupported: %+v", def.Platform)
			}
		case "session-backend-persistent":
			if def.Platform.Support != PlatformUnsupported || def.Platform.Reason != "tmux is not available on this platform" {
				t.Fatalf("persistent Windows verdict = %+v", def.Platform)
			}
		}
	}
}
