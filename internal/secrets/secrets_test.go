package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=4 | LAST: 2026-04-12

func TestNewProjectStoreUsesDefaultBoundaries(t *testing.T) {
	root := t.TempDir()
	store := NewProjectStore(root)

	if store.Root != filepath.Clean(root) {
		t.Fatalf("Root = %q, want %q", store.Root, filepath.Clean(root))
	}
	if store.KeyProvider == nil {
		t.Fatal("KeyProvider is nil")
	}
	if store.EnvLookup == nil {
		t.Fatal("EnvLookup is nil")
	}
	if store.LoadPolicy != LoadPolicyStrict {
		t.Fatalf("LoadPolicy = %v, want %v", store.LoadPolicy, LoadPolicyStrict)
	}
	if store.deps.readFile == nil || store.deps.createTemp == nil || store.deps.rename == nil || store.deps.open == nil || store.deps.openFile == nil || store.deps.mkdirAll == nil || store.deps.removeFile == nil {
		t.Fatal("store deps not fully initialized")
	}
	if store.deps.randReader == nil || store.deps.now == nil || store.deps.sleep == nil {
		t.Fatal("store deps missing timing or entropy boundaries")
	}
}

func TestStoreMethodsUseDefaultDepsWhenConstructedDirectly(t *testing.T) {
	root := t.TempDir()
	store := &Store{
		Root:        root,
		KeyProvider: staticKeyProvider("test-passphrase"),
	}

	if err := store.Save(map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	values, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if values["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", values["API_KEY"])
	}
}

func TestStoreSaveAndLoadEncryptedRoundTrip(t *testing.T) {
	store := newTestStore(t)

	input := map[string]string{
		"POSTGRES_PASSWORD": "secret",
		"POSTGRES_USER":     "vrooli",
	}
	if err := store.Save(input); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(store.EncryptedPath())
	if err != nil {
		t.Fatalf("Stat(%s): %v", store.EncryptedPath(), err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}

	output, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if output["POSTGRES_PASSWORD"] != "secret" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want secret", output["POSTGRES_PASSWORD"])
	}
	if output["POSTGRES_USER"] != "vrooli" {
		t.Fatalf("POSTGRES_USER = %q, want vrooli", output["POSTGRES_USER"])
	}
}

func TestStoreSaveUsesAtomicRename(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(map[string]string{"API_KEY": "first"}); err != nil {
		t.Fatalf("Save initial payload: %v", err)
	}
	before, err := os.ReadFile(store.EncryptedPath())
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", store.EncryptedPath(), err)
	}

	store.deps.rename = func(oldPath, newPath string) error {
		return errors.New("rename failed")
	}
	err = store.Save(map[string]string{"API_KEY": "second"})
	if err == nil || !errors.Is(err, ErrEncryptedWrite) || !strings.Contains(err.Error(), "rename encrypted secrets") {
		t.Fatalf("Save error = %v, want ErrEncryptedWrite rename failure", err)
	}

	after, err := os.ReadFile(store.EncryptedPath())
	if err != nil {
		t.Fatalf("ReadFile(%s) after failed save: %v", store.EncryptedPath(), err)
	}
	if string(after) != string(before) {
		t.Fatal("failed save should not modify the original encrypted file")
	}
}

