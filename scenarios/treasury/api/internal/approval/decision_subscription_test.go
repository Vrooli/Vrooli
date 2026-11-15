package approval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReconcileDecisionEventSubscriptionCreatesAndUpdatesOnlyItsNamedRoute(t *testing.T) {
	type subscription struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		Owner          string `json:"owner_scenario"`
		EventPattern   string `json:"event_pattern"`
		DeliveryType   string `json:"delivery_type"`
		DeliveryTarget string `json:"delivery_target"`
		Enabled        bool   `json:"enabled"`
	}
	var rows []subscription
	creates, updates := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("owner") != "treasury" && r.Method == http.MethodGet {
			t.Errorf("owner query=%q, want treasury", r.URL.Query().Get("owner"))
		}
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPost:
			creates++
			var got subscription
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			got.ID = 41
			rows = append(rows, got)
			w.WriteHeader(http.StatusCreated)
		case http.MethodPut:
			updates++
			if !strings.HasSuffix(r.URL.Path, "/41") {
				t.Errorf("update route=%q, want subscription 41", r.URL.Path)
			}
			var got subscription
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Error(err)
			}
			got.ID = 41
			rows[0] = got
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	firstTarget := "http://127.0.0.1:4100" + DecisionWebhookPath
	if err := ReconcileDecisionEventSubscription(context.Background(), server.URL, firstTarget); err != nil {
		t.Fatal(err)
	}
	if creates != 1 || len(rows) != 1 {
		t.Fatalf("create count=%d rows=%+v", creates, rows)
	}
	want := subscription{ID: 41, Name: DecisionAnswerSubscriptionName, Owner: "treasury", EventPattern: DecisionAnswerEventPattern, DeliveryType: "webhook", DeliveryTarget: firstTarget, Enabled: true}
	if rows[0] != want {
		t.Fatalf("created subscription=%+v, want %+v", rows[0], want)
	}
	if err := ReconcileDecisionEventSubscription(context.Background(), server.URL, firstTarget); err != nil {
		t.Fatalf("idempotent reconcile: %v", err)
	}
	if creates != 1 || updates != 0 {
		t.Fatalf("exact repeat created or updated subscription: creates=%d updates=%d", creates, updates)
	}
	secondTarget := "http://127.0.0.1:4200" + DecisionWebhookPath
	if err := ReconcileDecisionEventSubscription(context.Background(), server.URL, secondTarget); err != nil {
		t.Fatalf("target migration: %v", err)
	}
	want.DeliveryTarget = secondTarget
	if rows[0] != want || updates != 1 || creates != 1 {
		t.Fatalf("route update lost exact contract: row=%+v want=%+v creates=%d updates=%d", rows[0], want, creates, updates)
	}
}
