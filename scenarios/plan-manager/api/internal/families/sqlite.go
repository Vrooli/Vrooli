package families

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	familiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/families"
	"google.golang.org/protobuf/encoding/protojson"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct{ db SQLExecutor }

const timeFormat = time.RFC3339Nano

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, family *familiesv1.PlanFamily) error {
	raw, err := protojson.Marshal(family)
	if err != nil {
		return fmt.Errorf("marshal family: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO plan_families (family_id, slug, revision, document, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, family.GetFamilyId(), family.GetSlug(), family.GetRevision(), raw, family.GetCreatedAt().AsTime().Format(timeFormat), family.GetUpdatedAt().AsTime().Format(timeFormat))
	if err != nil {
		return fmt.Errorf("create family: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (*familiesv1.PlanFamily, error) {
	var raw []byte
	if err := r.db.QueryRowContext(ctx, `SELECT document FROM plan_families WHERE family_id = ?`, id).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get family: %w", err)
	}
	family := &familiesv1.PlanFamily{}
	if err := protojson.Unmarshal(raw, family); err != nil {
		return nil, fmt.Errorf("decode family: %w", err)
	}
	return family, nil
}

func (r *sqliteRepository) List(ctx context.Context, size uint32, token string) ([]*familiesv1.PlanFamily, string, error) {
	if size == 0 || size > 100 {
		size = 50
	}
	offset := 0
	if token != "" {
		parsed, err := strconv.Atoi(token)
		if err != nil || parsed < 0 {
			return nil, "", fmt.Errorf("invalid page token")
		}
		offset = parsed
	}
	rows, err := r.db.QueryContext(ctx, `SELECT document FROM plan_families ORDER BY updated_at DESC, family_id LIMIT ? OFFSET ?`, size+1, offset)
	if err != nil {
		return nil, "", fmt.Errorf("list families: %w", err)
	}
	defer rows.Close()
	items := make([]*familiesv1.PlanFamily, 0, size)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, "", err
		}
		item := &familiesv1.PlanFamily{}
		if err := protojson.Unmarshal(raw, item); err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > int(size) {
		items = items[:size]
		next = strconv.Itoa(offset + int(size))
	}
	return items, next, nil
}

func (r *sqliteRepository) Commit(ctx context.Context, family *familiesv1.PlanFamily, expected uint64, graph *familiesv1.GraphRevision, review *familiesv1.GraphReview) error {
	beginner, ok := r.db.(interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	})
	if !ok {
		return errors.New("families repository requires transactional database")
	}
	tx, err := beginner.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if graph != nil {
		raw, marshalErr := protojson.Marshal(graph)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO plan_family_graph_revisions (family_id, graph_revision, document, created_at) VALUES (?, ?, ?, ?)`, family.GetFamilyId(), graph.GetRevision(), raw, graph.GetCreatedAt().AsTime().Format(timeFormat)); err != nil {
			return fmt.Errorf("append graph revision: %w", err)
		}
	}
	if review != nil {
		raw, marshalErr := protojson.Marshal(review)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO plan_family_reviews (family_id, graph_revision, document, reviewed_at) VALUES (?, ?, ?, ?)`, family.GetFamilyId(), review.GetGraphRevision(), raw, review.GetReviewedAt().AsTime().Format(timeFormat)); err != nil {
			return fmt.Errorf("append graph review: %w", err)
		}
	}
	raw, err := protojson.Marshal(family)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE plan_families SET slug = ?, revision = ?, document = ?, updated_at = ? WHERE family_id = ? AND revision = ?`, family.GetSlug(), family.GetRevision(), raw, family.GetUpdatedAt().AsTime().Format(timeFormat), family.GetFamilyId(), expected)
	if err != nil {
		return fmt.Errorf("update family: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return ErrConflict
	}
	return tx.Commit()
}
