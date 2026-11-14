package journeys

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

func (c *HTTPDeviceClient) Unlock(ctx context.Context, lease Lease, profileID string) error {
	var response struct {
		Outcome string `json:"outcome"`
	}
	err := c.post(ctx, "/api/v1/auth/unlock", map[string]string{
		"profile_id":  profileID,
		"device_id":   lease.DeviceID,
		"actor":       c.Actor,
		"lease_token": lease.Token,
	}, &response)
	if err != nil {
		return err
	}
	if response.Outcome != "unlocked" && response.Outcome != "already_unlocked" {
		return fmt.Errorf("device-control auth unlock outcome %q", response.Outcome)
	}
	return nil
}

type LogCaptureArtifact struct {
	Reference deliveryramp.EvidenceReference
	Lines     []string
}

type reviewRecordingResponse struct {
	Reference struct {
		ID, Kind, SHA256, Checksum string
		RedactionVerified          bool `json:"redaction_verified"`
	} `json:"reference"`
	Path string `json:"path"`
}

// ClockSample is a host/device wall-clock calibration captured through the
// device-control lease. The device timestamp is read from the target, while
// HostTime is the midpoint of the request round trip.
type ClockSample struct {
	HostTime      time.Time
	DeviceTime    time.Time
	OffsetMs      int64
	UncertaintyMs int64
	Evidence      deliveryramp.EvidenceReference
}

func (c *HTTPDeviceClient) SampleClock(ctx context.Context, lease Lease) (ClockSample, error) {
	started := time.Now().UTC()
	result, err := c.Execute(ctx, lease, "clock-sample", map[string]string{"step_id": "clock-sample"})
	ended := time.Now().UTC()
	if err != nil {
		return ClockSample{}, err
	}
	if len(result.Evidence) == 0 || result.Evidence[0].ID == "" {
		return ClockSample{}, fmt.Errorf("device-control clock sample omitted evidence")
	}
	data, err := c.getArtifact(ctx, result.Evidence[0].ID)
	if err != nil {
		return ClockSample{}, err
	}
	seconds, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return ClockSample{}, fmt.Errorf("parse device clock %q: %w", strings.TrimSpace(string(data)), err)
	}
	deviceSeconds := int64(seconds)
	deviceNanos := int64((seconds - float64(deviceSeconds)) * 1e9)
	deviceTime := time.Unix(deviceSeconds, deviceNanos).UTC()
	hostTime := started.Add(ended.Sub(started) / 2)
	return ClockSample{
		HostTime:      hostTime,
		DeviceTime:    deviceTime,
		OffsetMs:      hostTime.Sub(deviceTime).Milliseconds(),
		UncertaintyMs: ended.Sub(started).Milliseconds() / 2,
		Evidence:      result.Evidence[0],
	}, nil
}

func (c *HTTPDeviceClient) StartLogCapture(ctx context.Context, lease Lease) error {
	_, err := c.Execute(ctx, lease, "logcat-start", map[string]string{"step_id": "logcat-start"})
	return err
}

