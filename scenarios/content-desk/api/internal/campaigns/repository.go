package campaigns

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	Create(context.Context, Campaign, []string, []Slot) (Campaign, error)
	Activate(context.Context, string) error
	List(context.Context) ([]Campaign, error)
	ReserveSlot(context.Context, string, string, string) error
	ReleaseSlot(context.Context, string, string, string) error
	Slots(context.Context, string) ([]Slot, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Create(ctx context.Context, campaign Campaign, refs []string, slots []Slot) (Campaign, error) {
	if campaign.ID == "" {
		campaign.ID = uuid.NewString()
	}
	if campaign.Status == "" {
		campaign.Status = StatusProposed
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Campaign{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `INSERT INTO campaigns (id, name, status, created_at) VALUES (?, ?, ?, ?)`, campaign.ID, campaign.Name, campaign.Status, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	for _, ref := range refs {
		if ref == "" {
			continue
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO campaign_evidence_refs (campaign_id, ref) VALUES (?, ?)`, campaign.ID, ref); err != nil {
			return Campaign{}, err
		}
	}
	for _, slot := range slots {
		if _, err = tx.ExecContext(ctx, `INSERT INTO campaign_slots (campaign_id, channel, format, capacity, reserved) VALUES (?, ?, ?, ?, 0)`, campaign.ID, slot.Channel, slot.Format, slot.Capacity); err != nil {
			return Campaign{}, err
		}
	}
	if campaign.Status == StatusActive {
		if err = activationAllowed(ctx, tx, campaign.ID); err != nil {
			return Campaign{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Campaign{}, err
	}
	return campaign, nil
}

func (r *sqliteRepository) Activate(ctx context.Context, id string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err = activationAllowed(ctx, tx, id); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE campaigns SET status = ? WHERE id = ? AND status = ?`, StatusActive, id, StatusProposed)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("activate campaign %q: invalid status or missing", id)
	}
	return tx.Commit()
}

func activationAllowed(ctx context.Context, tx *sql.Tx, id string) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaign_evidence_refs WHERE campaign_id = ?`, id).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return ErrEvidenceRequired
	}
	return nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]Campaign, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, status FROM campaigns ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Campaign
	for rows.Next() {
		var campaign Campaign
		if err := rows.Scan(&campaign.ID, &campaign.Name, &campaign.Status); err != nil {
			return nil, err
		}
		out = append(out, campaign)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) ReserveSlot(ctx context.Context, campaignID, channel, format string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE campaign_slots SET reserved = reserved + 1 WHERE campaign_id = ? AND channel = ? AND format = ? AND reserved < capacity`, campaignID, channel, format)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrSlotExhausted
	}
	return nil
}

func (r *sqliteRepository) ReleaseSlot(ctx context.Context, campaignID, channel, format string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE campaign_slots SET reserved = reserved - 1 WHERE campaign_id = ? AND channel = ? AND format = ? AND reserved > 0`, campaignID, channel, format)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return errors.New("campaign slot is not reserved")
	}
	return nil
}

func (r *sqliteRepository) Slots(ctx context.Context, campaignID string) ([]Slot, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT channel, format, capacity, reserved FROM campaign_slots WHERE campaign_id = ? ORDER BY channel, format`, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Slot
	for rows.Next() {
		var slot Slot
		if err := rows.Scan(&slot.Channel, &slot.Format, &slot.Capacity, &slot.Reserved); err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	return out, rows.Err()
}
