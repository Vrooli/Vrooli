package ingest

import (
	"context"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"money-ledger/internal/ledger"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Store struct {
	db      *database.RoutedDB
	journal *ledger.Store
	now     func() time.Time
}

func NewStore(db *database.RoutedDB, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, journal: ledger.NewStore(db, now), now: now}
}

func (s *Store) RegisterAdapter(ctx context.Context, a *ingestpb.Adapter) (*ingestpb.Adapter, error) {
	if a == nil || strings.TrimSpace(a.Id) == "" || strings.TrimSpace(a.Name) == "" || a.Kind == ingestpb.AdapterKind_ADAPTER_KIND_UNSPECIFIED {
		return nil, errors.New("adapter id, name, and kind are required")
	}
	if a.Id == ledger.TreasuryAdapterID {
		return nil, errors.New("the Treasury adapter is registered by authenticated receipt acceptance")
	}
	if a.Id == "" {
		a.Id = uuid.NewString()
	}
	if !a.Enabled {
		// An explicit false is useful for disabling an adapter, but newly
		// registered adapters should be usable by default.
		a.Enabled = true
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO adapters(id,name,kind,enabled,last_success_at,availability_reason,cursor,created_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(id) DO UPDATE SET name=excluded.name,kind=excluded.kind,enabled=excluded.enabled`, a.Id, a.Name, int32(a.Kind), a.Enabled, nil, a.AvailabilityReason, a.Cursor, s.now().UTC().Format(time.RFC3339Nano))
	return a, err
}

func (s *Store) ListAdapters(ctx context.Context) ([]*ingestpb.Adapter, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,kind,enabled,last_success_at,availability_reason,cursor FROM adapters ORDER BY name,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ingestpb.Adapter
	for rows.Next() {
		var id, name, reason, cursor string
		var last sql.NullString
		var kind int32
		var enabled bool
		if err := rows.Scan(&id, &name, &kind, &enabled, &last, &reason, &cursor); err != nil {
			return nil, err
		}
		a := &ingestpb.Adapter{Id: id, Name: name, Kind: ingestpb.AdapterKind(kind), Enabled: enabled, AvailabilityReason: reason, Cursor: cursor}
		if last.Valid && last.String != "" {
			if t, parseErr := time.Parse(time.RFC3339Nano, last.String); parseErr == nil {
				a.LastSuccessAt = timestamppb.New(t)
			}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) adapter(ctx context.Context, id string) (*ingestpb.Adapter, error) {
	var a ingestpb.Adapter
	var kind int32
	var enabled bool
	var last sql.NullString
	var reason string
	if err := s.db.QueryRowContext(ctx, `SELECT id,name,kind,enabled,last_success_at,availability_reason,cursor FROM adapters WHERE id=?`, id).Scan(&a.Id, &a.Name, &kind, &enabled, &last, &reason, &a.Cursor); err != nil {
		return nil, fmt.Errorf("adapter %q not found: %w", id, err)
	}
	a.Kind, a.Enabled, a.AvailabilityReason = ingestpb.AdapterKind(kind), enabled, reason
	if last.Valid && last.String != "" {
		if t, err := time.Parse(time.RFC3339Nano, last.String); err == nil {
			a.LastSuccessAt = timestamppb.New(t)
		}
	}
	if !a.Enabled {
		return &a, fmt.Errorf("adapter %q is disabled", id)
	}
	return &a, nil
}

// IngestEvent journals one event; actor is the verified caller the handler
// authorized, recorded on the posting.
func (s *Store) IngestEvent(ctx context.Context, event *sharedpb.MoneyEvent, actor string) (*sharedpb.Posting, bool, *ingestpb.Receipt, error) {
	if event == nil {
		return nil, false, nil, errors.New("money event is required")
	}
	a, err := s.adapter(ctx, event.AdapterId)
	if err != nil {
		return nil, false, nil, err
	}
	normalized := proto.Clone(event).(*sharedpb.MoneyEvent)
	if a.Kind == ingestpb.AdapterKind_ADAPTER_KIND_MANUAL {
		normalized.Basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	}
	p, duplicate, err := s.journal.Ingest(ctx, normalized, actor)
	r := s.newReceipt(event.AdapterId, nil, nil, 1, 0, 0, "failed", err)
	if err != nil {
		_, _ = s.insertReceipt(ctx, r)
		return nil, false, r, err
	}
	if duplicate {
		r.SkippedDuplicates = 1
	} else {
		r.Written = 1
	}
	r.Status = "succeeded"
	_, err = s.insertReceipt(ctx, r)
	if err == nil && !duplicate {
		err = s.markSuccess(ctx, event.AdapterId)
	}
	return p, duplicate, r, err
}

// ImportFile journals each CSV row through a file adapter as actor.
func (s *Store) ImportFile(ctx context.Context, actor, adapterID string, data []byte, from, to *timestamppb.Timestamp) (*ingestpb.Receipt, error) {
	a, err := s.adapter(ctx, adapterID)
	if err != nil {
		return nil, err
	}
	if a.Kind != ingestpb.AdapterKind_ADAPTER_KIND_FILE {
		return nil, errors.New("only file adapters accept CSV imports")
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	header, err := reader.Read()
	if err != nil {
		return nil, errors.New("file must start with a header row")
	}
	columns, err := csvColumns(header)
	if err != nil {
		return nil, err
	}
	reader.FieldsPerRecord = len(header)
	var events []*sharedpb.MoneyEvent
	for row := 1; ; row++ {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return s.failedReceipt(ctx, adapterID, from, to, row-1, fmt.Errorf("malformed CSV row %d", row))
		}
		field := func(name string) string {
			if index, ok := columns[name]; ok {
				return strings.TrimSpace(record[index])
			}
			return ""
		}
		amount := int64(0)
		if _, scanErr := fmt.Sscan(field("amount_minor"), &amount); scanErr != nil {
			return s.failedReceipt(ctx, adapterID, from, to, row-1, fmt.Errorf("row %d amount_minor: %w", row, scanErr))
		}
		occurred, parseErr := time.Parse(time.RFC3339, field("occurred_at"))
		if parseErr != nil {
			return s.failedReceipt(ctx, adapterID, from, to, row-1, fmt.Errorf("row %d occurred_at: %w", row, parseErr))
		}
		events = append(events, &sharedpb.MoneyEvent{ExternalId: field("external_id"), AccountId: field("account_id"), BookId: field("book_id"), AdapterId: adapterID, AmountMinor: amount, Currency: strings.ToUpper(field("currency")), OccurredAt: timestamppb.New(occurred), FetchedAt: timestamppb.New(s.now()), Basis: sharedpb.Basis_BASIS_AUTHORITATIVE, Description: field("description"), Category: field("category")})
	}
	r := s.newReceipt(adapterID, from, to, len(events), 0, 0, "succeeded", nil)
	for _, event := range events {
		_, duplicate, ingestErr := s.journal.Ingest(ctx, event, actor)
		if ingestErr != nil {
			r.Status, r.Error = "failed", ingestErr.Error()
			_, _ = s.insertReceipt(ctx, r)
			return r, ingestErr
		}
		if duplicate {
			r.SkippedDuplicates++
		} else {
			r.Written++
		}
	}
	if err := s.markSuccess(ctx, adapterID); err != nil {
		return r, err
	}
	_, err = s.insertReceipt(ctx, r)
	return r, err
}

// csvRequiredColumns are the columns every imported row must carry; a file
// may also carry description and category, and may order its columns freely.
var csvRequiredColumns = []string{"external_id", "account_id", "book_id", "amount_minor", "currency", "occurred_at"}

// csvColumns maps each header name to its column. Columns are read by name,
// never by position, so a file whose columns are reordered cannot journal an
// amount as a date or a currency as an account.
func csvColumns(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for index, name := range header {
		name = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("file header names column %q twice", name)
		}
		columns[name] = index
	}
	var missing []string
	for _, name := range csvRequiredColumns {
		if _, ok := columns[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("file header is missing required columns: %s", strings.Join(missing, ", "))
	}
	return columns, nil
}

func (s *Store) RunAdapter(ctx context.Context, adapterID string, from, to *timestamppb.Timestamp) (*ingestpb.Receipt, []*ingestpb.Availability, error) {
	a, err := s.adapter(ctx, adapterID)
	if err != nil {
		return nil, nil, err
	}
	if a.Kind == ingestpb.AdapterKind_ADAPTER_KIND_COMMERCE {
		return s.runCommerceAdapter(ctx, a, from, to)
	}
	reason := "adapter has no configured upstream; supply an explicit file or manual event"
	if a.AvailabilityReason != "" {
		reason = a.AvailabilityReason
	}
	r, receiptErr := s.failedReceipt(ctx, adapterID, from, to, 0, errors.New(reason))
	_, _ = s.db.ExecContext(ctx, `UPDATE adapters SET availability_reason=? WHERE id=?`, reason, adapterID)
	return r, []*ingestpb.Availability{{AdapterId: adapterID, Reason: reason, LastSuccessAt: a.LastSuccessAt}}, receiptErr
}

// runCommerceAdapter refuses LPBS's revenue summary until the producer exposes
// immutable transaction events. A rolling aggregate cannot satisfy the journal's
// occurrence identity or amount semantics.
func (s *Store) runCommerceAdapter(ctx context.Context, adapter *ingestpb.Adapter, from, to *timestamppb.Timestamp) (*ingestpb.Receipt, []*ingestpb.Availability, error) {
	return s.commerceUnavailable(ctx, adapter, from, to, errors.New("LPBS revenue summary is an aggregate projection, not a transaction event feed"))
}

func (s *Store) commerceUnavailable(ctx context.Context, adapter *ingestpb.Adapter, from, to *timestamppb.Timestamp, cause error) (*ingestpb.Receipt, []*ingestpb.Availability, error) {
	reason := "commerce adapter unavailable: " + cause.Error()
	r, err := s.failedReceipt(ctx, adapter.Id, from, to, 0, errors.New(reason))
	_, _ = s.db.ExecContext(ctx, `UPDATE adapters SET availability_reason=? WHERE id=?`, reason, adapter.Id)
	return r, []*ingestpb.Availability{{AdapterId: adapter.Id, Reason: reason, LastSuccessAt: adapter.LastSuccessAt}}, err
}

func (s *Store) newReceipt(adapterID string, from, to *timestamppb.Timestamp, read, written, skipped int, status string, err error) *ingestpb.Receipt {
	r := &ingestpb.Receipt{Id: uuid.NewString(), AdapterId: adapterID, From: from, To: to, Read: int32(read), Written: int32(written), SkippedDuplicates: int32(skipped), Status: status, CreatedAt: timestamppb.New(s.now())}
	if err != nil {
		r.Error = err.Error()
	}
	return r
}

func (s *Store) insertReceipt(ctx context.Context, r *ingestpb.Receipt) (bool, error) {
	var from, to string
	if r.From != nil {
		from = r.From.AsTime().UTC().Format(time.RFC3339Nano)
	}
	if r.To != nil {
		to = r.To.AsTime().UTC().Format(time.RFC3339Nano)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ingest_receipts(id,adapter_id,from_at,to_at,read_count,written_count,skipped_duplicates,status,error,created_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, r.Id, r.AdapterId, from, to, r.Read, r.Written, r.SkippedDuplicates, r.Status, r.Error, r.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano))
	return err == nil, err
}

func (s *Store) markSuccess(ctx context.Context, adapterID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE adapters SET last_success_at=?,availability_reason='' WHERE id=?`, s.now().UTC().Format(time.RFC3339Nano), adapterID)
	return err
}

func (s *Store) failedReceipt(ctx context.Context, adapterID string, from, to *timestamppb.Timestamp, read int, err error) (*ingestpb.Receipt, error) {
	r := s.newReceipt(adapterID, from, to, read, 0, 0, "failed", err)
	_, insertErr := s.insertReceipt(ctx, r)
	if insertErr != nil {
		return r, insertErr
	}
	return r, err
}
