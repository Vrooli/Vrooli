// Package toolexecution implements the Tool Execution Protocol for workspace-sandbox.
//
// This file provides adapter implementations that bridge the executor interfaces
// to the existing driver, process tracker, and sandbox service.
package toolexecution

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	driverexec "workspace-sandbox/internal/driver/exec"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"
)

// ProcessExecutorAdapter adapts the driver and process tracker to the ProcessExecutor interface.
type ProcessExecutorAdapter struct {
	sandboxService sandbox.ServiceAPI
	driver         driver.Driver
	processTracker *process.Tracker
	processLogger  *process.Logger
	profileStore   config.ProfileStore
	execConfig     config.ExecutionConfig
}

// ProcessExecutorConfig holds the configuration for ProcessExecutorAdapter.
type ProcessExecutorConfig struct {
	SandboxService sandbox.ServiceAPI
	Driver         driver.Driver
	ProcessTracker *process.Tracker
	ProcessLogger  *process.Logger
	ProfileStore   config.ProfileStore
	ExecConfig     config.ExecutionConfig
}

// NewProcessExecutorAdapter creates a new ProcessExecutorAdapter.
func NewProcessExecutorAdapter(cfg ProcessExecutorConfig) *ProcessExecutorAdapter {
	return &ProcessExecutorAdapter{
		sandboxService: cfg.SandboxService,
		driver:         cfg.Driver,
		processTracker: cfg.ProcessTracker,
		processLogger:  cfg.ProcessLogger,
		profileStore:   cfg.ProfileStore,
		execConfig:     cfg.ExecConfig,
	}
}

// ExecSync executes a command synchronously and returns the result.
func (a *ProcessExecutorAdapter) ExecSync(ctx context.Context, sandboxID uuid.UUID, req ExecRequest) (*ExecResult, error) {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		return nil, fmt.Errorf("sandbox must be active to execute commands")
	}

	// Build bwrap config: defaults → host env → isolation profile.
	cfg := driverexec.DefaultBwrapConfig()
	driverexec.CaptureEnv().ApplyTo(&cfg)
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	if err := a.applyIsolationProfile(&cfg, req.IsolationProfile); err != nil {
		return nil, err
	}

	// Set resource limits
	cfg.ResourceLimits = driverexec.ResourceLimits{
		TimeoutSec: req.TimeoutSec,
	}
	if cfg.ResourceLimits.TimeoutSec == 0 {
		cfg.ResourceLimits.TimeoutSec = 60
	}

	// Execute the command via the canonical exec path. The driver
	// declares its isolation mode via RequiresBwrap.
	result, err := driverexec.Exec(ctx, sb, a.driver.RequiresBwrap(), cfg, req.Command, req.Args...)
	if err != nil {
		return nil, err
	}

	// Track the process
	if a.processTracker != nil && result.PID > 0 {
		_, _ = a.processTracker.Track(sandboxID, result.PID, req.Command, "")
	}

	return &ExecResult{
		ExitCode: result.ExitCode,
		Stdout:   string(result.Stdout),
		Stderr:   string(result.Stderr),
		PID:      result.PID,
		TimedOut: result.ExitCode == 124 && result.Error != nil,
	}, nil
}

