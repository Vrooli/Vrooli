package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
)

const plaintextFilename = "secrets.json"

const (
	SourceEnv     = "env"
	SourceFile    = "file"
	SourceMissing = "missing"
)

type Config struct {
	RepoRoot  string
	EnvLookup func(string) string
}

type Store struct {
	repoRoot  string
	path      string
	envLookup func(string) string
}

type Resolution struct {
	Value      string
	Source     string
	SourcePath string
}

type Document struct {
	Secrets  map[string]string
	Metadata map[string]json.RawMessage
}

var (
	findRepoRootFromEnvOrCWD = repocontract.FindRepoRootFromEnvOrCWD
	findRepoRootFromPath     = repocontract.FindRepoRootFromPath
	loadDefaultContract      = repocontract.LoadDefault
	getwdFn                  = os.Getwd
	readFileFn               = os.ReadFile
	lstatFn                  = os.Lstat
)

func NewProjectStore(cfg Config) (*Store, error) {
	repoRoot, path, err := resolveProjectSecretsPath(cfg)
	if err != nil {
		return nil, err
	}
	return &Store{
		repoRoot:  repoRoot,
		path:      path,
		envLookup: normalizeEnvLookup(cfg.EnvLookup),
	}, nil
}

func NewProjectStoreFromEnvOrCWD(cfg Config) (*Store, error) {
	cfg.RepoRoot = ""
	return NewProjectStore(cfg)
}

func NewFileStore(path string) (*Store, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return nil, &Error{Kind: ErrInvalidInput, Message: "secrets path is required"}
	}
	return &Store{
		path:      path,
		envLookup: normalizeEnvLookup(nil),
	}, nil
}

func (s *Store) RepoRoot() string {
	return s.repoRoot
}

func (s *Store) PlaintextPath() string {
	return s.path
}

func (s *Store) Load() (map[string]string, error) {
	doc, err := LoadFileDocument(s.path)
	if err != nil {
		return nil, err
	}
	return cloneStringMap(doc.Secrets), nil
}

func (s *Store) Document() (Document, error) {
	return LoadFileDocument(s.path)
}

func (s *Store) Resolve(key string) (Resolution, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return Resolution{}, &Error{Kind: ErrInvalidInput, Message: "secret key is required"}
	}
	if value := strings.TrimSpace(s.envLookup(key)); value != "" {
		return Resolution{Value: value, Source: SourceEnv}, nil
	}
	doc, err := LoadFileDocument(s.path)
	if err != nil {
		return Resolution{}, err
	}
	if value, ok := doc.Secrets[key]; ok {
		value = strings.TrimSpace(value)
		if value != "" {
			return Resolution{Value: value, Source: SourceFile, SourcePath: s.path}, nil
		}
	}
	return Resolution{Source: SourceMissing}, nil
}

func (s *Store) ResolveValue(key string) string {
	resolved, err := s.Resolve(key)
	if err != nil {
		return ""
	}
	return resolved.Value
}

func (s *Store) Save(values map[string]string) error {
	doc := Document{
		Secrets:  cloneStringMap(values),
		Metadata: map[string]json.RawMessage{},
	}
	return WriteFileDocument(s.path, doc)
}

func (s *Store) SaveKey(key, value string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return &Error{Kind: ErrInvalidInput, Message: "secret key is required"}
	}
	if strings.HasPrefix(key, "_") {
		return &Error{Kind: ErrInvalidInput, Message: "secret keys must not start with underscore", Details: key}
	}
	if value == "" {
		return &Error{Kind: ErrInvalidInput, Message: "secret value is required", Details: key}
	}
	return s.Update(func(doc *Document) error {
		if doc.Secrets == nil {
			doc.Secrets = map[string]string{}
		}
		doc.Secrets[key] = value
		return nil
	})
}

func (s *Store) DeleteKey(key string) (bool, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return false, &Error{Kind: ErrInvalidInput, Message: "secret key is required"}
	}
	deleted := false
	err := s.Update(func(doc *Document) error {
		if _, ok := doc.Secrets[key]; !ok {
			return nil
		}
		delete(doc.Secrets, key)
		deleted = true
		return nil
	})
	if err != nil {
		return false, err
	}
	return deleted, nil
}

func (s *Store) Update(update func(doc *Document) error) error {
	if update == nil {
		return &Error{Kind: ErrInvalidInput, Message: "update callback is required"}
	}
	doc, err := LoadFileDocument(s.path)
	if err != nil {
		return err
	}
	if doc.Secrets == nil {
		doc.Secrets = map[string]string{}
	}
	if doc.Metadata == nil {
		doc.Metadata = map[string]json.RawMessage{}
	}
	if err := update(&doc); err != nil {
		return err
	}
	return WriteFileDocument(s.path, doc)
}

func LoadFile(path string) (map[string]string, error) {
	doc, err := LoadFileDocument(path)
	if err != nil {
		return nil, err
	}
	return cloneStringMap(doc.Secrets), nil
}

