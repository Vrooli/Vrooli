package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	defaultPrecommitTimeoutSeconds = 300
	maxPrecommitTimeoutSeconds     = 1800
	precommitOutputLimit           = 24000
)

type PrecommitService struct {
	db *sql.DB
}

func NewPrecommitService(db *sql.DB) *PrecommitService {
	return &PrecommitService{db: db}
}

func (s *PrecommitService) Get(ctx context.Context, repoDir string) (PrecommitConfig, error) {
	cfg := defaultPrecommitConfig(repoDir)
	if s == nil || s.db == nil {
		return cfg, nil
	}
	row := s.db.QueryRowContext(ctx, `
		SELECT enabled, command, working_directory, timeout_seconds, run_before_commit, allow_override,
			last_status, last_exit_code, last_summary, last_stdout, last_stderr, last_duration_ms, last_timestamp
		FROM git_repo_precommit
		WHERE repo_path = ?
	`, repoDir)
	var (
		enabled, runBeforeCommit, allowOverride int
		lastStatus, lastSummary                 sql.NullString
		lastStdout, lastStderr                  sql.NullString
		lastExitCode, lastDuration              sql.NullInt64
		lastTimestamp                           sql.NullString
	)
	err := row.Scan(&enabled, &cfg.Command, &cfg.WorkingDirectory, &cfg.TimeoutSeconds, &runBeforeCommit, &allowOverride,
		&lastStatus, &lastExitCode, &lastSummary, &lastStdout, &lastStderr, &lastDuration, &lastTimestamp)
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
	return s.Get(ctx, repoDir)
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
	cmd := exec.CommandContext(runCtx, "bash", "-lc", cfg.Command)
	cmd.Dir = cfg.WorkingDirectory
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := PrecommitRunResult{
		Status:          "passed",
		ExitCode:        0,
		Summary:         "Precommit checks passed",
		Stdout:          capOutput(stdout.String()),
		Stderr:          capOutput(stderr.String()),
		DurationMs:      time.Since(started).Milliseconds(),
		OverrideAllowed: cfg.AllowOverride,
		Timestamp:       time.Now().UTC(),
	}
	if err != nil {
		result.Status = "failed"
		result.Summary = "Precommit checks failed"
		result.ExitCode = 1
		if exitErr, ok := err.(*exec.ExitError); ok {
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
