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

var declaredAccountKinds = map[ledgerpb.AccountKind]string{
	ledgerpb.AccountKind_ASSET:     "ASSET",
	ledgerpb.AccountKind_LIABILITY: "LIABILITY",
	ledgerpb.AccountKind_REVENUE:   "REVENUE",
	ledgerpb.AccountKind_EXPENSE:   "EXPENSE",
	ledgerpb.AccountKind_EQUITY:    "EQUITY",
}

const acceptedAccountKinds = "ASSET, LIABILITY, REVENUE, EXPENSE, EQUITY"

func (s *Store) EnsureMigrations(ctx context.Context) error {
	for _, table := range []string{"books", "goals"} {
		rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
		if err != nil {
			return err
		}
		defer rows.Close()
		found := false
		for rows.Next() {
			var cid int
			var name, columnType string
			var notNull, primaryKey int
			var defaultValue sql.NullString
			if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
				_ = rows.Close()
				return err
			}
			if name == "archived" {
				found = true
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return err
		}
		if err := rows.Close(); err != nil {
			return err
		}
		if !found {
			if _, err := s.db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil {
				return err
			}
		}
	}

	rows, err := s.db.QueryContext(ctx, `SELECT id,kind FROM accounts ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type accountKindRow struct{ id, kind string }
	var accounts []accountKindRow
	for rows.Next() {
		var row accountKindRow
		if err := rows.Scan(&row.id, &row.kind); err != nil {
			_ = rows.Close()
			return err
		}
		accounts = append(accounts, row)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, row := range accounts {
		normalized := strings.ToUpper(strings.TrimSpace(row.kind))
		if normalized != row.kind {
			if _, err := s.db.ExecContext(ctx, `UPDATE accounts SET kind=? WHERE id=?`, normalized, row.id); err != nil {
				return err
			}
			if err := s.recordAudit(ctx, "account", row.id, "system", "normalized account kind to uppercase", row.kind); err != nil {
				return err
			}
		}
		if !isDeclaredAccountKindName(normalized) {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO operator_input_findings(id,path,reason,created_at) SELECT NULL,?,?,? WHERE NOT EXISTS (SELECT 1 FROM operator_input_findings WHERE path=? AND reason=?)`, "account.kind:"+row.id, "unknown account kind "+normalized+"; operator review required", s.now().UTC().Format(time.RFC3339Nano), "account.kind:"+row.id, "unknown account kind "+normalized+"; operator review required"); err != nil {
				return err
			}
		}
	}
	return nil
}

func isDeclaredAccountKindName(kind string) bool {
	switch kind {
	case "ASSET", "LIABILITY", "REVENUE", "EXPENSE", "EQUITY":
		return true
	default:
		return false
	}
}

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
	_, err := s.db.ExecContext(ctx, `INSERT INTO books(id,name,currency,created_at,archived) VALUES(?,?,?,?,0)`, b.Id, b.Name, b.Currency, b.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano))
	return b, err
}

func (s *Store) ListBooks(ctx context.Context, includeArchived bool) ([]*ledgerpb.Book, error) {
	query := `SELECT id,name,currency,created_at,archived FROM books`
	if !includeArchived {
		query += ` WHERE archived=0`
	}
	query += ` ORDER BY created_at,id`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*ledgerpb.Book
	for rows.Next() {
		var id, n, c, ts string
		var archived int
		if err := rows.Scan(&id, &n, &c, &ts, &archived); err != nil {
			return nil, err
		}
		t, _ := time.Parse(time.RFC3339Nano, ts)
		out = append(out, &ledgerpb.Book{Id: id, Name: n, Currency: c, CreatedAt: timestamppb.New(t), Archived: archived != 0})
	}
	return out, rows.Err()
}

func (s *Store) ArchiveBook(ctx context.Context, bookID, actor string) (*ledgerpb.Book, error) {
	var book ledgerpb.Book
	var created string
	var archived int
	if err := s.db.QueryRowContext(ctx, `SELECT id,name,currency,created_at,archived FROM books WHERE id=?`, bookID).Scan(&book.Id, &book.Name, &book.Currency, &created, &archived); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	if archived == 0 {
		if _, err := s.db.ExecContext(ctx, `UPDATE books SET archived=1 WHERE id=?`, bookID); err != nil {
			return nil, err
		}
		if err := s.recordAudit(ctx, "book", bookID, actor, "archived book", "archived=false"); err != nil {
			return nil, err
		}
	}
	createdAt, _ := time.Parse(time.RFC3339Nano, created)
	book.CreatedAt = timestamppb.New(createdAt)
	book.Archived = true
	return &book, nil
}

