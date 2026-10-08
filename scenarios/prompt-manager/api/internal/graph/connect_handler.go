// DOC: docs/internal/SEAMS.md#graph-connect-handler
package graph

import (
	"context"
	"errors"
	"math"
	"net/http"

	"connectrpc.com/connect"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph"
	"github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/graph/graph_v1connect"
)

// ConnectHandler implements the generated GraphServiceHandler. It is the
// proto-typed transport edge for the graph domain. It owns no domain logic:
// reads and regeneration delegate to the graph index and configuration stores,
// then map domain values onto their proto wire shapes.
type ConnectHandler struct {
	indexStore graphIndexProvider
	config     HealthConfigProvider
}

var _ graph_v1connect.GraphServiceHandler = (*ConnectHandler)(nil)

// NewConnectHandler constructs the Connect graph handler over the graph stores.
func NewConnectHandler(indexStore graphIndexProvider, config ...HealthConfigProvider) *ConnectHandler {
	h := &ConnectHandler{indexStore: indexStore}
	if len(config) > 0 {
		h.config = config[0]
	}
	return h
}

// NewConnectMount builds the Connect service mount (procedure path + handler)
// for registration on the existing router via connectx.RegisterServices.
func NewConnectMount(indexStore graphIndexProvider, config HealthConfigProvider, opts ...connect.HandlerOption) (string, http.Handler) {
	return graph_v1connect.NewGraphServiceHandler(NewConnectHandler(indexStore, config), opts...)
}

func (h *ConnectHandler) GetGraph(ctx context.Context, _ *connect.Request[graphv1.GetGraphRequest]) (*connect.Response[graphv1.GraphIndex], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(graphIndexToProto(idx)), nil
}

func (h *ConnectHandler) RegenerateGraph(ctx context.Context, _ *connect.Request[graphv1.RegenerateGraphRequest]) (*connect.Response[graphv1.GraphIndex], error) {
	regenerator, ok := h.indexStore.(interface{ Regenerate(context.Context) error })
	if !ok {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("graph index does not support regeneration"))
	}
	if err := regenerator.Regenerate(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(graphIndexToProto(idx)), nil
}

func (h *ConnectHandler) ListOrphanedSkills(ctx context.Context, _ *connect.Request[graphv1.ListNodesRequest]) (*connect.Response[graphv1.ListNodesResponse], error) {
	return h.nodeList(ctx, OrphanedSkills)
}

func (h *ConnectHandler) ListSkilllessAgents(ctx context.Context, _ *connect.Request[graphv1.ListNodesRequest]) (*connect.Response[graphv1.ListNodesResponse], error) {
	return h.nodeList(ctx, SkilllessAgents)
}

func (h *ConnectHandler) ListEmptyTeams(ctx context.Context, _ *connect.Request[graphv1.ListNodesRequest]) (*connect.Response[graphv1.ListNodesResponse], error) {
	return h.nodeList(ctx, EmptyTeams)
}

func (h *ConnectHandler) ListUnaffiliatedAgents(ctx context.Context, _ *connect.Request[graphv1.ListNodesRequest]) (*connect.Response[graphv1.ListNodesResponse], error) {
	return h.nodeList(ctx, UnaffiliatedAgents)
}

