package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// WorkspaceSandboxClient is a lightweight client for workspace-sandbox APIs.
type WorkspaceSandboxClient struct {
	httpClient *http.Client
}

// NewWorkspaceSandboxClient creates a new workspace-sandbox client.
func NewWorkspaceSandboxClient(timeout time.Duration) *WorkspaceSandboxClient {
	return &WorkspaceSandboxClient{
		httpClient: &http.Client{Timeout: timeout},
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
	FilePath     string `json:"filePath"`
	RelativePath string `json:"relativePath"`
	ChangeType   string `json:"changeType"`
	SandboxID    string `json:"sandboxId"`
	SandboxOwner string `json:"sandboxOwner"`
	Status       string `json:"status"`
}

func (c *WorkspaceSandboxClient) GetCommitPreview(ctx context.Context, projectRoot string) (*workspaceSandboxCommitPreview, error) {
	params := ""
	if projectRoot != "" {
		params = "?projectRoot=" + url.QueryEscape(projectRoot)
	}
	return c.doRequest(ctx, http.MethodGet, "/api/v1/commit-preview"+params, nil)
}

func (c *WorkspaceSandboxClient) GetCommitPreviewForPaths(ctx context.Context, projectRoot string, paths []string) (*workspaceSandboxCommitPreview, error) {
	req := workspaceSandboxCommitPreviewRequest{
		ProjectRoot: projectRoot,
		FilePaths:   paths,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	return c.doRequest(ctx, http.MethodPost, "/api/v1/commit-preview", body)
}

func (c *WorkspaceSandboxClient) doRequest(ctx context.Context, method, path string, body []byte) (*workspaceSandboxCommitPreview, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	baseURL, err := c.resolveBaseURL(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("workspace-sandbox request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseWorkspaceSandboxError(resp)
	}

	var preview workspaceSandboxCommitPreview
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &preview, nil
}

func (c *WorkspaceSandboxClient) resolveBaseURL(ctx context.Context) (string, error) {
	url, err := discovery.ResolveScenarioURLDefault(ctx, "workspace-sandbox")
	if err != nil {
		return "", fmt.Errorf("resolve workspace-sandbox url: %w", err)
	}
	return url, nil
}

func parseWorkspaceSandboxError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err == nil {
		if errResp.Error != "" {
			return fmt.Errorf("workspace-sandbox error: %s", errResp.Error)
		}
		if errResp.Message != "" {
			return fmt.Errorf("workspace-sandbox error: %s", errResp.Message)
		}
	}
	return fmt.Errorf("workspace-sandbox error: status %d, body: %s", resp.StatusCode, string(body))
}
