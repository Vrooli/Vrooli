package control

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	if clock == nil {
		clock = schedule.System()
	}
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) SyncControllers(ctx context.Context, controllers []*controlv1.Controller, zones []ZoneMapping) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin controller sync: %w", err)
	}
	defer tx.Rollback()
	now := r.clock.Now().UTC().Format(time.RFC3339Nano)
	for _, c := range controllers {
		_, err = tx.ExecContext(ctx, `INSERT INTO lighting_controllers(controller_id,backend_id,name,vendor,description,enrolled,last_seen_at)
VALUES(?,?,?,?,?,COALESCE((SELECT enrolled FROM lighting_controllers WHERE controller_id=?),0),?)
ON CONFLICT(controller_id) DO UPDATE SET backend_id=excluded.backend_id,name=excluded.name,vendor=excluded.vendor,description=excluded.description,last_seen_at=excluded.last_seen_at`,
			c.ControllerId, c.BackendId, c.Name, c.Vendor, c.Description, c.ControllerId, now)
		if err != nil {
			return fmt.Errorf("upsert controller: %w", err)
		}
	}
	for _, c := range controllers {
		if _, err = tx.ExecContext(ctx, `DELETE FROM lighting_zones WHERE controller_id=?`, c.ControllerId); err != nil {
			return fmt.Errorf("replace zone mappings: %w", err)
		}
	}
	for _, z := range zones {
		_, err = tx.ExecContext(ctx, `INSERT INTO lighting_zones(controller_id,zone_id,fingerprint,name,led_count,supports_static_color) VALUES(?,?,?,?,?,?)
ON CONFLICT(controller_id,zone_id) DO UPDATE SET fingerprint=excluded.fingerprint,name=excluded.name,led_count=excluded.led_count,supports_static_color=excluded.supports_static_color`,
			z.ControllerID, z.ZoneID, z.Fingerprint, z.Name, z.LEDCount, z.Static)
		if err != nil {
			return fmt.Errorf("upsert zone mapping: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit controller sync: %w", err)
	}
	return nil
}