// StartAsync starts a background process and returns its info.
func (a *ProcessExecutorAdapter) StartAsync(ctx context.Context, sandboxID uuid.UUID, req ExecRequest) (*ProcessInfo, error) {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	// Verify sandbox is active
	if sb.Status != types.StatusActive {
		return nil, fmt.Errorf("sandbox must be active to start processes")
	}

	// Build bwrap config: defaults → host env → isolation profile.
	cfg := driverexec.DefaultBwrapConfig()
	driverexec.CaptureEnv().ApplyTo(&cfg)
	if req.WorkingDir != "" {
		cfg.WorkingDir = req.WorkingDir
	}
	for k, v := range req.Env {
		cfg.Env[k] = v
	}

	if err := a.applyIsolationProfile(&cfg, req.IsolationProfile); err != nil {
		return nil, err
	}

	// No timeout for background processes
	cfg.ResourceLimits.TimeoutSec = 0

	// Create pending log pair (stdout + stderr)
	var pendingPair *process.PendingLogPair
	if a.processLogger != nil {
		var logErr error
		pendingPair, logErr = a.processLogger.CreatePendingLogPair(sandboxID)
		if logErr == nil {
			cfg.StdoutWriter = pendingPair.Stdout
			cfg.StderrWriter = pendingPair.Stderr
		}
	}

	// onExit hook: record structured exit info on the tracker and
	// finalise the log pair.
	var onExitOnce sync.Once
	pidCh := make(chan int, 1)
	cfg.OnExit = func(exitCode, signal int, oomKilled bool) {
		var pid int
		select {
		case pid = <-pidCh:
		case <-time.After(2 * time.Second):
			return
		}
		select {
		case pidCh <- pid:
		default:
		}
		onExitOnce.Do(func() {
			info := process.ExitInfo{ExitCode: exitCode, Signal: signal, OOMKilled: oomKilled, StoppedAt: time.Now()}
			if a.processTracker != nil {
				a.processTracker.RecordExit(sandboxID, pid, info)
			}
			if a.processLogger != nil {
				_ = a.processLogger.CloseLogPair(sandboxID, pid, info)
			}
		})
	}

	// Start the background process via the canonical exec path.
	pid, err := driverexec.StartProcess(ctx, sb, a.driver.RequiresBwrap(), cfg, req.Command, req.Args...)
	if err != nil {
		if pendingPair != nil {
			_ = a.processLogger.AbortPair(pendingPair)
		}
		return nil, err
	}

	pidCh <- pid

	// Finalize log pair
	if pendingPair != nil {
		_, _, _ = a.processLogger.FinalizePair(pendingPair, pid)
	}

	// Track the process
	var trackedProc *process.TrackedProcess
	if a.processTracker != nil {
		trackedProc, _ = a.processTracker.Track(sandboxID, pid, req.Command, "")
	}

	info := &ProcessInfo{
		PID:     pid,
		Command: req.Command,
		Status:  "running",
	}
	if trackedProc != nil {
		info.StartedAt = trackedProc.StartedAt.Format(time.RFC3339)
	}

	return info, nil
}

// GetStatus returns the status of a process.
func (a *ProcessExecutorAdapter) GetStatus(ctx context.Context, sandboxID uuid.UUID, pid int) (*ProcessStatus, error) {
	if a.processTracker == nil {
		return nil, fmt.Errorf("process tracking not available")
	}

	procs := a.processTracker.GetProcesses(sandboxID)
	for _, p := range procs {
		if p.PID == pid {
			status := "running"
			if p.ExitCode != nil {
				status = "exited"
			}
			return &ProcessStatus{
				PID:      p.PID,
				Status:   status,
				ExitCode: p.ExitCode,
				Running:  p.ExitCode == nil,
			}, nil
		}
	}

	return nil, fmt.Errorf("process %d not found", pid)
}

// List returns all processes for a sandbox.
func (a *ProcessExecutorAdapter) List(ctx context.Context, sandboxID uuid.UUID) ([]*ProcessInfo, error) {
	if a.processTracker == nil {
		return nil, fmt.Errorf("process tracking not available")
	}

	procs := a.processTracker.GetProcesses(sandboxID)
	result := make([]*ProcessInfo, len(procs))
	for i, p := range procs {
		status := "running"
		if p.ExitCode != nil {
			status = "exited"
		}
		result[i] = &ProcessInfo{
			PID:       p.PID,
			Command:   p.Command,
			Status:    status,
			StartedAt: p.StartedAt.Format(time.RFC3339),
			ExitCode:  p.ExitCode,
		}
	}

	return result, nil
}

// Kill terminates a process.
func (a *ProcessExecutorAdapter) Kill(ctx context.Context, sandboxID uuid.UUID, pid int) error {
	if a.processTracker == nil {
		return fmt.Errorf("process tracking not available")
	}

	return a.processTracker.KillProcess(ctx, sandboxID, pid)
}

