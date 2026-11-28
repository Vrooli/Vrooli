package ledger_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	mlpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	mlconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	_ "modernc.org/sqlite"

	"treasury/internal/ledger"
)

func newDestinationStore(t *testing.T) (*ledger.DestinationStore, *ledger.SQLiteRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	_, err = db.Exec(ledger.Schema())
	require.NoError(t, err)
	return ledger.NewDestinationStore(db), ledger.NewSQLiteRepository(db), db
}

// The operator's choice is one row: choosing again replaces it, and an empty
// store is the "not chosen" state, not an error to delivery.
func TestDestinationStoreHoldsOneChoice(t *testing.T) {
	store, _, _ := newDestinationStore(t)
	ctx := context.Background()
	_, err := store.Destination(ctx)
	require.ErrorIs(t, err, ledger.ErrDestinationUnset)
	at := time.Date(2026, 11, 14, 12, 0, 0, 123, time.UTC)
	require.NoError(t, store.Set(ctx, ledger.Destination{BookID: "b1", AccountID: "a1", BookName: "Book", AccountName: "Cash", Currency: "USD", SetAt: at}))
	require.NoError(t, store.Set(ctx, ledger.Destination{BookID: "b2", AccountID: "a2", BookName: "Other", AccountName: "Cash 2", Currency: "USD", SetAt: at.Add(time.Hour)}))
	got, err := store.Destination(ctx)
	require.NoError(t, err)
	require.Equal(t, ledger.Destination{BookID: "b2", AccountID: "a2", BookName: "Other", AccountName: "Cash 2", Currency: "USD", Source: ledger.DestinationSourceOperator, SetAt: at.Add(time.Hour)}, got)
}

// The rehearsal seed is a fallback only: a stored choice always wins, and the
// seed is never written to the store.
func TestSeededDestinationNeverOverridesTheOperator(t *testing.T) {
	store, _, _ := newDestinationStore(t)
	ctx := context.Background()
	seeded := ledger.SeededDestination{Store: store, SeedBook: "seed-book", SeedAccount: "seed-account"}
	got, err := seeded.Destination(ctx)
	require.NoError(t, err)
	require.Equal(t, ledger.Destination{BookID: "seed-book", AccountID: "seed-account", Source: ledger.DestinationSourceEnvironment}, got)
	_, stored, err := store.Get(ctx)
	require.NoError(t, err)
	require.False(t, stored, "reading the seed stores nothing")
	require.NoError(t, store.Set(ctx, ledger.Destination{BookID: "b1", AccountID: "a1", Currency: "USD", SetAt: time.Now()}))
	got, err = seeded.Destination(ctx)
	require.NoError(t, err)
	require.Equal(t, "b1", got.BookID)
	require.Equal(t, ledger.DestinationSourceOperator, got.Source)
	for _, half := range []ledger.SeededDestination{{Store: newEmptyStore(t), SeedBook: "only-book"}, {Store: newEmptyStore(t), SeedAccount: "only-account"}, {Store: newEmptyStore(t)}} {
		_, err = half.Destination(ctx)
		require.ErrorIs(t, err, ledger.ErrDestinationUnset, "a half seed is not a destination")
	}
}

func newEmptyStore(t *testing.T) *ledger.DestinationStore {
	store, _, _ := newDestinationStore(t)
	return store
}

type fakeBooks struct {
	mlconnect.UnimplementedBooksServiceHandler
	books    []*mlpb.Book
	accounts []*mlpb.Account
	// includeArchived is what the validator asked for.
	includeArchived bool
}

func (f *fakeBooks) ListBooks(_ context.Context, request *connect.Request[mlpb.ListBooksRequest]) (*connect.Response[mlpb.ListBooksResponse], error) {
	f.includeArchived = request.Msg.GetIncludeArchived()
	return connect.NewResponse(&mlpb.ListBooksResponse{Books: f.books}), nil
}

