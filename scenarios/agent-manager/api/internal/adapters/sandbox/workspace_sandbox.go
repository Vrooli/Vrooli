// Package sandbox provides sandbox provider implementations.
//
// This file implements the workspace-sandbox integration that provides
// isolated execution environments for agent runs.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

// =============================================================================
// Workspace Sandbox Provider Implementation
// =============================================================================

// WorkspaceSandboxProvider implements the Provider interface using workspace-sandbox.
type WorkspaceSandboxProvider struct {
	baseURL    string
	httpClient *http.Client
}

// NewWorkspaceSandboxProvider creates a new workspace-sandbox provider.
func NewWorkspaceSandboxProvider(baseURL string) *WorkspaceSandboxProvider {
	return &WorkspaceSandboxProvider{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Create creates a new sandbox for the given scope.
func (p *WorkspaceSandboxProvider) Create(ctx context.Context, req CreateRequest) (*Sandbox, error) {
	body := map[string]interface{}{
		"scopePath":   req.ScopePath,
		"projectRoot": req.ProjectRoot,
		"owner":       req.Owner,
		"ownerType":   req.OwnerType,
		"metadata":    req.Metadata,
	}
	if req.IdempotencyKey != "" {
		body["idempotencyKey"] = req.IdempotencyKey
	}

	resp, err := p.doRequest(ctx, "POST", "/api/v1/sandboxes", body)
	if err != nil {
		return nil, &domain.SandboxError{
			Operation:   "create",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, p.parseError("create", nil, resp)
	}

	var result wsSandboxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			Operation: "create",
			Cause:     err,
		}
	}

	return result.toSandbox(), nil
}

// Get retrieves a sandbox by ID.
func (p *WorkspaceSandboxProvider) Get(ctx context.Context, id uuid.UUID) (*Sandbox, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/sandboxes/%s", id), nil)
	if err != nil {
		return nil, &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "get",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.NewNotFoundErrorWithID("Sandbox", id.String())
	}
	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("get", &id, resp)
	}

	var result wsSandboxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			SandboxID: &id,
			Operation: "get",
			Cause:     err,
		}
	}

	return result.toSandbox(), nil
}

// Delete removes a sandbox and its resources.
func (p *WorkspaceSandboxProvider) Delete(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/sandboxes/%s", id), nil)
	if err != nil {
		return &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "delete",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return p.parseError("delete", &id, resp)
	}

	return nil
}

// GetWorkspacePath returns the path where agents should execute.
func (p *WorkspaceSandboxProvider) GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error) {
	sandbox, err := p.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return sandbox.WorkDir, nil
}

// GetDiff generates a diff of changes made in the sandbox.
func (p *WorkspaceSandboxProvider) GetDiff(ctx context.Context, id uuid.UUID) (*DiffResult, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/sandboxes/%s/diff", id), nil)
	if err != nil {
		return nil, &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "diff",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("diff", &id, resp)
	}

	var result wsDiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			SandboxID: &id,
			Operation: "diff",
			Cause:     err,
		}
	}

	return result.toDiffResult(id), nil
}

// Approve applies sandbox changes to the canonical repository.
func (p *WorkspaceSandboxProvider) Approve(ctx context.Context, req ApproveRequest) (*ApproveResult, error) {
	body := map[string]interface{}{
		"actor":     req.Actor,
		"commitMsg": req.CommitMsg,
		"force":     req.Force,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/approve", req.SandboxID), body)
	if err != nil {
		return nil, &domain.SandboxError{
			SandboxID:   &req.SandboxID,
			Operation:   "approve",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("approve", &req.SandboxID, resp)
	}

	var result wsApproveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			SandboxID: &req.SandboxID,
			Operation: "approve",
			Cause:     err,
		}
	}

	return result.toApproveResult(), nil
}

