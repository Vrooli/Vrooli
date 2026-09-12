package audit

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitCSV(t *testing.T) {
	require.Equal(t, []string{"TS_CONFIG_STRICT", "ui"}, splitCSV([]string{"TS_CONFIG_STRICT,ui", "ui"}))
}
