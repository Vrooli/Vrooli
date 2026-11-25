package system

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writePackageConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "package.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	return path
}

func TestInspectBuildPrerequisitesUsesGeneratedTemplateDefault(t *testing.T) {
	for _, tc := range []struct {
		platform string
		want     string
	}{
		{platform: "linux", want: "AppImage"},
		{platform: "win", want: "nsis"},
		{platform: "mac", want: "zip"},
	} {
		t.Run(tc.platform, func(t *testing.T) {
			report, err := inspectBuildPrerequisites(tc.platform, tc.platform, func(string) (string, error) {
				t.Fatal("native default must not need an additional tool in this case")
				return "", nil
			})
			if err != nil {
				t.Fatalf("inspectBuildPrerequisites() error: %v", err)
			}
			if report.Status != "ready" || report.PackageConfigStatus != "template-default" || report.PackageConfigSource != "generated-template" || !reflect.DeepEqual(report.PackageFormats, []string{tc.want}) {
				t.Fatalf("report = %+v, want generated default %q", report, tc.want)
			}
		})
	}
}

func TestInspectBuildPrerequisitesUsesPerAppDEBOverride(t *testing.T) {
	config := writePackageConfig(t, `{"build":{"linux":{"target":[{"target":"deb","arch":["x64"]}]}}}`)
	available := map[string]bool{"fpm": true}
	var lookups []string
	report, err := inspectBuildPrerequisites("linux", "linux", func(name string) (string, error) {
		lookups = append(lookups, name)
		if available[name] {
			return "/usr/bin/" + name, nil
		}
		return "", errors.New("not found")
	}, config)
	if err != nil {
		t.Fatalf("inspectBuildPrerequisites() error: %v", err)
	}
	if report.Status != "blocked" || report.PackageConfigStatus != "ready" || report.PackageConfigSource != "selected-package-json" || !reflect.DeepEqual(report.PackageFormats, []string{"deb"}) {
		t.Fatalf("report = %+v, want blocked selected DEB override", report)
	}
	if !reflect.DeepEqual(lookups, []string{"fpm", "ruby"}) {
		t.Fatalf("lookups = %v, want fpm and ruby", lookups)
	}
	if len(report.Checks) != 2 || report.Checks[0].Status != "present" || report.Checks[1].Status != "missing" || !report.Checks[1].Required {
		t.Fatalf("checks = %+v, want fpm present and required Ruby missing", report.Checks)
	}
}

func TestInspectBuildPrerequisitesRecognizesAppImageCaseInsensitively(t *testing.T) {
	config := writePackageConfig(t, `{"build":{"linux":{"target":"AppImage"}}}`)
	report, err := inspectBuildPrerequisites("linux", "linux", func(name string) (string, error) {
		t.Fatalf("AppImage must not trigger lookup for %q", name)
		return "", nil
	}, config)
	if err != nil {
		t.Fatalf("inspectBuildPrerequisites() error: %v", err)
	}
	if report.Status != "ready" || !reflect.DeepEqual(report.PackageFormats, []string{"AppImage"}) {
		t.Fatalf("report = %+v, want ready AppImage override", report)
	}
}

func TestInspectBuildPrerequisitesRetainsUnsupportedAndUnavailableOverrides(t *testing.T) {
	t.Run("unsupported format", func(t *testing.T) {
		config := writePackageConfig(t, `{"build":{"linux":{"target":"rpm"}}}`)
		report, err := inspectBuildPrerequisites("linux", "linux", func(string) (string, error) {
			t.Fatal("unsupported RPM host rule must not be guessed")
			return "", nil
		}, config)
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "limited" || report.PackageConfigStatus != "ready" || !reflect.DeepEqual(report.PackageFormats, []string{"rpm"}) || len(report.Limitations) != 1 || !strings.Contains(report.Limitations[0], "no validated host-prerequisite rule") {
			t.Fatalf("report = %+v, want explicit unknown RPM rule", report)
		}
	})

	t.Run("missing config", func(t *testing.T) {
		report, err := inspectBuildPrerequisites("linux", "linux", func(string) (string, error) {
			t.Fatal("unavailable config must not infer format prerequisites")
			return "", nil
		}, filepath.Join(t.TempDir(), "missing-package.json"))
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "limited" || report.PackageConfigStatus != "unavailable" || len(report.PackageFormats) != 0 || len(report.Limitations) != 1 {
			t.Fatalf("report = %+v, want unavailable config retained", report)
		}
	})

	t.Run("implicit target", func(t *testing.T) {
		config := writePackageConfig(t, `{"build":{"linux":{}}}`)
		report, err := inspectBuildPrerequisites("linux", "linux", func(string) (string, error) {
			t.Fatal("implicit target must not be guessed")
			return "", nil
		}, config)
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "limited" || report.PackageConfigStatus != "unconfigured" || len(report.PackageFormats) != 0 {
			t.Fatalf("report = %+v, want unconfigured target retained", report)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		config := writePackageConfig(t, `not json`)
		report, err := inspectBuildPrerequisites("linux", "linux", func(string) (string, error) {
			t.Fatal("invalid config must not infer format prerequisites")
			return "", nil
		}, config)
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "limited" || report.PackageConfigStatus != "invalid" || len(report.PackageFormats) != 0 {
			t.Fatalf("report = %+v, want invalid config retained", report)
		}
	})
}

func TestInspectBuildPrerequisitesChecksFormatSpecificHostTools(t *testing.T) {
	t.Run("Windows NSIS on Linux", func(t *testing.T) {
		config := writePackageConfig(t, `{"build":{"win":{"target":[{"target":"nsis","arch":["x64"]}]}}}`)
		report, err := inspectBuildPrerequisites("win", "linux", func(name string) (string, error) {
			if name != "wine" {
				t.Fatalf("unexpected tool lookup %q", name)
			}
			return "/usr/bin/wine", nil
		}, config)
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "ready" || len(report.Checks) != 1 || report.Checks[0].Tool != "wine" || report.Checks[0].Status != "present" {
			t.Fatalf("report = %+v, want ready NSIS with Wine", report)
		}
	})

	t.Run("macOS ZIP on Linux", func(t *testing.T) {
		config := writePackageConfig(t, `{"build":{"mac":{"target":"zip"}}}`)
		report, err := inspectBuildPrerequisites("mac", "linux", func(name string) (string, error) {
			if name != "docker" {
				t.Fatalf("unexpected tool lookup %q", name)
			}
			return "/usr/bin/docker", nil
		}, config)
		if err != nil {
			t.Fatalf("inspectBuildPrerequisites() error: %v", err)
		}
		if report.Status != "ready" || len(report.Checks) != 1 || report.Checks[0].Tool != "docker" {
			t.Fatalf("report = %+v, want Docker check for documented cross-host ZIP", report)
		}
	})
}

func TestInspectBuildPrerequisitesRetainsCrossHostUncertainty(t *testing.T) {
	report, err := inspectBuildPrerequisites("linux", "windows", func(string) (string, error) {
		t.Fatal("cross-host Docker/WSL requirements must not be guessed")
		return "", nil
	})
	if err != nil {
		t.Fatalf("inspectBuildPrerequisites() error: %v", err)
	}
	if report.Status != "limited" || len(report.Limitations) != 1 {
		t.Fatalf("report = %+v, want a limited result with a retained caveat", report)
	}
}

func TestInspectBuildPrerequisitesRejectsUnknownPlatform(t *testing.T) {
	if _, err := inspectBuildPrerequisites("all", "linux", func(string) (string, error) { return "", nil }); err == nil {
		t.Fatal("unknown target should be rejected")
	}
}