// Reject marks sandbox changes as rejected without applying.
func (p *WorkspaceSandboxProvider) Reject(ctx context.Context, id uuid.UUID, actor string) error {
	body := map[string]interface{}{
		"actor": actor,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/reject", id), body)
	if err != nil {
		return &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "reject",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError("reject", &id, resp)
	}

	return nil
}

// PartialApprove approves only selected files from the sandbox.
func (p *WorkspaceSandboxProvider) PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error) {
	// Convert UUIDs to strings for the API
	fileIDs := make([]string, len(req.FileIDs))
	for i, id := range req.FileIDs {
		fileIDs[i] = id.String()
	}

	body := map[string]interface{}{
		"actor":     req.Actor,
		"commitMsg": req.CommitMsg,
		"fileIds":   fileIDs,
	}

	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/partial-approve", req.SandboxID), body)
	if err != nil {
		return nil, &domain.SandboxError{
			SandboxID:   &req.SandboxID,
			Operation:   "partial_approve",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("partial_approve", &req.SandboxID, resp)
	}

	var result wsApproveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			SandboxID: &req.SandboxID,
			Operation: "partial_approve",
			Cause:     err,
		}
	}

	return result.toApproveResult(), nil
}

// Stop suspends a sandbox (keeps data but releases mount).
func (p *WorkspaceSandboxProvider) Stop(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/stop", id), nil)
	if err != nil {
		return &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "stop",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError("stop", &id, resp)
	}

	return nil
}

