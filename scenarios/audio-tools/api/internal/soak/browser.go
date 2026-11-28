//go:build audio_lab

package soak

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"audio-tools/internal/conformance"
)

func (d *driver) pageURL() string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		page := strings.TrimRight(d.opt.UIURL, "/") + "/plan"
		if d.opt.Fault != "" {
			page += "?stt_test_mode=1&stt_test_fault=" + url.QueryEscape(d.opt.Fault)
		}
		return page
	}
	page := strings.TrimRight(d.opt.UIURL, "/") + "/dictation-studio"
	page += "?stt_engine_id=" + d.opt.EngineID
	if strings.TrimSpace(d.opt.Strategy) != "" {
		page += "&stt_strategy=" + url.QueryEscape(strings.TrimSpace(d.opt.Strategy))
	}
	if d.opt.Lane == conformance.LaneAccelerated {
		page += "&stt_test_mode=1&stt_capture_source=virtual&stt_capture_shape=" + url.QueryEscape(normalizeShape(d.opt.Shape))
		page += "&stt_virtual_samples=" + strconv.FormatInt(targetSamplesPerTurn(d.opt), 10)
		page += "&stt_corpus_url=" + url.QueryEscape(strings.TrimRight(d.opt.UIURL, "/")+"/api/v1/validation/soak/virtual-corpus?clip=quick-brown-fox")
	}
	if d.opt.Fault != "" {
		page += "&stt_test_mode=1&stt_test_fault=" + d.opt.Fault
	}
	return page
}

func normalizeSurface(surface string) string {
	if strings.EqualFold(strings.TrimSpace(surface), "swarm-manager") {
		return "swarm-manager"
	}
	return "audio-tools"
}

// productSelector keeps the browser driver honest about which consumer it is
// exercising. Audio Tools owns the full Dictation Studio recorder state;
// swarm-manager owns a MessageComposer that embeds the same shared capture
// package, so its lifecycle is represented by the RCL voice-control state.
func (d *driver) productSelector(name string) string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		const mic = `[data-testid="captures-quick-input-mic"]`
		switch name {
		case "ready":
			return mic
		case "recording":
			return mic + `[data-state="recording"]`
		case "terminal":
			return mic + `[data-state="idle"], ` + mic + `[data-state="error"]`
		case "interim":
			return `[data-testid="captures-quick-composer-interim"]`
		default:
			return ""
		}
	}
	switch name {
	case "ready":
		return `[data-testid="dictation-record-start"]`
	case "recording":
		return `[data-testid="dictation-record-state"][data-recorder-state="recording"]`
	case "processed":
		return `[data-testid="dictation-turn-processed-ready"]`
	case "terminal":
		return `[data-testid="dictation-record-state"][data-recorder-state="captured"], [data-testid="dictation-record-state"][data-recorder-state="failed"]`
	case "interim":
		return `[data-testid="dictation-interim-transcript"]`
	default:
		return ""
	}
}

