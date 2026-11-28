package resources

import (
	"maps"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpenRGBConstructorEnvironmentPreservesLegacyLinuxAndNativeIsolation(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	for _, goos := range []string{"linux", "darwin", "windows"} {
		t.Run(goos, func(t *testing.T) {
			current := map[string]string{"XDG_CONFIG_HOME": dir, "APPDATA": dir, "QT_QPA_PLATFORM": "offscreen"}
			if err := verifyOpenRGBConstructorEnvironment(goos, dir, current); err != nil {
				t.Fatal(err)
			}
			legacy := maps.Clone(current)
			delete(legacy, "APPDATA")
			err := verifyOpenRGBConstructorEnvironment(goos, dir, legacy)
			if goos == "linux" && err != nil {
				t.Fatalf("qualified pre-native Linux constructor environment rejected: %v", err)
			}
			if goos != "linux" && err == nil {
				t.Fatal("native platform lost existing required APPDATA isolation")
			}
			for _, key := range []string{"XDG_CONFIG_HOME", "APPDATA", "QT_QPA_PLATFORM"} {
				bad := maps.Clone(current)
				bad[key] = "ambient"
				if err := verifyOpenRGBConstructorEnvironment(goos, dir, bad); err == nil {
					t.Fatalf("explicit non-isolated %s accepted", key)
				}
			}
		})
	}
	if err := verifyOpenRGBConstructorEnvironment("unknown", dir, map[string]string{"XDG_CONFIG_HOME": dir, "APPDATA": dir, "QT_QPA_PLATFORM": "offscreen"}); err == nil {
		t.Fatal("unknown platform admitted")
	}
}
func TestOpenRGBLinuxLegacyEndpointPreservesAllOtherLaunchGuards(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("old qualified profile is Linux")
	}
	for _, name := range []string{"legacy", "current", "ambientAPPDATA", "missingXDG", "ambientXDG", "wrongQt", "wildcard", "changedArgs"} {
		t.Run(name, func(t *testing.T) {
			_, manifest, _ := openRGBHeadlessFixture(t)
			service := *manifest.ManagedService
			manifest.ManagedService = &service
			service.Environment = maps.Clone(service.Environment)
			delete(service.Environment, "APPDATA")
			switch name {
			case "current":
				service.Environment["APPDATA"] = "${RESOURCE_CONFIG_DIR}"
			case "ambientAPPDATA":
				service.Environment["APPDATA"] = "/ambient"
			case "missingXDG":
				delete(service.Environment, "XDG_CONFIG_HOME")
			case "ambientXDG":
				service.Environment["XDG_CONFIG_HOME"] = "/ambient"
			case "wrongQt":
				service.Environment["QT_QPA_PLATFORM"] = "desktop"
			case "wildcard", "changedArgs":
				service.Arguments = append([]string(nil), service.ArgumentsFor(runtime.GOOS, runtime.GOARCH)...)
				service.ArgumentsByPlatform = nil
				if name == "wildcard" {
					service.Arguments[2] = "0.0.0.0"
				} else {
					service.Arguments = append(service.Arguments, "--gui")
				}
			}
			_, _, err := openRGBHeadlessEndpoint(manifest)
			if name == "legacy" || name == "current" {
				if err != nil {
					t.Fatalf("qualified Linux %s rejected: %v", name, err)
				}
			} else if err == nil {
				t.Fatalf("unsafe %s accepted", name)
			}
		})
	}
}
