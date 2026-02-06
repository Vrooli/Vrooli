// Package deploy provides an HTTP client for deploying desktop artifacts through LPBS.
//
// The client discovers the local LPBS instance via api-core/discovery, authenticates
// with a service bearer token, and orchestrates the artifact upload flow
// (presign → S3 upload → commit → apply) through LPBS's remote profile proxy.
package deploy

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
)

// BaseURLResolver abstracts URL resolution for testability.
type BaseURLResolver func(ctx context.Context) (string, error)

// HTTPDoer abstracts HTTP operations for testability.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// LPBSClient calls LPBS admin endpoints to deploy desktop artifacts.
type LPBSClient struct {
	resolver   BaseURLResolver
	httpClient HTTPDoer
	token      string
}

// NewLPBSClient creates a client that discovers LPBS via the given scenario name
// and authenticates with the provided service token.
func NewLPBSClient(scenarioName, serviceToken string) *LPBSClient {
	return &LPBSClient{
		resolver: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, scenarioName)
		},
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		token:      serviceToken,
	}
}

// NewLPBSClientWithResolver creates a client with custom resolver and HTTP client (for testing).
func NewLPBSClientWithResolver(resolver BaseURLResolver, httpClient HTTPDoer, token string) *LPBSClient {
	return &LPBSClient{
		resolver:   resolver,
		httpClient: httpClient,
		token:      token,
	}
}

// RemoteProfile represents an LPBS remote profile.
type RemoteProfile struct {
	ID      int64  `json:"id"`
	Tag     string `json:"tag"`
	APIBase string `json:"api_base"`
	Status  string `json:"status"`
}

// UploadRequest describes one artifact to upload.
type UploadRequest struct {
	RemoteProfile  string
	AppKey         string
	Platform       string
	FilePath       string
	ReleaseVersion string
	ReleaseNotes   string
}

// UploadResult is the outcome of a single artifact upload.
type UploadResult struct {
	ArtifactID int64  `json:"artifact_id"`
	Platform   string `json:"platform"`
}

// ListRemoteProfiles returns all remote profiles from the local LPBS.
func (c *LPBSClient) ListRemoteProfiles(ctx context.Context) ([]RemoteProfile, error) {
	body, err := c.adminRequest(ctx, "GET", "/api/v1/admin/remote-profiles", nil)
	if err != nil {
		return nil, fmt.Errorf("list remote profiles: %w", err)
	}
	var profiles []RemoteProfile
	if err := json.Unmarshal(body, &profiles); err != nil {
		return nil, fmt.Errorf("decode remote profiles: %w", err)
	}
	return profiles, nil
}

// TestRemoteProfile validates that a remote profile has an active session.
func (c *LPBSClient) TestRemoteProfile(ctx context.Context, profileTag string) error {
	profileID, err := c.resolveProfileID(ctx, profileTag)
	if err != nil {
		return err
	}
	_, err = c.adminRequest(ctx, "POST",
		"/api/v1/admin/remote-profiles/"+url.PathEscape(profileID)+"/test", nil)
	if err != nil {
		return fmt.Errorf("test remote profile %q: %w", profileTag, err)
	}
	return nil
}

// ProxyRequest forwards an admin API call through a remote profile.
func (c *LPBSClient) ProxyRequest(ctx context.Context, profileTag, method, path string, payload interface{}) ([]byte, error) {
	profileID, err := c.resolveProfileID(ctx, profileTag)
	if err != nil {
		return nil, err
	}

	proxyPayload := map[string]interface{}{
		"method": method,
		"path":   path,
	}
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode proxy body: %w", err)
		}
		headers := map[string]string{"Content-Type": "application/json"}
		proxyPayload["headers"] = headers
		proxyPayload["body"] = json.RawMessage(payloadBytes)
	}

	return c.adminRequest(ctx, "POST",
		"/api/v1/admin/remote-profiles/"+url.PathEscape(profileID)+"/proxy",
		proxyPayload)
}

// UploadArtifact orchestrates the full deploy flow for one artifact:
//  1. Proxy → presign-upload → get S3 URL
//  2. HTTP PUT to S3 presigned URL (direct, no proxy)
//  3. Proxy → commit → register artifact
//  4. Proxy → apply → link to download asset
func (c *LPBSClient) UploadArtifact(ctx context.Context, req *UploadRequest) (*UploadResult, error) {
	// Open and hash the file
	f, err := os.Open(req.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open artifact: %w", err)
	}
	defer f.Close()

	hasher := sha512.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return nil, fmt.Errorf("hash artifact: %w", err)
	}
	sha512Hex := hex.EncodeToString(hasher.Sum(nil))

	// Rewind for upload
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind artifact: %w", err)
	}

	filename := filepath.Base(req.FilePath)
	contentType := inferContentType(filename)

	// Step 1: Presign upload
	presignResp, err := c.proxyPresign(ctx, req, filename, contentType)
	if err != nil {
		return nil, err
	}

	// Step 2: Direct S3 PUT
	if err := c.uploadToS3(ctx, presignResp, f, contentType); err != nil {
		return nil, err
	}

	// Step 3: Commit
	artifactID, err := c.proxyCommit(ctx, req, presignResp, filename, contentType, sha512Hex)
	if err != nil {
		return nil, err
	}

	// Step 4: Apply
	if err := c.proxyApply(ctx, req, artifactID); err != nil {
		return nil, err
	}

	return &UploadResult{
		ArtifactID: artifactID,
		Platform:   req.Platform,
	}, nil
}

