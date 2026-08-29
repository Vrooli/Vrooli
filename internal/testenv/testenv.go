package testenv

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/tuning"
)

// CredentialStore is the shared in-memory secure-store fixture. Error fields
// let a test inject the backend failures that cannot be produced by a healthy
// in-memory implementation.
type CredentialStore struct {
	mu          sync.Mutex
	values      map[string]string
	notFound    error
	PutError    error
	GetError    error
	DeleteError error
	Adapter     string
}

// NewCredentialStore returns an empty credential store. Pass the owning
// package's not-found sentinel when callers distinguish missing values with
// errors.Is; keeping the sentinel injected avoids coupling testenv to a
// credential backend package.
func NewCredentialStore(notFound ...error) *CredentialStore {
	err := errors.New("credential not found")
	if len(notFound) > 0 && notFound[0] != nil {
		err = notFound[0]
	}
	return &CredentialStore{values: map[string]string{}, notFound: err, Adapter: "memory"}
}

func (s *CredentialStore) Put(service, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.PutError != nil {
		return s.PutError
	}
	s.values[service+"/"+key] = value
	return nil
}

func (s *CredentialStore) Get(service, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.GetError != nil {
		return "", s.GetError
	}
	value, ok := s.values[service+"/"+key]
	if !ok {
		return "", s.notFound
	}
	return value, nil
}

func (s *CredentialStore) Delete(service, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.DeleteError != nil {
		return s.DeleteError
	}
	delete(s.values, service+"/"+key)
	return nil
}

func (s *CredentialStore) AdapterName() string { return s.Adapter }

// Clock is the shared concurrency-safe clock fixture.
type Clock struct {
	mu  sync.Mutex
	now time.Time
}

func NewClock(start time.Time) *Clock { return &Clock{now: start.UTC()} }

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Advance(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(duration)
}

// DecodeJSON decodes a JSON fixture and reports malformed output at the test
// call site.
func DecodeJSON[T any](t testing.TB, raw []byte) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, string(raw))
	}
	return value
}

// NewSQLiteStore owns the repeated temporary-path, open, failure and cleanup
// mechanics while the domain supplies its typed store constructor and schema.
func NewSQLiteStore[T interface{ Close() error }](t testing.TB, filename string, open func(string) (T, error)) T {
	t.Helper()
	store, err := open(filepath.Join(t.TempDir(), filename))
	if err != nil {
		t.Fatalf("open SQLite test store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

// StampSQLiteUserVersion creates or opens a SQLite fixture and stamps the
// compatibility version expected by schema-rejection tests.
func StampSQLiteUserVersion(t testing.TB, dbPath string, version int) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, version)); err != nil {
		t.Fatalf("stamp user_version: %v", err)
	}
}

var identityVariables = map[string]struct{}{
	"HOME": {}, "USER": {}, "SUDO_UID": {}, "SUDO_GID": {}, "SUDO_USER": {},
	"XDG_CACHE_HOME": {}, "XDG_DATA_HOME": {}, "XDG_RUNTIME_DIR": {},
}

