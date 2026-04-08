package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// WorkspaceSandboxClient is a lightweight client for workspace-sandbox APIs.
type WorkspaceSandboxClient struct {
	BaseClient
}

// NewWorkspaceSandboxClient creates a new workspace-sandbox client.
func NewWorkspaceSandboxClient(timeout time.Duration) *WorkspaceSandboxClient {
	return &WorkspaceSandboxClient{
		BaseClient: NewBaseClient("workspace-sandbox", timeout),
	}
}

type workspaceSandboxCommitPreviewRequest struct {
	ProjectRoot string   `json:"projectRoot,omitempty"`
	FilePaths   []string `json:"filePaths,omitempty"`
}

type workspaceSandboxCommitPreview struct {
	Files            []workspaceSandboxCommitPreviewFile `json:"files"`
	CommittableFiles int                                 `json:"committableFiles"`
	SuggestedMessage string                              `json:"suggestedMessage"`
}

type workspaceSandboxCommitPreviewFile struct {
	FilePath          string `json:"filePath"`
	RelativePath      string `json:"relativePath"`
	ChangeType        string `json:"changeType"`
	SandboxID         string `json:"sandboxId"`
	SandboxOwner      string `json:"sandboxOwner"`
	AgentManagerRunID string `json:"agentManagerRunId"`
	Status            string `json:"status"`
}

type workspaceSandboxMarkCommittedRequest struct {
	ProjectRoot   string   `json:"projectRoot"`
	FilePaths     []string `json:"filePaths"`
	CommitHash    string   `json:"commitHash"`
	CommitMessage string   `json:"commitMessage"`
}

type workspaceSandboxProvenanceResponse struct {
	RunGroups []workspaceSandboxProvenanceRunGroup `json:"runGroups"`
}

type workspaceSandboxProvenanceRunGroup struct {
	RunID           string                           `json:"runId"`
	SandboxID       string                           `json:"sandboxId"`
	SandboxOwner    string                           `json:"sandboxOwner"`
	Files           []workspaceSandboxProvenanceFile `json:"files"`
	LatestAppliedAt string                           `json:"latestAppliedAt"`
}

type workspaceSandboxProvenanceFile struct {
	FilePath     string `json:"filePath"`
	RelativePath string `json:"relativePath"`
	ChangeType   string `json:"changeType"`
	AppliedAt    string `json:"appliedAt"`
}

func (c *WorkspaceSandboxClient) GetCommitPreview(ctx context.Context, projectRoot string) (*workspaceSandboxCommitPreview, error) {
	params := ""
	if projectRoot != "" {
		params = "?projectRoot=" + url.QueryEscape(projectRoot)
	}
	var result workspaceSandboxCommitPreview
	if err := c.doGet(ctx, "/api/v1/commit-preview"+params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *WorkspaceSandboxClient) GetCommitPreviewForPaths(ctx context.Context, projectRoot string, paths []string) (*workspaceSandboxCommitPreview, error) {
	req := workspaceSandboxCommitPreviewRequest{
		ProjectRoot: projectRoot,
		FilePaths:   paths,
	}
	var result workspaceSandboxCommitPreview
	if err := c.doJSON(ctx, "/api/v1/commit-preview", req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// MarkCommitted notifies workspace-sandbox that files have been committed externally.
func (c *WorkspaceSandboxClient) MarkCommitted(ctx context.Context, projectRoot string, filePaths []string, commitHash, commitMessage string) error {
	body, err := json.Marshal(workspaceSandboxMarkCommittedRequest{
		ProjectRoot:   projectRoot,
		FilePaths:     filePaths,
		CommitHash:    commitHash,
		CommitMessage: commitMessage,
	})
	if err != nil {
		return fmt.Errorf("marshal mark-committed request: %w", err)
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return fmt.Errorf("resolve workspace-sandbox url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/mark-committed", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create mark-committed request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mark-committed request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.parseError(resp)
	}
	return nil
}

// GetProvenanceByRun returns pending applied changes grouped by agent-manager run ID.
func (c *WorkspaceSandboxClient) GetProvenanceByRun(ctx context.Context, projectRoot string) (*workspaceSandboxProvenanceResponse, error) {
	params := ""
	if projectRoot != "" {
		params = "?projectRoot=" + url.QueryEscape(projectRoot)
	}
	var result workspaceSandboxProvenanceResponse
	if err := c.doGet(ctx, "/api/v1/provenance/by-run"+params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