func (s *Store) CreateAccount(ctx context.Context, bookID, name string, kind ledgerpb.AccountKind) (*ledgerpb.Account, error) {
	kindName, ok := declaredAccountKinds[kind]
	if strings.TrimSpace(bookID) == "" || strings.TrimSpace(name) == "" {
		return nil, errors.New("book_id and name are required")
	}
	if !ok {
		return nil, fmt.Errorf("account kind is invalid; accepted values: %s", acceptedAccountKinds)
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id=?`, bookID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	a := &ledgerpb.Account{Id: uuid.NewString(), BookId: bookID, Name: strings.TrimSpace(name), Kind: kindName, CreatedAt: timestamppb.New(s.now())}
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
		p.Audit, err = s.loadAudit(ctx, p.Id)
		return &p, true, err
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
	p.Audit, err = s.loadAudit(ctx, p.Id)
	return &p, false, err
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, posting := range out {
		posting.Audit, err = s.loadAudit(ctx, posting.Id)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
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
	p.Audit, err = s.loadAudit(ctx, p.Id)
	if err != nil {
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
	for _, posting := range posts {
		posting.Audit, err = s.loadAudit(ctx, posting.Id)
		if err != nil {
			return nil, err
		}
	}
	return posts, nil
}

func (s *Store) Position(ctx context.Context, bookID string) (*ledgerpb.PositionResponse, error) {
	var cash, revenue, expense int64
	var currency string
	var archived int
	if err := s.db.QueryRowContext(ctx, `SELECT currency,archived FROM books WHERE id=?`, bookID).Scan(&currency, &archived); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	if archived != 0 {
		return nil, fmt.Errorf("book %q is archived and excluded from position", bookID)
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
		switch strings.ToUpper(kind) {
		case "ASSET":
			cash += value
		case "REVENUE":
			revenue += value
		case "EXPENSE":
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
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN p.amount_minor > 0 THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN p.amount_minor < 0 THEN -p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN upper(a.kind)='REVENUE' THEN p.amount_minor ELSE 0 END),0), COALESCE(SUM(CASE WHEN upper(a.kind)='EXPENSE' THEN -p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE `+period, periodArgs...).Scan(&inflow, &outflow, &revenue, &expense); err != nil {
		return nil, err
	}
	ending := to
	if ending == "" {
		ending = "9999-12-31T23:59:59Z"
	}
	var openingCash, closingCash, nonCashAssets, liabilities int64
	if from != "" {
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN upper(a.kind)='ASSET' THEN p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? AND p.occurred_at<?`, bookID, from).Scan(&openingCash); err != nil {
			return nil, err
		}
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN upper(a.kind)='ASSET' THEN p.amount_minor ELSE 0 END),0), 0, COALESCE(SUM(CASE WHEN upper(a.kind)='LIABILITY' THEN p.amount_minor ELSE 0 END),0) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? AND p.occurred_at<=?`, bookID, ending).Scan(&closingCash, &nonCashAssets, &liabilities); err != nil {
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
	if g.SustainPeriodUnit == ledgerpb.SustainPeriodUnit_SUSTAIN_PERIOD_UNIT_UNSPECIFIED {
		g.SustainPeriodUnit = ledgerpb.SustainPeriodUnit_MONTH
	}
	if g.ThresholdRatio < 0 || g.ThresholdRatio > 1 {
		return nil, errors.New("goal threshold_ratio must be between 0 and 1")
	}
	if g.ThresholdRatio == 0 && strings.TrimSpace(g.ComparandMetric) == "" && g.ThresholdMinor == 0 {
		return nil, errors.New("goal must declare a threshold, ratio, or comparand metric")
	}
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id=? AND archived=0`, bookID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("book %q not found: %w", bookID, err)
	}
	g.Id = uuid.NewString()
	_, err := s.db.ExecContext(ctx, `INSERT INTO goals(id,book_id,name,metric,comparator,threshold_minor,sustain_periods,buffer_multiple,threshold_ratio,comparand_metric,sustain_period_unit) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, g.Id, bookID, g.Name, g.Metric, g.Comparator, g.ThresholdMinor, g.SustainPeriods, g.BufferMultiple, g.ThresholdRatio, g.ComparandMetric, int32(g.SustainPeriodUnit))
	return g, err
}

