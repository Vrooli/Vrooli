package secrets

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/infra"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/testutil"
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
	store := NewManager(m)
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
	store := NewManager(m)
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

// A file-target secret must land on ephemeral storage, never in APP_DATA_DIR,
// and must be removable once the service that needed it has started. A
// mode-0600 file on durable storage is the protection level the encrypted
// store exists to replace, and this package's own README promises production
// values are never persisted to app data.
func TestInjectorFileTargetStaysOffDurableStorageAndIsRemovable(t *testing.T) {
	required := true
	const value = "-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"
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
	store := NewManager(m)
	store.Set(map[string]string{"CERT": value})

	const appData = "/app/data"
	ephemeral := "/run/ephemeral/bundle-secrets"
	inj := NewInjector(store, mockFS, appData)
	inj.EphemeralDir = ephemeral

	env := make(map[string]string)
	if err := inj.Apply(env, manifest.Service{ID: "api", Secrets: []string{"CERT"}}); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}

	path := env["SECRET_FILE_CERT"]
	if path == "" {
		t.Fatal("no path was published for the file-target secret")
	}
	if strings.HasPrefix(path, appData) {
		t.Fatalf("secret was written under the durable app data dir: %q", path)
	}
	if !strings.HasPrefix(path, ephemeral) {
		t.Fatalf("secret path = %q, want it under the ephemeral dir %q", path, ephemeral)
	}

	data, err := mockFS.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != value {
		t.Errorf("file content did not round-trip (%d bytes, want %d)", len(data), len(value))
	}

	// Once the service has started the file has no remaining purpose, and a
	// value left on disk past that point is exposure for nothing.
	if err := inj.RemoveMaterializedSecrets(); err != nil {
		t.Fatalf("RemoveMaterializedSecrets() error = %v", err)
	}
	if _, err := mockFS.ReadFile(path); err == nil {
		t.Fatal("the materialized secret survived removal")
	}
}

// With no ephemeral location available the bundle must refuse. Falling back to
// durable storage would write a plaintext credential an operator cannot see and
// would never learn they had.
func TestInjectorRefusesAFileTargetWithNoEphemeralLocation(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{{
			ID:       "CERT",
			Required: &required,
			Target:   manifest.SecretTarget{Type: "file", Name: "certs/cert.pem"},
		}},
	}
	store := NewManager(m)
	store.Set(map[string]string{"CERT": "value"})

	inj := NewInjector(store, refusingMkdirFS{testutil.NewMockFileSystem()}, "/app/data")
	err := inj.Apply(make(map[string]string), manifest.Service{ID: "api", Secrets: []string{"CERT"}})
	if err == nil {
		t.Fatal("Apply() succeeded with nowhere ephemeral to write; it must refuse")
	}
	if !strings.Contains(err.Error(), "ephemeral") {
		t.Fatalf("error = %v, want it to name the missing ephemeral location", err)
	}
}

// refusingMkdirFS is a host on which no candidate ephemeral directory can be
// created.
type refusingMkdirFS struct{ infra.FileSystem }

func (refusingMkdirFS) MkdirAll(string, fs.FileMode) error {
	return errors.New("read-only file system")
}

func TestInjector_Apply_MissingRequired(t *testing.T) {
	required := true
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Required: &required, Target: manifest.SecretTarget{Type: "env"}},
		},
	}

	mockFS := testutil.NewMockFileSystem()
	store := NewManager(m)
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
	store := NewManager(m)
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
	store := NewManager(m)

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
	store := NewManager(m)
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