func (c *HTTPDeviceClient) StopLogCapture(ctx context.Context, lease Lease) (LogCaptureArtifact, error) {
	result, err := c.Execute(ctx, lease, "logcat-stop", map[string]string{"step_id": "logcat-stop"})
	if err != nil {
		return LogCaptureArtifact{}, err
	}
	if len(result.Evidence) == 0 || result.Evidence[0].ID == "" {
		return LogCaptureArtifact{}, fmt.Errorf("device-control logcat stop omitted evidence")
	}
	ref := result.Evidence[0]
	data, err := c.getArtifact(ctx, ref.ID)
	if err != nil {
		return LogCaptureArtifact{}, err
	}
	return LogCaptureArtifact{Reference: ref, Lines: strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")}, nil
}

func recordingKey(lease Lease, chapterID string) string {
	return lease.ID + "\x00" + chapterID
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
		if key == "timeout_ms" {
			timeoutMS, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
			if err != nil || timeoutMS <= 0 {
				return ActionResult{}, fmt.Errorf("invalid device-control timeout_ms %q", value)
			}
			step["timeout_ms"] = timeoutMS
			continue
		}
		switch key {
		case "step_id", "target", "chapter_id", "required_capabilities", "reference":
			continue
		}
		stepArgs[key] = value
	}
	var response struct {
		Disposition      string `json:"disposition"`
		DisconnectReason string `json:"disconnect_reason"`
		Chapters         []struct {
			Disposition          string                   `json:"disposition"`
			Message              string                   `json:"message"`
			ArtifactInstall      *artifactInstallResponse `json:"artifactInstall,omitempty"`
			ArtifactInstallSnake *artifactInstallResponse `json:"artifact_install,omitempty"`
		} `json:"chapters"`
		Evidence []struct {
			ID, Kind, SHA256, Checksum string
			RedactionVerified          bool `json:"redaction_verified"`
		} `json:"evidence"`
	}
	transport := strings.TrimSpace(c.DeviceTransport)
	if transport == "" {
		transport = "usb"
	}
	if strings.EqualFold(action, "network") && strings.EqualFold(transport, "wireless") {
		// A network-control step can disable the Wi-Fi radio that carries the
		// selected wireless ADB session. Use the USB route for the mutation so
		// the owner can still restore connectivity and finish the chapter.
		transport = "usb"
	}
	err := c.post(ctx, "/api/v1/flows/run", map[string]any{
		"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token,
		"confirmed": action == "install-artifact",
		"flow":      map[string]any{"id": "android-ramp-" + arguments["step_id"], "name": "scenario-to-android", "transport": transport, "steps": []any{step}},
	}, &response)
	if err != nil {
		return ActionResult{}, err
	}
	result := ActionResult{}
	for _, chapter := range response.Chapters {
		proof := chapter.ArtifactInstall
		if proof == nil {
			proof = chapter.ArtifactInstallSnake
		}
		if proof != nil {
			result.ArtifactSigningCertificateSHA256 = strings.TrimSpace(proof.signingCertificateSHA256())
		}
	}
	for _, item := range response.Evidence {
		checksum := item.Checksum
		if checksum == "" {
			checksum = item.SHA256
		}
		result.Evidence = append(result.Evidence, deliveryramp.EvidenceReference{ID: item.ID, Kind: item.Kind, Checksum: "sha256:" + strings.TrimPrefix(checksum, "sha256:"), Redacted: item.RedactionVerified})
	}
	if response.Disposition != "passed" {
		reason := strings.TrimSpace(response.DisconnectReason)
		for _, chapter := range response.Chapters {
			if strings.TrimSpace(chapter.Message) == "" || chapter.Disposition == "passed" {
				continue
			}
			reason = strings.TrimSpace(chapter.Message)
			break
		}
		if reason != "" {
			return result, fmt.Errorf("device-control action %q disposition=%q: %s", action, response.Disposition, reason)
		}
		return result, fmt.Errorf("device-control action %q disposition=%q", action, response.Disposition)
	}
	return result, nil
}

type artifactInstallResponse struct {
	SigningCertificateSHA256Camel string `json:"signingCertificateSha256,omitempty"`
	SigningCertificateSHA256Snake string `json:"signing_certificate_sha256,omitempty"`
}

func (r artifactInstallResponse) signingCertificateSHA256() string {
	if value := strings.TrimSpace(r.SigningCertificateSHA256Camel); value != "" {
		return value
	}
	return strings.TrimSpace(r.SigningCertificateSHA256Snake)
}

func (c *HTTPDeviceClient) StartRecording(ctx context.Context, lease Lease) error {
	return c.startRecording(ctx, lease, "")
}

func (c *HTTPDeviceClient) StartChapterRecording(ctx context.Context, lease Lease, chapterID string) error {
	return c.startRecording(ctx, lease, chapterID)
}

