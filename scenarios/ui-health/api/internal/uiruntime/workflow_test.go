package uiruntime

import (
	"strings"
	"testing"
)

func TestArtifactCaptureScriptCollectsOnlyDeclaredExperienceSurfaces(t *testing.T) {
	script := artifactCaptureScript()
	for _, fragment := range []string{"[data-experience-surface]", "data-experience-state", "experienceSurfaces: experienceSurfaces(doc)"} {
		if !strings.Contains(script, fragment) {
			t.Fatalf("artifact capture script missing %q", fragment)
		}
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
