package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	defaultPrecommitTimeoutSeconds = 300
	maxPrecommitTimeoutSeconds     = 1800
	precommitOutputLimit           = 24000

	defaultPrecommitCommand = "vrooli hygiene --fail-on error"
)

type PrecommitService struct {
	db     *sql.DB
	runner CommandRunner
}

type CommandRunner interface {
	Run(ctx context.Context, req CommandRunRequest) (CommandRunResult, error)
}

type CommandRunRequest struct {
	Command          string
	WorkingDirectory string
}

type CommandRunResult struct {
	Stdout string
	Stderr string
}

type ShellCommandRunner struct{}

// shellCommandWaitDelay bounds how long Wait blocks on I/O copying after the
// process group is signaled. It is a backstop: the group kill (below) normally
// terminates every child, but a check that double-forks or calls setsid() can
// escape the group and keep an inherited pipe open. WaitDelay forces those
// pipes closed so Wait returns instead of hanging forever.
const shellCommandWaitDelay = 2 * time.Second

// newShellCommand builds the `bash -lc <command>` invocation used to run
// precommit checks, hardened so that cancelling ctx terminates the entire
// process tree — not just the bash parent.
//
// exec.CommandContext's default cancellation only SIGKILLs the direct child
// (bash). The linters/tests bash spawns are grandchildren; they would orphan
// and keep the stdout/stderr pipes open, blocking the reader in RunStream from
// ever seeing EOF. Placing the command in its own process group (Setpgid) and
// signalling the whole group on cancel kills those grandchildren too.
func newShellCommand(ctx context.Context, req CommandRunRequest) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "bash", "-lc", req.Command)
	cmd.Dir = req.WorkingDirectory
	// Setpgid makes bash the leader of a new process group whose pgid equals its
	// pid, so every descendant shares that group unless it deliberately escapes.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	// On ctx cancel, SIGKILL the whole group (negative pid == the group).
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return err
		}
		return os.ErrProcessDone
	}
	cmd.WaitDelay = shellCommandWaitDelay
	return cmd
}

func (ShellCommandRunner) Run(ctx context.Context, req CommandRunRequest) (CommandRunResult, error) {
	cmd := newShellCommand(ctx, req)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return CommandRunResult{Stdout: stdout.String(), Stderr: stderr.String()}, err
}

func NewPrecommitService(db *sql.DB) *PrecommitService {
	return NewPrecommitServiceWithRunner(db, ShellCommandRunner{})
}

func NewPrecommitServiceWithRunner(db *sql.DB, runner CommandRunner) *PrecommitService {
	if runner == nil {
		runner = ShellCommandRunner{}
	}
	return &PrecommitService{db: db, runner: runner}
}

