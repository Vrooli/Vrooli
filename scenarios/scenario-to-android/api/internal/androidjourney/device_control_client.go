package androidjourney

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// HTTPDeviceClient is the production device-control adapter. It has no adb
// knowledge: every device verb, lease check, WebView attach, and recording
// operation crosses device-control's lease-scoped HTTP boundary.
type HTTPDeviceClient struct {
	BaseURL         string
	Actor           string
	DeviceTransport string
	Client          *http.Client
	recordings      map[string]string
}

func (c *HTTPDeviceClient) Acquire(ctx context.Context, deviceID, actor string, ttl time.Duration) (Lease, error) {
	var response struct {
		Session struct {
			ID        string    `json:"id"`
			DeviceID  string    `json:"device_id"`
			Token     string    `json:"lease_token"`
			ExpiresAt time.Time `json:"expires_at"`
		} `json:"session"`
	}
	err := c.post(ctx, "/api/v1/sessions/acquire", map[string]any{"device_id": deviceID, "actor": c.actor(actor), "ttl_seconds": int(ttl.Seconds())}, &response)
	if err != nil {
		return Lease{}, err
	}
	if response.Session.ID == "" || response.Session.Token == "" {
		return Lease{}, fmt.Errorf("device-control acquire response omitted lease identity")
	}
	if response.Session.DeviceID == "" {
		response.Session.DeviceID = deviceID
	}
	return Lease{ID: response.Session.ID, DeviceID: response.Session.DeviceID, Token: response.Session.Token, ExpiresAt: response.Session.ExpiresAt}, nil
}

func (c *HTTPDeviceClient) ValidateLease(ctx context.Context, lease Lease) error {
	return c.post(ctx, "/api/v1/sessions/"+urlPathSegment(lease.ID)+"/validate", map[string]string{"device_id": lease.DeviceID, "lease_token": lease.Token}, nil)
}

func (c *HTTPDeviceClient) Execute(ctx context.Context, lease Lease, action string, arguments map[string]string) (ActionResult, error) {
	step := map[string]any{"id": arguments["step_id"], "kind": action, "target": arguments["target"], "arguments": map[string]any{}}
	stepArgs := step["arguments"].(map[string]any)
	for key, value := range arguments {
		if key != "step_id" && key != "target" {
			stepArgs[key] = value
		}
	}
	var response struct {
		Disposition string `json:"disposition"`
		Evidence    []struct {
			ID, Kind, SHA256, Checksum string
			RedactionVerified          bool `json:"redaction_verified"`
		} `json:"evidence"`
	}
	transport := strings.TrimSpace(c.DeviceTransport)
	if transport == "" {
		transport = "usb"
	}
	err := c.post(ctx, "/api/v1/flows/run", map[string]any{
		"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token,
		"flow": map[string]any{"id": "android-ramp-" + arguments["step_id"], "name": "scenario-to-android", "transport": transport, "steps": []any{step}},
	}, &response)
	if err != nil {
		return ActionResult{}, err
	}
	if response.Disposition != "passed" {
		return ActionResult{}, fmt.Errorf("device-control action %q disposition=%q", action, response.Disposition)
	}
	result := ActionResult{}
	for _, item := range response.Evidence {
		checksum := item.Checksum
		if checksum == "" {
			checksum = item.SHA256
		}
		result.Evidence = append(result.Evidence, deliveryramp.EvidenceReference{ID: item.ID, Kind: item.Kind, Checksum: "sha256:" + strings.TrimPrefix(checksum, "sha256:"), Redacted: item.RedactionVerified})
	}
	return result, nil
}

func (c *HTTPDeviceClient) StartRecording(ctx context.Context, lease Lease) error {
	var response struct {
		Handle struct {
			ID string `json:"id"`
		} `json:"handle"`
	}
	if err := c.post(ctx, "/api/v1/recordings/start", map[string]string{"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token}, &response); err != nil {
		return err
	}
	if response.Handle.ID == "" {
		return fmt.Errorf("device-control recording start omitted handle")
	}
	if c.recordings == nil {
		c.recordings = map[string]string{}
	}
	c.recordings[lease.ID] = response.Handle.ID
	return nil
}

func (c *HTTPDeviceClient) StopRecording(ctx context.Context, lease Lease) (RecordingArtifact, error) {
	handle := c.recordings[lease.ID]
	if handle == "" {
		return RecordingArtifact{}, fmt.Errorf("no active recording for lease %q", lease.ID)
	}
	delete(c.recordings, lease.ID)
	var response struct {
		Reference struct {
			ID, Kind, SHA256, Checksum string
			RedactionVerified          bool `json:"redaction_verified"`
		} `json:"reference"`
		StartOffset int64 `json:"start_offset_ms"`
		EndOffset   int64 `json:"end_offset_ms"`
	}
	if err := c.post(ctx, "/api/v1/recordings/stop", map[string]string{"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token, "handle_id": handle}, &response); err != nil {
		return RecordingArtifact{}, err
	}
	checksum := response.Reference.Checksum
	if checksum == "" {
		checksum = response.Reference.SHA256
	}
	return RecordingArtifact{Reference: deliveryramp.EvidenceReference{ID: response.Reference.ID, Kind: response.Reference.Kind, Checksum: "sha256:" + strings.TrimPrefix(checksum, "sha256:"), Redacted: response.Reference.RedactionVerified}, StartMs: response.StartOffset, EndMs: response.EndOffset, HasOffsets: response.EndOffset >= response.StartOffset}, nil
}

func (c *HTTPDeviceClient) Release(ctx context.Context, lease Lease) error {
	return c.post(ctx, "/api/v1/sessions/"+urlPathSegment(lease.ID)+"/release", nil, nil)
}

func (c *HTTPDeviceClient) AttachWebView(ctx context.Context, lease Lease, packageName string) (WebViewAttachment, error) {
	var response struct {
		Endpoint struct {
			CDPEndpoint string `json:"cdp_endpoint"`
			RendererID  string `json:"renderer_id"`
		} `json:"endpoint"`
	}
	if err := c.post(ctx, "/api/v1/devices/"+urlPathSegment(lease.DeviceID)+"/webview/attach", map[string]string{"actor": c.actor(""), "lease_token": lease.Token, "package": packageName}, &response); err != nil {
		return WebViewAttachment{}, err
	}
	if strings.TrimSpace(response.Endpoint.CDPEndpoint) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach omitted cdp endpoint")
	}
	if strings.TrimSpace(response.Endpoint.RendererID) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach omitted renderer id")
	}
	return WebViewAttachment{CDPEndpoint: response.Endpoint.CDPEndpoint, RendererID: response.Endpoint.RendererID}, nil
}

func (c *HTTPDeviceClient) actor(actor string) string {
	if strings.TrimSpace(actor) != "" {
		return actor
	}
	if strings.TrimSpace(c.Actor) != "" {
		return c.Actor
	}
	return "scenario-to-android"
}

func (c *HTTPDeviceClient) post(ctx context.Context, path string, payload any, result any) error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("device-control URL is not configured")
	}
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := c.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(response.Body, 16<<10))
		return fmt.Errorf("device-control %s returned %s: %s", path, response.Status, strings.TrimSpace(string(data)))
	}
	if result != nil {
		if err := json.NewDecoder(response.Body).Decode(result); err != nil {
			return fmt.Errorf("decode device-control %s: %w", path, err)
		}
	}
	return nil
}

var _ DeviceClient = (*HTTPDeviceClient)(nil)
