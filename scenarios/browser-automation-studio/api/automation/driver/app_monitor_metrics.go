package driver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const AppMonitorMetricsContract = "app-monitor.preview-workspace.metrics.v1"

// AppMonitorMetricsResponse is the fixed scalar observation owned by the
// Playwright driver. It carries no DOM content, storage values, or selectors.
type AppMonitorMetricsResponse struct {
	Contract              string `json:"contract"`
	SessionID             string `json:"sessionId"`
	PageID                string `json:"pageId"`
	URL                   string `json:"url"`
	ObservedAt            string `json:"observedAt"`
	IframeCount           int    `json:"iframeCount"`
	PaneCount             int    `json:"paneCount"`
	TabCardCount          int    `json:"tabCardCount"`
	ProfilerMarkCount     int    `json:"profilerMarkCount"`
	WorkspaceStorageBytes int    `json:"workspaceStorageBytes"`
}

// GetAppMonitorMetrics requests only the driver's fixed App Monitor metric
// contract. There is no caller-provided URL, selector, script, or output path.
func (c *Client) GetAppMonitorMetrics(ctx context.Context, sessionID, executionID, leaseID string) (*AppMonitorMetricsResponse, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("App Monitor metric read requires a session ID")
	}
	if err := validateSessionOwner(executionID, leaseID); err != nil {
		return nil, err
	}
	var response AppMonitorMetricsResponse
	path := fmt.Sprintf("/session/%s/app-monitor-metrics", url.PathEscape(sessionID))
	if err := c.getWithSessionOwner(ctx, path, executionID, leaseID, &response); err != nil {
		return nil, err
	}
	if response.Contract != AppMonitorMetricsContract || response.SessionID != sessionID || strings.TrimSpace(response.PageID) == "" || response.URL == "" || response.ObservedAt == "" ||
		response.IframeCount < 0 || response.PaneCount < 0 || response.TabCardCount < 0 || response.ProfilerMarkCount < 0 || response.WorkspaceStorageBytes < 0 {
		return nil, errors.New("driver returned an invalid App Monitor metric receipt")
	}
	return &response, nil
}

// AppMonitorTraceArtifact contains the one authorized, sanitized trace result.
// It is intentionally separate from the ordinary metrics response.
type AppMonitorTraceArtifact struct {
	Contract   string `json:"contract"`
	SessionID  string `json:"sessionId"`
	PageID     string `json:"pageId"`
	CaptureID  string `json:"captureId"`
	SHA256     string `json:"sha256"`
	ByteLength int    `json:"byteLength"`
	TraceJSON  string `json:"traceJson"`
}

func (c *Client) BeginAppMonitorCapture(ctx context.Context, sessionID, executionID, leaseID, captureID string) error {
	var response struct {
		Contract  string `json:"contract"`
		SessionID string `json:"sessionId"`
		CaptureID string `json:"captureId"`
		Status    string `json:"status"`
	}
	if err := c.appMonitorCapture(ctx, sessionID, executionID, leaseID, "start", captureID, &response); err != nil {
		return err
	}
	if response.Contract != "app-monitor.preview-workspace.trace.v1" || response.SessionID != sessionID || response.CaptureID != captureID || response.Status != "started" {
		return errors.New("driver returned an invalid App Monitor trace start receipt")
	}
	return nil
}

func (c *Client) FinishAppMonitorCapture(ctx context.Context, sessionID, executionID, leaseID, captureID string) (*AppMonitorTraceArtifact, error) {
	var artifact AppMonitorTraceArtifact
	if err := c.appMonitorCapture(ctx, sessionID, executionID, leaseID, "finish", captureID, &artifact); err != nil {
		return nil, err
	}
	if artifact.Contract != "app-monitor.preview-workspace.trace.v1" || artifact.SessionID != sessionID ||
		artifact.CaptureID != captureID || strings.TrimSpace(artifact.PageID) == "" || artifact.ByteLength < 0 ||
		len(artifact.SHA256) != 64 || !isLowerHex(artifact.SHA256) || len(artifact.TraceJSON) == 0 ||
		len(artifact.TraceJSON) != artifact.ByteLength || sha256Hex([]byte(artifact.TraceJSON)) != artifact.SHA256 {
		return nil, errors.New("driver returned an invalid App Monitor trace receipt")
	}
	return &artifact, nil
}

func (c *Client) appMonitorCapture(ctx context.Context, sessionID, executionID, leaseID, phase, captureID string, response any) error {
	if strings.TrimSpace(sessionID) == "" || !validUUID(captureID) {
		return errors.New("App Monitor capture requires a session and capture ID")
	}
	if err := validateSessionOwner(executionID, leaseID); err != nil {
		return err
	}
	path := fmt.Sprintf("/session/%s/app-monitor-capture/%s", url.PathEscape(sessionID), phase)
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(map[string]string{"capture_id": captureID}); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &body)
	if err != nil {
		return fmt.Errorf("create App Monitor capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setSessionOwnerHeaders(req.Header, executionID, leaseID)
	return c.doRequest(req, response, "POST "+path)
}

func validUUID(value string) bool {
	return len(value) == 36 && value[8] == '-' && value[13] == '-' && value[18] == '-' && value[23] == '-' && isUUIDHex(value)
}

func isUUIDHex(value string) bool {
	for i, r := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}

func isLowerHex(value string) bool {
	for _, r := range value {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return false
		}
	}
	return true
}

func sha256Hex(value []byte) string { return fmt.Sprintf("%x", sha256.Sum256(value)) }
