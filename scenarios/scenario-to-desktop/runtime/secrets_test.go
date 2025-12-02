package bundleruntime

import (
	"path/filepath"
	"testing"
	"time"

	"scenario-to-desktop-runtime/manifest"
)

func TestSecretManagerLoad_MissingFile(t *testing.T) {
	mockFS := NewMockFileSystem()
	sm := NewSecretManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("Load() returned %d secrets, want 0", len(secrets))
	}
}

func TestSecretManagerLoad_NewFormat(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.Files["/app/data/secrets.json"] = []byte(`{"secrets": {"API_KEY": "secret123", "DB_PASS": "password"}}`)

	sm := NewSecretManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("Load() returned %d secrets, want 2", len(secrets))
	}
	if secrets["API_KEY"] != "secret123" {
		t.Errorf("secrets[API_KEY] = %q, want %q", secrets["API_KEY"], "secret123")
	}
	if secrets["DB_PASS"] != "password" {
		t.Errorf("secrets[DB_PASS] = %q, want %q", secrets["DB_PASS"], "password")
	}
}

func TestSecretManagerLoad_LegacyFormat(t *testing.T) {
	mockFS := NewMockFileSystem()
	mockFS.Files["/app/data/secrets.json"] = []byte(`{"API_KEY": "legacy_key"}`)

	sm := NewSecretManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if secrets["API_KEY"] != "legacy_key" {
		t.Errorf("secrets[API_KEY] = %q, want %q", secrets["API_KEY"], "legacy_key")
	}
}

func TestSecretManagerPersist(t *testing.T) {
	tmp := t.TempDir()
	secretsPath := filepath.Join(tmp, "subdir", "secrets.json")

	sm := NewSecretManager(&manifest.Manifest{}, RealFileSystem{}, secretsPath)

	secrets := map[string]string{
		"API_KEY": "test_key",
		"SECRET":  "test_secret",
	}

	if err := sm.Persist(secrets); err != nil {
		t.Fatalf("Persist() error = %v", err)
	}

	// Reload and verify
	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded["API_KEY"] != "test_key" {
		t.Errorf("loaded API_KEY = %q, want %q", loaded["API_KEY"], "test_key")
	}
}

func TestSecretManagerGet(t *testing.T) {
	sm := NewSecretManager(&manifest.Manifest{}, NewMockFileSystem(), "/tmp/secrets.json")
	sm.Set(map[string]string{"KEY": "value"})

	copy := sm.Get()
	copy["KEY"] = "modified"

	// Original should be unchanged
	original := sm.Get()
	if original["KEY"] != "value" {
		t.Error("Get() returned reference, not copy")
	}
}

func TestSecretManagerMissingRequired(t *testing.T) {
	required := true
	optional := false

	tests := []struct {
		name     string
		manifest []manifest.Secret
		secrets  map[string]string
		want     []string
	}{
		{
			name: "all present",
			manifest: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "KEY2", Required: &required},
			},
			secrets: map[string]string{"KEY1": "val1", "KEY2": "val2"},
			want:    nil,
		},
		{
			name: "one missing",
			manifest: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "KEY2", Required: &required},
			},
			secrets: map[string]string{"KEY1": "val1"},
			want:    []string{"KEY2"},
		},
		{
			name: "optional missing is ok",
			manifest: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "OPTIONAL", Required: &optional},
			},
			secrets: map[string]string{"KEY1": "val1"},
			want:    nil,
		},
		{
			name: "default required is true",
			manifest: []manifest.Secret{
				{ID: "KEY1"}, // nil Required defaults to true
			},
			secrets: map[string]string{},
			want:    []string{"KEY1"},
		},
		{
			name: "empty value is missing",
			manifest: []manifest.Secret{
				{ID: "KEY1", Required: &required},
			},
			secrets: map[string]string{"KEY1": "  "},
			want:    []string{"KEY1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewSecretManager(&manifest.Manifest{Secrets: tt.manifest}, NewMockFileSystem(), "/tmp/secrets.json")
			sm.Set(tt.secrets)

			got := sm.MissingRequired()
			if len(got) != len(tt.want) {
				t.Errorf("MissingRequired() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MissingRequired()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSecretManagerFindSecret(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Description: "API Key"},
			{ID: "DB_PASS", Description: "Database Password"},
		},
	}
	sm := NewSecretManager(m, NewMockFileSystem(), "/tmp/secrets.json")

	// Found
	sec := sm.FindSecret("API_KEY")
	if sec == nil {
		t.Fatal("FindSecret() returned nil for existing secret")
	}
	if sec.Description != "API Key" {
		t.Errorf("FindSecret().Description = %q, want %q", sec.Description, "API Key")
	}

	// Not found
	sec = sm.FindSecret("NONEXISTENT")
	if sec != nil {
		t.Error("FindSecret() returned non-nil for missing secret")
	}
}

// testSecretsSupervisor creates a Supervisor configured for secrets testing.
func testSecretsSupervisor(t *testing.T, m *manifest.Manifest, secrets map[string]string) *Supervisor {
	t.Helper()

	tmp := t.TempDir()
	mockFS := NewMockFileSystem()
	mockClock := NewMockClock(timeNow())

	sm := NewSecretManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	if secrets != nil {
		sm.Set(secrets)
	}

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		appData:       tmp,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testMockPortAllocator{Ports: map[string]map[string]int{}},
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
		secretStore:   NewSecretManager(m, RealFileSystem{}, filepath.Join(tmp, "secrets.json")),
		portAllocator: &testMockPortAllocator{Ports: map[string]map[string]int{}},
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
	mockFS := NewMockFileSystem()
	mockClock := NewMockClock(timeNow())

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{}, // No required secrets
	}
	sm := NewSecretManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{"EXISTING": "value"})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testMockPortAllocator{Ports: map[string]map[string]int{}},
	}

	newSecrets := map[string]string{"NEW_KEY": "new_value"}
	err := s.UpdateSecrets(newSecrets)
	if err != nil {
		t.Fatalf("UpdateSecrets() error = %v", err)
	}

	// Check merged secrets
	secrets := s.secretsCopy()
	if secrets["EXISTING"] != "value" {
		t.Error("UpdateSecrets() lost existing secret")
	}
	if secrets["NEW_KEY"] != "new_value" {
		t.Error("UpdateSecrets() didn't add new secret")
	}
}

func TestUpdateSecrets_MissingRequired(t *testing.T) {
	tmp := t.TempDir()
	mockFS := NewMockFileSystem()
	mockClock := NewMockClock(timeNow())
	required := true

	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "REQUIRED_KEY", Required: &required},
		},
	}
	sm := NewSecretManager(m, mockFS, filepath.Join(tmp, "secrets.json"))
	sm.Set(map[string]string{})

	s := &Supervisor{
		opts:          Options{Manifest: m},
		fs:            mockFS,
		clock:         mockClock,
		telemetryPath: filepath.Join(tmp, "telemetry.jsonl"),
		secretStore:   sm,
		portAllocator: &testMockPortAllocator{Ports: map[string]map[string]int{}},
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
