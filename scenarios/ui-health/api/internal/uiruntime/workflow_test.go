package uiruntime

import (
	"strings"
	"testing"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
)

func TestArtifactCaptureScriptCollectsOnlyDeclaredExperienceSurfaces(t *testing.T) {
	script := artifactCaptureScript()
	for _, fragment := range []string{"[data-experience-surface]", "data-experience-state", "experienceSurfaces: experienceSurfaces(doc)", "Math.max(root.scrollHeight, body.scrollHeight) > window.innerHeight + 1"} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("artifact capture script missing %q", fragment)
		}
	}
}

func TestArtifactCaptureScriptRecordsNativeDisabledState(t *testing.T) {
	if !strings.Contains(artifactCaptureScript(), "el.matches(':disabled')") {
		t.Fatal("artifact capture must retain native disabled state for focusability checks")
	}
}

func TestHandshakeWorkflowWaitsForDeclaredRequiredSurfaceTerminalState(t *testing.T) {
	def := buildHandshakeWorkflow("http://scenario", nil, []requiredSurface{{id: "results", required: true, states: map[string]bool{"loading": true, "ready": true, "empty": true}}}, 0, 1280, 720)
	encoded := ""
	for _, raw := range def["nodes"].([]any) {
		node := raw.(map[string]any)
		if node["id"] == nodeReadiness {
			action := node["action"].(map[string]any)
			assert := action["assert"].(map[string]any)
			if assert["selector"] != "[data-smoke-experience-settled]" {
				t.Fatalf("readiness selector = %v", assert["selector"])
			}
		}
		if node["id"] == nodeInject {
			encoded = node["action"].(map[string]any)["evaluate"].(map[string]any)["expression"].(string)
		}
	}
	if !strings.Contains(encoded, `"id":"results"`) || !strings.Contains(encoded, `"ready"`) || strings.Contains(encoded, `"loading"`) {
		t.Fatalf("injection expectations must contain only declared terminal states: %s", encoded)
	}
}

func TestHandshakeWorkflowKeepsHandshakeOnlyPathWithoutDeclaredSurfaces(t *testing.T) {
	def := buildHandshakeWorkflow("http://scenario", nil, nil, 0, 1280, 720)
	foundRenderSettle := false
	for _, raw := range def["nodes"].([]any) {
		node := raw.(map[string]any)
		if node["id"] == nodeReadiness {
			t.Fatal("legacy runtime checks must not gain an undeclared readiness gate")
		}
		if node["id"] == nodeRenderSettle {
			wait := node["action"].(map[string]any)["wait"].(map[string]any)
			if wait["duration_ms"] != renderSettleDelay.Milliseconds() {
				t.Fatalf("render settle duration = %v", wait["duration_ms"])
			}
			foundRenderSettle = true
		}
	}
	if !foundRenderSettle {
		t.Fatal("runtime workflow must wait for its first post-handshake paint")
	}
}

