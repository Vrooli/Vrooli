package journal

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "source-ledger/internal/database"
	internaljournal "source-ledger/internal/journal"
	"source-ledger/internal/testutil/mocks"

	journalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	journalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
)

type durableJournalRoute struct {
	server *httptest.Server
	db     *apidb.RoutedDB
	client journalconnect.JournalServiceClient
}

func openJournalTestDB(t testing.TB, dsn string) *apidb.RoutedDB {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	if err := apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(internaljournal.Schema)); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close journal test database: %v", err)
		}
	})
	return db
}

func openDurableJournalRoute(t *testing.T, dsn string, inference *mocks.FakeInference) durableJournalRoute {
	t.Helper()
	db := openJournalTestDB(t, dsn)
	service := internaljournal.NewService(internaljournal.NewSQLiteRepository(db.Primary()), inference)
	_, handler := journalconnect.NewJournalServiceHandler(NewConnectHandler(service, nil))
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return durableJournalRoute{server: server, db: db, client: journalconnect.NewJournalServiceClient(server.Client(), server.URL)}
}

func TestWalkBriefingSurvivesInferenceLossAndJournalRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "source-ledger.sqlite")
	inference := &mocks.FakeInference{
		ClassifyOut: "episode",
		EmbedErr:    errors.New("embedding provider unavailable during briefing save"),
	}
	route := openDurableJournalRoute(t, dbPath, inference)
	request := walkBriefingAppendRequest(t)
	ctx := context.Background()

	first, err := route.client.AppendEntry(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	entryID := first.Msg.GetEntry().GetId()
	if entryID == "" || first.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("initial durable append = %+v", first.Msg.GetEntry())
	}
	if first.Msg.GetEntry().GetFacetId() != "episode" {
		t.Fatalf("provider outage facet = %q, want classified entry with deferred embedding", first.Msg.GetEntry().GetFacetId())
	}
	deferred, err := route.client.ProcessEmbeddingRetries(ctx, connect.NewRequest(&journalv1.ProcessEmbeddingRetriesRequest{Scope: request.GetScope(), Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if deferred.Msg.GetProcessed() != 0 || deferred.Msg.GetDeferred() != 1 {
		t.Fatalf("retry pass during provider loss = %+v", deferred.Msg)
	}
	duringLoss, err := route.client.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: request.GetScope(), RequestKey: request.GetRequestKey()}))
	if err != nil {
		t.Fatal(err)
	}
	if duringLoss.Msg.GetEntry().GetId() != entryID || duringLoss.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("provider loss changed durable briefing: %+v", duringLoss.Msg.GetEntry())
	}

	// Close both the HTTP surface and the SQLite pools, then reconstruct the
	// production Connect handler against the same on-disk database path.
	route.server.Close()
	if err := route.db.Close(); err != nil {
		t.Fatal(err)
	}
	inference.EmbedErr = nil
	inference.EmbedOut = []float64{0.125, 0.5}
	route = openDurableJournalRoute(t, dbPath, inference)

	read, err := route.client.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: request.GetScope(), RequestKey: request.GetRequestKey()}))
	if err != nil {
		t.Fatal(err)
	}
	if read.Msg.GetEntry().GetId() != entryID || read.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("reopened entry differs from original: first=%+v reopened=%+v", first.Msg.GetEntry(), read.Msg.GetEntry())
	}

	retried, err := route.client.AppendEntry(ctx, connect.NewRequest(request))
	if err != nil {
		t.Fatal(err)
	}
	if !retried.Msg.GetExisting() || retried.Msg.GetEntry().GetId() != entryID || retried.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("same request key did not recover the original briefing: %+v", retried.Msg)
	}

	processed, err := route.client.ProcessEmbeddingRetries(ctx, connect.NewRequest(&journalv1.ProcessEmbeddingRetriesRequest{Scope: request.GetScope(), Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if processed.Msg.GetProcessed() != 1 {
		t.Fatalf("embedding retry after provider recovery = %+v", processed.Msg)
	}
	final, err := route.client.GetEntry(ctx, connect.NewRequest(&journalv1.GetEntryRequest{Scope: request.GetScope(), RequestKey: request.GetRequestKey()}))
	if err != nil {
		t.Fatal(err)
	}
	if final.Msg.GetEntry().GetId() != entryID || final.Msg.GetEntry().GetBody() != request.GetBody() {
		t.Fatalf("derived-data recovery changed immutable user work: %+v", final.Msg.GetEntry())
	}
	listed, err := route.client.ListEntries(ctx, connect.NewRequest(&journalv1.ListEntriesRequest{Scope: request.GetScope(), Kind: request.GetKind(), NewestFirst: true, Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetEntries()) != 1 || listed.Msg.GetEntries()[0].GetId() != entryID {
		t.Fatalf("durable replay created duplicate briefing rows: %+v", listed.Msg.GetEntries())
	}
}

func walkBriefingAppendRequest(t *testing.T) *journalv1.AppendEntryRequest {
	t.Helper()
	phases := []string{"1", "2", "3", "4", "5", "5.3", "5.5", "5.7", "5.9", "6", "7", "8", "9"}
	phaseRows := make([]map[string]string, 0, len(phases))
	for _, phase := range phases {
		phaseRows = append(phaseRows, map[string]string{"phase": phase})
	}
	envelope, err := json.Marshal(map[string]any{
		"program": "command-center.vision-walk-prep", "status": "partial",
		"signals": map[string]any{"generated_at": time.Now().UTC().Format(time.RFC3339Nano), "phases": phaseRows, "checkpoint": map[string]string{"status": "none"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	briefing := "Saved walk: investigate the throughput change and resume at phase 5.3."
	body, err := json.Marshal(map[string]any{
		"schema_version": 1, "program_id": "prog_cross_product", "envelope": json.RawMessage(envelope),
		"briefing": briefing, "fleet_health": map[string]string{"status": "unavailable", "reason": "provider transition test"}, "expected_previous_id": "",
	})
	if err != nil {
		t.Fatal(err)
	}
	previous := ""
	return &journalv1.AppendEntryRequest{
		Scope: "team:director-swarm", Kind: "vision-walk-briefing-test", Body: string(body),
		RequestKey: "command-center/vision-walk-briefing-test/durable-briefing", ExpectedLatestId: &previous,
	}
}
