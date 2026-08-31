package catalog

import (
	"context"
	"database/sql"
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
	s := &Store{db: db, now: now}
	// Keep the declarative schema useful for greenfield databases while making
	// the additive release-rank field safe for already-running SQLite stores.
	var present int
	if err := db.QueryRowContext(context.Background(), `SELECT count(*) FROM pragma_table_info('nodes') WHERE name='release_rank'`).Scan(&present); err == nil && present == 0 {
		_, _ = db.ExecContext(context.Background(), `ALTER TABLE nodes ADD COLUMN release_rank INTEGER NOT NULL DEFAULT 0`)
	}
	return s
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
	name = strings.TrimSpace(name)
	var existingID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM nodes WHERE kind=? AND name=?`, int32(kind), name).Scan(&existingID); err == nil {
		return nil, fmt.Errorf("node identity (%s,%q) already exists as %s", kind.String(), name, existingID)
	} else if err != sql.ErrNoRows {
		return nil, err
	}
	if status == offerspb.Status_CANDIDATE && strings.TrimSpace(trigger) == "" {
		return nil, errors.New("rule candidate_requires_trigger: candidate nodes require a machine-evaluable trigger")
	}
	n := &offerspb.Node{Id: uuid.NewString(), Kind: kind, Name: name, Status: status, TriggerId: trigger, ActualAccountId: actualAccountID, CreatedAt: timestamppb.New(s.now())}
	if status == offerspb.Status_CANDIDATE {
		var triggerNode string
		if err := s.db.QueryRowContext(ctx, `SELECT node_id FROM triggers WHERE id=?`, trigger).Scan(&triggerNode); err != nil || triggerNode != n.Id {
			return nil, errors.New("rule candidate_requires_trigger refused creation: candidate nodes require an attached machine-evaluable trigger")
		}
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes(id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank) VALUES(?,?,?,?,?,?,?,?)`, n.Id, int32(kind), n.Name, int32(status), trigger, n.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano), actualAccountID, 0)
	return n, err
}

func (s *Store) ListNodes(ctx context.Context, kind offerspb.NodeKind, status offerspb.Status) ([]*offerspb.Node, error) {
	q := `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank FROM nodes WHERE 1=1`
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
		if err := rows.Scan(&n.Id, &k, &n.Name, &st, &n.TriggerId, &ts, &n.ActualAccountId, &n.ReleaseRank); err != nil {
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
		return to == offerspb.Status_CANDIDATE || to == offerspb.Status_PROPOSED || to == offerspb.Status_RETIRED
	case offerspb.Status_CANDIDATE:
		return to == offerspb.Status_TRIGGER_MET || to == offerspb.Status_IDEA || to == offerspb.Status_RETIRED
	case offerspb.Status_TRIGGER_MET:
		return to == offerspb.Status_ACTIVE || to == offerspb.Status_CANDIDATE || to == offerspb.Status_RETIRED
	case offerspb.Status_PROPOSED:
		return to == offerspb.Status_ACTIVE || to == offerspb.Status_RETIRED
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
	case offerspb.Status_PROPOSED:
		return "active, retired"
	case offerspb.Status_ACTIVE:
		return "shipped, retired"
	case offerspb.Status_SHIPPED:
		return "retired"
	}
	return "none"
}

