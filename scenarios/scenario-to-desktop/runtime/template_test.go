package bundleruntime

import (
	"os"
	"testing"

	"scenario-to-desktop-runtime/manifest"
)

func TestRenderEnvMap(t *testing.T) {
	t.Setenv("EXISTING_VAR", "existing_value")

	s := &Supervisor{
		appData: "/app/data",
		opts: Options{
			BundlePath: "/bundle/root",
			Manifest:   &manifest.Manifest{},
		},
		portAllocator: &testMockPortAllocator{ports: map[string]map[string]int{}},
		envReader:     RealEnvReader{},
	}

	svc := manifest.Service{
		ID: "api",
		Env: map[string]string{
			"SERVICE_VAR": "service_value",
			"DATA_PATH":   "${data}/service",
		},
	}

	bin := manifest.Binary{
		Path: "bin/api",
		Env: map[string]string{
			"BINARY_VAR":  "binary_value",
			"BUNDLE_PATH": "${bundle}/assets",
		},
	}

	env, err := s.renderEnvMap(svc, bin)
	if err != nil {
		t.Fatalf("renderEnvMap() error = %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		// Inherited environment
		{"EXISTING_VAR", "existing_value"},
		// Standard bundle hints
		{"APP_DATA_DIR", "/app/data"},
		{"BUNDLE_ROOT", "/bundle/root"},
		// Service environment with template expansion
		{"SERVICE_VAR", "service_value"},
		{"DATA_PATH", "/app/data/service"},
		// Binary environment (overrides service)
		{"BINARY_VAR", "binary_value"},
		{"BUNDLE_PATH", "/bundle/root/assets"},
	}

	for _, tt := range tests {
		if env[tt.key] != tt.want {
			t.Errorf("env[%q] = %q, want %q", tt.key, env[tt.key], tt.want)
		}
	}
}

func TestRenderArgs(t *testing.T) {
	s := &Supervisor{
		appData: "/app/data",
		opts: Options{
			BundlePath: "/bundle",
		},
		portAllocator: &testMockPortAllocator{ports: map[string]map[string]int{
			"api": {"http": 47000, "grpc": 47001},
		}},
	}

	args := []string{
		"--config",
		"${data}/config.json",
		"--root",
		"${bundle}",
		"--port",
		"${api.http}",
		"--grpc-port",
		"${api.grpc}",
		"--literal",
		"no-template",
	}

	got := s.renderArgs(args)

	want := []string{
		"--config",
		"/app/data/config.json",
		"--root",
		"/bundle",
		"--port",
		"47000",
		"--grpc-port",
		"47001",
		"--literal",
		"no-template",
	}

	if len(got) != len(want) {
		t.Fatalf("renderArgs() returned %d args, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("renderArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRenderValue(t *testing.T) {
	s := &Supervisor{
		appData: "/home/user/.config/myapp",
		opts: Options{
			BundlePath: "/opt/myapp",
		},
		portAllocator: &testMockPortAllocator{ports: map[string]map[string]int{
			"api":      {"http": 8080},
			"database": {"postgres": 5432},
		}},
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"data variable", "${data}", "/home/user/.config/myapp"},
		{"bundle variable", "${bundle}", "/opt/myapp"},
		{"port lookup", "${api.http}", "8080"},
		{"nested port lookup", "${database.postgres}", "5432"},
		{"mixed template", "${data}/logs/${bundle}/file.log", "/home/user/.config/myapp/logs//opt/myapp/file.log"},
		{"no template", "plain-string", "plain-string"},
		{"unknown variable", "${unknown}", "${unknown}"},
		{"unknown port", "${nonexistent.port}", "${nonexistent.port}"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.renderValue(tt.input)
			if got != tt.want {
				t.Errorf("renderValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderEnvMap_BinaryOverridesService(t *testing.T) {
	s := &Supervisor{
		appData: "/app",
		opts: Options{
			BundlePath: "/bundle",
			Manifest:   &manifest.Manifest{},
		},
		portAllocator: &testMockPortAllocator{ports: map[string]map[string]int{}},
		envReader:     RealEnvReader{},
	}

	svc := manifest.Service{
		ID: "api",
		Env: map[string]string{
			"PORT":     "8080",
			"LOG_LEVEL": "info",
		},
	}

	bin := manifest.Binary{
		Env: map[string]string{
			"PORT": "9090", // Override service PORT
		},
	}

	env, err := s.renderEnvMap(svc, bin)
	if err != nil {
		t.Fatalf("renderEnvMap() error = %v", err)
	}

	// Binary PORT should override service PORT
	if env["PORT"] != "9090" {
		t.Errorf("env[PORT] = %q, want %q (binary should override service)", env["PORT"], "9090")
	}

	// Service LOG_LEVEL should still be present
	if env["LOG_LEVEL"] != "info" {
		t.Errorf("env[LOG_LEVEL] = %q, want %q", env["LOG_LEVEL"], "info")
	}
}

func TestRenderEnvMap_InheritsOSEnvironment(t *testing.T) {
	uniqueKey := "TEST_RUNTIME_UNIQUE_VAR_12345"
	uniqueValue := "unique_test_value"
	os.Setenv(uniqueKey, uniqueValue)
	defer os.Unsetenv(uniqueKey)

	s := &Supervisor{
		appData: "/app",
		opts: Options{
			BundlePath: "/bundle",
			Manifest:   &manifest.Manifest{},
		},
		portAllocator: &testMockPortAllocator{ports: map[string]map[string]int{}},
		envReader:     RealEnvReader{},
	}

	env, err := s.renderEnvMap(manifest.Service{}, manifest.Binary{})
	if err != nil {
		t.Fatalf("renderEnvMap() error = %v", err)
	}

	if env[uniqueKey] != uniqueValue {
		t.Errorf("env[%s] = %q, want %q (should inherit OS environment)", uniqueKey, env[uniqueKey], uniqueValue)
	}
}
