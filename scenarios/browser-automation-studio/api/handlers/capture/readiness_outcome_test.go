package capture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeclaredReadinessOutcomeAggregatesTerminalSurfaceStates(t *testing.T) {
	tests := []struct {
		name   string
		frames []map[string]any
		want   string
	}{
		{
			name: "ready wins over valid empty",
			frames: []map[string]any{
				{"step_type": "wait", "extracted_data_preview": map[string]any{"experience_surface_state": "empty"}},
				{"step_type": "wait", "extracted_data_preview": map[string]any{"experience_surface_state": "ready"}},
			},
			want: "ready",
		},
		{
			name:   "partial is reported",
			frames: []map[string]any{{"step_type": "wait", "extracted_data_preview": map[string]any{"experience_surface_state": "partial"}}},
			want:   "partial",
		},
		{
			name: "error takes precedence",
			frames: []map[string]any{
				{"step_type": "wait", "extracted_data_preview": map[string]any{"experience_surface_state": "ready"}},
				{"step_type": "wait", "extracted_data_preview": map[string]any{"experience_surface_state": "error"}},
			},
			want: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir := t.TempDir()
			raw, err := json.Marshal(map[string]any{"frames": tt.frames})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filepath.Join(outDir, "timeline.json"), raw, 0o644))

			got, err := declaredReadinessOutcome(outDir)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDeclaredReadinessOutcomeLeavesLegacyTimelineUnspecified(t *testing.T) {
	outDir := t.TempDir()
	raw, err := json.Marshal(map[string]any{"frames": []map[string]any{{"step_type": "wait"}}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "timeline.json"), raw, 0o644))

	got, err := declaredReadinessOutcome(outDir)
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestReadinessTimingSeparatesNavigationFromPostNavigationWaits(t *testing.T) {
	outDir := t.TempDir()
	raw, err := json.Marshal(map[string]any{"frames": []map[string]any{
		{"step_type": "navigate", "duration_ms": 120},
		{"step_type": "wait", "duration_ms": 35, "extracted_data_preview": map[string]any{"experience_surface_state": "ready"}},
		{"step_type": "wait", "duration_ms": 15},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(outDir, "timeline.json"), raw, 0o644))

	timing, err := readinessTiming(outDir)
	require.NoError(t, err)
	require.EqualValues(t, 120, timing.navigationMS)
	require.EqualValues(t, 50, timing.readinessWaitMS)
	require.Equal(t, "ready", timing.outcome)
}
