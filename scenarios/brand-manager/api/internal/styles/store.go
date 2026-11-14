package styles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store is the persistence surface for container styles and product lines.
type Store interface {
	ListStyles(ctx context.Context) ([]ContainerStyle, error)
	GetStyle(ctx context.Context, id string) (ContainerStyle, error)
	GetStyleByName(ctx context.Context, name string) (ContainerStyle, error)
	CreateStyle(ctx context.Context, s ContainerStyle) (ContainerStyle, error)
	UpdateStyle(ctx context.Context, s ContainerStyle) (ContainerStyle, error)

	ListProductLines(ctx context.Context) ([]ProductLine, error)
	GetProductLineByName(ctx context.Context, name string) (ProductLine, error)
	CreateProductLine(ctx context.Context, l ProductLine) (ProductLine, error)
}

// ErrNotFound is returned when no row matches.
type ErrNotFound struct{ Name string }

func (e ErrNotFound) Error() string { return fmt.Sprintf("styles: %s not found", e.Name) }

// SQLExecutor is the narrow database surface the store depends on.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteStore struct {
	db    SQLExecutor
	clock func() time.Time
}

// NewSQLiteStore constructs the production store.
func NewSQLiteStore(db SQLExecutor, clock func() time.Time) Store {
	if clock == nil {
		clock = time.Now
	}
	return &sqliteStore{db: db, clock: clock}
}

const styleColumns = `id, name, shape, corner_ratio, background_kind, background_top, background_bottom, mark_scale, maskable_scale, accent_color, glow, small_mark_threshold_px, created_at, updated_at`

func (s *sqliteStore) CreateStyle(ctx context.Context, st ContainerStyle) (ContainerStyle, error) {
	if st.ID == "" {
		st.ID = uuid.NewString()
	}
	now := s.clock().UTC()
	if st.CreatedAt.IsZero() {
		st.CreatedAt = now
	}
	st.UpdatedAt = st.CreatedAt
	glow, err := marshalGlow(st.Glow)
	if err != nil {
		return ContainerStyle{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO container_styles (`+styleColumns+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		st.ID, st.Name, st.Shape, st.CornerRatio, st.BackgroundKind, st.BackgroundTop, st.BackgroundBottom,
		st.MarkScale, st.MaskableScale, st.AccentColor, glow, st.SmallMarkThresholdPx,
		st.CreatedAt.Format(time.RFC3339Nano), st.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return ContainerStyle{}, fmt.Errorf("insert container style %q: %w", st.Name, err)
	}
	return st, nil
}

func (s *sqliteStore) UpdateStyle(ctx context.Context, st ContainerStyle) (ContainerStyle, error) {
	now := s.clock().UTC()
	st.UpdatedAt = now
	glow, err := marshalGlow(st.Glow)
	if err != nil {
		return ContainerStyle{}, err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE container_styles SET name=?, shape=?, corner_ratio=?, background_kind=?, background_top=?, background_bottom=?, mark_scale=?, maskable_scale=?, accent_color=?, glow=?, small_mark_threshold_px=?, updated_at=? WHERE id=?`,
		st.Name, st.Shape, st.CornerRatio, st.BackgroundKind, st.BackgroundTop, st.BackgroundBottom,
		st.MarkScale, st.MaskableScale, st.AccentColor, glow, st.SmallMarkThresholdPx,
		st.UpdatedAt.Format(time.RFC3339Nano), st.ID)
	if err != nil {
		return ContainerStyle{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ContainerStyle{}, ErrNotFound{Name: "container style"}
	}
	return st, nil
}

func (s *sqliteStore) GetStyle(ctx context.Context, id string) (ContainerStyle, error) {
	return scanStyle(s.db.QueryRowContext(ctx, `SELECT `+styleColumns+` FROM container_styles WHERE id = ?`, id))
}

func (s *sqliteStore) GetStyleByName(ctx context.Context, name string) (ContainerStyle, error) {
	return scanStyle(s.db.QueryRowContext(ctx, `SELECT `+styleColumns+` FROM container_styles WHERE name = ?`, name))
}

func (s *sqliteStore) ListStyles(ctx context.Context) ([]ContainerStyle, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+styleColumns+` FROM container_styles ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ContainerStyle
	for rows.Next() {
		st, err := scanStyle(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

func (s *sqliteStore) CreateProductLine(ctx context.Context, l ProductLine) (ProductLine, error) {
	if l.ID == "" {
		l.ID = uuid.NewString()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = s.clock().UTC()
	}
	products, err := json.Marshal(l.Products)
	if err != nil {
		return ProductLine{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO product_lines (id, name, container_style_id, products, created_at) VALUES (?,?,?,?,?)`,
		l.ID, l.Name, l.ContainerStyleID, string(products), l.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return ProductLine{}, fmt.Errorf("insert product line %q: %w", l.Name, err)
	}
	return l, nil
}

func (s *sqliteStore) GetProductLineByName(ctx context.Context, name string) (ProductLine, error) {
	var l ProductLine
	var products, created string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, container_style_id, products, created_at FROM product_lines WHERE name = ?`, name).
		Scan(&l.ID, &l.Name, &l.ContainerStyleID, &products, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return ProductLine{}, ErrNotFound{Name: "product line"}
	}
	if err != nil {
		return ProductLine{}, err
	}
	_ = json.Unmarshal([]byte(products), &l.Products)
	l.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	return l, nil
}

func (s *sqliteStore) ListProductLines(ctx context.Context) ([]ProductLine, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, container_style_id, products, created_at FROM product_lines ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ProductLine
	for rows.Next() {
		var l ProductLine
		var products, created string
		if err := rows.Scan(&l.ID, &l.Name, &l.ContainerStyleID, &products, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(products), &l.Products)
		l.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		out = append(out, l)
	}
	return out, rows.Err()
}

type rowScanner interface{ Scan(dest ...any) error }

func scanStyle(sc rowScanner) (ContainerStyle, error) {
	var st ContainerStyle
	var glow, created, updated string
	err := sc.Scan(&st.ID, &st.Name, &st.Shape, &st.CornerRatio, &st.BackgroundKind, &st.BackgroundTop, &st.BackgroundBottom,
		&st.MarkScale, &st.MaskableScale, &st.AccentColor, &glow, &st.SmallMarkThresholdPx, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return ContainerStyle{}, ErrNotFound{Name: "container style"}
	}
	if err != nil {
		return ContainerStyle{}, err
	}
	_ = json.Unmarshal([]byte(glow), &st.Glow)
	st.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	st.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return st, nil
}

func marshalGlow(g []GlowLayer) (string, error) {
	b, err := json.Marshal(g)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