// SetIdentityEnv sets exact identity values for resolver tests and unusual
// process identities that do not fit the common RuntimeHome/AsSudoUser cases.
func SetIdentityEnv(t *testing.T, values map[string]string) {
	t.Helper()
	keys := make([]string, 0, len(values))
	for key := range values {
		if _, ok := identityVariables[key]; !ok {
			t.Fatalf("%s is not an identity variable", key)
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		t.Setenv(key, values[key])
	}
}

// AsCurrentUser installs a non-elevated identity and clears stale sudo facts
// inherited from the process running the test.
func AsCurrentUser(t *testing.T, user string) {
	t.Helper()
	SetIdentityEnv(t, map[string]string{
		"USER": user, "SUDO_USER": "", "SUDO_UID": "", "SUDO_GID": "",
	})
}

// RepositoryTree is the small cross-domain repository skeleton shared by
// tests that need project and scenario paths on disk.
type RepositoryTree struct {
	Root                string
	Scenario            string
	ScenarioRoot        string
	ProjectServicePath  string
	ScenarioServicePath string
}

// RepositoryTreeOption customizes the files written by NewRepositoryTree.
type RepositoryTreeOption func(*repositoryTreeOptions)

type repositoryTreeOptions struct {
	projectService  []byte
	scenarioService []byte
}

// WithProjectServiceJSON replaces the default project service fixture.
func WithProjectServiceJSON(raw []byte) RepositoryTreeOption {
	return func(options *repositoryTreeOptions) {
		options.projectService = append([]byte(nil), raw...)
	}
}

// WithScenarioServiceJSON replaces the default scenario service fixture.
func WithScenarioServiceJSON(raw []byte) RepositoryTreeOption {
	return func(options *repositoryTreeOptions) {
		options.scenarioService = append([]byte(nil), raw...)
	}
}

// NewRepositoryTree creates an isolated project with one scenario skeleton.
// All well-known paths are resolved through repo-contract-go; the fixture does
// not embed a second copy of the repository layout contract.
func NewRepositoryTree(t *testing.T, scenario string, opts ...RepositoryTreeOption) RepositoryTree {
	t.Helper()
	if strings.TrimSpace(scenario) == "" {
		t.Fatal("scenario name is required")
	}
	options := repositoryTreeOptions{
		projectService:  []byte("{}\n"),
		scenarioService: []byte("{}\n"),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	root := t.TempDir()
	scenarioRoot := repocontract.ScenarioRoot(root, scenario)
	scenarioServicePath, err := repocontract.ScenarioServiceManifestPath(root, scenario)
	if err != nil {
		t.Fatalf("resolve scenario service path: %v", err)
	}
	serviceRel, err := repocontract.ScenarioWellKnownRel(root, repocontract.ScenarioPathService)
	if err != nil {
		t.Fatalf("resolve project service path: %v", err)
	}
	projectServicePath := filepath.Join(root, filepath.FromSlash(serviceRel))
	for _, directory := range []string{filepath.Dir(projectServicePath), scenarioRoot, filepath.Dir(scenarioServicePath)} {
		if err := os.MkdirAll(directory, tuning.PermDir); err != nil {
			t.Fatalf("create repository fixture directory %q: %v", directory, err)
		}
	}
	writeFixtureFile(t, projectServicePath, options.projectService)
	writeFixtureFile(t, scenarioServicePath, options.scenarioService)
	return RepositoryTree{
		Root:                root,
		Scenario:            scenario,
		ScenarioRoot:        scenarioRoot,
		ProjectServicePath:  projectServicePath,
		ScenarioServicePath: scenarioServicePath,
	}
}

func writeFixtureFile(t *testing.T, path string, contents []byte) {
	t.Helper()
	if err := os.WriteFile(path, contents, tuning.PermFile); err != nil {
		t.Fatalf("write repository fixture file %q: %v", path, err)
	}
}

// AssertFileContents verifies a fixture file without duplicating read and
// comparison scaffolding in each domain's fixture package.
func AssertFileContents(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture file %q: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("fixture file %q = %q, want %q", path, data, want)
	}
}

// SetSudoUser changes only the sudo username, for tests that intentionally
// exercise missing or non-default sudo IDs.
func SetSudoUser(t *testing.T, user string) {
	t.Helper()
	SetIdentityEnv(t, map[string]string{"SUDO_USER": user})
}

// AsSudoUser installs a complete, internally consistent sudo identity.
func AsSudoUser(t *testing.T, user string) {
	t.Helper()
	SetIdentityEnv(t, map[string]string{
		"SUDO_USER": user, "SUDO_UID": "1000", "SUDO_GID": "1000", "USER": "root",
	})
}

// SudoRuntimeHome combines the complete sudo identity with an isolated XDG
// runtime home for tests that exercise both environment dimensions.
func SudoRuntimeHome(t *testing.T, user string) string {
	t.Helper()
	home := RuntimeHome(t)
	AsSudoUser(t, user)
	return home
}

// RuntimeHome creates an isolated runtime home and configures the standard
// XDG locations beneath it.
func RuntimeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	SetIdentityEnv(t, map[string]string{
		"HOME":            home,
		"XDG_CACHE_HOME":  filepath.Join(home, ".cache"),
		"XDG_DATA_HOME":   filepath.Join(home, ".local", "share"),
		"XDG_RUNTIME_DIR": filepath.Join(home, ".runtime"),
	})
	for _, directory := range []string{
		os.Getenv("XDG_CACHE_HOME"),
		os.Getenv("XDG_DATA_HOME"),
		os.Getenv("XDG_RUNTIME_DIR"),
	} {
		if err := os.MkdirAll(directory, tuning.PermDir); err != nil {
			t.Fatalf("create runtime directory %q: %v", directory, err)
		}
	}
	return home
}