func TestStoreLoadPrefersEncryptedOverLegacy(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy"}`)
	if err := store.Save(map[string]string{"POSTGRES_PASSWORD": "encrypted"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	values, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := values["POSTGRES_PASSWORD"]; got != "encrypted" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want encrypted", got)
	}
}

func TestStoreLoadFallsBackToLegacyPlaintext(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy"}`)

	values, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if values["POSTGRES_PASSWORD"] != "legacy" {
		t.Fatalf("POSTGRES_PASSWORD = %q, want legacy", values["POSTGRES_PASSWORD"])
	}
}

func TestStoreLoadWithPolicyMakesLegacyFallbackExplicit(t *testing.T) {
	t.Run("strict load fails when encrypted file exists but key is missing", func(t *testing.T) {
		writer := newTestStore(t)
		writeLegacySecrets(t, writer, `{"API_KEY":"legacy"}`)
		if err := writer.Save(map[string]string{"API_KEY": "encrypted"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		reader := NewProjectStore(writer.Root)
		_, err := reader.Load()
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("Load error = %v, want ErrMissingKey", err)
		}
	})

	t.Run("best effort load falls back to legacy when encrypted read cannot succeed", func(t *testing.T) {
		writer := newTestStore(t)
		writeLegacySecrets(t, writer, `{"API_KEY":"legacy"}`)
		if err := writer.Save(map[string]string{"API_KEY": "encrypted"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		reader := NewProjectStore(writer.Root)
		values, err := reader.LoadWithPolicy(LoadPolicyBestEffortLegacy)
		if err != nil {
			t.Fatalf("LoadWithPolicy: %v", err)
		}
		if got := values["API_KEY"]; got != "legacy" {
			t.Fatalf("API_KEY = %q, want legacy", got)
		}
	})

	t.Run("best effort load also falls back when encrypted payload is invalid", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"API_KEY":"legacy"}`)
		writeEncryptedPayload(t, store, `{`)

		values, err := store.LoadWithPolicy(LoadPolicyBestEffortLegacy)
		if err != nil {
			t.Fatalf("LoadWithPolicy: %v", err)
		}
		if got := values["API_KEY"]; got != "legacy" {
			t.Fatalf("API_KEY = %q, want legacy", got)
		}
	})
}

func TestStoreLoadEncryptedPropagatesReadFailure(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	_, err := store.LoadEncrypted()
	if err == nil || !errors.Is(err, ErrEncryptedRead) || !strings.Contains(err.Error(), "read encrypted secrets") {
		t.Fatalf("LoadEncrypted error = %v, want ErrEncryptedRead", err)
	}
}

func TestStoreLoadEncryptedRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		wantErr  error
		wantText string
	}{
		{
			name:     "invalid-json",
			payload:  `{`,
			wantErr:  ErrEncryptedInvalid,
			wantText: "parse encrypted secrets",
		},
		{
			name:     "unsupported-version",
			payload:  `{"version":2,"algorithm":"AES-256-GCM","nonce":"bm9uY2U=","ciphertext":"Y2lwaGVy"}`,
			wantErr:  ErrUnsupportedVersion,
			wantText: "unsupported secrets version",
		},
		{
			name:     "unsupported-algorithm",
			payload:  `{"version":1,"algorithm":"age","nonce":"bm9uY2U=","ciphertext":"Y2lwaGVy"}`,
			wantErr:  ErrUnsupportedAlgorithm,
			wantText: "unsupported secrets algorithm",
		},
		{
			name:     "invalid-nonce",
			payload:  `{"version":1,"algorithm":"AES-256-GCM","nonce":"***","ciphertext":"Y2lwaGVy"}`,
			wantErr:  ErrEncryptedInvalid,
			wantText: "decode nonce",
		},
		{
			name:     "invalid-ciphertext",
			payload:  `{"version":1,"algorithm":"AES-256-GCM","nonce":"bm9uY2U=","ciphertext":"***"}`,
			wantErr:  ErrEncryptedInvalid,
			wantText: "decode ciphertext",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			writeEncryptedPayload(t, store, tc.payload)
			_, err := store.LoadEncrypted()
			if err == nil || !errors.Is(err, tc.wantErr) || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("LoadEncrypted error = %v, want %v containing %q", err, tc.wantErr, tc.wantText)
			}
		})
	}
}

func TestStoreLoadEncryptedRejectsWrongKeyAndTampering(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	wrongKeyStore := NewProjectStore(store.Root)
	wrongKeyStore.KeyProvider = staticKeyProvider("different-passphrase")
	_, err := wrongKeyStore.LoadEncrypted()
	if err == nil || !errors.Is(err, ErrDecryptFailed) || !strings.Contains(err.Error(), "decrypt secrets") {
		t.Fatalf("LoadEncrypted wrong-key error = %v, want ErrDecryptFailed", err)
	}

	payloadBytes, err := os.ReadFile(store.EncryptedPath())
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", store.EncryptedPath(), err)
	}
	var payload encryptedFile
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		t.Fatalf("Unmarshal encrypted payload: %v", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		t.Fatalf("Decode ciphertext: %v", err)
	}
	ciphertext[0] ^= 0x01
	payload.Ciphertext = base64.StdEncoding.EncodeToString(ciphertext)
	mutated, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal mutated payload: %v", err)
	}
	writeEncryptedPayload(t, store, string(mutated))

	_, err = store.LoadEncrypted()
	if err == nil || !errors.Is(err, ErrDecryptFailed) || !strings.Contains(err.Error(), "decrypt secrets") {
		t.Fatalf("LoadEncrypted tamper error = %v, want ErrDecryptFailed", err)
	}
}

func TestStoreLoadLegacyReturnsEmptyWhenMissing(t *testing.T) {
	store := newTestStore(t)
	values, err := store.LoadLegacy()
	if err != nil {
		t.Fatalf("LoadLegacy: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("len(values) = %d, want 0", len(values))
	}
}

func TestStoreLoadLegacyPropagatesReadFailure(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecrets(t, store, `{"API_KEY":"legacy"}`)
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	_, err := store.LoadLegacy()
	if err == nil || !errors.Is(err, ErrLegacyRead) || !strings.Contains(err.Error(), "read legacy secrets") {
		t.Fatalf("LoadLegacy error = %v, want ErrLegacyRead", err)
	}
}

func TestStoreLoadLegacyRejectsInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecrets(t, store, `{`)
	_, err := store.LoadLegacy()
	if err == nil || !errors.Is(err, ErrLegacyInvalid) || !strings.Contains(err.Error(), "parse legacy secrets") {
		t.Fatalf("LoadLegacy error = %v, want ErrLegacyInvalid", err)
	}
}

func TestStoreRejectsInsecureSecretFilePermissions(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecretsWithMode(t, store, `{"API_KEY":"legacy"}`, 0o644)

	_, err := store.LoadLegacy()
	if err == nil || !errors.Is(err, ErrInsecurePermissions) {
		t.Fatalf("LoadLegacy error = %v, want ErrInsecurePermissions", err)
	}
}

func TestStoreRejectsSymlinkSecretFiles(t *testing.T) {
	store := newTestStore(t)
	target := filepath.Join(t.TempDir(), "target.json")
	if err := os.WriteFile(target, []byte("{\"API_KEY\":\"legacy\"}\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(%s): %v", target, err)
	}
	path := store.LegacyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("Symlink(%s -> %s): %v", path, target, err)
	}

	_, err := store.LoadLegacy()
	if err == nil || !errors.Is(err, ErrSymlinkPath) {
		t.Fatalf("LoadLegacy error = %v, want ErrSymlinkPath", err)
	}
}

func TestStoreSaveHonorsInjectedBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		override   func(*Store)
		wantErr    error
		wantSubstr string
	}{
		{
			name: "mkdir-failure",
			override: func(store *Store) {
				store.deps.mkdirAll = func(path string, mode os.FileMode) error {
					return errors.New("mkdir failed")
				}
			},
			wantSubstr: "mkdir secrets dir",
		},
		{
			name: "temp-create-failure",
			override: func(store *Store) {
				store.deps.createTemp = func(dir, pattern string) (*os.File, error) {
					return nil, errors.New("temp failed")
				}
			},
			wantErr:    ErrEncryptedWrite,
			wantSubstr: "create temporary secrets file",
		},
		{
			name: "rename-failure",
			override: func(store *Store) {
				store.deps.rename = func(oldPath, newPath string) error {
					return errors.New("rename failed")
				}
			},
			wantErr:    ErrEncryptedWrite,
			wantSubstr: "rename encrypted secrets",
		},
		{
			name: "dir-open-failure-after-rename",
			override: func(store *Store) {
				store.deps.open = func(path string) (*os.File, error) {
					return nil, errors.New("open failed")
				}
			},
			wantErr:    ErrEncryptedWrite,
			wantSubstr: "open secrets dir for sync",
		},
		{
			name: "nonce-failure",
			override: func(store *Store) {
				store.deps.randReader = failingReader{err: errors.New("entropy failed")}
			},
			wantSubstr: "generate nonce",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			tc.override(store)
			err := store.Save(map[string]string{"API_KEY": "secret"})
			if err == nil || !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("Save error = %v, want substring %q", err, tc.wantSubstr)
			}
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("Save error = %v, want errors.Is(_, %v)", err, tc.wantErr)
			}
		})
	}
}

func TestStoreSaveRequiresEncryptionKey(t *testing.T) {
	store := newTestStore(t)
	store.KeyProvider = nil

	err := store.Save(map[string]string{"API_KEY": "secret"})
	if !errors.Is(err, ErrMissingKey) {
		t.Fatalf("Save error = %v, want ErrMissingKey", err)
	}
}

func TestStoreSaveRejectsReservedAndConflictingKeys(t *testing.T) {
	store := newTestStore(t)

	err := store.Save(map[string]string{"_metadata": "value"})
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Save reserved-key error = %v, want ErrInvalidInput", err)
	}

	err = store.Save(map[string]string{
		" API_KEY ": "first",
		"API_KEY":   "second",
	})
	if err == nil || !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("Save duplicate-normalized-key error = %v, want ErrInvalidInput", err)
	}
}

func TestStoreSaveKeyValidationAndMerge(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveKey("   ", "value"); err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("SaveKey blank-name error = %v", err)
	}
	if err := store.SaveKey("_metadata", "value"); err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "reserved for metadata") {
		t.Fatalf("SaveKey reserved-name error = %v", err)
	}
	if err := store.SaveKey("NAME", ""); err == nil || !errors.Is(err, ErrInvalidInput) || !strings.Contains(err.Error(), "secret value is required") {
		t.Fatalf("SaveKey blank-value error = %v", err)
	}

	if err := store.Save(map[string]string{"POSTGRES_PASSWORD": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.SaveKey("POSTGRES_USER", "vrooli"); err != nil {
		t.Fatalf("SaveKey: %v", err)
	}

	values, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("len(values) = %d, want 2", len(values))
	}
	if values["POSTGRES_PASSWORD"] != "secret" || values["POSTGRES_USER"] != "vrooli" {
		t.Fatalf("values = %#v", values)
	}
}

func TestStoreSaveKeyPropagatesLoadFailure(t *testing.T) {
	store := newTestStore(t)
	if err := store.Save(map[string]string{"POSTGRES_PASSWORD": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	err := store.SaveKey("POSTGRES_USER", "vrooli")
	if err == nil || !errors.Is(err, ErrEncryptedRead) || !strings.Contains(err.Error(), "read encrypted secrets") {
		t.Fatalf("SaveKey error = %v, want ErrEncryptedRead", err)
	}
}

func TestStoreSaveKeyUsesExclusiveLockForReadModifyWrite(t *testing.T) {
	store := newTestStore(t)
	lockPath := store.LockPath()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(lockPath), err)
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("OpenFile(%s): %v", lockPath, err)
	}
	t.Cleanup(func() {
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
	})

	now := time.Now()
	store.deps.now = sequentialNow(now, now.Add(lockTimeout), now.Add(lockTimeout))
	store.deps.sleep = func(time.Duration) {}

	err = store.SaveKey("POSTGRES_USER", "vrooli")
	if err == nil || !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("SaveKey error = %v, want ErrLockTimeout", err)
	}
}

func TestStoreMigrateLegacyScenarios(t *testing.T) {
	t.Run("migrates and removes source", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy-secret"}`)

		migrated, err := store.MigrateLegacy(true)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if !migrated {
			t.Fatal("expected migration")
		}
		if _, err := os.Stat(store.LegacyPath()); !os.IsNotExist(err) {
			t.Fatalf("legacy file still exists or wrong error: %v", err)
		}
		values, err := store.Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if values["POSTGRES_PASSWORD"] != "legacy-secret" {
			t.Fatalf("POSTGRES_PASSWORD = %q, want legacy-secret", values["POSTGRES_PASSWORD"])
		}
	})

	t.Run("returns false when legacy missing", func(t *testing.T) {
		store := newTestStore(t)
		migrated, err := store.MigrateLegacy(true)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if migrated {
			t.Fatal("expected no migration")
		}
	})

	t.Run("migrates without removing source when requested", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy-secret"}`)

		migrated, err := store.MigrateLegacy(false)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if !migrated {
			t.Fatal("expected migration")
		}
		if _, err := os.Stat(store.LegacyPath()); err != nil {
			t.Fatalf("legacy file missing after keep-source migration: %v", err)
		}
	})

	t.Run("matching encrypted secrets only remove legacy when requested", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"same-secret"}`)
		if err := store.Save(map[string]string{"POSTGRES_PASSWORD": "same-secret"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		migrated, err := store.MigrateLegacy(true)
		if err != nil {
			t.Fatalf("MigrateLegacy: %v", err)
		}
		if !migrated {
			t.Fatal("expected cleanup migration when removing matching legacy source")
		}
		if _, err := os.Stat(store.LegacyPath()); !os.IsNotExist(err) {
			t.Fatalf("legacy file still exists or wrong error: %v", err)
		}
	})

	t.Run("conflicting encrypted secrets fail closed", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy-secret"}`)
		if err := store.Save(map[string]string{"POSTGRES_PASSWORD": "encrypted-secret"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		migrated, err := store.MigrateLegacy(true)
		if migrated {
			t.Fatal("expected migration to report false when encrypted secrets conflict")
		}
		if err == nil || !errors.Is(err, ErrMigrationConflict) {
			t.Fatalf("MigrateLegacy error = %v, want ErrMigrationConflict", err)
		}

		values, loadErr := store.Load()
		if loadErr != nil {
			t.Fatalf("Load after conflict: %v", loadErr)
		}
		if got := values["POSTGRES_PASSWORD"]; got != "encrypted-secret" {
			t.Fatalf("POSTGRES_PASSWORD = %q, want encrypted-secret", got)
		}
		if _, statErr := os.Stat(store.LegacyPath()); statErr != nil {
			t.Fatalf("legacy file should remain after conflict: %v", statErr)
		}
	})

	t.Run("remove failure surfaces after save", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy-secret"}`)
		store.deps.removeFile = func(path string) error { return errors.New("remove failed") }

		migrated, err := store.MigrateLegacy(true)
		if !migrated {
			t.Fatal("expected migration to report true when save completed")
		}
		if err == nil || !strings.Contains(err.Error(), "remove legacy secrets") {
			t.Fatalf("MigrateLegacy error = %v, want remove error", err)
		}
	})

	t.Run("save failure aborts migration", func(t *testing.T) {
		store := newTestStore(t)
		writeLegacySecrets(t, store, `{"POSTGRES_PASSWORD":"legacy-secret"}`)
		store.KeyProvider = nil

		migrated, err := store.MigrateLegacy(true)
		if migrated {
			t.Fatal("expected migration to report false when save fails")
		}
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("MigrateLegacy error = %v, want ErrMissingKey", err)
		}
	})
}

