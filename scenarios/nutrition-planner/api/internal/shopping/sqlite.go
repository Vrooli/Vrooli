package shopping

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"nutrition-planner/internal/decimalx"

	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func (r *sqliteRepository) HaveThis(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT line_key,asserted FROM shopping_have_this WHERE workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		var asserted int
		if err := rows.Scan(&key, &asserted); err != nil {
			return nil, err
		}
		out[key] = asserted != 0
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SetHaveThis(ctx context.Context, workspaceID, key string, asserted bool) error {
	if workspaceID == "" || key == "" {
		return errors.New("workspace and line key are required")
	}
	value := 0
	if asserted {
		value = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO shopping_have_this(workspace_id,line_key,asserted,updated_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,line_key) DO UPDATE SET asserted=excluded.asserted,updated_at=excluded.updated_at`, workspaceID, key, value, r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}

// ConfirmPurchases stores the reviewed trip and appends all actual purchase events
// in one transaction. Repeating an identical review is safe; changing its payload conflicts.
func (r *sqliteRepository) ConfirmPurchases(ctx context.Context, workspaceID, reviewID string, lines []PurchaseLine) error {
	if workspaceID == "" || reviewID == "" {
		return errors.New("workspace and review id are required")
	}
	if len(lines) == 0 {
		return errors.New("purchase review must include reviewed rows")
	}
	lines = append([]PurchaseLine(nil), lines...)
	sort.Slice(lines, func(i, j int) bool { return lines[i].Key < lines[j].Key })
	seen := map[string]bool{}
	for _, line := range lines {
		if line.Key == "" || line.ItemID == "" || line.Unit == "" || seen[line.Key] {
			return errors.New("purchase review rows require unique keys, item ids, and units")
		}
		seen[line.Key] = true
		if line.Omitted {
			continue
		}
		if line.Amount.IsUnknown() || line.Amount.IsZero() {
			return fmt.Errorf("purchase row %q requires a positive actual quantity", line.Key)
		}
		if comparison, _ := decimalx.Compare(line.Amount, decimalx.KnownInt(0)); comparison < 0 {
			return fmt.Errorf("purchase row %q requires a positive actual quantity", line.Key)
		}
	}
	type canonicalLine struct {
		Key, ItemID, Amount, Unit, Price string
		Omitted                          bool
	}
	canonical := make([]canonicalLine, 0, len(lines))
	for _, line := range lines {
		amount := ""
		if !line.Omitted {
			amount = line.Amount.String()
		}
		canonical = append(canonical, canonicalLine{line.Key, line.ItemID, amount, line.Unit, line.Price, line.Omitted})
	}
	payload, err := json.Marshal(canonical)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(digest[:])
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var existing string
	err = tx.QueryRowContext(ctx, `SELECT payload_hash FROM shopping_purchase_reviews WHERE workspace_id=? AND review_id=?`, workspaceID, reviewID).Scan(&existing)
	if err == nil {
		if existing != payloadHash {
			return fmt.Errorf("purchase review id %q reused with different payload", reviewID)
		}
		return tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	created := r.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err = tx.ExecContext(ctx, `INSERT INTO shopping_purchase_reviews(workspace_id,review_id,payload_hash,created_at) VALUES(?,?,?,?)`, workspaceID, reviewID, payloadHash, created); err != nil {
		return err
	}
	for _, line := range lines {
		amount := ""
		if !line.Omitted {
			amount = line.Amount.String()
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO shopping_purchase_review_lines(workspace_id,review_id,line_key,item_id,amount,unit,price,omitted) VALUES(?,?,?,?,?,?,?,?)`, workspaceID, reviewID, line.Key, line.ItemID, amount, line.Unit, line.Price, boolInt(line.Omitted)); err != nil {
			return err
		}
		if line.Omitted {
			continue
		}
		eventHash := sha256.Sum256([]byte(reviewID + "\x1f" + line.Key))
		eventID := "shopping-purchase:" + hex.EncodeToString(eventHash[:])
		if _, err = tx.ExecContext(ctx, `INSERT INTO inventory_events(workspace_id,event_id,kind,item_id,batch_id,amount,unit,recipe_id,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, workspaceID, eventID, "purchase", line.ItemID, "", line.Amount.String(), line.Unit, "", created); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) LatestPurchaseReview(ctx context.Context, workspaceID string) (map[string]PurchaseLine, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT line_key,item_id,amount,unit,price,omitted FROM shopping_purchase_review_lines WHERE workspace_id=? AND review_id=(SELECT review_id FROM shopping_purchase_reviews WHERE workspace_id=? ORDER BY created_at DESC,review_id DESC LIMIT 1)`, workspaceID, workspaceID)
	if err != nil { return nil, err }
	defer rows.Close()
	out := map[string]PurchaseLine{}
	for rows.Next() {
		var line PurchaseLine; var amount string; var omitted int
		if err := rows.Scan(&line.Key,&line.ItemID,&amount,&line.Unit,&line.Price,&omitted); err != nil { return nil, err }
		line.Omitted = omitted != 0
		if !line.Omitted { var err error; line.Amount,err = decimalx.Parse(amount); if err != nil { return nil, err } }
		out[line.Key] = line
	}
	return out, rows.Err()
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

func (r *sqliteRepository) Checked(ctx context.Context, workspaceID string) (map[string]bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT line_key,checked FROM shopping_checks WHERE workspace_id=?`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		var checked int
		if err := rows.Scan(&key, &checked); err != nil {
			return nil, err
		}
		out[key] = checked != 0
	}
	return out, rows.Err()
}

func (r *sqliteRepository) SetChecked(ctx context.Context, workspaceID, key string, checked bool) error {
	value := 0
	if checked {
		value = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO shopping_checks(workspace_id,line_key,checked,updated_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,line_key) DO UPDATE SET checked=excluded.checked,updated_at=excluded.updated_at`, workspaceID, key, value, r.clock.Now().UTC().Format(time.RFC3339Nano))
	return err
}
