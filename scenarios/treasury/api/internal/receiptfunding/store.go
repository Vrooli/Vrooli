package receiptfunding

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	"google.golang.org/protobuf/proto"
)

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

var (
	ErrNotFound = errors.New("receipt funding delivery not found")
	ErrConflict = errors.New("receipt funding submission already has different content")
	ErrInvalid  = errors.New("invalid receipt funding submission")
)

type Delivery struct {
	ID, OperatorIdentity, WalletID, BudgetID, SourceCheckID string
	CreditLotID, CreditLotOwner, CreditLotReceiptID         string
	Status, LastError                                       string
	Attempts                                                int
	CreatedAt, AcceptedAt                                   time.Time
	Request                                                 *ingestpb.AcceptCryptoReceiptRequest
	Receipt                                                 *ingestpb.AcceptedCryptoReceipt
}

type DB interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct{ db DB }

func NewStore(db DB) *Store { return &Store{db: db} }

func (s *Store) Queue(ctx context.Context, value Delivery) (Delivery, error) {
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(value.Request)
	if err != nil {
		return Delivery{}, err
	}
	hash := sha256.New()
	_, _ = hash.Write([]byte(value.BudgetID + "\x00" + value.OperatorIdentity + "\x00" + value.WalletID + "\x00"))
	_, _ = hash.Write(encoded)
	fingerprint := hex.EncodeToString(hash.Sum(nil))
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return Delivery{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO receipt_funding_deliveries(id,operator_identity,wallet_id,budget_id,source_check_id,request_proto,request_sha256,status,created_at) VALUES(?,?,?,?,?,?,?,'queued',?) ON CONFLICT(id) DO NOTHING`, value.ID, value.OperatorIdentity, value.WalletID, value.BudgetID, value.SourceCheckID, encoded, fingerprint, value.CreatedAt.UTC().Format(time.RFC3339Nano)); err != nil {
		return Delivery{}, err
	}
	var stored string
	if err := tx.QueryRowContext(ctx, `SELECT request_sha256 FROM receipt_funding_deliveries WHERE id=?`, value.ID).Scan(&stored); err != nil {
		return Delivery{}, err
	}
	if stored != fingerprint {
		return Delivery{}, ErrConflict
	}
	if err := tx.Commit(); err != nil {
		return Delivery{}, err
	}
	return s.Get(ctx, value.ID, value.OperatorIdentity)
}

func (s *Store) Get(ctx context.Context, id, operator string) (Delivery, error) {
	return scan(s.db.QueryRowContext(ctx, selectDelivery+` WHERE id=? AND operator_identity=?`, id, operator))
}

func (s *Store) Pending(ctx context.Context, limit int) ([]Delivery, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, selectDelivery+` WHERE d.status='queued' OR (d.status='accepted' AND l.delivery_id IS NULL) ORDER BY d.created_at,d.id LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []Delivery
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, err
		}
		if value.Status == "queued" || value.Receipt.GetAttribution().GetPayerIdentity() != "" {
			values = append(values, value)
		}
	}
	return values, rows.Err()
}

func (s *Store) Failure(ctx context.Context, id string, err error) error {
	_, updateErr := s.db.ExecContext(ctx, `UPDATE receipt_funding_deliveries SET attempts=attempts+1,last_error=? WHERE id=?`, err.Error(), id)
	return updateErr
}

func (s *Store) Accept(ctx context.Context, id string, receipt *ingestpb.AcceptedCryptoReceipt) error {
	encoded, err := proto.Marshal(receipt)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE receipt_funding_deliveries SET status='accepted',attempts=attempts+1,last_error='',accepted_proto=?,accepted_at=? WHERE id=? AND status='queued'`, encoded, receipt.GetAcceptedAt().AsTime().UTC().Format(time.RFC3339Nano), id)
	return err
}

// AssociateCreditLot records a post-acceptance LPBS link without mutating the
// immutable Money Ledger receipt. Replays must resolve to the same receipt and
// same owner-created lot.
func (s *Store) AssociateCreditLot(ctx context.Context, id, receiptID, owner, lotID string, linkedAt time.Time) error {
	if id == "" || receiptID == "" || owner == "" || lotID == "" {
		return errors.New("accepted receipt and LPBS credit-lot association are required")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var state string
	var encoded []byte
	if err := tx.QueryRowContext(ctx, `SELECT status,accepted_proto FROM receipt_funding_deliveries WHERE id=?`, id).Scan(&state, &encoded); err != nil {
		return err
	}
	var receipt ingestpb.AcceptedCryptoReceipt
	if state != "accepted" || proto.Unmarshal(encoded, &receipt) != nil || receipt.GetId() != receiptID {
		return errors.New("credit lot must link to this delivery's accepted Money Ledger receipt")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO receipt_funding_credit_lot_links(delivery_id,receipt_id,credit_lot_id,credit_lot_owner,linked_at) VALUES(?,?,?,?,?) ON CONFLICT(delivery_id) DO NOTHING`, id, receiptID, lotID, owner, linkedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	var storedReceiptID, storedLotID, storedOwner string
	if err := tx.QueryRowContext(ctx, `SELECT receipt_id,credit_lot_id,credit_lot_owner FROM receipt_funding_credit_lot_links WHERE delivery_id=?`, id).Scan(&storedReceiptID, &storedLotID, &storedOwner); err != nil {
		return err
	}
	if storedReceiptID != receiptID || storedLotID != lotID || storedOwner != owner {
		return errors.New("accepted receipt was already linked to a different credit lot")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE receipt_funding_deliveries SET last_error='' WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

const selectDelivery = `SELECT d.id,d.operator_identity,d.wallet_id,d.budget_id,d.source_check_id,d.request_proto,d.status,d.attempts,d.last_error,d.accepted_proto,d.created_at,d.accepted_at,COALESCE(l.credit_lot_id,''),COALESCE(l.credit_lot_owner,''),COALESCE(l.receipt_id,'') FROM receipt_funding_deliveries d LEFT JOIN receipt_funding_credit_lot_links l ON l.delivery_id=d.id`

type scanner interface{ Scan(...any) error }

func scan(row scanner) (Delivery, error) {
	var value Delivery
	var request, receipt []byte
	var created, accepted string
	if err := row.Scan(&value.ID, &value.OperatorIdentity, &value.WalletID, &value.BudgetID, &value.SourceCheckID, &request, &value.Status, &value.Attempts, &value.LastError, &receipt, &created, &accepted, &value.CreditLotID, &value.CreditLotOwner, &value.CreditLotReceiptID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Delivery{}, ErrNotFound
		}
		return Delivery{}, err
	}
	value.Request = new(ingestpb.AcceptCryptoReceiptRequest)
	if err := proto.Unmarshal(request, value.Request); err != nil {
		return Delivery{}, fmt.Errorf("decode retained funding request: %w", err)
	}
	if len(receipt) > 0 {
		value.Receipt = new(ingestpb.AcceptedCryptoReceipt)
		if err := proto.Unmarshal(receipt, value.Receipt); err != nil {
			return Delivery{}, fmt.Errorf("decode accepted funding receipt: %w", err)
		}
	}
	var err error
	if value.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
		return Delivery{}, err
	}
	if accepted != "" {
		value.AcceptedAt, err = time.Parse(time.RFC3339Nano, accepted)
	}
	return value, err
}