func (f *fakeBooks) ListAccounts(_ context.Context, request *connect.Request[mlpb.ListAccountsRequest]) (*connect.Response[mlpb.ListAccountsResponse], error) {
	var out []*mlpb.Account
	for _, a := range f.accounts {
		if a.GetBookId() == request.Msg.GetBookId() {
			out = append(out, a)
		}
	}
	return connect.NewResponse(&mlpb.ListAccountsResponse{Accounts: out}), nil
}

// A choice is checked against what Money Ledger actually has: the book exists,
// is not archived and is USD, and the account belongs to that book. Every refusal
// says which and is a DestinationRefusal, never a transport failure.
func TestMoneyLedgerValidatorChecksTheChoice(t *testing.T) {
	fake := &fakeBooks{
		books: []*mlpb.Book{
			{Id: "usd", Name: "Monetization Operating", Currency: "USD"},
			{Id: "old", Name: "Old", Currency: "USD", Archived: true},
			{Id: "eur", Name: "Euro book", Currency: "EUR"},
		},
		accounts: []*mlpb.Account{
			{Id: "cash", BookId: "usd", Name: "Operating Cash", Kind: "ASSET"},
			{Id: "old-cash", BookId: "old", Name: "Old cash", Kind: "ASSET"},
			{Id: "eur-cash", BookId: "eur", Name: "Euro cash", Kind: "ASSET"},
		},
	}
	path, handler := mlconnect.NewBooksServiceHandler(fake)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	validator, err := ledger.NewMoneyLedgerValidator(server.URL, server.Client())
	require.NoError(t, err)
	ctx := context.Background()

	got, err := validator.Validate(ctx, " usd ", "cash")
	require.NoError(t, err)
	require.Equal(t, ledger.Destination{BookID: "usd", AccountID: "cash", BookName: "Monetization Operating", AccountName: "Operating Cash", Currency: "USD"}, got)
	require.True(t, fake.includeArchived, "archived books are read so an archived choice is named, not reported missing")

	for _, test := range []struct{ name, book, account, reason string }{
		{"blank", "", "", "both a book id and an account id"},
		{"no such book", "nope", "cash", `no book "nope"`},
		{"archived book", "old", "old-cash", "archived"},
		{"other currency", "eur", "eur-cash", "is in EUR"},
		{"account in another book", "usd", "eur-cash", `not in book "usd"`},
		{"no such account", "usd", "nope", `account "nope" is not in book`},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := validator.Validate(ctx, test.book, test.account)
			var refusal ledger.DestinationRefusal
			require.ErrorAs(t, err, &refusal)
			require.Contains(t, refusal.Reason, test.reason)
		})
	}
}

func TestMoneyLedgerValidatorNeedsAnOrigin(t *testing.T) {
	_, err := ledger.NewMoneyLedgerValidator(" ", nil)
	require.ErrorContains(t, err, "origin could not be resolved")
}

func TestQueuedCountIsEveryWaitingSettlement(t *testing.T) {
	_, repo, db := newDestinationStore(t)
	ctx := context.Background()
	n, err := repo.QueuedCount(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	for _, row := range []struct{ id, status string }{{"q1", "queued"}, {"q2", "queued"}, {"done", "accepted"}} {
		_, err = db.Exec(`INSERT INTO ledger_emissions(id,settlement_id,external_id,adapter_id,account_id,book_id,amount_minor,currency,basis,occurred_at,fetched_at,description,status,attempts,last_error,created_at,accepted_at) VALUES(?,?,?,'treasury','','',-100,'USD','authoritative','2026-11-01T12:00:00Z','2026-11-01T12:00:01Z','test',?,0,'','2026-11-01T12:00:02Z','')`, row.id, row.id, "treasury:"+row.id, row.status)
		require.NoError(t, err)
	}
	n, err = repo.QueuedCount(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n, "only settlements still waiting count, whatever a list filters to")
}
