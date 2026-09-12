package brands

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (used by repository
// unit tests via testutil/db.NewSQLite) and *database.RoutedDB (used in
// production by main.go) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production impl of both Repository and
// VersionRepository. Unexported so callers depend on the interfaces — tests
// substitute fakes without reaching inside the struct.
type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production brand Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// NewSQLiteVersionRepository constructs the production VersionRepository. Backed
// by the same struct; the distinct method names (CreateVersion/ListVersions)
// avoid a collision with the brand Repository's Create/List.
func NewSQLiteVersionRepository(db SQLExecutor, clk schedule.Clock) VersionRepository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantees.
var (
	_ Repository        = (*sqliteRepository)(nil)
	_ VersionRepository = (*sqliteRepository)(nil)
)

// brandTimeFormat matches notes (RFC3339Nano), which sorts lexicographically in
// time order for a fixed zone so string range/ordering on the columns is correct.
const brandTimeFormat = time.RFC3339Nano

const (
	insertBrandSQL = `
INSERT INTO brands (id, name, description, identity, colors, typography, voice, notes, version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`
	selectBrandColumns = `id, name, description, identity, colors, typography, voice, notes, version, created_at, updated_at`

	selectBrandByIDSQL = `SELECT ` + selectBrandColumns + ` FROM brands WHERE id = ?`

	updateBrandSQL = `
UPDATE brands
SET name=?, description=?, identity=?, colors=?, typography=?, voice=?, notes=?, version=?, updated_at=?
WHERE id=?
`
	deleteBrandSQL = `DELETE FROM brands WHERE id = ?`

	insertVersionSQL = `
INSERT INTO brand_versions (id, brand_id, version, snapshot, created_at)
VALUES (?, ?, ?, ?, ?)
`
	listVersionsSQL = `
SELECT id, brand_id, version, snapshot, created_at
FROM brand_versions
WHERE brand_id = ?
ORDER BY version DESC
`
)

func (s *sqliteRepository) Create(ctx context.Context, b Brand) (Brand, error) {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = b.CreatedAt
	}
	if b.Version == 0 {
		b.Version = 1
	}

	identity, colors, typography, voice, err := marshalFacets(b)
	if err != nil {
		return Brand{}, err
	}

	if _, err := s.db.ExecContext(ctx, insertBrandSQL,
		b.ID, b.Name, b.Description, identity, colors, typography, voice, b.Notes, b.Version,
		b.CreatedAt.Format(brandTimeFormat), b.UpdatedAt.Format(brandTimeFormat),
	); err != nil {
		return Brand{}, fmt.Errorf("insert brand %q: %w", b.ID, err)
	}
	return b, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Brand, error) {
	row := s.db.QueryRowContext(ctx, selectBrandByIDSQL, id)
	b, err := scanBrand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Brand{}, ErrBrandNotFound{ID: id}
	}
	if err != nil {
		return Brand{}, fmt.Errorf("get brand %q: %w", id, err)
	}
	return b, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Brand, error) {
	query := `SELECT ` + selectBrandColumns + ` FROM brands`
	var args []any
	if filter.NameContains != "" {
		query += ` WHERE name LIKE ? COLLATE NOCASE`
		args = append(args, "%"+filter.NameContains+"%")
	}
	query += ` ORDER BY updated_at DESC, id DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	var brands []Brand
	for rows.Next() {
		b, err := scanBrand(rows)
		if err != nil {
			return nil, fmt.Errorf("scan brand: %w", err)
		}
		brands = append(brands, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate brands: %w", err)
	}
	return brands, nil
}

func (s *sqliteRepository) Update(ctx context.Context, b Brand) (Brand, error) {
	b.Version++
	b.UpdatedAt = s.clock.Now().UTC()

	identity, colors, typography, voice, err := marshalFacets(b)
	if err != nil {
		return Brand{}, err
	}

	res, err := s.db.ExecContext(ctx, updateBrandSQL,
		b.Name, b.Description, identity, colors, typography, voice, b.Notes, b.Version,
		b.UpdatedAt.Format(brandTimeFormat), b.ID,
	)
	if err != nil {
		return Brand{}, fmt.Errorf("update brand %q: %w", b.ID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Brand{}, fmt.Errorf("update brand %q rows: %w", b.ID, err)
	}
	if n == 0 {
		return Brand{}, ErrBrandNotFound{ID: b.ID}
	}
	return b, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, deleteBrandSQL, id)
	if err != nil {
		return fmt.Errorf("delete brand %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete brand %q rows: %w", id, err)
	}
	if n == 0 {
		return ErrBrandNotFound{ID: id}
	}
	return nil
}

func (s *sqliteRepository) CreateVersion(ctx context.Context, v BrandVersion) (BrandVersion, error) {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.CreatedAt.IsZero() {
		v.CreatedAt = s.clock.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, insertVersionSQL,
		v.ID, v.BrandID, v.Version, v.Snapshot, v.CreatedAt.Format(brandTimeFormat),
	); err != nil {
		return BrandVersion{}, fmt.Errorf("insert brand version %q v%d: %w", v.BrandID, v.Version, err)
	}
	return v, nil
}

func (s *sqliteRepository) ListVersions(ctx context.Context, brandID string) ([]BrandVersion, error) {
	rows, err := s.db.QueryContext(ctx, listVersionsSQL, brandID)
	if err != nil {
		return nil, fmt.Errorf("list versions for %q: %w", brandID, err)
	}
	defer rows.Close()

	var versions []BrandVersion
	for rows.Next() {
		var (
			v          BrandVersion
			createdRaw string
		)
		if err := rows.Scan(&v.ID, &v.BrandID, &v.Version, &v.Snapshot, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		created, err := time.Parse(brandTimeFormat, createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse version created_at %q: %w", createdRaw, err)
		}
		v.CreatedAt = created
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}
	return versions, nil
}

// marshalFacets serializes the four JSON facet columns for an insert/update.
func marshalFacets(b Brand) (identity, colors, typography, voice string, err error) {
	identityB, err := json.Marshal(b.Identity)
	if err != nil {
		return "", "", "", "", fmt.Errorf("marshal identity: %w", err)
	}
	colorsB, err := json.Marshal(b.Colors)
	if err != nil {
		return "", "", "", "", fmt.Errorf("marshal colors: %w", err)
	}
	typographyB, err := json.Marshal(b.Typography)
	if err != nil {
		return "", "", "", "", fmt.Errorf("marshal typography: %w", err)
	}
	voiceB, err := json.Marshal(b.Voice)
	if err != nil {
		return "", "", "", "", fmt.Errorf("marshal voice: %w", err)
	}
	return string(identityB), string(colorsB), string(typographyB), string(voiceB), nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanBrand(sc rowScanner) (Brand, error) {
	var (
		b                                               Brand
		identityRaw, colorsRaw, typographyRaw, voiceRaw string
		createdRaw, updatedRaw                          string
	)
	if err := sc.Scan(&b.ID, &b.Name, &b.Description,
		&identityRaw, &colorsRaw, &typographyRaw, &voiceRaw,
		&b.Notes, &b.Version, &createdRaw, &updatedRaw,
	); err != nil {
		return Brand{}, err
	}
	if err := unmarshalFacet(identityRaw, &b.Identity); err != nil {
		return Brand{}, fmt.Errorf("parse identity: %w", err)
	}
	if err := unmarshalFacet(colorsRaw, &b.Colors); err != nil {
		return Brand{}, fmt.Errorf("parse colors: %w", err)
	}
	if err := unmarshalFacet(typographyRaw, &b.Typography); err != nil {
		return Brand{}, fmt.Errorf("parse typography: %w", err)
	}
	if err := unmarshalFacet(voiceRaw, &b.Voice); err != nil {
		return Brand{}, fmt.Errorf("parse voice: %w", err)
	}
	created, err := time.Parse(brandTimeFormat, createdRaw)
	if err != nil {
		return Brand{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	updated, err := time.Parse(brandTimeFormat, updatedRaw)
	if err != nil {
		return Brand{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	b.CreatedAt = created
	b.UpdatedAt = updated
	return b, nil
}

// unmarshalFacet decodes a JSON facet column into dst, treating the empty
// string and the JSON null literal as a zero-valued facet.
func unmarshalFacet(raw string, dst any) error {
	if raw == "" || raw == "null" {
		return nil
	}
	return json.Unmarshal([]byte(raw), dst)
}
