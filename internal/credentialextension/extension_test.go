package credentialextension

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func executableHost(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "secrets-manager-native-host")
	if err := os.WriteFile(path, []byte("host"), 0755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInstallStatusUninstallLinuxAllBrowsers(t *testing.T) {
	home := t.TempDir()
	host := executableHost(t)
	opts := Options{HostPath: host, ExtensionID: "abcdefghijklmnopabcdefghijklmnop", Browser: "all", OS: "linux", HomeDir: home}

	installed, err := Install(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !installed.Changed || len(installed.Targets) != 3 {
		t.Fatalf("install result: %+v", installed)
	}
	for _, target := range installed.Targets {
		data, err := os.ReadFile(target.ManifestPath)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "password") || strings.Contains(string(data), "token") {
			t.Fatalf("manifest contains credential-like material: %s", data)
		}
		var manifest nativeManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.Path != host || manifest.Name != HostName {
			t.Fatalf("manifest: %+v", manifest)
		}
	}

	status, err := Status(opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range status.Targets {
		if !target.ManifestInstalled || !target.RegistryConfigured {
			t.Fatalf("status target: %+v", target)
		}
	}

	second, err := Install(opts)
	if err != nil {
		t.Fatal(err)
	}
	if second.Changed {
		t.Fatalf("idempotent install changed files: %+v", second)
	}

	removed, err := Uninstall(opts)
	if err != nil {
		t.Fatal(err)
	}
	if !removed.Changed {
		t.Fatal("uninstall did not report a change")
	}
	for _, target := range removed.Targets {
		if _, err := os.Stat(target.ManifestPath); !os.IsNotExist(err) {
			t.Fatalf("manifest still exists at %s", target.ManifestPath)
		}
	}
}

func TestFirefoxManifestUsesAllowedExtensions(t *testing.T) {
	host := executableHost(t)
	result, err := Install(Options{HostPath: host, ExtensionID: "vrooli.example.test", Browser: "firefox", OS: "windows", HomeDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.Targets[0].ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest nativeManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.AllowedExtensions) != 1 || manifest.AllowedExtensions[0] != "vrooli.example.test" || len(manifest.AllowedOrigins) != 0 {
		t.Fatalf("firefox manifest: %+v", manifest)
	}
	if result.Targets[0].RegistryKey == "" {
		t.Fatal("windows target has no registry key")
	}
}

func TestValidationRejectsUnsafeInputs(t *testing.T) {
	host := executableHost(t)
	tests := []Options{
		{HostPath: "relative", ExtensionID: "good", OS: "linux", HomeDir: t.TempDir()},
		{HostPath: host, ExtensionID: "bad/id", OS: "linux", HomeDir: t.TempDir()},
		{HostPath: host, ExtensionID: "good", Browser: "safari", OS: "linux", HomeDir: t.TempDir()},
	}
	for _, opts := range tests {
		if _, err := Install(opts); err == nil {
			t.Fatalf("expected validation error for %+v", opts)
		}
	}
}
