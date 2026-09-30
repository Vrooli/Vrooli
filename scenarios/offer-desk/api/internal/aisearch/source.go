package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"offer-desk/internal/catalog"

	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StoreSource projects the authoritative catalog store. It reads every node
// kind plus relationships, triggers, facts, evaluations, proposals, and audit
// rows; it writes nothing.
type StoreSource struct {
	store *catalog.Store
	now   func() time.Time
}

// NewStoreSource builds the production source over the catalog store. now may
// be nil, in which case time.Now is used.
func NewStoreSource(store *catalog.Store, now func() time.Time) *StoreSource {
	if now == nil {
		now = time.Now
	}
	return &StoreSource{store: store, now: now}
}

// Load builds one deterministic snapshot of the corpus.
func (s *StoreSource) Load(ctx context.Context) (*Snapshot, error) {
	nodes, err := s.store.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, fmt.Errorf("load nodes: %w", err)
	}
	edges, err := s.store.ListEdges(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("load edges: %w", err)
	}
	triggers, err := s.store.ListTriggers(ctx)
	if err != nil {
		return nil, fmt.Errorf("load triggers: %w", err)
	}
	facts, err := s.store.ListFacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("load facts: %w", err)
	}
	evaluations, err := s.store.ListEvaluations(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("load evaluations: %w", err)
	}
	proposals, err := s.store.ListProposals(ctx, "", offerspb.Status_STATUS_UNSPECIFIED)
	if err != nil {
		return nil, fmt.Errorf("load proposals: %w", err)
	}
	audit, err := s.store.ListAudit(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("load audit: %w", err)
	}

	names := make(map[string]string, len(nodes))
	for _, n := range nodes {
		names[n.GetId()] = n.GetName()
	}

	now := s.now()
	var records []Record
	var materialized time.Time
	bump := func(t time.Time) {
		if t.After(materialized) {
			materialized = t
		}
	}

	for _, n := range nodes {
		records = append(records, nodeRecord(n))
		if n.GetCreatedAt() != nil {
			bump(n.GetCreatedAt().AsTime())
		}
	}
	for _, e := range edges {
		records = append(records, edgeRecord(e, names))
	}
	for _, t := range triggers {
		records = append(records, triggerRecord(t, names))
	}
	for _, f := range facts {
		records = append(records, factRecord(f, now))
		if f.GetObservedAt() != nil {
			bump(f.GetObservedAt().AsTime())
		}
	}
	latestEvaluation := make(map[string]string, len(evaluations))
	for _, e := range evaluations {
		if _, seen := latestEvaluation[e.GetNodeId()]; !seen {
			latestEvaluation[e.GetNodeId()] = e.GetId()
		}
	}
	for _, e := range evaluations {
		records = append(records, evaluationRecord(e, names, latestEvaluation[e.GetNodeId()] != e.GetId()))
		if e.GetEvaluatedAt() != nil {
			bump(e.GetEvaluatedAt().AsTime())
		}
	}
	for _, p := range proposals {
		records = append(records, proposalRecord(p, names))
		if p.GetCreatedAt() != nil {
			bump(p.GetCreatedAt().AsTime())
		}
	}
	for _, a := range audit {
		records = append(records, auditRecord(a, names))
		bump(a.CreatedAt)
	}

	records = append(records, decisionContextRecords(nodes, triggers, facts, evaluations, proposals, audit, now)...)

	for i := range records {
		records[i].Revision = revisionOf(records[i])
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })

	return &Snapshot{
		Records:        records,
		Generation:     generationOf(records),
		MaterializedAt: materialized,
	}, nil
}

