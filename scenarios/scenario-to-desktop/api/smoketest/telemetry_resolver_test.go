package smoketest_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/smoketest/mocks"
)

func TestTelemetryPathResolver_ExtractFromOutput(t *testing.T) {
	config := smoketest.DefaultConfig()
	envReader := mocks.NewMockEnvironmentReader()
	fs := mocks.NewMockFileSystem()
	resolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)

	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
		{
			name:   "no marker",
			output: "Starting app...\nApp started",
			want:   "",
		},
		{
			name:   "marker with path",
			output: "[Desktop App] Telemetry initialized at /home/user/.config/myapp/deployment-telemetry.jsonl\nApp ready",
			want:   "/home/user/.config/myapp/deployment-telemetry.jsonl",
		},
		{
			name:   "marker at end of line",
			output: "[Desktop App] Telemetry initialized at /tmp/test.jsonl",
			want:   "/tmp/test.jsonl",
		},
		{
			name:   "marker with extra whitespace",
			output: "[Desktop App] Telemetry initialized at   /path/to/file.jsonl   ",
			want:   "/path/to/file.jsonl",
		},
		{
			name:   "marker in middle of output",
			output: "line1\nline2\n[Desktop App] Telemetry initialized at /correct/path.jsonl\nline4",
			want:   "/correct/path.jsonl",
		},
		{
			name:   "multiple lines with marker - first wins",
			output: "[Desktop App] Telemetry initialized at /first/path.jsonl\n[Desktop App] Telemetry initialized at /second/path.jsonl",
			want:   "/first/path.jsonl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolver.ExtractFromOutput(tt.output)
			if got != tt.want {
				t.Errorf("ExtractFromOutput() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTelemetryPathResolver_ResolveFromArtifact(t *testing.T) {
	config := smoketest.DefaultConfig()

	tests := []struct {
		name         string
		platform     string
		artifactPath string
		scenarioName string
		setupEnv     func(*mocks.MockEnvironmentReader)
		setupFS      func(*mocks.MockFileSystem)
		want         string
	}{
		{
			name:         "linux with package.json",
			platform:     "linux",
			artifactPath: "/path/to/dist/myapp.AppImage",
			scenarioName: "fallback-scenario",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.HomeDir = "/home/user"
			},
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddFile("/path/to/package.json", []byte(`{"name": "my-electron-app"}`))
			},
			want: "/home/user/.config/my-electron-app/deployment-telemetry.jsonl",
		},
		{
			name:         "linux with XDG_CONFIG_HOME",
			platform:     "linux",
			artifactPath: "/path/to/dist/myapp.AppImage",
			scenarioName: "fallback-scenario",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.SetEnv("XDG_CONFIG_HOME", "/custom/config")
			},
			setupFS: func(fs *mocks.MockFileSystem) {
				fs.AddFile("/path/to/package.json", []byte(`{"name": "myapp"}`))
			},
			want: "/custom/config/myapp/deployment-telemetry.jsonl",
		},
		{
			name:         "linux fallback to scenario name",
			platform:     "linux",
			artifactPath: "/path/to/dist/myapp.AppImage",
			scenarioName: "my-scenario",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.HomeDir = "/home/user"
			},
			setupFS: func(fs *mocks.MockFileSystem) {
				// No package.json
			},
			want: "/home/user/.config/my-scenario/deployment-telemetry.jsonl",
		},
		{
			name:         "mac platform",
			platform:     "mac",
			artifactPath: "/path/to/dist/MyApp.app",
			scenarioName: "myapp",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.HomeDir = "/Users/testuser"
			},
			setupFS: func(fs *mocks.MockFileSystem) {},
			want:    "/Users/testuser/Library/Application Support/myapp/deployment-telemetry.jsonl",
		},
		{
			name:         "win platform",
			platform:     "win",
			artifactPath: "C:\\path\\to\\dist\\myapp.exe",
			scenarioName: "myapp",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.SetEnv("APPDATA", "C:\\Users\\testuser\\AppData\\Roaming")
			},
			setupFS: func(fs *mocks.MockFileSystem) {},
			want:    filepath.Join("C:\\Users\\testuser\\AppData\\Roaming", "myapp", "deployment-telemetry.jsonl"),
		},
		{
			name:         "win platform without APPDATA",
			platform:     "win",
			artifactPath: "C:\\path\\to\\dist\\myapp.exe",
			scenarioName: "myapp",
			setupEnv:     func(env *mocks.MockEnvironmentReader) {},
			setupFS:      func(fs *mocks.MockFileSystem) {},
			want:         "",
		},
		{
			name:         "empty app name returns empty path",
			platform:     "linux",
			artifactPath: "/path/to/dist/myapp.AppImage",
			scenarioName: "",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.HomeDir = "/home/user"
			},
			setupFS: func(fs *mocks.MockFileSystem) {
				// package.json exists but has no name
				fs.AddFile("/path/to/package.json", []byte(`{}`))
			},
			want: "",
		},
		{
			name:         "mac without home dir",
			platform:     "mac",
			artifactPath: "/path/to/dist/MyApp.app",
			scenarioName: "myapp",
			setupEnv: func(env *mocks.MockEnvironmentReader) {
				env.HomeDirErr = errors.New("no home dir")
			},
			setupFS: func(fs *mocks.MockFileSystem) {},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			envReader := mocks.NewMockEnvironmentReader()
			fs := mocks.NewMockFileSystem()

			if tt.setupEnv != nil {
				tt.setupEnv(envReader)
			}
			if tt.setupFS != nil {
				tt.setupFS(fs)
			}

			resolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)
			got := resolver.ResolveFromArtifact(tt.platform, tt.artifactPath, tt.scenarioName)

			if got != tt.want {
				t.Errorf("ResolveFromArtifact() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTelemetryPathResolver_ReadTelemetryEvents(t *testing.T) {
	// Create a temp file with telemetry events
	tmpDir := t.TempDir()
	telemetryPath := filepath.Join(tmpDir, "telemetry.jsonl")

	content := `{"event": "startup", "ts": 1}
{"event": "ready", "ts": 2}
{"event": "shutdown", "ts": 3}
`
	if err := os.WriteFile(telemetryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := smoketest.DefaultConfig()
	envReader := smoketest.NewEnvironmentReader()
	fs := smoketest.NewFileSystem()
	resolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)

	t.Run("read all events", func(t *testing.T) {
		events, err := resolver.ReadTelemetryEvents(telemetryPath, 0)
		if err != nil {
			t.Fatalf("ReadTelemetryEvents() error = %v", err)
		}
		if len(events) != 3 {
			t.Errorf("ReadTelemetryEvents() returned %d events, want 3", len(events))
		}
	})

	t.Run("read with limit", func(t *testing.T) {
		events, err := resolver.ReadTelemetryEvents(telemetryPath, 2)
		if err != nil {
			t.Fatalf("ReadTelemetryEvents() error = %v", err)
		}
		if len(events) != 2 {
			t.Errorf("ReadTelemetryEvents() returned %d events, want 2", len(events))
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := resolver.ReadTelemetryEvents("/nonexistent/path", 0)
		if err == nil {
			t.Error("ReadTelemetryEvents() expected error for nonexistent file")
		}
	})
}

func TestTelemetryPathResolver_ReadTelemetryEvents_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	telemetryPath := filepath.Join(tmpDir, "telemetry.jsonl")

	content := `{"event": "first"}

{"event": "second"}

{"event": "third"}
`
	if err := os.WriteFile(telemetryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := smoketest.DefaultConfig()
	envReader := smoketest.NewEnvironmentReader()
	fs := smoketest.NewFileSystem()
	resolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)

	events, err := resolver.ReadTelemetryEvents(telemetryPath, 0)
	if err != nil {
		t.Fatalf("ReadTelemetryEvents() error = %v", err)
	}
	if len(events) != 3 {
		t.Errorf("ReadTelemetryEvents() returned %d events, want 3 (empty lines should be skipped)", len(events))
	}
}

func TestTelemetryPathResolver_ReadTelemetryEvents_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	telemetryPath := filepath.Join(tmpDir, "telemetry.jsonl")

	content := `{"event": "valid"}
not valid json
{"event": "after_invalid"}
`
	if err := os.WriteFile(telemetryPath, []byte(content), 0o644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	config := smoketest.DefaultConfig()
	envReader := smoketest.NewEnvironmentReader()
	fs := smoketest.NewFileSystem()
	resolver := smoketest.NewTelemetryPathResolver(config, envReader, fs)

	events, err := resolver.ReadTelemetryEvents(telemetryPath, 0)
	// Should return the valid event parsed before the error
	if len(events) != 1 {
		t.Errorf("ReadTelemetryEvents() returned %d events, want 1", len(events))
	}
	if err == nil {
		t.Error("ReadTelemetryEvents() expected error for invalid JSON")
	}
}
