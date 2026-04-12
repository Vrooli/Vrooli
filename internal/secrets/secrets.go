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
)

var (
	ErrMissingKey           = errors.New("missing VROOLI_SECRETS_KEY for encrypted secrets")
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
)

type LookupFunc func(string) (string, bool)
type KeyProvider func() (string, bool)

type LoadPolicy int

const (
	LoadPolicyStrict LoadPolicy = iota
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
	openFile   func(string, int, os.FileMode) (*os.File, error)
	randReader io.Reader
	now        func() time.Time
	sleep      func(time.Duration)
}

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

func (s *Store) EncryptedPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(ProjectSecretsPath))
}

func (s *Store) LegacyPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(LegacySecretsPath))
}

func (s *Store) LockPath() string {
	return filepath.Join(filepath.Dir(s.EncryptedPath()), lockFileName)
}

func (s *Store) Load() (map[string]string, error) {
	return s.LoadWithPolicy(s.LoadPolicy)
}

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

func (s *Store) LoadEncrypted() (map[string]string, error) {
	return s.loadEncryptedUnlocked()
}

func (s *Store) loadEncryptedUnlocked() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.EncryptedPath()
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

func (s *Store) LoadLegacy() (map[string]string, error) {
	return s.loadLegacyUnlocked()
}

func (s *Store) loadLegacyUnlocked() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.LegacyPath()
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

func (s *Store) Save(values map[string]string) error {
	return s.withWriteLock(func() error {
		return s.saveUnlocked(values)
	})
}

func (s *Store) saveUnlocked(values map[string]string) error {
	deps := s.storeDeps()
	key, err := s.encryptionKey()
	if err != nil {
		return err
	}
	payload, err := encryptValuesWithReader(values, key, deps.randReader)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal encrypted secrets: %w", err)
	}
	path := s.EncryptedPath()
	if err := deps.mkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir secrets dir: %w", err)
	}
	if err := s.writeFileAtomically(path, append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Store) SaveKey(name, value string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("secret name is required")
	}
	if value == "" {
		return fmt.Errorf("secret value is required")
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

func (s *Store) MigrateLegacy(removeSource bool) (bool, error) {
	deps := s.storeDeps()
	var migrated bool
	err := s.withWriteLock(func() error {
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
				return fmt.Errorf("remove legacy secrets %s: %w", s.LegacyPath(), err)
			}
		}
		return nil
	})
	return migrated, err
}

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
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(payload))
	for key, value := range payload {
		switch typed := value.(type) {
		case string:
			result[key] = typed
		case bool:
			if typed {
				result[key] = "true"
			} else {
				result[key] = "false"
			}
		case float64:
			if typed == float64(int64(typed)) {
				result[key] = fmt.Sprintf("%d", int64(typed))
			} else {
				result[key] = fmt.Sprintf("%v", typed)
			}
		}
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
		openFile:   os.OpenFile,
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
	if deps.openFile == nil {
		deps.openFile = defaults.openFile
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
		return nil, fmt.Errorf("mkdir secrets dir: %w", err)
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
		if !deps.now().Before(deadline) {
			return nil, &Error{Kind: ErrLockTimeout, Op: "acquire secrets lock", Path: lockPath, Err: err}
		}
		deps.sleep(lockRetryInterval)
	}
}
