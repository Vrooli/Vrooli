package hostreq

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/scenario"
	testscenario "github.com/vrooli/vrooli/packages/testkit-go/scenariofixture"
)

func TestResolveSafeguardConfigAppliesDefaultsAndOperatorOverrides(t *testing.T) {
	manifest := hostreqkit.SafeguardManifest{
		Name: "example",
		Config: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"reservation": map[string]any{"type": "string", "default": "512M-:256M"},
				"enabled":     map[string]any{"type": "boolean", "default": false},
			},
		},
	}
	config, configError, unconfigured := resolveSafeguardConfig("example", manifest, map[string]any{
		"reservation": "768M",
		"enabled":     true,
	}, true)
	if configError != "" {
		t.Fatal(configError)
	}
	if unconfigured != "" {
		t.Fatalf("unexpected unconfigured reason: %q", unconfigured)
	}
	if got, ok := config["reservation"].(string); !ok || got != "768M" {
		t.Fatalf("reservation = %#v, want 768M", config["reservation"])
	}
	if got, ok := config["enabled"].(bool); !ok || !got {
		t.Fatalf("enabled = %#v, want true", config["enabled"])
	}
}

func TestResolveSafeguardConfigRejectsInvalidParameterWithoutFallback(t *testing.T) {
	manifest := hostreqkit.SafeguardManifest{
		Name: "example",
		Config: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"experience": map[string]any{"type": "string", "enum": []any{"login-screen", "observe-only"}, "default": "observe-only"},
			},
		},
	}
	config, configError, _ := resolveSafeguardConfig("example", manifest, map[string]any{"experience": "direct-desktop"}, true)
	if config != nil || !strings.Contains(configError, "experience") {
		t.Fatalf("config=%#v error=%q; want rejection naming experience", config, configError)
	}
}

func TestResolveSafeguardConfigTreatsUnconfiguredOptionalAsNotApplicable(t *testing.T) {
	manifest := hostreqkit.SafeguardManifest{
		Name: "example",
		Config: map[string]any{
			"type":     "object",
			"required": []any{"target"},
			"properties": map[string]any{
				"target": map[string]any{"type": "string", "minLength": float64(1)},
			},
		},
	}
	config, configError, unconfigured := resolveSafeguardConfig("example", manifest, nil, false)
	if configError != "" || config == nil || !strings.Contains(unconfigured, "target") {
		t.Fatalf("config=%#v error=%q unconfigured=%q", config, configError, unconfigured)
	}
}

func TestResolveSafeguardConfigKeepsMissingRequiredConfigInvalid(t *testing.T) {
	manifest := hostreqkit.SafeguardManifest{
		Name: "example",
		Config: map[string]any{
			"type":     "object",
			"required": []any{"target"},
			"properties": map[string]any{
				"target": map[string]any{"type": "string", "minLength": float64(1)},
			},
		},
	}
	config, configError, unconfigured := resolveSafeguardConfig("example", manifest, nil, true)
	if config != nil || unconfigured != "" || !strings.Contains(configError, "target") {
		t.Fatalf("config=%#v error=%q unconfigured=%q", config, configError, unconfigured)
	}
}

func TestResolvedRequirementConfigAccessorsAreTyped(t *testing.T) {
	requirement := ResolvedRequirement{Config: map[string]any{
		"text":     "value",
		"count":    float64(3),
		"enabled":  true,
		"fraction": 1.5,
	}}
	if got, ok := requirement.ConfigString("text"); !ok || got != "value" {
		t.Fatalf("ConfigString = %q, %v", got, ok)
	}
	if got, ok := requirement.ConfigInt("count"); !ok || got != 3 {
		t.Fatalf("ConfigInt = %d, %v", got, ok)
	}
	if got, ok := requirement.ConfigBool("enabled"); !ok || !got {
		t.Fatalf("ConfigBool = %v, %v", got, ok)
	}
	if _, ok := requirement.ConfigInt("fraction"); ok {
		t.Fatal("fractional ConfigInt unexpectedly accepted")
	}
}

func TestResolveSafeguardCarriesOperatorConfigToRequirement(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	state := `{"version":"1.0.0","updated_at":"2026-08-05T17:24:03Z","host_safeguards":{"netconsole":{"config":{"target":"6666@10.0.0.5/dev,6666@10.0.0.6/00:11:22:33:44:55"}}}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "operator-state.json"), []byte(state), 0o644); err != nil {
		t.Fatal(err)
	}
	requirement, err := ResolveSafeguard(root, "netconsole", "linux")
	if err != nil {
		t.Fatal(err)
	}
	if requirement.ConfigError != "" {
		t.Fatalf("ConfigError = %q", requirement.ConfigError)
	}
	if got, ok := requirement.ConfigString("target"); !ok || got == "" {
		t.Fatalf("target config = %q, %v", got, ok)
	}
}

func TestResolveHostRequirementsIsReadOnlyAccessor(t *testing.T) {
	root := t.TempDir()
	testscenario.WriteProjectService(t, root, scenario.ServiceManifest{
		Service: scenario.ServiceMetadata{Name: "vrooli"},
		HostSafeguards: []hostreqspec.Declaration{{
			Name:     "remote_desktop_access",
			Required: false,
			Reason:   "test remote desktop resolution",
			When:     []string{"setup"},
		}},
	})

	resolution, err := ResolveHostRequirements(root, t.TempDir(), ResolveOptions{
		Environment: "development",
		When:        "setup",
		Resources:   "none",
		Scenarios:   "none",
		Platform:    "linux",
	})
	if err != nil {
		t.Fatalf("ResolveHostRequirements: %v", err)
	}
	requirement := findRequirement(t, resolution.Safeguards, "remote_desktop_access")
	if got, ok := requirement.ConfigString("experience"); !ok || got != "observe-only" {
		t.Fatalf("experience config = %q, %v; want observe-only", got, ok)
	}
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "operator-state.json")); !os.IsNotExist(err) {
		t.Fatalf("read-only accessor created operator state: err=%v", err)
	}
}
