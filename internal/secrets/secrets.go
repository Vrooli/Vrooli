// Package secrets manages project-level secret persistence under .vrooli/.
//
// The package supports two on-disk formats:
//   - .vrooli/secrets.enc.json: the current encrypted format
//   - .vrooli/secrets.json: the legacy plaintext format used during migration
//
// Store.Load defaults to strict behavior. If an encrypted file exists but cannot
// be read, validated, or decrypted, strict loads fail rather than silently
// falling back. Callers that are explicitly migration-tolerant can opt into
// LoadPolicyBestEffortLegacy via Store.LoadWithPolicy.
//
// Secret files are expected to be regular files with private permissions.
// Symlinks are rejected, and on Unix-like platforms group/world-readable secret
// files are rejected before parsing.
//
// The project-level store treats encrypted secrets as authoritative once they
// exist. Migration helpers may read legacy plaintext during controlled
// transition paths, but they do not overwrite a readable encrypted file with
// conflicting legacy state.
//
// Keys that begin with "_" are reserved for metadata in legacy plaintext files.
// Those keys are ignored on read and rejected on encrypted writes so write and
// read behavior stay symmetric.
//
// VROOLI_SECRETS_KEY accepts either:
//   - a base64-encoded 32-byte AES key
//   - an arbitrary passphrase, which is normalized through SHA-256
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	KeyEnvVar           = "VROOLI_SECRETS_KEY"
	ProjectSecretsPath  = ".vrooli/secrets.enc.json"
	LegacySecretsPath   = ".vrooli/secrets.json"
	lockFileName        = "secrets.lock"
	encryptionAlgorithm = "AES-256-GCM"
	encryptionVersion   = 1
	lockTimeout         = 5 * time.Second
	lockRetryInterval   = 50 * time.Millisecond
	lockStaleAfter      = 30 * time.Second
)

var (
	ErrMissingKey           = errors.New("missing VROOLI_SECRETS_KEY for encrypted secrets")
	ErrInvalidInput         = errors.New("invalid secrets input")
	ErrEncryptedRead        = errors.New("encrypted secrets read failed")
	ErrLegacyRead           = errors.New("legacy secrets read failed")
	ErrEncryptedWrite       = errors.New("encrypted secrets write failed")
	ErrEncryptedInvalid     = errors.New("invalid encrypted secrets payload")
	ErrLegacyInvalid        = errors.New("invalid legacy secrets payload")
	ErrInvalidSecretData    = errors.New("invalid secrets data")
	ErrUnsupportedVersion   = errors.New("unsupported secrets version")
	ErrUnsupportedAlgorithm = errors.New("unsupported secrets algorithm")
	ErrDecryptFailed        = errors.New("decrypt secrets failed")
	ErrLockTimeout          = errors.New("timed out waiting for secrets lock")
	ErrInsecurePermissions  = errors.New("secret file has insecure permissions")
	ErrSymlinkPath          = errors.New("secret file must not be a symlink")
	ErrMigrationConflict    = errors.New("legacy secrets conflict with encrypted secrets")
)

type (
	LookupFunc  func(string) (string, bool)
	KeyProvider func() (string, bool)
)

type LoadPolicy int

const (
	// LoadPolicyStrict requires the encrypted file to be authoritative whenever it
	// exists. Any encrypted-file read, validation, or decrypt error is returned to
	// the caller without falling back to legacy plaintext.
	LoadPolicyStrict LoadPolicy = iota
	// LoadPolicyBestEffortLegacy allows callers to fall back to the legacy
	// plaintext file when encrypted secrets exist but cannot be used. This is
	// intended only for explicitly migration-tolerant read paths.
	LoadPolicyBestEffortLegacy
)

type encryptedFile struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Store struct {
	Root        string
	KeyProvider KeyProvider
	EnvLookup   LookupFunc
	LoadPolicy  LoadPolicy
	deps        storeDeps
}

type storeDeps struct {
	readFile   func(string) ([]byte, error)
	removeFile func(string) error
	mkdirAll   func(string, os.FileMode) error
	createTemp func(string, string) (*os.File, error)
	rename     func(string, string) error
	open       func(string) (*os.File, error)
	openFile   func(string, int, os.FileMode) (*os.File, error)
	lstat      func(string) (os.FileInfo, error)
	randReader io.Reader
	now        func() time.Time
	sleep      func(time.Duration)
}

