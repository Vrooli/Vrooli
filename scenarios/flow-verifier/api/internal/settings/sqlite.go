package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

const settingsTimeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository. db is the
// connection pool opened in main.go; clk supplies UpdatedAt on Upsert
// so tests can advance time deterministically.
func NewSQLiteRepository(db *sql.DB, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const selectSettingsSQL = `
SELECT principal_id, theme, font_scale, reduced_motion, rtl,
       default_root, density, sidebar_width, inventory_filters, updated_at
FROM user_settings
WHERE principal_id = ?
`

// upsertSettingsSQL is intentionally a single statement. The ON CONFLICT
// clause means callers don't have to branch on "row exists yet" — the
// service's merge logic happens in Go before this statement runs.
const upsertSettingsSQL = `
INSERT INTO user_settings (
  principal_id, theme, font_scale, reduced_motion, rtl,
  default_root, density, sidebar_width, inventory_filters, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(principal_id) DO UPDATE SET
  theme             = excluded.theme,
  font_scale        = excluded.font_scale,
  reduced_motion    = excluded.reduced_motion,
  rtl               = excluded.rtl,
  default_root      = excluded.default_root,
  density           = excluded.density,
  sidebar_width     = excluded.sidebar_width,
  inventory_filters = excluded.inventory_filters,
  updated_at        = excluded.updated_at
`

func (r *sqliteRepository) Get(ctx context.Context, principalID string) (Settings, error) {
	if principalID == "" {
		principalID = PrincipalLocal
	}
	row := r.db.QueryRowContext(ctx, selectSettingsSQL, principalID)
	got, err := scanSettings(row)
	if errors.Is(err, sql.ErrNoRows) {
		out := DefaultSettings()
		out.PrincipalID = principalID
		return out, nil
	}
	if err != nil {
		return Settings{}, fmt.Errorf("get user_settings %q: %w", principalID, err)
	}
	return got, nil
}

func (r *sqliteRepository) Upsert(ctx context.Context, s Settings) (Settings, error) {
	if s.PrincipalID == "" {
		s.PrincipalID = PrincipalLocal
	}
	s.UpdatedAt = r.clock.Now().UTC()
	filtersJSON, err := json.Marshal(s.InventoryFilters)
	if err != nil {
		return Settings{}, fmt.Errorf("marshal inventory_filters: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, upsertSettingsSQL,
		s.PrincipalID,
		string(s.Theme),
		string(s.FontScale),
		boolToInt(s.ReducedMotion),
		boolToInt(s.RTL),
		s.DefaultRoot,
		string(s.Density),
		s.SidebarWidth,
		string(filtersJSON),
		s.UpdatedAt.Format(settingsTimeFormat),
	); err != nil {
		return Settings{}, fmt.Errorf("upsert user_settings %q: %w", s.PrincipalID, err)
	}
	return s, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSettings(s rowScanner) (Settings, error) {
	var (
		out           Settings
		theme         string
		fontScale     string
		reducedMotion int64
		rtl           int64
		density       string
		filtersRaw    string
		updatedRaw    string
	)
	if err := s.Scan(
		&out.PrincipalID, &theme, &fontScale, &reducedMotion, &rtl,
		&out.DefaultRoot, &density, &out.SidebarWidth, &filtersRaw, &updatedRaw,
	); err != nil {
		return Settings{}, err
	}
	out.Theme = Theme(theme)
	out.FontScale = FontScale(fontScale)
	out.ReducedMotion = reducedMotion != 0
	out.RTL = rtl != 0
	out.Density = Density(density)
	if err := json.Unmarshal([]byte(filtersRaw), &out.InventoryFilters); err != nil {
		return Settings{}, fmt.Errorf("decode inventory_filters: %w", err)
	}
	// SQLite's CURRENT_TIMESTAMP default is "YYYY-MM-DD HH:MM:SS" UTC;
	// our own writes use RFC3339Nano. Accept both.
	if t, err := time.Parse(settingsTimeFormat, updatedRaw); err == nil {
		out.UpdatedAt = t
	} else if t, err2 := time.Parse("2006-01-02 15:04:05", updatedRaw); err2 == nil {
		out.UpdatedAt = t.UTC()
	} else {
		return Settings{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	return out, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