func (c *HTTPDeviceClient) startRecording(ctx context.Context, lease Lease, key string) error {
	var response struct {
		Handle struct {
			ID string `json:"id"`
		} `json:"handle"`
	}
	request := map[string]string{"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token}
	if strings.EqualFold(key, "offline_transition") {
		request["transport"] = "usb"
	}
	if err := c.post(ctx, "/api/v1/recordings/start", request, &response); err != nil {
		return err
	}
	if response.Handle.ID == "" {
		return fmt.Errorf("device-control recording start omitted handle")
	}
	if c.recordings == nil {
		c.recordings = map[string]string{}
	}
	c.recordings[recordingKey(lease, key)] = response.Handle.ID
	return nil
}

func (c *HTTPDeviceClient) StopRecording(ctx context.Context, lease Lease) (RecordingArtifact, error) {
	return c.stopRecording(ctx, lease, "")
}

func (c *HTTPDeviceClient) StopChapterRecording(ctx context.Context, lease Lease, chapterID string) (RecordingArtifact, error) {
	return c.stopRecording(ctx, lease, chapterID)
}

func (c *HTTPDeviceClient) FinalizeReviewRecording(ctx context.Context, lease Lease, chapters []deliveryramp.EvidenceReference) (ReviewRecording, error) {
	ids := make([]string, 0, len(chapters))
	for _, chapter := range chapters {
		if strings.TrimSpace(chapter.ID) == "" {
			return ReviewRecording{}, fmt.Errorf("review recording chapter reference omitted id")
		}
		ids = append(ids, chapter.ID)
	}
	var response reviewRecordingResponse
	if err := c.post(ctx, "/api/v1/recordings/concat", map[string]any{"device_id": lease.DeviceID, "actor": c.actor(""), "lease_token": lease.Token, "reference_ids": ids}, &response); err != nil {
		return ReviewRecording{}, err
	}
	checksum := response.Reference.Checksum
	if checksum == "" {
		checksum = response.Reference.SHA256
	}
	if response.Reference.ID == "" || checksum == "" {
		return ReviewRecording{}, fmt.Errorf("device-control review recording omitted identity")
	}
	return ReviewRecording{Reference: deliveryramp.EvidenceReference{ID: response.Reference.ID, Kind: response.Reference.Kind, Checksum: "sha256:" + strings.TrimPrefix(checksum, "sha256:"), Redacted: response.Reference.RedactionVerified}, Path: response.Path}, nil
}

func (c *HTTPDeviceClient) stopRecording(ctx context.Context, lease Lease, key string) (RecordingArtifact, error) {
	recordingKey := recordingKey(lease, key)
	handle := c.recordings[recordingKey]
	if handle == "" {
		return RecordingArtifact{}, fmt.Errorf("no active recording for lease %q", lease.ID)
	}
	delete(c.recordings, recordingKey)
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
	return c.AttachWebViewWithTransport(ctx, lease, packageName, "")
}

func (c *HTTPDeviceClient) AttachWebViewWithTransport(ctx context.Context, lease Lease, packageName, transport string) (WebViewAttachment, error) {
	if strings.TrimSpace(transport) == "" {
		transport = strings.TrimSpace(c.DeviceTransport)
	}
	requestCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	var response struct {
		Endpoint struct {
			CDPEndpoint string `json:"cdp_endpoint"`
			RendererID  string `json:"renderer_id"`
			RendererURL string `json:"renderer_url"`
		} `json:"endpoint"`
	}
	request := map[string]string{"actor": c.actor(""), "lease_token": lease.Token, "package": packageName}
	if strings.TrimSpace(transport) != "" {
		request["transport"] = transport
	}
	if err := c.post(requestCtx, "/api/v1/devices/"+urlPathSegment(lease.DeviceID)+"/webview/attach", request, &response); err != nil {
		return WebViewAttachment{}, err
	}
	if strings.TrimSpace(response.Endpoint.CDPEndpoint) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach omitted cdp endpoint")
	}
	if strings.TrimSpace(response.Endpoint.RendererID) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach omitted renderer id")
	}
	if strings.TrimSpace(response.Endpoint.RendererURL) == "" {
		return WebViewAttachment{}, fmt.Errorf("device-control WebView attach omitted renderer url")
	}
	return WebViewAttachment{CDPEndpoint: response.Endpoint.CDPEndpoint, RendererID: response.Endpoint.RendererID, RendererURL: response.Endpoint.RendererURL}, nil
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

func (c *HTTPDeviceClient) ReadState(ctx context.Context, deviceID string) (DeviceStateSnapshot, error) {
	var state DeviceStateSnapshot
	path := "/api/v1/devices/" + urlPathSegment(deviceID) + "/state"
	if err := c.get(ctx, path, &state); err != nil {
		return DeviceStateSnapshot{}, err
	}
	return state, nil
}