// Start resumes a stopped sandbox.
func (p *WorkspaceSandboxProvider) Start(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/start", id), nil)
	if err != nil {
		return &domain.SandboxError{
			SandboxID:   &id,
			Operation:   "start",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError("start", &id, resp)
	}

	return nil
}

// IsAvailable checks if the sandbox provider is operational.
func (p *WorkspaceSandboxProvider) IsAvailable(ctx context.Context) (bool, string) {
	resp, err := p.doRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return false, fmt.Sprintf("workspace-sandbox unreachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("workspace-sandbox unhealthy: status %d", resp.StatusCode)
	}

	var health struct {
		Status    string `json:"status"`
		Readiness bool   `json:"readiness"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return false, "failed to parse health response"
	}

	if !health.Readiness {
		return false, "workspace-sandbox not ready"
	}

	return true, "workspace-sandbox is available"
}

// =============================================================================
// HTTP Helpers
// =============================================================================

func (p *WorkspaceSandboxProvider) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return p.httpClient.Do(req)
}

// SandboxAPIError represents a structured error response from the workspace-sandbox API.
type SandboxAPIError struct {
	ErrorMsg  string                 `json:"error"`
	Code      int                    `json:"code"`
	Hint      string                 `json:"hint,omitempty"`
	Retryable bool                   `json:"retryable,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func (e *SandboxAPIError) Error() string {
	return e.ErrorMsg
}

// ConflictingSandbox represents a sandbox that conflicts with a requested operation.
type ConflictingSandbox struct {
	SandboxID    string `json:"sandboxId"`
	Scope        string `json:"scope"`
	ConflictType string `json:"conflictType"`
}

// GetConflicts extracts conflict details if this is a scope conflict error.
func (e *SandboxAPIError) GetConflicts() []ConflictingSandbox {
	if e.Details == nil {
		return nil
	}
	conflictsRaw, ok := e.Details["conflicts"]
	if !ok {
		return nil
	}
	conflictsSlice, ok := conflictsRaw.([]interface{})
	if !ok {
		return nil
	}

	conflicts := make([]ConflictingSandbox, 0, len(conflictsSlice))
	for _, c := range conflictsSlice {
		cMap, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		conflict := ConflictingSandbox{}
		if id, ok := cMap["sandboxId"].(string); ok {
			conflict.SandboxID = id
		}
		if scope, ok := cMap["scope"].(string); ok {
			conflict.Scope = scope
		}
		if ct, ok := cMap["conflictType"].(string); ok {
			conflict.ConflictType = ct
		}
		conflicts = append(conflicts, conflict)
	}
	return conflicts
}

func (p *WorkspaceSandboxProvider) parseError(operation string, sandboxID *uuid.UUID, resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp SandboxAPIError
	if err := json.Unmarshal(body, &errResp); err != nil {
		return &domain.SandboxError{
			SandboxID: sandboxID,
			Operation: operation,
			Cause:     domain.NewInternalError("failed to parse sandbox error response", err),
		}
	}
	errResp.Code = resp.StatusCode
	if errResp.ErrorMsg != "" {
		details := map[string]interface{}{}
		for k, v := range errResp.Details {
			details[k] = v
		}
		if errResp.Hint != "" {
			details["hint"] = errResp.Hint
		}
		return &domain.SandboxError{
			SandboxID:    sandboxID,
			Operation:    operation,
			Cause:        errors.New(errResp.ErrorMsg),
			IsTransient:  errResp.Retryable || resp.StatusCode >= http.StatusInternalServerError,
			CanRetry:     errResp.Retryable,
			ExtraDetails: details,
		}
	}
	return &domain.SandboxError{
		SandboxID:   sandboxID,
		Operation:   operation,
		Cause:       errors.New(fmt.Sprintf("request failed with status %d", resp.StatusCode)),
		IsTransient: resp.StatusCode >= http.StatusInternalServerError,
	}
}

// =============================================================================
// Response Types (map workspace-sandbox API responses to our types)
// =============================================================================

type wsSandboxResponse struct {
	ID          string            `json:"id"`
	ScopePath   string            `json:"scopePath"`
	ProjectRoot string            `json:"projectRoot"`
	Status      string            `json:"status"`
	MergedDir   string            `json:"mergedDir"`
	CreatedAt   time.Time         `json:"createdAt"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (r *wsSandboxResponse) toSandbox() *Sandbox {
	id, _ := uuid.Parse(r.ID)
	return &Sandbox{
		ID:          id,
		ScopePath:   r.ScopePath,
		ProjectRoot: r.ProjectRoot,
		Status:      SandboxStatus(r.Status),
		WorkDir:     r.MergedDir,
		CreatedAt:   r.CreatedAt,
		Metadata:    r.Metadata,
	}
}

type wsDiffResponse struct {
	Files       []wsFileChange `json:"files"`
	UnifiedDiff string         `json:"unifiedDiff"`
	Stats       wsDiffStats    `json:"stats"`
}

type wsFileChange struct {
	ID           string `json:"id"`
	FilePath     string `json:"filePath"`
	ChangeType   string `json:"changeType"`
	FileSize     int64  `json:"fileSize"`
	LinesAdded   int    `json:"linesAdded"`
	LinesRemoved int    `json:"linesRemoved"`
}

type wsDiffStats struct {
	FilesChanged  int   `json:"filesChanged"`
	FilesAdded    int   `json:"filesAdded"`
	FilesModified int   `json:"filesModified"`
	FilesDeleted  int   `json:"filesDeleted"`
	TotalLines    int   `json:"totalLines"`
	LinesAdded    int   `json:"linesAdded"`
	LinesRemoved  int   `json:"linesRemoved"`
	TotalBytes    int64 `json:"totalBytes"`
}

func (r *wsDiffResponse) toDiffResult(sandboxID uuid.UUID) *DiffResult {
	files := make([]FileChange, len(r.Files))
	for i, f := range r.Files {
		id, _ := uuid.Parse(f.ID)
		files[i] = FileChange{
			ID:           id,
			FilePath:     f.FilePath,
			ChangeType:   FileChangeType(f.ChangeType),
			FileSize:     f.FileSize,
			LinesAdded:   f.LinesAdded,
			LinesRemoved: f.LinesRemoved,
		}
	}

	return &DiffResult{
		SandboxID:   sandboxID,
		Files:       files,
		UnifiedDiff: r.UnifiedDiff,
		Generated:   time.Now(),
		Stats: DiffStats{
			FilesChanged:  r.Stats.FilesChanged,
			FilesAdded:    r.Stats.FilesAdded,
			FilesModified: r.Stats.FilesModified,
			FilesDeleted:  r.Stats.FilesDeleted,
			TotalLines:    r.Stats.TotalLines,
			LinesAdded:    r.Stats.LinesAdded,
			LinesRemoved:  r.Stats.LinesRemoved,
			TotalBytes:    r.Stats.TotalBytes,
		},
	}
}

type wsApproveResponse struct {
	Success    bool      `json:"success"`
	Applied    int       `json:"applied"`
	Remaining  int       `json:"remaining"`
	IsPartial  bool      `json:"isPartial"`
	CommitHash string    `json:"commitHash"`
	AppliedAt  time.Time `json:"appliedAt"`
	ErrorMsg   string    `json:"errorMsg"`
}

func (r *wsApproveResponse) toApproveResult() *ApproveResult {
	return &ApproveResult{
		Success:    r.Success,
		Applied:    r.Applied,
		Remaining:  r.Remaining,
		IsPartial:  r.IsPartial,
		CommitHash: r.CommitHash,
		AppliedAt:  r.AppliedAt,
		ErrorMsg:   r.ErrorMsg,
	}
}

// =============================================================================
// Conflict Detection and Cleanup
// =============================================================================

// CheckConflicts checks if a scope path would conflict with existing sandboxes.
// Returns any conflicting sandboxes without attempting to create a new one.
func (p *WorkspaceSandboxProvider) CheckConflicts(ctx context.Context, scopePath string) ([]ConflictingSandbox, error) {
	resp, err := p.doRequest(ctx, "GET", "/api/v1/sandboxes", nil)
	if err != nil {
		return nil, &domain.SandboxError{
			Operation:   "check_conflicts",
			Cause:       err,
			IsTransient: true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("check_conflicts", nil, resp)
	}

	var result struct {
		Sandboxes []wsSandboxResponse `json:"sandboxes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			Operation:   "check_conflicts",
			Cause:       err,
			IsTransient: false,
		}
	}

	// Check for overlapping scopes
	var conflicts []ConflictingSandbox
	for _, sb := range result.Sandboxes {
		// Skip deleted/rejected sandboxes
		if sb.Status == "deleted" || sb.Status == "rejected" {
			continue
		}
		// Check for overlap (simple prefix matching)
		if pathsOverlap(scopePath, sb.ScopePath) {
			conflicts = append(conflicts, ConflictingSandbox{
				SandboxID:    sb.ID,
				Scope:        sb.ScopePath,
				ConflictType: "scope_overlap",
			})
		}
	}

	return conflicts, nil
}

