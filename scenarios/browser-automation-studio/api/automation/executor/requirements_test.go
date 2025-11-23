package executor

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestDeriveRequirementsFromMetadataAndInstructions(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
			"executionViewport": map[string]any{
				"width":  float64(1024),
				"height": float64(768),
			},
			"requiresDownloads": true,
			"requiresHar":       true,
			"requiresVideo":     true,
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index: 0,
				Type:  "upload",
				Params: map[string]any{
					"filePaths": []any{"one.txt"},
				},
			},
			{
				Index: 1,
				Type:  "tab_switch",
				Params: map[string]any{
					"tabSwitchBy":   "title",
					"tabTitleMatch": "Docs",
				},
			},
			{
				Index: 2,
				Type:  "frame_switch",
				Params: map[string]any{
					"frameSelector": "#frame",
				},
			},
			{
				Index: 3,
				Type:  "screenshot",
				Params: map[string]any{
					"viewportWidth":  float64(1600),
					"viewportHeight": float64(900),
				},
			},
			{
				Index: 4,
				Type:  "networkMock",
				Params: map[string]any{
					"networkMockType":   "delay",
					"networkUrlPattern": "https://example.com",
				},
			},
		},
	}

	req := deriveRequirements(plan)

	if !req.NeedsFileUploads {
		t.Fatalf("expected NeedsFileUploads to be true")
	}
	if !req.NeedsParallelTabs {
		t.Fatalf("expected NeedsParallelTabs to be true")
	}
	if !req.NeedsIframes {
		t.Fatalf("expected NeedsIframes to be true")
	}
	if !req.NeedsDownloads || !req.NeedsHAR || !req.NeedsVideo || !req.NeedsTracing {
		t.Fatalf("expected metadata/network-driven needs (downloads/HAR/video/tracing) to be true, got %+v", req)
	}
	if req.MinViewportWidth != 1600 || req.MinViewportHeight != 900 {
		t.Fatalf("expected viewport minima 1600x900, got %dx%d", req.MinViewportWidth, req.MinViewportHeight)
	}
}

func TestDeriveRequirementsIgnoresEmptyParams(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Instructions: []contracts.CompiledInstruction{
			{
				Index: 0,
				Type:  "noop",
				Params: map[string]any{
					"filePaths": []any{},
				},
			},
		},
	}

	req := deriveRequirements(plan)
	if req.NeedsFileUploads || req.NeedsParallelTabs || req.NeedsIframes {
		t.Fatalf("expected no capability requirements, got %+v", req)
	}
}