func (h *ConnectHandler) ListPopularNodes(ctx context.Context, req *connect.Request[graphv1.ListPopularNodesRequest]) (*connect.Response[graphv1.ListNodesResponse], error) {
	limit := int(req.Msg.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	return h.nodeList(ctx, func(g Graph) []Node { return Popular(g, limit) })
}

func (h *ConnectHandler) ListCycles(ctx context.Context, _ *connect.Request[graphv1.ListCyclesRequest]) (*connect.Response[graphv1.ListCyclesResponse], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &graphv1.ListCyclesResponse{}
	for _, cycle := range DetectCircularRefs(idx.Graph) {
		out.Cycles = append(out.Cycles, &graphv1.Cycle{NodeIds: cycle})
	}
	return connect.NewResponse(out), nil
}

func (h *ConnectHandler) GetNode(ctx context.Context, req *connect.Request[graphv1.GetNodeRequest]) (*connect.Response[graphv1.NodeDetail], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var out *graphv1.NodeDetail
	for _, n := range idx.Graph.Nodes {
		if n.ID == req.Msg.GetNodeId() {
			out = &graphv1.NodeDetail{Node: nodeToProto(n)}
			break
		}
	}
	if out == nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("graph node not found"))
	}
	for _, e := range idx.Graph.Edges {
		if e.From == req.Msg.GetNodeId() || e.To == req.Msg.GetNodeId() {
			out.AdjacentEdges = append(out.AdjacentEdges, edgeToProto(e))
		}
	}
	for _, s := range idx.Graph.HealthScores {
		if s.NodeID == req.Msg.GetNodeId() {
			out.HealthScore = healthScoreToProto(s)
			break
		}
	}
	return connect.NewResponse(out), nil
}

func (h *ConnectHandler) ListNodeEdges(ctx context.Context, req *connect.Request[graphv1.ListNodeEdgesRequest]) (*connect.Response[graphv1.ListEdgesResponse], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &graphv1.ListEdgesResponse{}
	for _, e := range idx.Graph.Edges {
		if e.From == req.Msg.GetNodeId() || e.To == req.Msg.GetNodeId() {
			out.Edges = append(out.Edges, edgeToProto(e))
		}
	}
	return connect.NewResponse(out), nil
}

func (h *ConnectHandler) GetHealthConfig(ctx context.Context, _ *connect.Request[graphv1.GetHealthConfigRequest]) (*connect.Response[graphv1.HealthConfig], error) {
	if h.config == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("graph health configuration is unavailable"))
	}
	cfg, err := h.config.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(healthConfigToProto(cfg)), nil
}

func (h *ConnectHandler) UpdateHealthConfig(ctx context.Context, req *connect.Request[graphv1.UpdateHealthConfigRequest]) (*connect.Response[graphv1.HealthConfig], error) {
	store, ok := h.config.(interface {
		Put(context.Context, HealthConfig) error
	})
	if !ok {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("graph health configuration is read-only"))
	}
	cfg := healthConfigFromProto(req.Msg.GetConfig())
	if err := ValidateHealthConfig(cfg); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := store.Put(ctx, cfg); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if regen, ok := h.indexStore.(interface{ Regenerate(context.Context) error }); ok {
		if err := regen.Regenerate(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(healthConfigToProto(cfg)), nil
}

func (h *ConnectHandler) nodeList(ctx context.Context, query func(Graph) []Node) (*connect.Response[graphv1.ListNodesResponse], error) {
	idx, err := h.indexStore.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &graphv1.ListNodesResponse{}
	for _, n := range query(idx.Graph) {
		out.Nodes = append(out.Nodes, nodeToProto(n))
	}
	return connect.NewResponse(out), nil
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
		Score:    finiteHealthValue(s.Score),
		Messages: make([]*graphv1.HealthMessage, 0, len(s.Messages)),
	}
	if len(s.Factors) > 0 {
		out.Factors = make(map[string]float64, len(s.Factors))
		for k, v := range s.Factors {
			out.Factors[k] = finiteHealthValue(v)
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
			MetricValue:    finiteHealthValue(m.MetricValue),
			Target:         m.Target,
		})
	}
	return out
}

// finiteHealthValue keeps persisted or externally sourced diagnostics inside
// protobuf's JSON-safe numeric domain. A single NaN score must not cause a
// strict UI client to reject the complete health response.
func finiteHealthValue(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return value
}

