package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitCSV(t *testing.T) {
	require.Equal(t, []string{"api", "ui"}, splitCSV([]string{"api,ui", "ui"}))
}

func TestFirstFlag(t *testing.T) {
	require.Equal(t, "scenarios/foo", firstFlag([]string{"", "  scenarios/foo  "}))
	require.Equal(t, "", firstFlag(nil))
}
