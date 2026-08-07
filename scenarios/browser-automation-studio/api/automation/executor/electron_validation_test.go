package executor

import (
	"testing"
)

func TestExtractElectronValidationRequiresLeaseBoundContext(t *testing.T) {
	metadata := map[string]any{
		"electron_target": map[string]any{
			"target_id":       "target-1",
			"cdp_endpoint":    "http://127.0.0.1:43123",
			"renderer_id":     "renderer-1",
			"renderer_url":    "file:///controlled/index.html",
			"scenario_name":   "controlled-scenario",
			"artifact_digest": "sha256:controlled",
			"context_id":      "ctx-1",
			"cdp_transport":   "loopback-authenticated",
		},
		"validation_context": map[string]any{
			"context_id":         "ctx-1",
			"scenario_name":      "controlled-scenario",
			"artifact_digest":    "sha256:controlled",
			"target_id":          "target-1",
			"workflow_id":        "workflow-1",
			"profile_id":         "normal",
			"isolation_lease_id": "lease-1",
		},
	}

	target, context, err := extractElectronValidation(metadata)
	if err != nil {
		t.Fatalf("extract validation: %v", err)
	}
	if target == nil || context == nil || context.IsolationLeaseID != "lease-1" {
		t.Fatalf("unexpected extracted validation: target=%+v context=%+v", target, context)
	}

	delete(metadata["validation_context"].(map[string]any), "isolation_lease_id")
	if _, _, err := extractElectronValidation(metadata); err == nil {
		t.Fatal("expected missing isolation lease to refuse Electron validation")
	}
}

func TestExtractElectronValidationAllowsNormalBrowserExecution(t *testing.T) {
	target, context, err := extractElectronValidation(map[string]any{"workflow_id": "browser-only"})
	if err != nil || target != nil || context != nil {
		t.Fatalf("normal browser metadata should not create Electron validation: target=%+v context=%+v err=%v", target, context, err)
	}
}
