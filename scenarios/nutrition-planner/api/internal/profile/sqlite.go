package profile

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	sqliteRepository struct {
		db      SQLExecutor
		clock   schedule.Clock
		initErr error
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	r := &sqliteRepository{db: db, clock: clock}
	r.initErr = r.ensureEquipmentColumns(context.Background())
	return r
}

func (r *sqliteRepository) ensureEquipmentColumns(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, `PRAGMA table_info(profiles)`)
	if err != nil {
		return fmt.Errorf("inspect profile schema: %w", err)
	}
	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, kind string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read profile schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("read profile schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close profile schema: %w", err)
	}
	for _, column := range []string{"cookware_json", "tools_json", "kitchen_devices_json"} {
		if columns[column] {
			continue
		}
		if _, err := r.db.ExecContext(ctx, `ALTER TABLE profiles ADD COLUMN `+column+` TEXT NOT NULL DEFAULT 'null'`); err != nil {
			return fmt.Errorf("add profile column %s: %w", column, err)
		}
	}
	return nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Profile, error) {
	if r.initErr != nil {
		return Profile{}, r.initErr
	}
	row := r.db.QueryRowContext(ctx, `SELECT workspace_id,revision,preset,preset_version,active_rules_json,excluded_groups_json,allergies_json,appliances_json,cookware_json,tools_json,kitchen_devices_json,cost_weight,effort_weight,variety_weight,draft_json FROM profiles WHERE workspace_id=?`, id)
	profile, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Profile{}, ErrNotFound{WorkspaceID: id}
	}
	return profile, err
}

func (r *sqliteRepository) SaveDraft(ctx context.Context, id, draft string) (Profile, error) {
	cur, err := r.Get(ctx, id)
	if err != nil {
		var missing ErrNotFound
		if !errors.As(err, &missing) {
			return Profile{}, err
		}
		cur = Profile{WorkspaceID: id, Revision: 0, Preset: "everything", PresetVersion: CurrentPresetVersion, CostWeight: .34, EffortWeight: .33, VarietyWeight: .33}
	}
	cur.Revision++
	cur.DraftJSON = draft
	if err := r.write(ctx, cur); err != nil {
		return Profile{}, err
	}
	return cur, nil
}

func (r *sqliteRepository) Apply(ctx context.Context, in ApplyInput) (Profile, error) {
	cur, err := r.Get(ctx, in.WorkspaceID)
	if err != nil {
		var missing ErrNotFound
		if !errors.As(err, &missing) {
			return Profile{}, err
		}
	}
	cur.WorkspaceID = in.WorkspaceID
	cur.Revision++
	cur.Preset = in.Preset
	cur.PresetVersion = CurrentPresetVersion
	cur.ActiveRules = ExpandPreset(in.Preset)
	cur.ExcludedGroups = in.ExcludedGroups
	cur.Allergies = in.Allergies
	cur.Appliances = in.Appliances
	cur.Cookware = in.Cookware
	cur.Tools = in.Tools
	if in.KitchenDevices != nil {
		cur.KitchenDevices = in.KitchenDevices
	}
	cur.CostWeight = in.CostWeight
	cur.EffortWeight = in.EffortWeight
	cur.VarietyWeight = in.VarietyWeight
	if cur.CostWeight == 0 && cur.EffortWeight == 0 && cur.VarietyWeight == 0 {
		cur.CostWeight = .34
		cur.EffortWeight = .33
		cur.VarietyWeight = .33
	}
	if err := r.write(ctx, cur); err != nil {
		return Profile{}, err
	}
	return cur, nil
}

func (r *sqliteRepository) write(ctx context.Context, p Profile) error {
	b := func(v []string) string { x, _ := json.Marshal(v); return string(x) }
	devices, _ := json.Marshal(p.KitchenDevices)
	_, err := r.db.ExecContext(ctx, `INSERT INTO profiles(workspace_id,revision,preset,preset_version,active_rules_json,excluded_groups_json,allergies_json,appliances_json,cookware_json,tools_json,kitchen_devices_json,cost_weight,effort_weight,variety_weight,draft_json,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?) ON CONFLICT(workspace_id) DO UPDATE SET revision=excluded.revision,preset=excluded.preset,preset_version=excluded.preset_version,active_rules_json=excluded.active_rules_json,excluded_groups_json=excluded.excluded_groups_json,allergies_json=excluded.allergies_json,appliances_json=excluded.appliances_json,cookware_json=excluded.cookware_json,tools_json=excluded.tools_json,kitchen_devices_json=excluded.kitchen_devices_json,cost_weight=excluded.cost_weight,effort_weight=excluded.effort_weight,variety_weight=excluded.variety_weight,draft_json=excluded.draft_json,updated_at=excluded.updated_at`, p.WorkspaceID, p.Revision, p.Preset, p.PresetVersion, b(p.ActiveRules), b(p.ExcludedGroups), b(p.Allergies), b(p.Appliances), b(p.Cookware), b(p.Tools), string(devices), strconv.FormatFloat(p.CostWeight, 'f', -1, 64), strconv.FormatFloat(p.EffortWeight, 'f', -1, 64), strconv.FormatFloat(p.VarietyWeight, 'f', -1, 64), p.DraftJSON, r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func scan(s interface{ Scan(...any) error }) (Profile, error) {
	var p Profile
	var a, b, c, d, cookware, tools, devices, ew, vw, cw string
	err := s.Scan(&p.WorkspaceID, &p.Revision, &p.Preset, &p.PresetVersion, &a, &b, &c, &d, &cookware, &tools, &devices, &cw, &ew, &vw, &p.DraftJSON)
	if err != nil {
		return Profile{}, err
	}
	if err = json.Unmarshal([]byte(devices), &p.KitchenDevices); err != nil {
		return Profile{}, fmt.Errorf("decode profile kitchen devices: %w", err)
	}
	for raw, out := range map[string]*[]string{a: &p.ActiveRules, b: &p.ExcludedGroups, c: &p.Allergies, d: &p.Appliances, cookware: &p.Cookware, tools: &p.Tools} {
		if err = json.Unmarshal([]byte(raw), out); err != nil {
			return Profile{}, fmt.Errorf("decode profile list: %w", err)
		}
	}
	p.CostWeight, _ = strconv.ParseFloat(cw, 64)
	p.EffortWeight, _ = strconv.ParseFloat(ew, 64)
	p.VarietyWeight, _ = strconv.ParseFloat(vw, 64)
	return p, nil
}
