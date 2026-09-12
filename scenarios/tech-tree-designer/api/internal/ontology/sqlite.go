package ontology

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type sqlDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct {
	db  sqlDB
	now func() time.Time
}

func NewSQLiteRepository(db sqlDB) *SQLiteRepository {
	return &SQLiteRepository{db: db, now: func() time.Time { return time.Now().UTC() }}
}

var _ Repository = (*SQLiteRepository)(nil)

func (r *SQLiteRepository) ListCapabilities(ctx context.Context, filter CapabilityFilter) ([]Capability, error) {
	query := `SELECT id, slug, name, description, kind, COALESCE(parent_id, ''), sort_order, importance, created_at, updated_at FROM capability`
	args := []any{}
	where := ""
	if filter.ParentID != "" && !filter.IncludeDescendants {
		where = appendWhere(where, `parent_id = ?`)
		args = append(args, filter.ParentID)
	}
	if filter.Kind != 0 && !filter.IncludeDescendants {
		where = appendWhere(where, `kind = ?`)
		args = append(args, KindToStorage(filter.Kind))
	}
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY parent_id, sort_order, slug`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()
	var out []Capability
	for rows.Next() {
		capability, err := scanCapability(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, capability)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate capabilities: %w", err)
	}
	if filter.IncludeDescendants && filter.ParentID != "" {
		out = includeDescendants(out, filter.ParentID)
		if filter.Kind != 0 {
			filtered := out[:0]
			for _, capability := range out {
				if capability.Kind == filter.Kind {
					filtered = append(filtered, capability)
				}
			}
			out = filtered
		}
	}
	return out, nil
}

func (r *SQLiteRepository) GetCapability(ctx context.Context, ref CapabilityRef) (Capability, error) {
	query := `SELECT id, slug, name, description, kind, COALESCE(parent_id, ''), sort_order, importance, created_at, updated_at FROM capability WHERE id = ?`
	arg := ref.ID
	if arg == "" {
		query = `SELECT id, slug, name, description, kind, COALESCE(parent_id, ''), sort_order, importance, created_at, updated_at FROM capability WHERE slug = ?`
		arg = ref.Slug
	}
	return scanCapability(r.db.QueryRowContext(ctx, query, arg))
}

func (r *SQLiteRepository) UpsertCapability(ctx context.Context, capability Capability) (Capability, error) {
	now := r.now().Format(time.RFC3339Nano)
	var parent any
	if capability.ParentID != "" {
		parent = capability.ParentID
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO capability (id, slug, name, description, kind, parent_id, sort_order, importance, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    slug = excluded.slug,
    name = excluded.name,
    description = excluded.description,
    kind = excluded.kind,
    parent_id = excluded.parent_id,
    sort_order = excluded.sort_order,
    importance = excluded.importance,
    updated_at = excluded.updated_at`,
		capability.ID, capability.Slug, capability.Name, capability.Description, KindToStorage(capability.Kind), parent, capability.SortOrder, capability.Importance, now, now)
	if err != nil {
		return Capability{}, fmt.Errorf("upsert capability: %w", err)
	}
	return r.GetCapability(ctx, CapabilityRef{ID: capability.ID})
}

func (r *SQLiteRepository) DeleteCapability(ctx context.Context, ref CapabilityRef) (bool, error) {
	query := `DELETE FROM capability WHERE id = ?`
	arg := ref.ID
	if arg == "" {
		query = `DELETE FROM capability WHERE slug = ?`
		arg = ref.Slug
	}
	res, err := r.db.ExecContext(ctx, query, arg)
	if err != nil {
		return false, fmt.Errorf("delete capability: %w", err)
	}
	return rowsAffected(res)
}

func (r *SQLiteRepository) UpsertCapabilityEdge(ctx context.Context, edge CapabilityEdge) (CapabilityEdge, error) {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO capability_edge (from_id, to_id, type)
VALUES (?, ?, ?)
ON CONFLICT(from_id, to_id, type) DO NOTHING`,
		edge.FromID, edge.ToID, EdgeTypeToStorage(edge.Type))
	if err != nil {
		return CapabilityEdge{}, fmt.Errorf("upsert capability edge: %w", err)
	}
	return edge, nil
}

func (r *SQLiteRepository) DeleteCapabilityEdge(ctx context.Context, edge CapabilityEdge) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM capability_edge WHERE from_id = ? AND to_id = ? AND type = ?`,
		edge.FromID, edge.ToID, EdgeTypeToStorage(edge.Type))
	if err != nil {
		return false, fmt.Errorf("delete capability edge: %w", err)
	}
	return rowsAffected(res)
}

