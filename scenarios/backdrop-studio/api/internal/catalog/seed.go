package catalog

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// The catalog is the product, so its content is versioned data rather than a Go
// literal. Each file under seed/ is one immutable seed version; the store
// applies every version an install has not seen yet. That is what makes the
// catalog upgradeable: before this, `Seed` guarded on "is the table empty?", so
// an install bootstrapped at four styles could never receive a fifth.
//
//go:embed seed/*.json
var seedFS embed.FS

const seedDir = "seed"

// seedFile is one version of the shipped catalog.
type seedFile struct {
	Version  int       `json:"version"`
	Note     string    `json:"note"`
	Surfaces []Surface `json:"surfaces"`
	Styles   []Style   `json:"styles"`
	// Reasons carries the art-direction justification for each value a seed
	// version changes, one entry per retuned parameter. Nothing reads it: it is
	// declared so the key is visibly part of the format rather than an
	// undeclared string a later reader would take for debris and delete. A
	// number chosen for a reason nobody wrote down gets re-litigated forever.
	Reasons []string `json:"retune_reasons,omitempty"`
}

// Origins a row can carry. Seed rows are ours to upgrade; operator rows are
// never overwritten and never deleted, because losing an operator's authored
// style to a routine upgrade is the worst failure this table can have.
const (
	OriginSeed     = "seed"
	OriginOperator = "operator"
)

// LoadSeeds returns every embedded seed version in ascending version order.
// Exported so a build-time test can validate the shipped content without going
// through a database.
func LoadSeeds() ([]seedFile, error) {
	entries, err := fs.ReadDir(seedFS, seedDir)
	if err != nil {
		return nil, fmt.Errorf("catalog: read seed directory: %w", err)
	}
	out := make([]seedFile, 0, len(entries))
	seen := map[int]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, readErr := seedFS.ReadFile(path.Join(seedDir, entry.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("catalog: read seed %s: %w", entry.Name(), readErr)
		}
		var file seedFile
		if err := json.Unmarshal(raw, &file); err != nil {
			return nil, fmt.Errorf("catalog: parse seed %s: %w", entry.Name(), err)
		}
		if file.Version <= 0 {
			return nil, fmt.Errorf("catalog: seed %s declares no version", entry.Name())
		}
		// The filename carries the version so a reviewer can order the diff
		// without opening every file; disagreement between the two is a
		// mistake worth failing on rather than silently preferring one.
		wanted := fmt.Sprintf("v%d.json", file.Version)
		if entry.Name() != wanted {
			return nil, fmt.Errorf("catalog: seed %s declares version %d and should be named %s", entry.Name(), file.Version, wanted)
		}
		if prior, dup := seen[file.Version]; dup {
			return nil, fmt.Errorf("catalog: seed version %d declared twice (%s and %s)", file.Version, prior, entry.Name())
		}
		seen[file.Version] = entry.Name()
		out = append(out, file)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("catalog: no seed versions are embedded")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// Seed brings an install up to the current catalog version.
//
// It applies every unapplied seed version in order, upserting each row on its
// id. A row whose origin is `operator` is left exactly as the operator wrote
// it, and no row is ever deleted — a style dropped from a later seed version
// stays on installs that already have it, because someone may have released a
// backdrop against it.
func (s *Store) Seed(ctx context.Context) error {
	if err := s.migrate(ctx); err != nil {
		return err
	}
	seeds, err := LoadSeeds()
	if err != nil {
		return err
	}
	applied, err := s.appliedSeedVersions(ctx)
	if err != nil {
		return err
	}
	for _, file := range seeds {
		if applied[file.Version] {
			continue
		}
		for _, surface := range file.Surfaces {
			if err := s.upsertSurface(ctx, surface, file.Version); err != nil {
				return fmt.Errorf("catalog: seed v%d surface %q: %w", file.Version, surface.ID, err)
			}
		}
		for _, style := range file.Styles {
			if err := s.upsertStyle(ctx, style, file.Version); err != nil {
				return fmt.Errorf("catalog: seed v%d style %q: %w", file.Version, style.ID, err)
			}
		}
		if _, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO backdrop_seed_versions(version, applied_on) VALUES(?, datetime('now'))`, file.Version); err != nil {
			return fmt.Errorf("catalog: record seed version %d: %w", file.Version, err)
		}
	}
	// The settled catalog, not each version it passed through, is what anyone
	// renders from — so subject coherence is judged here, over the final rows.
	// A style whose subject no generator depicts and which no later seed
	// version corrected would substitute a different picture under the
	// requested name, which is the failure this check exists to make impossible.
	styles, err := s.ListStyles(ctx, "", "", "", "", "")
	if err != nil {
		return fmt.Errorf("catalog: read back seeded styles: %w", err)
	}
	for _, style := range styles {
		if style.Origin == OriginOperator {
			// An operator's row was validated when they wrote it, and refusing
			// to start over one they can no longer reach would be worse than
			// carrying it.
			continue
		}
		if err := ValidateSubjectCoherence(style); err != nil {
			return fmt.Errorf("catalog: the seeded catalog is not coherent after applying every version: %w", err)
		}
	}
	return nil
}

// migrate brings an older install's schema up to what the current seed needs.
// Every statement is additive and idempotent: this runs on every start, and a
// column that already exists must not be an error.
func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS backdrop_seed_versions (version INTEGER PRIMARY KEY, applied_on TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("catalog: create seed version table: %w", err)
	}
	// SQLite has no ADD COLUMN IF NOT EXISTS, and a duplicate-column error here
	// is the expected steady state rather than a failure.
	for _, stmt := range []string{
		`ALTER TABLE backdrop_styles ADD COLUMN parent_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE backdrop_styles ADD COLUMN treatment_params TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backdrop_styles ADD COLUMN quality TEXT NOT NULL DEFAULT 'null'`,
		`ALTER TABLE backdrop_styles ADD COLUMN inks TEXT NOT NULL DEFAULT '{}'`,
		`ALTER TABLE backdrop_styles ADD COLUMN origin TEXT NOT NULL DEFAULT 'seed'`,
		`ALTER TABLE backdrop_styles ADD COLUMN seed_version INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE backdrop_surfaces ADD COLUMN origin TEXT NOT NULL DEFAULT 'seed'`,
		`ALTER TABLE backdrop_surfaces ADD COLUMN seed_version INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil && !isDuplicateColumn(err) {
			return fmt.Errorf("catalog: migrate schema: %w", err)
		}
	}
	// `payload` was a whole-style JSON copy written beside the columns it
	// duplicated. Nothing ever read it back — ListStyles reconstructs a style
	// from the typed columns — so it was a second authority for the same fact,
	// silently drifting from the first. It is dropped rather than maintained.
	//
	// This is not cosmetic: the column is NOT NULL with no default, so an
	// install created before this migration rejects every seed insert until it
	// goes. A fresh install never had it, which is exactly why the unit suite
	// could not see the failure and a real upgrade could.
	if _, err := s.db.ExecContext(ctx, `ALTER TABLE backdrop_styles DROP COLUMN payload`); err != nil && !isMissingColumn(err) {
		return fmt.Errorf("catalog: drop legacy payload column: %w", err)
	}
	return nil
}

func isDuplicateColumn(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column name")
}

func isMissingColumn(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "no such column") || strings.Contains(text, "cannot drop")
}

func (s *Store) appliedSeedVersions(ctx context.Context) (map[int]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT version FROM backdrop_seed_versions`)
	if err != nil {
		return nil, fmt.Errorf("catalog: read applied seed versions: %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[int]bool{}
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

// upsertStyle writes one seed row, refusing to disturb operator-authored work.
func (s *Store) upsertStyle(ctx context.Context, v Style, version int) error {
	if err := validateStyle(&v); err != nil {
		return err
	}
	var origin string
	err := s.db.QueryRowContext(ctx, `SELECT origin FROM backdrop_styles WHERE id=?`, v.ID).Scan(&origin)
	switch {
	case err == sql.ErrNoRows:
		return s.insertStyle(ctx, v, OriginSeed, version)
	case err != nil:
		return err
	case origin == OriginOperator:
		// The operator owns this id now. Upgrading it would silently discard
		// their art direction, which is a worse outcome than a stale row.
		return nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE backdrop_styles SET name=?,version=?,role=?,subject=?,lineage=?,strategy=?,treatments=?,placements=?,regions=?,contrast_threshold=?,scaffold=?,generation=?,parent_id=?,treatment_params=?,inks=?,quality=?,seed_version=? WHERE id=? AND origin=?`,
		v.Name, styleVersion(v), v.Role, v.Subject, v.Lineage, v.Strategy, mustJSON(v.Treatments), mustJSON(v.Placements), mustJSON(v.Regions), v.ContrastThreshold, mustJSON(v.Scaffold), mustJSON(v.Generation), v.ParentID, mustJSON(v.TreatmentParams), mustJSON(v.Inks), mustJSON(v.Quality), version, v.ID, OriginSeed)
	return err
}

