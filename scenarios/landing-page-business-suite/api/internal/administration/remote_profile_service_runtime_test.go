package administration

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	"landing-page-business-suite-api/internal/securevalue"
)

func TestNewRemoteProfileServiceWithRuntimeUsesInjectedSecretResolver(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	lookups := []string{}
	service, err := NewRemoteProfileServiceWithRuntime(
		nil,
		nil,
		func(name string) string {
			lookups = append(lookups, name)
			if name == "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY" {
				return key
			}
			return ""
		},
		func() bool { return true },
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("NewRemoteProfileServiceWithRuntime returned error: %v", err)
	}
	if len(service.EncryptionKey) != 32 {
		t.Fatalf("expected decoded 32-byte encryption key, got %d bytes", len(service.EncryptionKey))
	}
	if len(lookups) != 1 || lookups[0] != "LPBS_REMOTE_PROFILE_ENCRYPTION_KEY" {
		t.Fatalf("expected injected resolver to provide preferred key, got %v", lookups)
	}
}

func TestNewRemoteProfileServiceWithRuntimeRejectsMissingProductionKey(t *testing.T) {
	_, err := NewRemoteProfileServiceWithRuntime(nil, nil, func(string) string { return "" }, func() bool { return true }, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "required in production") {
		t.Fatalf("expected production encryption-key error, got %v", err)
	}
}

func TestNewRemoteProfileServiceWithCredentialResolverPropagatesProviderFailure(t *testing.T) {
	providerErr := fmt.Errorf("provider unavailable: %w", credentialauthority.ErrProviderUnavailable)
	_, err := NewRemoteProfileServiceWithCredentialResolver(
		nil, nil, func(string) string { return "" },
		func(string) (string, error) { return "", providerErr },
		func() bool { return false }, nil, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "resolve remote profile encryption credential") {
		t.Fatalf("expected provider failure to propagate, got %v", err)
	}
}

func TestRemoteProfileServiceUsesInjectedProductionPolicy(t *testing.T) {
	service := &RemoteProfileService{IsProduction: func() bool { return true }}
	_, err := service.normalizeAPIBase("http://example.com/api/v1")
	if err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("expected injected production policy to require HTTPS, got %v", err)
	}
}

func TestRemoteProfileServiceMigrateEncryptionResealsLegacyRows(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE remote_profiles (id INTEGER PRIMARY KEY, encrypted_session TEXT, encryption_state TEXT)`); err != nil {
		t.Fatalf("create remote_profiles: %v", err)
	}
	legacyKey := []byte("01234567890123456789012345678901")
	activeKey := []byte("abcdefghijklmnopqrstuvwxyz123456")
	legacyCiphertext, err := securevalue.Encrypt(legacyKey, "remote-session")
	if err != nil {
		t.Fatalf("encrypt legacy session: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO remote_profiles (id, encrypted_session, encryption_state) VALUES (1, ?, 'legacy')`, legacyCiphertext); err != nil {
		t.Fatalf("insert legacy session: %v", err)
	}

	service := &RemoteProfileService{
		DB:            db,
		EncryptionKey: legacyKey,
		EncryptionRing: securevalue.Ring{Active: 2, Keys: []securevalue.VersionedKey{
			{Version: 1, Key: base64.StdEncoding.EncodeToString(legacyKey)},
			{Version: 2, Key: base64.StdEncoding.EncodeToString(activeKey)},
		}},
	}
	count, err := service.MigrateEncryption(context.Background())
	if err != nil {
		t.Fatalf("migrate remote profile encryption: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one migrated profile, got %d", count)
	}
	var sealed, state string
	if err := db.QueryRow(`SELECT encrypted_session, encryption_state FROM remote_profiles WHERE id = 1`).Scan(&sealed, &state); err != nil {
		t.Fatalf("read migrated profile: %v", err)
	}
	if !strings.HasPrefix(sealed, "v2:") || state != "v2" {
		t.Fatalf("expected v2 sealed state, got ciphertext %q and state %q", sealed, state)
	}
	rotated := *service
	rotated.EncryptionRing = securevalue.Ring{Active: 2, Keys: []securevalue.VersionedKey{{Version: 2, Key: base64.StdEncoding.EncodeToString(activeKey)}}}
	plaintext, err := rotated.Decrypt(sealed)
	if err != nil || plaintext != "remote-session" {
		t.Fatalf("expected migrated session to decrypt with active key: plaintext=%q err=%v", plaintext, err)
	}
}
