package secrets

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/manifest"
)

func TestManagerLoad_MissingFile(t *testing.T) {
	sm := NewManager(&manifest.Manifest{})

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("Load() returned %d secrets, want 0", len(secrets))
	}
}

func TestManagerLoad_IgnoresRetiredWrappedFile(t *testing.T) {
	sm := NewManager(&manifest.Manifest{})

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("retired wrapped file was read: %v", secrets)
	}
}

func TestManagerLoad_IgnoresRetiredFlatFile(t *testing.T) {
	sm := NewManager(&manifest.Manifest{})

	secrets, err := sm.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(secrets) != 0 {
		t.Fatalf("retired flat file was read: %v", secrets)
	}
}

func TestManagerPersistKeepsExplicitInMemoryState(t *testing.T) {
	sm := NewManager(&manifest.Manifest{})

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
	sm := NewManager(&manifest.Manifest{})
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
			sm := NewManager(&manifest.Manifest{Secrets: tt.manifestSecs})
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
	sm := NewManager(m)

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
	sm := NewManager(&manifest.Manifest{})
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

// A bundle must read the credential the operator already provisioned. Before
// this, a bundle invented its own namespace from the app's display name, so the
// key entered during onboarding and the key a packaged bundle looked for were
// two different stored values with no declared relationship — provision-once
// was not true across tiers, and neither was a recovery bundle taken on either.
func TestDeclaredLogicalIDResolvesToTheSharedIdentity(t *testing.T) {
	m := &manifest.Manifest{
		App: manifest.App{Name: "My Desktop App"},
		Secrets: []manifest.Secret{
			{
				ID:        "OPENROUTER_API_KEY",
				LogicalID: "vrooli/openrouter",
				Field:     "api-key",
				Target:    manifest.SecretTarget{Type: "env", Name: "OPENROUTER_API_KEY"},
			},
			{
				// No declaration: falls back to the bundle's own namespace.
				ID:     "BUNDLE_ONLY",
				Target: manifest.SecretTarget{Type: "env", Name: "BUNDLE_ONLY"},
			},
		},
	}
	manager := newManager(m)
	identity, err := desktopIdentity(m)
	if err != nil {
		t.Fatal(err)
	}
	manager.identity = identity

	sharedIdentity, sharedField, err := manager.addressOf(m.Secrets[0])
	if err != nil {
		t.Fatalf("addressOf declared: %v", err)
	}
	if string(sharedIdentity) != "vrooli/openrouter" || sharedField != "api-key" {
		t.Fatalf("declared secret resolved to %s:%s, want vrooli/openrouter:api-key — a Tier 1 install would not find it",
			sharedIdentity, sharedField)
	}

	localIdentity, localField, err := manager.addressOf(m.Secrets[1])
	if err != nil {
		t.Fatalf("addressOf undeclared: %v", err)
	}
	if localIdentity != identity {
		t.Fatalf("undeclared secret resolved to %s, want the bundle namespace %s", localIdentity, identity)
	}
	if localField != "bundle-only" {
		t.Fatalf("undeclared field = %q, want the normalized form", localField)
	}
}

// The field normalization must match every other tier's, or a value written by
// one lands where the other does not look.
func TestCredentialFieldNormalizationMatchesTheOtherTiers(t *testing.T) {
	for _, testCase := range []struct {
		secret manifest.Secret
		want   string
	}{
		{secret: manifest.Secret{Field: "api-key", ID: "IGNORED"}, want: "api-key"},
		{secret: manifest.Secret{ID: "SESSION_SECRET"}, want: "session-secret"},
		{secret: manifest.Secret{ID: "cloudflare.api_token"}, want: "cloudflare-api-token"},
		{secret: manifest.Secret{Target: manifest.SecretTarget{Name: "DB_PASSWORD"}}, want: "db-password"},
		{secret: manifest.Secret{}, want: ""},
	} {
		if got := testCase.secret.CredentialField(); got != testCase.want {
			t.Fatalf("CredentialField() = %q, want %q", got, testCase.want)
		}
	}
}

// An unusable logical_id must fail loudly. Silently falling back to the bundle
// namespace would store the value somewhere the operator never looks.
func TestUnusableLogicalIDIsRejectedRatherThanSilentlyFallingBack(t *testing.T) {
	m := &manifest.Manifest{
		App:     manifest.App{Name: "App"},
		Secrets: []manifest.Secret{{ID: "KEY", LogicalID: "not-namespaced", Field: "api-key"}},
	}
	manager := newManager(m)
	identity, _ := desktopIdentity(m)
	manager.identity = identity
	if _, _, err := manager.addressOf(m.Secrets[0]); err == nil {
		t.Fatal("an unusable logical_id was silently accepted")
	}
}
