// Package reindex hosts the Connect-RPC handler for cli-health's
// ReindexService. Phase 1 returns Unimplemented; Phase 3 wires the job
// orchestrator + Qdrant upserts/deletes.
package reindex

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/reindex"
)

// Deps wires the seams the Connect reindex handler needs. Logger is the
// only seam in Phase 1; Phase 3 adds the job store and Qdrant client.
type Deps struct {
	Logger *log.Logger
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Reindex(_ context.Context, _ *connect.Request[reindexv1.ReindexRequest]) (*connect.Response[reindexv1.ReindexResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("reindex.Reindex: not yet implemented"))
}

func (h *connectHandler) ReindexStatus(_ context.Context, _ *connect.Request[reindexv1.ReindexStatusRequest]) (*connect.Response[reindexv1.ReindexStatusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("reindex.ReindexStatus: not yet implemented"))
}

func (h *connectHandler) ReindexCancel(_ context.Context, _ *connect.Request[reindexv1.ReindexCancelRequest]) (*connect.Response[reindexv1.ReindexCancelResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("reindex.ReindexCancel: not yet implemented"))
}
