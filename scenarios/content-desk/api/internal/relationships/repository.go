package relationships

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type Repository interface {
	CreateRelation(context.Context, Relation) (Relation, error)
	ListRelations(context.Context, string) ([]Relation, error)
	CreateCollection(context.Context, string, string) (Collection, error)
	ListCollections(context.Context, string) ([]Collection, error)
	AddCollectionItem(context.Context, CollectionItem) error
	RemoveCollectionItem(context.Context, CollectionItem) error
	ListCollectionItems(context.Context, string) ([]CollectionItem, error)
	PreviewImpact(context.Context, string, string) (Impact, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) CreateRelation(ctx context.Context, rel Relation) (Relation, error) {
	rel.SourceDraftID, rel.SourceRevisionID = strings.TrimSpace(rel.SourceDraftID), strings.TrimSpace(rel.SourceRevisionID)
	rel.TargetDraftID, rel.TargetRevisionID = strings.TrimSpace(rel.TargetDraftID), strings.TrimSpace(rel.TargetRevisionID)
	rel.ProjectID, rel.Kind = strings.TrimSpace(rel.ProjectID), strings.TrimSpace(rel.Kind)
	if rel.SourceDraftID == "" || rel.SourceRevisionID == "" || rel.TargetDraftID == "" || rel.TargetRevisionID == "" || rel.ProjectID == "" || (rel.Kind != "related" && rel.Kind != "family") || (rel.Kind != "family" && rel.Primary) || rel.SourceDraftID == rel.TargetDraftID {
		return Relation{}, ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Relation{}, err
	}
	defer tx.Rollback()
	for _, pair := range [][2]string{{rel.SourceDraftID, rel.SourceRevisionID}, {rel.TargetDraftID, rel.TargetRevisionID}} {
		var project string
		err = tx.QueryRowContext(ctx, `SELECT p.project_id FROM draft_project_refs p JOIN drafts d ON d.id = p.draft_id JOIN draft_revisions v ON v.draft_id = d.id WHERE d.id = ? AND v.id = ?`, pair[0], pair[1]).Scan(&project)
		if err != nil {
			return Relation{}, fmt.Errorf("resolve pinned draft revision: %w", ErrNotFound)
		}
		if project != rel.ProjectID {
			return Relation{}, fmt.Errorf("drafts must belong to the requested same project: %w", ErrInvalid)
		}
	}
	rel.ID = uuid.NewString()
	_, err = tx.ExecContext(ctx, `INSERT INTO content_relationships (id, source_draft_id, source_revision_id, target_draft_id, target_revision_id, project_id, relation_kind, is_primary, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, rel.ID, rel.SourceDraftID, rel.SourceRevisionID, rel.TargetDraftID, rel.TargetRevisionID, rel.ProjectID, rel.Kind, boolInt(rel.Primary), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Relation{}, fmt.Errorf("insert relationship: %w", ErrConflict)
	}
	if err = tx.Commit(); err != nil {
		return Relation{}, err
	}
	return rel, nil
}

func (r *sqliteRepository) ListRelations(ctx context.Context, draftID string) ([]Relation, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, source_draft_id, source_revision_id, target_draft_id, target_revision_id, project_id, relation_kind, is_primary FROM content_relationships WHERE source_draft_id = ? OR target_draft_id = ? ORDER BY created_at, id`, draftID, draftID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Relation
	for rows.Next() {
		var x Relation
		var primary int
		if err := rows.Scan(&x.ID, &x.SourceDraftID, &x.SourceRevisionID, &x.TargetDraftID, &x.TargetRevisionID, &x.ProjectID, &x.Kind, &primary); err != nil {
			return nil, err
		}
		x.Primary = primary != 0
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) CreateCollection(ctx context.Context, projectID, name string) (Collection, error) {
	projectID, name = strings.TrimSpace(projectID), strings.TrimSpace(name)
	if projectID == "" || name == "" {
		return Collection{}, ErrInvalid
	}
	var active int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM projects WHERE id = ? AND status = 'active'`, projectID).Scan(&active); err != nil {
		return Collection{}, err
	}
	if active != 1 {
		return Collection{}, ErrNotFound
	}
	c := Collection{ID: uuid.NewString(), ProjectID: projectID, Name: name}
	_, err := r.db.ExecContext(ctx, `INSERT INTO saved_collections(id,project_id,name,created_at) VALUES(?,?,?,?)`, c.ID, c.ProjectID, c.Name, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Collection{}, err
	}
	return c, nil
}

func (r *sqliteRepository) ListCollections(ctx context.Context, projectID string) ([]Collection, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, ErrInvalid
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id,project_id,name FROM saved_collections WHERE project_id = ? ORDER BY name COLLATE NOCASE,id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Collection
	for rows.Next() {
		var item Collection
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) AddCollectionItem(ctx context.Context, item CollectionItem) error {
	var collectionProject, draftProject string
	if err := r.db.QueryRowContext(ctx, `SELECT project_id FROM saved_collections WHERE id = ?`, item.CollectionID).Scan(&collectionProject); err != nil {
		return ErrNotFound
	}
	if err := r.db.QueryRowContext(ctx, `SELECT p.project_id FROM draft_project_refs p JOIN draft_revisions v ON v.draft_id = p.draft_id WHERE p.draft_id = ? AND v.id = ?`, item.DraftID, item.RevisionID).Scan(&draftProject); err != nil {
		return ErrNotFound
	}
	if collectionProject != draftProject {
		return ErrInvalid
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO saved_collection_items(collection_id,draft_id,revision_id,created_at) VALUES(?,?,?,?)`, item.CollectionID, item.DraftID, item.RevisionID, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func (r *sqliteRepository) RemoveCollectionItem(ctx context.Context, item CollectionItem) error {
	item.CollectionID, item.DraftID, item.RevisionID = strings.TrimSpace(item.CollectionID), strings.TrimSpace(item.DraftID), strings.TrimSpace(item.RevisionID)
	if item.CollectionID == "" || item.DraftID == "" || item.RevisionID == "" {
		return ErrInvalid
	}
	result, err := r.db.ExecContext(ctx, `DELETE FROM saved_collection_items WHERE collection_id = ? AND draft_id = ? AND revision_id = ?`, item.CollectionID, item.DraftID, item.RevisionID)
	if err != nil {
		return err
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *sqliteRepository) ListCollectionItems(ctx context.Context, collectionID string) ([]CollectionItem, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT collection_id,draft_id,revision_id FROM saved_collection_items WHERE collection_id = ? ORDER BY created_at,draft_id,revision_id`, collectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CollectionItem
	for rows.Next() {
		var i CollectionItem
		if err := rows.Scan(&i.CollectionID, &i.DraftID, &i.RevisionID); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) PreviewImpact(ctx context.Context, draftID, operation string) (Impact, error) {
	if operation != "detach" && operation != "move" && operation != "archive" {
		return Impact{}, ErrInvalid
	}
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM drafts WHERE id = ?`, draftID).Scan(&exists); err != nil {
		return Impact{}, err
	}
	if exists == 0 {
		return Impact{}, ErrNotFound
	}
	i := Impact{DraftID: draftID, Operation: operation}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM content_relationships WHERE source_draft_id = ? OR target_draft_id = ?`, draftID, draftID).Scan(&i.Relationships); err != nil {
		return Impact{}, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM saved_collection_items WHERE draft_id = ?`, draftID).Scan(&i.CollectionMemberships); err != nil {
		return Impact{}, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM draft_attachments WHERE draft_id = ?`, draftID).Scan(&i.Attachments); err != nil {
		return Impact{}, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM draft_slots WHERE draft_id = ?)`, draftID).Scan(&i.HasDraftSlot); err != nil {
		return Impact{}, err
	}
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM draft_approvals WHERE draft_id = ?)`, draftID).Scan(&i.HasApproval); err != nil {
		return Impact{}, err
	}
	return i, nil
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