// NewProjectStore returns a Store rooted at the given project path.
//
// The default store is strict, reads the encryption key from
// VROOLI_SECRETS_KEY, and uses process environment lookup for Resolve
// fallbacks.
func NewProjectStore(root string) *Store {
	return &Store{
		Root: filepath.Clean(root),
		KeyProvider: func() (string, bool) {
			return os.LookupEnv(KeyEnvVar)
		},
		EnvLookup:  os.LookupEnv,
		LoadPolicy: LoadPolicyStrict,
		deps:       defaultStoreDeps(),
	}
}

// EncryptedPath returns the canonical encrypted project secrets path.
func (s *Store) EncryptedPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(ProjectSecretsPath))
}

// LegacyPath returns the canonical legacy plaintext secrets path.
func (s *Store) LegacyPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(LegacySecretsPath))
}

// LockPath returns the advisory lock used to serialize mutating operations.
func (s *Store) LockPath() string {
	return filepath.Join(filepath.Dir(s.EncryptedPath()), lockFileName)
}

// Load reads secrets using the store's configured LoadPolicy.
func (s *Store) Load() (map[string]string, error) {
	return s.LoadWithPolicy(s.LoadPolicy)
}

// LoadWithPolicy reads secrets using the provided policy.
//
// Strict loads use the encrypted file whenever it exists. Best-effort loads
// fall back to the legacy plaintext file only when the encrypted file cannot be
// consumed.
func (s *Store) LoadWithPolicy(policy LoadPolicy) (map[string]string, error) {
	if data, err := s.loadEncryptedUnlocked(); err == nil {
		return data, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return s.loadLegacyUnlocked()
	} else if policy != LoadPolicyBestEffortLegacy {
		return nil, err
	}
	return s.loadLegacyUnlocked()
}

// LoadEncrypted reads only the encrypted file and never falls back.
func (s *Store) LoadEncrypted() (map[string]string, error) {
	return s.loadEncryptedUnlocked()
}

func (s *Store) loadEncryptedUnlocked() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.EncryptedPath()
	if err := s.validateSecretFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}
	data, err := deps.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, &Error{Kind: ErrEncryptedRead, Op: "read encrypted secrets", Path: path, Err: err}
	}

	var payload encryptedFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, &Error{Kind: ErrEncryptedInvalid, Op: "parse encrypted secrets", Path: path, Err: err}
	}
	if payload.Version != encryptionVersion {
		return nil, fmt.Errorf("%w: encrypted secrets %s version %d", ErrUnsupportedVersion, path, payload.Version)
	}
	if payload.Algorithm != encryptionAlgorithm {
		return nil, fmt.Errorf("%w: encrypted secrets %s algorithm %q", ErrUnsupportedAlgorithm, path, payload.Algorithm)
	}

	key, err := s.encryptionKey()
	if err != nil {
		return nil, err
	}
	plaintext, err := decryptPayload(payload, key)
	if err != nil {
		return nil, err
	}
	result, err := parseSecretMap(plaintext)
	if err != nil {
		return nil, &Error{Kind: ErrInvalidSecretData, Op: "parse decrypted secrets", Path: path, Err: err}
	}
	return result, nil
}

// LoadLegacy reads only the legacy plaintext file.
func (s *Store) LoadLegacy() (map[string]string, error) {
	return s.loadLegacyUnlocked()
}

func (s *Store) loadLegacyUnlocked() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.LegacyPath()
	if err := s.validateSecretFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	data, err := deps.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, &Error{Kind: ErrLegacyRead, Op: "read legacy secrets", Path: path, Err: err}
	}
	result, err := parseSecretMap(data)
	if err != nil {
		return nil, &Error{Kind: ErrLegacyInvalid, Op: "parse legacy secrets", Path: path, Err: err}
	}
	return result, nil
}

// Save replaces the encrypted secrets file with the provided values.
//
// The write is serialized through the store lock and committed via atomic
// rename so readers never observe a partially written encrypted file.
func (s *Store) Save(values map[string]string) error {
	return s.withWriteLock(func() error {
		return s.saveUnlocked(values)
	})
}

func (s *Store) saveUnlocked(values map[string]string) error {
	deps := s.storeDeps()
	normalized, err := normalizeSecretValues(values)
	if err != nil {
		return err
	}
	key, err := s.encryptionKey()
	if err != nil {
		return err
	}
	payload, err := encryptValuesWithReader(normalized, key, deps.randReader)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "marshal encrypted secrets", Path: s.EncryptedPath(), Err: err}
	}
	path := s.EncryptedPath()
	if err := deps.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "mkdir secrets dir", Path: filepath.Dir(path), Err: err}
	}
	if err := s.writeFileAtomically(path, append(data, '\n')); err != nil {
		return err
	}
	return nil
}