func nodeRecord(n *offerspb.Node) Record {
	kindLabel := nodeKindLabel(n.GetKind())
	kind := "node-" + kindLabel
	status := n.GetStatus().String()
	// Lead with the catalog identity and the node's human-readable role. The
	// cross-provider reranker reads only type/title/path and the first 120
	// characters of the snippet, so a terse tag line leaves a real catalog
	// entity semantically invisible next to a governance rule that merely
	// mentions the word "offer". Every term here comes from the node's own
	// fields; no query text is copied.
	snippet := fmt.Sprintf("Existing Offer Desk catalog %s named %q; status %s", kindLabel, n.GetName(), status)
	if class := n.GetDeliverableClass().String(); class != "" && class != "UNSPECIFIED" {
		snippet += "; deliverable class " + class
	}
	if bar := n.GetFinishBar().String(); bar != "" && bar != "UNSPECIFIED" {
		snippet += "; finish bar " + bar
	}
	if n.GetReleaseRank() > 0 {
		snippet += fmt.Sprintf("; release rank %d", n.GetReleaseRank())
	}
	body := strings.Join([]string{
		n.GetName(),
		"Offer Desk catalog " + kindLabel,
		kind,
		"status " + status,
		"deliverable class " + n.GetDeliverableClass().String(),
		"finish bar " + n.GetFinishBar().String(),
	}, ". ")
	return Record{
		ID:         "node:" + n.GetId(),
		Kind:       kind,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/node/" + n.GetId(),
		Title:      n.GetName(),
		Snippet:    snippet,
		Body:       body,
		Metadata: map[string]any{
			"node_id":           n.GetId(),
			"node_kind":         n.GetKind().String(),
			"status":            status,
			"release_rank":      n.GetReleaseRank(),
			"deliverable_class": n.GetDeliverableClass().String(),
			"finish_bar":        n.GetFinishBar().String(),
		},
	}
}

func edgeRecord(e *offerspb.Edge, names map[string]string) Record {
	from := nodeLabel(e.GetFromId(), names)
	to := nodeLabel(e.GetToId(), names)
	price := "price undeclared"
	if e.GetIntendedPriceDeclared() {
		price = fmt.Sprintf("intended price %d %s", e.GetIntendedPriceMinor(), e.GetCurrency())
	}
	return Record{
		ID:         "edge:" + e.GetId(),
		Kind:       KindEdge,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/edge/" + e.GetId(),
		Title:      fmt.Sprintf("%s %s %s", from, e.GetKind(), to),
		Snippet:    fmt.Sprintf("relationship %q from %s to %s; %s", e.GetKind(), from, to, price),
		Body:       strings.Join([]string{from, e.GetKind(), to, price}, ". "),
		Metadata: map[string]any{
			"edge_id":                 e.GetId(),
			"from_id":                 e.GetFromId(),
			"to_id":                   e.GetToId(),
			"kind":                    e.GetKind(),
			"intended_price_minor":    e.GetIntendedPriceMinor(),
			"currency":                e.GetCurrency(),
			"intended_price_declared": e.GetIntendedPriceDeclared(),
		},
	}
}

func triggerRecord(t *offerspb.Trigger, names map[string]string) Record {
	clauses := make([]string, 0, len(t.GetClauses()))
	for _, c := range t.GetClauses() {
		if c == nil {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("%s %s %v", c.GetFactName(), c.GetOperator(), c.GetThreshold()))
	}
	condition := t.GetFactName() + " " + t.GetOperator() + " " + fmt.Sprintf("%v", t.GetThreshold())
	if len(clauses) > 0 {
		condition = strings.Join(clauses, " "+strings.ToLower(t.GetComposition().String())+" ")
	}
	return Record{
		ID:         "trigger:" + t.GetId(),
		Kind:       KindTrigger,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/trigger/" + t.GetId(),
		Title:      "trigger for " + nodeLabel(t.GetNodeId(), names),
		Snippet:    "fires when " + condition,
		Body:       "offer decision trigger: " + condition + " for " + nodeLabel(t.GetNodeId(), names),
		Metadata: map[string]any{
			"trigger_id":  t.GetId(),
			"node_id":     t.GetNodeId(),
			"fact_name":   t.GetFactName(),
			"operator":    t.GetOperator(),
			"threshold":   t.GetThreshold(),
			"expression":  t.GetExpression(),
			"composition": t.GetComposition().String(),
			"clauses":     clauses,
		},
	}
}

