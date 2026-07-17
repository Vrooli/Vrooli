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
	for _, raw := range def["nodes"].([]any) {
		if raw.(map[string]any)["id"] == nodeReadiness {
			t.Fatal("legacy runtime checks must not gain an undeclared readiness gate")
		}
	}
}