func (s *Store) ListGoals(ctx context.Context, bookID string, includeArchived bool) ([]*ledgerpb.GoalVerdict, error) {
	query := `SELECT g.id,g.name,g.metric,g.comparator,g.threshold_minor,g.sustain_periods,g.buffer_multiple,g.threshold_ratio,g.comparand_metric,g.sustain_period_unit,g.archived FROM goals g JOIN books b ON b.id=g.book_id WHERE g.book_id=?`
	if !includeArchived {
		query += ` AND g.archived=0 AND b.archived=0`
	}
	query += ` ORDER BY g.name`
	rows, err := s.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	goals := make([]*ledgerpb.Goal, 0)
	for rows.Next() {
		var g ledgerpb.Goal
		var unit int32
		var archived int
		if err := rows.Scan(&g.Id, &g.Name, &g.Metric, &g.Comparator, &g.ThresholdMinor, &g.SustainPeriods, &g.BufferMultiple, &g.ThresholdRatio, &g.ComparandMetric, &unit, &archived); err != nil {
			_ = rows.Close()
			return nil, err
		}
		g.SustainPeriodUnit = ledgerpb.SustainPeriodUnit(unit)
		g.BookId = bookID
		g.Archived = archived != 0
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
		value, valueLabel, available, availabilityReason := s.goalMetricValue(ctx, bookID, position, g.Metric)
		observedLabel := valueLabel
		threshold := float64(g.ThresholdMinor)
		if g.ThresholdRatio > 0 {
			threshold = g.ThresholdRatio
		}
		if g.ComparandMetric != "" {
			comparand, _, comparandAvailable, comparandReason := s.goalMetricValue(ctx, bookID, position, g.ComparandMetric)
			if !comparandAvailable {
				available = false
				availabilityReason = "comparand " + g.ComparandMetric + " unavailable: " + comparandReason
			}
			if g.BufferMultiple > 0 {
				comparand *= g.BufferMultiple
			}
			value = value - comparand
			observedLabel = fmt.Sprintf("%.4g", value)
			threshold = 0
		}
		if !available {
			out = append(out, &ledgerpb.GoalVerdict{
				Goal:             g,
				SustainedPeriods: 0,
				RequiredPeriods:  g.SustainPeriods,
				PeriodUnit:       g.SustainPeriodUnit,
				Explanation:      "UNKNOWN: " + availabilityReason,
			})
			continue
		}
		sustained, sustainErr := s.sustainedGoalPeriods(ctx, bookID, g)
		if sustainErr != nil {
			return nil, sustainErr
		}
		out = append(out, &ledgerpb.GoalVerdict{Goal: g, Met: sustained >= g.SustainPeriods, SustainedPeriods: sustained, RequiredPeriods: g.SustainPeriods, PeriodUnit: g.SustainPeriodUnit, Explanation: fmt.Sprintf("read-time %s=%s; required %s %.4g for %d sustained %s period(s)", g.Metric, observedLabel, g.Comparator, threshold, g.SustainPeriods, strings.ToLower(g.SustainPeriodUnit.String()))})
	}
	return out, nil
}

func (s *Store) ArchiveGoal(ctx context.Context, goalID, actor string) (*ledgerpb.Goal, error) {
	goal, err := s.goalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	if !goal.Archived {
		if _, err := s.db.ExecContext(ctx, `UPDATE goals SET archived=1 WHERE id=?`, goalID); err != nil {
			return nil, err
		}
		if err := s.recordAudit(ctx, "goal", goalID, actor, "archived goal", "archived=false"); err != nil {
			return nil, err
		}
		goal.Archived = true
	}
	return goal, nil
}

