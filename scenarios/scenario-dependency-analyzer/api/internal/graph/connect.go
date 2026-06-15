package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"

	"scenario-dependency-analyzer/internal/interfacegraph"
	"scenario-dependency-analyzer/internal/store"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph"
	graphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/graph/graph_v1connect"
)

type ConnectOptions struct {
	CacheTTL     time.Duration
	BuildTimeout time.Duration
}

// RegisterConnectRoutes mounts the durable Connect interface graph contract.
func RegisterConnectRoutes(router *gin.Engine, scenariosDir func() string, graphStore *store.Store, opts ConnectOptions) {
	connectPath, connectHandler := graphconnect.NewInterfaceGraphServiceHandler(&connectHandler{
		scenariosDir: scenariosDir,
		store:        graphStore,
		opts:         opts.withDefaults(),
	})
	router.Any(connectPath+"*path", gin.WrapH(connectHandler))
}

type connectHandler struct {
	scenariosDir func() string
	store        *store.Store
	opts         ConnectOptions
}

func (h *connectHandler) DescribeInterfaceGraph(ctx context.Context, req *connect.Request[graphv1.DescribeInterfaceGraphRequest]) (*connect.Response[graphv1.DescribeInterfaceGraphResponse], error) {
	if h == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("interface graph handler is not configured"))
	}

	msg := req.Msg
	if msg == nil {
		msg = &graphv1.DescribeInterfaceGraphRequest{}
	}

	buildReq := interfacegraph.BuildRequest{
		Scenarios:       msg.GetScenarios(),
		Limit:           msg.GetLimit(),
		RepoRoot:        filepath.Dir(h.resolveScenariosDir()),
		StabilityFilter: msg.GetStabilityFilter(),
		LanguageFilter:  msg.GetLanguageFilter(),
		MaxScenarioHops: msg.GetMaxScenarioHops(),
	}
	graph, computedAt, err := h.describeInterfaceGraph(ctx, buildReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("describe interface graph: %w", err))
	}

	resp := &graphv1.DescribeInterfaceGraphResponse{Graph: interfaceGraphToProto(graph)}
	if !computedAt.IsZero() {
		resp.ComputedAt = computedAt.UTC().Format(time.RFC3339Nano)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) describeInterfaceGraph(ctx context.Context, req interfacegraph.BuildRequest) (interfacegraph.Graph, time.Time, error) {
	cacheReq := req
	if req.MaxScenarioHops > 0 && len(req.Scenarios) > 0 {
		cacheReq.Scenarios = nil
		cacheReq.Limit = 0
		cacheReq.MaxScenarioHops = 0
	}
	signature := graphCacheSignature(cacheReq)
	if h.store != nil {
		if entry, ok, err := h.store.LoadInterfaceGraphCache(signature); err != nil {
			return interfacegraph.Graph{}, time.Time{}, err
		} else if ok && h.opts.CacheTTL > 0 && time.Since(entry.ComputedAt) <= h.opts.CacheTTL {
			graph := entry.Graph
			if req.MaxScenarioHops > 0 && len(req.Scenarios) > 0 {
				graph = graph.Neighborhood(req.Scenarios, int(req.MaxScenarioHops))
			}
			return graph, entry.ComputedAt, nil
		}
	}
	buildCtx := ctx
	cancel := func() {}
	if h.opts.BuildTimeout > 0 {
		buildCtx, cancel = context.WithTimeout(ctx, h.opts.BuildTimeout)
	}
	defer cancel()
	builder := interfacegraph.NewBuilder(
		interfacegraph.NewProtoHealthClient(nil, nil),
		interfacegraph.NewCodeFactsClient(nil, nil),
	)
	graph, err := builder.Build(buildCtx, cacheReq)
	if err != nil {
		return interfacegraph.Graph{}, time.Time{}, err
	}
	computedAt := time.Now().UTC()
	if h.store != nil {
		if err := h.store.StoreInterfaceGraphCache(store.InterfaceGraphCacheEntry{
			Signature:  signature,
			Graph:      graph,
			ComputedAt: computedAt,
		}); err != nil {
			return interfacegraph.Graph{}, time.Time{}, err
		}
	}
	if req.MaxScenarioHops > 0 && len(req.Scenarios) > 0 {
		graph = graph.Neighborhood(req.Scenarios, int(req.MaxScenarioHops))
	}
	return graph, computedAt, nil
}

func (h *connectHandler) resolveScenariosDir() string {
	if h.scenariosDir == nil {
		return ""
	}
	return strings.TrimSpace(h.scenariosDir())
}

func (opts ConnectOptions) withDefaults() ConnectOptions {
	if opts.CacheTTL == 0 {
		opts.CacheTTL = 5 * time.Minute
	}
	if opts.BuildTimeout == 0 {
		opts.BuildTimeout = 90 * time.Second
	}
	return opts
}

func graphCacheSignature(req interfacegraph.BuildRequest) string {
	scenarios := append([]string(nil), req.Scenarios...)
	languages := append([]string(nil), req.LanguageFilter...)
	sort.Strings(scenarios)
	sort.Strings(languages)
	h := sha256.New()
	for _, part := range []string{
		req.RepoRoot,
		fmt.Sprint(req.Limit),
		req.StabilityFilter,
		strings.Join(scenarios, ","),
		strings.Join(languages, ","),
	} {
		_, _ = h.Write([]byte(part))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func interfaceGraphToProto(in interfacegraph.Graph) *graphv1.InterfaceGraph {
	out := &graphv1.InterfaceGraph{
		Nodes:  make([]*graphv1.GraphNode, 0, len(in.Nodes)),
		Edges:  make([]*graphv1.GraphEdge, 0, len(in.Edges)),
		Errors: make([]*graphv1.GraphError, 0, len(in.Errors)),
	}
	for _, node := range in.Nodes {
		out.Nodes = append(out.Nodes, &graphv1.GraphNode{Scenario: node.Scenario})
	}
	for _, edge := range in.Edges {
		item := &graphv1.GraphEdge{
			FromScenario:   edge.FromScenario,
			ToScenario:     edge.ToScenario,
			TransportWorld: edge.TransportWorld,
			Stability:      append([]string(nil), edge.Stability...),
			Evidence:       make([]*graphv1.GraphEvidence, 0, len(edge.Evidence)),
		}
		for _, evidence := range edge.Evidence {
			item.Evidence = append(item.Evidence, evidenceToProto(evidence))
		}
		out.Edges = append(out.Edges, item)
	}
	for _, graphErr := range in.Errors {
		out.Errors = append(out.Errors, &graphv1.GraphError{
			Source:   graphErr.Source,
			Scenario: graphErr.Scenario,
			Message:  graphErr.Message,
		})
	}
	return out
}

func evidenceToProto(in interfacegraph.Evidence) *graphv1.GraphEvidence {
	return &graphv1.GraphEvidence{
		Source:     evidenceSourceToProto(in.Source),
		ImportPath: in.ImportPath,
		FromFile:   in.FromFile,
		ToFile:     in.ToFile,
		Path:       in.Path,
		Analyzer:   in.Analyzer,
	}
}

func evidenceSourceToProto(source string) graphv1.EvidenceSource {
	switch source {
	case interfacegraph.EvidenceProtoImport:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_PROTO_IMPORT
	case interfacegraph.EvidenceGoImport:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_GO_IMPORT
	default:
		return graphv1.EvidenceSource_EVIDENCE_SOURCE_UNSPECIFIED
	}
}

var _ graphconnect.InterfaceGraphServiceHandler = (*connectHandler)(nil)
