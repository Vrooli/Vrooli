package graph_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
)

func TestErrorToConnectCode_NilIsZero(t *testing.T) {
	require.Equal(t, connect.Code(0), graph.ErrorToConnectCode(nil))
}

func TestErrorToConnectCode_MapsKinds(t *testing.T) {
	cases := []struct {
		kind graph.ExtractErrorKind
		want connect.Code
	}{
		{graph.ExtractErrorNoTsConfig, connect.CodeInvalidArgument},
		{graph.ExtractErrorMultipleTsConfig, connect.CodeInvalidArgument},
		{graph.ExtractErrorInvalidInput, connect.CodeInvalidArgument},
		{graph.ExtractErrorWorkspaceUnsupported, connect.CodeUnimplemented},
		{graph.ExtractErrorPathUnreadable, connect.CodeNotFound},
		{graph.ExtractErrorSidecarUnavailable, connect.CodeUnavailable},
		{graph.ExtractErrorSidecarTimeout, connect.CodeDeadlineExceeded},
		{graph.ExtractErrorInternal, connect.CodeInternal},
	}
	for _, tc := range cases {
		got := graph.ErrorToConnectCode(graph.ExtractError{Kind: tc.kind})
		require.Equal(t, tc.want, got, "kind %s", tc.kind)
	}
}

func TestErrorToConnectCode_UnknownIsInternal(t *testing.T) {
	require.Equal(t, connect.CodeInternal,
		graph.ErrorToConnectCode(errors.New("boom")))
}

func TestErrorToConnectCode_DeadlineExceeded(t *testing.T) {
	require.Equal(t, connect.CodeDeadlineExceeded,
		graph.ErrorToConnectCode(context.DeadlineExceeded))
}

func TestErrorToConnectCode_Canceled(t *testing.T) {
	require.Equal(t, connect.CodeCanceled,
		graph.ErrorToConnectCode(context.Canceled))
}

func TestToConnectError_WrapsAndPreservesCode(t *testing.T) {
	in := graph.ExtractError{Kind: graph.ExtractErrorWorkspaceUnsupported, Message: "nope"}
	out := graph.ToConnectError(in)
	require.Error(t, out)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(out))
}

func TestToConnectError_NilIn_NilOut(t *testing.T) {
	require.Nil(t, graph.ToConnectError(nil))
}
