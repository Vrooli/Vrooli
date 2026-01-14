// Package toolexecution implements the Tool Execution Protocol for workspace-sandbox.
//
// This file provides the ServerExecutor which dispatches tool calls to the appropriate
// handlers and services.
package toolexecution

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/types"

	"github.com/google/uuid"
)

// ProcessExecutor provides process execution methods.
type ProcessExecutor interface {
	ExecSync(ctx context.Context, sandboxID uuid.UUID, req ExecRequest) (*ExecResult, error)
	StartAsync(ctx context.Context, sandboxID uuid.UUID, req ExecRequest) (*ProcessInfo, error)
	GetStatus(ctx context.Context, sandboxID uuid.UUID, pid int) (*ProcessStatus, error)
	List(ctx context.Context, sandboxID uuid.UUID) ([]*ProcessInfo, error)
	Kill(ctx context.Context, sandboxID uuid.UUID, pid int) error
	GetLogs(ctx context.Context, sandboxID uuid.UUID, pid int, tailLines int, stream string) (*ProcessLogs, error)
}

// FileOperator provides file operation methods.
type FileOperator interface {
	ListFiles(ctx context.Context, sandboxID uuid.UUID, path string, recursive, includeHidden bool, pattern string) ([]*FileInfo, error)
	ReadFile(ctx context.Context, sandboxID uuid.UUID, path, encoding string, startLine, endLine int) (*FileContent, error)
	WriteFile(ctx context.Context, sandboxID uuid.UUID, path, content, encoding string, createDirs bool, mode string) error
	DeleteFile(ctx context.Context, sandboxID uuid.UUID, path string, recursive bool) error
	CreateDirectory(ctx context.Context, sandboxID uuid.UUID, path string, parents bool, mode string) error
}

// ExecRequest is the request for command execution.
type ExecRequest struct {
	Command          string
	Args             []string
	WorkingDir       string
	Env              map[string]string
	TimeoutSec       int
	IsolationProfile string
}

// ExecResult is the result of a synchronous command execution.
type ExecResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	PID      int    `json:"pid"`
	TimedOut bool   `json:"timed_out"`
}

// ProcessInfo contains information about a process.
type ProcessInfo struct {
	PID       int    `json:"pid"`
	Command   string `json:"command"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at,omitempty"`
	ExitCode  *int   `json:"exit_code,omitempty"`
}

// ProcessStatus contains the current status of a process.
type ProcessStatus struct {
	PID      int    `json:"pid"`
	Status   string `json:"status"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Running  bool   `json:"running"`
}

// ProcessLogs contains logs from a process.
type ProcessLogs struct {
	PID    int    `json:"pid"`
	Stdout string `json:"stdout,omitempty"`
	Stderr string `json:"stderr,omitempty"`
	Lines  int    `json:"lines"`
}

// FileInfo contains information about a file.
type FileInfo struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	IsDir      bool   `json:"is_dir"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
	Mode       string `json:"mode"`
}

// FileContent contains the content of a file.
type FileContent struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Size     int64  `json:"size"`
	Lines    int    `json:"lines,omitempty"`
}

// ServerExecutorConfig holds dependencies for the ServerExecutor.
type ServerExecutorConfig struct {
	SandboxService  sandbox.ServiceAPI
	ProcessExecutor ProcessExecutor
	FileOperator    FileOperator
}

// ServerExecutor implements tool execution using the sandbox service and handlers.
type ServerExecutor struct {
	sandboxService  sandbox.ServiceAPI
	processExecutor ProcessExecutor
	fileOperator    FileOperator
}

// NewServerExecutor creates a new ServerExecutor with the given configuration.
func NewServerExecutor(cfg ServerExecutorConfig) *ServerExecutor {
	return &ServerExecutor{
		sandboxService:  cfg.SandboxService,
		processExecutor: cfg.ProcessExecutor,
		fileOperator:    cfg.FileOperator,
	}
}

