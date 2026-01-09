package secrets

import (
	"path/filepath"
	"testing"

	"scenario-to-desktop-runtime/infra"
	"scenario-to-desktop-runtime/manifest"
	"scenario-to-desktop-runtime/testutil"
)

func TestManagerLoad_MissingFile(t *testing.T) {
	mockFS := testutil.NewMockFileSystem()
	sm := NewManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("Load() returned %d secrets, want 0", len(secrets))
	}
}

func TestManagerLoad_NewFormat(t *testing.T) {
	mockFS := testutil.NewMockFileSystem()
	mockFS.Files["/app/data/secrets.json"] = []byte(`{"secrets": {"API_KEY": "secret123", "DB_PASS": "password"}}`)

	sm := NewManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

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

func TestManagerLoad_LegacyFormat(t *testing.T) {
	mockFS := testutil.NewMockFileSystem()
	mockFS.Files["/app/data/secrets.json"] = []byte(`{"API_KEY": "legacy_key"}`)

	sm := NewManager(&manifest.Manifest{}, mockFS, "/app/data/secrets.json")

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if secrets["API_KEY"] != "legacy_key" {
		t.Errorf("secrets[API_KEY] = %q, want %q", secrets["API_KEY"], "legacy_key")
	}
}

func TestManagerPersist(t *testing.T) {
	tmp := t.TempDir()
	secretsPath := filepath.Join(tmp, "subdir", "secrets.json")

	sm := NewManager(&manifest.Manifest{}, infra.RealFileSystem{}, secretsPath)

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

func TestManagerGet(t *testing.T) {
	sm := NewManager(&manifest.Manifest{}, testutil.NewMockFileSystem(), "/tmp/secrets.json")
	sm.Set(map[string]string{"KEY": "value"})

	copy := sm.Get()
	copy["KEY"] = "modified"

	// Original should be unchanged
	original := sm.Get()
	if original["KEY"] != "value" {
		t.Error("Get() returned reference, not copy")
	}
}

func TestManagerMissingRequired(t *testing.T) {
	required := true
	optional := false

	tests := []struct {
		name         string
		manifestSecs []manifest.Secret
		secrets      map[string]string
		want         []string
	}{
		{
			name: "all present",
			manifestSecs: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "KEY2", Required: &required},
			},
			secrets: map[string]string{"KEY1": "val1", "KEY2": "val2"},
			want:    nil,
		},
		{
			name: "one missing",
			manifestSecs: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "KEY2", Required: &required},
			},
			secrets: map[string]string{"KEY1": "val1"},
			want:    []string{"KEY2"},
		},
		{
			name: "optional missing is ok",
			manifestSecs: []manifest.Secret{
				{ID: "KEY1", Required: &required},
				{ID: "OPTIONAL", Required: &optional},
			},
			secrets: map[string]string{"KEY1": "val1"},
			want:    nil,
		},
		{
			name: "default required is true",
			manifestSecs: []manifest.Secret{
				{ID: "KEY1"}, // nil Required defaults to true
			},
			secrets: map[string]string{},
			want:    []string{"KEY1"},
		},
		{
			name: "empty value is missing",
			manifestSecs: []manifest.Secret{
				{ID: "KEY1", Required: &required},
			},
			secrets: map[string]string{"KEY1": "  "},
			want:    []string{"KEY1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewManager(&manifest.Manifest{Secrets: tt.manifestSecs}, testutil.NewMockFileSystem(), "/tmp/secrets.json")
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

func TestManagerFindSecret(t *testing.T) {
	m := &manifest.Manifest{
		Secrets: []manifest.Secret{
			{ID: "API_KEY", Description: "API Key"},
			{ID: "DB_PASS", Description: "Database Password"},
		},
	}
	sm := NewManager(m, testutil.NewMockFileSystem(), "/tmp/secrets.json")

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

func TestManagerMerge(t *testing.T) {
	sm := NewManager(&manifest.Manifest{}, testutil.NewMockFileSystem(), "/tmp/secrets.json")
	sm.Set(map[string]string{"EXISTING": "value1", "OVERWRITE": "old"})

	merged := sm.Merge(map[string]string{"NEW": "value2", "OVERWRITE": "new"})

	if merged["EXISTING"] != "value1" {
		t.Errorf("merged[EXISTING] = %q, want %q", merged["EXISTING"], "value1")
	}
	if merged["NEW"] != "value2" {
		t.Errorf("merged[NEW] = %q, want %q", merged["NEW"], "value2")
	}
	if merged["OVERWRITE"] != "new" {
		t.Errorf("merged[OVERWRITE] = %q, want %q", merged["OVERWRITE"], "new")
	}
}