func (s *Store) Transition(ctx context.Context, id string, to offerspb.Status, actor string) (*offerspb.Node, error) {
	declineReason := ""
	if strings.HasPrefix(actor, "operator:decline:") {
		declineReason = strings.TrimSpace(strings.TrimPrefix(actor, "operator:decline:"))
		actor = "operator"
		if declineReason == "" {
			declineReason = "Operator declined the promotion and retired the node."
		}
	}
	var n offerspb.Node
	var k, st int32
	var ts string
	if err := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank FROM nodes WHERE id=?`, id).Scan(&n.Id, &k, &n.Name, &st, &n.TriggerId, &ts, &n.ActualAccountId, &n.ReleaseRank); err != nil {
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
	if err == nil && to == offerspb.Status_RETIRED && declineReason != "" {
		err = s.recordLatestProposalDecline(ctx, id, actor, declineReason)
	}
	n.Status = to
	t, _ := time.Parse(time.RFC3339Nano, ts)
	n.CreatedAt = timestamppb.New(t)
	return &n, err
}

func (s *Store) ListEdges(ctx context.Context, nodeID string) ([]*offerspb.Edge, error) {
	query := `SELECT id,from_id,to_id,kind,intended_price_minor,currency,intended_price_declared FROM edges`
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
		var declared int32
		if err := rows.Scan(&e.Id, &e.FromId, &e.ToId, &e.Kind, &e.IntendedPriceMinor, &e.Currency, &declared); err != nil {
			return nil, err
		}
		e.IntendedPriceDeclared = declared != 0
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
		(e.Kind == "requires" && from == int32(offerspb.NodeKind_OFFER) && to == int32(offerspb.NodeKind_OFFER)) ||
		(e.Kind == "unlocks" && from == int32(offerspb.NodeKind_DELIVERABLE) && (to == int32(offerspb.NodeKind_RAMP) || to == int32(offerspb.NodeKind_STREAM))) ||
		(e.Kind == "serves" && ((from == int32(offerspb.NodeKind_DELIVERABLE) && to == int32(offerspb.NodeKind_AUDIENCE)) || (from == int32(offerspb.NodeKind_AUDIENCE) && to == int32(offerspb.NodeKind_AUDIENCE))))
	if !valid {
		return nil, fmt.Errorf("rule typed_edge_matrix refused %s -> %s for edge kind %q", offerspb.NodeKind(from).String(), offerspb.NodeKind(to).String(), e.Kind)
	}
	if e.Id == "" {
		e.Id = uuid.NewString()
	}
	declared := e.IntendedPriceDeclared || e.IntendedPriceMinor != 0 || strings.TrimSpace(e.Currency) != ""
	var existingID string
	var existingPrice int64
	var existingCurrency string
	var existingDeclared int
	lookupErr := s.db.QueryRowContext(ctx, `SELECT id,intended_price_minor,currency,intended_price_declared FROM edges WHERE from_id=? AND to_id=? AND kind=?`, e.FromId, e.ToId, e.Kind).Scan(&existingID, &existingPrice, &existingCurrency, &existingDeclared)
	if lookupErr == nil {
		e.Id = existingID
		if existingDeclared != 0 && !declared {
			e.IntendedPriceMinor, e.Currency, e.IntendedPriceDeclared = existingPrice, existingCurrency, true
		} else {
			e.IntendedPriceDeclared = declared
		}
		declaredInt := 0
		if e.IntendedPriceDeclared {
			declaredInt = 1
		}
		_, err := s.db.ExecContext(ctx, `UPDATE edges SET intended_price_minor=?,currency=?,intended_price_declared=? WHERE id=?`, e.IntendedPriceMinor, e.Currency, declaredInt, e.Id)
		return e, err
	}
	if lookupErr != sql.ErrNoRows {
		return nil, lookupErr
	}
	declaredInt := 0
	if declared {
		declaredInt = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO edges(id,from_id,to_id,kind,intended_price_minor,currency,intended_price_declared) VALUES(?,?,?,?,?,?,?)`, e.Id, e.FromId, e.ToId, e.Kind, e.IntendedPriceMinor, e.Currency, declaredInt)
	return e, err
}

