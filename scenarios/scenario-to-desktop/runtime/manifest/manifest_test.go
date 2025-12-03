package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// LoadManifest Tests
// =============================================================================

func TestLoadManifest_Success(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "bundle.json")

	content := `{
		"schema_version": "desktop.v0.1",
		"target": "desktop",
		"app": {"name": "Test App", "version": "1.0.0"},
		"ipc": {"host": "127.0.0.1", "port": 47710, "auth_token_path": "runtime/token"},
		"telemetry": {"file": "telemetry.jsonl"},
		"services": [{
			"id": "api",
			"binaries": {"linux-x64": {"path": "bin/api"}},
			"health": {"type": "tcp"},
			"readiness": {"type": "tcp"}
		}]
	}`
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	m, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}

	if m.App.Name != "Test App" {
		t.Errorf("LoadManifest() App.Name = %q, want %q", m.App.Name, "Test App")
	}
	if m.App.Version != "1.0.0" {
		t.Errorf("LoadManifest() App.Version = %q, want %q", m.App.Version, "1.0.0")
	}
	if len(m.Services) != 1 {
		t.Errorf("LoadManifest() len(Services) = %d, want 1", len(m.Services))
	}
}

func TestLoadManifest_FileNotFound(t *testing.T) {
	_, err := LoadManifest("/nonexistent/bundle.json")
	if err == nil {
		t.Fatal("LoadManifest() expected error for missing file")
	}
}

func TestLoadManifest_InvalidJSON(t *testing.T) {
	tmp := t.TempDir()
	manifestPath := filepath.Join(tmp, "bundle.json")

	if err := os.WriteFile(manifestPath, []byte("{ invalid json }"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadManifest(manifestPath)
	if err == nil {
		t.Fatal("LoadManifest() expected error for invalid JSON")
	}
}

// =============================================================================
// Validate Tests
// =============================================================================

func TestManifestValidateEnforcesStructureAndPlatforms(t *testing.T) {
	makeBaseManifest := func() Manifest {
		return Manifest{
			SchemaVersion: "desktop.v0.1",
			Target:        "desktop",
			App:           App{Name: "demo", Version: "1.0.0"},
			IPC:           IPC{Host: "127.0.0.1", Port: 47710},
			Services: []Service{{
				ID: "api",
				Binaries: map[string]Binary{
					"windows-x64": {Path: "bin/api.exe"},
				},
				Health:    HealthCheck{Type: "tcp", PortName: "http"},
				Readiness: ReadinessCheck{Type: "tcp", PortName: "http"},
			}},
		}
	}

	t.Run("accepts platform aliases when binaries exist only under alias key", func(t *testing.T) {
		m := makeBaseManifest()
		m.Services[0].Binaries = map[string]Binary{
			"win-x64": {Path: "bin/api.exe"},
		}

		if err := m.Validate("windows", "amd64"); err != nil {
			t.Fatalf("Validate() with alias binary returned error: %v", err)
		}
	})

	t.Run("rejects missing binaries for target platform", func(t *testing.T) {
		m := makeBaseManifest()
		m.Services[0].Binaries = map[string]Binary{
			"linux-x64": {Path: "bin/api"},
		}

		err := m.Validate("windows", "amd64")
		if err == nil || err.Error() != "service api missing binary for platform windows-x64" {
			t.Fatalf("Validate() error = %v, want missing binary message", err)
		}
	})

	t.Run("requires health and readiness definitions per service", func(t *testing.T) {
		m := makeBaseManifest()
		m.Services[0].Health = HealthCheck{}

		err := m.Validate("windows", "amd64")
		if err == nil || err.Error() != "service api requires health and readiness definitions" {
			t.Fatalf("Validate() error = %v, want health/readiness requirement", err)
		}
	})
}

func TestValidate_MissingSchemaVersion(t *testing.T) {
	m := Manifest{
		Target: "desktop",
		App:    App{Name: "demo", Version: "1.0.0"},
		IPC:    IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil || err.Error() != "schema_version missing" {
		t.Fatalf("Validate() error = %v, want schema_version missing", err)
	}
}

func TestValidate_WrongTarget(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "server",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for wrong target")
	}
}

func TestValidate_MissingAppName(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for missing app.name")
	}
}

func TestValidate_MissingAppVersion(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for missing app.version")
	}
}

