package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
)

// Destination sources, as reported to the operator.
const (
	DestinationSourceOperator    = "operator"
	DestinationSourceEnvironment = "environment"
)

// SettlementCurrency is the one currency Treasury settles in today; a
// destination book in any other currency would post mismatched amounts.
const SettlementCurrency = "USD"

// ErrDestinationUnset means no Money Ledger book and account has been chosen,
// so settlements wait in the durable queue.
var ErrDestinationUnset = errors.New("settlement destination is not configured: the operator has not chosen a Money Ledger book and account (treasury ledger destination-set --book-id <id> --account-id <id>; money-ledger ledger books-list and accounts-list show the choices)")

// Destination is the one book and account settlements post to.
type Destination struct {
	BookID, AccountID, BookName, AccountName, Currency string
	Source                                             string
	SetAt                                              time.Time
}

// DestinationSource yields the current destination, or ErrDestinationUnset.
type DestinationSource interface {
	Destination(context.Context) (Destination, error)
}

// DestinationStore is the one-row table holding the operator's choice.
type DestinationStore struct{ db DB }

func NewDestinationStore(db DB) *DestinationStore { return &DestinationStore{db: db} }

func (s *DestinationStore) Get(ctx context.Context) (Destination, bool, error) {
	var d Destination
	var setAt string
	err := s.db.QueryRowContext(ctx, `SELECT book_id,account_id,book_name,account_name,currency,set_at FROM ledger_destination WHERE id=1`).
		Scan(&d.BookID, &d.AccountID, &d.BookName, &d.AccountName, &d.Currency, &setAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Destination{}, false, nil
	}
	if err != nil {
		return Destination{}, false, err
	}
	d.Source = DestinationSourceOperator
	d.SetAt, err = time.Parse(time.RFC3339Nano, setAt)
	if err != nil {
		return Destination{}, false, fmt.Errorf("read ledger destination time: %w", err)
	}
	return d, true, nil
}

// Set replaces the one destination. The caller has already validated it.
func (s *DestinationStore) Set(ctx context.Context, d Destination) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO ledger_destination(id,book_id,account_id,book_name,account_name,currency,set_at) VALUES(1,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET book_id=excluded.book_id,account_id=excluded.account_id,book_name=excluded.book_name,account_name=excluded.account_name,currency=excluded.currency,set_at=excluded.set_at`,
		d.BookID, d.AccountID, d.BookName, d.AccountName, d.Currency, d.SetAt.UTC().Format(time.RFC3339Nano))
	return err
}

// Destination implements DestinationSource.
func (s *DestinationStore) Destination(ctx context.Context) (Destination, error) {
	d, ok, err := s.Get(ctx)
	if err != nil {
		return Destination{}, err
	}
	if !ok {
		return Destination{}, ErrDestinationUnset
	}
	return d, nil
}

// SeededDestination is the destination Treasury delivers to: the operator's
// stored choice, else the rehearsal seed from TREASURY_LEDGER_BOOK_ID and
// TREASURY_LEDGER_ACCOUNT_ID. A stored choice always wins, so a seed can never
// override the operator. The seed is never written to the store.
type SeededDestination struct {
	Store                 *DestinationStore
	SeedBook, SeedAccount string
}

func (s SeededDestination) Destination(ctx context.Context) (Destination, error) {
	d, ok, err := s.Store.Get(ctx)
	if err != nil {
		return Destination{}, err
	}
	if ok {
		return d, nil
	}
	book, account := strings.TrimSpace(s.SeedBook), strings.TrimSpace(s.SeedAccount)
	if book == "" || account == "" {
		return Destination{}, ErrDestinationUnset
	}
	return Destination{BookID: book, AccountID: account, Source: DestinationSourceEnvironment}, nil
}

// DestinationValidator checks a chosen book and account against Money Ledger.
type DestinationValidator interface {
	Validate(ctx context.Context, bookID, accountID string) (Destination, error)
}

// DestinationRefusal is a choice Money Ledger shows to be unusable; its message
// is safe to show the operator.
type DestinationRefusal struct{ Reason string }

func (r DestinationRefusal) Error() string { return r.Reason }

// MoneyLedgerValidator reads Money Ledger's books and accounts. It reads only:
// Treasury never creates the operator's journal structure.
type MoneyLedgerValidator struct {
	client ledgerconnect.BooksServiceClient
}

func NewMoneyLedgerValidator(baseURL string, client *http.Client) (*MoneyLedgerValidator, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, errors.New("the Money Ledger origin could not be resolved (is money-ledger running?)")
	}
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &MoneyLedgerValidator{client: ledgerconnect.NewBooksServiceClient(client, baseURL)}, nil
}

func (v *MoneyLedgerValidator) Validate(ctx context.Context, bookID, accountID string) (Destination, error) {
	bookID, accountID = strings.TrimSpace(bookID), strings.TrimSpace(accountID)
	if bookID == "" || accountID == "" {
		return Destination{}, DestinationRefusal{"both a book id and an account id are required"}
	}
	books, err := v.client.ListBooks(ctx, connect.NewRequest(&ledgerpb.ListBooksRequest{IncludeArchived: true}))
	if err != nil {
		return Destination{}, fmt.Errorf("read Money Ledger books: %w", err)
	}
	var book *ledgerpb.Book
	for _, candidate := range books.Msg.GetBooks() {
		if candidate.GetId() == bookID {
			book = candidate
		}
	}
	switch {
	case book == nil:
		return Destination{}, DestinationRefusal{fmt.Sprintf("Money Ledger has no book %q (money-ledger ledger books-list)", bookID)}
	case book.GetArchived():
		return Destination{}, DestinationRefusal{fmt.Sprintf("book %q (%s) is archived and takes no new postings", bookID, book.GetName())}
	case !strings.EqualFold(book.GetCurrency(), SettlementCurrency):
		return Destination{}, DestinationRefusal{fmt.Sprintf("book %q (%s) is in %s; Treasury settles in %s", bookID, book.GetName(), book.GetCurrency(), SettlementCurrency)}
	}
	accounts, err := v.client.ListAccounts(ctx, connect.NewRequest(&ledgerpb.ListAccountsRequest{BookId: bookID}))
	if err != nil {
		return Destination{}, fmt.Errorf("read Money Ledger accounts: %w", err)
	}
	for _, account := range accounts.Msg.GetAccounts() {
		if account.GetId() == accountID && account.GetBookId() == bookID {
			return Destination{BookID: bookID, AccountID: accountID, BookName: book.GetName(), AccountName: account.GetName(), Currency: strings.ToUpper(book.GetCurrency())}, nil
		}
	}
	return Destination{}, DestinationRefusal{fmt.Sprintf("account %q is not in book %q (%s) (money-ledger ledger accounts-list --book-id %s)", accountID, bookID, book.GetName(), bookID)}
}

var (
	_ DestinationSource    = (*DestinationStore)(nil)
	_ DestinationSource    = SeededDestination{}
	_ DestinationValidator = (*MoneyLedgerValidator)(nil)
)
