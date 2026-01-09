package secrets

import (
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

func TestInjector_Apply_EnvTarget(t *testing.T) {
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

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{"API_KEY": "secret_value"})

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"API_KEY"},
	}

	if err := inj.Apply(env, svc); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if env["MY_API_KEY"] != "secret_value" {
		t.Errorf("env[MY_API_KEY] = %q, want %q", env["MY_API_KEY"], "secret_value")
	}
}

func TestInjector_Apply_EnvTargetDefaultName(t *testing.T) {
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

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{"api_key": "secret_value"})

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"api_key"},
	}

	if err := inj.Apply(env, svc); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	if env["API_KEY"] != "secret_value" {
		t.Errorf("env[API_KEY] = %q, want %q", env["API_KEY"], "secret_value")
	}
}

func TestInjector_Apply_FileTarget(t *testing.T) {
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

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{"CERT": "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"})

	appData := "/app/data"
	inj := NewInjector(store, mockFS, appData)

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"CERT"},
	}

	if err := inj.Apply(env, svc); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	// Check env var was set with file path
	expectedPath := filepath.Join(appData, "certs/cert.pem")
	if env["SECRET_FILE_CERT"] != expectedPath {
		t.Errorf("env[SECRET_FILE_CERT] = %q, want %q", env["SECRET_FILE_CERT"], expectedPath)
	}

	// Verify file was written
	data, err := mockFS.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----" {
		t.Errorf("File content = %q, unexpected", string(data))
	}
}

func TestInjector_Apply_MissingRequired(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Required: &required, Target: manifest.SecretTarget{Type: "env"}},
		},
	}

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{}) // No secrets set

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"API_KEY"},
	}

	err := inj.Apply(env, svc)
	if err == nil {
		t.Error("Apply() expected error for missing required secret")
	}
}

func TestInjector_Apply_MissingOptional(t *testing.T) {
	optional := false
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "OPTIONAL_KEY", Required: &optional, Target: manifest.SecretTarget{Type: "env"}},
		},
	}

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{}) // No secrets set

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"OPTIONAL_KEY"},
	}

	err := inj.Apply(env, svc)
	if err != nil {
		t.Errorf("Apply() error = %v, expected nil for missing optional", err)
	}
}

func TestInjector_Apply_UnknownSecret(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{}, // No secrets defined
	}

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"UNKNOWN_SECRET"},
	}

	err := inj.Apply(env, svc)
	if err == nil {
		t.Error("Apply() expected error for unknown secret")
	}
}

func TestInjector_Apply_UnsupportedTargetType(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "KEY", Required: &required, Target: manifest.SecretTarget{Type: "unknown"}},
		},
	}

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m, mockFS, "/tmp/secrets.json")
	store.Set(map[string]string{"KEY": "value"})

	inj := NewInjector(store, mockFS, "/app/data")

	env := make(map[string]string)
	svc := manifest.Service{
		ID:      "api",
		Secrets: []string{"KEY"},
	}

	err := inj.Apply(env, svc)
	if err == nil {
		t.Error("Apply() expected error for unsupported target type")
	}
}
