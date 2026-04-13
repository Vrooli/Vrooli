package path

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

type mockEnv struct {
	envVars map[string]string
	cwd     string
	cwdErr  error
}

func (m *mockEnv) Getenv(key string) string { return m.envVars[key] }
func (m *mockEnv) Getwd() (string, error)   { return m.cwd, m.cwdErr }

type mockFS struct {
	dirs map[string]bool
}

func (m *mockFS) Stat(name string) (fs.FileInfo, error) {
	if isDir, ok := m.dirs[name]; ok {
		return &mockFileInfo{name: filepath.Base(name), isDir: isDir}, nil
	}
	return nil, errors.New("not found")
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() fs.FileMode  { return 0o755 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

func TestDetectRoot_EnvVarCanonicalizesDescendant(t *testing.T) {
	root := newPathContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	env := &mockEnv{envVars: map[string]string{"VROOLI_ROOT": nested}}
	fsys := &mockFS{dirs: map[string]bool{}}

	if got := detectRoot(env, fsys); got != root {
		t.Fatalf("detectRoot() = %q, want %q", got, root)
	}
}

func TestDetectRoot_MarkerWalk(t *testing.T) {
	env := &mockEnv{
		envVars: map[string]string{},
		cwd:     "/projects/vrooli/scenarios/test/api/handlers",
	}
	fsys := &mockFS{dirs: map[string]bool{
		"/projects/vrooli/.vrooli": true,
	}}

	got := detectRoot(env, fsys)
	if got != "/projects/vrooli" {
		t.Fatalf("detectRoot() = %q, want %q", got, "/projects/vrooli")
	}
}

func TestDetectRoot_Fallback(t *testing.T) {
	env := &mockEnv{
		envVars: map[string]string{},
		cwd:     "/some/random/directory",
	}
	fsys := &mockFS{dirs: map[string]bool{}}

	got := detectRoot(env, fsys)
	expected := filepath.Clean("/some/random/directory/../../..")
	if got != expected {
		t.Fatalf("detectRoot() = %q, want %q", got, expected)
	}
}

func TestDetectRoot_EmptyWhenCWDUnavailable(t *testing.T) {
	env := &mockEnv{
		envVars: map[string]string{},
		cwdErr:  errors.New("no cwd"),
	}
	fsys := &mockFS{dirs: map[string]bool{}}

	if got := detectRoot(env, fsys); got != "" {
		t.Fatalf("detectRoot() = %q, want empty string", got)
	}
}

func TestDetectScenariosRootUsesContract(t *testing.T) {
	root := newPathContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_ROOT", nested)

	got := DetectScenariosRoot()
	want := filepath.Join(root, "scenarios")
	if got != want {
		t.Fatalf("DetectScenariosRoot() = %q, want %q", got, want)
	}
}

func TestResolveScenarioRootUsesContract(t *testing.T) {
	root := newPathContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "scenario-to-desktop", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("VROOLI_ROOT", nested)

	got := ResolveScenarioRoot("alpha")
	want := filepath.Join(root, "scenarios", "alpha")
	if got != want {
		t.Fatalf("ResolveScenarioRoot() = %q, want %q", got, want)
	}
}

func newPathContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := pathRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/scenario-to-desktop-path-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func pathRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", ".."))
}