func TestValidate_MissingIPCHost(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for missing ipc.host")
	}
}

func TestValidate_MissingIPCPort(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1"},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for missing ipc.port")
	}
}

func TestValidate_EmptyServices(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services:      []Service{},
	}

	err := m.Validate("linux", "amd64")
	if err == nil || err.Error() != "services must not be empty" {
		t.Fatalf("Validate() error = %v, want services must not be empty", err)
	}
}

func TestValidate_MissingServiceID(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			Binaries:  map[string]Binary{"linux-x64": {Path: "bin/api"}},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil || err.Error() != "service.id is required" {
		t.Fatalf("Validate() error = %v, want service.id is required", err)
	}
}

func TestValidate_MissingServiceBinaries(t *testing.T) {
	m := Manifest{
		SchemaVersion: "desktop.v0.1",
		Target:        "desktop",
		App:           App{Name: "demo", Version: "1.0.0"},
		IPC:           IPC{Host: "127.0.0.1", Port: 47710},
		Services: []Service{{
			ID:        "api",
			Binaries:  map[string]Binary{},
			Health:    HealthCheck{Type: "tcp"},
			Readiness: ReadinessCheck{Type: "tcp"},
		}},
	}

	err := m.Validate("linux", "amd64")
	if err == nil {
		t.Fatal("Validate() expected error for missing binaries")
	}
}

// =============================================================================
// PlatformKey Tests
// =============================================================================

