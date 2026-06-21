package workflow

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/services/export"
)

func planFile(t *testing.T, plan *ExportPlan, relPath string) []byte {
	t.Helper()
	for _, f := range plan.Files {
		if f.RelPath == relPath {
			return f.Content
		}
	}
	t.Fatalf("planned file %q not found", relPath)
	return nil
}

func TestBuildExportPlan_ProducesCanonicalFileSet(t *testing.T) {
	timeline := &export.ExecutionTimeline{
		ExecutionID: uuid.New(),
		Status:      "completed",
		Frames:      []export.TimelineFrame{{NodeID: "n1", Status: "completed", Success: true}},
	}

	plan, err := BuildExportPlan(timeline, "My Flow")
	require.NoError(t, err)

	want := []string{
		"timeline.json", "result.json", "README.md",
		"execution-summary.md", "console-logs.md",
		"network-activity.md", "assertions.md",
	}
	got := map[string]bool{}
	for _, f := range plan.Files {
		got[f.RelPath] = true
	}
	for _, w := range want {
		require.Truef(t, got[w], "missing planned file %q", w)
	}
	require.Empty(t, plan.Screenshots, "no screenshots in this timeline")
}

func TestBuildExportPlan_ResultJSONReflectsStatus(t *testing.T) {
	timeline := &export.ExecutionTimeline{
		ExecutionID: uuid.New(),
		Status:      "completed",
		Frames: []export.TimelineFrame{
			{NodeID: "n1", Status: "completed", Success: true},
			{NodeID: "n2", Status: "failed", Error: "kaboom"},
		},
	}
	plan, err := BuildExportPlan(timeline, "")
	require.NoError(t, err)

	var result ResultSummary
	require.NoError(t, json.Unmarshal(planFile(t, plan, "result.json"), &result))
	require.True(t, result.Success)
	require.Equal(t, 2, result.StepsTotal)
	require.Equal(t, 1, result.StepsCompleted)
	require.Equal(t, 1, result.StepsFailed)
	require.Equal(t, "kaboom", result.Error)
}

func TestBuildExportPlan_PlansScreenshotCopies(t *testing.T) {
	timeline := &export.ExecutionTimeline{
		ExecutionID: uuid.New(),
		Status:      "completed",
		Frames: []export.TimelineFrame{
			{
				NodeID:     "shot node",
				Status:     "completed",
				Screenshot: &export.TimelineScreenshot{URL: "/api/v1/screenshots/obj-123", ContentType: "image/jpeg"},
			},
		},
	}
	plan, err := BuildExportPlan(timeline, "Flow")
	require.NoError(t, err)
	require.Len(t, plan.Screenshots, 1)
	require.Equal(t, "obj-123", plan.Screenshots[0].ObjectName)
	require.Equal(t, "jpg", plan.Screenshots[0].FallbackExt)
	require.Contains(t, plan.Screenshots[0].RelPathNoExt, "screenshots/step-01-")
}

func TestBuildExportPlan_DefaultsWorkflowName(t *testing.T) {
	timeline := &export.ExecutionTimeline{ExecutionID: uuid.New(), Status: "completed"}
	plan, err := BuildExportPlan(timeline, "")
	require.NoError(t, err)
	require.Contains(t, string(planFile(t, plan, "README.md")), "Unnamed Workflow")
}