func (r *sqliteRepository) ListControllers(ctx context.Context) ([]*controlv1.Controller, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT controller_id,backend_id,name,vendor,description,enrolled,last_seen_at FROM lighting_controllers ORDER BY name,controller_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*controlv1.Controller, 0)
	for rows.Next() {
		var c controlv1.Controller
		var enrolled int
		var seen string
		if err := rows.Scan(&c.ControllerId, &c.BackendId, &c.Name, &c.Vendor, &c.Description, &enrolled, &seen); err != nil {
			return nil, err
		}
		c.Enrolled = enrolled != 0
		if t, e := time.Parse(time.RFC3339Nano, seen); e == nil {
			c.LastSeenAt = timestamppb.New(t)
		}
		out = append(out, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, c := range out {
		zones, e := r.ListZones(ctx, c.ControllerId)
		if e != nil {
			return nil, e
		}
		for _, z := range zones {
			c.Zones = append(c.Zones, &controlv1.ZoneCapability{ZoneId: z.ZoneID, Name: z.Name, LedCount: z.LEDCount, SupportsStaticColor: z.Static, SupportsBrightness: z.Static})
		}
	}
	return out, nil
}

func (r *sqliteRepository) SetEnrollment(ctx context.Context, id string, enrolled bool) (*controlv1.Controller, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE lighting_controllers SET enrolled=? WHERE controller_id=?`, enrolled, id)
	if err != nil {
		return nil, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	controllers, err := r.ListControllers(ctx)
	if err != nil {
		return nil, err
	}
	for _, c := range controllers {
		if c.ControllerId == id {
			return c, nil
		}
	}
	return nil, sql.ErrNoRows
}

func (r *sqliteRepository) ListZones(ctx context.Context, controllerID string) ([]ZoneMapping, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT controller_id,zone_id,fingerprint,name,led_count,supports_static_color FROM lighting_zones WHERE controller_id=? ORDER BY zone_id`, controllerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	zones := make([]ZoneMapping, 0)
	for rows.Next() {
		var z ZoneMapping
		var count uint32
		var supported int
		if err := rows.Scan(&z.ControllerID, &z.ZoneID, &z.Fingerprint, &z.Name, &count, &supported); err != nil {
			return nil, err
		}
		z.LEDCount = count
		z.Static = supported != 0
		zones = append(zones, z)
	}
	return zones, rows.Err()
}

func (r *sqliteRepository) SaveDesired(ctx context.Context, desired *controlv1.ZoneState, apply *controlv1.ZoneApplyResult) error {
	stateBytes, err := proto.Marshal(desired)
	if err != nil {
		return err
	}
	var applyBytes []byte
	if apply != nil {
		applyBytes, err = proto.Marshal(apply)
		if err != nil {
			return err
		}
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO lighting_desired_state(controller_id,zone_id,state,last_apply,updated_at) VALUES(?,?,?,?,?)
ON CONFLICT(controller_id,zone_id) DO UPDATE SET state=excluded.state,last_apply=excluded.last_apply,updated_at=excluded.updated_at`, desired.ControllerId, desired.ZoneId, stateBytes, nullableBytes(applyBytes), r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

func nullableBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func (r *sqliteRepository) ListDesired(ctx context.Context) ([]*controlv1.ZoneStatus, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT state,last_apply FROM lighting_desired_state ORDER BY controller_id,zone_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*controlv1.ZoneStatus, 0)
	for rows.Next() {
		var stateBytes, applyBytes []byte
		if err := rows.Scan(&stateBytes, &applyBytes); err != nil {
			return nil, err
		}
		state := &controlv1.ZoneState{}
		if err := proto.Unmarshal(stateBytes, state); err != nil {
			return nil, fmt.Errorf("decode desired state: %w", err)
		}
		status := &controlv1.ZoneStatus{Desired: state}
		if len(applyBytes) > 0 {
			apply := &controlv1.ZoneApplyResult{}
			if err := proto.Unmarshal(applyBytes, apply); err != nil {
				return nil, fmt.Errorf("decode apply result: %w", err)
			}
			status.LastApply = apply
		}
		out = append(out, status)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SetManualOverride(ctx context.Context, override *controlv1.ManualOverride) error {
	ids, err := proto.Marshal(&controlv1.SetManualOffRequest{AllEnrolled: override.AllEnrolled, ControllerIds: override.ControllerIds})
	if err != nil {
		return err
	}
	var expires any
	if override.ExpiresAt != nil {
		expires = override.ExpiresAt.AsTime().UTC().Format(time.RFC3339Nano)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO lighting_manual_override(singleton,all_enrolled,controller_ids,created_at,expires_at) VALUES(1,?,?,?,?)
ON CONFLICT(singleton) DO UPDATE SET all_enrolled=excluded.all_enrolled,controller_ids=excluded.controller_ids,created_at=excluded.created_at,expires_at=excluded.expires_at`, override.AllEnrolled, ids, override.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano), expires)
	return err
}

func (r *sqliteRepository) GetManualOverride(ctx context.Context) (*controlv1.ManualOverride, error) {
	var ids []byte
	var all int
	var created string
	var expires sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT all_enrolled,controller_ids,created_at,expires_at FROM lighting_manual_override WHERE singleton=1`).Scan(&all, &ids, &created, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return &controlv1.ManualOverride{}, nil
	}
	if err != nil {
		return nil, err
	}
	request := &controlv1.SetManualOffRequest{}
	if err := proto.Unmarshal(ids, request); err != nil {
		return nil, fmt.Errorf("decode manual override: %w", err)
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	out := &controlv1.ManualOverride{Active: true, AllEnrolled: all != 0, ControllerIds: request.ControllerIds, CreatedAt: timestamppb.New(t)}
	if expires.Valid {
		e, parseErr := time.Parse(time.RFC3339Nano, expires.String)
		if parseErr != nil {
			return nil, parseErr
		}
		out.ExpiresAt = timestamppb.New(e)
	}
	return out, nil
}

func (r *sqliteRepository) ClearManualOverride(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lighting_manual_override WHERE singleton=1`)
	return err
}