func factRecord(f *offerspb.Fact, now time.Time) Record {
	freshness, observed, freshUntil := factFreshnessState(f, now)
	snippet := fmt.Sprintf("fact %s = %v; freshness %s", f.GetName(), f.GetValue(), freshness)
	if observed != "" {
		snippet += "; observed " + observed
	}
	return Record{
		ID:         "fact:" + f.GetName(),
		Kind:       KindFact,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/fact/" + f.GetName(),
		Title:      f.GetName(),
		Snippet:    snippet,
		Body:       fmt.Sprintf("observed fact %s = %v (dimension %s)", f.GetName(), f.GetValue(), f.GetDimension()),
		Freshness:  freshness,
		Metadata: map[string]any{
			"name":             f.GetName(),
			"value":            f.GetValue(),
			"dimension":        f.GetDimension(),
			"observed_at":      observed,
			"stale_after_days": f.GetStaleAfterDays(),
			"fresh_until":      freshUntil,
			"freshness":        freshness,
		},
	}
}

// factFreshnessState derives a fact's freshness label, observation timestamp,
// and stale-after window from owner fields only. A fact with no observation or
// no stale-after window is "unknown"; the read time never fabricates freshness.
func factFreshnessState(f *offerspb.Fact, now time.Time) (freshness, observed, freshUntil string) {
	freshness = FreshnessUnknown
	if f.GetObservedAt() == nil {
		return freshness, "", ""
	}
	at := f.GetObservedAt().AsTime()
	observed = at.UTC().Format(time.RFC3339Nano)
	if f.GetStaleAfterDays() <= 0 {
		return freshness, observed, ""
	}
	until := at.Add(time.Duration(f.GetStaleAfterDays()) * 24 * time.Hour)
	freshUntil = until.UTC().Format(time.RFC3339Nano)
	if now.After(until) {
		return FreshnessStale, observed, freshUntil
	}
	return FreshnessFresh, observed, freshUntil
}

// decisionContextRecords projects one owner-generated aggregate per node that
// carries trigger, evaluation, or proposal state. The index is keyed by node
// identity so a decision question resolves to the catalog entity and its
// recorded inputs instead of the individual rows scattered across the corpus.
func decisionContextRecords(
	nodes []*offerspb.Node,
	triggers []*offerspb.Trigger,
	facts []*offerspb.Fact,
	evaluations []*offerspb.Evaluation,
	proposals []*offerspb.Proposal,
	audits []catalog.AuditEntry,
	now time.Time,
) []Record {
	factsByName := make(map[string]*offerspb.Fact, len(facts))
	for _, f := range facts {
		if f == nil {
			continue
		}
		if prev, ok := factsByName[f.GetName()]; !ok || factObservedLater(f, prev) {
			factsByName[f.GetName()] = f
		}
	}
	triggersByNode := make(map[string][]*offerspb.Trigger, len(nodes))
	for _, t := range triggers {
		if t != nil {
			triggersByNode[t.GetNodeId()] = append(triggersByNode[t.GetNodeId()], t)
		}
	}
	currentEvalByNode := make(map[string]*offerspb.Evaluation, len(nodes))
	for _, e := range evaluations {
		if e == nil {
			continue
		}
		if prev, ok := currentEvalByNode[e.GetNodeId()]; !ok || evaluationLater(e, prev) {
			currentEvalByNode[e.GetNodeId()] = e
		}
	}
	proposalsByNode := make(map[string][]*offerspb.Proposal, len(nodes))
	for _, p := range proposals {
		if p != nil {
			proposalsByNode[p.GetNodeId()] = append(proposalsByNode[p.GetNodeId()], p)
		}
	}
	statusesByNode := make(map[string][]string, len(nodes))
	for _, a := range audits {
		statusesByNode[a.NodeID] = appendDistinctStatus(statusesByNode[a.NodeID], a.PriorStatus, a.NextStatus)
	}

	out := make([]Record, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		rec, ok := decisionContextRecord(n, triggersByNode[n.GetId()], factsByName, currentEvalByNode[n.GetId()], proposalsByNode[n.GetId()], statusesByNode[n.GetId()], now)
		if !ok {
			continue
		}
		out = append(out, rec)
	}
	return out
}

