//go:build legacydb
// +build legacydb

package executor

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
)

func TestDeriveRequirementsFromMetadataAndParams(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Metadata: map[string]any{
			"requiresDownloads": true,
			"requiresHar":       true,
			"executionViewport": map[string]any{"width": float64(1400), "height": float64(900)},
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "nav",
				Type:   "navigate",
				Params: map[string]any{
					"tabSwitchBy":    "index",
					"frameSelector":  "#app",
					"filePath":       "/tmp/file.txt",
					"viewportWidth":  float64(1600),
					"viewportHeight": float64(950),
				},
			},
		},
	}

	req := deriveRequirements(plan)

	if !req.NeedsDownloads || !req.NeedsHAR || !req.NeedsParallelTabs || !req.NeedsIframes || !req.NeedsFileUploads {
		t.Fatalf("capability flags not derived: %+v", req)
	}
	if req.MinViewportWidth != 1600 || req.MinViewportHeight != 950 {
		t.Fatalf("viewport minima not derived: %+v", req)
	}
}

func TestDeriveRequirementsNetworkMockSetsHarAndTracing(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  1,
				NodeID: "mock",
				Type:   "navigate",
				Params: map[string]any{
					"networkMockType":   "abort",
					"networkUrlPattern": "*",
				},
			},
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsHAR || !req.NeedsTracing {
		t.Fatalf("network mock should require HAR+tracing: %+v", req)
	}
}

func TestDeriveRequirementsDownloadsVideoViewport(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Metadata: map[string]any{
			"requiresDownloads": true,
			"requiresVideo":     true,
		},
		Instructions: []contracts.CompiledInstruction{
			{
				Index:  0,
				NodeID: "nav",
				Type:   "navigate",
				Params: map[string]any{
					"viewportWidth":  float64(1921),
					"viewportHeight": float64(1081),
				},
			},
		},
	}
	req := deriveRequirements(plan)
	if !req.NeedsDownloads || !req.NeedsVideo {
		t.Fatalf("expected downloads+video requirements derived, got %+v", req)
	}
	if req.MinViewportWidth != 1921 || req.MinViewportHeight != 1081 {
		t.Fatalf("viewport minima wrong: %+v", req)
	}

	caps := contracts.EngineCapabilities{
		SchemaVersion:         contracts.CapabilitiesSchemaVersion,
		Engine:                "stub",
		MaxConcurrentSessions: 1,
		AllowsParallelTabs:    true,
		SupportsHAR:           true,
		SupportsVideo:         false,
		SupportsIframes:       true,
		SupportsFileUploads:   true,
		SupportsDownloads:     false,
		SupportsTracing:       true,
		MaxViewportWidth:      1919,
		MaxViewportHeight:     1080,
	}
	gap := caps.CheckCompatibility(req)
	if gap.Satisfied() {
		t.Fatalf("expected capability gap, got satisfied")
	}
	if len(gap.Missing) == 0 {
		t.Fatalf("expected missing requirements reported, got %+v", gap)
	}
}

func TestAnalyzeRequirementsAggregatesGraphAndSubflowReasons(t *testing.T) {
	inner := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{Index: 0, NodeID: "inner-download", Type: "download_file"},
		},
	}
	graph := &contracts.PlanGraph{
		Steps: []contracts.PlanStep{
			{Index: 0, NodeID: "mock", Type: "network_mock"},
			{Index: 1, NodeID: "loop", Type: "loop", Loop: inner},
		},
	}

	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Graph:       graph,
		Metadata: map[string]any{
			"executionViewport": map[string]any{"width": float64(1440), "height": float64(900)},
		},
	}

	reqs, reasons := analyzeRequirements(plan)
	if !reqs.NeedsHAR || !reqs.NeedsTracing || !reqs.NeedsDownloads {
		t.Fatalf("expected HAR+tracing+downloads requirements from graph/subflow, got %+v", reqs)
	}
	if reqs.MinViewportWidth != 1440 || reqs.MinViewportHeight != 900 {
		t.Fatalf("expected viewport minima from metadata, got %+v", reqs)
	}
	if len(reasons["har"]) == 0 || len(reasons["tracing"]) == 0 {
		t.Fatalf("expected reasons to include graph nodes for HAR/tracing: %+v", reasons)
	}
	if len(reasons["downloads"]) == 0 || !strings.Contains(strings.ToLower(strings.Join(reasons["downloads"], " ")), "inner-download") {
		t.Fatalf("expected subflow download reason to be captured, got %+v", reasons["downloads"])
	}
}

func TestDeriveRequirementsMetadataOnlyFlags(t *testing.T) {
	plan := contracts.ExecutionPlan{
		ExecutionID: uuid.New(),
		WorkflowID:  uuid.New(),
		Metadata: map[string]any{
			"requiresDownloads":    true,
			"requiresFileUploads":  true,
			"requiresVideo":        true,
			"requiresTracing":      true,
			"requiresIframes":      true,
			"requiresParallelTabs": true,
			"requiresHAR":          true, // uppercase variant should still be honored
		},
	}

	req := deriveRequirements(plan)
	if !req.NeedsDownloads ||
		!req.NeedsFileUploads ||
		!req.NeedsVideo ||
		!req.NeedsTracing ||
		!req.NeedsIframes ||
		!req.NeedsParallelTabs ||
		!req.NeedsHAR {
		t.Fatalf("expected metadata flags to map to requirements, got %+v", req)
	}
}