// Execute dispatches tool execution to the appropriate handler.
func (e *ServerExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*ExecutionResult, error) {
	switch toolName {
	// Tier 1: Sandbox Lifecycle
	case "create_sandbox":
		return e.createSandbox(ctx, args)
	case "get_sandbox":
		return e.getSandbox(ctx, args)
	case "list_sandboxes":
		return e.listSandboxes(ctx, args)
	case "delete_sandbox":
		return e.deleteSandbox(ctx, args)
	case "start_sandbox":
		return e.startSandbox(ctx, args)
	case "stop_sandbox":
		return e.stopSandbox(ctx, args)

	// Tier 2: Command Execution
	case "execute_command":
		return e.executeCommand(ctx, args)
	case "start_process":
		return e.startProcess(ctx, args)
	case "get_process_status":
		return e.getProcessStatus(ctx, args)
	case "list_processes":
		return e.listProcesses(ctx, args)
	case "stop_process":
		return e.stopProcess(ctx, args)
	case "get_process_logs":
		return e.getProcessLogs(ctx, args)

	// Tier 3: File Operations
	case "list_files":
		return e.listFiles(ctx, args)
	case "read_file":
		return e.readFile(ctx, args)
	case "write_file":
		return e.writeFile(ctx, args)
	case "delete_file":
		return e.deleteFile(ctx, args)
	case "create_directory":
		return e.createDirectory(ctx, args)

	// Tier 4: Diff & Approval
	case "get_diff":
		return e.getDiff(ctx, args)
	case "approve_changes":
		return e.approveChanges(ctx, args)
	case "reject_changes":
		return e.rejectChanges(ctx, args)
	case "discard_files":
		return e.discardFiles(ctx, args)

	default:
		return ErrorResult("unknown tool: "+toolName, CodeUnknownTool), nil
	}
}

// -----------------------------------------------------------------------------
// Tier 1: Sandbox Lifecycle
// -----------------------------------------------------------------------------

func (e *ServerExecutor) createSandbox(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	scopePath := getStringArg(args, "scope_path", "")
	if scopePath == "" {
		return ErrorResult("scope_path is required", CodeInvalidArgs), nil
	}

	projectRoot := getStringArg(args, "project_root", "")
	if projectRoot == "" {
		projectRoot = os.Getenv("VROOLI_ROOT")
		if projectRoot == "" {
			projectRoot = os.Getenv("HOME") + "/Vrooli"
		}
	}

	owner := getStringArg(args, "owner", "")
	ownerTypeStr := getStringArg(args, "owner_type", "agent")
	noLock := getBoolArg(args, "no_lock", false)

	// Extract metadata if provided
	var metadata map[string]interface{}
	if m, ok := args["metadata"].(map[string]interface{}); ok {
		metadata = m
	}

	// Create the sandbox request using types package
	req := &types.CreateRequest{
		ScopePath:   scopePath,
		ProjectRoot: projectRoot,
		Owner:       owner,
		OwnerType:   types.OwnerType(ownerTypeStr),
		NoLock:      noLock,
		Metadata:    metadata,
	}

	sb, err := e.sandboxService.Create(ctx, req)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"sandbox_id":   sb.ID.String(),
		"status":       string(sb.Status),
		"scope_path":   sb.ScopePath,
		"project_root": sb.ProjectRoot,
		"merged_path":  sb.MergedDir,
		"created_at":   sb.CreatedAt,
	}), nil
}

func (e *ServerExecutor) getSandbox(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	sb, err := e.sandboxService.Get(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(sandboxToResponse(sb)), nil
}

func (e *ServerExecutor) listSandboxes(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	filter := &types.ListFilter{
		Owner:       getStringArg(args, "owner", ""),
		ProjectRoot: getStringArg(args, "project_root", ""),
		Limit:       getIntArg(args, "limit", 50),
		Offset:      getIntArg(args, "offset", 0),
	}

	// Parse status filter if provided
	if statusStr := getStringArg(args, "status", ""); statusStr != "" {
		filter.Status = []types.Status{types.Status(statusStr)}
	}

	result, err := e.sandboxService.List(ctx, filter)
	if err != nil {
		return handleDomainError(err)
	}

	results := make([]map[string]interface{}, len(result.Sandboxes))
	for i, sb := range result.Sandboxes {
		results[i] = sandboxToResponse(sb)
	}

	return SuccessResult(map[string]interface{}{
		"sandboxes": results,
		"count":     len(results),
		"total":     result.TotalCount,
	}), nil
}

func (e *ServerExecutor) deleteSandbox(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	// Note: force parameter handled at service level via sandbox state
	err = e.sandboxService.Delete(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success":    true,
		"sandbox_id": sandboxID.String(),
		"message":    "Sandbox deleted successfully",
	}), nil
}

