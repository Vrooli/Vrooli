package offers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/vrooli/cli-core/cliapp"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	oc "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
)

type handlers struct {
	c oc.CatalogServiceClient
	g oc.GatesServiceClient
	b oc.BoardServiceClient
	l oc.ReleaseLadderServiceClient
	s oc.SpaceServiceClient
}

func newHandlers(a *cliapp.ScenarioApp) *handlers {
	hc, base := cliapp.NewConnectHTTPClient(a)
	return &handlers{c: oc.NewCatalogServiceClient(hc, base), g: oc.NewGatesServiceClient(hc, base), b: oc.NewBoardServiceClient(hc, base), l: oc.NewReleaseLadderServiceClient(hc, base), s: oc.NewSpaceServiceClient(hc, base)}
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
	r, err := h.c.ImportCatalog(context.Background(), connect.NewRequest(&offerspb.ImportCatalogRequest{SourcePath: c.Flag("source-path"), SourceMode: mode, Apply: strings.EqualFold(c.Flag("apply"), "true"), Actor: c.Flag("actor")}))
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
	r, err := h.c.MergeNodes(context.Background(), connect.NewRequest(&offerspb.MergeNodesRequest{SurvivingId: c.Flag("surviving-id"), DuplicateId: c.Flag("duplicate-id"), Actor: c.Flag("actor"), Reason: c.Flag("reason"), DryRun: dryRun}))
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
	// Kind was previously hardcoded to OFFER, so no deliverable, channel,
	// variant, or revenue-line node could be created outside the importer.
	kind, err := parseKind(c.Flag("kind"))
	if err != nil {
		return nil, err
	}
	status := offerspb.Status_IDEA
	if raw := strings.TrimSpace(c.Flag("status")); raw != "" {
		status = parseStatus(raw)
	}
	r, e := h.c.CreateNode(context.Background(), connect.NewRequest(&offerspb.CreateNodeRequest{
		Name: c.Flag("name"), Kind: kind, Status: status, ActualAccountId: c.Flag("actual-account-id"), DeliverableClass: parseDeliverableClass(c.Flag("class")), FinishBar: parseFinishBar(c.Flag("finish-bar")), Actor: c.Flag("actor"), Reason: c.Flag("reason"),
	}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) setClass(c cliapp.OperationContext) (*offerspb.SetDeliverableClassResponse, error) {
	r, err := h.c.SetDeliverableClass(context.Background(), connect.NewRequest(&offerspb.SetDeliverableClassRequest{NodeId: c.Flag("node-id"), DeliverableClass: parseDeliverableClass(c.Flag("class")), FinishBar: parseFinishBar(c.Flag("finish-bar")), Actor: c.Flag("actor"), Reason: c.Flag("reason")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) meters(_ cliapp.OperationContext) (*offerspb.MeterInventoryResponse, error) {
	r, err := h.c.GetMeterInventory(context.Background(), connect.NewRequest(&offerspb.MeterInventoryRequest{}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

// parseKind refuses an unrecognised kind rather than defaulting, because a
// silent default would file a record under the wrong half of the graph.
func parseKind(v string) (offerspb.NodeKind, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "offer":
		return offerspb.NodeKind_OFFER, nil
	case "variant":
		return offerspb.NodeKind_VARIANT, nil
	case "channel":
		return offerspb.NodeKind_CHANNEL, nil
	case "revenue-line", "revenue_line":
		return offerspb.NodeKind_REVENUE_LINE, nil
	case "deliverable":
		return offerspb.NodeKind_DELIVERABLE, nil
	case "ramp":
		return offerspb.NodeKind_RAMP, nil
	case "stream":
		return offerspb.NodeKind_STREAM, nil
	case "audience":
		return offerspb.NodeKind_AUDIENCE, nil
	default:
		return offerspb.NodeKind_OFFER, fmt.Errorf("unknown node kind %q; expected offer, variant, channel, revenue-line, deliverable, ramp, stream, or audience", v)
	}
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
	trigger := &offerspb.Trigger{NodeId: c.Flag("node-id"), FactName: c.Flag("fact-name"), Operator: c.Flag("operator"), Threshold: parseFloat(c.Flag("threshold"))}
	// The store already supports multi-clause triggers, but the CLI could only
	// declare a single clause — so a real revisit condition with more than one
	// part could not be filed as a record at all.
	if raw := strings.TrimSpace(c.Flag("clauses")); raw != "" {
		var clauses []struct {
			FactName  string  `json:"fact_name"`
			Operator  string  `json:"operator"`
			Threshold float64 `json:"threshold"`
		}
		if err := json.Unmarshal([]byte(raw), &clauses); err != nil {
			return nil, fmt.Errorf("parse --clauses as JSON array of {fact_name,operator,threshold}: %w", err)
		}
		for _, clause := range clauses {
			trigger.Clauses = append(trigger.Clauses, &offerspb.TriggerClause{FactName: clause.FactName, Operator: clause.Operator, Threshold: clause.Threshold})
		}
	}
	switch strings.ToLower(strings.TrimSpace(c.Flag("composition"))) {
	case "", "all":
		trigger.Composition = offerspb.TriggerComposition_ALL
	case "any":
		trigger.Composition = offerspb.TriggerComposition_ANY
	default:
		return nil, fmt.Errorf("unknown --composition %q; expected all or any", c.Flag("composition"))
	}
	r, e := h.g.DeclareTrigger(context.Background(), connect.NewRequest(&offerspb.DeclareTriggerRequest{Trigger: trigger}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}

func (h *handlers) fact(c cliapp.OperationContext) (*offerspb.AddFactResponse, error) {
	observedAt := time.Now().UTC()
	if raw := strings.TrimSpace(c.Flag("observed-at")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			return nil, fmt.Errorf("--observed-at must be RFC3339: %w", err)
		}
		observedAt = parsed
	}
	fact := &offerspb.Fact{Name: c.Flag("name"), Value: parseFloat(c.Flag("value")), StaleAfterDays: int32(parseInt(c.Flag("stale-after-days"))), Dimension: strings.TrimSpace(c.Flag("dimension")), ObservedAt: timestamppb.New(observedAt)}
	r, e := h.g.AddFact(context.Background(), connect.NewRequest(&offerspb.AddFactRequest{Fact: fact}))
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

func (h *handlers) ladder(c cliapp.OperationContext) (*offerspb.ReleaseLadderResponse, error) {
	r, err := h.l.GetReleaseLadder(context.Background(), connect.NewRequest(&offerspb.ReleaseLadderRequest{IncludeRetired: strings.EqualFold(c.Flag("include-retired"), "true")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) enabling(c cliapp.OperationContext) (*offerspb.ReleaseLadderResponse, error) {
	r, err := h.l.GetEnablingDeliverables(context.Background(), connect.NewRequest(&offerspb.ReleaseLadderRequest{IncludeRetired: strings.EqualFold(c.Flag("include-retired"), "true")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) rank(c cliapp.OperationContext) (*offerspb.SetReleaseRankResponse, error) {
	rank := int32(parseInt(c.Flag("release-rank")))
	r, err := h.c.SetReleaseRank(context.Background(), connect.NewRequest(&offerspb.SetReleaseRankRequest{NodeId: c.Flag("node-id"), ReleaseRank: rank, Actor: c.Flag("actor"), Reason: c.Flag("reason")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func (h *handlers) rename(c cliapp.OperationContext) (*offerspb.RenameNodeResponse, error) {
	r, err := h.c.RenameNode(context.Background(), connect.NewRequest(&offerspb.RenameNodeRequest{NodeId: c.Flag("node-id"), Name: c.Flag("name"), Actor: c.Flag("actor"), Reason: c.Flag("reason")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func renameReport(_ cliapp.OperationContext, m *offerspb.RenameNodeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Renamed %s to %s (prior name %s).", m.Node.Id, m.Node.Name, m.PriorName)}}
}

func (h *handlers) prerequisites(c cliapp.OperationContext) (*offerspb.PrerequisiteWalkResponse, error) {
	r, err := h.l.GetPrerequisites(context.Background(), connect.NewRequest(&offerspb.PrerequisiteWalkRequest{StreamNodeId: c.Flag("stream-node-id"), MaxDepth: int32(parseInt(c.Flag("max-depth"))), IncludeShipped: strings.EqualFold(c.Flag("include-shipped"), "true")}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func setClassReport(_ cliapp.OperationContext, m *offerspb.SetDeliverableClassResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Classified %s as %s (%s).", m.Node.Name, m.Node.DeliverableClass.String(), m.Node.FinishBar.String())}}
}

func metersReport(_ cliapp.OperationContext, m *offerspb.MeterInventoryResponse) cliapp.ListReport {
	results := mapStrings(len(m.Meters), func(i int) string {
		meter := m.Meters[i]
		return fmt.Sprintf("%s — class=%s declared_by=%s", meter.LimitKey, meter.Class, strings.Join(meter.DeclaredBy, ","))
	})
	for _, stream := range m.UndeclaredStreams {
		results = append(results, "DRIFT undeclared stream: "+stream)
	}
	for _, gap := range m.DeliverableMeterGaps {
		results = append(results, "DRIFT deliverable meter gap: "+gap)
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Meter inventory: %d meter(s), %d undeclared stream(s), %d deliverable gap(s).", len(m.Meters), len(m.UndeclaredStreams), len(m.DeliverableMeterGaps))}, ResultsHeading: "Meter vocabulary", Results: results}
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

func ladderReport(_ cliapp.OperationContext, m *offerspb.ReleaseLadderResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d scheduled deliverable(s).", len(m.Entries))}, ResultsHeading: "Release ladder", Results: mapStrings(len(m.Entries), func(i int) string {
		e := m.Entries[i]
		return fmt.Sprintf("%d. %s — ramps=%d streams=%d audiences=%d cumulative_ramps=%d", e.Deliverable.ReleaseRank, e.Deliverable.Name, len(e.UnlockedRamps), len(e.UnlockedStreams), len(e.Audiences), len(e.CumulativeRamps))
	})}
}

func enablingReport(_ cliapp.OperationContext, m *offerspb.ReleaseLadderResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d enabling deliverable(s).", len(m.Enabling))}, ResultsHeading: "Enabling deliverables", Results: mapStrings(len(m.Enabling), func(i int) string {
		n := m.Enabling[i]
		return fmt.Sprintf("%s — urgency=%d depth=%d finish=%s status=%s", n.Node.Name, n.DerivedUrgency, n.Depth, n.Node.FinishBar.String(), n.Node.Status.String())
	})}
}

func rankReport(_ cliapp.OperationContext, m *offerspb.SetReleaseRankResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Set %s release rank to %d (was %d).", m.Node.Name, m.Node.ReleaseRank, m.PriorReleaseRank)}}
}

func prerequisitesReport(_ cliapp.OperationContext, m *offerspb.PrerequisiteWalkResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d prerequisite deliverable(s); %d unshipped.", len(m.Deliverables), len(m.Unshipped))}, ResultsHeading: "Prerequisites", Results: mapStrings(len(m.Deliverables), func(i int) string {
		if i < len(m.Tree) {
			n := m.Tree[i]
			return fmt.Sprintf("%*s%s — class=%s finish=%s status=%s urgency=%d depth=%d path=%s", int(n.Depth)*2, "", n.Node.Name, n.Node.DeliverableClass.String(), n.Node.FinishBar.String(), n.Node.Status.String(), n.DerivedUrgency, n.Depth, strings.Join(n.Path, " -> "))
		}
		return fmt.Sprintf("%s — %s", m.Deliverables[i].Name, m.Deliverables[i].Status.String())
	})}
}

func catalogImportReport(_ cliapp.OperationContext, m *offerspb.ImportCatalogResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Catalog import inspected %d source file(s), applied=%t, findings=%d.", len(m.Files), m.Applied, m.TotalFindings)}}
}

func (h *handlers) catalogMapAccount(c cliapp.OperationContext) (*offerspb.MapAccountResponse, error) {
	r, err := h.c.MapAccount(context.Background(), connect.NewRequest(&offerspb.MapAccountRequest{
		NodeId: c.Flag("node-id"), ActualAccountId: c.Flag("account-id"), Actor: c.Flag("actor"), Reason: c.Flag("reason"),
	}))
	if err != nil {
		return nil, err
	}
	return r.Msg, nil
}

func catalogMapAccountReport(_ cliapp.OperationContext, m *offerspb.MapAccountResponse) cliapp.MutationReport {
	prior := m.PriorAccountId
	if strings.TrimSpace(prior) == "" {
		prior = "(unmapped)"
	}
	next := m.Node.GetActualAccountId()
	if strings.TrimSpace(next) == "" {
		next = "(unmapped)"
	}
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Mapped %s (%s): %s -> %s.", m.Node.GetName(), m.Node.GetId(), prior, next)}}
}

func catalogMergeReport(_ cliapp.OperationContext, m *offerspb.MergeNodesResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Merge %s: edges=%d triggers=%d evaluations=%d proposals=%d findings=%d collapsed_edges=%d.", m.Surviving.Id, m.MovedEdges, m.MovedTriggers, m.MovedEvaluations, m.MovedProposals, m.MovedFindings, len(m.CollapsedEdgeIds))}}
}

func catalogVerifyReport(_ cliapp.OperationContext, m *offerspb.VerifyCatalogResponse) cliapp.ListReport {
	summary := []string{fmt.Sprintf("Catalog verification reconciled=%t, comparable=%t, files=%d, drift=%d, duplicate_identities=%d, orphan_edges=%d, extra_nodes=%d, scenario_gaps=%d.", m.Reconciled, m.Comparable, len(m.Files), m.TotalDrift, len(m.DuplicateIdentities), len(m.OrphanEdgeIds), len(m.ExtraNodeIds), len(m.ScenarioGaps))}
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

func parseDeliverableClass(v string) offerspb.DeliverableClass {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "enabling":
		return offerspb.DeliverableClass_ENABLING
	case "marketed", "":
		return offerspb.DeliverableClass_MARKETED
	default:
		return offerspb.DeliverableClass_DELIVERABLE_CLASS_UNSPECIFIED
	}
}
func parseFinishBar(v string) offerspb.FinishBar {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "customer-facing", "customer_facing":
		return offerspb.FinishBar_CUSTOMER_FACING
	case "operator-facing", "operator_facing":
		return offerspb.FinishBar_OPERATOR_FACING
	case "internal":
		return offerspb.FinishBar_INTERNAL
	default:
		return offerspb.FinishBar_FINISH_BAR_UNSPECIFIED
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