// decisionContextRecord renders the aggregate only when the node has recorded
// decision state; an empty node is not given a fabricated verdict.
func decisionContextRecord(
	n *offerspb.Node,
	triggers []*offerspb.Trigger,
	facts map[string]*offerspb.Fact,
	current *offerspb.Evaluation,
	proposals []*offerspb.Proposal,
	recordedStatuses []string,
	now time.Time,
) (Record, bool) {
	if len(triggers) == 0 && current == nil && len(proposals) == 0 {
		return Record{}, false
	}
	// A retired node has no current decision state. Its history stays visible
	// through the node and audit records, but a current-state aggregate must not
	// present a retired test probe as the answer to "why is this offer still a
	// candidate".
	if n.GetStatus() == offerspb.Status_RETIRED {
		return Record{}, false
	}
	name := n.GetName()
	kindLabel := nodeKindLabel(n.GetKind())
	status := n.GetStatus().String()

	// Lead with the status and the node's recorded lifecycle statuses. The
	// reranker sees only the first 120 characters, and a question about why an
	// offer is still a candidate must resolve to the node whose audit history
	// actually contains that status; the terms come from the owner's own audit
	// rows, never from the query.
	head := fmt.Sprintf("Offer decision context %s: catalog status %s", name, status)
	if len(recordedStatuses) > 0 {
		head += "; recorded " + strings.Join(recordedStatuses, ", ")
	}
	parts := []string{head + "."}

	followUps := []string{}
	for _, t := range triggers {
		condition := triggerCondition(t)
		parts = append(parts, fmt.Sprintf("Trigger %s fires when %s.", t.GetId(), condition))
		if f := facts[t.GetFactName()]; f != nil {
			freshness, observed, _ := factFreshnessState(f, now)
			observation := ""
			if observed != "" {
				observation = " observed " + observed
			}
			parts = append(parts, fmt.Sprintf("Fact %s is %s%s.", f.GetName(), freshness, observation))
		}
		followUps = append(followUps, "catalog/trigger/"+t.GetId())
	}
	if current != nil {
		parts = append(parts, fmt.Sprintf("Latest evaluation %s reached verdict %s: %s.", current.GetId(), current.GetVerdict().String(), current.GetExplanation()))
		followUps = append(followUps, "catalog/evaluation/"+current.GetId())
	} else {
		parts = append(parts, "No current evaluation is recorded for this node.")
	}
	if len(proposals) == 0 {
		parts = append(parts, "No promotion proposal is recorded, so no open proposal explains the catalog status.")
	} else {
		latest := latestProposal(proposals)
		parts = append(parts, fmt.Sprintf("Latest proposal %s requests %s by %s: %s (%d decline(s) recorded).",
			latest.GetId(), latest.GetRequestedStatus().String(), latest.GetActor(), latest.GetReason(), len(latest.GetDeclineHistory())))
		followUps = append(followUps, "catalog/proposal/"+latest.GetId())
	}

	return Record{
		ID:         "decision:" + n.GetId(),
		Kind:       KindDecisionContext,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/node/" + n.GetId(),
		Title:      fmt.Sprintf("Offer decision context: %s (%s)", name, status),
		Snippet:    strings.Join(parts, " "),
		Body:       strings.Join(append([]string{name, "Offer Desk catalog " + kindLabel, "decision context", "status " + status}, parts[1:]...), " "),
		Historical: false,
		Metadata: map[string]any{
			"node_id":            n.GetId(),
			"node_kind":          n.GetKind().String(),
			"status":             status,
			"recorded_statuses":  recordedStatuses,
			"trigger_ids":        triggerIDs(triggers),
			"current_eval_id":    evaluationID(current),
			"proposal_ids":       proposalIDs(proposals),
			"decision_followups": followUps,
		},
	}, true
}