func (s *Store) ReparentGoal(ctx context.Context, goalID, bookID, actor string) (*ledgerpb.Goal, error) {
	goal, err := s.goalByID(ctx, goalID)
	if err != nil {
		return nil, err
	}
	var active int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id=? AND archived=0`, bookID).Scan(&active); err != nil {
		return nil, fmt.Errorf("book %q not found or archived: %w", bookID, err)
	}
	if goal.BookId != bookID {
		prior := goal.BookId
		if _, err := s.db.ExecContext(ctx, `UPDATE goals SET book_id=? WHERE id=?`, bookID, goalID); err != nil {
			return nil, err
		}
		if err := s.recordAudit(ctx, "goal", goalID, actor, "reparented goal", prior); err != nil {
			return nil, err
		}
		goal.BookId = bookID
	}
	return goal, nil
}

func (s *Store) goalByID(ctx context.Context, goalID string) (*ledgerpb.Goal, error) {
	var goal ledgerpb.Goal
	var unit, archived int32
	if err := s.db.QueryRowContext(ctx, `SELECT id,book_id,name,metric,comparator,threshold_minor,sustain_periods,buffer_multiple,threshold_ratio,comparand_metric,sustain_period_unit,archived FROM goals WHERE id=?`, goalID).Scan(&goal.Id, &goal.BookId, &goal.Name, &goal.Metric, &goal.Comparator, &goal.ThresholdMinor, &goal.SustainPeriods, &goal.BufferMultiple, &goal.ThresholdRatio, &goal.ComparandMetric, &unit, &archived); err != nil {
		return nil, fmt.Errorf("goal %q not found: %w", goalID, err)
	}
	goal.SustainPeriodUnit = ledgerpb.SustainPeriodUnit(unit)
	goal.Archived = archived != 0
	return &goal, nil
}

func (s *Store) sustainedGoalPeriods(ctx context.Context, bookID string, goal *ledgerpb.Goal) (int32, error) {
	if goal.Metric == "services_capacity" || goal.SustainPeriodUnit == ledgerpb.SustainPeriodUnit_WEEK {
		rows, err := s.db.QueryContext(ctx, `SELECT strftime('%Y-%W',observed_at),value FROM operator_measures WHERE path='timeAllocation.services' AND status NOT IN ('stale','pending-operator') AND (source_path='shared/operator-inputs.json' OR source_path LIKE '%/shared/operator-inputs.json') ORDER BY observed_at DESC`)
		if err != nil {
			return 0, err
		}
		defer rows.Close()
		weeks := map[string]float64{}
		for rows.Next() {
			var week string
			var value float64
			if err := rows.Scan(&week, &value); err != nil {
				return 0, err
			}
			if _, ok := weeks[week]; !ok {
				if value > 1 {
					value /= 100
				}
				weeks[week] = value
			}
		}
		if err := rows.Err(); err != nil {
			return 0, err
		}
		keys := make([]string, 0, len(weeks))
		for key := range weeks {
			keys = append(keys, key)
		}
		sort.Sort(sort.Reverse(sort.StringSlice(keys)))
		var sustained int32
		threshold := goal.ThresholdRatio
		if threshold == 0 {
			threshold = float64(goal.ThresholdMinor) / 100
		}
		for _, key := range keys {
			if !compareGoalFloat(weeks[key], goal.Comparator, threshold) {
				break
			}
			sustained++
			if sustained >= goal.SustainPeriods {
				break
			}
		}
		return sustained, nil
	}
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
		// A cash-only observation cannot prove a revenue-versus-burn rule.
		// Treating both absent measures as zero would turn an unknown posture
		// into a passing default-alive verdict.
		if strings.EqualFold(goal.Metric, "revenue") && strings.EqualFold(goal.ComparandMetric, "burn") && v.revenue == 0 && v.expense == 0 {
			break
		}
		value := float64(v.cash)
		switch strings.ToLower(goal.Metric) {
		case "revenue", "income":
			value = float64(v.revenue)
		case "expense":
			value = float64(v.expense)
		case "burn":
			value = float64(v.expense - v.revenue)
		}
		threshold := float64(goal.ThresholdMinor)
		if goal.ComparandMetric != "" {
			comparand := float64(v.expense - v.revenue)
			if strings.EqualFold(goal.ComparandMetric, "revenue") {
				comparand = float64(v.revenue)
			}
			if strings.EqualFold(goal.ComparandMetric, "burn") && goal.BufferMultiple > 0 {
				comparand *= goal.BufferMultiple
			}
			value -= comparand
			threshold = 0
		}
		if !compareGoalFloat(value, goal.Comparator, threshold) {
			break
		}
		sustained++
		if sustained >= goal.SustainPeriods {
			break
		}
	}
	return sustained, nil
}

func compareGoalFloat(value float64, comparator string, threshold float64) bool {
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

func (s *Store) goalMetricValue(ctx context.Context, bookID string, position *ledgerpb.PositionResponse, metric string) (float64, string, bool, string) {
	switch strings.ToLower(strings.TrimSpace(metric)) {
	case "revenue", "income":
		available, reason := s.hasAccountObservation(ctx, bookID, "revenue", "income")
		return float64(position.RevenueMinor), fmt.Sprintf("%d", position.RevenueMinor), available, reason
	case "expense":
		available, reason := s.hasAccountObservation(ctx, bookID, "expense")
		return float64(position.ExpenseMinor), fmt.Sprintf("%d", position.ExpenseMinor), available, reason
	case "burn":
		available, reason := s.hasAccountObservation(ctx, bookID, "expense")
		return float64(position.BurnMinor), fmt.Sprintf("%d", position.BurnMinor), available, reason
	case "runway":
		if !position.RunwayAvailable {
			return position.RunwayMonths, fmt.Sprintf("%.2f months", position.RunwayMonths), false, position.RunwayReason
		}
		return position.RunwayMonths, fmt.Sprintf("%.2f months", position.RunwayMonths), true, ""
	case "services_capacity":
		var value float64
		var observed int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM operator_measures WHERE path='timeAllocation.services' AND status NOT IN ('stale','pending-operator') AND (source_path='shared/operator-inputs.json' OR source_path LIKE '%/shared/operator-inputs.json')`).Scan(&observed); err != nil {
			return 0, "", false, err.Error()
		}
		if observed == 0 {
			return 0, "0.0000", false, "timeAllocation.services observation is absent"
		}
		_ = s.db.QueryRowContext(ctx, `SELECT value FROM operator_measures WHERE path='timeAllocation.services' AND status NOT IN ('stale','pending-operator') AND (source_path='shared/operator-inputs.json' OR source_path LIKE '%/shared/operator-inputs.json') ORDER BY observed_at DESC,id DESC LIMIT 1`).Scan(&value)
		if value > 1 {
			value /= 100
		}
		return value, fmt.Sprintf("%.4f", value), true, ""
	case "services_revenue", "subscription_revenue":
		var value float64
		needle := "services"
		if strings.Contains(strings.ToLower(metric), "subscription") {
			needle = "subscription"
		}
		var observed int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM postings p WHERE p.book_id=? AND lower(p.category) LIKE '%'||?||'%'`, bookID, needle).Scan(&observed); err != nil {
			return 0, "", false, err.Error()
		}
		if observed == 0 {
			return 0, "0", false, metric + " observation is absent"
		}
		_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(p.amount_minor),0) FROM postings p WHERE p.book_id=? AND lower(p.category) LIKE '%'||?||'%'`, bookID, needle).Scan(&value)
		return value, fmt.Sprintf("%.0f", value), true, ""
	default:
		return float64(position.CashMinor), fmt.Sprintf("%d", position.CashMinor), true, ""
	}
}

func (s *Store) hasAccountObservation(ctx context.Context, bookID string, kinds ...string) (bool, string) {
	placeholders := make([]string, len(kinds))
	args := []any{bookID}
	for i, kind := range kinds {
		placeholders[i] = "?"
		args = append(args, kind)
	}
	var observed int
	query := `SELECT COUNT(*) FROM postings p JOIN accounts a ON a.id=p.account_id WHERE p.book_id=? AND lower(a.kind) IN (` + strings.Join(placeholders, ",") + ")"
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&observed); err != nil {
		return false, err.Error()
	}
	if observed == 0 {
		return false, "no " + strings.Join(kinds, "/") + " observation exists"
	}
	return true, ""
}
