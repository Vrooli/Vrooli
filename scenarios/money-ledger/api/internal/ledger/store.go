package ledger

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
)

//go:embed schema.sql
var schemaSQL embed.FS

// Store is the sole persistence boundary for ledger facts. It intentionally has
// no update/delete methods for postings: corrections are new reversing entries.
type Store struct {
	db  *database.RoutedDB
	now func() time.Time
}

func NewStore(db *database.RoutedDB, now func() time.Time) *Store {
	if now == nil {
		now = time.Now
	}
	return &Store{db: db, now: now}
}
func (s *Store) Schema() string { b, _ := schemaSQL.ReadFile("schema.sql"); return string(b) }

func (s *Store) recordAudit(ctx context.Context, entityType, entityID, actor, reason, priorValue string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ledger_audit(id,entity_type,entity_id,actor,reason,prior_value,created_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), entityType, entityID, actor, reason, priorValue, s.now().UTC().Format(time.RFC3339Nano))
	return err
}

func (s *Store) loadAudit(ctx context.Context, entityID string) ([]*sharedpb.AuditEntry, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,entity_type,entity_id,actor,reason,prior_value,created_at FROM ledger_audit WHERE entity_id=? ORDER BY created_at,id`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*sharedpb.AuditEntry
	for rows.Next() {
		var entry sharedpb.AuditEntry
		var timestamp string
		if err := rows.Scan(&entry.Id, &entry.EntityType, &entry.EntityId, &entry.Actor, &entry.Reason, &entry.PriorValue, &timestamp); err != nil {
			return nil, err
		}
		at, err := time.Parse(time.RFC3339Nano, timestamp)
		if err != nil {
			return nil, err
		}
		entry.CreatedAt = timestamppb.New(at)
		out = append(out, &entry)
	}
	return out, rows.Err()
}

func (s *Store) CreateBook(ctx context.Context, name, currency string) (*ledgerpb.Book, error) {
	name, currency = strings.TrimSpace(name), strings.ToUpper(strings.TrimSpace(currency))
	if name == "" || len(currency) != 3 {
		return nil, errors.New("book name and three-letter currency are required")
	}
	b := &ledgerpb.Book{Id: uuid.NewString(), Name: name, Currency: currency, CreatedAt: timestamppb.New(s.now())}
	_, err := s.db.ExecContext(ctx, `INSERT INTO books(id,name,currency,created_at) VALUES(?,?,?,?)`, b.Id, b.Name, b.Currency, b.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano))
	return b, err
}

func (s *Store) ListBooks(ctx context.Context) ([]*ledgerpb.Book, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,currency,created_at FROM books ORDER BY created_at,id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ledgerpb.Book
	for rows.Next() {
		var id, n, c, ts string
		if err := rows.Scan(&id, &n, &c, &ts); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339Nano, ts)
		out = append(out, &ledgerpb.Book{Id: id, Name: n, Currency: c, CreatedAt: timestamppb.New(t)})
	}
	return out, rows.Err()
}

func (s *Store) CreateAccount(ctx context.Context, bookID, name, kind string) (*ledgerpb.Account, error) {
	if strings.TrimSpace(bookID) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(kind) == "" {
		return nil, errors.New("book_id, name, and kind are required")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id=?`, bookID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	a := &ledgerpb.Account{Id: uuid.NewString(), BookId: bookID, Name: strings.TrimSpace(name), Kind: strings.TrimSpace(kind), CreatedAt: timestamppb.New(s.now())}
	_, err := s.db.ExecContext(ctx, `INSERT INTO accounts(id,book_id,name,kind,created_at) VALUES(?,?,?,?,?)`, a.Id, a.BookId, a.Name, a.Kind, a.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano))
	return a, err
}

func (s *Store) ListAccounts(ctx context.Context, bookID string) ([]*ledgerpb.Account, error) {
	q := `SELECT id,book_id,name,kind,created_at FROM accounts`
	args := []any{}
	if bookID != "" {
		q += ` WHERE book_id=?`
		args = append(args, bookID)
	}
	q += ` ORDER BY name,id`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ledgerpb.Account
	for rows.Next() {
		var id, b, n, k, ts string
		if err := rows.Scan(&id, &b, &n, &k, &ts); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339Nano, ts)
		out = append(out, &ledgerpb.Account{Id: id, BookId: b, Name: n, Kind: k, CreatedAt: timestamppb.New(t)})
	}
	return out, rows.Err()
}

