package ledger

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"content-desk/internal/artifacts"
	"content-desk/internal/claims"
	localdb "content-desk/internal/database"
	internalledger "content-desk/internal/ledger"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	ledgerv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/ledger/ledger_v1connect"
	_ "modernc.org/sqlite"
)

// [REQ:CHANMGR-P1-007] The generated Content Desk ingress accepts one metric
// sample identity exactly once, proving the replay-safe handoff contract.
func TestIngestMetricSampleOverConnectIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:ledger-connect?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(localdb.SystemSchema), database.SchemaProviderFunc(artifacts.Schema), database.SchemaProviderFunc(claims.Schema), database.SchemaProviderFunc(internalledger.Schema)); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	Module(database.NewFromPrimary(db)).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := ledgerconnect.NewLedgerServiceClient(http.DefaultClient, server.URL)
	req := &ledgerv1.IngestMetricSampleRequest{SampleId: "sample-connect", ReleaseId: "release-connect", DraftId: "draft-connect", Metric: "impressions", Value: 55, ObservedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	first, err := client.IngestMetricSample(t.Context(), connect.NewRequest(req))
	if err != nil || !first.Msg.Accepted {
		t.Fatalf("first=%v err=%v", first, err)
	}
	second, err := client.IngestMetricSample(t.Context(), connect.NewRequest(req))
	if err != nil || second.Msg.SampleId != first.Msg.SampleId {
		t.Fatalf("second=%v err=%v", second, err)
	}
}
