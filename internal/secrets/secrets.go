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
)

const (
	KeyEnvVar           = "VROOLI_SECRETS_KEY"
	ProjectSecretsPath  = ".vrooli/secrets.enc.json"
	LegacySecretsPath   = ".vrooli/secrets.json"
	encryptionAlgorithm = "AES-256-GCM"
	encryptionVersion   = 1
)

var ErrMissingKey = errors.New("missing VROOLI_SECRETS_KEY for encrypted secrets")

type encryptedFile struct {
	Version    int    `json:"version"`
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type Store struct {
	Root      string
	KeySource func(string) (string, bool)
	deps      storeDeps
}

type storeDeps struct {
	readFile   func(string) ([]byte, error)
	writeFile  func(string, []byte, os.FileMode) error
	removeFile func(string) error
	mkdirAll   func(string, os.FileMode) error
	randReader io.Reader
}

func NewProjectStore(root string) *Store {
	return &Store{
		Root: filepath.Clean(root),
		KeySource: func(key string) (string, bool) {
			return os.LookupEnv(key)
		},
		deps: defaultStoreDeps(),
	}
}

func (s *Store) EncryptedPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(ProjectSecretsPath))
}

func (s *Store) LegacyPath() string {
	return filepath.Join(s.Root, filepath.FromSlash(LegacySecretsPath))
}

func (s *Store) Load() (map[string]string, error) {
	if data, err := s.LoadEncrypted(); err == nil {
		return data, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return s.LoadLegacy()
}

func (s *Store) LoadEncrypted() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.EncryptedPath()
	data, err := deps.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, os.ErrNotExist
		}
		return nil, fmt.Errorf("read encrypted secrets %s: %w", path, err)
	}

	var payload encryptedFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("parse encrypted secrets %s: %w", path, err)
	}
	if payload.Version != encryptionVersion {
		return nil, fmt.Errorf("unsupported secrets version %d", payload.Version)
	}
	if payload.Algorithm != encryptionAlgorithm {
		return nil, fmt.Errorf("unsupported secrets algorithm %q", payload.Algorithm)
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
		return nil, fmt.Errorf("parse decrypted secrets %s: %w", path, err)
	}
	return result, nil
}

func (s *Store) LoadLegacy() (map[string]string, error) {
	deps := s.storeDeps()
	path := s.LegacyPath()
	data, err := deps.readFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read legacy secrets %s: %w", path, err)
	}
	result, err := parseSecretMap(data)
	if err != nil {
		return nil, fmt.Errorf("parse legacy secrets %s: %w", path, err)
	}
	return result, nil
}

func (s *Store) Save(values map[string]string) error {
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
	if err := deps.writeFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write encrypted secrets %s: %w", path, err)
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
	values, err := s.Load()
	if err != nil {
		return err
	}
	values[name] = value
	return s.Save(values)
}

func (s *Store) MigrateLegacy(removeSource bool) (bool, error) {
	deps := s.storeDeps()
	values, err := s.LoadLegacy()
	if err != nil {
		return false, err
	}
	if len(values) == 0 {
		return false, nil
	}
	if err := s.Save(values); err != nil {
		return false, err
	}
	if removeSource {
		if err := deps.removeFile(s.LegacyPath()); err != nil && !os.IsNotExist(err) {
			return true, fmt.Errorf("remove legacy secrets %s: %w", s.LegacyPath(), err)
		}
	}
	return true, nil
}

func (s *Store) Resolve(name string) (string, bool, error) {
	values, err := s.Load()
	if err != nil {
		return "", false, err
	}
	if value, ok := values[name]; ok && strings.TrimSpace(value) != "" {
		return value, true, nil
	}
	if s.KeySource == nil {
		return "", false, nil
	}
	if value, ok := s.KeySource(name); ok && strings.TrimSpace(value) != "" {
		return value, true, nil
	}
	return "", false, nil
}

func (s *Store) encryptionKey() ([]byte, error) {
	if s.KeySource == nil {
		return nil, ErrMissingKey
	}
	raw, ok := s.KeySource(KeyEnvVar)
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
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
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
		return nil, fmt.Errorf("decrypt secrets: %w", err)
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
		writeFile:  os.WriteFile,
		removeFile: os.Remove,
		mkdirAll:   os.MkdirAll,
		randReader: rand.Reader,
	}
}

func (s *Store) storeDeps() storeDeps {
	deps := s.deps
	defaults := defaultStoreDeps()
	if deps.readFile == nil {
		deps.readFile = defaults.readFile
	}
	if deps.writeFile == nil {
		deps.writeFile = defaults.writeFile
	}
	if deps.removeFile == nil {
		deps.removeFile = defaults.removeFile
	}
	if deps.mkdirAll == nil {
		deps.mkdirAll = defaults.mkdirAll
	}
	if deps.randReader == nil {
		deps.randReader = defaults.randReader
	}
	return deps
}