// MergeNodes is the only catalog operation that removes a node. The caller
// supplies the survivor explicitly; every reference move, edge collapse,
// audit write, and final delete happens in one transaction.
func (s *Store) MergeNodes(ctx context.Context, request *offerspb.MergeNodesRequest) (*offerspb.MergeNodesResponse, error) {
	if request == nil || strings.TrimSpace(request.SurvivingId) == "" || strings.TrimSpace(request.DuplicateId) == "" {
		return nil, errors.New("merge requires surviving_id and duplicate_id")
	}
	if request.SurvivingId == request.DuplicateId {
		return nil, fmt.Errorf("merge refused: surviving and duplicate ids are the same: %s", request.SurvivingId)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	type nodeRow struct {
		node         offerspb.Node
		kind, status int32
		createdAt    string
	}
	load := func(id string) (*nodeRow, error) {
		row := &nodeRow{}
		if err := tx.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank FROM nodes WHERE id=?`, id).Scan(&row.node.Id, &row.kind, &row.node.Name, &row.status, &row.node.TriggerId, &row.createdAt, &row.node.ActualAccountId, &row.node.ReleaseRank); err != nil {
			return row, fmt.Errorf("merge refused: node %q not found: %w", id, err)
		}
		row.node.Kind = offerspb.NodeKind(row.kind)
		row.node.Status = offerspb.Status(row.status)
		if parsed, parseErr := time.Parse(time.RFC3339Nano, row.createdAt); parseErr == nil {
			row.node.CreatedAt = timestamppb.New(parsed)
		}
		return row, nil
	}
	survivor, err := load(request.SurvivingId)
	if err != nil {
		return nil, err
	}
	duplicate, err := load(request.DuplicateId)
	if err != nil {
		return nil, err
	}
	if survivor.node.Kind != duplicate.node.Kind {
		return nil, fmt.Errorf("merge refused: node kinds differ: surviving %s (%s), duplicate %s (%s)", survivor.node.Kind.String(), survivor.node.Id, duplicate.node.Kind.String(), duplicate.node.Id)
	}

	response := &offerspb.MergeNodesResponse{Surviving: &survivor.node}
	type edgeRow struct {
		id, fromID, toID, kind string
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,from_id,to_id,kind FROM edges ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var edges []edgeRow
	for rows.Next() {
		var edge edgeRow
		if err := rows.Scan(&edge.id, &edge.fromID, &edge.toID, &edge.kind); err != nil {
			rows.Close()
			return nil, err
		}
		edges = append(edges, edge)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	keptByKey := make(map[string]string, len(edges))
	duplicateEdgeIDs := make(map[string]struct{})
	for _, edge := range edges {
		if edge.fromID != request.DuplicateId && edge.toID != request.DuplicateId {
			key := edge.fromID + "\x00" + edge.toID + "\x00" + edge.kind
			keptByKey[key] = edge.id
		}
	}
	for _, edge := range edges {
		if edge.fromID != request.DuplicateId && edge.toID != request.DuplicateId {
			continue
		}
		duplicateEdgeIDs[edge.id] = struct{}{}
		fromID, toID := edge.fromID, edge.toID
		if fromID == request.DuplicateId {
			fromID = request.SurvivingId
		}
		if toID == request.DuplicateId {
			toID = request.SurvivingId
		}
		key := fromID + "\x00" + toID + "\x00" + edge.kind
		if _, exists := keptByKey[key]; exists {
			response.CollapsedEdgeIds = append(response.CollapsedEdgeIds, edge.id)
			continue
		}
		keptByKey[key] = edge.id
	}
	response.MovedEdges = int32(len(duplicateEdgeIDs))

	count := func(table, column string) (int32, error) {
		var n int32
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+column+`=?`, request.DuplicateId).Scan(&n); err != nil {
			return 0, err
		}
		return n, nil
	}
	if response.MovedTriggers, err = count("triggers", "node_id"); err != nil {
		return nil, err
	}
	if response.MovedEvaluations, err = count("evaluations", "node_id"); err != nil {
		return nil, err
	}
	if response.MovedProposals, err = count("proposals", "node_id"); err != nil {
		return nil, err
	}
	if response.MovedFindings, err = count("migration_findings", "node_id"); err != nil {
		return nil, err
	}
	if request.DryRun {
		return response, nil
	}

	for _, edge := range edges {
		if _, isDuplicate := duplicateEdgeIDs[edge.id]; !isDuplicate {
			continue
		}
		collapsed := false
		for _, collapsedID := range response.CollapsedEdgeIds {
			if collapsedID == edge.id {
				collapsed = true
				break
			}
		}
		if collapsed {
			if _, err := tx.ExecContext(ctx, `DELETE FROM edges WHERE id=?`, edge.id); err != nil {
				return nil, err
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE edges SET from_id=CASE WHEN from_id=? THEN ? ELSE from_id END,to_id=CASE WHEN to_id=? THEN ? ELSE to_id END WHERE id=?`, request.DuplicateId, request.SurvivingId, request.DuplicateId, request.SurvivingId, edge.id); err != nil {
			return nil, err
		}
	}
	for _, table := range []string{"triggers", "evaluations", "proposals", "migration_findings"} {
		if _, err := tx.ExecContext(ctx, `UPDATE `+table+` SET node_id=? WHERE node_id=?`, request.SurvivingId, request.DuplicateId); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE catalog_audit SET node_id=? WHERE node_id=?`, request.SurvivingId, request.DuplicateId); err != nil {
		return nil, err
	}
	if strings.TrimSpace(survivor.node.TriggerId) == "" && strings.TrimSpace(duplicate.node.TriggerId) != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE nodes SET trigger_id=? WHERE id=?`, duplicate.node.TriggerId, request.SurvivingId); err != nil {
			return nil, err
		}
		response.Surviving.TriggerId = duplicate.node.TriggerId
	}
	reason := fmt.Sprintf("merged duplicate node: surviving_id=%s duplicate_id=%s", request.SurvivingId, request.DuplicateId)
	if _, err := tx.ExecContext(ctx, `INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at,related_node_id) VALUES(?,?,?,?,?,?,?,?)`, uuid.NewString(), request.SurvivingId, request.Actor, survivor.status, survivor.status, reason, s.now().UTC().Format(time.RFC3339Nano), request.DuplicateId); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM nodes WHERE id=?`, request.DuplicateId); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return response, nil
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
	now := s.now().UTC()
	p := &offerspb.Proposal{Id: uuid.NewString(), NodeId: nodeID, Actor: actor, RequestedStatus: status, Reason: reason, CreatedAt: timestamppb.New(now), EvidenceReference: "catalog/node/" + nodeID}
	_, err := s.db.ExecContext(ctx, `INSERT INTO proposals(id,node_id,actor,requested_status,reason,created_at) VALUES(?,?,?,?,?,?)`, p.Id, p.NodeId, p.Actor, int32(p.RequestedStatus), p.Reason, now.Format(time.RFC3339Nano))
	return p, err
}

func (s *Store) ListProposals(ctx context.Context, nodeID string, status offerspb.Status) ([]*offerspb.Proposal, error) {
	query := `SELECT id,node_id,actor,requested_status,reason,created_at FROM proposals WHERE 1=1`
	args := []any{}
	if strings.TrimSpace(nodeID) != "" {
		query += ` AND node_id=?`
		args = append(args, nodeID)
	}
	if status != offerspb.Status_STATUS_UNSPECIFIED {
		query += ` AND requested_status=?`
		args = append(args, int32(status))
	}
	query += ` ORDER BY created_at DESC,id DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var proposals []*offerspb.Proposal
	for rows.Next() {
		var p offerspb.Proposal
		var requested int32
		var created string
		if err := rows.Scan(&p.Id, &p.NodeId, &p.Actor, &requested, &p.Reason, &created); err != nil {
			return nil, err
		}
		p.RequestedStatus = offerspb.Status(requested)
		if parsed, parseErr := time.Parse(time.RFC3339Nano, created); parseErr == nil {
			p.CreatedAt = timestamppb.New(parsed)
		}
		p.EvidenceReference = "catalog/node/" + p.NodeId
		proposals = append(proposals, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, proposal := range proposals {
		declines, declineErr := s.listProposalDeclines(ctx, proposal.Id)
		if declineErr != nil {
			return nil, declineErr
		}
		proposal.DeclineHistory = declines
	}
	return proposals, nil
}

func (s *Store) RecordEvaluation(ctx context.Context, result string, nodes int, reason string, evaluatedAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO evaluation_runs(id,result,nodes_scored,reason,evaluated_at) VALUES(?,?,?,?,?)`, uuid.NewString(), result, nodes, reason, evaluatedAt.UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) LatestEvaluation(ctx context.Context) (string, int, string, time.Time, error) {
	var result string
	var scored int
	var reason, evaluated string
	if err := s.db.QueryRowContext(ctx, `SELECT result,nodes_scored,reason,evaluated_at FROM evaluation_runs ORDER BY evaluated_at DESC LIMIT 1`).Scan(&result, &scored, &reason, &evaluated); err != nil {
		return "", 0, "", time.Time{}, err
	}
	at, err := time.Parse(time.RFC3339Nano, evaluated)
	if err != nil {
		return "", 0, "", time.Time{}, err
	}
	return result, scored, reason, at, nil
}

func (s *Store) listProposalDeclines(ctx context.Context, proposalID string) ([]*offerspb.ProposalDecline, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT actor,reason,created_at FROM proposal_declines WHERE proposal_id=? ORDER BY created_at,id`, proposalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var declines []*offerspb.ProposalDecline
	for rows.Next() {
		var decline offerspb.ProposalDecline
		var created string
		if err := rows.Scan(&decline.Actor, &decline.Reason, &created); err != nil {
			return nil, err
		}
		if parsed, parseErr := time.Parse(time.RFC3339Nano, created); parseErr == nil {
			decline.CreatedAt = timestamppb.New(parsed)
		}
		declines = append(declines, &decline)
	}
	return declines, rows.Err()
}

func (s *Store) recordLatestProposalDecline(ctx context.Context, nodeID, actor, reason string) error {
	var proposalID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM proposals WHERE node_id=? ORDER BY created_at DESC,id DESC LIMIT 1`, nodeID).Scan(&proposalID); err != nil {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO proposal_declines(id,proposal_id,actor,reason,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), proposalID, actor, reason, s.now().UTC().Format(time.RFC3339Nano))
	return err
}

// MapAccount attaches (or clears) the Money Ledger account whose postings
// represent what a node actually earned.
//
// This exists because actual_account_id was previously settable only at
// CreateNode, while the operator importer writes an empty string for every
// record it materializes. The result was a catalog in which no node could ever
// be joined to actuals, so the board reported "no ledger account mapping" for
// every row and the pair's headline claim — an offer that is active and has
// earned nothing — was structurally unreachable.
//
// The change is audited like any other state change: corrections are new audit
// rows, never edits, and the prior value is returned so a caller can see what
// it replaced.
func (s *Store) MapAccount(ctx context.Context, request *offerspb.MapAccountRequest) (*offerspb.Node, string, error) {
	if request == nil || strings.TrimSpace(request.NodeId) == "" {
		return nil, "", errors.New("map-account requires node_id")
	}
	if strings.TrimSpace(request.Actor) == "" {
		return nil, "", errors.New("map-account requires an actor")
	}
	var node offerspb.Node
	var kind, status int32
	var created string
	if err := s.db.QueryRowContext(ctx, `SELECT id,kind,name,status,trigger_id,created_at,actual_account_id,release_rank FROM nodes WHERE id=?`, request.NodeId).
		Scan(&node.Id, &kind, &node.Name, &status, &node.TriggerId, &created, &node.ActualAccountId, &node.ReleaseRank); err != nil {
		return nil, "", fmt.Errorf("node %q not found: %w", request.NodeId, err)
	}
	node.Kind = offerspb.NodeKind(kind)
	node.Status = offerspb.Status(status)
	prior := node.ActualAccountId
	next := strings.TrimSpace(request.ActualAccountId)
	if prior == next {
		return &node, prior, nil
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		if next == "" {
			reason = "cleared ledger account mapping"
		} else {
			reason = "mapped node to ledger account " + next
		}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE nodes SET actual_account_id=? WHERE id=?`, next, request.NodeId); err != nil {
		return nil, "", err
	}
	priorLabel := prior
	if priorLabel == "" {
		priorLabel = "(unmapped)"
	}
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO catalog_audit(id,node_id,actor,prior_status,next_status,reason,created_at) VALUES(?,?,?,?,?,?,?)`,
		uuid.NewString(), request.NodeId, request.Actor, int32(node.Status), int32(node.Status),
		reason+" (prior: "+priorLabel+")", s.now().UTC().Format(time.RFC3339Nano)); err != nil {
		return nil, "", err
	}
	node.ActualAccountId = next
	return &node, prior, nil
}
