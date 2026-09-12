package coverage

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetentionFloorMovesWithSetpointWindow(t *testing.T) {
	root := t.TempDir()
	setpointDir := filepath.Join(root, "scenarios", "infrastructure-manager", "setpoint")
	require.NoError(t, os.MkdirAll(setpointDir, 0o755))
	path := filepath.Join(setpointDir, "reliability-setpoint.json")
	write := func(sustain string, margin int) {
		require.NoError(t, os.WriteFile(path, []byte(`{
  "schema_version": "1",
  "constants": {"retention_margin_days": `+strconv.Itoa(margin)+`},
  "bars": [{"id":"bar-1","cell_ref":"availability/A1","projection":"availability","decision_ref":"decisions/test","unit":"percent","min":99.5,"gradeable":true,"sustain":"`+sustain+`"}]
}`), 0o644))
	}
	write("24h", 2)
	first, err := RetentionFloor(root)
	require.NoError(t, err)
	require.Equal(t, 3*24*time.Hour, first)
	write("7d", 2)
	second, err := RetentionFloor(root)
	require.NoError(t, err)
	require.Equal(t, 9*24*time.Hour, second)
	require.Greater(t, second, first)
}