func (r *SQLiteRepository) ListCapabilityEdges(ctx context.Context) ([]CapabilityEdge, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT from_id, to_id, type FROM capability_edge ORDER BY from_id, to_id, type`)
	if err != nil {
		return nil, fmt.Errorf("list capability edges: %w", err)
	}
	defer rows.Close()
	var out []CapabilityEdge
	for rows.Next() {
		var edge CapabilityEdge
		var edgeType string
		if err := rows.Scan(&edge.FromID, &edge.ToID, &edgeType); err != nil {
			return nil, fmt.Errorf("scan capability edge: %w", err)
		}
		edge.Type = EdgeTypeFromStorage(edgeType)
		out = append(out, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate capability edges: %w", err)
	}
	return out, nil
}

func (r *SQLiteRepository) LinkFulfillment(ctx context.Context, fulfillment Fulfillment) (Fulfillment, error) {
	now := r.now().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `
INSERT INTO fulfillment (capability_id, scenario_slug, note, created_at)
VALUES (?, ?, ?, ?)
ON CONFLICT(capability_id, scenario_slug) DO UPDATE SET note = excluded.note`,
		fulfillment.CapabilityID, fulfillment.ScenarioSlug, fulfillment.Note, now)
	if err != nil {
		return Fulfillment{}, fmt.Errorf("link fulfillment: %w", err)
	}
	rows, err := r.ListFulfillments(ctx, FulfillmentFilter{CapabilityID: fulfillment.CapabilityID, ScenarioSlug: fulfillment.ScenarioSlug})
	if err != nil {
		return Fulfillment{}, err
	}
	if len(rows) == 0 {
		return Fulfillment{}, sql.ErrNoRows
	}
	return rows[0], nil
}

func (r *SQLiteRepository) UnlinkFulfillment(ctx context.Context, capabilityID, scenarioSlug string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM fulfillment WHERE capability_id = ? AND scenario_slug = ?`, capabilityID, scenarioSlug)
	if err != nil {
		return false, fmt.Errorf("unlink fulfillment: %w", err)
	}
	return rowsAffected(res)
}

func (r *SQLiteRepository) ListFulfillments(ctx context.Context, filter FulfillmentFilter) ([]Fulfillment, error) {
	query := `SELECT capability_id, scenario_slug, note, created_at FROM fulfillment`
	args := []any{}
	where := ""
	if filter.CapabilityID != "" {
		where = appendWhere(where, `capability_id = ?`)
		args = append(args, filter.CapabilityID)
	}
	if filter.ScenarioSlug != "" {
		where = appendWhere(where, `scenario_slug = ?`)
		args = append(args, filter.ScenarioSlug)
	}
	if where != "" {
		query += ` WHERE ` + where
	}
	query += ` ORDER BY capability_id, scenario_slug`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list fulfillments: %w", err)
	}
	defer rows.Close()
	var out []Fulfillment
	for rows.Next() {
		fulfillment, err := scanFulfillment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, fulfillment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fulfillments: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCapability(row rowScanner) (Capability, error) {
	var capability Capability
	var kind, created, updated string
	if err := row.Scan(&capability.ID, &capability.Slug, &capability.Name, &capability.Description, &kind, &capability.ParentID, &capability.SortOrder, &capability.Importance, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Capability{}, err
		}
		return Capability{}, fmt.Errorf("scan capability: %w", err)
	}
	capability.Kind = KindFromStorage(kind)
	capability.CreatedAt = parseTime(created)
	capability.UpdatedAt = parseTime(updated)
	return capability, nil
}

func scanFulfillment(row rowScanner) (Fulfillment, error) {
	var fulfillment Fulfillment
	var created string
	if err := row.Scan(&fulfillment.CapabilityID, &fulfillment.ScenarioSlug, &fulfillment.Note, &created); err != nil {
		return Fulfillment{}, fmt.Errorf("scan fulfillment: %w", err)
	}
	fulfillment.CreatedAt = parseTime(created)
	return fulfillment, nil
}

func appendWhere(existing, clause string) string {
	if existing == "" {
		return clause
	}
	return existing + ` AND ` + clause
}

func rowsAffected(res sql.Result) (bool, error) {
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected row count: %w", err)
	}
	return n > 0, nil
}

func includeDescendants(capabilities []Capability, rootID string) []Capability {
	byParent := map[string][]Capability{}
	for _, capability := range capabilities {
		byParent[capability.ParentID] = append(byParent[capability.ParentID], capability)
	}
	out := make([]Capability, 0, len(capabilities))
	var visit func(string)
	visit = func(parentID string) {
		for _, child := range byParent[parentID] {
			out = append(out, child)
			visit(child.ID)
		}
	}
	visit(rootID)
	return out
}