// pathsOverlap checks if two paths would conflict (one is prefix of other).
func pathsOverlap(path1, path2 string) bool {
	// Normalize paths
	if path1 == path2 {
		return true
	}
	// Check if one is a prefix of the other
	if len(path1) > len(path2) {
		return len(path2) > 0 && (path1[:len(path2)] == path2 && (len(path1) == len(path2) || path1[len(path2)] == '/'))
	}
	return len(path1) > 0 && (path2[:len(path1)] == path1 && (len(path2) == len(path1) || path2[len(path1)] == '/'))
}

// FormatConflictError creates a user-friendly error message for scope conflicts.
func FormatConflictError(conflicts []ConflictingSandbox) string {
	if len(conflicts) == 0 {
		return ""
	}

	var msg string
	msg = fmt.Sprintf("Cannot create sandbox - scope conflicts detected with %d existing sandbox(es):\n", len(conflicts))
	for _, c := range conflicts {
		shortID := c.SandboxID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		msg += fmt.Sprintf("  - Sandbox %s manages scope: %s\n", shortID, c.Scope)
	}
	msg += "\nHint: Delete conflicting sandboxes or choose a different scope path.\n"
	msg += "Use 'vrooli sandbox list' to see all sandboxes and 'vrooli sandbox delete <id>' to remove conflicts."
	return msg
}

// List returns all sandboxes, optionally filtered by status.
func (p *WorkspaceSandboxProvider) List(ctx context.Context, status string) ([]*Sandbox, error) {
	path := "/api/v1/sandboxes"
	if status != "" {
		path += "?status=" + status
	}

	resp, err := p.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, &domain.SandboxError{
			Operation:   "list",
			Cause:       err,
			IsTransient: true,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError("list", nil, resp)
	}

	var result struct {
		Sandboxes []wsSandboxResponse `json:"sandboxes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, &domain.SandboxError{
			Operation:   "list",
			Cause:       err,
			IsTransient: false,
		}
	}

	sandboxes := make([]*Sandbox, len(result.Sandboxes))
	for i, sb := range result.Sandboxes {
		sandboxes[i] = sb.toSandbox()
	}
	return sandboxes, nil
}

// CleanupStaleSandboxes deletes sandboxes that haven't been used within the given duration.
// Returns the number of sandboxes deleted and any errors encountered.
func (p *WorkspaceSandboxProvider) CleanupStaleSandboxes(ctx context.Context, olderThan time.Duration) (int, error) {
	sandboxes, err := p.List(ctx, "")
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().Add(-olderThan)
	deleted := 0

	for _, sb := range sandboxes {
		// Skip recently used sandboxes
		if sb.CreatedAt.After(cutoff) {
			continue
		}
		// Skip already deleted sandboxes
		if sb.Status == SandboxStatusDeleted {
			continue
		}
		// Delete stale sandbox
		if err := p.Delete(ctx, sb.ID); err != nil {
			// Log but continue with other deletions
			continue
		}
		deleted++
	}

	return deleted, nil
}

// Verify interface compliance
var _ Provider = (*WorkspaceSandboxProvider)(nil)