func TestHandshakeWorkflowCapturesExactKeyboardFocusAfterInitialLayout(t *testing.T) {
	def := buildHandshakeWorkflow("http://scenario", nil, nil, 0, 1280, 720)
	nodes := def["nodes"].([]any)
	positions := map[string]int{}
	for i, raw := range nodes {
		node := raw.(map[string]any)
		positions[node["id"].(string)] = i
	}
	if !(positions[nodeArtifacts] < positions[nodeScreens] && positions[nodeScreens] < positions[nodeFocusFrame] && positions[nodeFocusFrame] < positions[nodeKeyboardTab] && positions[nodeKeyboardTab] < positions[nodeFocusCapture]) {
		t.Fatalf("focus probe must capture layout before entering the target frame and sending Tab: %#v", positions)
	}
	var focusExpression string
	var frameSwitch map[string]any
	for _, raw := range nodes {
		node := raw.(map[string]any)
		action := node["action"].(map[string]any)
		if node["id"] == nodeFocusFrame {
			if action["type"] != "ACTION_TYPE_FRAME_SWITCH" {
				t.Fatalf("keyboard focus must switch BAS execution into the embedded UI frame; action=%#v", action)
			}
			frameSwitch = action["frameSwitch"].(map[string]any)
		}
		if node["id"] == nodeKeyboardTab && action["type"] != "ACTION_TYPE_KEYBOARD" {
			t.Fatalf("keyboard traversal must remain a real BAS keyboard action; action=%#v", action)
		}
		if node["id"] == nodeFocusCapture {
			focusExpression = action["evaluate"].(map[string]any)["expression"].(string)
		}
	}
	if frameSwitch["action"] != "FRAME_SWITCH_ACTION_ENTER" || frameSwitch["selector"] != hostFrameSelector {
		t.Fatalf("keyboard focus must enter the exact embedded UI frame: %#v", frameSwitch)
	}
	compiled, err := definitionToProto(def)
	if err != nil {
		t.Fatalf("workflow frame-switch action must satisfy BAS's typed contract: %v", err)
	}
	var enteredFrame, sentTab bool
	for _, node := range compiled.GetNodes() {
		switch node.GetId() {
		case nodeFocusFrame:
			params := node.GetAction().GetFrameSwitch()
			if params == nil || params.GetAction() != basactions.FrameSwitchAction_FRAME_SWITCH_ACTION_ENTER || params.GetSelector() != hostFrameSelector {
				t.Fatalf("typed BAS workflow does not enter the target iframe: %+v", node.GetAction())
			}
			enteredFrame = true
		case nodeKeyboardTab:
			sentTab = node.GetAction().GetKeyboard() != nil && node.GetAction().GetKeyboard().GetKey() == "Tab"
		}
	}
	if !enteredFrame || !sentTab {
		t.Fatalf("typed BAS workflow must enter the target frame before a real Tab; entered=%v tab=%v", enteredFrame, sentTab)
	}
	for _, fragment := range []string{"activeElement", "matches(':focus-visible')", "nodeKey", "nodeIdentity", "getBoundingClientRect", "parent.getBoundingClientRect", "rect.right <= right"} {
		if !strings.Contains(focusExpression, fragment) {
			t.Fatalf("keyboard focus capture missing %q", fragment)
		}
	}
}

func TestHandshakeWorkflowRequestsFreshSessionAndRecordsValidationContext(t *testing.T) {
	def := buildHandshakeWorkflow("https://scenario.example/dashboard", nil, nil, 0, 390, 844)
	metadata := def["metadata"].(map[string]any)
	labels := metadata["labels"].(map[string]string)
	if labels["session_reuse_mode"] != "fresh" {
		t.Fatalf("session reuse mode = %q, want fresh", labels["session_reuse_mode"])
	}
	if labels["validation_route"] != "https://scenario.example/dashboard" || labels["validation_viewport"] != "390x844" {
		t.Fatalf("validation labels = %#v", labels)
	}
}

func TestHandshakeWorkflowBindsBridgeReadinessToChildOriginAndNonce(t *testing.T) {
	script := injectionScript("https://scenario.example/path?existing=1", nil, nil)
	for _, fragment := range []string{
		"__vrooli_bridge_nonce=",
		"ev.source !== frame.contentWindow",
		"ev.origin !== expectedOrigin",
		"d.nonce !== nonce",
		"frame.addEventListener('load'",
		"if (!frameLoaded) { return; }",
	} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("bridge injection missing %q", fragment)
		}
	}
}

func TestArtifactCaptureKeepsEffectiveAndRawInsetsAndSeparateBottomOwner(t *testing.T) {
	script := artifactCaptureScript()
	for _, fragment := range []string{"out.raw =", "out.bottom = parseFloat", "var(--rcl-safe-bottom", "safeAreaColors: bottomColor", "[data-rcl-bottom-nav]", "[data-state=\"open\"]"} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("capture is missing safe-area evidence %q", fragment)
		}
	}
}
