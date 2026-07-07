// DOC: docs/internal/SEAMS.md#graph-connect-handler
package graph

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
)

// ConnectHandler implements the generated GraphServiceHandler. It is the
// proto-typed transport edge for the graph health read: it owns no domain
// logic, delegating to the same graphIndexProvider the REST GetHealthScores
// handler uses, and maps the domain HealthScore/HealthMessage values onto their
// proto wire shapes. The legacy REST GET /api/v1/graph/health route stays live
// alongside this Connect surface (additive, not a migration).
type ConnectHandler struct {
	indexStore graphIndexProvider
}

var _ graph_v1connect.GraphServiceHandler = (*ConnectHandler)(nil)

// NewConnectHandler constructs the Connect graph handler over the same index
// store the REST handlers read.
func NewConnectHandler(indexStore graphIndexProvider) *ConnectHandler {
	return &ConnectHandler{indexStore: indexStore}
}

// NewConnectMount builds the Connect service mount (procedure path + handler)
// for registration on the existing router via connectx.RegisterServices.
func NewConnectMount(indexStore graphIndexProvider, opts ...connect.HandlerOption) (string, http.Handler) {
	return graph_v1connect.NewGraphServiceHandler(NewConnectHandler(indexStore), opts...)
}

// GetHealthScores returns the current graph index's per-node health scores,
// mirroring the legacy REST handler's body.
func (h *ConnectHandler) GetHealthScores(ctx context.Context, _ *connect.Request[graphv1.GetHealthScoresRequest]) (*connect.Response[graphv1.GetHealthScoresResponse], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &graphv1.GetHealthScoresResponse{
		Scores: make([]*graphv1.HealthScore, 0, len(idx.Graph.HealthScores)),
	}
	for _, s := range idx.Graph.HealthScores {
		resp.Scores = append(resp.Scores, healthScoreToProto(s))
	}
	return connect.NewResponse(resp), nil
}

// healthScoreToProto maps a domain HealthScore onto its proto wire shape.
func healthScoreToProto(s HealthScore) *graphv1.HealthScore {
	out := &graphv1.HealthScore{
		NodeId:   s.NodeID,
		Score:    s.Score,
		Messages: make([]*graphv1.HealthMessage, 0, len(s.Messages)),
	}
	if len(s.Factors) > 0 {
		out.Factors = make(map[string]float64, len(s.Factors))
		for k, v := range s.Factors {
			out.Factors[k] = v
		}
	}
	for _, m := range s.Messages {
		out.Messages = append(out.Messages, &graphv1.HealthMessage{
			Key:            m.Key,
			Severity:       m.Severity,
			Factor:         m.Factor,
			Summary:        m.Summary,
			Detail:         m.Detail,
			Recommendation: m.Recommendation,
			MetricValue:    m.MetricValue,
			Target:         m.Target,
		})
	}
	return out
}
