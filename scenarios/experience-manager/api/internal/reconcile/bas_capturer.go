package reconcile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/servicetoken"
)

type BASCapturer struct {
	Resolve       func(ctx context.Context) (string, error)
	ResolveTarget func(ctx context.Context, target CaptureTarget) (string, error)
	HTTPClient    *http.Client
	Tokens        *servicetoken.Tokens
}

// CaptureAccessibility implements Capturer.
func (c BASCapturer) CaptureAccessibility(ctx context.Context, target CaptureTarget) (Snapshot, error) {
	baseURL, err := c.resolve(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		if err != nil {
			return Snapshot{}, fmt.Errorf("%w: resolve BAS endpoint: %v", ErrCaptureUnavailable, err)
		}
		return Snapshot{}, fmt.Errorf("%w: BAS endpoint is empty", ErrCaptureUnavailable)
	}
	// Preserve the scenario identity instead of pre-resolving localhost. BAS
	// uses this shorthand to obtain the Experience Manager-owned readiness
	// profile and wait for a terminal ExperienceSurface state.
	targetURL := "scenario=" + target.Scenario + ",path=" + firstRoute([]string{target.Route})

	type waitForPayload struct {
		TimeoutMs int `json:"timeout_ms"`
	}
	type dimensionsPayload struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
	}
	type fingerprintPayload struct {
		Locale      string `json:"locale,omitempty"`
		ColorScheme string `json:"colorScheme,omitempty"`
	}
	type browserProfilePayload struct {
		Fingerprint      *fingerprintPayload `json:"fingerprint,omitempty"`
		MotionPreference string              `json:"motionPreference,omitempty"`
		InteractionState string              `json:"interactionState,omitempty"`
	}
	type captureRequestPayload struct {
		URL string `json:"url"`
		// CaptureService is a Connect endpoint, whose canonical JSON field name
		// is lowerCamelCase. The response has always accepted both shapes, but a
		// snake_case request silently loses this optional capture flag.
		InlineAccessibility bool                   `json:"inlineAccessibility"`
		Label               string                 `json:"label"`
		Dimensions          dimensionsPayload      `json:"dimensions,omitempty"`
		WaitFor             *waitForPayload        `json:"wait_for,omitempty"`
		InteractionFlowJSON string                 `json:"interaction_flow_json,omitempty"`
		InlineComputedStyle bool                   `json:"inlineComputedStyle,omitempty"`
		BrowserProfile      *browserProfilePayload `json:"browserProfile,omitempty"`
		InteractionState    string                 `json:"interactionState,omitempty"`
	}
	payload := captureRequestPayload{
		URL:                 targetURL,
		InlineAccessibility: true,
		InlineComputedStyle: true,
		Label:               "experience-manager structure reconciliation",
	}
	// Geometry floors compare node bounds against the viewport, so a running
	// CSS transition makes them nondeterministic: an element captured mid-slide
	// reports an off-viewport x and fails a page that is fine once settled.
	// Reconciliation therefore measures with motion reduced unless the target
	// deliberately declares a preference, which still wins.
	motionPreference := target.MotionPreference
	if motionPreference == "" {
		motionPreference = "reduce"
	}
	payload.BrowserProfile = &browserProfilePayload{
		Fingerprint:      &fingerprintPayload{Locale: target.Locale, ColorScheme: target.ColorScheme},
		MotionPreference: motionPreference,
		InteractionState: target.InteractionState,
	}
	payload.InteractionState = target.InteractionState
	if target.ViewportWidth > 0 && target.ViewportHeight > 0 {
		payload.Dimensions = dimensionsPayload{Width: target.ViewportWidth, Height: target.ViewportHeight}
	}
	// No explicit caller wait: BAS first applies declared semantic readiness
	// and falls back to its compatibility settle delay only when no profile is
	// available. SettleMs remains part of CaptureTarget for legacy callers and
	// state metadata, but it must not mask a declared readiness surface.

	encoded, err := json.Marshal(payload)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: encode BAS capture request: %v", ErrCaptureUnavailable, err)
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/browser_automation_studio.v1.capture.CaptureService/Capture",
		strings.NewReader(string(encoded)),
	)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: create BAS capture request: %v", ErrCaptureUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Tokens == nil {
		return Snapshot{}, fmt.Errorf("%w: BAS capture service identity is unavailable", ErrCaptureUnavailable)
	}
	token, err := c.Tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return Snapshot{}, fmt.Errorf("%w: BAS capture service identity is unavailable", ErrCaptureUnavailable)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.captureHTTPClient().Do(req)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: call BAS CaptureService: %v", ErrCaptureUnavailable, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Snapshot{}, fmt.Errorf("%w: BAS CaptureService returned HTTP %d", ErrCaptureUnavailable, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Snapshot{}, fmt.Errorf("%w: read BAS CaptureService response: %v", ErrCaptureUnavailable, err)
	}
	var decoded struct {
		AccessibilityJSON      string `json:"accessibility_json"`
		AccessibilityJSONCamel string `json:"accessibilityJson"`
		Artifacts              []struct {
			Type      any               `json:"type"`
			Reference string            `json:"reference"`
			SizeBytes int64             `json:"size_bytes"`
			Metadata  map[string]string `json:"metadata"`
		} `json:"artifacts"`
		Readiness struct {
			DurationMS                   json.RawMessage `json:"duration_ms"`
			DurationMSCamel              json.RawMessage `json:"durationMs"`
			NavigationDurationMS         json.RawMessage `json:"navigation_duration_ms"`
			NavigationDurationMSCamel    json.RawMessage `json:"navigationDurationMs"`
			ReadinessWaitDurationMS      json.RawMessage `json:"readiness_wait_duration_ms"`
			ReadinessWaitDurationMSCamel json.RawMessage `json:"readinessWaitDurationMs"`
			SelectedStrategy             string          `json:"selected_strategy"`
			SelectedStrategyCamel        string          `json:"selectedStrategy"`
			Outcome                      string          `json:"outcome"`
		} `json:"readiness"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode BAS CaptureService response: %v", ErrCaptureUnavailable, err)
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		decoded.AccessibilityJSON = decoded.AccessibilityJSONCamel
	}
	if strings.TrimSpace(decoded.AccessibilityJSON) == "" {
		return Snapshot{}, fmt.Errorf("%w: BAS response omitted inline accessibility data", ErrCaptureUnavailable)
	}
	var snapshot Snapshot
	if err := json.Unmarshal([]byte(decoded.AccessibilityJSON), &snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("%w: decode accessibility snapshot: %v", ErrCaptureUnavailable, err)
	}
	if snapshot.Contract != snapshotContract {
		return Snapshot{}, fmt.Errorf("%w: unsupported accessibility snapshot contract %q", ErrCaptureUnavailable, snapshot.Contract)
	}
	snapshot.ScreenshotRef = screenshotRefFromArtifacts(ctx, baseURL, decoded.Artifacts, c.httpClient())
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	snapshot.Timing = CaptureTiming{
		TotalMilliseconds:         firstNonZero(parseCaptureMilliseconds(decoded.Readiness.DurationMS), parseCaptureMilliseconds(decoded.Readiness.DurationMSCamel)),
		NavigationMilliseconds:    firstNonZero(parseCaptureMilliseconds(decoded.Readiness.NavigationDurationMS), parseCaptureMilliseconds(decoded.Readiness.NavigationDurationMSCamel)),
		ReadinessWaitMilliseconds: firstNonZero(parseCaptureMilliseconds(decoded.Readiness.ReadinessWaitDurationMS), parseCaptureMilliseconds(decoded.Readiness.ReadinessWaitDurationMSCamel)),
		Strategy:                  firstNonEmpty(decoded.Readiness.SelectedStrategy, decoded.Readiness.SelectedStrategyCamel),
		Outcome:                   decoded.Readiness.Outcome,
	}
	return snapshot, nil
}

// parseCaptureMilliseconds accepts standard JSON numbers and the quoted int64
// representation emitted by protobuf's canonical JSON mapping.
func parseCaptureMilliseconds(raw json.RawMessage) int64 {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number
	}
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0
	}
	number, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0
	}
	return number
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func screenshotRefFromArtifacts(ctx context.Context, baseURL string, artifacts []struct {
	Type      any               `json:"type"`
	Reference string            `json:"reference"`
	SizeBytes int64             `json:"size_bytes"`
	Metadata  map[string]string `json:"metadata"`
}, client *http.Client,
) string {
	for _, artifact := range artifacts {
		if !isScreenshotArtifactType(artifact.Type) {
			continue
		}
		if artifact.SizeBytes <= 0 || strings.TrimSpace(artifact.Metadata["unavailable"]) != "" || strings.TrimSpace(artifact.Metadata["reason"]) != "" {
			continue
		}
		if ref := dataURLFromArtifact(ctx, baseURL, artifact, client); ref != "" {
			return ref
		}
	}
	return ""
}

func isScreenshotArtifactType(raw any) bool {
	switch value := raw.(type) {
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		return normalized == "capture_type_screenshot" || normalized == "screenshot" || normalized == "1"
	case float64:
		return value == 1
	default:
		return false
	}
}

// dataURLFromArtifact fetches screenshot bytes through BAS's artifact owner.
// CaptureArtifact.path is a BAS-local filesystem path and must never be opened
// by this cross-scenario consumer.
func dataURLFromArtifact(ctx context.Context, baseURL string, artifact struct {
	Type      any               `json:"type"`
	Reference string            `json:"reference"`
	SizeBytes int64             `json:"size_bytes"`
	Metadata  map[string]string `json:"metadata"`
}, client *http.Client,
) string {
	if strings.TrimSpace(artifact.Reference) == "" || artifact.SizeBytes <= 0 || client == nil {
		return ""
	}
	identity, err := url.Parse(artifact.Reference)
	if err != nil || identity.Scheme != "bas-capture" || identity.Host == "" || identity.Path != "/screenshot" || identity.RawQuery != "" || identity.Fragment != "" {
		return ""
	}
	viewPath := strings.TrimSpace(artifact.Metadata["view_url"])
	parsed, err := url.Parse(viewPath)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(parsed.Path, "/api/v1/artifacts/") || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.EscapedPath() != parsed.Path {
		return ""
	}
	pathParts := strings.Split(strings.TrimPrefix(parsed.Path, "/"), "/")
	if len(pathParts) < 6 || pathParts[3] != "artifacts" || pathParts[4] != identity.Host {
		return ""
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return ""
		}
	}
	requestURL := strings.TrimRight(baseURL, "/") + parsed.EscapedPath()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}
	// Screenshot evidence is small relative to the inline AX cap; bound this
	// additional transfer so a corrupt or misconfigured artifact cannot exhaust
	// the reconciliation worker.
	const maxScreenshotBytes = 16 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxScreenshotBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxScreenshotBytes || int64(len(data)) != artifact.SizeBytes {
		return ""
	}
	contentType, _, err := mime.ParseMediaType(artifact.Metadata["content_type"])
	if err != nil || !strings.HasPrefix(contentType, "image/") {
		contentType, _, err = mime.ParseMediaType(resp.Header.Get("Content-Type"))
	}
	if err != nil || !strings.HasPrefix(contentType, "image/") {
		return ""
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (c BASCapturer) resolve(ctx context.Context) (string, error) {
	if c.Resolve != nil {
		return c.Resolve(ctx)
	}
	return discovery.ResolveScenarioURLDefault(ctx, basScenarioID)
}

func (c BASCapturer) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	// Full-fidelity capture evaluates the declared viewport/state matrix and
	// may drive several isolated browser sessions. Keep the default long
	// enough for that matrix to finish; callers can still inject a tighter
	// client for bounded operations and tests.
	return &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
}

// captureHTTPClient prevents a redirected Capture call from carrying the
// service bearer to a second BAS route or another host. Artifact reads use the
// unauthenticated base client separately.
func (c BASCapturer) captureHTTPClient() *http.Client {
	client := *c.httpClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &client
}