func (c *HTTPDeviceClient) ReadPackageState(ctx context.Context, lease Lease, packageName string) (InstalledPackageState, error) {
	var response struct {
		Result struct {
			Operation   string `json:"operation"`
			Package     string `json:"package,omitempty"`
			Status      string `json:"status"`
			Installed   bool   `json:"installed"`
			Version     string `json:"version,omitempty"`
			VersionCode int64  `json:"version_code,omitempty"`
			Reason      string `json:"reason,omitempty"`
		} `json:"result"`
	}
	payload := map[string]any{
		"actor": c.actor(""), "lease_token": lease.Token, "confirmed": false,
		"operation": "package-state", "package": packageName,
	}
	path := "/api/v1/devices/" + urlPathSegment(lease.DeviceID) + "/apps"
	if err := c.post(ctx, path, payload, &response); err != nil {
		return InstalledPackageState{}, err
	}
	if response.Result.Operation != "package-state" || response.Result.Package != packageName || response.Result.Status != "available" {
		return InstalledPackageState{}, fmt.Errorf("Device Control package-state readback did not verify %s: %s", packageName, response.Result.Reason)
	}
	return InstalledPackageState{Installed: response.Result.Installed, Version: response.Result.Version, VersionCode: response.Result.VersionCode}, nil
}

func (c *HTTPDeviceClient) ReadNetworkState(ctx context.Context, lease Lease) (NetworkStateSnapshot, error) {
	var response struct {
		Result struct {
			Operation    string               `json:"operation"`
			Status       string               `json:"status"`
			Reason       string               `json:"reason,omitempty"`
			NetworkState NetworkStateSnapshot `json:"network_state"`
		} `json:"result"`
	}
	payload := map[string]any{
		"actor": c.actor(""), "lease_token": lease.Token, "confirmed": false,
		"operation": "network-state",
	}
	path := "/api/v1/devices/" + urlPathSegment(lease.DeviceID) + "/apps"
	if err := c.post(ctx, path, payload, &response); err != nil {
		return NetworkStateSnapshot{}, err
	}
	state := response.Result.NetworkState
	if response.Result.Operation != "network-state" || response.Result.Status != "available" || state.WifiEnabled == nil || state.MobileDataEnabled == nil || state.AirplaneModeEnabled == nil {
		return NetworkStateSnapshot{}, fmt.Errorf("Device Control network-state readback is incomplete: %s", response.Result.Reason)
	}
	return state, nil
}

func (c *HTTPDeviceClient) ReadPackageSigningState(ctx context.Context, lease Lease, packageName string) (string, error) {
	var response struct {
		Result struct {
			Operation                string `json:"operation"`
			Package                  string `json:"package,omitempty"`
			Status                   string `json:"status"`
			Installed                bool   `json:"installed"`
			SigningCertificateSHA256 string `json:"signing_certificate_sha256,omitempty"`
			Reason                   string `json:"reason,omitempty"`
		} `json:"result"`
	}
	payload := map[string]any{
		"actor": c.actor(""), "lease_token": lease.Token, "confirmed": false,
		"operation": "package-signing-state", "package": packageName,
	}
	path := "/api/v1/devices/" + urlPathSegment(lease.DeviceID) + "/apps"
	if err := c.post(ctx, path, payload, &response); err != nil {
		return "", err
	}
	if response.Result.Operation != "package-signing-state" || response.Result.Package != packageName || response.Result.Status != "available" || !response.Result.Installed || strings.TrimSpace(response.Result.SigningCertificateSHA256) == "" {
		return "", fmt.Errorf("Device Control package-signing-state did not verify %s: %s", packageName, response.Result.Reason)
	}
	return strings.TrimSpace(response.Result.SigningCertificateSHA256), nil
}

func (c *HTTPDeviceClient) get(ctx context.Context, path string, result any) error {
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("device-control URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+path, nil)
	if err != nil {
		return err
	}
	response, err := c.ClientOrDefault().Do(req)
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

func (c *HTTPDeviceClient) getArtifact(ctx context.Context, id string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(c.BaseURL, "/")+"/api/v1/evidence/"+urlPathSegment(id), nil)
	if err != nil {
		return nil, err
	}
	response, err := c.ClientOrDefault().Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("device-control evidence request returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return io.ReadAll(response.Body)
}

func (c *HTTPDeviceClient) ClientOrDefault() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return http.DefaultClient
}

var _ DeviceClient = (*HTTPDeviceClient)(nil)