func (s *PrecommitService) Get(ctx context.Context, repoDir string) (PrecommitConfig, error) {
	cfg := defaultPrecommitConfig(repoDir)
	if s == nil || s.db == nil {
		return cfg, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT enabled, command, working_directory, timeout_seconds, run_before_commit, allow_override,
			last_status, last_exit_code, last_summary, last_stdout, last_stderr, last_duration_ms, last_timestamp,
			hook_install_status, hook_install_reason, hook_existing_kind, hook_installed_at
		FROM git_repo_precommit
		WHERE repo_path = ?
	`, repoDir)
	var (
		enabled, runBeforeCommit, allowOverride                   int
		lastStatus, lastSummary                                   sql.NullString
		lastStdout, lastStderr                                    sql.NullString
		lastExitCode, lastDuration                                sql.NullInt64
		lastTimestamp                                             sql.NullString
		hookStatus, hookReason, hookExistingKind, hookInstalledAt sql.NullString
	)
	err := row.Scan(&enabled, &cfg.Command, &cfg.WorkingDirectory, &cfg.TimeoutSeconds, &runBeforeCommit, &allowOverride,
		&lastStatus, &lastExitCode, &lastSummary, &lastStdout, &lastStderr, &lastDuration, &lastTimestamp,
		&hookStatus, &hookReason, &hookExistingKind, &hookInstalledAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return cfg, nil
		}
		return PrecommitConfig{}, fmt.Errorf("load precommit config: %w", err)
	}
	cfg.Enabled = enabled != 0
	cfg.RunBeforeCommit = runBeforeCommit != 0
	cfg.AllowOverride = allowOverride != 0
	if lastStatus.Valid {
		result := PrecommitRunResult{
			Status:          lastStatus.String,
			Command:         cfg.Command,
			ExitCode:        int(lastExitCode.Int64),
			Summary:         lastSummary.String,
			Stdout:          lastStdout.String,
			Stderr:          lastStderr.String,
			DurationMs:      lastDuration.Int64,
			OverrideAllowed: cfg.AllowOverride,
		}
		if parsed, err := time.Parse(time.RFC3339Nano, lastTimestamp.String); err == nil {
			result.Timestamp = parsed
		}
		cfg.LastResult = &result
	}
	if hookStatus.Valid {
		hook := &PrecommitHookState{
			Status:       hookStatus.String,
			Reason:       hookReason.String,
			ExistingKind: hookExistingKind.String,
		}
		if hookInstalledAt.Valid {
			if parsed, err := time.Parse(time.RFC3339Nano, hookInstalledAt.String); err == nil {
				hook.InstalledAt = parsed
			}
		}
		cfg.Hook = hook
	}
	if info, err := ReadInstalledHook(ctx, repoDir); err == nil {
		if cfg.Hook == nil {
			cfg.Hook = &PrecommitHookState{}
		}
		cfg.Hook.Path = info.Path
		cfg.Hook.HooksPath = info.HooksPath
		if info.Kind != HookKindGCT && info.Kind != HookKindNone {
			cfg.Hook.ExistingHookPreview = info.Preview
			if cfg.Hook.ExistingKind == "" {
				cfg.Hook.ExistingKind = info.Kind
			}
		}
	}
	return normalizePrecommitConfig(repoDir, cfg), nil
}

func (s *PrecommitService) Save(ctx context.Context, repoDir string, cfg PrecommitConfig) (PrecommitConfig, error) {
	if s == nil || s.db == nil {
		return PrecommitConfig{}, fmt.Errorf("precommit store not configured")
	}
	cfg = normalizePrecommitConfig(repoDir, cfg)
	if cfg.Enabled && strings.TrimSpace(cfg.Command) == "" {
		return PrecommitConfig{}, fmt.Errorf("command is required when precommit is enabled")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_repo_precommit (repo_path, enabled, command, working_directory, timeout_seconds, run_before_commit, allow_override, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_path) DO UPDATE SET
			enabled = excluded.enabled,
			command = excluded.command,
			working_directory = excluded.working_directory,
			timeout_seconds = excluded.timeout_seconds,
			run_before_commit = excluded.run_before_commit,
			allow_override = excluded.allow_override,
			updated_at = excluded.updated_at
	`, repoDir, boolInt(cfg.Enabled), cfg.Command, cfg.WorkingDirectory, cfg.TimeoutSeconds, boolInt(cfg.RunBeforeCommit), boolInt(cfg.AllowOverride), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return PrecommitConfig{}, fmt.Errorf("save precommit config: %w", err)
	}
	hookResult := s.reconcileHook(ctx, repoDir, cfg)
	_ = s.persistHookState(ctx, repoDir, hookResult)
	return s.Get(ctx, repoDir)
}

func (s *PrecommitService) reconcileHook(ctx context.Context, repoDir string, cfg PrecommitConfig) HookInstallResult {
	if cfg.Enabled {
		result, err := InstallHook(ctx, repoDir, cfg.Command)
		if err != nil && result.Reason == "" {
			result.Reason = err.Error()
		}
		return result
	}
	result, err := UninstallHook(ctx, repoDir)
	if err != nil && result.Reason == "" {
		result.Reason = err.Error()
	}
	return result
}

