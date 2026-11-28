// Package ledgertest serves Money Ledger's real modules behind the production
// server and access gate, over a per-test SQLite journal, with the fixture
// identity providers configured. Tests reach it over HTTP with the generated
// clients, exactly as callers do.
package ledgertest

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vrooli/api-core/browsersession"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access/access_v1connect"
	ingestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest/ingest_v1connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"

	accessH "money-ledger/handlers/access"
	ingestH "money-ledger/handlers/ingest"
	ledgerH "money-ledger/handlers/ledger"
	"money-ledger/internal/access"
	"money-ledger/internal/ledger"
	"money-ledger/internal/modules"
	"money-ledger/internal/server"
	"money-ledger/internal/testutil/authfixture"
)

// Ledger is one served journal with a book and an asset account to write to.
type Ledger struct {
	DB       *database.RoutedDB
	URL      string
	Identity *authfixture.Identity
	Book     *ledgerpb.Book
	Account  *ledgerpb.Account
	Access   accessconnect.AccessServiceClient
	Books    ledgerconnect.BooksServiceClient
	Journal  ledgerconnect.JournalServiceClient
	Position ledgerconnect.PositionServiceClient
	Ingest   ingestconnect.IngestServiceClient
}

// New serves the ledger. wrap middleware runs outside the server, outermost
// last, the way the platform server installs request provenance.
func New(t *testing.T, wrap ...func(http.Handler) http.Handler) *Ledger {
	t.Helper()
	ctx := context.Background()
	identity := authfixture.New(t)
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	if err := database.EnsureSchemas(ctx, db.Primary(), modules.AllSchemas()...); err != nil {
		t.Fatalf("schemas: %v", err)
	}
	journal := ledger.NewStore(db, nil)
	if err := journal.EnsureMigrations(ctx); err != nil {
		t.Fatalf("migrations: %v", err)
	}
	book, err := journal.CreateBook(ctx, "Fixture book", "USD")
	if err != nil {
		t.Fatalf("book: %v", err)
	}
	account, err := journal.CreateAccount(ctx, book.Id, "Fixture cash", ledgerpb.AccountKind_ASSET)
	if err != nil {
		t.Fatalf("account: %v", err)
	}
	cfg, err := access.Authentication(os.Getenv)
	if err != nil {
		t.Fatalf("authentication: %v", err)
	}
	gate := access.NewGate(cfg)
	logger := log.New(io.Discard, "", 0)
	srv := server.New(server.Deps{Clock: schedule.System(), Logger: logger, Gate: gate},
		accessH.Module(gate, &browsersession.Edge{Resource: access.Resource, Accounts: func(context.Context) (browsersession.Accounts, error) { return identity.Accounts(), nil }}), ledgerH.Module(db, nil, logger), ingestH.Module(db, nil, logger))
	var handler http.Handler = srv.Handler()
	for _, w := range wrap {
		handler = w(handler)
	}
	served := httptest.NewServer(handler)
	t.Cleanup(served.Close)
	client := served.Client()
	return &Ledger{
		DB: db, URL: served.URL, Identity: identity, Book: book, Account: account,
		Access:   accessconnect.NewAccessServiceClient(client, served.URL),
		Books:    ledgerconnect.NewBooksServiceClient(client, served.URL),
		Journal:  ledgerconnect.NewJournalServiceClient(client, served.URL),
		Position: ledgerconnect.NewPositionServiceClient(client, served.URL),
		Ingest:   ingestconnect.NewIngestServiceClient(client, served.URL),
	}
}

// Browser is one browser profile on the ledger's own page: it keeps cookies
// across calls and marks each request same-origin, as a browser does for the
// page's own script.
type Browser struct {
	Access  accessconnect.AccessServiceClient
	Ingest  ingestconnect.IngestServiceClient
	Journal ledgerconnect.JournalServiceClient
	Jar     http.CookieJar
}

// Browser opens a fresh browser profile with no session.
func (l *Ledger) Browser(t testing.TB) *Browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	client := &http.Client{Jar: jar, Transport: sameOrigin{http.DefaultTransport}}
	return &Browser{
		Access:  accessconnect.NewAccessServiceClient(client, l.URL),
		Ingest:  ingestconnect.NewIngestServiceClient(client, l.URL),
		Journal: ledgerconnect.NewJournalServiceClient(client, l.URL),
		Jar:     jar,
	}
}

type sameOrigin struct{ next http.RoundTripper }

func (s sameOrigin) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	return s.next.RoundTrip(r)
}

// Count returns the number of rows in table, for asserting a refused write
// stored nothing.
func (l *Ledger) Count(t testing.TB, table string) int {
	t.Helper()
	var n int
	if err := l.DB.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}
