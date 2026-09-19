package graph_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"typescript-code-graph/internal/graph"
	"typescript-code-graph/internal/sidecar"
	sidecarmocks "typescript-code-graph/internal/sidecar/mocks"
)

func newService(t *testing.T, client sidecar.SidecarClient) *graph.Service {
	t.Helper()
	return graph.NewService(client, graph.NewPathMutex())
}

func TestService_Extract_RejectsEmptyPath(t *testing.T) {
	svc := newService(t, &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady})
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "   "})
	require.Error(t, err)

	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorInvalidInput, ee.Kind)
}

func TestService_Extract_RejectsRelativePath(t *testing.T) {
	svc := newService(t, &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusReady})
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "relative/path"})
	require.Error(t, err)
	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorInvalidInput, ee.Kind)
}

func TestService_Extract_SidecarUnavailable(t *testing.T) {
	svc := newService(t, &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusUnhealthy})
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs/path"})
	require.Error(t, err)

	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorSidecarUnavailable, ee.Kind)
}

func TestService_Extract_PermanentlyUnhealthy(t *testing.T) {
	svc := newService(t, &sidecarmocks.FakeSidecarClient{StatusValue: sidecar.StatusPermanentlyUnhealthy})
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs/path"})
	require.Error(t, err)
	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorSidecarUnavailable, ee.Kind)
}

func TestService_Extract_SidecarExtractError_NoTsConfig(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, &sidecar.ExtractError{Kind: "no_tsconfig_found", Message: "no tsconfig"}
		},
	}
	svc := newService(t, fake)
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs"})
	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorNoTsConfig, ee.Kind)
}

func TestService_Extract_SidecarExtractError_Workspace(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, &sidecar.ExtractError{Kind: "workspace_unsupported"}
		},
	}
	svc := newService(t, fake)
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs"})
	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorWorkspaceUnsupported, ee.Kind)
}

func TestService_Extract_SidecarUnavailableSentinel(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, sidecar.ErrSidecarUnavailable
		},
	}
	svc := newService(t, fake)
	_, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs"})
	var ee graph.ExtractError
	require.ErrorAs(t, err, &ee)
	require.Equal(t, graph.ExtractErrorSidecarUnavailable, ee.Kind)
}

func TestService_Extract_HappyPath_NormalizesAndHashes(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			require.Equal(t, "/abs/proj", p)
			return sidecar.ExtractResult{
				Graph: sidecar.RawGraph{
					Nodes: []sidecar.RawNode{
						{ID: "z", Kind: 1, Name: "z.ts", Path: "src/z.ts"},
						{ID: "a", Kind: 201, Name: "Btn", Path: "src/Btn.tsx"},
					},
					Edges: []sidecar.RawEdge{
						{ID: "e1", Kind: 1, FromNodeID: "a", ToNodeID: "z"},
					},
				},
				Warnings:  []sidecar.Warning{{Kind: 1, Message: "x", File: "src/z.ts"}},
				RequestID: "req-happy-1",
			}, nil
		},
	}
	svc := newService(t, fake)
	out, err := svc.Extract(context.Background(), graph.ExtractInput{ProjectPath: "/abs/proj"})
	require.NoError(t, err)
	require.Len(t, out.Graph.Nodes, 2)
	require.Equal(t, "a", out.Graph.Nodes[0].ID, "nodes must be sorted by id")
	require.Equal(t, graph.NodeKindComponent, out.Graph.Nodes[0].Kind)
	require.NotEmpty(t, out.GraphHash)
	require.Equal(t, "req-happy-1", out.SidecarRequestID, "sidecar_request_id must be threaded through")
	require.Len(t, out.Warnings, 1)
	require.Equal(t, graph.WarningKindParseError, out.Warnings[0].Kind)
	require.Equal(t, 1, fake.ExtractCalls)
}

func TestService_Extract_PropagatesContextCancel(t *testing.T) {
	fake := &sidecarmocks.FakeSidecarClient{
		StatusValue: sidecar.StatusReady,
		ExtractFn: func(ctx context.Context, p string) (sidecar.ExtractResult, error) {
			return sidecar.ExtractResult{}, context.Canceled
		},
	}
	svc := newService(t, fake)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Extract(ctx, graph.ExtractInput{ProjectPath: "/abs"})
	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled) || isExtractInternal(err))
}

func isExtractInternal(err error) bool {
	var ee graph.ExtractError
	if errors.As(err, &ee) {
		return ee.Kind == graph.ExtractErrorInternal
	}
	return false
}
