package workspace

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateProfileIntFlagRejectsNarrowingOverflow(t *testing.T) {
	// The parser used by updateCall must reject values that cannot be represented
	// by the proto int32 fields before any narrowing conversion occurs.
	_, err := parseInt32Flag("2147483648", "daily-capacity-minutes")
	require.Error(t, err)
	value, err := parseInt32Flag("1440", "daily-capacity-minutes")
	require.NoError(t, err)
	require.Equal(t, int32(1440), value)
}
