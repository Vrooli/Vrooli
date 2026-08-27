package testenv

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/tuning"
)

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
	sort.Strings(keys)
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
