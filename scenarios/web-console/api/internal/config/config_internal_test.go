package config

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:P1-001a] Session Policy Controls - config defaults
func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.TerminalScrollbackLines != 10_000 {
		t.Errorf("TerminalScrollbackLines: want 10000, got %d", cfg.TerminalScrollbackLines)
	}
	if cfg.PTYReadBuffer != 4096 {
		t.Errorf("PTYReadBuffer: want 4096, got %d", cfg.PTYReadBuffer)
	}
	if cfg.WSBufferSize != 4096 {
		t.Errorf("WSBufferSize: want 4096, got %d", cfg.WSBufferSize)
	}
	if cfg.DefaultCols != 80 {
		t.Errorf("DefaultCols: want 80, got %d", cfg.DefaultCols)
	}
	if cfg.DefaultRows != 24 {
		t.Errorf("DefaultRows: want 24, got %d", cfg.DefaultRows)
	}
	if cfg.MaxSessions != 0 {
		t.Errorf("MaxSessions: want 0, got %d", cfg.MaxSessions)
	}
	if cfg.ClientChannelBuffer != 256 {
		t.Errorf("ClientChannelBuffer: want 256, got %d", cfg.ClientChannelBuffer)
	}
	if cfg.DefaultShell == "" {
		t.Error("DefaultShell should not be empty")
	}
}

// [REQ:P1-001a] Session Policy Controls - env override
func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("WC_TERMINAL_SCROLLBACK_LINES", "5000")
	t.Setenv("WC_DEFAULT_COLS", "120")
	t.Setenv("WC_MAX_SESSIONS", "10")

	cfg := Load()

	if cfg.TerminalScrollbackLines != 5000 {
		t.Errorf("TerminalScrollbackLines: want 5000, got %d", cfg.TerminalScrollbackLines)
	}
	if cfg.DefaultCols != 120 {
		t.Errorf("DefaultCols: want 120, got %d", cfg.DefaultCols)
	}
	if cfg.MaxSessions != 10 {
		t.Errorf("MaxSessions: want 10, got %d", cfg.MaxSessions)
	}
}

// [REQ:P1-001a] Session Policy Controls - value clamping
func TestLoad_Clamping(t *testing.T) {
	t.Setenv("WC_PTY_READ_BUFFER", "100")
	t.Setenv("WC_DEFAULT_ROWS", "999")
	t.Setenv("WC_MAX_SESSIONS", "-5")

	cfg := Load()

	if cfg.PTYReadBuffer != 512 {
		t.Errorf("PTYReadBuffer should clamp to 512, got %d", cfg.PTYReadBuffer)
	}
	if cfg.DefaultRows != 200 {
		t.Errorf("DefaultRows should clamp to 200, got %d", cfg.DefaultRows)
	}
	if cfg.MaxSessions != 0 {
		t.Errorf("MaxSessions should clamp to 0, got %d", cfg.MaxSessions)
	}
}

// [REQ:P1-001a] Session Policy Controls - invalid values use defaults
func TestLoad_InvalidFallback(t *testing.T) {
	t.Setenv("WC_TERMINAL_SCROLLBACK_LINES", "not_a_number")

	cfg := Load()

	if cfg.TerminalScrollbackLines != 10_000 {
		t.Errorf("TerminalScrollbackLines should fall back to default, got %d", cfg.TerminalScrollbackLines)
	}
}

func TestLoad_ShellOverride(t *testing.T) {
	t.Setenv("WC_DEFAULT_SHELL", "/usr/bin/fish")

	cfg := Load()

	if cfg.DefaultShell != "/usr/bin/fish" {
		t.Errorf("DefaultShell: want /usr/bin/fish, got %s", cfg.DefaultShell)
	}
}

func TestEnvInt_Unset(t *testing.T) {
	os.Unsetenv("WC_TEST_UNSET")
	val := envInt("WC_TEST_UNSET", 42, 0, 100)
	if val != 42 {
		t.Errorf("unset env should return default 42, got %d", val)
	}
}

// [REQ:P0-002a] resolveShell falls back to $SHELL when WC_DEFAULT_SHELL is unset
func TestResolveShell_FallbackToSHELL(t *testing.T) {
	t.Setenv("WC_DEFAULT_SHELL", "")
	t.Setenv("SHELL", "/usr/bin/zsh")

	shell := resolveShell()
	if shell != "/usr/bin/zsh" {
		t.Errorf("expected /usr/bin/zsh, got %s", shell)
	}
}

func TestResolveWorkingDir_PrefersProjectRoot(t *testing.T) {
	projectRoot := t.TempDir()
	scenarioDir := t.TempDir()
	explicit := t.TempDir()

	t.Setenv("WC_DEFAULT_CWD", explicit)
	t.Setenv("PROJECT_ROOT", projectRoot)
	t.Setenv("SCENARIO_DIR", scenarioDir)

	got := resolveWorkingDir()
	if got != explicit {
		t.Fatalf("expected WC_DEFAULT_CWD %q, got %q", explicit, got)
	}

	t.Setenv("WC_DEFAULT_CWD", "")
	got = resolveWorkingDir()
	if got != projectRoot {
		t.Fatalf("expected PROJECT_ROOT %q, got %q", projectRoot, got)
	}

	t.Setenv("PROJECT_ROOT", "")
	got = resolveWorkingDir()
	if got != scenarioDir {
		t.Fatalf("expected SCENARIO_DIR %q, got %q", scenarioDir, got)
	}
}

func TestInferScenarioDirFromWD_ApiSubdir(t *testing.T) {
	base := t.TempDir()
	apiDir := filepath.Join(base, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api dir: %v", err)
	}

	got := inferScenarioDirFromWD(apiDir)
	if got != base {
		t.Fatalf("expected %q, got %q", base, got)
	}
}

func TestResolveWorkingDir_FallsBackToCurrentWD(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	scenarioDir := t.TempDir()
	apiDir := filepath.Join(scenarioDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api dir: %v", err)
	}
	if err := os.Chdir(apiDir); err != nil {
		t.Fatalf("chdir api dir: %v", err)
	}

	t.Setenv("WC_DEFAULT_CWD", "")
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("SCENARIO_DIR", "")

	got := resolveWorkingDir()
	if got != scenarioDir {
		t.Fatalf("expected scenario dir %q, got %q", scenarioDir, got)
	}
}

func TestResolveWorkingDir_InfersProjectRootFromScenarioDir(t *testing.T) {
	projectRoot := t.TempDir()
	scenarioDir := filepath.Join(projectRoot, "scenarios", "web-console")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir scenario dir: %v", err)
	}

	t.Setenv("WC_DEFAULT_CWD", "")
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("SCENARIO_DIR", scenarioDir)

	got := resolveWorkingDir()
	if got != projectRoot {
		t.Fatalf("expected inferred project root %q, got %q", projectRoot, got)
	}
}

func TestResolveWorkingDir_InfersProjectRootFromWD(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	projectRoot := t.TempDir()
	apiDir := filepath.Join(projectRoot, "scenarios", "web-console", "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir api dir: %v", err)
	}
	if err := os.Chdir(apiDir); err != nil {
		t.Fatalf("chdir api dir: %v", err)
	}

	t.Setenv("WC_DEFAULT_CWD", "")
	t.Setenv("PROJECT_ROOT", "")
	t.Setenv("SCENARIO_DIR", "")

	got := resolveWorkingDir()
	if got != projectRoot {
		t.Fatalf("expected inferred project root %q, got %q", projectRoot, got)
	}
}