func (d *driver) prepareSurface(ctx context.Context, sessionID string) error {
	if normalizeSurface(d.opt.Surface) != "swarm-manager" {
		return d.step(ctx, sessionID, 1, "wait-recorder", map[string]any{"wait": map[string]any{"selector": d.productSelector("ready"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}})
	}
	if err := d.step(ctx, sessionID, 1, "wait-action-menu", map[string]any{"wait": map[string]any{"selector": `[data-testid="graph-action-fab"]`, "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}}); err != nil {
		return err
	}
	if err := d.step(ctx, sessionID, 2, "open-action-menu", map[string]any{"click": map[string]any{"selector": `[data-testid="graph-action-fab"]`}}); err != nil {
		return err
	}
	if err := d.step(ctx, sessionID, 3, "open-quick-capture", map[string]any{"click": map[string]any{"selector": `[role="menuitem"][aria-label="Quick Capture"]`}}); err != nil {
		return err
	}
	return d.step(ctx, sessionID, 4, "wait-recorder", map[string]any{"wait": map[string]any{"selector": d.productSelector("ready"), "state": "WAIT_STATE_ATTACHED", "timeout_ms": 15000}})
}

const virtualCorpusSamples = int64(31_268)

func targetSamplesPerTurn(o Options) int64 {
	if o.SimulatedMinutes <= 0 || o.Turns <= 0 {
		return virtualCorpusSamples
	}
	total := int64(o.SimulatedMinutes) * 60 * canonicalSampleRate
	perTurn := total / int64(o.Turns)
	if perTurn < virtualCorpusSamples {
		return virtualCorpusSamples
	}
	return perTurn
}

func normalizeShape(shape string) string {
	if strings.EqualFold(strings.TrimSpace(shape), "chunked") {
		return "chunked"
	}
	return "burst"
}

// realtimeShape is deliberately separate from normalizeShape. The accelerated
// lane shapes the virtual corpus (burst/chunked), while the realtime lane
// shapes host-device playback (realistic/continuous). The CLI historically
// sends the accelerated default shape for every lane, so only the two
// realtime-specific values are allowed to override profile here.
func realtimeShape(o Options) string {
	switch strings.ToLower(strings.TrimSpace(o.Shape)) {
	case "realistic", "continuous":
		return strings.ToLower(strings.TrimSpace(o.Shape))
	}
	if strings.EqualFold(strings.TrimSpace(o.Profile), "continuous") {
		return "continuous"
	}
	return "realistic"
}

func realtimePlaybackPauseMS(o Options) int {
	// A short pause keeps repeated fixture cycles acoustically separable. Without
	// it, the native decoder merges the boundary and consistently turns the
	// second "a quick" into "the quick", which is a real product-quality error.
	if realtimeShape(o) == "realistic" {
		return 250
	}
	return 0
}

func (d *driver) start(ctx context.Context) (sessionStart, error) {
	var out sessionStart
	if d.owner == nil {
		return out, errors.New("managed browser owner context is unavailable")
	}
	audioMode := "fake_microphone"
	if d.opt.Lane == conformance.LaneRealtime {
		audioMode = "pipewire_qualification"
	}
	request := map[string]any{
		"backend":                 "managed",
		"reuse_mode":              "fresh",
		"start_url":               d.pageURL(),
		"audio_mode":              audioMode,
		"audio_playback_pause_ms": realtimePlaybackPauseMS(d.opt),
	}
	if d.opt.MobileRecovery {
		request["viewport_width"], request["viewport_height"] = 390, 844
	} else {
		request["viewport_width"], request["viewport_height"] = 1280, 900
	}
	var response struct {
		SessionID           string               `json:"session_id"`
		AudioDeviceEvidence *AudioDeviceEvidence `json:"audio_device_evidence"`
	}
	if err := d.postBroker(ctx, "/browser/sessions", request, &response); err != nil {
		return out, err
	}
	out.SessionID, out.AudioDeviceEvidence = response.SessionID, response.AudioDeviceEvidence
	if out.SessionID == "" {
		return sessionStart{}, errors.New("BAS did not return a managed browser session")
	}
	if d.opt.Lane == conformance.LaneRealtime && (out.AudioDeviceEvidence == nil || !out.AudioDeviceEvidence.Enumerated) {
		closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if closeErr := d.close(closeCtx, out); closeErr != nil {
			return sessionStart{}, fmt.Errorf("BAS did not return a PipeWire-qualified session and device evidence; cleanup failed: %w", closeErr)
		}
		return sessionStart{}, errors.New("BAS did not return a PipeWire-qualified session and device evidence")
	}
	return out, nil
}

func (d *driver) close(ctx context.Context, s sessionStart) error {
	return d.postBroker(ctx, "/browser/sessions/"+url.PathEscape(s.SessionID)+"/close", map[string]any{}, nil)
}

func (d *driver) step(ctx context.Context, sessionID string, index int, nodeID string, params map[string]any) error {
	if nodeID == "navigate" {
		// Broker open already navigated to this exact task URL.
		return nil
	}
	actionType := actionTypeForStep(nodeID, params)
	instruction := map[string]any{
		"index": index, "node_id": nodeID,
		"action":    mergeAction(actionType, params),
		"telemetry": map[string]string{"screenshot": "SCREENSHOT_CAPTURE_POLICY_NEVER"},
	}
	var managed struct {
		Outcome struct {
			Success       bool           `json:"success"`
			ExtractedData map[string]any `json:"extracted_data"`
		} `json:"outcome"`
	}
	if err := d.postBroker(ctx, "/browser/sessions/"+url.PathEscape(sessionID)+"/act", map[string]any{"instruction": instruction}, &managed); err != nil {
		return err
	}
	if !managed.Outcome.Success {
		return fmt.Errorf("BAS broker step %q failed", nodeID)
	}
	return nil
}

func actionTypeForStep(nodeID string, params map[string]any) string {
	actionType := map[string]string{"navigate": "ACTION_TYPE_NAVIGATE", "click": "ACTION_TYPE_CLICK", "wait": "ACTION_TYPE_WAIT"}[nodeID]
	if nodeID == "start" || nodeID == "stop" {
		actionType = "ACTION_TYPE_CLICK"
	}
	if strings.HasPrefix(nodeID, "feed-") || nodeID == "clear-processed" || nodeID == "processed" || nodeID == "fault-terminal" || nodeID == "settle" || nodeID == "wait-recorder" || nodeID == "recording" || nodeID == "captured" {
		actionType = "ACTION_TYPE_WAIT"
	}
	// Derive named surface-preparation actions from their payload when the node
	// id is new, so a selector/lifecycle addition cannot emit an untyped action.
	if actionType == "" {
		switch {
		case params["navigate"] != nil:
			return "ACTION_TYPE_NAVIGATE"
		case params["click"] != nil:
			return "ACTION_TYPE_CLICK"
		case params["wait"] != nil:
			return "ACTION_TYPE_WAIT"
		}
	}
	return actionType
}

func (d *driver) evaluate(ctx context.Context, sessionID string, index int, out *pageObservation) error {
	var managed struct {
		Diagnostics pageObservation `json:"diagnostics"`
	}
	if err := d.postBroker(ctx, "/browser/sessions/"+url.PathEscape(sessionID)+"/audio/diagnostics", map[string]string{"surface": normalizeSurface(d.opt.Surface)}, &managed); err != nil {
		return err
	}
	*out = managed.Diagnostics
	return nil
}

func (d *driver) evaluationExpression() string {
	if normalizeSurface(d.opt.Surface) == "swarm-manager" {
		return `(function(){const interim=document.querySelector('[data-testid="captures-quick-composer-interim"]');const style=interim?window.getComputedStyle(interim):null;const rendered=!!interim&&!!style&&style.display!=="none"&&style.visibility!=="hidden"&&interim.getClientRects().length>0;return {diag:window.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest||null,final:document.querySelector('[data-testid="captures-quick-input-submit"]')?.value||'',interim:rendered?(interim.textContent||''):''};})()`
	}
	return `(function(){const interim=document.querySelector('[data-testid="dictation-interim-transcript"]');const style=interim?window.getComputedStyle(interim):null;const rendered=!!interim&&!!style&&style.display!=="none"&&style.visibility!=="hidden"&&interim.getClientRects().length>0;const diag=window.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest||null;const details=document.querySelector('[data-testid="dictation-turn-details"]');return {diag:diag,final:document.querySelector('[data-testid="dictation-final-transcript"]')?.textContent||'',interim:rendered?(interim.textContent||''):'',coverage_visible:!!details&&!!details.querySelector('[data-testid="dictation-turn-processed-status"]'),export_available:!!details&&!!details.querySelector('[data-testid="dictation-export-diagnostic"]'),diagnostic_metadata_only:!!diag&&!('audio' in diag)&&!('transcript' in diag)};})()`
}

func mergeAction(typ string, params map[string]any) map[string]any {
	out := map[string]any{"type": typ}
	for key, value := range params {
		out[key] = value
	}
	return out
}

func (d *driver) postBroker(ctx context.Context, path string, body, out any) error {
	if d.owner == nil || strings.TrimSpace(d.owner.brokerURL) == "" {
		return errors.New("managed BAS owner endpoint is unavailable")
	}
	var payload io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return errors.New("could not encode the typed BAS browser request")
		}
		payload = strings.NewReader(string(data))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.owner.brokerURL+path, payload)
	if err != nil {
		return errors.New("could not create BAS browser request")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Agent-Identity-Token", d.owner.identityToken)
	req.Header.Set("X-Vrooli-Browser-Grant-Receipt", d.owner.grantReceipt)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("BAS browser broker is unavailable: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return errors.New("could not read BAS browser response")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("BAS browser broker returned HTTP %d", resp.StatusCode)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return errors.New("BAS browser broker returned an invalid response")
		}
	}
	return nil
}

func (d *driver) postAt(ctx context.Context, baseURL, targetName, path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+path, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.Contains(targetName, "audio admin") {
		req.Header.Set("Connect-Protocol-Version", "1")
	}
	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", targetName, path, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s returned HTTP %d: %s", targetName, path, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	if out != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return fmt.Errorf("decode %s response: %w", targetName, err)
		}
	}
	return nil
}

func (d *driver) getPersistentMode(ctx context.Context) (bool, error) {
	var out streamConfigResponse
	err := d.postAt(ctx, d.opt.UIURL, "audio admin", audioAdminConfigPath+"GetStreamConfig", map[string]any{}, &out)
	return out.Config.PersistentMode, err
}

func (d *driver) setPersistentMode(ctx context.Context, enabled bool) error {
	return d.postAt(ctx, d.opt.UIURL, "audio admin", audioAdminConfigPath+"UpdateStreamConfig", map[string]any{
		// Connect's JSON mapping represents google.protobuf.FieldMask as its
		// canonical comma-separated string, and uses proto field names in that
		// string even though the message fields remain lowerCamelCase.
		"updateMask": "persistentMode",
		"config":     map[string]any{"persistentMode": enabled},
	}, nil)
}