func graphIndexToProto(idx *GraphIndex) *graphv1.GraphIndex {
	out := &graphv1.GraphIndex{GeneratedAt: idx.GeneratedAt, Graph: &graphv1.Graph{}}
	for _, n := range idx.Graph.Nodes {
		out.Graph.Nodes = append(out.Graph.Nodes, nodeToProto(n))
	}
	for _, e := range idx.Graph.Edges {
		out.Graph.Edges = append(out.Graph.Edges, edgeToProto(e))
	}
	for _, s := range idx.Graph.HealthScores {
		out.Graph.HealthScores = append(out.Graph.HealthScores, healthScoreToProto(s))
	}
	return out
}

func nodeToProto(n Node) *graphv1.Node {
	return &graphv1.Node{Id: n.ID, Type: string(n.Type), Label: n.Label, Description: n.Description, Status: n.Status, Tags: n.Tags}
}

func edgeToProto(e Edge) *graphv1.Edge {
	return &graphv1.Edge{From: e.From, To: e.To, Kind: string(e.Kind), Category: string(e.Category), Command: e.Command, Subcommand: e.Subcommand, CommandText: e.CommandText, SourceFile: e.SourceFile, LineNumber: int32(e.LineNumber)}
}

func healthWeightsToProto(w HealthWeights) *graphv1.HealthWeights {
	return &graphv1.HealthWeights{OutgoingEdges: w.OutgoingEdges, IncomingEdges: w.IncomingEdges, CodeUsage: w.CodeUsage, RecentActivity: w.RecentActivity, SkillContentLength: w.SkillContentLength, AgentContextLoad: w.AgentContextLoad, TeamMemberCountBalance: w.TeamMemberCountBalance, TeamRoleCoverage: w.TeamRoleCoverage, ActionContract: w.ActionContract, ActionCommand: w.ActionCommand, ActionExamples: w.ActionExamples, ActionOwner: w.ActionOwner}
}

func healthWeightsFromProto(w *graphv1.HealthWeights) HealthWeights {
	if w == nil {
		return HealthWeights{}
	}
	return HealthWeights{OutgoingEdges: w.GetOutgoingEdges(), IncomingEdges: w.GetIncomingEdges(), CodeUsage: w.GetCodeUsage(), RecentActivity: w.GetRecentActivity(), SkillContentLength: w.GetSkillContentLength(), AgentContextLoad: w.GetAgentContextLoad(), TeamMemberCountBalance: w.GetTeamMemberCountBalance(), TeamRoleCoverage: w.GetTeamRoleCoverage(), ActionContract: w.GetActionContract(), ActionCommand: w.GetActionCommand(), ActionExamples: w.GetActionExamples(), ActionOwner: w.GetActionOwner()}
}

func healthConfigToProto(c HealthConfig) *graphv1.HealthConfig {
	return &graphv1.HealthConfig{Team: healthWeightsToProto(c.Team), Agent: healthWeightsToProto(c.Agent), Skill: healthWeightsToProto(c.Skill), Action: healthWeightsToProto(c.Action), Cli: &graphv1.CLIHealthConfig{NeutralCommands: c.CLI.NeutralCommands, ExternalToolScore: c.CLI.ExternalToolScore, ScenarioFallbackScore: c.CLI.ScenarioFallbackScore}}
}

func healthConfigFromProto(c *graphv1.HealthConfig) HealthConfig {
	if c == nil {
		return HealthConfig{}
	}
	out := HealthConfig{Team: healthWeightsFromProto(c.GetTeam()), Agent: healthWeightsFromProto(c.GetAgent()), Skill: healthWeightsFromProto(c.GetSkill()), Action: healthWeightsFromProto(c.GetAction())}
	if c.GetCli() != nil {
		out.CLI = CLIHealthConfig{NeutralCommands: c.GetCli().GetNeutralCommands(), ExternalToolScore: c.GetCli().GetExternalToolScore(), ScenarioFallbackScore: c.GetCli().GetScenarioFallbackScore()}
	}
	return out
}