func (e *ServerExecutor) startSandbox(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	sb, err := e.sandboxService.Start(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success":     true,
		"sandbox_id":  sb.ID.String(),
		"status":      string(sb.Status),
		"merged_path": sb.MergedDir,
		"message":     "Sandbox started successfully",
	}), nil
}

func (e *ServerExecutor) stopSandbox(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	sb, err := e.sandboxService.Stop(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success":    true,
		"sandbox_id": sb.ID.String(),
		"status":     string(sb.Status),
		"message":    "Sandbox stopped successfully",
	}), nil
}

// -----------------------------------------------------------------------------
// Tier 2: Command Execution
// -----------------------------------------------------------------------------

func (e *ServerExecutor) executeCommand(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	command := getStringArg(args, "command", "")
	if command == "" {
		return ErrorResult("command is required", CodeInvalidArgs), nil
	}

	req := ExecRequest{
		Command:          command,
		Args:             getStringArrayArg(args, "args"),
		WorkingDir:       getStringArg(args, "working_dir", ""),
		Env:              getStringMapArg(args, "env"),
		TimeoutSec:       getIntArg(args, "timeout_sec", 60),
		IsolationProfile: getStringArg(args, "isolation_profile", "restricted"),
	}

	result, err := e.processExecutor.ExecSync(ctx, sandboxID, req)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"exit_code": result.ExitCode,
		"stdout":    result.Stdout,
		"stderr":    result.Stderr,
		"pid":       result.PID,
		"timed_out": result.TimedOut,
	}), nil
}

func (e *ServerExecutor) startProcess(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	command := getStringArg(args, "command", "")
	if command == "" {
		return ErrorResult("command is required", CodeInvalidArgs), nil
	}

	req := ExecRequest{
		Command:          command,
		Args:             getStringArrayArg(args, "args"),
		WorkingDir:       getStringArg(args, "working_dir", ""),
		Env:              getStringMapArg(args, "env"),
		IsolationProfile: getStringArg(args, "isolation_profile", "restricted"),
	}

	result, err := e.processExecutor.StartAsync(ctx, sandboxID, req)
	if err != nil {
		return handleDomainError(err)
	}

	return AsyncResult(map[string]interface{}{
		"sandbox_id": sandboxID.String(),
		"pid":        result.PID,
		"status":     "running",
		"message":    fmt.Sprintf("Process started with PID %d", result.PID),
	}, strconv.Itoa(result.PID)), nil
}

func (e *ServerExecutor) getProcessStatus(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	pid := getIntArg(args, "pid", 0)
	if pid == 0 {
		return ErrorResult("pid is required", CodeInvalidArgs), nil
	}

	status, err := e.processExecutor.GetStatus(ctx, sandboxID, pid)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"pid":       status.PID,
		"status":    status.Status,
		"running":   status.Running,
		"exit_code": status.ExitCode,
	}), nil
}

func (e *ServerExecutor) listProcesses(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	processes, err := e.processExecutor.List(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"processes": processes,
		"count":     len(processes),
	}), nil
}

func (e *ServerExecutor) stopProcess(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	pid := getIntArg(args, "pid", 0)
	if pid == 0 {
		return ErrorResult("pid is required", CodeInvalidArgs), nil
	}

	err = e.processExecutor.Kill(ctx, sandboxID, pid)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"pid":     pid,
		"message": fmt.Sprintf("Process %d stopped", pid),
	}), nil
}

func (e *ServerExecutor) getProcessLogs(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	pid := getIntArg(args, "pid", 0)
	if pid == 0 {
		return ErrorResult("pid is required", CodeInvalidArgs), nil
	}

	tailLines := getIntArg(args, "tail_lines", 100)
	stream := getStringArg(args, "stream", "combined")

	logs, err := e.processExecutor.GetLogs(ctx, sandboxID, pid, tailLines, stream)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"pid":    logs.PID,
		"stdout": logs.Stdout,
		"stderr": logs.Stderr,
		"lines":  logs.Lines,
	}), nil
}