func TestStoreResolveBoundaries(t *testing.T) {
	t.Run("prefers stored secret over env fallback", func(t *testing.T) {
		store := newTestStore(t)
		if err := store.Save(map[string]string{"API_KEY": "stored"}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		store.EnvLookup = staticEnvLookup(map[string]string{"API_KEY": "from-env"})

		value, ok, err := store.Resolve("API_KEY")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !ok || value != "stored" {
			t.Fatalf("Resolve = (%q, %t), want (stored, true)", value, ok)
		}
	})

	t.Run("falls back to env value", func(t *testing.T) {
		store := newTestStore(t)
		store.EnvLookup = staticEnvLookup(map[string]string{"API_KEY": "from-env"})

		value, ok, err := store.Resolve("API_KEY")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !ok || value != "from-env" {
			t.Fatalf("Resolve = (%q, %t), want (from-env, true)", value, ok)
		}
	})

	t.Run("ignores blank env value", func(t *testing.T) {
		store := newTestStore(t)
		store.EnvLookup = staticEnvLookup(map[string]string{"API_KEY": "   "})

		value, ok, err := store.Resolve("API_KEY")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if ok || value != "" {
			t.Fatalf("Resolve = (%q, %t), want empty false", value, ok)
		}
	})

	t.Run("returns load error before env fallback", func(t *testing.T) {
		store := newTestStore(t)
		if err := store.Save(map[string]string{"OTHER_KEY": "secret"}); err != nil {
			t.Fatalf("Save: %v", err)
		}
		store.deps.readFile = func(path string) ([]byte, error) {
			return nil, errors.New("read failed")
		}
		store.EnvLookup = staticEnvLookup(map[string]string{"API_KEY": "from-env"})

		_, _, err := store.Resolve("API_KEY")
		if err == nil || !errors.Is(err, ErrEncryptedRead) {
			t.Fatalf("Resolve error = %v, want ErrEncryptedRead", err)
		}
	})

	t.Run("returns empty when env lookup absent and no stored value", func(t *testing.T) {
		store := newTestStore(t)
		store.EnvLookup = nil

		value, ok, err := store.Resolve("API_KEY")
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if ok || value != "" {
			t.Fatalf("Resolve = (%q, %t), want empty false", value, ok)
		}
	})
}

func TestDeriveKeyBoundaries(t *testing.T) {
	t.Run("accepts base64 encoded 32-byte key", func(t *testing.T) {
		raw := bytesRepeat(0x2a, 32)
		encoded := base64.StdEncoding.EncodeToString(raw)
		key, err := deriveKey(encoded)
		if err != nil {
			t.Fatalf("deriveKey: %v", err)
		}
		if string(key) != string(raw) {
			t.Fatalf("deriveKey returned unexpected bytes")
		}
	})

	t.Run("hashes passphrase when not base64 key", func(t *testing.T) {
		key, err := deriveKey("passphrase")
		if err != nil {
			t.Fatalf("deriveKey: %v", err)
		}
		if len(key) != 32 {
			t.Fatalf("len(key) = %d, want 32", len(key))
		}
	})

	t.Run("rejects empty input", func(t *testing.T) {
		_, err := deriveKey("   ")
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("deriveKey error = %v, want ErrMissingKey", err)
		}
	})
}

