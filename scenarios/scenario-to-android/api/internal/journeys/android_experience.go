package journeys

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	visualpb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
	visualconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth/visualhealth_v1connect"
)

type AndroidExperienceRequest struct {
	Scenario                string
	Route                   string
	BuildID                 string
	RunID                   string
	StepID                  string
	TargetID                string
	ViewportID              string
	ExpectedKeyboardVisible *bool
	Evidence                []deliveryramp.EvidenceReference
}

type AndroidExperienceResult struct {
	Status           string             `json:"status"`
	Reason           string             `json:"reason,omitempty"`
	AssessmentScope  string             `json:"assessment_scope"`
	AnalyzerVerdict  string             `json:"analyzer_verdict,omitempty"`
	Findings         []string           `json:"findings,omitempty"`
	ObservedAt       time.Time          `json:"observed_at"`
	DeviceEvidence   string             `json:"device_evidence_id,omitempty"`
	ScreenshotSHA256 string             `json:"screenshot_sha256,omitempty"`
	Viewport         experienceViewport `json:"viewport,omitempty"`
	Window           map[string]any     `json:"android_window,omitempty"`
}

type experienceViewport struct {
	ID                string  `json:"id"`
	Width             int     `json:"width"`
	Height            int     `json:"height"`
	DeviceScaleFactor float64 `json:"device_scale_factor"`
}

type androidDeviceStateResponse struct {
	DisplayWidth   int `json:"display_width"`
	DisplayHeight  int `json:"display_height"`
	DisplayDensity int `json:"display_density"`
	Properties     map[string]struct {
		Value  any    `json:"value"`
		Status string `json:"status"`
		Reason string `json:"reason,omitempty"`
	} `json:"properties"`
	Unavailable map[string]string `json:"unavailable"`
}

type AndroidExperienceEvaluator interface {
	AnalyzeAndroidExperience(context.Context, AndroidExperienceRequest) (AndroidExperienceResult, error)
}