// SaveKey merges a single key/value pair into encrypted storage.
func (s *Store) SaveKey(name, value string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return &Error{Kind: ErrInvalidInput, Op: "validate secret key", Err: errors.New("secret name is required")}
	}
	if strings.HasPrefix(name, "_") {
		return &Error{Kind: ErrInvalidInput, Op: "validate secret key", Err: fmt.Errorf("secret name %q is reserved for metadata", name)}
	}
	if value == "" {
		return &Error{Kind: ErrInvalidInput, Op: "validate secret value", Err: fmt.Errorf("secret value is required for %q", name)}
	}
	return s.withWriteLock(func() error {
		values, err := s.LoadWithPolicy(s.LoadPolicy)
		if err != nil {
			return err
		}
		values[name] = value
		return s.saveUnlocked(values)
	})
}

// MigrateLegacy copies legacy plaintext secrets into encrypted storage.
//
// When removeSource is true, the legacy plaintext file is removed after a
// successful encrypted write or when the legacy file already matches the
// authoritative encrypted file. If both files exist and differ, migration fails
// with ErrMigrationConflict rather than overwriting encrypted state.
func (s *Store) MigrateLegacy(removeSource bool) (bool, error) {
	deps := s.storeDeps()
	var migrated bool
	err := s.withWriteLock(func() error {
		encryptedValues, encryptedErr := s.loadEncryptedUnlocked()
		switch {
		case encryptedErr == nil:
			legacyValues, legacyErr := s.loadLegacyUnlocked()
			if legacyErr != nil {
				return legacyErr
			}
			if len(legacyValues) == 0 {
				return nil
			}
			if !secretMapsEqual(encryptedValues, legacyValues) {
				return &Error{
					Kind: ErrMigrationConflict,
					Op:   "migrate legacy secrets",
					Path: s.LegacyPath(),
					Err:  fmt.Errorf("encrypted secrets at %s already differ", s.EncryptedPath()),
				}
			}
			if removeSource {
				if err := deps.removeFile(s.LegacyPath()); err != nil && !os.IsNotExist(err) {
					return &Error{Kind: ErrEncryptedWrite, Op: "remove legacy secrets", Path: s.LegacyPath(), Err: err}
				}
				migrated = true
			}
			return nil
		case !errors.Is(encryptedErr, os.ErrNotExist):
			return encryptedErr
		}

		values, err := s.loadLegacyUnlocked()
		if err != nil {
			return err
		}
		if len(values) == 0 {
			return nil
		}
		if err := s.saveUnlocked(values); err != nil {
			return err
		}
		migrated = true
		if removeSource {
			if err := deps.removeFile(s.LegacyPath()); err != nil && !os.IsNotExist(err) {
				return &Error{Kind: ErrEncryptedWrite, Op: "remove legacy secrets", Path: s.LegacyPath(), Err: err}
			}
		}
		return nil
	})
	return migrated, err
}

// Resolve returns the named secret from stored values first, then from the
// configured EnvLookup fallback. Project-level resolution remains store-first
// so on-disk operator state is authoritative once persisted.
func (s *Store) Resolve(name string) (string, bool, error) {
	values, err := s.LoadWithPolicy(s.LoadPolicy)
	if err != nil {
		return "", false, err
	}
	if value, ok := values[name]; ok && strings.TrimSpace(value) != "" {
		return value, true, nil
	}
	if s.EnvLookup == nil {
		return "", false, nil
	}
	if value, ok := s.EnvLookup(name); ok && strings.TrimSpace(value) != "" {
		return value, true, nil
	}
	return "", false, nil
}

func (s *Store) encryptionKey() ([]byte, error) {
	if s.KeyProvider == nil {
		return nil, ErrMissingKey
	}
	raw, ok := s.KeyProvider()
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, ErrMissingKey
	}
	return deriveKey(raw)
}

func deriveKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrMissingKey
	}
	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	sum := sha256.Sum256([]byte(trimmed))
	return sum[:], nil
}

func encryptValues(values map[string]string, key []byte) (encryptedFile, error) {
	return encryptValuesWithReader(values, key, rand.Reader)
}

