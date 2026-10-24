package designcapture

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	executionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/execution"
	"google.golang.org/protobuf/encoding/protojson"
)

// BASDispatcher supplies a bounded workflow to the existing browser owner.
// RCLTargetBaseURL must serve this operation's immutable target HTML.
type BASDispatcher struct {
	BASBaseURL       string
	RCLTargetBaseURL string
	Client           *http.Client
}

func (d BASDispatcher) Start(ctx context.Context, op Operation) (string, error) {
	reject := func(err error) (string, error) { return "", NotDispatchedError{Err: err} }
	if err := validateRequest(op.Request); err != nil {
		return reject(err)
	}
	if op.State != Dispatching || !strings.HasPrefix(op.ID, "capture_") || !hashPattern.MatchString(strings.TrimPrefix(op.ID, "capture_")) {
		return reject(fmt.Errorf("persisted dispatch identity is required"))
	}
	for _, base := range []string{d.BASBaseURL, d.RCLTargetBaseURL} {
		u, err := url.Parse(base)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return reject(fmt.Errorf("valid browser and target service URLs are required"))
		}
	}
	targetURL := strings.TrimRight(d.RCLTargetBaseURL, "/") + "/design-captures/" + op.ID + "/target.html"
	expected, _ := json.Marshal(op.Request.Target.RenderHash)
	expression := `(async()=>{const expected=` + string(expected) + `;if(document.querySelector('meta[name="bundle-sha256"]')?.content!==expected)throw Error("stale_render_target");await document.fonts.ready;await new Promise(r=>requestAnimationFrame(()=>requestAnimationFrame(r)));return {renderHash:expected,viewport:{width:innerWidth,height:innerHeight},regions:[...document.querySelectorAll('[data-design-region]')].map(n=>{const r=n.getBoundingClientRect();return {region:n.getAttribute('data-design-region'),x:r.x,y:r.y,width:r.width,height:r.height}})};})()`
	payload := map[string]any{
		"waitForCompletion": false,
		"metadata":          map[string]any{"name": op.ID, "description": "RCL preview capture " + op.Request.Target.Revision + " render " + op.Request.Target.RenderHash},
		"flowDefinition": map[string]any{
			"metadata": map[string]any{"name": op.ID, "version": "1"},
			"settings": map[string]any{"viewportWidth": op.Request.Width, "viewportHeight": op.Request.Height},
			"nodes": []any{
				map[string]any{"id": "open", "action": map[string]any{"type": "ACTION_TYPE_NAVIGATE", "navigate": map[string]any{"destinationType": "NAVIGATE_DESTINATION_TYPE_URL", "url": targetURL, "waitUntil": "NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED", "timeoutMs": 30000}}},
				map[string]any{"id": "ready", "action": map[string]any{"type": "ACTION_TYPE_WAIT", "wait": map[string]any{"selector": "[data-preview-readiness-marker][data-preview-ready=\"true\"]", "state": "WAIT_STATE_VISIBLE", "timeoutMs": 20000}}},
				map[string]any{"id": "target", "action": map[string]any{"type": "ACTION_TYPE_EVALUATE", "evaluate": map[string]any{"expression": expression}}},
			},
			"edges": []any{map[string]any{"id": "open-ready", "source": "open", "target": "ready", "type": "WORKFLOW_EDGE_TYPE_SMOOTHSTEP"}, map[string]any{"id": "ready-target", "source": "ready", "target": "target", "type": "WORKFLOW_EDGE_TYPE_SMOOTHSTEP"}},
		},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return reject(err)
	}
	// Decode through the producer-owned schema before any network dispatch.
	var typed executionv1.ExecuteAdhocRequest
	if err := protojson.Unmarshal(raw, &typed); err != nil {
		return reject(fmt.Errorf("invalid BAS capture workflow: %w", err))
	}
	raw, err = protojson.Marshal(&typed)
	if err != nil {
		return reject(err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BASBaseURL, "/")+"/browser_automation_studio.v1.WorkflowsService/ExecuteAdhocWorkflow", bytes.NewReader(raw))
	if err != nil {
		return reject(err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 65537))
	if err != nil {
		return "", err
	}
	if len(body) > 65536 {
		return "", fmt.Errorf("BAS acknowledgement exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("BAS dispatch returned HTTP %d", response.StatusCode)
	}
	var result executionv1.ExecuteAdhocResponse
	if err := protojson.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if result.ExecutionId == "" || len(result.ExecutionId) > 200 {
		return "", fmt.Errorf("BAS omitted bounded execution identity")
	}
	return result.ExecutionId, nil
}
