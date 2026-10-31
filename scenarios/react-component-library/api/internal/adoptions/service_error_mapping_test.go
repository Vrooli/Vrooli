package adoptions_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"react-component-library/internal/adoptions"
)

func TestToConnectErrorMapsBatchDependencyConflictToFailedPrecondition(t *testing.T) {
	err := adoptions.ErrBatchDependencyConflict{Dependency: "rcl:Shared", FirstRoot: "first", FirstVersion: "1.0.0", SecondRoot: "second", SecondVersion: "2.0.0"}
	connectErr := adoptions.ToConnectError(err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(connectErr))
	require.Contains(t, connectErr.Error(), "rcl:Shared")
}
