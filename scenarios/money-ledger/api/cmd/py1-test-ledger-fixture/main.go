// Command py1-test-ledger-fixture starts an isolated, loopback-only Money
// Ledger used by the LPBS PY1 local Stripe test-mode replay. It verifies
// Scenario Authenticator tokens through the production Money Ledger gate and
// emits only the fixture URL and scope identifiers.
package main

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/api-core/browsersession"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	_ "modernc.org/sqlite"
	accessH "money-ledger/handlers/access"
	ingestH "money-ledger/handlers/ingest"
	ledgerH "money-ledger/handlers/ledger"
	"money-ledger/internal/access"
	"money-ledger/internal/ingest"
	"money-ledger/internal/ledger"
	"money-ledger/internal/modules"
	"money-ledger/internal/server"
)

type fixtureIdentity struct {
	FixtureID        string `json:"fixture_id"`
	AdapterID        string `json:"adapter_id"`
	BookID           string `json:"book_id"`
	RevenueAccountID string `json:"revenue_account_id"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "PY1 test Money Ledger fixture failed")
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	dir, err := os.MkdirTemp("", "vrooli-py1-ledger-")
	if err != nil {
		return errorsNoDetails("create isolated ledger directory")
	}
	defer os.RemoveAll(dir)
	dbPath := filepath.Join(dir, "ledger.sqlite")
	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return errorsNoDetails("open isolated ledger database")
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	db := database.NewFromPrimary(raw)
	if err := database.EnsureSchemas(ctx, raw, modules.AllSchemas()...); err != nil {
		return errorsNoDetails("initialize isolated ledger schemas")
	}
	journal := ledger.NewStore(db, nil)
	if err := journal.EnsureMigrations(ctx); err != nil {
		return errorsNoDetails("initialize isolated ledger journal")
	}
	book, err := journal.CreateBook(ctx, "TEST ONLY PY1 checkout receipts", "USD")
	if err != nil {
		return errorsNoDetails("create isolated USD book")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(dbPath)))
	identity := fixtureIdentity{FixtureID: "money-ledger-test-sqlite:" + digest, AdapterID: "lpbs-py1-test", BookID: book.Id}
	account, err := journal.CreateAccount(ctx, book.Id, "TEST ONLY LPBS checkout revenue", ledgerpb.AccountKind_REVENUE)
	if err != nil {
		return errorsNoDetails("create isolated checkout revenue account")
	}
	identity.RevenueAccountID = account.Id
	if _, err := ingest.NewStore(db, nil).RegisterAdapter(ctx, &ingestpb.Adapter{Id: identity.AdapterID, Name: "LPBS PY1 test Checkout", Kind: ingestpb.AdapterKind_ADAPTER_KIND_FILE, Enabled: true}); err != nil {
		return errorsNoDetails("register isolated checkout adapter")
	}

	// No authfixture or generated token is used here. Money Ledger's normal
	// scenario-authenticator verifier discovers the local issuer JWKS.
	authentication, err := access.Authentication(os.Getenv)
	if err != nil {
		return errorsNoDetails("configure Scenario Authenticator verification")
	}
	gate := access.NewGate(authentication)
	logger := log.New(os.Stderr, "", 0)
	api := server.New(server.Deps{Clock: schedule.System(), Logger: logger, Gate: gate},
		accessH.Module(gate, browsersession.New(access.Resource)),
		ingestH.Module(db, nil, logger),
		ledgerH.Module(db, nil, logger),
	)
	stop := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/__journey/identity", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(identity)
	})
	mux.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		select {
		case <-stop:
		default:
			close(stop)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	mux.Handle("/", api.Handler())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return errorsNoDetails("bind loopback fixture listener")
	}
	serverHTTP := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	serveDone := make(chan error, 1)
	go func() { serveDone <- serverHTTP.Serve(listener) }()
	identityURL := "http://" + listener.Addr().String()
	if !strings.HasPrefix(identityURL, "http://127.0.0.1:") {
		_ = serverHTTP.Close()
		return errorsNoDetails("fixture listener was not loopback")
	}
	identityPayload := map[string]string{
		"url": identityURL, "fixture_id": identity.FixtureID,
		"adapter": identity.AdapterID, "book": identity.BookID,
		"service_order_revenue": identity.RevenueAccountID,
	}
	if err := json.NewEncoder(os.Stdout).Encode(identityPayload); err != nil {
		_ = serverHTTP.Close()
		return errorsNoDetails("write fixture endpoint and scope")
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-signals:
	case <-stop:
	case err := <-serveDone:
		if err != nil && err != http.ErrServerClosed {
			return errorsNoDetails("serve isolated ledger fixture")
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return serverHTTP.Shutdown(shutdownCtx)
}

func errorsNoDetails(operation string) error {
	return fmt.Errorf("%s", operation)
}
