// Package sandbox provides sandbox provider implementations.
//
// This file implements the workspace-sandbox integration that provides
// isolated execution environments for agent runs.
package sandbox

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		return nil, fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, p.parseError(resp)
	}

	var result wsSandboxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.toSandbox(), nil
}

// Get retrieves a sandbox by ID.
func (p *WorkspaceSandboxProvider) Get(ctx context.Context, id uuid.UUID) (*Sandbox, error) {
	resp, err := p.doRequest(ctx, "GET", fmt.Sprintf("/api/v1/sandboxes/%s", id), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("sandbox not found: %s", id)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result wsSandboxResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.toSandbox(), nil
}

// Delete removes a sandbox and its resources.
func (p *WorkspaceSandboxProvider) Delete(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/sandboxes/%s", id), nil)
	if err != nil {
		return fmt.Errorf("failed to delete sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return p.parseError(resp)
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
		return nil, fmt.Errorf("failed to get diff: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result wsDiffResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return nil, fmt.Errorf("failed to approve sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result wsApproveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
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
		return fmt.Errorf("failed to reject sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError(resp)
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
		return nil, fmt.Errorf("failed to partial approve sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, p.parseError(resp)
	}

	var result wsApproveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.toApproveResult(), nil
}

// Stop suspends a sandbox (keeps data but releases mount).
func (p *WorkspaceSandboxProvider) Stop(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/stop", id), nil)
	if err != nil {
		return fmt.Errorf("failed to stop sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError(resp)
	}

	return nil
}

// Start resumes a stopped sandbox.
func (p *WorkspaceSandboxProvider) Start(ctx context.Context, id uuid.UUID) error {
	resp, err := p.doRequest(ctx, "POST", fmt.Sprintf("/api/v1/sandboxes/%s/start", id), nil)
	if err != nil {
		return fmt.Errorf("failed to start sandbox: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return p.parseError(resp)
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
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, p.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return p.httpClient.Do(req)
}

func (p *WorkspaceSandboxProvider) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}
	if errResp.Error != "" {
		return fmt.Errorf("%s: %s", errResp.Error, errResp.Message)
	}
	return fmt.Errorf("request failed with status %d", resp.StatusCode)
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

// Verify interface compliance
var _ Provider = (*WorkspaceSandboxProvider)(nil)
