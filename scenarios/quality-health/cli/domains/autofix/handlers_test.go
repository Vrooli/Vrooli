package autofix

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitCSV(t *testing.T) {
	require.Equal(t, []string{"TS_CONFIG_STRICT", "ESLINT_SAFETY_RULES"}, splitCSV([]string{"TS_CONFIG_STRICT,ESLINT_SAFETY_RULES"}))
}