func encryptValuesWithReader(values map[string]string, key []byte, random io.Reader) (encryptedFile, error) {
	plaintext, err := json.Marshal(values)
	if err != nil {
		return encryptedFile{}, fmt.Errorf("marshal secrets: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return encryptedFile{}, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedFile{}, fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(random, nonce); err != nil {
		return encryptedFile{}, fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := aead.Seal(nil, nonce, plaintext, nil)
	return encryptedFile{
		Version:    encryptionVersion,
		Algorithm:  encryptionAlgorithm,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}, nil
}

func decryptPayload(payload encryptedFile, key []byte) ([]byte, error) {
	nonce, err := base64.StdEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, &Error{Kind: ErrEncryptedInvalid, Op: "decode nonce", Err: err}
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, &Error{Kind: ErrEncryptedInvalid, Op: "decode ciphertext", Err: err}
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, &Error{Kind: ErrDecryptFailed, Op: "decrypt secrets", Err: err}
	}
	return plaintext, nil
}

func parseSecretMap(data []byte) (map[string]string, error) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(payload))
	for key, value := range payload {
		if strings.HasPrefix(key, "_") {
			continue
		}
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			return nil, fmt.Errorf("secret %q must be a JSON string", key)
		}
		result[key] = parsed
	}
	return result, nil
}

func defaultStoreDeps() storeDeps {
	return storeDeps{
		readFile:   os.ReadFile,
		removeFile: os.Remove,
		mkdirAll:   os.MkdirAll,
		createTemp: os.CreateTemp,
		rename:     os.Rename,
		open:       os.Open,
		openFile:   os.OpenFile,
		lstat:      os.Lstat,
		randReader: rand.Reader,
		now:        time.Now,
		sleep:      time.Sleep,
	}
}

func (s *Store) storeDeps() storeDeps {
	deps := s.deps
	defaults := defaultStoreDeps()
	if deps.readFile == nil {
		deps.readFile = defaults.readFile
	}
	if deps.removeFile == nil {
		deps.removeFile = defaults.removeFile
	}
	if deps.mkdirAll == nil {
		deps.mkdirAll = defaults.mkdirAll
	}
	if deps.createTemp == nil {
		deps.createTemp = defaults.createTemp
	}
	if deps.rename == nil {
		deps.rename = defaults.rename
	}
	if deps.open == nil {
		deps.open = defaults.open
	}
	if deps.openFile == nil {
		deps.openFile = defaults.openFile
	}
	if deps.lstat == nil {
		deps.lstat = defaults.lstat
	}
	if deps.randReader == nil {
		deps.randReader = defaults.randReader
	}
	if deps.now == nil {
		deps.now = defaults.now
	}
	if deps.sleep == nil {
		deps.sleep = defaults.sleep
	}
	return deps
}

func (s *Store) validateSecretFile(path string) error {
	deps := s.storeDeps()
	info, err := deps.lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.ErrNotExist
		}
		return &Error{Kind: ErrInvalidSecretData, Op: "stat secret file", Path: path, Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &Error{Kind: ErrSymlinkPath, Op: "validate secret file", Path: path, Err: errors.New("symlink paths are not allowed")}
	}
	if !info.Mode().IsRegular() {
		return &Error{Kind: ErrInvalidSecretData, Op: "validate secret file", Path: path, Err: fmt.Errorf("expected regular file, got mode %s", info.Mode())}
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return &Error{Kind: ErrInsecurePermissions, Op: "validate secret file permissions", Path: path, Err: fmt.Errorf("mode %o is too broad", info.Mode().Perm())}
	}
	return nil
}

type Error struct {
	Kind error
	Op   string
	Path string
	Err  error
}

