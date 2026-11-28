package export

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenerateConsoleLogsMarkdown_EmptyTimeline(t *testing.T) {
	out := GenerateConsoleLogsMarkdown(&ExecutionTimeline{})
	require.Contains(t, out, "# Console Logs")
	require.Contains(t, out, "No console logs captured")
}

func TestGenerateConsoleLogsMarkdown_RendersEntriesByStep(t *testing.T) {
	timeline := &ExecutionTimeline{
		Frames: []TimelineFrame{
			{
				NodeID: "nav-1",
				Artifacts: []TimelineArtifact{
					{
						Type: "console",
						Payload: map[string]any{
							"entries": []any{
								map[string]any{"type": "error", "text": "boom", "timestamp": float64(0)},
								map[string]any{"type": "log", "text": "hello"},
							},
						},
					},
				},
			},
			// A frame with no console artifacts must be skipped (no header).
			{NodeID: "nav-2"},
		},
	}

	out := GenerateConsoleLogsMarkdown(timeline)
	require.Contains(t, out, "## Step 1: nav-1")
	require.NotContains(t, out, "nav-2")
	require.Contains(t, out, "[ERROR] boom")
	require.Contains(t, out, "[LOG] hello")
}

func TestGenerateNetworkActivityMarkdown_EmptyTimeline(t *testing.T) {
	out := GenerateNetworkActivityMarkdown(&ExecutionTimeline{})
	require.Contains(t, out, "# Network Activity")
	require.Contains(t, out, "No network activity captured")
}

func TestGenerateNetworkActivityMarkdown_RendersRequestsAndResponses(t *testing.T) {
	timeline := &ExecutionTimeline{
		Frames: []TimelineFrame{
			{
				NodeID: "load",
				Artifacts: []TimelineArtifact{
					{
						Type: "network",
						Payload: map[string]any{
							"events": []any{
								map[string]any{"type": "request", "method": "GET", "url": "https://x.test/a"},
								map[string]any{"type": "response", "status": float64(200), "url": "https://x.test/a"},
								map[string]any{"type": "response", "status": float64(404), "url": "https://x.test/missing"},
							},
						},
					},
				},
			},
		},
	}

	out := GenerateNetworkActivityMarkdown(timeline)
	require.Contains(t, out, "## Step 1: load")
	require.Contains(t, out, "**GET** `https://x.test/a`")
	require.Contains(t, out, "200")
	require.Contains(t, out, "404")
	// 404 must be flagged as an error response.
	require.True(t, strings.Contains(out, "❌ 404"))
}