// -----------------------------------------------------------------------------
// Tier 3: File Operations
// -----------------------------------------------------------------------------

func (e *ServerExecutor) listFiles(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path := getStringArg(args, "path", ".")
	recursive := getBoolArg(args, "recursive", false)
	includeHidden := getBoolArg(args, "include_hidden", false)
	pattern := getStringArg(args, "pattern", "")

	files, err := e.fileOperator.ListFiles(ctx, sandboxID, path, recursive, includeHidden, pattern)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"files": files,
		"count": len(files),
	}), nil
}

func (e *ServerExecutor) readFile(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path := getStringArg(args, "path", "")
	if path == "" {
		return ErrorResult("path is required", CodeInvalidArgs), nil
	}

	encoding := getStringArg(args, "encoding", "auto")
	startLine := getIntArg(args, "start_line", 0)
	endLine := getIntArg(args, "end_line", 0)

	content, err := e.fileOperator.ReadFile(ctx, sandboxID, path, encoding, startLine, endLine)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"path":     content.Path,
		"content":  content.Content,
		"encoding": content.Encoding,
		"size":     content.Size,
		"lines":    content.Lines,
	}), nil
}

func (e *ServerExecutor) writeFile(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path := getStringArg(args, "path", "")
	if path == "" {
		return ErrorResult("path is required", CodeInvalidArgs), nil
	}

	content := getStringArg(args, "content", "")
	encoding := getStringArg(args, "encoding", "utf-8")
	createDirs := getBoolArg(args, "create_dirs", true)
	mode := getStringArg(args, "mode", "0644")

	err = e.fileOperator.WriteFile(ctx, sandboxID, path, content, encoding, createDirs, mode)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "File written successfully",
	}), nil
}

func (e *ServerExecutor) deleteFile(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path := getStringArg(args, "path", "")
	if path == "" {
		return ErrorResult("path is required", CodeInvalidArgs), nil
	}

	recursive := getBoolArg(args, "recursive", false)

	err = e.fileOperator.DeleteFile(ctx, sandboxID, path, recursive)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "File deleted successfully",
	}), nil
}

func (e *ServerExecutor) createDirectory(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	path := getStringArg(args, "path", "")
	if path == "" {
		return ErrorResult("path is required", CodeInvalidArgs), nil
	}

	parents := getBoolArg(args, "parents", true)
	mode := getStringArg(args, "mode", "0755")

	err = e.fileOperator.CreateDirectory(ctx, sandboxID, path, parents, mode)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success": true,
		"path":    path,
		"message": "Directory created successfully",
	}), nil
}

// -----------------------------------------------------------------------------
// Tier 4: Diff & Approval
// -----------------------------------------------------------------------------

func (e *ServerExecutor) getDiff(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	// Note: context_lines and path_filter parameters may require service changes
	// For now, use the basic GetDiff method
	diff, err := e.sandboxService.GetDiff(ctx, sandboxID)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(diff), nil
}

func (e *ServerExecutor) approveChanges(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	commitMessage := getStringArg(args, "commit_message", "")
	actor := getStringArg(args, "actor", "tool-execution")

	// Determine approval mode based on files parameter
	mode := "all"
	var filePaths []string
	if files := getStringArrayArg(args, "files"); len(files) > 0 {
		mode = "files"
		filePaths = files
	}

	req := &types.ApprovalRequest{
		SandboxID: sandboxID,
		Mode:      mode,
		Actor:     actor,
		CommitMsg: commitMessage,
	}

	// Note: The current API uses FileIDs (UUIDs), but we're passing file paths
	// This may need adjustment based on actual service implementation
	_ = filePaths // TODO: Convert file paths to FileIDs if needed

	result, err := e.sandboxService.Approve(ctx, req)
	if err != nil {
		return handleDomainError(err)
	}

	response := map[string]interface{}{
		"success":     result.Success,
		"applied":     result.Applied,
		"remaining":   result.Remaining,
		"commit_hash": result.CommitHash,
	}
	if result.ErrorMsg != "" {
		response["error"] = result.ErrorMsg
	}
	return SuccessResult(response), nil
}