// AnalyzeAndroidExperience feeds a lease-owned Device Control frame and the
// matching Android window state into UI Health's existing AnalyzeArtifacts
// contract. The screenshot remains owned by Device Control; this method stores
// no second capture copy or bundle.
func (c *HTTPDeviceClient) AnalyzeAndroidExperience(ctx context.Context, request AndroidExperienceRequest) (AndroidExperienceResult, error) {
	result := AndroidExperienceResult{Status: "not-evaluated", AssessmentScope: "device screenshot plus Android system insets and IME state; no app-overlap verdict", ObservedAt: time.Now().UTC(), DeviceEvidence: "", Findings: []string{}}
	if strings.TrimSpace(request.Scenario) == "" || strings.TrimSpace(request.RunID) == "" || strings.TrimSpace(request.StepID) == "" || strings.TrimSpace(request.TargetID) == "" {
		result.Reason = "scenario, run, step, and Android target identity are required"
		return result, nil
	}
	if len(request.Evidence) != 1 || strings.TrimSpace(request.Evidence[0].ID) == "" || !request.Evidence[0].Redacted {
		result.Reason = "one redacted Device Control screenshot reference is required"
		return result, nil
	}
	ref := request.Evidence[0]
	result.DeviceEvidence = ref.ID
	imageBytes, err := c.getArtifact(ctx, ref.ID)
	if err != nil {
		result.Reason = "Device Control screenshot bytes are unavailable: " + err.Error()
		return result, nil
	}
	digest := sha256.Sum256(imageBytes)
	checksum := hex.EncodeToString(digest[:])
	if checksum != strings.TrimPrefix(ref.Checksum, "sha256:") {
		result.Reason = "Device Control screenshot bytes do not match the evidence checksum"
		return result, nil
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(imageBytes))
	if err != nil || imageConfig.Width <= 0 || imageConfig.Height <= 0 || int64(imageConfig.Width) > (64*1024*1024)/int64(imageConfig.Height) {
		result.Reason = "Device Control screenshot is not a supported PNG frame"
		return result, nil
	}
	result.ScreenshotSHA256 = checksum

	state, err := c.readAndroidDeviceState(ctx, request.TargetID)
	if err != nil {
		result.Reason = "Android window state is unavailable: " + err.Error()
		return result, nil
	}
	required := []string{"android.status_bar.visible", "android.status_bar.height_px", "android.navigation_bar.visible", "android.navigation_bar.height_px", "android.ime.visible", "android.ime.height_px", "android.safe_inset.top_px", "android.safe_inset.bottom_px", "android.safe_inset.left_px", "android.safe_inset.right_px"}
	window := make(map[string]any, len(required))
	for _, name := range required {
		property, ok := state.Properties[name]
		if !ok || property.Status != "available" {
			result.Reason = "Android window metric is not observed: " + name
			if state.Unavailable["android_window_metrics"] != "" {
				result.Reason += " (" + state.Unavailable["android_window_metrics"] + ")"
			}
			return result, nil
		}
		window[name] = property.Value
	}
	keyboardVisible, ok := window["android.ime.visible"].(bool)
	if !ok {
		result.Reason = "Android IME visibility has an invalid value type"
		return result, nil
	}
	keyboardMismatch := false
	if request.ExpectedKeyboardVisible != nil && keyboardVisible != *request.ExpectedKeyboardVisible {
		keyboardMismatch = true
		result.Reason = fmt.Sprintf("IME visibility was %t; expected %t", keyboardVisible, *request.ExpectedKeyboardVisible)
	}
	if state.DisplayWidth <= 0 || state.DisplayHeight <= 0 || state.DisplayDensity <= 0 {
		result.Reason = "Android display dimensions or density are unavailable"
		result.Window = window
		return result, nil
	}

	width, height := imageConfig.Width, imageConfig.Height
	densityScale := 1.0
	if state.DisplayDensity > 0 {
		densityScale = float64(state.DisplayDensity) / 160
	}
	viewportID := strings.TrimSpace(request.ViewportID)
	if viewportID == "" {
		viewportID = "android-device"
	}
	result.Viewport = experienceViewport{ID: viewportID, Width: width, Height: height, DeviceScaleFactor: densityScale}
	layout := map[string]any{
		"schema":   "experience.android-window-evidence/v1",
		"scenario": request.Scenario, "route": request.Route, "run_id": request.RunID,
		"step_id": request.StepID, "target_id": request.TargetID, "build_id": request.BuildID,
		"device_evidence_id": ref.ID, "screenshot_sha256": checksum,
		"observed_at": result.ObservedAt.Format(time.RFC3339Nano), "viewport": result.Viewport,
		"android_window": window,
	}
	layoutJSON, err := json.Marshal(layout)
	if err != nil {
		return result, fmt.Errorf("encode Android EX9 step metadata: %w", err)
	}
	route := strings.TrimSpace(request.Route)
	if route == "" {
		route = "/"
	}
	baseURL, err := (discovery.NewResolver(discovery.ResolverConfig{})).ResolveScenarioURL(ctx, "ui-health", "API_PORT")
	if err != nil {
		result.Reason = "UI Health AnalyzeArtifacts service is unavailable: " + err.Error()
		return result, nil
	}
	visual := visualconnect.NewVisualHealthServiceClient(c.ClientOrDefault(), baseURL)
	response, err := visual.AnalyzeArtifacts(ctx, connect.NewRequest(&visualpb.AnalyzeArtifactsRequest{
		Scenario: request.Scenario,
		RunId:    request.RunID,
		Steps: []*visualpb.VisualStepArtifact{{
			StepId: request.StepID, Label: "Android " + viewportID + " device frame", Url: route,
			Viewport:      &visualpb.Viewport{Width: int32(width), Height: int32(height), DeviceProfile: viewportID, DeviceScaleFactor: densityScale},
			ScreenshotPng: imageBytes, LayoutJson: string(layoutJSON),
		}},
	}))
	if err != nil {
		result.Reason = "UI Health could not analyze the Android frame: " + err.Error()
		result.Window = window
		return result, nil
	}
	result.AnalyzerVerdict = response.Msg.GetVerdict()
	result.Window = window
	verdict := strings.ToLower(strings.TrimSpace(result.AnalyzerVerdict))
	if verdict == "" || verdict == "not_evaluated" || verdict == "not-evaluated" || verdict == "skipped" {
		result.Reason = "UI Health did not evaluate the Android screenshot"
		return result, nil
	}
	for _, finding := range response.Msg.GetFindings() {
		if finding == nil {
			continue
		}
		result.Findings = append(result.Findings, fmt.Sprintf("%s %s: %s", finding.GetSeverity(), finding.GetCode(), finding.GetMessage()))
	}
	if verdict == "failed" {
		result.Status = "failed"
		result.Reason = "UI Health reported a failed Android screenshot verdict"
	} else if keyboardMismatch {
		result.Status = "failed"
	} else if verdict == "passed" {
		result.Status = "evidence-collected"
	} else {
		result.Reason = "UI Health returned an unsupported verdict: " + result.AnalyzerVerdict
		return AndroidExperienceResult{Status: "not-evaluated", AssessmentScope: result.AssessmentScope, Reason: result.Reason, ObservedAt: result.ObservedAt, DeviceEvidence: result.DeviceEvidence, ScreenshotSHA256: result.ScreenshotSHA256, Viewport: result.Viewport, Window: result.Window}, nil
	}
	return result, nil
}

func (c *HTTPDeviceClient) readAndroidDeviceState(ctx context.Context, deviceID string) (androidDeviceStateResponse, error) {
	baseURL, err := url.Parse(strings.TrimRight(c.BaseURL, "/"))
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return androidDeviceStateResponse{}, fmt.Errorf("device-control base URL is invalid")
	}
	requestURL := strings.TrimRight(c.BaseURL, "/") + "/api/v1/devices/" + urlPathSegment(deviceID) + "/state"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return androidDeviceStateResponse{}, err
	}
	response, err := c.ClientOrDefault().Do(request)
	if err != nil {
		return androidDeviceStateResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return androidDeviceStateResponse{}, fmt.Errorf("device-control state returned %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	var state androidDeviceStateResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 256*1024)).Decode(&state); err != nil {
		return androidDeviceStateResponse{}, fmt.Errorf("decode device-control state: %w", err)
	}
	return state, nil
}
