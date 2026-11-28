package generation

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestNativeExtensionAdmission(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		valid     bool
	}{
		{"vanilla", `{}`, true},
		{"presentation", `{"native_extension":{"version":1,"module":"presentation","permissions":["window.presentation"],"platforms":["linux"]}}`, true},
		{"future", `{"native_extension":{"version":3,"module":"presentation","permissions":["window.presentation"],"platforms":["linux"]}}`, false},
		{"import", `{"native_extension":{"version":1,"module":"presentation","entrypoint":"/tmp/code.js","permissions":["window.presentation"],"platforms":["linux"]}}`, false},
		{"permission", `{"native_extension":{"version":1,"module":"presentation","permissions":["filesystem"],"platforms":["linux"]}}`, false},
		{"target", `{"native_extension":{"version":1,"module":"presentation","permissions":["window.presentation"],"platforms":["mac"]}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := DesktopConfig{Framework: "electron", Platforms: []string{"linux"}}
			err := json.Unmarshal([]byte(tc.raw), &config)
			if err == nil {
				err = config.ValidateNativeExtension()
			}
			if (err == nil) != tc.valid {
				t.Fatalf("valid=%v, err=%v", tc.valid, err)
			}
		})
	}
}

func TestNativeExtensionCompatibilityErrorIsTyped(t *testing.T) {
	config := DesktopConfig{
		Framework: "electron",
		Platforms: []string{"windows-arm64"},
		NativeExtension: &NativeExtension{
			Version:     1,
			Module:      "presentation",
			Permissions: []string{"window.presentation"},
			Platforms:   []string{"linux"},
		},
	}
	var compatibility *NativeExtensionCompatibilityError
	if err := config.ValidateNativeExtension(); !errors.As(err, &compatibility) {
		t.Fatalf("expected typed compatibility error, got %v", err)
	}
	if compatibility.Code != "NATIVE_EXTENSION_TARGET_UNSUPPORTED" || compatibility.Target != "windows-arm64" || compatibility.Field != "platforms" {
		t.Fatalf("unexpected compatibility error: %+v", compatibility)
	}
}

func TestNativeExtensionCanonicalPipelineTargets(t *testing.T) {
	config := DesktopConfig{Framework: "electron", NativeExtension: &NativeExtension{Version: 1, Module: "presentation", Permissions: []string{"window.presentation"}, Platforms: []string{"linux", "mac", "win"}}}
	for _, target := range []string{"linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64", "windows-amd64", "windows-arm64"} {
		config.Platforms = []string{target}
		if err := config.ValidateNativeExtension(); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
	}
	for _, target := range []string{"linux-unknown", "linux-amd64-extra", "unknown-amd64"} {
		config.Platforms = []string{target}
		if err := config.ValidateNativeExtension(); err == nil {
			t.Fatalf("accepted %s", target)
		}
	}
}

func TestNativeExtensionGovernedHelperProvider(t *testing.T) {
	config := DesktopConfig{Framework: "electron", Platforms: []string{"linux"}, NativeExtension: &NativeExtension{
		Version: 3, Module: "presentation", Permissions: []string{"window.presentation", "global-shortcut", "desktop.context"}, Platforms: []string{"linux"}, ActivationShortcut: "Control+Shift+Space",
		HelperProviders: []HelperProvider{{Owner: "device-control", Capability: "desktop.session"}},
	}}
	if err := config.ValidateNativeExtension(); err != nil {
		t.Fatalf("governed helper provider should be admitted: %v", err)
	}
	config.NativeExtension.HelperProviders[0].Capability = "arbitrary.exec"
	if err := config.ValidateNativeExtension(); err == nil {
		t.Fatal("arbitrary helper capability must be refused")
	}
}

func TestNativeActivationContract(t *testing.T) {
	config := DesktopConfig{Framework: "electron", Platforms: []string{"linux-amd64"}, NativeExtension: &NativeExtension{Version: 2, Module: "presentation", Permissions: []string{"window.presentation", "global-shortcut"}, Platforms: []string{"linux"}, ActivationShortcut: "CommandOrControl+Shift+Space"}}
	if err := config.ValidateNativeExtension(); err != nil {
		t.Fatal(err)
	}
	config.NativeExtension.Version = 1
	if err := config.ValidateNativeExtension(); err == nil {
		t.Fatal("v1 granted shortcut")
	}
	config.NativeExtension.Version = 2
	config.NativeExtension.ActivationShortcut = "Alt+Alt+Space"
	if err := config.ValidateNativeExtension(); err == nil {
		t.Fatal("duplicate modifiers accepted")
	}
}

func TestNativeContextPermission(t *testing.T) {
	c := DesktopConfig{Framework: "electron", Platforms: []string{"linux"}, NativeExtension: &NativeExtension{Version: 3, Module: "presentation", Permissions: []string{"window.presentation", "global-shortcut", "desktop.context"}, Platforms: []string{"linux"}, ActivationShortcut: "Control+Shift+Space"}}
	if err := c.ValidateNativeExtension(); err != nil {
		t.Fatal(err)
	}
	c.NativeExtension.Version = 2
	if c.ValidateNativeExtension() == nil {
		t.Fatal("older contract granted context")
	}
	c.NativeExtension.Version = 3
	c.NativeExtension.Permissions = []string{"window.presentation", "global-shortcut"}
	if c.ValidateNativeExtension() == nil {
		t.Fatal("missing context permission accepted")
	}
}