// GetLogs returns logs for a process. When stream is "stdout" or "stderr",
// only that stream is returned. When stream is "" or "both", both streams
// are read and surfaced separately on the result.
func (a *ProcessExecutorAdapter) GetLogs(ctx context.Context, sandboxID uuid.UUID, pid int, tailLines int, stream string) (*ProcessLogs, error) {
	if a.processLogger == nil {
		return nil, fmt.Errorf("process logging not available")
	}

	var wanted []process.Stream
	switch stream {
	case "", "both":
		wanted = []process.Stream{process.StreamStdout, process.StreamStderr}
	case "stdout":
		wanted = []process.Stream{process.StreamStdout}
	case "stderr":
		wanted = []process.Stream{process.StreamStderr}
	default:
		return nil, fmt.Errorf("invalid stream %q (want stdout, stderr, or both)", stream)
	}

	out := &ProcessLogs{PID: pid, Lines: tailLines}
	for _, s := range wanted {
		content, err := a.processLogger.ReadLog(sandboxID, pid, s, tailLines, 0)
		if err != nil {
			// A missing single-stream log shouldn't surface as an error
			// when the caller asked for both — return whatever's there.
			if len(wanted) == 1 {
				return nil, err
			}
			continue
		}
		switch s {
		case process.StreamStdout:
			out.Stdout = string(content)
		case process.StreamStderr:
			out.Stderr = string(content)
		}
	}
	return out, nil
}

// applyIsolationProfile resolves the requested profile ID against the
// store and composes it onto cfg. Returns IsolationProfileNotFoundError
// when the ID does not resolve so callers can surface a 400 to the
// requestor. There is no preset fallback; missing builtins are a
// configuration bug we want to surface rather than mask.
func (a *ProcessExecutorAdapter) applyIsolationProfile(cfg *driverexec.BwrapConfig, requestedID string) error {
	id := requestedID
	if id == "" {
		id = a.execConfig.DefaultIsolationProfile
	}
	if id == "" {
		id = "full"
	}
	if a.profileStore == nil {
		return types.NewIsolationProfileNotFoundError(id)
	}
	profile, err := a.profileStore.Get(id)
	if err != nil {
		return types.NewIsolationProfileNotFoundError(id)
	}
	return driverexec.ApplyIsolationProfile(cfg, convertProfileToDriver(profile))
}

// convertProfileToDriver converts a config.IsolationProfile to driverexec.IsolationProfile.
func convertProfileToDriver(p *config.IsolationProfile) *driverexec.IsolationProfile {
	if p == nil {
		return nil
	}
	return &driverexec.IsolationProfile{
		ID:             p.ID,
		Name:           p.Name,
		Description:    p.Description,
		Builtin:        p.Builtin,
		NetworkAccess:  p.NetworkAccess,
		ReadOnlyBinds:  p.ReadOnlyBinds,
		ReadWriteBinds: p.ReadWriteBinds,
		Environment:    p.Environment,
		Hostname:       p.Hostname,
	}
}

// FileOperatorAdapter adapts direct file operations to the FileOperator interface.
type FileOperatorAdapter struct {
	sandboxService sandbox.ServiceAPI
}

// NewFileOperatorAdapter creates a new FileOperatorAdapter.
func NewFileOperatorAdapter(svc sandbox.ServiceAPI) *FileOperatorAdapter {
	return &FileOperatorAdapter{sandboxService: svc}
}

// ListFiles lists files in a sandbox directory.
func (a *FileOperatorAdapter) ListFiles(ctx context.Context, sandboxID uuid.UUID, path string, recursive, includeHidden bool, pattern string) ([]*FileInfo, error) {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	if sb.Status != types.StatusActive {
		return nil, fmt.Errorf("sandbox must be active to list files")
	}

	// Resolve path
	if path == "" || path == "." {
		path = "/"
	}
	fullPath := filepath.Join(sb.MergedDir, path)

	// Validate path is within sandbox
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(sb.MergedDir)) {
		return nil, fmt.Errorf("path escapes sandbox")
	}

	var results []*FileInfo
	walkFn := func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip hidden files unless requested
		if !includeHidden && strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Apply pattern filter
		if pattern != "" {
			matched, _ := filepath.Match(pattern, info.Name())
			if !matched {
				return nil
			}
		}

		relPath, _ := filepath.Rel(sb.MergedDir, walkPath)
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}

		results = append(results, &FileInfo{
			Name:       info.Name(),
			Path:       relPath,
			IsDir:      info.IsDir(),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().Format(time.RFC3339),
			Mode:       info.Mode().String(),
		})

		// Non-recursive: skip subdirectories
		if !recursive && info.IsDir() && walkPath != fullPath {
			return filepath.SkipDir
		}

		return nil
	}

	if err := filepath.Walk(fullPath, walkFn); err != nil {
		return nil, err
	}

	return results, nil
}