func (s *Store) upsertSurface(ctx context.Context, v Surface, version int) error {
	if err := validateSurface(v); err != nil {
		return err
	}
	var origin string
	err := s.db.QueryRowContext(ctx, `SELECT origin FROM backdrop_surfaces WHERE id=?`, v.ID).Scan(&origin)
	switch {
	case err == sql.ErrNoRows:
		return s.insertSurface(ctx, v, OriginSeed, version)
	case err != nil:
		return err
	case origin == OriginOperator:
		return nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE backdrop_surfaces SET name=?,kind=?,width=?,height=?,placements=?,authority=?,confirmed_on=?,seed_version=? WHERE id=? AND origin=?`,
		v.Name, v.Kind, v.Width, v.Height, mustJSON(v.Placements), v.Authority, v.ConfirmedOn, version, v.ID, OriginSeed)
	return err
}

// SeedVersion reports the highest seed version this binary carries. `status`
// prints it so an operator can tell an install that is behind from one that is
// current without reading the database.
func SeedVersion() (int, error) {
	seeds, err := LoadSeeds()
	if err != nil {
		return 0, err
	}
	return seeds[len(seeds)-1].Version, nil
}

// AppliedSeedVersion reports the highest seed version this install has applied.
func (s *Store) AppliedSeedVersion(ctx context.Context) (int, error) {
	var v sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `SELECT MAX(version) FROM backdrop_seed_versions`).Scan(&v); err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}

// parseSeedVersionFromName is used by the seed-content test to prove the
// filename and the declared version agree for every shipped file.
func parseSeedVersionFromName(name string) (int, bool) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(name, "v"), ".json")
	n, err := strconv.Atoi(trimmed)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
