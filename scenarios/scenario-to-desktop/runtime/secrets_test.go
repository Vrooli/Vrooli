package bundleruntime

import (
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/secrets"
	"scenario-to-desktop-runtime/testutil"
)

// testSecretsSupervisor creates a Supervisor configured for secrets testing.
func testSecretsSupervisor(t *testing.T, m *manifest.Manifest, secretVals map[string]string) *Supervisor {
	t.Helper()

	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(timeNow())

	sm := secrets.NewManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	if secretVals != nil {
		sm.Set(secretVals)
	}

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		appData:       tmp,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
	}
	return s
}

func TestApplySecrets_EnvTarget(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:       "API_KEY",
				Required: &required,
				Target:   manifest.SecretTarget{Type: "env", Name: "MY_API_KEY"},
			},
		},
	}
	s := testSecretsSupervisor(t, m, map[string]string{"API_KEY": "secret_value"})

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"API_KEY"},
	}

	if err := s.applySecrets(env, svc); err != nil {
		t.Fatalf("applySecrets() error = %v", err)
	}

	if env["MY_API_KEY"] != "secret_value" {
		t.Errorf("env[MY_API_KEY] = %q, want %q", env["MY_API_KEY"], "secret_value")
	}
}

func TestApplySecrets_EnvTargetDefaultName(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:       "api_key",
				Required: &required,
				Target:   manifest.SecretTarget{Type: "env"}, // No name, should use uppercase ID
			},
		},
	}
	s := testSecretsSupervisor(t, m, map[string]string{"api_key": "secret_value"})

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"api_key"},
	}

	if err := s.applySecrets(env, svc); err != nil {
		t.Fatalf("applySecrets() error = %v", err)
	}

	if env["API_KEY"] != "secret_value" {
		t.Errorf("env[API_KEY] = %q, want %q", env["API_KEY"], "secret_value")
	}
}

func TestApplySecrets_FileTarget(t *testing.T) {
	tmp := t.TempDir()
	required := true

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{
				ID:       "CERT",
				Required: &required,
				Target:   manifest.SecretTarget{Type: "file", Name: "certs/cert.pem"},
			},
		},
	}

	s := &Supervisor{
		opts:          Options{Manifest: m},
		appData:       tmp,
		fs:            RealFileSystem{},
		secretStore:   secrets.NewManager(m, RealFileSystem{}, filepath.Join(tmp, "secrets.json")),
		portAllocator: &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
	}
	s.secretStore.Set(map[string]string{"CERT": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"})

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"CERT"},
	}

	if err := s.applySecrets(env, svc); err != nil {
		t.Fatalf("applySecrets() error = %v", err)
	}

	// Check env var was set with file path
	expectedPath := filepath.Join(tmp, "certs/cert.pem")
	if env["SECRET_FILE_CERT"] != expectedPath {
		t.Errorf("env[SECRET_FILE_CERT] = %q, want %q", env["SECRET_FILE_CERT"], expectedPath)
	}

	// Verify file was written
	data, err := s.fs.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----" {
		t.Errorf("File content = %q, unexpected", string(data))
	}
}

func TestApplySecrets_MissingRequired(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Required: &required, Target: manifest.SecretTarget{Type: "env"}},
		},
	}
	s := testSecretsSupervisor(t, m, map[string]string{}) // No secrets set

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"API_KEY"},
	}

	err := s.applySecrets(env, svc)
	if err == nil {
		t.Error("applySecrets() expected error for missing required secret")
	}
}

func TestApplySecrets_MissingOptional(t *testing.T) {
	optional := false
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "OPTIONAL_KEY", Required: &optional, Target: manifest.SecretTarget{Type: "env"}},
		},
	}
	s := testSecretsSupervisor(t, m, map[string]string{}) // No secrets set

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"OPTIONAL_KEY"},
	}

	err := s.applySecrets(env, svc)
	if err != nil {
		t.Errorf("applySecrets() error = %v, expected nil for missing optional", err)
	}
}

func TestApplySecrets_UnknownSecret(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{}, // No secrets defined
	}
	s := testSecretsSupervisor(t, m, map[string]string{})

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"UNKNOWN_SECRET"},
	}

	err := s.applySecrets(env, svc)
	if err == nil {
		t.Error("applySecrets() expected error for unknown secret")
	}
}

func TestApplySecrets_UnsupportedTargetType(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "KEY", Required: &required, Target: manifest.SecretTarget{Type: "unknown"}},
		},
	}
	s := testSecretsSupervisor(t, m, map[string]string{"KEY": "value"})

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"KEY"},
	}

	err := s.applySecrets(env, svc)
	if err == nil {
		t.Error("applySecrets() expected error for unsupported target type")
	}
}

func TestUpdateSecrets(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(timeNow())

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{}, // No required secrets
	}
	sm := secrets.NewManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{"EXISTING": "value"})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
	}

	newSecrets := map[string]string{"NEW_KEY": "new_value"}
	err := s.UpdateSecrets(newSecrets)
	if err != nil {
		t.Fatalf("UpdateSecrets() error = %v", err)
	}

	// Check merged secrets
	secrets := s.secretStore.Get()
	if secrets["EXISTING"] != "value" {
		t.Error("UpdateSecrets() lost existing secret")
	}
	if secrets["NEW_KEY"] != "new_value" {
		t.Error("UpdateSecrets() didn't add new secret")
	}
}

func TestUpdateSecrets_MissingRequired(t *testing.T) {
	tmp := t.TempDir()
	mockFS := testutil.NewMockFileSystem()
	mockClock := testutil.NewMockClock(timeNow())
	required := true

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "REQUIRED_KEY", Required: &required},
		},
	}
	sm := secrets.NewManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testutil.MockPortAllocator{Ports: map[string]map[string]int{}},
	}

	// Update without providing required secret
	err := s.UpdateSecrets(map[string]string{"OTHER": "value"})
	if err == nil {
		t.Error("UpdateSecrets() expected error for missing required secret")
	}
}

func timeNow() time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}
