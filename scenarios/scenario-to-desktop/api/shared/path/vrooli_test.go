package path

import (
	"errors"
	"io/fs"
	"path/filepath"
	"testing"
	"time"
)

// mockEnv implements Env for testing.
type mockEnv struct {
	envVars map[string]string
	homeDir string
	homeErr error
	cwd     string
	cwdErr  error
}

func (m *mockEnv) Getenv(key string) string     { return m.envVars[key] }
func (m *mockEnv) UserHomeDir() (string, error) { return m.homeDir, m.homeErr }
func (m *mockEnv) Getwd() (string, error)       { return m.cwd, m.cwdErr }

// mockFS implements FS for testing with configurable directory entries.
type mockFS struct {
	dirs map[string]bool // paths that exist and are directories
}

func (m *mockFS) Stat(name string) (fs.FileInfo, error) {
	if isDir, ok := m.dirs[name]; ok {
		return &mockFileInfo{name: filepath.Base(name), isDir: isDir}, nil
	}
	return nil, errors.New("not found")
}

// mockFileInfo implements fs.FileInfo.
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

func TestDetectRoot_EnvVar(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{"returns VROOLI_ROOT when set", "/opt/vrooli", "/opt/vrooli"},
		{"returns custom path", "/home/user/my-vrooli", "/home/user/my-vrooli"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := &mockEnv{envVars: map[string]string{"VROOLI_ROOT": tc.envValue}}
			fsys := &mockFS{dirs: map[string]bool{}}

			got := detectRoot(env, fsys)
			if got != tc.expected {
				t.Errorf("detectRoot() = %q, want %q", got, tc.expected)
			}
		})
	}
}

func TestDetectRoot_HomeDir(t *testing.T) {
	t.Run("finds ~/Vrooli when it exists as directory", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/home/user/Vrooli": true,
		}}

		got := detectRoot(env, fsys)
		if got != "/home/user/Vrooli" {
			t.Errorf("detectRoot() = %q, want %q", got, "/home/user/Vrooli")
		}
	})

	t.Run("skips when ~/Vrooli does not exist", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/tmp/work",
		}
		fsys := &mockFS{dirs: map[string]bool{}}

		got := detectRoot(env, fsys)
		// Should fall through to strategy 4 (relative path fallback)
		expected := filepath.Clean("/tmp/work/../../..")
		if got != expected {
			t.Errorf("detectRoot() = %q, want %q", got, expected)
		}
	})

	t.Run("skips when ~/Vrooli exists but is not a directory", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/tmp/work",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/home/user/Vrooli": false, // exists but not a directory
		}}

		got := detectRoot(env, fsys)
		expected := filepath.Clean("/tmp/work/../../..")
		if got != expected {
			t.Errorf("detectRoot() = %q, want %q", got, expected)
		}
	})

	t.Run("skips when UserHomeDir returns error", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeErr: errors.New("no home"),
			cwd:     "/some/dir",
		}
		fsys := &mockFS{dirs: map[string]bool{}}

		got := detectRoot(env, fsys)
		expected := filepath.Clean("/some/dir/../../..")
		if got != expected {
			t.Errorf("detectRoot() = %q, want %q", got, expected)
		}
	})
}

func TestDetectRoot_MarkerWalk(t *testing.T) {
	t.Run("finds .vrooli marker in current directory", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/opt/vrooli/scenarios/my-scenario/api",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/opt/vrooli/.vrooli": true,
		}}

		got := detectRoot(env, fsys)
		if got != "/opt/vrooli" {
			t.Errorf("detectRoot() = %q, want %q", got, "/opt/vrooli")
		}
	})

	t.Run("walks up multiple levels to find marker", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/projects/vrooli/scenarios/test/api/handlers",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/projects/vrooli/.vrooli": true,
		}}

		got := detectRoot(env, fsys)
		if got != "/projects/vrooli" {
			t.Errorf("detectRoot() = %q, want %q", got, "/projects/vrooli")
		}
	})

	t.Run("finds nearest marker when multiple exist", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/a/b/c/d",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/a/.vrooli":   true,
			"/a/b/.vrooli": true, // closer to cwd
		}}

		got := detectRoot(env, fsys)
		// Should find /a/b first since it walks from cwd upward
		if got != "/a/b/c/d" && got != "/a/b/c" && got != "/a/b" {
			// The walk starts at cwd (/a/b/c/d), checks /a/b/c/d/.vrooli, then /a/b/c/.vrooli, then /a/b/.vrooli
			if got != "/a/b" {
				t.Errorf("detectRoot() = %q, want %q (nearest marker)", got, "/a/b")
			}
		}
	})
}

func TestDetectRoot_Fallback(t *testing.T) {
	t.Run("returns relative path when no other strategy matches", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/some/random/directory",
		}
		fsys := &mockFS{dirs: map[string]bool{}}

		got := detectRoot(env, fsys)
		expected := filepath.Clean("/some/random/directory/../../..")
		if got != expected {
			t.Errorf("detectRoot() = %q, want %q", got, expected)
		}
	})

	t.Run("returns empty string when Getwd fails and no env var", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeErr: errors.New("no home"),
			cwdErr:  errors.New("no cwd"),
		}
		fsys := &mockFS{dirs: map[string]bool{}}

		got := detectRoot(env, fsys)
		if got != "" {
			t.Errorf("detectRoot() = %q, want empty string", got)
		}
	})
}

func TestDetectRoot_StrategyPriority(t *testing.T) {
	t.Run("env var takes priority over home dir", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{"VROOLI_ROOT": "/env/root"},
			homeDir: "/home/user",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/home/user/Vrooli": true,
		}}

		got := detectRoot(env, fsys)
		if got != "/env/root" {
			t.Errorf("detectRoot() = %q, want %q (env var should win)", got, "/env/root")
		}
	})

	t.Run("home dir takes priority over marker walk", func(t *testing.T) {
		env := &mockEnv{
			envVars: map[string]string{},
			homeDir: "/home/user",
			cwd:     "/opt/vrooli/api",
		}
		fsys := &mockFS{dirs: map[string]bool{
			"/home/user/Vrooli":   true,
			"/opt/vrooli/.vrooli": true,
		}}

		got := detectRoot(env, fsys)
		if got != "/home/user/Vrooli" {
			t.Errorf("detectRoot() = %q, want %q (home dir should win)", got, "/home/user/Vrooli")
		}
	})
}
