package offers

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	oc "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
)

type handlers struct {
	c oc.CatalogServiceClient
	g oc.GatesServiceClient
	b oc.BoardServiceClient
	s oc.SpaceServiceClient
}

func newHandlers(a *cliapp.ScenarioApp) *handlers {
	hc, base := cliapp.NewConnectHTTPClient(a)
	return &handlers{c: oc.NewCatalogServiceClient(hc, base), g: oc.NewGatesServiceClient(hc, base), b: oc.NewBoardServiceClient(hc, base), s: oc.NewSpaceServiceClient(hc, base)}
}

func (h *handlers) list(_ cliapp.OperationContext) (*offerspb.ListNodesResponse, error) {
	r, e := h.c.ListNodes(context.Background(), connect.NewRequest(&offerspb.ListNodesRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) catalogImport(c cliapp.OperationContext) (*offerspb.ImportCatalogResponse, error) {
	mode := offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED
	if strings.EqualFold(c.Flag("source-mode"), "fixture") {
		mode = offerspb.SourceMode_SOURCE_MODE_FIXTURE
	}
	r, err := h.c.ImportCatalog(context.Background(), connect.NewRequest(&offerspb.ImportCatalogRequest{SourcePath: c.Flag("source-path"), SourceMode: mode, Apply: strings.EqualFold(c.Flag("apply"), "true"), Actor: "operator"}))
	if err != nil && r == nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("catalog import returned no report")
	}
	return r.Msg, err
}

func (h *handlers) catalogMerge(c cliapp.OperationContext) (*offerspb.MergeNodesResponse, error) {
	dryRun := true
	if raw := strings.TrimSpace(c.Flag("dry-run")); raw != "" {
		dryRun = !strings.EqualFold(raw, "false")
	}
	r, err := h.c.MergeNodes(context.Background(), connect.NewRequest(&offerspb.MergeNodesRequest{SurvivingId: c.Flag("surviving-id"), DuplicateId: c.Flag("duplicate-id"), Actor: "operator", DryRun: dryRun}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) catalogVerify(c cliapp.OperationContext) (*offerspb.VerifyCatalogResponse, error) {
	mode := offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED
	if strings.EqualFold(c.Flag("source-mode"), "fixture") {
		mode = offerspb.SourceMode_SOURCE_MODE_FIXTURE
	}
	r, err := h.c.VerifyCatalog(context.Background(), connect.NewRequest(&offerspb.VerifyCatalogRequest{SourcePath: c.Flag("source-path"), SourceMode: mode}))
	if err != nil && r == nil {
		return nil, err
	}
	if r == nil {
		return nil, fmt.Errorf("catalog verify returned no report")
	}
	return r.Msg, nil
}

func (h *handlers) space(c cliapp.OperationContext) (*offerspb.SpaceResponse, error) {
	r, err := h.s.GetProjection(context.Background(), connect.NewRequest(&offerspb.ProjectionRequest{Projection: c.Flag("projection")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) create(c cliapp.OperationContext) (*offerspb.CreateNodeResponse, error) {
	r, e := h.c.CreateNode(context.Background(), connect.NewRequest(&offerspb.CreateNodeRequest{Name: c.Flag("name"), Kind: offerspb.NodeKind_OFFER, ActualAccountId: c.Flag("actual-account-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) transition(c cliapp.OperationContext) (*offerspb.TransitionResponse, error) {
	r, e := h.c.Transition(context.Background(), connect.NewRequest(&offerspb.TransitionRequest{NodeId: c.Flag("node-id"), Status: parseStatus(c.Flag("status")), Actor: c.Flag("actor")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) edge(c cliapp.OperationContext) (*offerspb.CreateEdgeResponse, error) {
	r, e := h.c.CreateEdge(context.Background(), connect.NewRequest(&offerspb.CreateEdgeRequest{Edge: &offerspb.Edge{FromId: c.Flag("from-id"), ToId: c.Flag("to-id"), Kind: c.Flag("kind"), Currency: strings.ToUpper(c.Flag("currency"))}}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) edgesList(_ cliapp.OperationContext) (*offerspb.ListEdgesResponse, error) {
	r, e := h.c.ListEdges(context.Background(), connect.NewRequest(&offerspb.ListEdgesRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) trigger(c cliapp.OperationContext) (*offerspb.DeclareTriggerResponse, error) {
	r, e := h.g.DeclareTrigger(context.Background(), connect.NewRequest(&offerspb.DeclareTriggerRequest{Trigger: &offerspb.Trigger{NodeId: c.Flag("node-id"), FactName: c.Flag("fact-name"), Operator: c.Flag("operator"), Threshold: parseFloat(c.Flag("threshold"))}}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) fact(c cliapp.OperationContext) (*offerspb.AddFactResponse, error) {
	r, e := h.g.AddFact(context.Background(), connect.NewRequest(&offerspb.AddFactRequest{Fact: &offerspb.Fact{Name: c.Flag("name"), Value: parseFloat(c.Flag("value")), StaleAfterDays: int32(parseInt(c.Flag("stale-after-days")))}}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) evaluate(_ cliapp.OperationContext) (*offerspb.EvaluateResponse, error) {
	r, e := h.g.Evaluate(context.Background(), connect.NewRequest(&offerspb.EvaluateRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) promote(c cliapp.OperationContext) (*offerspb.PromoteResponse, error) {
	r, e := h.g.Promote(context.Background(), connect.NewRequest(&offerspb.PromoteRequest{NodeId: c.Flag("node-id"), Actor: c.Flag("actor"), Role: c.Flag("role")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) proposals(c cliapp.OperationContext) (*offerspb.ListProposalsResponse, error) {
	r, e := h.g.ListProposals(context.Background(), connect.NewRequest(&offerspb.ListProposalsRequest{NodeId: c.Flag("node-id"), Status: parseFilterStatus(c.Flag("status"))}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) board(_ cliapp.OperationContext) (*offerspb.BoardResponse, error) {
	r, e := h.b.GetBoard(context.Background(), connect.NewRequest(&offerspb.ProjectionRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func listReport(_ cliapp.OperationContext, m *offerspb.ListNodesResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d node(s).", len(m.Nodes))}, ResultsHeading: "Offer graph", Results: nodeStrings(m.Nodes)}
}

func createReport(_ cliapp.OperationContext, m *offerspb.CreateNodeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created node " + m.Node.Id + "."}}
}

func transitionReport(_ cliapp.OperationContext, m *offerspb.TransitionResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Transitioned " + m.Node.Id + " to " + m.Node.Status.String() + "."}}
}

func evaluateReport(_ cliapp.OperationContext, m *offerspb.EvaluateResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Evaluated %d candidate trigger(s).", len(m.Evaluations))}}
}

func promoteReport(_ cliapp.OperationContext, m *offerspb.PromoteResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created proposal " + m.Proposal.Id + "; operator approval remains required."}}
}

func proposalsReport(_ cliapp.OperationContext, m *offerspb.ListProposalsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d promotion proposal(s).", len(m.Proposals))}, ResultsHeading: "Proposals", Results: mapStrings(len(m.Proposals), func(i int) string {
		p := m.Proposals[i]
		return fmt.Sprintf("%s — %s requested by %s (%d decline(s)); evidence=%s", p.Id, p.RequestedStatus.String(), p.Actor, len(p.DeclineHistory), p.EvidenceReference)
	})}
}

func edgeReport(_ cliapp.OperationContext, m *offerspb.CreateEdgeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created typed edge " + m.Edge.Id + "."}}
}

func edgesListReport(_ cliapp.OperationContext, m *offerspb.ListEdgesResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d graph edge(s).", len(m.Edges))}, ResultsHeading: "Edges", Results: mapStrings(len(m.Edges), func(i int) string {
		return fmt.Sprintf("%s — %s -> %s [%s]", m.Edges[i].Id, m.Edges[i].FromId, m.Edges[i].ToId, m.Edges[i].Kind)
	})}
}

func triggerReport(_ cliapp.OperationContext, m *offerspb.DeclareTriggerResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Declared trigger " + m.Trigger.Id + "."}}
}

func factReport(_ cliapp.OperationContext, m *offerspb.AddFactResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Recorded fact " + m.Fact.Name + "."}}
}

func boardReport(_ cliapp.OperationContext, m *offerspb.BoardResponse) cliapp.ListReport {
	r := make([]string, len(m.Entries))
	for i, e := range m.Entries {
		r[i] = fmt.Sprintf("%s — %s [%s]", e.Title, e.RankReason, e.Status.String())
	}
	return cliapp.ListReport{Summary: []string{"Offer Desk board"}, ResultsHeading: "Priority", Results: r}
}

func catalogImportReport(_ cliapp.OperationContext, m *offerspb.ImportCatalogResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Catalog import inspected %d source file(s), applied=%t, findings=%d.", len(m.Files), m.Applied, m.TotalFindings)}}
}

func catalogMergeReport(_ cliapp.OperationContext, m *offerspb.MergeNodesResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Merge %s: edges=%d triggers=%d evaluations=%d proposals=%d findings=%d collapsed_edges=%d.", m.Surviving.Id, m.MovedEdges, m.MovedTriggers, m.MovedEvaluations, m.MovedProposals, m.MovedFindings, len(m.CollapsedEdgeIds))}}
}

func catalogVerifyReport(_ cliapp.OperationContext, m *offerspb.VerifyCatalogResponse) cliapp.ListReport {
	summary := []string{fmt.Sprintf("Catalog verification reconciled=%t, comparable=%t, files=%d, drift=%d, duplicate_identities=%d, orphan_edges=%d, extra_nodes=%d.", m.Reconciled, m.Comparable, len(m.Files), m.TotalDrift, len(m.DuplicateIdentities), len(m.OrphanEdgeIds), len(m.ExtraNodeIds))}
	// A bare "reconciled=true" reads as "the import was verified" even when no
	// count comparison ran. The reason is printed beside it so a human cannot
	// draw that conclusion from this output alone.
	if !m.Comparable && m.NotComparableReason != "" {
		summary = append(summary, "Not comparable: "+m.NotComparableReason)
	}
	return cliapp.ListReport{Summary: summary, ResultsHeading: "Reconciliation", Results: mapStrings(len(m.Files), func(i int) string {
		file := m.Files[i]
		return fmt.Sprintf("%s — expected=%d live=%d", file.Path, file.Expected, file.Live)
	})}
}

func catalogVerifyOutcome(m *offerspb.VerifyCatalogResponse) error {
	if m == nil || m.Reconciled {
		return nil
	}
	return fmt.Errorf("catalog verification found drift: total_drift=%d", m.TotalDrift)
}

func spaceReport(_ cliapp.OperationContext, m *offerspb.SpaceResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("%s projection (%s confidence)", m.Projection, m.DenominatorConfidence)}, ResultsHeading: "Obligations", Results: mapStrings(len(m.Cells), func(i int) string {
		return fmt.Sprintf("%s — %s [%s]", m.Cells[i].Id, m.Cells[i].Question, m.Cells[i].Status)
	})}
}

func parseStatus(v string) offerspb.Status {
	switch strings.ToLower(v) {
	case "candidate":
		return offerspb.Status_CANDIDATE
	case "trigger-met":
		return offerspb.Status_TRIGGER_MET
	case "active":
		return offerspb.Status_ACTIVE
	case "shipped":
		return offerspb.Status_SHIPPED
	case "retired":
		return offerspb.Status_RETIRED
	case "proposed":
		return offerspb.Status_PROPOSED
	default:
		return offerspb.Status_IDEA
	}
}

func parseFilterStatus(v string) offerspb.Status {
	if strings.TrimSpace(v) == "" {
		return offerspb.Status_STATUS_UNSPECIFIED
	}
	return parseStatus(v)
}
func parseFloat(v string) float64 { var n float64; _, _ = fmt.Sscan(v, &n); return n }
func parseInt(v string) int64     { var n int64; _, _ = fmt.Sscan(v, &n); return n }
func nodeStrings(ns []*offerspb.Node) []string {
	r := make([]string, len(ns))
	for i, n := range ns {
		r[i] = fmt.Sprintf("%s — %s [%s]", n.Id, n.Name, n.Status.String())
	}
	return r
}

func mapStrings(n int, f func(int) string) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = strings.TrimSpace(f(i))
	}
	return out
}