// triggerCondition renders a trigger's own clause expression.
func triggerCondition(t *offerspb.Trigger) string {
	clauses := make([]string, 0, len(t.GetClauses()))
	for _, c := range t.GetClauses() {
		if c == nil {
			continue
		}
		clauses = append(clauses, fmt.Sprintf("%s %s %v", c.GetFactName(), c.GetOperator(), c.GetThreshold()))
	}
	if len(clauses) > 0 {
		return strings.Join(clauses, " "+strings.ToLower(t.GetComposition().String())+" ")
	}
	return t.GetFactName() + " " + t.GetOperator() + " " + fmt.Sprintf("%v", t.GetThreshold())
}

func factObservedLater(a, b *offerspb.Fact) bool {
	if a.GetObservedAt() == nil {
		return false
	}
	if b.GetObservedAt() == nil {
		return true
	}
	return a.GetObservedAt().AsTime().After(b.GetObservedAt().AsTime())
}

func evaluationLater(a, b *offerspb.Evaluation) bool {
	if a.GetEvaluatedAt() == nil {
		return false
	}
	if b.GetEvaluatedAt() == nil {
		return true
	}
	return a.GetEvaluatedAt().AsTime().After(b.GetEvaluatedAt().AsTime())
}

func latestProposal(proposals []*offerspb.Proposal) *offerspb.Proposal {
	latest := proposals[0]
	for _, p := range proposals[1:] {
		if p.GetCreatedAt() == nil {
			continue
		}
		if latest.GetCreatedAt() == nil || p.GetCreatedAt().AsTime().After(latest.GetCreatedAt().AsTime()) {
			latest = p
		}
	}
	return latest
}

// appendDistinctStatus adds the statuses introduced by one audit transition.
func appendDistinctStatus(out []string, prior, next offerspb.Status) []string {
	for _, s := range []offerspb.Status{prior, next} {
		label := s.String()
		if label == "" || label == "UNSPECIFIED" {
			continue
		}
		found := false
		for _, existing := range out {
			if existing == label {
				found = true
				break
			}
		}
		if !found {
			out = append(out, label)
		}
	}
	return out
}

func triggerIDs(triggers []*offerspb.Trigger) []string {
	out := make([]string, 0, len(triggers))
	for _, t := range triggers {
		out = append(out, t.GetId())
	}
	return out
}

func evaluationID(e *offerspb.Evaluation) string {
	if e == nil {
		return ""
	}
	return e.GetId()
}

func proposalIDs(proposals []*offerspb.Proposal) []string {
	out := make([]string, 0, len(proposals))
	for _, p := range proposals {
		out = append(out, p.GetId())
	}
	return out
}

func evaluationRecord(e *offerspb.Evaluation, names map[string]string, historical bool) Record {
	evaluated := ""
	if e.GetEvaluatedAt() != nil {
		evaluated = e.GetEvaluatedAt().AsTime().UTC().Format(time.RFC3339Nano)
	}
	label := "current evaluation"
	if historical {
		label = "historical evaluation"
	}
	return Record{
		ID:         "evaluation:" + e.GetId(),
		Kind:       KindEvaluation,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/evaluation/" + e.GetId(),
		Title:      label + " of " + nodeLabel(e.GetNodeId(), names),
		Snippet:    fmt.Sprintf("%s: %s — %s", label, e.GetVerdict().String(), e.GetExplanation()),
		Body:       fmt.Sprintf("%s of %s: verdict %s, %s", label, nodeLabel(e.GetNodeId(), names), e.GetVerdict().String(), e.GetExplanation()),
		Historical: historical,
		Metadata: map[string]any{
			"evaluation_id": e.GetId(),
			"node_id":       e.GetNodeId(),
			"verdict":       e.GetVerdict().String(),
			"fact_name":     e.GetFactName(),
			"explanation":   e.GetExplanation(),
			"evaluated_at":  evaluated,
			"historical":    historical,
		},
	}
}