// ReadFile reads a file from a sandbox.
func (a *FileOperatorAdapter) ReadFile(ctx context.Context, sandboxID uuid.UUID, path, encoding string, startLine, endLine int) (*FileContent, error) {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return nil, err
	}

	if sb.Status != types.StatusActive {
		return nil, fmt.Errorf("sandbox must be active to read files")
	}

	// Resolve path
	fullPath := filepath.Join(sb.MergedDir, path)

	// Validate path is within sandbox
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(sb.MergedDir)) {
		return nil, fmt.Errorf("path escapes sandbox")
	}

	// Read file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, err
	}

	content := &FileContent{
		Path:     path,
		Size:     info.Size(),
		Encoding: encoding,
	}

	// Handle encoding
	if encoding == "base64" {
		content.Content = string(data)
		content.Encoding = "base64"
	} else {
		// Auto-detect or use utf-8
		content.Content = string(data)
		content.Encoding = "utf-8"
		content.Lines = strings.Count(content.Content, "\n") + 1
	}

	return content, nil
}

// WriteFile writes content to a file in a sandbox.
func (a *FileOperatorAdapter) WriteFile(ctx context.Context, sandboxID uuid.UUID, path, content, encoding string, createDirs bool, mode string) error {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return err
	}

	if sb.Status != types.StatusActive {
		return fmt.Errorf("sandbox must be active to write files")
	}

	// Resolve path
	fullPath := filepath.Join(sb.MergedDir, path)

	// Validate path is within sandbox
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(sb.MergedDir)) {
		return fmt.Errorf("path escapes sandbox")
	}

	// Create parent directories if requested
	if createDirs {
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
	}

	// Parse file mode
	fileMode := os.FileMode(0o644)
	if mode != "" {
		var parsed uint32
		_, _ = fmt.Sscanf(mode, "%o", &parsed)
		if parsed != 0 {
			fileMode = os.FileMode(parsed)
		}
	}

	// Handle encoding
	var data []byte
	if encoding == "base64" {
		var decodeErr error
		data, decodeErr = decodeBase64(content)
		if decodeErr != nil {
			return fmt.Errorf("invalid base64 content: %w", decodeErr)
		}
	} else {
		data = []byte(content)
	}

	return os.WriteFile(fullPath, data, fileMode)
}

// DeleteFile deletes a file or directory from a sandbox.
func (a *FileOperatorAdapter) DeleteFile(ctx context.Context, sandboxID uuid.UUID, path string, recursive bool) error {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return err
	}

	if sb.Status != types.StatusActive {
		return fmt.Errorf("sandbox must be active to delete files")
	}

	// Resolve path
	fullPath := filepath.Join(sb.MergedDir, path)

	// Validate path is within sandbox
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(sb.MergedDir)) {
		return fmt.Errorf("path escapes sandbox")
	}

	// Don't allow deleting the root
	if filepath.Clean(fullPath) == filepath.Clean(sb.MergedDir) {
		return fmt.Errorf("cannot delete sandbox root")
	}

	if recursive {
		return os.RemoveAll(fullPath)
	}
	return os.Remove(fullPath)
}

// CreateDirectory creates a directory in a sandbox.
func (a *FileOperatorAdapter) CreateDirectory(ctx context.Context, sandboxID uuid.UUID, path string, parents bool, mode string) error {
	// Get the sandbox
	sb, err := a.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return err
	}

	if sb.Status != types.StatusActive {
		return fmt.Errorf("sandbox must be active to create directories")
	}

	// Resolve path
	fullPath := filepath.Join(sb.MergedDir, path)

	// Validate path is within sandbox
	if !strings.HasPrefix(filepath.Clean(fullPath), filepath.Clean(sb.MergedDir)) {
		return fmt.Errorf("path escapes sandbox")
	}

	// Parse mode
	dirMode := os.FileMode(0o755)
	if mode != "" {
		var parsed uint32
		_, _ = fmt.Sscanf(mode, "%o", &parsed)
		if parsed != 0 {
			dirMode = os.FileMode(parsed)
		}
	}

	if parents {
		return os.MkdirAll(fullPath, dirMode)
	}
	return os.Mkdir(fullPath, dirMode)
}

// decodeBase64 decodes a base64 string using standard encoding.
func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
