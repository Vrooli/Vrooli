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

func TestDeriveRequirementsUsesStepTypeMatrix(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, Type: "uploadFile"},
			{Index: 1, Type: "tabSwitch"},
			{Index: 2, Type: "frame_switch"},
			{Index: 3, Type: "networkMock"},
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsFileUploads || !req.NeedsParallelTabs || !req.NeedsIframes || !req.NeedsHAR || !req.NeedsTracing {
		t.Fatalf("expected matrix-derived requirements to include uploads/tabs/iframes/HAR/tracing, got %+v", req)
	}
}

func TestDeriveRequirementsIncludesGraphStepsAndTypes(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Graph: &contracts.PlanGraph{
			Steps: []contracts.PlanStep{
				{
					Index:  0,
					Type:   "download_file",
					Params: map[string]any{"downloadTarget": "file"},
				},
				{
					Index:  1,
					Type:   "upload",
					Params: map[string]any{"filePath": "one.txt"},
				},
				{
					Index:  2,
					Type:   "loop",
					Params: map[string]any{"loopType": "repeat"},
					Loop: &contracts.PlanGraph{
						Steps: []contracts.PlanStep{
							{Index: 3, Type: "networkMock", Params: map[string]any{"networkMockType": "delay"}},
						},
					},
				},
			},
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsDownloads || !req.NeedsFileUploads || !req.NeedsHAR || !req.NeedsTracing {
		t.Fatalf("expected graph-derived requirements to mark downloads/uploads/HAR/tracing, got %+v", req)
	}
}

func TestDeriveRequirementsHonorsTracingAndViewportMetadata(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Metadata: map[string]any{
			"executionViewport": map[string]any{
				"width":  float64(1400),
				"height": float64(900),
			},
			"requiresTracing": true,
			"requiresVideo":   true,
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index: 0,
				Type:  "navigate",
				Params: map[string]any{
					"trace": true,
				},
			},
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsTracing || !req.NeedsVideo {
		t.Fatalf("expected tracing/video requirements, got %+v", req)
	}
	if req.MinViewportWidth != 1400 || req.MinViewportHeight != 900 {
		t.Fatalf("expected viewport minima 1400x900, got %dx%d", req.MinViewportWidth, req.MinViewportHeight)
	}
}

func TestDeriveRequirementsIncludesTraceVideoHarStepTypes(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Instructions: []contracts.CompiledInstruction{
			{Index: 0, Type: "trace"},
			{Index: 1, Type: "video"},
			{Index: 2, Type: "har"},
		},
	}
	req := deriveRequirements(plan)
	if !req.NeedsTracing || !req.NeedsVideo || !req.NeedsHAR {
		t.Fatalf("expected trace/video/har requirements, got %+v", req)
	}
}

func TestDeriveRequirementsRecordingBooleans(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Instructions: []contracts.CompiledInstruction{
			{
				Index: 0,
				Type:  "navigate",
				Params: map[string]any{
					"recordNetwork": true,
					"recordTrace":   "true",
					"recordVideo":   "yes",
				},
			},
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsHAR || !req.NeedsTracing || !req.NeedsVideo {
		t.Fatalf("expected recording flags to enforce HAR/tracing/video, got %+v", req)
	}
}

func TestAnalyzeRequirementsReturnsReasons(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		CreatedAt:   time.Now().UTC(),
		Instructions: []contracts.CompiledInstruction{
			{
				Index: 0,
				Type:  "networkMock",
				Params: map[string]any{
					"networkMockType": "abort",
				},
			},
			{
				Index: 1,
				Type:  "downloadFile",
			},
		},
	}

	req, reasons := analyzeRequirements(plan)
	if !req.NeedsHAR || !req.NeedsTracing || !req.NeedsDownloads {
		t.Fatalf("expected requirements to include HAR/tracing/downloads, got %+v", req)
	}
	if len(reasons["har"]) == 0 || len(reasons["tracing"]) == 0 || len(reasons["downloads"]) == 0 {
		t.Fatalf("expected reasons for missing capabilities, got %+v", reasons)
	}
}
