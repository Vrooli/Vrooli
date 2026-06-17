package devices

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"device-sync-hub/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer (seam-discovery): both *sql.DB (unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production Repository impl. Unexported so callers
// depend on the Repository interface and tests substitute the fake.
type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository. db is the pool from
// main.go; clk supplies timestamps so tests advance time deterministically.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// deviceTimeFormat matches the wire format used across this scenario's
// round-trips: RFC3339Nano sorts lexicographically in time order for a fixed
// zone, so string range comparisons on the *_at columns are correct filters.
const deviceTimeFormat = time.RFC3339Nano

// capSep is the unit separator joining capability tags in the single TEXT
// column — the ASCII unit-separator avoids collisions with tag text.
const capSep = "\x1f"

const (
	insertDeviceSQL = `
INSERT INTO devices (id, owner_id, name, kind, platform, capabilities,
                     trust_state, session_id, token_hash, last_seen_at,
                     created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	// 'online' is a presence runtime concept (Phase 3 realtime), not a stored
	// column; selected as the literal 0 so the scan column list stays uniform
	// with the Device struct and presence is overlaid live by the caller.
	deviceColumns = `id, owner_id, name, kind, platform, capabilities,
                     trust_state, session_id, 0 AS online, last_seen_at,
                     created_at, updated_at`

	selectDeviceByIDSQL = `
SELECT ` + deviceColumns + `
FROM devices WHERE owner_id = ? AND id = ?
`

	selectDeviceByTokenSQL = `
SELECT ` + deviceColumns + `
FROM devices WHERE token_hash = ? AND token_hash != ''
`

	listDevicesSQL = `
SELECT ` + deviceColumns + `
FROM devices WHERE owner_id = ?
ORDER BY created_at DESC, id DESC
`

	setTrustSQL = `
UPDATE devices SET trust_state = ?, updated_at = ?
WHERE owner_id = ? AND id = ?
`

	renameDeviceSQL = `
UPDATE devices SET name = ?, updated_at = ?
WHERE owner_id = ? AND id = ?
`

	resolveOwnerSQL = `
SELECT owner_id FROM devices ORDER BY created_at DESC, id DESC LIMIT 1
`

	insertPairingCodeSQL = `
INSERT INTO pairing_codes (code_hash, owner_id, device_name, expires_at,
                           redeemed_at, created_at)
VALUES (?, ?, ?, ?, '', ?)
`

	// claimPairingCodeSQL is the conditional single-use claim: it stamps
	// redeemed_at only if the code is unredeemed and unexpired, so two
	// concurrent redeems can never both win (rows-affected disambiguates).
	claimPairingCodeSQL = `
UPDATE pairing_codes SET redeemed_at = ?
WHERE code_hash = ? AND redeemed_at = '' AND expires_at > ?
`

	selectClaimedCodeSQL = `
SELECT owner_id, device_name, expires_at, created_at
FROM pairing_codes WHERE code_hash = ?
`
)

func (r *sqliteRepository) now() time.Time { return r.clock.Now().UTC() }

func (r *sqliteRepository) CreateDevice(ctx context.Context, d Device, tokenHash string) (Device, error) {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	now := r.now()
	if d.CreatedAt.IsZero() {
		d.CreatedAt = now
	}
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = d.CreatedAt
	}
	if d.LastSeenAt.IsZero() {
		d.LastSeenAt = d.CreatedAt
	}
	_, err := r.db.ExecContext(ctx, insertDeviceSQL,
		d.ID, d.OwnerID, d.Name, d.Kind, d.Platform, joinCaps(d.Capabilities),
		string(d.TrustState), d.SessionID, tokenHash,
		d.LastSeenAt.Format(deviceTimeFormat),
		d.CreatedAt.Format(deviceTimeFormat),
		d.UpdatedAt.Format(deviceTimeFormat),
	)
	if err != nil {
		return Device{}, fmt.Errorf("insert device %q: %w", d.ID, err)
	}
	return d, nil
}

func (r *sqliteRepository) GetDevice(ctx context.Context, ownerID, id string) (Device, error) {
	row := r.db.QueryRowContext(ctx, selectDeviceByIDSQL, ownerID, id)
	d, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, ErrDeviceNotFound{ID: id}
	}
	if err != nil {
		return Device{}, fmt.Errorf("get device %q: %w", id, err)
	}
	return d, nil
}

func (r *sqliteRepository) DeviceByToken(ctx context.Context, tokenHash string) (Device, error) {
	if tokenHash == "" {
		return Device{}, ErrDeviceNotFound{ID: ""}
	}
	row := r.db.QueryRowContext(ctx, selectDeviceByTokenSQL, tokenHash)
	d, err := scanDevice(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, ErrDeviceNotFound{ID: ""}
	}
	if err != nil {
		return Device{}, fmt.Errorf("get device by token: %w", err)
	}
	return d, nil
}

func (r *sqliteRepository) ListDevices(ctx context.Context, ownerID string) ([]Device, error) {
	rows, err := r.db.QueryContext(ctx, listDevicesSQL, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) SetTrust(ctx context.Context, ownerID, id string, state TrustState) (Device, error) {
	res, err := r.db.ExecContext(ctx, setTrustSQL, string(state), r.now().Format(deviceTimeFormat), ownerID, id)
	if err != nil {
		return Device{}, fmt.Errorf("set trust on %q: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Device{}, ErrDeviceNotFound{ID: id}
	}
	return r.GetDevice(ctx, ownerID, id)
}

func (r *sqliteRepository) Rename(ctx context.Context, ownerID, id, name string) (Device, error) {
	res, err := r.db.ExecContext(ctx, renameDeviceSQL, name, r.now().Format(deviceTimeFormat), ownerID, id)
	if err != nil {
		return Device{}, fmt.Errorf("rename device %q: %w", id, err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Device{}, ErrDeviceNotFound{ID: id}
	}
	return r.GetDevice(ctx, ownerID, id)
}

func (r *sqliteRepository) ResolveOwner(ctx context.Context) (string, error) {
	var ownerID string
	err := r.db.QueryRowContext(ctx, resolveOwnerSQL).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrDeviceConflict{Reason: "no owner yet: the first device must pair with a code"}
	}
	if err != nil {
		return "", fmt.Errorf("resolve owner: %w", err)
	}
	return ownerID, nil
}

func (r *sqliteRepository) CreatePairingCode(ctx context.Context, c PairingCode) error {
	now := c.CreatedAt
	if now.IsZero() {
		now = r.now()
	}
	_, err := r.db.ExecContext(ctx, insertPairingCodeSQL,
		c.CodeHash, c.OwnerID, c.DeviceName,
		c.ExpiresAt.Format(deviceTimeFormat),
		now.Format(deviceTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("insert pairing code: %w", err)
	}
	return nil
}

func (r *sqliteRepository) ClaimPairingCode(ctx context.Context, codeHash string, now time.Time) (PairingCode, error) {
	res, err := r.db.ExecContext(ctx, claimPairingCodeSQL,
		now.Format(deviceTimeFormat), codeHash, now.Format(deviceTimeFormat),
	)
	if err != nil {
		return PairingCode{}, fmt.Errorf("claim pairing code: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return PairingCode{}, ErrInvalidPairingCode{}
	}

	var (
		pc         PairingCode
		expiresRaw string
		createdRaw string
	)
	err = r.db.QueryRowContext(ctx, selectClaimedCodeSQL, codeHash).
		Scan(&pc.OwnerID, &pc.DeviceName, &expiresRaw, &createdRaw)
	if err != nil {
		return PairingCode{}, fmt.Errorf("read claimed pairing code: %w", err)
	}
	pc.CodeHash = codeHash
	pc.RedeemedAt = now
	if pc.ExpiresAt, err = time.Parse(deviceTimeFormat, expiresRaw); err != nil {
		return PairingCode{}, fmt.Errorf("parse expires_at %q: %w", expiresRaw, err)
	}
	if pc.CreatedAt, err = time.Parse(deviceTimeFormat, createdRaw); err != nil {
		return PairingCode{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	return pc, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(s rowScanner) (Device, error) {
	var (
		d          Device
		capsRaw    string
		state      string
		online     int
		lastSeen   string
		createdRaw string
		updatedRaw string
	)
	if err := s.Scan(
		&d.ID, &d.OwnerID, &d.Name, &d.Kind, &d.Platform, &capsRaw,
		&state, &d.SessionID, &online, &lastSeen, &createdRaw, &updatedRaw,
	); err != nil {
		return Device{}, err
	}
	d.TrustState = TrustState(state)
	d.Online = online != 0
	d.Capabilities = splitCaps(capsRaw)

	var err error
	if d.LastSeenAt, err = time.Parse(deviceTimeFormat, lastSeen); err != nil {
		return Device{}, fmt.Errorf("parse last_seen_at %q: %w", lastSeen, err)
	}
	if d.CreatedAt, err = time.Parse(deviceTimeFormat, createdRaw); err != nil {
		return Device{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	if d.UpdatedAt, err = time.Parse(deviceTimeFormat, updatedRaw); err != nil {
		return Device{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	return d, nil
}

func joinCaps(caps []string) string { return strings.Join(caps, capSep) }

func splitCaps(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, capSep)
}