func (s *PrecommitService) persistHookState(ctx context.Context, repoDir string, result HookInstallResult) error {
	if s == nil || s.db == nil {
		return nil
	}
	status := "fallback"
	if result.Installed {
		status = "installed"
	} else if strings.HasPrefix(result.Reason, "removed") || strings.HasPrefix(result.Reason, "no hook installed") {
		status = "uninstalled"
	}
	installedAt := sql.NullString{}
	if !result.InstalledAt.IsZero() {
		installedAt.Valid = true
		installedAt.String = result.InstalledAt.UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE git_repo_precommit
		SET hook_install_status = ?,
			hook_install_reason = ?,
			hook_existing_kind = ?,
			hook_installed_at = ?,
			updated_at = ?
		WHERE repo_path = ?
	`, status, result.Reason, result.ExistingHookKind, installedAt, time.Now().UTC().Format(time.RFC3339Nano), repoDir)
	return err
}

func (s *PrecommitService) Run(ctx context.Context, repoDir string, req PrecommitRunRequest) (PrecommitRunResult, error) {
	cfg, err := s.Get(ctx, repoDir)
	if err != nil {
		return PrecommitRunResult{}, err
	}
	if strings.TrimSpace(req.Command) != "" {
		cfg.Command = req.Command
	}
	if strings.TrimSpace(req.WorkingDirectory) != "" {
		cfg.WorkingDirectory = req.WorkingDirectory
	}
	if req.TimeoutSeconds > 0 {
		cfg.TimeoutSeconds = req.TimeoutSeconds
	}
	return s.runConfig(ctx, repoDir, cfg)
}

func (s *PrecommitService) RunBeforeCommit(ctx context.Context, repoDir string) (PrecommitRunResult, bool, error) {
	cfg, err := s.Get(ctx, repoDir)
	if err != nil {
		return PrecommitRunResult{}, false, err
	}
	if !cfg.Enabled || !cfg.RunBeforeCommit {
		return PrecommitRunResult{}, false, nil
	}
	result, err := s.runConfig(ctx, repoDir, cfg)
	return result, true, err
}

func (s *PrecommitService) runConfig(ctx context.Context, repoDir string, cfg PrecommitConfig) (PrecommitRunResult, error) {
	cfg = normalizePrecommitConfig(repoDir, cfg)
	if strings.TrimSpace(cfg.Command) == "" {
		return PrecommitRunResult{}, fmt.Errorf("precommit command is required")
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	started := time.Now()
	runner := s.runner
	if runner == nil {
		runner = ShellCommandRunner{}
	}
	runResult, err := runner.Run(runCtx, CommandRunRequest{
		Command:          cfg.Command,
		WorkingDirectory: cfg.WorkingDirectory,
	})
	result := PrecommitRunResult{
		Status:          "passed",
		Command:         cfg.Command,
		ExitCode:        0,
		Summary:         "Precommit checks passed",
		Stdout:          capOutput(runResult.Stdout),
		Stderr:          capOutput(runResult.Stderr),
		DurationMs:      time.Since(started).Milliseconds(),
		OverrideAllowed: cfg.AllowOverride,
		Timestamp:       time.Now().UTC(),
	}
	if err != nil {
		result.Status = "failed"
		result.Summary = "Precommit checks failed"
		result.ExitCode = 1
		if exitErr, ok := err.(interface{ ExitCode() int }); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		if runCtx.Err() == context.DeadlineExceeded {
			result.Status = "timeout"
			result.Summary = "Precommit checks timed out"
			result.ExitCode = 124
		}
	}
	_ = s.saveLastResult(context.Background(), repoDir, result)
	return result, nil
}

func (s *PrecommitService) saveLastResult(ctx context.Context, repoDir string, result PrecommitRunResult) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_repo_precommit (repo_path, timeout_seconds, run_before_commit, allow_override, last_status, last_exit_code, last_summary, last_stdout, last_stderr, last_duration_ms, last_timestamp, updated_at)
		VALUES (?, ?, 1, 1, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(repo_path) DO UPDATE SET
			last_status = excluded.last_status,
			last_exit_code = excluded.last_exit_code,
			last_summary = excluded.last_summary,
			last_stdout = excluded.last_stdout,
			last_stderr = excluded.last_stderr,
			last_duration_ms = excluded.last_duration_ms,
			last_timestamp = excluded.last_timestamp,
			updated_at = excluded.updated_at
	`, repoDir, defaultPrecommitTimeoutSeconds, result.Status, result.ExitCode, result.Summary, result.Stdout, result.Stderr, result.DurationMs, result.Timestamp.Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func defaultPrecommitConfig(repoDir string) PrecommitConfig {
	return PrecommitConfig{
		Enabled:          true,
		Command:          defaultPrecommitCommand,
		WorkingDirectory: repoDir,
		TimeoutSeconds:   defaultPrecommitTimeoutSeconds,
		RunBeforeCommit:  true,
		AllowOverride:    true,
	}
}

func normalizePrecommitConfig(repoDir string, cfg PrecommitConfig) PrecommitConfig {
	if strings.TrimSpace(cfg.WorkingDirectory) == "" {
		cfg.WorkingDirectory = repoDir
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = defaultPrecommitTimeoutSeconds
	}
	if cfg.TimeoutSeconds > maxPrecommitTimeoutSeconds {
		cfg.TimeoutSeconds = maxPrecommitTimeoutSeconds
	}
	return cfg
}

func capOutput(value string) string {
	if len(value) <= precommitOutputLimit {
		return value
	}
	return value[:precommitOutputLimit] + "\n[output truncated]"
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
