package explain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupName(t *testing.T) {
	require.Equal(t, "explain", GroupName)
}