func (s *Store) Ingest(ctx context.Context, e *sharedpb.MoneyEvent, actor string) (*sharedpb.Posting, bool, error) {
	if e == nil {
		return nil, false, errors.New("money event is required")
	}
	if e.Basis == sharedpb.Basis_BASIS_UNSPECIFIED || e.Basis == sharedpb.Basis_BASIS_PROJECTED {
		return nil, false, errors.New("basis must be authoritative, derived, or operator-asserted")
	}
	if e.FetchedAt == nil {
		e.FetchedAt = timestamppb.New(s.now())
	}
	if e.OccurredAt == nil {
		return nil, false, errors.New("occurred_at is required")
	}
	if strings.TrimSpace(e.ExternalId) == "" || strings.TrimSpace(e.AdapterId) == "" || strings.TrimSpace(e.AccountId) == "" || strings.TrimSpace(e.BookId) == "" {
		return nil, false, errors.New("external_id, adapter_id, account_id, and book_id are required")
	}
	if len(e.Currency) != 3 {
		return nil, false, errors.New("currency must be three letters")
	}
	var accountBook, accountCurrency string
	if err := s.db.QueryRowContext(ctx, `SELECT book_id,(SELECT currency FROM books WHERE id=accounts.book_id) FROM accounts WHERE id=?`, e.AccountId).Scan(&accountBook, &accountCurrency); err != nil {
		return nil, false, fmt.Errorf("account %q not found: %w", e.AccountId, err)
	}
	if accountBook != e.BookId {
		return nil, false, errors.New("account belongs to a different book")
	}
	if !strings.EqualFold(e.Currency, accountCurrency) {
		return nil, false, fmt.Errorf("event currency %q does not match book currency %q", e.Currency, accountCurrency)
	}
	var p sharedpb.Posting
	var eventID, adapterID, accountID, bookID, currency, occurred, fetched, rev, desc, cat, act, created string
	var amount int64
	var basis int32
	err := s.db.QueryRowContext(ctx, `SELECT id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,reversal_of,actor,created_at FROM postings WHERE adapter_id=? AND external_id=?`, e.AdapterId, e.ExternalId).Scan(&p.Id, &eventID, &adapterID, &accountID, &bookID, &amount, &currency, &occurred, &fetched, &basis, &desc, &cat, &rev, &act, &created)
	if err == nil {
		ot, _ := time.Parse(time.RFC3339Nano, occurred)
		ft, _ := time.Parse(time.RFC3339Nano, fetched)
		p.Event = &sharedpb.MoneyEvent{Id: p.Id, ExternalId: eventID, AdapterId: adapterID, AccountId: accountID, BookId: bookID, AmountMinor: amount, Currency: currency, OccurredAt: timestamppb.New(ot), FetchedAt: timestamppb.New(ft), Basis: sharedpb.Basis(basis), Description: desc, Category: cat}
		p.ReversalOf, p.Actor = rev, act
		return &p, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, false, err
	}
	p.Id = uuid.NewString()
	e.Id = p.Id
	p.Event = e
	p.ReversalOf = rev
	p.Actor = actor
	_, err = s.db.ExecContext(ctx, `INSERT INTO postings(id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,actor,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, p.Id, e.ExternalId, e.AdapterId, e.AccountId, e.BookId, e.AmountMinor, e.Currency, e.OccurredAt.AsTime().UTC().Format(time.RFC3339Nano), e.FetchedAt.AsTime().UTC().Format(time.RFC3339Nano), int32(e.Basis), e.Description, e.Category, actor, s.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, false, err
	}
	if err := s.recordAudit(ctx, "posting", p.Id, actor, "ingested money event", ""); err != nil {
		return nil, false, err
	}
	return &p, false, nil
}

func (s *Store) GetPosting(ctx context.Context, id string) (*sharedpb.Posting, error) {
	var p sharedpb.Posting
	var external, adapter, account, book, currency, occurred, fetched, rev, desc, cat, actor string
	var amount int64
	var basis int32
	if err := s.db.QueryRowContext(ctx, `SELECT id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,reversal_of,actor FROM postings WHERE id=?`, id).Scan(&p.Id, &external, &adapter, &account, &book, &amount, &currency, &occurred, &fetched, &basis, &desc, &cat, &rev, &actor); err != nil {
		return nil, fmt.Errorf("posting %q not found: %w", id, err)
	}
	ot, err := time.Parse(time.RFC3339Nano, occurred)
	if err != nil {
		return nil, err
	}
	ft, err := time.Parse(time.RFC3339Nano, fetched)
	if err != nil {
		return nil, err
	}
	p.Event = &sharedpb.MoneyEvent{Id: p.Id, ExternalId: external, AdapterId: adapter, AccountId: account, BookId: book, AmountMinor: amount, Currency: currency, OccurredAt: timestamppb.New(ot), FetchedAt: timestamppb.New(ft), Basis: sharedpb.Basis(basis), Description: desc, Category: cat}
	p.ReversalOf, p.Actor = rev, actor
	p.Audit, err = s.loadAudit(ctx, id)
	return &p, err
}

func (s *Store) ListPostings(ctx context.Context, accountID, bookID, from, to string, limit int) ([]*sharedpb.Posting, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,reversal_of,actor FROM postings`
	args := []any{}
	if accountID != "" {
		q += ` WHERE account_id=?`
		args = append(args, accountID)
	}
	if bookID != "" {
		if len(args) == 0 {
			q += ` WHERE `
		} else {
			q += ` AND `
		}
		q += `book_id=?`
		args = append(args, bookID)
	}
	if from != "" {
		if len(args) == 0 {
			q += ` WHERE `
		} else {
			q += ` AND `
		}
		q += `occurred_at>=?`
		args = append(args, from)
	}
	if to != "" {
		if len(args) == 0 {
			q += ` WHERE `
		} else {
			q += ` AND `
		}
		q += `occurred_at<=?`
		args = append(args, to)
	}
	q += ` ORDER BY occurred_at DESC,id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*sharedpb.Posting
	for rows.Next() {
		var p sharedpb.Posting
		var external, adapter, account, book, currency, occurred, fetched, desc, cat, rev, actor string
		var amount, basis int64
		if err := rows.Scan(&p.Id, &external, &adapter, &account, &book, &amount, &currency, &occurred, &fetched, &basis, &desc, &cat, &rev, &actor); err != nil {
			return nil, err
		}
		ot, _ := time.Parse(time.RFC3339Nano, occurred)
		ft, _ := time.Parse(time.RFC3339Nano, fetched)
		p.Event = &sharedpb.MoneyEvent{Id: p.Id, ExternalId: external, AdapterId: adapter, AccountId: account, BookId: book, AmountMinor: amount, Currency: currency, OccurredAt: timestamppb.New(ot), FetchedAt: timestamppb.New(ft), Basis: sharedpb.Basis(basis), Description: desc, Category: cat}
		p.ReversalOf = rev
		p.Actor = actor
		out = append(out, &p)
	}
	return out, rows.Err()
}

func (s *Store) Reverse(ctx context.Context, id, reason, actor string) (*sharedpb.Posting, error) {
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("reversal reason is required")
	}
	var e sharedpb.MoneyEvent
	var occurred, fetched string
	err := s.db.QueryRowContext(ctx, `SELECT external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,description,category FROM postings WHERE id=?`, id).Scan(&e.ExternalId, &e.AdapterId, &e.AccountId, &e.BookId, &e.AmountMinor, &e.Currency, &occurred, &fetched, &e.Description, &e.Category)
	if err != nil {
		return nil, fmt.Errorf("posting %q not found: %w", id, err)
	}
	ot, _ := time.Parse(time.RFC3339Nano, occurred)
	ft, _ := time.Parse(time.RFC3339Nano, fetched)
	e.OccurredAt = timestamppb.New(ot)
	e.FetchedAt = timestamppb.New(ft)
	e.AmountMinor = -e.AmountMinor
	e.ExternalId = "reversal:" + id + ":" + uuid.NewString()
	e.Basis = sharedpb.Basis_BASIS_OPERATOR_ASSERTED
	e.Description = strings.TrimSpace(reason)
	p := &sharedpb.Posting{Id: uuid.NewString(), Event: &e, ReversalOf: id, Actor: actor}
	_, err = s.db.ExecContext(ctx, `INSERT INTO postings(id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,reversal_of,actor,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, p.Id, e.ExternalId, e.AdapterId, e.AccountId, e.BookId, e.AmountMinor, e.Currency, e.OccurredAt.AsTime().UTC().Format(time.RFC3339Nano), e.FetchedAt.AsTime().UTC().Format(time.RFC3339Nano), int32(e.Basis), e.Description, e.Category, id, actor, s.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	if err := s.recordAudit(ctx, "posting", p.Id, actor, "reversing entry: "+reason, id); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) Transfer(ctx context.Context, fromAccount, toAccount string, amount int64, currency, externalID, description string, occurred *timestamppb.Timestamp, actor string) ([]*sharedpb.Posting, error) {
	if amount <= 0 || strings.TrimSpace(externalID) == "" || occurred == nil {
		return nil, errors.New("transfer requires a positive amount, external_id, and occurred_at")
	}
	if strings.TrimSpace(currency) == "" {
		return nil, errors.New("transfer currency is required")
	}
	var fromBook, fromCurrency, toBook, toCurrency string
	if err := s.db.QueryRowContext(ctx, `SELECT book_id,(SELECT currency FROM books WHERE id=accounts.book_id) FROM accounts WHERE id=?`, fromAccount).Scan(&fromBook, &fromCurrency); err != nil {
		return nil, fmt.Errorf("source account not found: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT book_id,(SELECT currency FROM books WHERE id=accounts.book_id) FROM accounts WHERE id=?`, toAccount).Scan(&toBook, &toCurrency); err != nil {
		return nil, fmt.Errorf("destination account not found: %w", err)
	}
	if !strings.EqualFold(currency, fromCurrency) || !strings.EqualFold(currency, toCurrency) {
		return nil, errors.New("transfer currency must match both account books")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	created := s.now().UTC().Format(time.RFC3339Nano)
	entries := []*sharedpb.MoneyEvent{
		{Id: uuid.NewString(), ExternalId: externalID + ":from", AdapterId: "transfer", AccountId: fromAccount, BookId: fromBook, AmountMinor: -amount, Currency: strings.ToUpper(currency), OccurredAt: occurred, FetchedAt: timestamppb.New(s.now()), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, Description: description},
		{Id: uuid.NewString(), ExternalId: externalID + ":to", AdapterId: "transfer", AccountId: toAccount, BookId: toBook, AmountMinor: amount, Currency: strings.ToUpper(currency), OccurredAt: occurred, FetchedAt: timestamppb.New(s.now()), Basis: sharedpb.Basis_BASIS_OPERATOR_ASSERTED, Description: description},
	}
	posts := make([]*sharedpb.Posting, 0, len(entries))
	for _, event := range entries {
		if _, err := tx.ExecContext(ctx, `INSERT INTO postings(id,external_id,adapter_id,account_id,book_id,amount_minor,currency,occurred_at,fetched_at,basis,description,category,actor,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, event.Id, event.ExternalId, event.AdapterId, event.AccountId, event.BookId, event.AmountMinor, event.Currency, event.OccurredAt.AsTime().UTC().Format(time.RFC3339Nano), event.FetchedAt.AsTime().UTC().Format(time.RFC3339Nano), int32(event.Basis), event.Description, event.Category, actor, created); err != nil {
			return nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO ledger_audit(id,entity_type,entity_id,actor,reason,prior_value,created_at) VALUES(?,?,?,?,?,?,?)`, uuid.NewString(), "posting", event.Id, actor, "paired inter-book transfer", "", created); err != nil {
			return nil, err
		}
		posts = append(posts, &sharedpb.Posting{Id: event.Id, Event: event, Actor: actor})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return posts, nil
}

func (s *Store) Position(ctx context.Context, bookID string) (*ledgerpb.PositionResponse, error) {
	var cash, revenue, expense int64
	var currency string
	if err := s.db.QueryRowContext(ctx, `SELECT currency FROM books WHERE id=?`, bookID).Scan(&currency); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT a.kind,COALESCE(SUM(p.amount_minor),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? GROUP BY a.kind`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind string
		var value int64
		if err := rows.Scan(&kind, &value); err != nil {
			return nil, err
		}
		switch strings.ToLower(kind) {
		case "cash", "asset":
			cash += value
		case "revenue", "income":
			revenue += value
		case "expense":
			expense += value
		}
	}
	burn := expense - revenue
	runway := 0.0
	runwayAvailable := false
	runwayReason := "runway is undefined until the journal has a positive burn rate"
	if burn > 0 {
		runway = float64(cash) / float64(burn)
		runwayAvailable = true
		runwayReason = ""
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	response := &ledgerpb.PositionResponse{CashMinor: cash, RevenueMinor: revenue, ExpenseMinor: expense, BurnMinor: burn, RunwayMonths: runway, Currency: currency, RunwayAvailable: runwayAvailable, RunwayReason: runwayReason}
	inputRows, err := s.db.QueryContext(ctx, `SELECT adapter_id,MAX(fetched_at),MAX(basis) FROM postings WHERE book_id=? GROUP BY adapter_id`, bookID)
	if err != nil {
		return nil, err
	}
	for inputRows.Next() {
		var source, fetched string
		var basis int32
		if err := inputRows.Scan(&source, &fetched, &basis); err != nil {
			inputRows.Close()
			return nil, err
		}
		at, parseErr := time.Parse(time.RFC3339Nano, fetched)
		if parseErr != nil {
			inputRows.Close()
			return nil, parseErr
		}
		age := int64(s.now().Sub(at).Seconds())
		if age < 0 {
			age = 0
		}
		response.Inputs = append(response.Inputs, &ledgerpb.PositionInput{Source: source, Basis: sharedpb.Basis(basis), AgeSeconds: age, Available: true})
	}
	if err := inputRows.Err(); err != nil {
		inputRows.Close()
		return nil, err
	}
	inputRows.Close()
	availability, err := s.availability(ctx)
	if err != nil {
		return nil, err
	}
	response.Availability = availability
	response.Partial = len(availability) > 0
	for _, missing := range availability {
		response.Inputs = append(response.Inputs, &ledgerpb.PositionInput{Source: missing.AdapterId, Available: false, Reason: missing.Reason})
	}
	return response, nil
}

func (s *Store) Statement(ctx context.Context, bookID, from, to string) (*ledgerpb.StatementResponse, error) {
	book, err := s.Position(ctx, bookID)
	if err != nil {
		return nil, err
	}
	period := "p.book_id=?"
	periodArgs := []any{bookID}
	if from != "" {
		period += " AND p.occurred_at>=?"
		periodArgs = append(periodArgs, from)
	}
	if to != "" {
		period += " AND p.occurred_at<=?"
		periodArgs = append(periodArgs, to)
	}
	var inflow, outflow, revenue, expense int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN p.amount_minor > 0 THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN p.amount_minor < 0 THEN -p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN lower(a.kind) IN ('revenue','income') THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN lower(a.kind)='expense' THEN -p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE `+period, periodArgs...).Scan(&inflow, &outflow, &revenue, &expense); err != nil {
		return nil, err
	}
	ending := to
	if ending == "" {
		ending = "9999-12-31T23:59:59Z"
	}
	var openingCash, closingCash, nonCashAssets, liabilities int64
	if from != "" {
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN lower(a.kind) IN ('cash','asset') THEN p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? AND p.occurred_at<?`, bookID, from).Scan(&openingCash); err != nil {
			return nil, err
		}
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN lower(a.kind) IN ('cash') THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN lower(a.kind) IN ('asset') THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN lower(a.kind) IN ('liability','liabilities') THEN p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? AND p.occurred_at<=?`, bookID, ending).Scan(&closingCash, &nonCashAssets, &liabilities); err != nil {
		return nil, err
	}
	assets := closingCash + nonCashAssets
	return &ledgerpb.StatementResponse{BookId: bookID, Currency: book.Currency, OpeningCashMinor: openingCash, InflowMinor: inflow, OutflowMinor: outflow, ClosingCashMinor: closingCash, RevenueMinor: revenue, ExpenseMinor: expense, Partial: book.Partial, Availability: book.Availability, AssetsMinor: assets, LiabilitiesMinor: liabilities, From: from, To: to}, nil
}

func (s *Store) availability(ctx context.Context) ([]*ledgerpb.Availability, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,availability_reason,last_success_at FROM adapters WHERE enabled=1 AND availability_reason<>'' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ledgerpb.Availability
	for rows.Next() {
		var id, reason string
		var last sql.NullString
		if err := rows.Scan(&id, &reason, &last); err != nil {
			return nil, err
		}
		a := &ledgerpb.Availability{AdapterId: id, Reason: reason}
		if last.Valid && last.String != "" {
			if parsed, parseErr := time.Parse(time.RFC3339Nano, last.String); parseErr == nil {
				a.LastSuccessAt = timestamppb.New(parsed)
			}
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) DeclareGoal(ctx context.Context, bookID string, g *ledgerpb.Goal) (*ledgerpb.Goal, error) {
	if g == nil || strings.TrimSpace(g.Name) == "" || strings.TrimSpace(g.Metric) == "" {
		return nil, errors.New("goal name and metric are required")
	}
	if g.SustainPeriods <= 0 {
		g.SustainPeriods = 1
	}
	if strings.TrimSpace(g.Comparator) == "" {
		return nil, errors.New("goal comparator is required")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id=?`, bookID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	g.Id = uuid.NewString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO goals(id,book_id,name,metric,comparator,threshold_minor,sustain_periods,buffer_multiple) VALUES(?,?,?,?,?,?,?,?)`, g.Id, bookID, g.Name, g.Metric, g.Comparator, g.ThresholdMinor, g.SustainPeriods, g.BufferMultiple)
	return g, err
}

func (s *Store) ListGoals(ctx context.Context, bookID string) ([]*ledgerpb.GoalVerdict, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,metric,comparator,threshold_minor,sustain_periods,buffer_multiple FROM goals WHERE book_id=? ORDER BY name`, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	goals := make([]*ledgerpb.Goal, 0)
	for rows.Next() {
		var g ledgerpb.Goal
		if err := rows.Scan(&g.Id, &g.Name, &g.Metric, &g.Comparator, &g.ThresholdMinor, &g.SustainPeriods, &g.BufferMultiple); err != nil {
			_ = rows.Close()
			return nil, err
		}
		goals = append(goals, &g)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	out := make([]*ledgerpb.GoalVerdict, 0, len(goals))
	for _, g := range goals {
		position, positionErr := s.Position(ctx, bookID)
		if positionErr != nil {
			return nil, positionErr
		}
		value := position.CashMinor
		switch strings.ToLower(g.Metric) {
		case "revenue", "income":
			value = position.RevenueMinor
		case "expense":
			value = position.ExpenseMinor
		case "burn":
			value = position.BurnMinor
		case "runway":
			value = int64(position.RunwayMonths)
		}
		sustained, sustainErr := s.sustainedGoalPeriods(ctx, bookID, g)
		if sustainErr != nil {
			return nil, sustainErr
		}
		out = append(out, &ledgerpb.GoalVerdict{Goal: g, Met: sustained >= g.SustainPeriods, SustainedPeriods: sustained, RequiredPeriods: g.SustainPeriods, Explanation: fmt.Sprintf("read-time %s=%d; required %s %d for %d sustained period(s)", g.Metric, value, g.Comparator, g.ThresholdMinor, g.SustainPeriods)})
	}
	return out, nil
}

func (s *Store) sustainedGoalPeriods(ctx context.Context, bookID string, goal *ledgerpb.Goal) (int32, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT substr(p.occurred_at,1,7), lower(a.kind), COALESCE(SUM(p.amount_minor),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? GROUP BY substr(p.occurred_at,1,7), lower(a.kind)`, bookID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	type monthValues struct{ cash, revenue, expense int64 }
	months := map[string]*monthValues{}
	for rows.Next() {
		var month, kind string
		var amount int64
		if err := rows.Scan(&month, &kind, &amount); err != nil {
			return 0, err
		}
		v := months[month]
		if v == nil {
			v = &monthValues{}
			months[month] = v
		}
		switch kind {
		case "cash", "asset":
			v.cash += amount
		case "revenue", "income":
			v.revenue += amount
		case "expense":
			v.expense += amount
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(months))
	for key := range months {
		keys = append(keys, key)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	var sustained int32
	for _, key := range keys {
		v := months[key]
		value := v.cash
		switch strings.ToLower(goal.Metric) {
		case "revenue", "income":
			value = v.revenue
		case "expense":
			value = v.expense
		case "burn":
			value = v.expense - v.revenue
		}
		if !compareGoal(value, goal.Comparator, goal.ThresholdMinor) {
			break
		}
		sustained++
		if sustained >= goal.SustainPeriods {
			break
		}
	}
	return sustained, nil
}

func compareGoal(value int64, comparator string, threshold int64) bool {
	switch strings.TrimSpace(comparator) {
	case ">":
		return value > threshold
	case ">=", "at_least":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=", "at_most":
		return value <= threshold
	default:
		return false
	}
}
