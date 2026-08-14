package catalog

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed schema.sql
var schemaSQL embed.FS

type Store struct {
	db  *database.RoutedDB
	now func() time.Time
}

func NewStore(db *database.RoutedDB, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, now: now}
}
func (s *Store) DB() *database.RoutedDB { return s.db }
func (s *Store) Schema() string         { b, _ := schemaSQL.ReadFile("schema.sql"); return string(b) }

func (s *Store) CreateNode(ctx context.Context, kind offerspb.NodeKind, name string, status offerspb.Status, trigger, actualAccountID string) (*offerspb.Node, error) {
	if kind == offerspb.NodeKind_NODE_KIND_UNSPECIFIED || strings.TrimSpace(name) == "" {
		return nil, errors.New("node kind and name are required")
	}
	if status == offerspb.Status_STATUS_UNSPECIFIED {
		status = offerspb.Status_IDEA
	}
	if status == offerspb.Status_CANDIDATE && strings.TrimSpace(trigger) == "" {
		return nil, errors.New("rule candidate_requires_trigger: candidate nodes require a machine-evaluable trigger")
	}
	n := &offerspb.Node{Id: uuid.NewString(), Kind: kind, Name: strings.TrimSpace(name), Status: status, TriggerId: trigger, ActualAccountId: actualAccountID, CreatedAt: timestamppb.New(s.now())}
	if status == offerspb.Status_CANDIDATE {
		var triggerNode string
		if err := s.db.QueryRowContext(ctx, `SELECT node_id FROM triggers WHERE id=?`, trigger).Scan(&triggerNode); err != nil || triggerNode != n.Id {
			return nil, errors.New("rule candidate_requires_trigger refused creation: candidate nodes require an attached machine-evaluable trigger")
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes(id,kind,name,status,trigger_id,created_at,actual_account_id) VALUES(?,?,?,?,?,?,?)`, n.Id, int32(kind), n.Name, int32(status), trigger, n.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano), actualAccountID)
	return n, err
}

func (s *Store) ListNodes(ctx context.Context, kind offerspb.NodeKind, status offerspb.Status) ([]*offerspb.Node, error) {
	q := `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id FROM nodes WHERE 1=1`
	args := []any{}
	if kind != offerspb.NodeKind_NODE_KIND_UNSPECIFIED {
		q += ` AND kind=?`
		args = append(args, int32(kind))
	}
	if status != offerspb.Status_STATUS_UNSPECIFIED {
		q += ` AND status=?`
		args = append(args, int32(status))
	}
	q += ` ORDER BY created_at,id`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*offerspb.Node
	for rows.Next() {
		var n offerspb.Node
		var k, st int32
		var ts string
		if err := rows.Scan(&n.Id, &k, &n.Name, &st, &n.TriggerId, &ts, &n.ActualAccountId); err != nil {
			return nil, err
		}
		n.Kind = offerspb.NodeKind(k)
		n.Status = offerspb.Status(st)
		t, _ := time.Parse(time.RFC3339Nano, ts)
		n.CreatedAt = timestamppb.New(t)
		out = append(out, &n)
	}
	return out, rows.Err()
}

func allowed(from, to offerspb.Status) bool {
	switch from {
	case offerspb.Status_IDEA:
		return to == offerspb.Status_CANDIDATE || to == offerspb.Status_RETIRED
	case offerspb.Status_CANDIDATE:
		return to == offerspb.Status_TRIGGER_MET || to == offerspb.Status_IDEA || to == offerspb.Status_RETIRED
	case offerspb.Status_TRIGGER_MET:
		return to == offerspb.Status_ACTIVE || to == offerspb.Status_CANDIDATE || to == offerspb.Status_RETIRED
	case offerspb.Status_ACTIVE:
		return to == offerspb.Status_SHIPPED || to == offerspb.Status_RETIRED
	case offerspb.Status_SHIPPED:
		return to == offerspb.Status_RETIRED
	}
	return false
}

func legal(from offerspb.Status) string {
	switch from {
	case offerspb.Status_IDEA:
		return "candidate, retired"
	case offerspb.Status_CANDIDATE:
		return "trigger-met, idea, retired"
	case offerspb.Status_TRIGGER_MET:
		return "active, candidate, retired"
	case offerspb.Status_ACTIVE:
		return "shipped, retired"
	case offerspb.Status_SHIPPED:
		return "retired"
	}
	return "none"
}

func (s *Store) Transition(ctx context.Context, id string, to offerspb.Status, actor string) (*offerspb.Node, error) {
	var n offerspb.Node
	var k, st int32
	var ts string
	if err := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id FROM nodes WHERE id=?`, id).Scan(&n.Id, &k, &n.Name, &st, &n.TriggerId, &ts, &n.ActualAccountId); err != nil {
		return nil, fmt.Errorf("node %q not found: %w", id, err)
	}
	n.Kind = offerspb.NodeKind(k)
	n.Status = offerspb.Status(st)
	if !allowed(n.Status, to) {
		return nil, fmt.Errorf("rule legal_lifecycle_transition refused %s -> %s; legal transitions from %s: %s", n.Status.String(), to.String(), n.Status.String(), legal(n.Status))
	}
	if to == offerspb.Status_CANDIDATE || to == offerspb.Status_TRIGGER_MET {
		var triggerNode string
		if err := s.db.QueryRowContext(ctx, `SELECT node_id FROM triggers WHERE id=?`, n.TriggerId).Scan(&triggerNode); err != nil || triggerNode != n.Id {
			return nil, errors.New("rule candidate_requires_trigger refused transition: node needs an attached machine-evaluable trigger")
		}
	}
	if to == offerspb.Status_ACTIVE && actor != "operator" {
		return nil, errors.New("rule operator_only_promotion refused transition: required role operator; agents may create proposals")
	}
	if to == offerspb.Status_ACTIVE {
		var parentID string
		if err := s.db.QueryRowContext(ctx, `SELECT to_id FROM edges WHERE from_id=? AND kind='requires' ORDER BY id LIMIT 1`, n.Id).Scan(&parentID); err == nil {
			var payingUsers float64
			if err := s.db.QueryRowContext(ctx, `SELECT value FROM facts WHERE name=?`, "paying_users:"+parentID).Scan(&payingUsers); err != nil || payingUsers <= 0 {
				return nil, errors.New("rule parent_requires_paying_users refused transition: required parent paying-users fact is missing or not positive")
			}
		}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE nodes SET status=? WHERE id=?`, int32(to), id); err != nil {
		return nil, err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), id, actor, st, int32(to), "state transition", s.now().UTC().Format(time.RFC3339Nano))
	n.Status = to
	t, _ := time.Parse(time.RFC3339Nano, ts)
	n.CreatedAt = timestamppb.New(t)
	return &n, err
}

func (s *Store) ListEdges(ctx context.Context, nodeID string) ([]*offerspb.Edge, error) {
	query := `SELECT id,from_id,to_id,kind,intended_price_minor,currency FROM edges`
	args := []any{}
	if strings.TrimSpace(nodeID) != "" {
		query += ` WHERE from_id=? OR to_id=?`
		args = append(args, nodeID, nodeID)
	}
	query += ` ORDER BY id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*offerspb.Edge
	for rows.Next() {
		var e offerspb.Edge
		if err := rows.Scan(&e.Id, &e.FromId, &e.ToId, &e.Kind, &e.IntendedPriceMinor, &e.Currency); err != nil {
			return nil, err
		}
		out = append(out, &e)
	}
	return out, rows.Err()
}

func (s *Store) CreateEdge(ctx context.Context, e *offerspb.Edge) (*offerspb.Edge, error) {
	if e == nil || strings.TrimSpace(e.FromId) == "" || strings.TrimSpace(e.ToId) == "" || strings.TrimSpace(e.Kind) == "" {
		return nil, errors.New("edge requires from_id, to_id, and kind")
	}
	if e.FromId == e.ToId {
		return nil, errors.New("rule no_self_edges refused self-referential edge")
	}
	var from, to int32
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM nodes WHERE id=?`, e.FromId).Scan(&from); err != nil {
		return nil, fmt.Errorf("from node %q not found: %w", e.FromId, err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT kind FROM nodes WHERE id=?`, e.ToId).Scan(&to); err != nil {
		return nil, fmt.Errorf("to node %q not found: %w", e.ToId, err)
	}
	valid := (e.Kind == "sells_at" && from == int32(offerspb.NodeKind_OFFER) && to == int32(offerspb.NodeKind_VARIANT)) ||
		(e.Kind == "feeds" && from == int32(offerspb.NodeKind_CHANNEL) && to == int32(offerspb.NodeKind_REVENUE_LINE)) ||
		(e.Kind == "belongs_to" && from == int32(offerspb.NodeKind_DELIVERABLE) && to == int32(offerspb.NodeKind_OFFER)) ||
		(e.Kind == "requires" && from == int32(offerspb.NodeKind_OFFER) && to == int32(offerspb.NodeKind_OFFER))
	if !valid {
		return nil, fmt.Errorf("rule typed_edge_matrix refused %s -> %s for edge kind %q", offerspb.NodeKind(from).String(), offerspb.NodeKind(to).String(), e.Kind)
	}
	if e.Id == "" {
		e.Id = uuid.NewString()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO edges(id,from_id,to_id,kind,intended_price_minor,currency) VALUES(?,?,?,?,?,?)`, e.Id, e.FromId, e.ToId, e.Kind, e.IntendedPriceMinor, e.Currency)
	return e, err
}

func (s *Store) AddTrigger(ctx context.Context, t *offerspb.Trigger) (*offerspb.Trigger, error) {
	if t == nil || strings.TrimSpace(t.NodeId) == "" || strings.TrimSpace(t.FactName) == "" {
		return nil, errors.New("trigger requires node_id and fact_name")
	}
	switch t.Operator {
	case ">=", "<=", "=", "==", ">", "<":
	default:
		return nil, errors.New("trigger grammar admits only >=, <=, =, >, <")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM nodes WHERE id=?`, t.NodeId).Scan(&exists); err != nil {
		return nil, fmt.Errorf("node %q not found: %w", t.NodeId, err)
	}
	if len(t.Clauses) == 0 {
		t.Clauses = []*offerspb.TriggerClause{{FactName: t.FactName, Operator: t.Operator, Threshold: t.Threshold}}
	}
	for _, clause := range t.Clauses {
		if clause == nil || strings.TrimSpace(clause.FactName) == "" {
			return nil, errors.New("trigger grammar requires a fact name in every clause")
		}
		switch clause.Operator {
		case ">=", "<=", "=", "==", ">", "<":
		default:
			return nil, errors.New("trigger grammar admits only >=, <=, =, >, <")
		}
	}
	if t.Composition == offerspb.TriggerComposition_TRIGGER_COMPOSITION_UNSPECIFIED {
		t.Composition = offerspb.TriggerComposition_ALL
	}
	clauses, err := json.Marshal(t.Clauses)
	if err != nil {
		return nil, err
	}
	if t.Id == "" {
		t.Id = uuid.NewString()
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO triggers(id,node_id,fact_name,operator,threshold,expression,clauses_json,composition) VALUES(?,?,?,?,?,?,?,?)`, t.Id, t.NodeId, t.FactName, t.Operator, t.Threshold, t.Expression, string(clauses), int32(t.Composition))
	if err == nil {
		_, err = s.db.ExecContext(ctx, `UPDATE nodes SET trigger_id=?,status=CASE WHEN status=0 THEN 0 ELSE status END WHERE id=?`, t.Id, t.NodeId)
	}
	return t, err
}

func (s *Store) AddFact(ctx context.Context, f *offerspb.Fact) (*offerspb.Fact, error) {
	if f == nil || strings.TrimSpace(f.Name) == "" {
		return nil, errors.New("fact name is required")
	}
	if f.ObservedAt == nil {
		f.ObservedAt = timestamppb.New(s.now())
	}
	if f.StaleAfterDays <= 0 {
		f.StaleAfterDays = staleWindow(f.Dimension)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO facts(name,value,observed_at,stale_after_days,dimension) VALUES(?,?,?,?,?) ON CONFLICT(name) DO UPDATE SET value=excluded.value,observed_at=excluded.observed_at,stale_after_days=excluded.stale_after_days,dimension=excluded.dimension`, f.Name, f.Value, f.ObservedAt.AsTime().UTC().Format(time.RFC3339Nano), f.StaleAfterDays, f.Dimension)
	return f, err
}

func staleWindow(dimension string) int32 {
	switch strings.ToLower(strings.TrimSpace(dimension)) {
	case "pricing":
		return 90
	case "retention", "activation":
		return 180
	case "channel-cac":
		return 120
	default:
		return 365
	}
}

func compare(value float64, op string, threshold float64) bool {
	switch op {
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case "=", "==":
		return value == threshold
	}
	return false
}

func combineVerdicts(verdicts []offerspb.Verdict, composition offerspb.TriggerComposition) offerspb.Verdict {
	if len(verdicts) == 0 {
		return offerspb.Verdict_UNKNOWN
	}
	if composition == offerspb.TriggerComposition_ANY {
		hasUnknown := false
		for _, verdict := range verdicts {
			if verdict == offerspb.Verdict_SATISFIED {
				return offerspb.Verdict_SATISFIED
			}
			if verdict == offerspb.Verdict_UNKNOWN {
				hasUnknown = true
			}
		}
		if hasUnknown {
			return offerspb.Verdict_UNKNOWN
		}
		return offerspb.Verdict_UNSATISFIED
	}
	hasUnknown := false
	for _, verdict := range verdicts {
		if verdict == offerspb.Verdict_UNSATISFIED {
			return offerspb.Verdict_UNSATISFIED
		}
		if verdict == offerspb.Verdict_UNKNOWN {
			hasUnknown = true
		}
	}
	if hasUnknown {
		return offerspb.Verdict_UNKNOWN
	}
	return offerspb.Verdict_SATISFIED
}

func (s *Store) Evaluate(ctx context.Context, dry bool) ([]*offerspb.Evaluation, error) {
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_CANDIDATE)
	if err != nil {
		return nil, err
	}
	var out []*offerspb.Evaluation
	for _, n := range nodes {
		var t offerspb.Trigger
		var clausesJSON string
		var composition int32
		err := s.db.QueryRowContext(ctx, `SELECT id,node_id,fact_name,operator,threshold,expression,clauses_json,composition FROM triggers WHERE id=?`, n.TriggerId).Scan(&t.Id, &t.NodeId, &t.FactName, &t.Operator, &t.Threshold, &t.Expression, &clausesJSON, &composition)
		if err != nil {
			continue
		}
		if err := json.Unmarshal([]byte(clausesJSON), &t.Clauses); err != nil || len(t.Clauses) == 0 {
			t.Clauses = []*offerspb.TriggerClause{{FactName: t.FactName, Operator: t.Operator, Threshold: t.Threshold}}
		}
		t.Composition = offerspb.TriggerComposition(composition)
		verdicts := make([]offerspb.Verdict, 0, len(t.Clauses))
		factNames := make([]string, 0, len(t.Clauses))
		maxAge := int64(0)
		explanations := make([]string, 0, len(t.Clauses))
		for _, clause := range t.Clauses {
			factNames = append(factNames, clause.FactName)
			var value float64
			var observed string
			var stale int
			err := s.db.QueryRowContext(ctx, `SELECT value,observed_at,stale_after_days FROM facts WHERE name=?`, clause.FactName).Scan(&value, &observed, &stale)
			if err != nil {
				verdicts = append(verdicts, offerspb.Verdict_UNKNOWN)
				explanations = append(explanations, fmt.Sprintf("fact %s missing: unknown is not false", clause.FactName))
				continue
			}
			at, parseErr := time.Parse(time.RFC3339Nano, observed)
			if parseErr != nil {
				verdicts = append(verdicts, offerspb.Verdict_UNKNOWN)
				explanations = append(explanations, fmt.Sprintf("fact %s has invalid observation time", clause.FactName))
				continue
			}
			age := int64(s.now().Sub(at).Seconds())
			if age < 0 {
				age = 0
			}
			if age > maxAge {
				maxAge = age
			}
			if s.now().After(at.Add(time.Duration(stale) * 24 * time.Hour)) {
				verdicts = append(verdicts, offerspb.Verdict_UNKNOWN)
				explanations = append(explanations, fmt.Sprintf("fact %s is stale: unknown is not false", clause.FactName))
				continue
			}
			if compare(value, clause.Operator, clause.Threshold) {
				verdicts = append(verdicts, offerspb.Verdict_SATISFIED)
				explanations = append(explanations, fmt.Sprintf("fact %s satisfied %s %v", clause.FactName, clause.Operator, clause.Threshold))
			} else {
				verdicts = append(verdicts, offerspb.Verdict_UNSATISFIED)
				explanations = append(explanations, fmt.Sprintf("fact %s did not satisfy %s %v", clause.FactName, clause.Operator, clause.Threshold))
			}
		}
		v := combineVerdicts(verdicts, t.Composition)
		e := &offerspb.Evaluation{Id: uuid.NewString(), NodeId: n.Id, Verdict: v, FactName: t.FactName, FactNames: factNames, FactAgeSeconds: maxAge, Explanation: strings.Join(explanations, "; "), EvaluatedAt: timestamppb.New(s.now())}
		out = append(out, e)
		if !dry {
			_, err = s.db.ExecContext(ctx, `INSERT INTO evaluations(id,node_id,verdict,fact_name,explanation,evaluated_at) VALUES(?,?,?,?,?,?)`, e.Id, e.NodeId, int32(e.Verdict), e.FactName, e.Explanation, e.EvaluatedAt.AsTime().UTC().Format(time.RFC3339Nano))
			if err != nil {
				return nil, err
			}
			if v == offerspb.Verdict_SATISFIED {
				if _, err := s.Transition(ctx, n.Id, offerspb.Status_TRIGGER_MET, "scheduler"); err != nil {
					return nil, err
				}
			}
		}
	}
	return out, nil
}

func (s *Store) Proposal(ctx context.Context, nodeID, actor string, status offerspb.Status, reason string) (*offerspb.Proposal, error) {
	if actor == "" {
		actor = "agent"
	}
	p := &offerspb.Proposal{Id: uuid.NewString(), NodeId: nodeID, Actor: actor, RequestedStatus: status, Reason: reason}
	_, err := s.db.ExecContext(ctx, `INSERT INTO proposals(id,node_id,actor,requested_status,reason,created_at) VALUES(?,?,?,?,?,?)`, p.Id, p.NodeId, p.Actor, int32(p.RequestedStatus), p.Reason, s.now().UTC().Format(time.RFC3339Nano))
	return p, err
}