func (e *Error) Error() string {
	parts := []string{}
	if strings.TrimSpace(e.Op) != "" {
		parts = append(parts, e.Op)
	}
	if strings.TrimSpace(e.Path) != "" {
		parts = append(parts, e.Path)
	}
	prefix := strings.Join(parts, " ")
	if prefix == "" {
		return e.Err.Error()
	}
	return prefix + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func (e *Error) Is(target error) bool {
	return target == e.Kind || errors.Is(e.Err, target)
}

func (s *Store) writeFileAtomically(path string, data []byte) error {
	deps := s.storeDeps()
	dir := filepath.Dir(path)
	tmp, err := deps.createTemp(dir, ".secrets-*.tmp")
	if err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "create temporary secrets file", Path: dir, Err: err}
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = deps.removeFile(tmpPath)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "chmod temporary secrets file", Path: tmpPath, Err: err}
	}
	if _, err := tmp.Write(data); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "write encrypted secrets", Path: tmpPath, Err: err}
	}
	if err := tmp.Sync(); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "sync encrypted secrets", Path: tmpPath, Err: err}
	}
	if err := tmp.Close(); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "close temporary secrets file", Path: tmpPath, Err: err}
	}
	if err := deps.rename(tmpPath, path); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "rename encrypted secrets", Path: path, Err: err}
	}
	if err := s.syncDir(dir); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (s *Store) withWriteLock(fn func() error) error {
	release, err := s.acquireWriteLock()
	if err != nil {
		return err
	}
	defer release()
	return fn()
}

func (s *Store) acquireWriteLock() (func(), error) {
	deps := s.storeDeps()
	lockPath := s.LockPath()
	if err := deps.mkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, &Error{Kind: ErrEncryptedWrite, Op: "mkdir secrets dir", Path: filepath.Dir(lockPath), Err: err}
	}
	deadline := deps.now().Add(lockTimeout)
	for {
		lockFile, err := deps.openFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = lockFile.WriteString(fmt.Sprintf("pid=%d\n", os.Getpid()))
			return func() {
				_ = lockFile.Close()
				_ = deps.removeFile(lockPath)
			}, nil
		}
		if !os.IsExist(err) {
			return nil, &Error{Kind: ErrEncryptedWrite, Op: "acquire secrets lock", Path: lockPath, Err: err}
		}
		recovered, recoverErr := s.recoverStaleLock(lockPath)
		if recoverErr != nil {
			return nil, recoverErr
		}
		if recovered {
			continue
		}
		if !deps.now().Before(deadline) {
			return nil, &Error{Kind: ErrLockTimeout, Op: "acquire secrets lock", Path: lockPath, Err: err}
		}
		deps.sleep(lockRetryInterval)
	}
}

func (s *Store) syncDir(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	deps := s.storeDeps()
	dir, err := deps.open(path)
	if err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "open secrets dir for sync", Path: path, Err: err}
	}
	defer dir.Close()
	if err := dir.Sync(); err != nil {
		return &Error{Kind: ErrEncryptedWrite, Op: "sync secrets dir", Path: path, Err: err}
	}
	return nil
}

func (s *Store) recoverStaleLock(lockPath string) (bool, error) {
	deps := s.storeDeps()
	info, err := deps.lstat(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Kind: ErrEncryptedWrite, Op: "stat secrets lock", Path: lockPath, Err: err}
	}
	if deps.now().Sub(info.ModTime()) < lockStaleAfter {
		return false, nil
	}
	data, err := deps.readFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Kind: ErrEncryptedWrite, Op: "read secrets lock", Path: lockPath, Err: err}
	}
	pid, ok := parseLockPID(data)
	if ok && pid > 0 && processAlive(pid) {
		return false, nil
	}
	if err := deps.removeFile(lockPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, &Error{Kind: ErrEncryptedWrite, Op: "remove stale secrets lock", Path: lockPath, Err: err}
	}
	return true, nil
}

func parseLockPID(data []byte) (int, bool) {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "pid=") {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "pid=")))
		if err != nil {
			return 0, false
		}
		return pid, true
	}
	return 0, false
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid)))
	return err == nil
}

func normalizeSecretValues(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return map[string]string{}, nil
	}
	normalized := make(map[string]string, len(values))
	for rawKey, value := range values {
		key := strings.TrimSpace(rawKey)
		if key == "" {
			return nil, &Error{Kind: ErrInvalidInput, Op: "validate secret key", Err: errors.New("secret key is required")}
		}
		if strings.HasPrefix(key, "_") {
			return nil, &Error{Kind: ErrInvalidInput, Op: "validate secret key", Err: fmt.Errorf("secret key %q is reserved for metadata", key)}
		}
		if existing, ok := normalized[key]; ok && existing != value {
			return nil, &Error{Kind: ErrInvalidInput, Op: "normalize secret values", Err: fmt.Errorf("multiple values provided for normalized key %q", key)}
		}
		normalized[key] = value
	}
	return normalized, nil
}

func secretMapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, leftValue := range left {
		if rightValue, ok := right[key]; !ok || rightValue != leftValue {
			return false
		}
	}
	return true
}