func LoadFileDocument(path string) (Document, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return Document{}, &Error{Kind: ErrInvalidInput, Message: "secrets path is required"}
	}
	if err := validateSecretFile(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Document{
				Secrets:  map[string]string{},
				Metadata: map[string]json.RawMessage{},
			}, nil
		}
		return Document{}, err
	}
	data, err := readFileFn(path)
	if err != nil {
		return Document{}, &Error{Kind: ErrIO, Message: "read secret file", Details: path, Err: err}
	}
	if len(data) == 0 {
		return Document{
			Secrets:  map[string]string{},
			Metadata: map[string]json.RawMessage{},
		}, nil
	}
	return parseDocument(path, data)
}

func WriteFileDocument(path string, doc Document) error {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" {
		return &Error{Kind: ErrInvalidInput, Message: "secrets path is required"}
	}
	payload := map[string]any{}
	keys := make([]string, 0, len(doc.Metadata))
	for key := range doc.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if !strings.HasPrefix(key, "_") {
			return &Error{Kind: ErrInvalidInput, Message: "metadata keys must start with underscore", Details: key}
		}
		payload[key] = doc.Metadata[key]
	}
	keys = keys[:0]
	for key := range doc.Secrets {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return &Error{Kind: ErrInvalidInput, Message: "secret key is required"}
		}
		if strings.HasPrefix(trimmedKey, "_") {
			return &Error{Kind: ErrInvalidInput, Message: "secret keys must not start with underscore", Details: trimmedKey}
		}
		payload[trimmedKey] = doc.Secrets[key]
	}
	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &Error{Kind: ErrInvalidData, Message: "marshal secret document", Details: path, Err: err}
	}
	return storage.WriteFileAtomic(path, append(encoded, '\n'), storage.SecretFilePerm)
}

func resolveProjectSecretsPath(cfg Config) (string, string, error) {
	if root := strings.TrimSpace(cfg.RepoRoot); root != "" {
		return resolveProjectSecretsPathFromRoot(root)
	}

	for _, key := range []string{"VROOLI_SOURCE_ROOT", "VROOLI_ROOT"} {
		if root := strings.TrimSpace(normalizeEnvLookup(cfg.EnvLookup)(key)); root != "" {
			return resolveProjectSecretsPathFromRoot(root)
		}
	}

	if root, err := findRepoRootFromEnvOrCWD(); err == nil {
		return resolveProjectSecretsPathFromRoot(root)
	}

	cwd, err := getwdFn()
	if err != nil {
		return "", "", &Error{Kind: ErrResolve, Message: "resolve current working directory", Err: err}
	}
	return discoverNearestSecretsPath(cwd)
}

func resolveProjectSecretsPathFromRoot(root string) (string, string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return "", "", &Error{Kind: ErrInvalidInput, Message: "repo root is required"}
	}
	if contractRoot, err := findRepoRootFromPath(root); err == nil {
		contract, loadErr := loadDefaultContract(contractRoot)
		if loadErr == nil {
			configDir, dirErr := contract.TopLevelDir(contractRoot, "project_config")
			if dirErr == nil {
				return contractRoot, filepath.Join(configDir, plaintextFilename), nil
			}
		}
		root = contractRoot
	}
	return root, filepath.Join(root, ".vrooli", plaintextFilename), nil
}

func discoverNearestSecretsPath(start string) (string, string, error) {
	current := filepath.Clean(start)
	info, err := lstatFn(current)
	if err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}

	for {
		configDir := filepath.Join(current, ".vrooli")
		if info, err := lstatFn(configDir); err == nil && info.IsDir() {
			return current, filepath.Join(configDir, plaintextFilename), nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "", "", &Error{Kind: ErrResolve, Message: "repo root not found from current working directory", Details: start}
}

func validateSecretFile(path string) error {
	info, err := lstatFn(path)
	if err != nil {
		if os.IsNotExist(err) {
			return os.ErrNotExist
		}
		return &Error{Kind: ErrIO, Message: "stat secret file", Details: path, Err: err}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return &Error{Kind: ErrSymlinkPath, Message: "refusing to read symlinked secret file", Details: path}
	}
	if !info.Mode().IsRegular() {
		return &Error{Kind: ErrInvalidData, Message: "secret path must be a regular file", Details: path}
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return &Error{Kind: ErrInsecurePermissions, Message: "secret file permissions are too broad", Details: fmt.Sprintf("%s mode=%o", path, info.Mode().Perm())}
	}
	return nil
}

func parseDocument(path string, data []byte) (Document, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Document{}, &Error{Kind: ErrInvalidData, Message: "decode secret JSON", Details: path, Err: err}
	}
	doc := Document{
		Secrets:  map[string]string{},
		Metadata: map[string]json.RawMessage{},
	}
	for key, value := range raw {
		if strings.HasPrefix(key, "_") {
			doc.Metadata[key] = append(json.RawMessage(nil), value...)
			continue
		}
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			return Document{}, &Error{Kind: ErrInvalidData, Message: "secret values must be JSON strings", Details: key, Err: err}
		}
		doc.Secrets[key] = parsed
	}
	return doc, nil
}

func normalizeEnvLookup(fn func(string) string) func(string) string {
	if fn != nil {
		return fn
	}
	return os.Getenv
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