func TestPlatformKey(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"linux", "amd64", "linux-x64"},
		{"linux", "arm64", "linux-arm64"},
		{"darwin", "amd64", "darwin-x64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"windows", "amd64", "windows-x64"},
		{"windows", "386", "windows-386"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			got := PlatformKey(tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("PlatformKey(%q, %q) = %q, want %q", tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestPlatformKeys(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   []string
	}{
		{"linux", "amd64", []string{"linux-x64"}},
		{"darwin", "amd64", []string{"darwin-x64", "mac-x64"}},
		{"darwin", "arm64", []string{"darwin-arm64", "mac-arm64"}},
		{"windows", "amd64", []string{"windows-x64", "win-x64"}},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			got := PlatformKeys(tt.goos, tt.goarch)
			if len(got) != len(tt.want) {
				t.Errorf("PlatformKeys(%q, %q) = %v, want %v", tt.goos, tt.goarch, got, tt.want)
				return
			}
			for i, key := range got {
				if key != tt.want[i] {
					t.Errorf("PlatformKeys(%q, %q)[%d] = %q, want %q", tt.goos, tt.goarch, i, key, tt.want[i])
				}
			}
		})
	}
}

func TestPlatformAlias(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"windows-x64", "win-x64"},
		{"win-x64", "windows-x64"},
		{"darwin-arm64", "mac-arm64"},
		{"mac-arm64", "darwin-arm64"},
		{"linux-x64", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := platformAlias(tt.key)
			if got != tt.want {
				t.Errorf("platformAlias(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// =============================================================================
// ResolveBinary Tests
// =============================================================================

func TestResolveBinary(t *testing.T) {
	m := &Manifest{}
	svc := Service{
		ID: "api",
		Binaries: map[string]Binary{
			"linux-x64":   {Path: "bin/api-linux"},
			"darwin-x64":  {Path: "bin/api-darwin"},
			"windows-x64": {Path: "bin/api.exe"},
		},
	}

	// This test is platform-dependent, so just verify it returns something
	bin, found := m.ResolveBinary(svc)
	if !found {
		t.Skip("No binary for current platform, skipping")
	}
	if bin.Path == "" {
		t.Error("ResolveBinary() returned empty path")
	}
}

func TestResolveBinary_NotFound(t *testing.T) {
	m := &Manifest{}
	svc := Service{
		ID: "api",
		Binaries: map[string]Binary{
			"freebsd-amd64": {Path: "bin/api-freebsd"},
		},
	}

	_, found := m.ResolveBinary(svc)
	// May or may not find depending on platform, just verify no panic
	_ = found
}

func TestResolveBinary_UsesAlias(t *testing.T) {
	m := &Manifest{}

	// If on darwin, should find mac-arm64 as alias
	svc := Service{
		ID: "api",
		Binaries: map[string]Binary{
			"mac-arm64": {Path: "bin/api-mac"},
			"win-x64":   {Path: "bin/api.exe"},
		},
	}

	// This is platform-dependent
	_, _ = m.ResolveBinary(svc)
	// Just verify no panic
}

// =============================================================================
// GPURequirement Tests
// =============================================================================

func TestGPURequirement(t *testing.T) {
	tests := []struct {
		name    string
		service Service
		want    string
	}{
		{
			name:    "nil GPU",
			service: Service{ID: "api"},
			want:    "",
		},
		{
			name: "empty requirement",
			service: Service{
				ID:  "api",
				GPU: &GPURequirements{},
			},
			want: "",
		},
		{
			name: "required GPU",
			service: Service{
				ID:  "api",
				GPU: &GPURequirements{Requirement: "required"},
			},
			want: "required",
		},
		{
			name: "optional GPU with whitespace",
			service: Service{
				ID:  "api",
				GPU: &GPURequirements{Requirement: "  optional_with_cpu_fallback  "},
			},
			want: "optional_with_cpu_fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.service.GPURequirement()
			if got != tt.want {
				t.Errorf("GPURequirement() = %q, want %q", got, tt.want)
			}
		})
	}
}

// =============================================================================
// ResolvePath Tests
// =============================================================================

func TestResolvePathNormalizesWindowsSeparators(t *testing.T) {
	bundleDir := "/opt/app/bundle"
	rel := `resources\playwright\chromium\chrome.exe`

	got := ResolvePath(bundleDir, rel)
	want := filepath.Join(bundleDir, "resources", "playwright", "chromium", "chrome.exe")

	if got != want {
		t.Fatalf("ResolvePath() = %q, want %q", got, want)
	}
}

func TestResolvePath_UnixSeparators(t *testing.T) {
	bundleDir := "/opt/app/bundle"
	rel := "resources/data/config.json"

	got := ResolvePath(bundleDir, rel)
	want := filepath.Join(bundleDir, "resources", "data", "config.json")

	if got != want {
		t.Errorf("ResolvePath() = %q, want %q", got, want)
	}
}

func TestResolvePath_EmptyRel(t *testing.T) {
	bundleDir := "/opt/app/bundle"

	got := ResolvePath(bundleDir, "")
	want := filepath.Join(bundleDir, ".")

	if got != want {
		t.Errorf("ResolvePath() = %q, want %q", got, want)
	}
}

func TestResolvePath_DotPath(t *testing.T) {
	bundleDir := "/opt/app/bundle"
	rel := "./config/app.json"

	got := ResolvePath(bundleDir, rel)
	want := filepath.Join(bundleDir, "config", "app.json")

	if got != want {
		t.Errorf("ResolvePath() = %q, want %q", got, want)
	}
}

func TestResolvePath_MixedSeparators(t *testing.T) {
	bundleDir := "/opt/app/bundle"
	rel := `resources/playwright\browsers/chromium`

	got := ResolvePath(bundleDir, rel)
	want := filepath.Join(bundleDir, "resources", "playwright", "browsers", "chromium")

	if got != want {
		t.Errorf("ResolvePath() = %q, want %q", got, want)
	}
}