func proposalRecord(p *offerspb.Proposal, names map[string]string) Record {
	created := ""
	if p.GetCreatedAt() != nil {
		created = p.GetCreatedAt().AsTime().UTC().Format(time.RFC3339Nano)
	}
	declines := make([]map[string]any, 0, len(p.GetDeclineHistory()))
	for _, d := range p.GetDeclineHistory() {
		if d == nil {
			continue
		}
		declines = append(declines, map[string]any{"actor": d.GetActor(), "reason": d.GetReason(), "created_at": timestampString(d.GetCreatedAt())})
	}
	return Record{
		ID:         "proposal:" + p.GetId(),
		Kind:       KindProposal,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/proposal/" + p.GetId(),
		Title:      fmt.Sprintf("proposal to %s %s", p.GetRequestedStatus().String(), nodeLabel(p.GetNodeId(), names)),
		Snippet:    fmt.Sprintf("proposed by %s: %s", p.GetActor(), p.GetReason()),
		Body:       fmt.Sprintf("promotion proposal for %s to %s by %s: %s", nodeLabel(p.GetNodeId(), names), p.GetRequestedStatus().String(), p.GetActor(), p.GetReason()),
		Metadata: map[string]any{
			"proposal_id":      p.GetId(),
			"node_id":          p.GetNodeId(),
			"actor":            p.GetActor(),
			"requested_status": p.GetRequestedStatus().String(),
			"reason":           p.GetReason(),
			"created_at":       created,
			"decline_count":    len(p.GetDeclineHistory()),
			"declines":         declines,
		},
	}
}

func auditRecord(a catalog.AuditEntry, names map[string]string) Record {
	created := ""
	if !a.CreatedAt.IsZero() {
		created = a.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	reason := a.Reason
	if a.RelatedNodeID != "" {
		reason = fmt.Sprintf("%s (related to %s)", reason, nodeLabel(a.RelatedNodeID, names))
	}
	return Record{
		ID:         "audit:" + a.ID,
		Kind:       KindAudit,
		Visibility: VisibilityOperator,
		FollowUp:   "catalog/audit/" + a.ID,
		Title:      fmt.Sprintf("audit %s → %s on %s", a.PriorStatus.String(), a.NextStatus.String(), nodeLabel(a.NodeID, names)),
		Snippet:    fmt.Sprintf("%s by %s: %s", created, a.Actor, reason),
		Body:       fmt.Sprintf("catalog history: %s transitioned from %s to %s by %s: %s", nodeLabel(a.NodeID, names), a.PriorStatus.String(), a.NextStatus.String(), a.Actor, reason),
		Historical: true,
		Metadata: map[string]any{
			"audit_id":        a.ID,
			"node_id":         a.NodeID,
			"actor":           a.Actor,
			"prior_status":    a.PriorStatus.String(),
			"next_status":     a.NextStatus.String(),
			"reason":          reason,
			"created_at":      created,
			"related_node_id": a.RelatedNodeID,
		},
	}
}

func nodeLabel(id string, names map[string]string) string {
	if name := strings.TrimSpace(names[id]); name != "" {
		return name
	}
	if strings.TrimSpace(id) == "" {
		return "unknown node"
	}
	return id
}

func nodeKindLabel(kind offerspb.NodeKind) string {
	switch kind {
	case offerspb.NodeKind_OFFER:
		return "offer"
	case offerspb.NodeKind_VARIANT:
		return "variant"
	case offerspb.NodeKind_CHANNEL:
		return "channel"
	case offerspb.NodeKind_REVENUE_LINE:
		return "revenue-line"
	case offerspb.NodeKind_DELIVERABLE:
		return "deliverable"
	case offerspb.NodeKind_RAMP:
		return "ramp"
	case offerspb.NodeKind_STREAM:
		return "stream"
	case offerspb.NodeKind_AUDIENCE:
		return "audience"
	default:
		return "unspecified"
	}
}

func timestampString(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	t := ts.AsTime()
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func revisionOf(r Record) string {
	payload, err := json.Marshal(struct {
		Kind       string
		Title      string
		Snippet    string
		Body       string
		Freshness  string
		Historical bool
		Metadata   map[string]any
	}{r.Kind, r.Title, r.Snippet, r.Body, r.Freshness, r.Historical, r.Metadata})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func generationOf(records []Record) string {
	h := sha256.New()
	for _, r := range records {
		h.Write([]byte(r.ID))
		h.Write([]byte{0})
		h.Write([]byte(r.Revision))
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)[:16])
}
