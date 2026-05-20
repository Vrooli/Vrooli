// Package search hosts the Connect-RPC handler for cli-health's
// SearchService. Phase 1 returns Unimplemented; Phase 3 wires AI + text.
package search

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/search"
)

// Deps wires the seams the Connect search handler needs. Logger is the only
// seam in Phase 1; Phase 3 adds the Qdrant client and embedding service.
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

func (h *connectHandler) Search(_ context.Context, _ *connect.Request[searchv1.SearchRequest]) (*connect.Response[searchv1.SearchResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("search.Search: not yet implemented"))
}

func (h *connectHandler) Status(_ context.Context, _ *connect.Request[searchv1.StatusRequest]) (*connect.Response[searchv1.StatusResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("search.Status: not yet implemented"))
}