// DeriveUpdateURL returns the update URL for an app from the remote profile's API base.
func (c *LPBSClient) DeriveUpdateURL(ctx context.Context, profileTag, appKey string) (string, error) {
	profiles, err := c.ListRemoteProfiles(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range profiles {
		if p.Tag == profileTag {
			base := strings.TrimRight(p.APIBase, "/")
			return base + "/updates/" + appKey, nil
		}
	}
	return "", fmt.Errorf("remote profile %q not found", profileTag)
}

// --- internal helpers ---

type presignResponse struct {
	UploadURL       string            `json:"upload_url"`
	RequiredHeaders map[string]string `json:"required_headers"`
	Bucket          string            `json:"bucket"`
	ObjectKey       string            `json:"object_key"`
}

func (c *LPBSClient) proxyPresign(ctx context.Context, req *UploadRequest, filename, contentType string) (*presignResponse, error) {
	body, err := c.ProxyRequest(ctx, req.RemoteProfile, "POST",
		"/admin/download-artifacts/presign-upload",
		map[string]interface{}{
			"filename":        filename,
			"content_type":    contentType,
			"app_key":         req.AppKey,
			"platform":        req.Platform,
			"release_version": req.ReleaseVersion,
		})
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}
	var resp presignResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode presign response: %w", err)
	}
	return &resp, nil
}

func (c *LPBSClient) uploadToS3(ctx context.Context, presign *presignResponse, body io.Reader, contentType string) error {
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", presign.UploadURL, body)
	if err != nil {
		return fmt.Errorf("create S3 request: %w", err)
	}
	for key, value := range presign.RequiredHeaders {
		if strings.EqualFold(key, "host") || strings.TrimSpace(value) == "" {
			continue
		}
		httpReq.Header.Set(key, value)
	}
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("S3 upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("S3 upload failed (status %d): %s", resp.StatusCode, string(errBody))
	}
	return nil
}

func (c *LPBSClient) proxyCommit(ctx context.Context, req *UploadRequest, presign *presignResponse, filename, contentType, sha512Hex string) (int64, error) {
	body, err := c.ProxyRequest(ctx, req.RemoteProfile, "POST",
		"/admin/download-artifacts/commit",
		map[string]interface{}{
			"bucket":            presign.Bucket,
			"object_key":        presign.ObjectKey,
			"original_filename": filename,
			"content_type":      contentType,
			"app_key":           req.AppKey,
			"platform":          req.Platform,
			"release_version":   req.ReleaseVersion,
			"sha512":            sha512Hex,
		})
	if err != nil {
		return 0, fmt.Errorf("commit artifact: %w", err)
	}
	var resp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, fmt.Errorf("decode commit response: %w", err)
	}
	return resp.ID, nil
}

func (c *LPBSClient) proxyApply(ctx context.Context, req *UploadRequest, artifactID int64) error {
	payload := map[string]interface{}{
		"app_key":         req.AppKey,
		"platform":        req.Platform,
		"artifact_id":     artifactID,
		"release_version": req.ReleaseVersion,
	}
	if strings.TrimSpace(req.ReleaseNotes) != "" {
		payload["release_notes"] = strings.TrimSpace(req.ReleaseNotes)
	}

	_, err := c.ProxyRequest(ctx, req.RemoteProfile, "POST",
		"/admin/download-assets/apply", payload)
	if err != nil {
		return fmt.Errorf("apply asset: %w", err)
	}
	return nil
}

// adminRequest makes an authenticated request to the local LPBS instance.
func (c *LPBSClient) adminRequest(ctx context.Context, method, path string, payload interface{}) ([]byte, error) {
	baseURL, err := c.resolver(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve LPBS URL: %w", err)
	}

	fullURL := strings.TrimRight(baseURL, "/") + path

	var bodyReader io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("encode request: %w", err)
		}
		bodyReader = strings.NewReader(string(payloadBytes))
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// resolveProfileID looks up a remote profile by tag and returns its ID as a string.
func (c *LPBSClient) resolveProfileID(ctx context.Context, profileTag string) (string, error) {
	profiles, err := c.ListRemoteProfiles(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range profiles {
		if p.Tag == profileTag {
			return fmt.Sprintf("%d", p.ID), nil
		}
	}
	return "", fmt.Errorf("remote profile %q not found", profileTag)
}

// inferContentType returns a MIME type based on the file extension.
func inferContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".exe", ".msi":
		return "application/octet-stream"
	case ".dmg":
		return "application/x-apple-diskimage"
	case ".zip":
		return "application/zip"
	case ".tar.gz", ".tgz":
		return "application/gzip"
	case ".deb":
		return "application/vnd.debian.binary-package"
	case ".rpm":
		return "application/x-rpm"
	case ".appimage":
		return "application/octet-stream"
	case ".yml", ".yaml":
		return "text/yaml"
	default:
		return "application/octet-stream"
	}
}
