//go:build integration

package bridgeadmin

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/vrooli/api-core/nodereach"
	"vrooli-bridge/integrationfixture"
)

const integrationCaller = "admin@example.test"

func TestPostgresReceiptAndBridgeDispatchSurviveAmbiguousRestart(t *testing.T) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:16-alpine", postgres.WithDatabase("bridgeadmin"), postgres.WithUsername("bridgeadmin"), postgres.WithPassword("isolated-integration-password"))
	if err != nil {
		t.Fatalf("start private Postgres container: %v", err)
	}
	t.Cleanup(func() { _ = container.Terminate(ctx) })
	conn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("postgres", conn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	deadline := time.Now().Add(30 * time.Second)
	for db.PingContext(ctx) != nil && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("private Postgres unavailable: %v", err)
	}
	if _, err := db.ExecContext(ctx, Schema()); err != nil {
		t.Fatalf("apply LPBS receipt schema: %v", err)
	}

	effects := &atomic.Int32{}
	bridgeDB := filepath.Join(t.TempDir(), "bridge.sqlite")
	bridge := integrationfixture.Start(t, bridgeDB, effects)
	drop := &dropResponseOnce{base: http.DefaultTransport}
	bridgeClient := nodereach.New(nodereach.Config{BridgeURL: bridge.Server.URL, Token: "integration-only", HTTPClient: &http.Client{Transport: drop}})
	store := NewSQLStore(db)
	service := New(bridgeClient, store, nil)
	api := adminAPI(service)

	request := Request{RequestID: "ambiguous-1", Scenario: "vrooli-bridge", Operation: "scenario status", NodeID: bridge.NodeID, Tier: "vps"}
	status, first := submitAPI(t, api, request)
	if status != http.StatusBadGateway || first.Status != "uncertain" {
		t.Fatalf("first outcome status=%d receipt=%+v; want durable uncertain", status, first)
	}
	if effects.Load() != 1 {
		t.Fatalf("target effect count after dropped response=%d, want 1", effects.Load())
	}

	// Reopen both owner services over their durable stores. The first response
	// was lost only after the Bridge handler pushed one accepted effect.
	bridge.Server.Close()
	_ = bridge.DB.Close()
	bridge = integrationfixture.Start(t, bridgeDB, effects)
	bridgeClient = nodereach.New(nodereach.Config{BridgeURL: bridge.Server.URL, Token: "integration-only"})
	service = New(bridgeClient, NewSQLStore(db), nil)
	api = adminAPI(service)
	status, replay := submitAPI(t, api, request)
	if status != http.StatusOK || replay.Status != "replayed" || !replay.Replayed || replay.RunID == "" {
		t.Fatalf("same-key retry status=%d receipt=%+v", status, replay)
	}
	if effects.Load() != 1 {
		t.Fatalf("same-key replay duplicated target effect: %d", effects.Load())
	}

	// Caller-visible receipt query returns the persisted replay state after the
	// service reconstruction.
	queried := httptest.NewRecorder()
	query := httptest.NewRequest(http.MethodGet, "/api/v1/admin/bridge/dispatch/ambiguous-1", nil)
	muxVars := mux.SetURLVars(query, map[string]string{"request_id": "ambiguous-1"})
	api.ServeHTTP(queried, muxVars)
	var queryReceipt Receipt
	if err := json.Unmarshal(queried.Body.Bytes(), &queryReceipt); err != nil || queryReceipt.Status != "replayed" || queryReceipt.RunID != replay.RunID {
		t.Fatalf("durable receipt query body=%s err=%v", queried.Body.String(), err)
	}

	// Reusing the caller key for another scenario conflicts in PostgreSQL and
	// never reaches Bridge. A distinct key for an undeclared operation is
	// refused by Bridge's real manifest policy before a target push.
	conflict := request
	conflict.Scenario = "some-other-scenario"
	status, _ = submitAPI(t, api, conflict)
	if status != http.StatusBadRequest {
		t.Fatalf("conflicting target status=%d, want 400", status)
	}
	denied := request
	denied.RequestID = "denied-scope"
	denied.Operation = "scenario deploy"
	status, refusal := submitAPI(t, api, denied)
	if status != http.StatusForbidden || refusal.Status != "refused" {
		t.Fatalf("Bridge policy refusal status=%d receipt=%+v", status, refusal)
	}
	if effects.Load() != 1 {
		t.Fatalf("refusal/conflict caused effect; count=%d", effects.Load())
	}

	// The older shared nodereach callers omit tier. Bridge keeps that additive
	// wire field optional while LPBS admin binds its selected tier explicitly.
	legacy, err := bridgeClient.Dispatch(ctx, nodereach.DispatchRequest{NodeID: bridge.NodeID, Scenario: "vrooli-bridge", Verb: "scenario status", RequestID: "legacy-no-tier"})
	if err != nil || legacy.RunID == "" || effects.Load() != 2 {
		t.Fatalf("legacy tier-omitted dispatch=%+v err=%v effects=%d", legacy, err, effects.Load())
	}
}

func adminAPI(service *Service) http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/admin/bridge/dispatch", Handler(service, func(*http.Request) (string, bool) { return integrationCaller, true })).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/admin/bridge/dispatch/{request_id}", ReceiptHandler(service, func(*http.Request) (string, bool) { return integrationCaller, true })).Methods(http.MethodGet)
	return r
}

func submitAPI(t *testing.T, h http.Handler, request Request) (int, Receipt) {
	t.Helper()
	data, _ := json.Marshal(request)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/admin/bridge/dispatch", strings.NewReader(string(data))))
	var receipt Receipt
	if response.Code == http.StatusOK {
		if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
	} else {
		var body struct {
			Receipt Receipt `json:"receipt"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode status %d response %q: %v", response.Code, response.Body.String(), err)
		}
		receipt = body.Receipt
	}
	return response.Code, receipt
}

type dropResponseOnce struct {
	base    http.RoundTripper
	dropped atomic.Bool
}

func (d *dropResponseOnce) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := d.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if strings.Contains(req.URL.Path, "DispatchJob") && d.dropped.CompareAndSwap(false, true) {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return nil, errors.New("simulated lost response after Bridge accepted effect")
	}
	return resp, nil
}
