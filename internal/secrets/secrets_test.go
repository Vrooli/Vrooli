package secrets

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// AI_CHECK: GO_MIGRATION_TEST_QUALITY=2 | LAST: 2026-04-11

func TestNewProjectStoreUsesDefaultBoundaries(t *testing.T) {
	root := t.TempDir()
	store := NewProjectStore(root)

	if store.Root != filepath.Clean(root) {
		t.Fatalf("Root = %q, want %q", store.Root, filepath.Clean(root))
	}
	if store.KeySource == nil {
		t.Fatal("KeySource is nil")
	}
	if store.deps.readFile == nil || store.deps.writeFile == nil || store.deps.mkdirAll == nil || store.deps.removeFile == nil {
		t.Fatal("store deps not fully initialized")
	}
	if store.deps.randReader == nil {
		t.Fatal("randReader is nil")
	}
}

func TestStoreMethodsUseDefaultDepsWhenConstructedDirectly(t *testing.T) {
	root := t.TempDir()
	store := &Store{
		Root:      root,
		KeySource: staticKeySource("test-passphrase"),
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

func TestStoreLoadEncryptedRequiresKey(t *testing.T) {
	writer := newTestStore(t)
	if err := writer.Save(map[string]string{"API_KEY": "secret"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reader := NewProjectStore(writer.Root)
	_, err := reader.Load()
	if !errors.Is(err, ErrMissingKey) {
		t.Fatalf("Load error = %v, want ErrMissingKey", err)
	}
}

func TestStoreLoadEncryptedPropagatesReadFailure(t *testing.T) {
	store := newTestStore(t)
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	_, err := store.LoadEncrypted()
	if err == nil || !strings.Contains(err.Error(), "read encrypted secrets") {
		t.Fatalf("LoadEncrypted error = %v, want wrapped read error", err)
	}
}

func TestStoreLoadEncryptedRejectsInvalidPayloads(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		wantText string
	}{
		{
			name:     "invalid-json",
			payload:  `{`,
			wantText: "parse encrypted secrets",
		},
		{
			name:     "unsupported-version",
			payload:  `{"version":2,"algorithm":"AES-256-GCM","nonce":"bm9uY2U=","ciphertext":"Y2lwaGVy"}`,
			wantText: "unsupported secrets version",
		},
		{
			name:     "unsupported-algorithm",
			payload:  `{"version":1,"algorithm":"age","nonce":"bm9uY2U=","ciphertext":"Y2lwaGVy"}`,
			wantText: "unsupported secrets algorithm",
		},
		{
			name:     "invalid-nonce",
			payload:  `{"version":1,"algorithm":"AES-256-GCM","nonce":"***","ciphertext":"Y2lwaGVy"}`,
			wantText: "decode nonce",
		},
		{
			name:     "invalid-ciphertext",
			payload:  `{"version":1,"algorithm":"AES-256-GCM","nonce":"bm9uY2U=","ciphertext":"***"}`,
			wantText: "decode ciphertext",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store := newTestStore(t)
			writeEncryptedPayload(t, store, tc.payload)
			_, err := store.LoadEncrypted()
			if err == nil || !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("LoadEncrypted error = %v, want substring %q", err, tc.wantText)
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
	wrongKeyStore.KeySource = staticKeySource("different-passphrase")
	_, err := wrongKeyStore.LoadEncrypted()
	if err == nil || !strings.Contains(err.Error(), "decrypt secrets") {
		t.Fatalf("LoadEncrypted wrong-key error = %v, want decrypt error", err)
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
	if err == nil || !strings.Contains(err.Error(), "decrypt secrets") {
		t.Fatalf("LoadEncrypted tamper error = %v, want decrypt error", err)
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
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	_, err := store.LoadLegacy()
	if err == nil || !strings.Contains(err.Error(), "read legacy secrets") {
		t.Fatalf("LoadLegacy error = %v, want wrapped read error", err)
	}
}

func TestStoreLoadLegacyRejectsInvalidJSON(t *testing.T) {
	store := newTestStore(t)
	writeLegacySecrets(t, store, `{`)
	_, err := store.LoadLegacy()
	if err == nil || !strings.Contains(err.Error(), "parse legacy secrets") {
		t.Fatalf("LoadLegacy error = %v, want parse error", err)
	}
}

func TestStoreSaveHonorsInjectedBoundaries(t *testing.T) {
	tests := []struct {
		name       string
		override   func(*Store)
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
			name: "write-failure",
			override: func(store *Store) {
				store.deps.writeFile = func(path string, data []byte, mode os.FileMode) error {
					return errors.New("write failed")
				}
			},
			wantSubstr: "write encrypted secrets",
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
		})
	}
}

func TestStoreSaveRequiresEncryptionKey(t *testing.T) {
	store := newTestStore(t)
	store.KeySource = nil

	err := store.Save(map[string]string{"API_KEY": "secret"})
	if !errors.Is(err, ErrMissingKey) {
		t.Fatalf("Save error = %v, want ErrMissingKey", err)
	}
}

func TestStoreSaveKeyValidationAndMerge(t *testing.T) {
	store := newTestStore(t)

	if err := store.SaveKey("   ", "value"); err == nil || !strings.Contains(err.Error(), "secret name is required") {
		t.Fatalf("SaveKey blank-name error = %v", err)
	}
	if err := store.SaveKey("NAME", ""); err == nil || !strings.Contains(err.Error(), "secret value is required") {
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
	store.deps.readFile = func(path string) ([]byte, error) {
		return nil, errors.New("read failed")
	}

	err := store.SaveKey("POSTGRES_USER", "vrooli")
	if err == nil || !strings.Contains(err.Error(), "read encrypted secrets") {
		t.Fatalf("SaveKey error = %v, want wrapped load error", err)
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
		store.KeySource = nil

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
		store.KeySource = func(key string) (string, bool) {
			if key == KeyEnvVar {
				return "test-passphrase", true
			}
			if key == "API_KEY" {
				return "from-env", true
			}
			return "", false
		}

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
		store.KeySource = func(key string) (string, bool) {
			if key == "API_KEY" {
				return "from-env", true
			}
			return "", false
		}

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
		store.KeySource = func(key string) (string, bool) {
			if key == "API_KEY" {
				return "   ", true
			}
			return "", false
		}

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
		store.deps.readFile = func(path string) ([]byte, error) {
			return nil, errors.New("read failed")
		}
		store.KeySource = func(key string) (string, bool) {
			if key == "API_KEY" {
				return "from-env", true
			}
			if key == KeyEnvVar {
				return "test-passphrase", true
			}
			return "", false
		}

		_, _, err := store.Resolve("API_KEY")
		if err == nil || !strings.Contains(err.Error(), "read encrypted secrets") {
			t.Fatalf("Resolve error = %v, want wrapped load error", err)
		}
	})

	t.Run("returns empty when key source absent and no stored value", func(t *testing.T) {
		store := newTestStore(t)
		store.KeySource = nil

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
	t.Run("requires key source", func(t *testing.T) {
		store := newTestStore(t)
		store.KeySource = nil

		_, err := store.encryptionKey()
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("encryptionKey error = %v, want ErrMissingKey", err)
		}
	})

	t.Run("requires configured env var", func(t *testing.T) {
		store := newTestStore(t)
		store.KeySource = func(key string) (string, bool) { return "", false }

		_, err := store.encryptionKey()
		if !errors.Is(err, ErrMissingKey) {
			t.Fatalf("encryptionKey error = %v, want ErrMissingKey", err)
		}
	})
}

func TestParseSecretMapConvertsSupportedScalarsAndIgnoresCompositeValues(t *testing.T) {
	values, err := parseSecretMap([]byte(`{
		"STRING":"value",
		"BOOL_TRUE":true,
		"BOOL_FALSE":false,
		"INT":42,
		"FLOAT":4.25,
		"OBJECT":{"nested":"x"},
		"ARRAY":[1,2,3]
	}`))
	if err != nil {
		t.Fatalf("parseSecretMap: %v", err)
	}

	want := map[string]string{
		"STRING":     "value",
		"BOOL_TRUE":  "true",
		"BOOL_FALSE": "false",
		"INT":        "42",
		"FLOAT":      "4.25",
	}
	if len(values) != len(want) {
		t.Fatalf("len(values) = %d, want %d (%#v)", len(values), len(want), values)
	}
	for key, expected := range want {
		if values[key] != expected {
			t.Fatalf("%s = %q, want %q", key, values[key], expected)
		}
	}
	if _, exists := values["OBJECT"]; exists {
		t.Fatal("OBJECT should be ignored")
	}
	if _, exists := values["ARRAY"]; exists {
		t.Fatal("ARRAY should be ignored")
	}
}

func TestParseSecretMapRejectsInvalidJSON(t *testing.T) {
	_, err := parseSecretMap([]byte(`{`))
	if err == nil {
		t.Fatal("expected parse error")
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
	if deps.readFile == nil || deps.writeFile == nil || deps.removeFile == nil || deps.mkdirAll == nil || deps.randReader == nil {
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

func newTestStore(t *testing.T) *Store {
	t.Helper()
	root := t.TempDir()
	store := NewProjectStore(root)
	store.KeySource = staticKeySource("test-passphrase")
	return store
}

func staticKeySource(value string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		if key == KeyEnvVar {
			return value, true
		}
		return "", false
	}
}

func writeLegacySecrets(t *testing.T, store *Store, contents string) {
	t.Helper()
	path := store.LegacyPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents+"\n"), 0o644); err != nil {
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
