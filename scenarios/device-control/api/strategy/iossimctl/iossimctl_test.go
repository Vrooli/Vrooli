package iossimctl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewWithUDIDKeepsResolvedConfiguration(t *testing.T) {
	adapter := NewWithUDID(" simulator-123 ")
	require.Equal(t, "simulator-123", adapter.udid)
}