func (e *ServerExecutor) rejectChanges(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	actor := getStringArg(args, "actor", "tool-execution")

	_, err = e.sandboxService.Reject(ctx, sandboxID, actor)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success":    true,
		"sandbox_id": sandboxID.String(),
		"message":    "Changes rejected successfully",
	}), nil
}

func (e *ServerExecutor) discardFiles(ctx context.Context, args map[string]interface{}) (*ExecutionResult, error) {
	sandboxID, err := parseSandboxID(args)
	if err != nil {
		return ErrorResult(err.Error(), CodeInvalidArgs), nil
	}

	files := getStringArrayArg(args, "files")
	if len(files) == 0 {
		return ErrorResult("files is required", CodeInvalidArgs), nil
	}

	actor := getStringArg(args, "actor", "tool-execution")

	req := &types.DiscardRequest{
		SandboxID: sandboxID,
		FilePaths: files,
		Actor:     actor,
	}

	result, err := e.sandboxService.Discard(ctx, req)
	if err != nil {
		return handleDomainError(err)
	}

	return SuccessResult(map[string]interface{}{
		"success":        result.Success,
		"discarded":      result.Discarded,
		"remaining":      result.Remaining,
		"files_reverted": result.Files,
		"message":        fmt.Sprintf("Discarded changes to %d files", result.Discarded),
	}), nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// parseSandboxID extracts and validates the sandbox_id argument.
func parseSandboxID(args map[string]interface{}) (uuid.UUID, error) {
	sandboxIDStr := getStringArg(args, "sandbox_id", "")
	if sandboxIDStr == "" {
		return uuid.Nil, fmt.Errorf("sandbox_id is required")
	}

	sandboxID, err := uuid.Parse(sandboxIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid sandbox_id: %v", err)
	}

	return sandboxID, nil
}

// handleDomainError converts domain errors to ExecutionResult.
func handleDomainError(err error) (*ExecutionResult, error) {
	if err == nil {
		return nil, nil
	}

	// Check if it's a domain error with specific code
	if domainErr, ok := err.(types.DomainError); ok {
		code := CodeInternalError
		switch {
		case domainErr.HTTPStatus() == 404:
			code = CodeNotFound
		case domainErr.HTTPStatus() == 400:
			code = CodeInvalidArgs
		case domainErr.HTTPStatus() == 409:
			code = CodeConflict
		}
		return ErrorResult(domainErr.Error(), code), nil
	}

	return ErrorResult(err.Error(), CodeInternalError), nil
}

// sandboxToResponse converts a sandbox to a response map.
func sandboxToResponse(sb *types.Sandbox) map[string]interface{} {
	resp := map[string]interface{}{
		"sandbox_id":   sb.ID.String(),
		"status":       string(sb.Status),
		"scope_path":   sb.ScopePath,
		"project_root": sb.ProjectRoot,
		"merged_path":  sb.MergedDir,
		"owner":        sb.Owner,
		"owner_type":   sb.OwnerType,
		"created_at":   sb.CreatedAt,
		"updated_at":   sb.UpdatedAt,
	}
	if sb.Metadata != nil {
		resp["metadata"] = sb.Metadata
	}
	return resp
}

// getStringArg extracts a string argument with a default value.
func getStringArg(args map[string]interface{}, key, defaultValue string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return defaultValue
}

// getIntArg extracts an int argument with a default value.
// Handles both int and float64 (JSON numbers decode as float64).
func getIntArg(args map[string]interface{}, key string, defaultValue int) int {
	if v, ok := args[key].(float64); ok {
		return int(v)
	}
	if v, ok := args[key].(int); ok {
		return v
	}
	return defaultValue
}

// getBoolArg extracts a bool argument with a default value.
func getBoolArg(args map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return defaultValue
}

// getStringArrayArg extracts a string array argument.
func getStringArrayArg(args map[string]interface{}, key string) []string {
	if arr, ok := args[key].([]interface{}); ok {
		result := make([]string, 0, len(arr))
		for _, v := range arr {
			if s, ok := v.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	if arr, ok := args[key].([]string); ok {
		return arr
	}
	return nil
}

// getStringMapArg extracts a string map argument.
func getStringMapArg(args map[string]interface{}, key string) map[string]string {
	if m, ok := args[key].(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result
	}
	if m, ok := args[key].(map[string]string); ok {
		return m
	}
	return nil
}
