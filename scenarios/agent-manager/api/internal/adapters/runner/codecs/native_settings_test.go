package codecs

import (
	"agent-manager/internal/domain"
	"testing"
)

func TestNativeCodexSettingsPreserveAdmittedControls(t *testing.T) {
	for _, network := range []domain.NetworkAccess{domain.NetworkAccessNone, domain.NetworkAccessFull} {
		t.Run(string(network), func(t *testing.T) {
			cfg := &domain.RunConfig{Model: "gpt-6-luna", Effort: domain.EffortHigh, ContextLimitTokens: 12345, NetworkAccess: network}
			got, err := NewCodexForTest().NativeCodexSettings(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if got.ControlsVersion != 1 || got.Model != cfg.Model || got.ReasoningEffort != "high" || got.ContextLimitTokens != 12345 || got.ApprovalPolicy != "never" {
				t.Fatalf("settings = %+v", got)
			}
			if network == domain.NetworkAccessNone {
				if got.Sandbox != "workspace-write" || !got.NetworkAccessDisabled {
					t.Fatalf("network denial lost: %+v", got)
				}
			} else if got.Sandbox != "danger-full-access" || got.NetworkAccessDisabled {
				t.Fatalf("host permissions changed: %+v", got)
			}
		})
	}
}

func TestNativeCodexSettingsRejectUnsupportedControls(t *testing.T) {
	for name, cfg := range map[string]*domain.RunConfig{
		"protected":              {SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeProtected}},
		"unspecified outer mode": {SandboxConfig: &domain.SandboxConfig{}},
		"tool allow":             {AllowedTools: []string{"shell"}},
		"tool deny":              {DeniedTools: []string{"shell"}},
		"extra flags":            {ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeCodex: {"--search"}}},
		"feature":                {Features: domain.FeatureFlags{EnableBrowser: true}},
		"allowed path":           {AllowedPaths: []string{"/tmp"}},
		"denied path":            {DeniedPaths: []string{"/etc"}},
		"containment":            {RequireEffectContainment: true},
		"effort":                 {Effort: domain.EffortMax},
		"provider":               {Model: "ollama/test"},
		"unspecified network":    {},
		"localhost only":         {NetworkAccess: domain.NetworkAccessLocalhost},
		"network":                {NetworkAccess: domain.NetworkAccess("invalid")},
		"negative limit":         {ContextLimitTokens: -1},
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := NewCodexForTest().NativeCodexSettings(cfg); err == nil {
				t.Fatalf("unsupported controls accepted: %+v", got)
			}
		})
	}
}
