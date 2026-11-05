package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	ingestpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ingest"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	_ "modernc.org/sqlite"
	ingestH "money-ledger/handlers/ingest"
	ledgerH "money-ledger/handlers/ledger"
	"money-ledger/internal/ingest"
	"money-ledger/internal/ledger"
	"money-ledger/internal/modules"
	"money-ledger/internal/server"
)

// Real Money Ledger modules and database; the parent owns the isolated test
// path and stdin lifetime. No financial service is replaced by a fake.
func TestJourneyFundingLedgerServer(t *testing.T) {
	path := os.Getenv("VROOLI_JOURNEY_LEDGER_DB")
	if path == "" {
		t.Skip("journey subprocess only")
	}
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	db := database.NewFromPrimary(raw)
	ctx := context.Background()
	if err := database.EnsureSchemas(ctx, raw, modules.AllSchemas()...); err != nil {
		t.Fatal(err)
	}
	journal := ledger.NewStore(db, nil)
	if err := journal.EnsureMigrations(ctx); err != nil {
		t.Fatal(err)
	}
	book, err := journal.CreateBook(ctx, "Isolated funding", "USD")
	if err != nil {
		t.Fatal(err)
	}
	config := map[string]string{"book": book.Id, "adapter": "journey-lpbs"}
	for _, kind := range []string{"loss", "refund", "fee"} {
		account, err := journal.CreateAccount(ctx, book.Id, kind, ledgerpb.AccountKind_EXPENSE)
		if err != nil {
			t.Fatal(err)
		}
		config[kind] = account.Id
	}
	if _, err := ingest.NewStore(db, nil).RegisterAdapter(ctx, &ingestpb.Adapter{Id: config["adapter"], Name: "LPBS journey", Kind: ingestpb.AdapterKind_ADAPTER_KIND_FILE, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	logger := log.New(io.Discard, "", 0)
	srv := server.New(server.Deps{Clock: schedule.System(), Logger: logger}, ledgerH.Module(db, nil, logger), ingestH.Module(db, nil, logger))
	stop := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/__journey/stop", func(w http.ResponseWriter, r *http.Request) { close(stop) })
	mux.Handle("/", srv.Handler())
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()
	config["url"] = httpServer.URL
	payload, _ := json.Marshal(config)
	fmt.Printf("LEDGER_FIXTURE=%s\n", payload)
	<-stop
}
