package env

import (
	"sort"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
)

func TestBundledEnvironmentAllowlistHasReasonForEveryEntry(t *testing.T) {
	previous := ""
	for _, key := range BundledEnvironmentAllowlist {
		if reason := BundledEnvironmentReasons[key]; reason == "" {
			t.Errorf("allowlisted environment %q has no reason", key)
		}
		if previous != "" && previous >= key {
			t.Errorf("allowlist is not sorted: %q before %q", previous, key)
		}
		previous = key
	}
	keys := make([]string, 0, len(BundledEnvironmentReasons))
	for key := range BundledEnvironmentReasons {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) != len(BundledEnvironmentAllowlist) {
		t.Fatalf("allowlist and reasons diverge")
	}
}

// mockEnvReader implements infra.EnvReader for testing.
type mockEnvReader struct {
	env map[string]string
}

func (r mockEnvReader) Getenv(key string) string {
	return r.env[key]
}

func (r mockEnvReader) Environ() []string {
	out := make([]string, 0, len(r.env))
	for k, v := range r.env {
		out = append(out, k+"="+v)
	}
	return out
}

func TestRenderer_RenderEnvMap(t *testing.T) {
	ports := testutil.NewMockPortAllocator()
	envReader := mockEnvReader{env: map[string]string{
		"EXISTING_VAR": "existing_value",
	}}

	r := NewRenderer("/app/data", "/bundle/root", ports, envReader)

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

	env, err := r.RenderEnvMap(svc, bin)
	if err != nil {
		t.Fatalf("RenderEnvMap() error = %v", err)
	}

	tests := []struct {
		key  string
		want string
	}{
		// Allowlisted inherited environment is retained only when present.
		// Standard bundle hints
		{"APP_DATA_DIR", "/app/data"},
		{"BUNDLE_ROOT", "/bundle/root"},
		{"VROOLI_DATA", "/app/data"},
		{"VROOLI_DESKTOP_MODE", "true"},
		{"VROOLI_LIFECYCLE_MANAGED", "true"},
		{"VROOLI_STORAGE_ROOT", "/app/data/storage"},
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

	if _, ok := env["VROOLI_ROOT"]; ok {
		t.Fatal("bundled environment must not inherit VROOLI_ROOT")
	}
	if _, ok := env["SCENARIO_ROOT"]; ok {
		t.Fatal("bundled environment must not inherit SCENARIO_ROOT")
	}
}

func TestRenderer_RenderArgs(t *testing.T) {
	ports := testutil.NewMockPortAllocator()
	ports.SetPort("api", "http", 47000)
	ports.SetPort("api", "grpc", 47001)

	r := NewRenderer("/app/data", "/bundle", ports, mockEnvReader{})

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

	got, err := r.RenderArgs(args)
	if err != nil {
		t.Fatalf("RenderArgs() error = %v", err)
	}

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
		t.Fatalf("RenderArgs() returned %d args, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Errorf("RenderArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestRenderer_RenderValue(t *testing.T) {
	ports := testutil.NewMockPortAllocator()
	ports.SetPort("api", "http", 8080)
	ports.SetPort("database", "postgres", 5432)

	r := NewRenderer("/home/user/.config/myapp", "/opt/myapp", ports, mockEnvReader{})

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
		{"unknown variable", "${unknown}", ""},
		{"unknown port", "${nonexistent.port}", ""},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.RenderValue(tt.input)
			if tt.name == "unknown variable" || tt.name == "unknown port" {
				if err == nil {
					t.Fatal("expected unresolved template error")
				}
				return
			}
			if err != nil {
				t.Fatalf("RenderValue() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("RenderValue(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderer_RenderEnvMap_BinaryOverridesService(t *testing.T) {
	ports := testutil.NewMockPortAllocator()
	r := NewRenderer("/app", "/bundle", ports, mockEnvReader{})

	svc := manifest.Service{
		ID: "api",
		Env: map[string]string{
			"PORT":      "8080",
			"LOG_LEVEL": "info",
		},
	}

	bin := manifest.Binary{
		Env: map[string]string{
			"PORT": "9090", // Override service PORT
		},
	}

	env, err := r.RenderEnvMap(svc, bin)
	if err != nil {
		t.Fatalf("RenderEnvMap() error = %v", err)
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

func TestRenderer_RenderEnvMap_InheritsEnvironment(t *testing.T) {
	ports := testutil.NewMockPortAllocator()
	envReader := mockEnvReader{env: map[string]string{
		"UNIQUE_VAR": "unique_value",
	}}

	r := NewRenderer("/app", "/bundle", ports, envReader)

	env, err := r.RenderEnvMap(manifest.Service{}, manifest.Binary{})
	if err != nil {
		t.Fatalf("RenderEnvMap() error = %v", err)
	}

	if _, ok := env["UNIQUE_VAR"]; ok {
		t.Error("environment must not inherit an unallowlisted variable")
	}
}

func TestRenderer_RenderEnvMapScrubsScenarioState(t *testing.T) {
	r := NewRenderer("/app", "/bundle", testutil.NewMockPortAllocator(), mockEnvReader{env: map[string]string{
		"WC_SESSION_STATE_ROOT": "/operator/sessions", "VROOLI_STORAGE_NAMESPACE": "live", "XDG_STATE_HOME": "/operator/state",
		"WC_WEB_CONSOLE_SESSION_ID": "operator", "HOME": "/operator/home", "LANG": "C.UTF-8",
	}})
	env, err := r.RenderEnvMap(manifest.Service{Env: map[string]string{"DECLARED": "yes"}}, manifest.Binary{})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"WC_SESSION_STATE_ROOT", "VROOLI_STORAGE_NAMESPACE", "XDG_STATE_HOME", "WC_WEB_CONSOLE_SESSION_ID"} {
		if _, ok := env[key]; ok {
			t.Errorf("env contains scrubbed key %s", key)
		}
	}
	if env["HOME"] != "/operator/home" || env["LANG"] != "C.UTF-8" || env["DECLARED"] != "yes" {
		t.Errorf("allowlisted or declared environment missing: %#v", env)
	}
}
