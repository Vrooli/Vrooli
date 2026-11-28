package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRGBUsesPinnedIsolatedDetectorAllowlist(t *testing.T) {
	manifestBytes, err := os.ReadFile(filepath.Join("..", "..", "resources", "openrgb", "resource.json"))
	if err != nil {
		t.Fatalf("read OpenRGB manifest: %v", err)
	}
	var manifest struct {
		Platforms      map[string]string `json:"platforms"`
		ManagedService struct {
			Acquisition struct {
				Targets []struct {
					When struct {
						OS   string `json:"os"`
						Arch string `json:"arch"`
					} `json:"when"`
					Unsupported string `json:"unsupported"`
					SHA256      string `json:"sha256"`
				} `json:"targets"`
			} `json:"acquisition"`
			Artifact struct {
				Version string `json:"version"`
			} `json:"artifact"`
			Arguments   []string          `json:"arguments"`
			Environment map[string]string `json:"environment"`
			Config      struct {
				Path        string `json:"path"`
				Content     string `json:"content"`
				ContentFile string `json:"content_file"`
			} `json:"config"`
		} `json:"managed_service"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatalf("decode OpenRGB manifest: %v", err)
	}
	if manifest.ManagedService.Artifact.Version != "1.1" {
		t.Fatalf("OpenRGB version = %q, want config-qualified 1.1", manifest.ManagedService.Artifact.Version)
	}
	if manifest.ManagedService.Config.Path != "OpenRGB/OpenRGB.json" {
		t.Fatalf("config path = %q, want resource-owned OpenRGB/OpenRGB.json", manifest.ManagedService.Config.Path)
	}
	if !containsOrdered(manifest.ManagedService.Arguments, "--config", "${RESOURCE_CONFIG_DIR}/OpenRGB") {
		t.Fatalf("arguments do not route config to RESOURCE_CONFIG_DIR: %q", manifest.ManagedService.Arguments)
	}
	if manifest.ManagedService.Environment["XDG_CONFIG_HOME"] != "${RESOURCE_CONFIG_DIR}" {
		t.Fatalf("XDG_CONFIG_HOME = %q, want resource-owned config root to isolate constructor writes", manifest.ManagedService.Environment["XDG_CONFIG_HOME"])
	}

	if manifest.ManagedService.Environment["APPDATA"] != "${RESOURCE_CONFIG_DIR}" {
		t.Fatal("Windows constructor must use resource-owned APPDATA")
	}
	native := map[string]bool{}
	for _, target := range manifest.ManagedService.Acquisition.Targets {
		key := target.When.OS + "-" + target.When.Arch
		if key == "windows-arm64" {
			if target.Unsupported == "" {
				t.Fatal("Windows ARM64 must remain explicitly unsupported")
			}
			continue
		}
		goos := target.When.OS
		if goos == "macos" {
			goos = "darwin"
		}
		profile, err := openRGBReleaseProfileFor(goos, target.When.Arch)
		if err != nil || profile.validate() != nil || target.SHA256 != profile.EnvelopeSHA256 || target.Unsupported != "" {
			t.Fatalf("native release profile %s is incomplete", key)
		}
		native[key] = true
	}
	if len(native) != 5 {
		t.Fatalf("native target count=%d, want five exact supported release profiles", len(native))
	}

	if manifest.ManagedService.Config.Content != "" || manifest.ManagedService.Config.ContentFile != "config/detectors-release_1.1.json" {
		t.Fatalf("config source = content %t, file %q; want versioned resource-owned asset", manifest.ManagedService.Config.Content != "", manifest.ManagedService.Config.ContentFile)
	}
	configBytes, err := os.ReadFile(filepath.Join("..", "..", "resources", "openrgb", filepath.FromSlash(manifest.ManagedService.Config.ContentFile)))
	if err != nil {
		t.Fatalf("read pinned detector asset: %v", err)
	}
	var settings struct {
		Detectors struct {
			Detectors map[string]bool `json:"detectors"`
		} `json:"Detectors"`
	}
	if err := json.Unmarshal(configBytes, &settings); err != nil {
		t.Fatalf("decode isolated OpenRGB detector settings: %v", err)
	}
	if len(settings.Detectors.Detectors) != 1948 {
		t.Fatalf("pinned detector map has %d entries, want all static 1.1 registrations (1948)", len(settings.Detectors.Detectors))
	}
	for _, name := range []string{
		"ASUS Aura Motherboard",
		"ASUS Aura SMBus Motherboard",
		"ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC",
	} {
		if settings.Detectors.Detectors[name] {
			t.Errorf("host-specific detector %q is enabled in the shared catalog", name)
		}
	}
	if settings.Detectors.Detectors["ENE SMBus DRAM"] {
		t.Error("write/remap-capable ENE SMBus DRAM detector must stay disabled")
	}
	enabled := 0
	for _, value := range settings.Detectors.Detectors {
		if value {
			enabled++
		}
	}
	if enabled != 0 {
		t.Errorf("shared detector catalog enables %d detectors, want none before explicit machine-local selection", enabled)
	}
}

func containsOrdered(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if values[index] == first && values[index+1] == second {
			return true
		}
	}
	return false
}
