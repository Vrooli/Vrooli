package aidoor

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestAC2CreditModelCatalogTracksGatewayAvailabilityWithoutCaching(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE business_account_members (
		business_account_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL,
		PRIMARY KEY (business_account_id, user_id)
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO business_account_members(business_account_id,user_id,role) VALUES('acct-fixture','actor-fixture','owner')`); err != nil {
		t.Fatal(err)
	}
	keys := KeyStore{DB: db}
	key, err := keys.IssueDevice(context.Background(), "acct-fixture", "actor-fixture", "web-console", 100)
	if err != nil {
		t.Fatal(err)
	}

	gatewayCalls := 0
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayCalls++
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/private/provider-routes/models" || r.URL.Query().Get("harness") != "opencode" {
			t.Errorf("Gateway request = %s %s?%s; want private model list for opencode", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if r.URL.Query().Has("key") {
			t.Error("customer key leaked into Gateway query")
		}
		if gatewayCalls == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"data":[{"id":"eligible-fixture-model","name":"Eligible fixture model"}],"reason":""}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"data":[],"reason":"no_eligible_route"}`)
	}))
	defer gateway.Close()

	door := &Door{Keys: keys, GatewayURL: gateway.URL}
	requestCatalog := func() *httptest.ResponseRecorder {
		t.Helper()
		request := httptest.NewRequest(http.MethodGet, "/api/v1/ai/models?harness=opencode", nil)
		request.Header.Set("Authorization", "Bearer "+key.Token)
		response := httptest.NewRecorder()
		door.ServeHTTP(response, request)
		return response
	}
	first := requestCatalog()
	if first.Code != http.StatusOK || first.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("first model response status/cache-control = %d/%q; want 200/no-store", first.Code, first.Header().Get("Cache-Control"))
	}
	var firstBody struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &firstBody); err != nil {
		t.Fatal(err)
	}
	if len(firstBody.Data) != 1 || firstBody.Data[0].ID != "eligible-fixture-model" {
		t.Fatalf("first routed catalog = %+v; want the current eligible model", firstBody)
	}

	second := requestCatalog()
	if second.Code != http.StatusOK || second.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("withdrawn model response status/cache-control = %d/%q; want 200/no-store", second.Code, second.Header().Get("Cache-Control"))
	}
	var secondBody struct {
		Data   []json.RawMessage `json:"data"`
		Reason string            `json:"reason"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &secondBody); err != nil {
		t.Fatal(err)
	}
	if len(secondBody.Data) != 0 || !strings.EqualFold(secondBody.Reason, "no_eligible_route") {
		t.Fatalf("withdrawn model remained in customer catalog: %+v", secondBody)
	}
	if gatewayCalls != 2 {
		t.Fatalf("Gateway calls = %d; want a fresh route snapshot per catalog request", gatewayCalls)
	}
}