func TestEncryptionKeyBoundaries(t *testing.T) {
	t.Run("requires key provider", func(t *testing.T) {
		store := newTestStore(t)
		store.KeyProvider = nil

		_, err := store.encryptionKey()
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("encryptionKey error = %v, want ErrMissingKey", err)
		}
	})

	t.Run("requires configured key", func(t *testing.T) {
		store := newTestStore(t)
		store.KeyProvider = func() (string, bool) { return "", false }

		_, err := store.encryptionKey()
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("encryptionKey error = %v, want ErrMissingKey", err)
		}
	})
}

func TestParseSecretMapRequiresStringValues(t *testing.T) {
	values, err := parseSecretMap([]byte(`{
		"_metadata":{"environment":"development"},
		"STRING":"value"
	}`))
	if err != nil {
		t.Fatalf("parseSecretMap: %v", err)
	}
	if values["STRING"] != "value" {
		t.Fatalf("STRING = %q, want value", values["STRING"])
	}
	if _, ok := values["_metadata"]; ok {
		t.Fatalf("expected metadata keys to be ignored, got %#v", values)
	}
}

func TestParseSecretMapRejectsInvalidJSON(t *testing.T) {
	_, err := parseSecretMap([]byte(`{`))
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestParseSecretMapRejectsNonStringValues(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{name: "bool", payload: `{"BOOL":true}`},
		{name: "number", payload: `{"INT":42}`},
		{name: "object", payload: `{"OBJECT":{"nested":"x"}}`},
		{name: "array", payload: `{"ARRAY":[1,2,3]}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSecretMap([]byte(tc.payload))
			if err == nil || !strings.Contains(err.Error(), "must be a JSON string") {
				t.Fatalf("parseSecretMap error = %v, want string-type validation error", err)
			}
		})
	}
}

func TestEncryptValuesWithReaderRejectsEntropyFailure(t *testing.T) {
	key, err := deriveKey("passphrase")
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}
	_, err = encryptValuesWithReader(map[string]string{"API_KEY": "secret"}, key, failingReader{err: errors.New("entropy failed")})
	if err == nil || !strings.Contains(err.Error(), "generate nonce") {
		t.Fatalf("encryptValuesWithReader error = %v, want generate nonce", err)
	}
}

func TestEncryptValuesWithReaderRejectsInvalidKeyLength(t *testing.T) {
	_, err := encryptValuesWithReader(map[string]string{"API_KEY": "secret"}, []byte("short"), strings.NewReader("0123456789abcdef"))
	if err == nil || !strings.Contains(err.Error(), "create cipher") {
		t.Fatalf("encryptValuesWithReader error = %v, want create cipher error", err)
	}
}

func TestEncryptValuesWrapperRoundTrip(t *testing.T) {
	key, err := deriveKey("passphrase")
	if err != nil {
		t.Fatalf("deriveKey: %v", err)
	}

	payload, err := encryptValues(map[string]string{"API_KEY": "secret"}, key)
	if err != nil {
		t.Fatalf("encryptValues: %v", err)
	}
	plaintext, err := decryptPayload(payload, key)
	if err != nil {
		t.Fatalf("decryptPayload: %v", err)
	}
	values, err := parseSecretMap(plaintext)
	if err != nil {
		t.Fatalf("parseSecretMap: %v", err)
	}
	if values["API_KEY"] != "secret" {
		t.Fatalf("API_KEY = %q, want secret", values["API_KEY"])
	}
}

func TestDecryptPayloadRejectsInvalidKeyLength(t *testing.T) {
	_, err := decryptPayload(encryptedFile{
		Version:    encryptionVersion,
		Algorithm:  encryptionAlgorithm,
		Nonce:      base64.StdEncoding.EncodeToString(bytesRepeat(0x01, 12)),
		Ciphertext: base64.StdEncoding.EncodeToString([]byte("ciphertext")),
	}, []byte("short"))
	if err == nil || !strings.Contains(err.Error(), "create cipher") {
		t.Fatalf("decryptPayload error = %v, want create cipher error", err)
	}
}

func TestStoreDepsAppliesDefaultsForPartialInjection(t *testing.T) {
	store := &Store{
		Root: t.TempDir(),
		deps: storeDeps{
			readFile: func(path string) ([]byte, error) { return []byte("fixture"), nil },
		},
	}

	deps := store.storeDeps()
	if deps.readFile == nil || deps.createTemp == nil || deps.rename == nil || deps.open == nil || deps.openFile == nil || deps.removeFile == nil || deps.mkdirAll == nil || deps.randReader == nil || deps.now == nil || deps.sleep == nil {
		t.Fatal("storeDeps did not fill default boundaries")
	}
	data, err := deps.readFile("ignored")
	if err != nil {
		t.Fatalf("readFile: %v", err)
	}
	if string(data) != "fixture" {
		t.Fatalf("readFile returned %q, want fixture", string(data))
	}
}

func TestStoreSaveKeyRecoversStaleLock(t *testing.T) {
	store := newTestStore(t)
	lockPath := store.LockPath()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(lockPath), err)
	}
	if err := os.WriteFile(lockPath, []byte("pid=999999\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", lockPath, err)
	}
	staleTime := time.Now().Add(-lockStaleAfter - time.Second)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("chtimes %s: %v", lockPath, err)
	}

	if err := store.SaveKey("API_KEY", "secret"); err != nil {
		t.Fatalf("SaveKey: %v", err)
	}

	values, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := values["API_KEY"]; got != "secret" {
		t.Fatalf("API_KEY = %q, want secret", got)
	}
}

func TestStoreSaveKeyDoesNotBreakLiveStaleAgedLock(t *testing.T) {
	store := newTestStore(t)
	lockPath := store.LockPath()
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(lockPath), err)
	}
	if err := os.WriteFile(lockPath, []byte("pid="+strconv.Itoa(os.Getpid())+"\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", lockPath, err)
	}
	staleTime := time.Now().Add(-lockStaleAfter - time.Second)
	if err := os.Chtimes(lockPath, staleTime, staleTime); err != nil {
		t.Fatalf("chtimes %s: %v", lockPath, err)
	}
	now := time.Now()
	store.deps.now = sequentialNow(now, now.Add(lockTimeout), now.Add(lockTimeout))
	store.deps.sleep = func(time.Duration) {}

	err := store.SaveKey("POSTGRES_USER", "vrooli")
	if err == nil || !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("SaveKey error = %v, want ErrLockTimeout", err)
	}
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	store := NewProjectStore(root)
	store.KeyProvider = staticKeyProvider("test-passphrase")
	return store
}

func staticKeyProvider(value string) KeyProvider {
	return func() (string, bool) {
		return value, true
	}
}

func staticEnvLookup(values map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

func sequentialNow(values ...time.Time) func() time.Time {
	index := 0
	return func() time.Time {
		if len(values) == 0 {
			return time.Now()
		}
		if index >= len(values) {
			return values[len(values)-1]
		}
		current := values[index]
		index++
		return current
	}
}

func writeLegacySecrets(t *testing.T, store *Store, contents string) {
	t.Helper()
	path := store.LegacyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeLegacySecretsWithMode(t *testing.T, store *Store, contents string, mode os.FileMode) {
	t.Helper()
	path := store.LegacyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), mode); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func writeEncryptedPayload(t *testing.T, store *Store, contents string) {
	t.Helper()
	path := store.EncryptedPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type failingReader struct {
	err error
}

func (r failingReader) Read(p []byte) (int, error) {
	if r.err == nil {
		return 0, io.ErrUnexpectedEOF
	}
	return 0, r.err
}

func bytesRepeat(value byte, count int) []byte {
	buf := make([]byte, count)
	for i := range buf {
		buf[i] = value
	}
	return buf
}
