package approval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/vrooli/api-core/eventbus"
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
			_ = json.NewEncoder(w).Encode(map[string]int64{"id": got.ID})
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

func TestReconcileDecisionEventSubscriptionLeavesForeignNamedRowsUntouched(t *testing.T) {
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]eventbus.Subscription{{ID: 1, Name: DecisionAnswerSubscriptionName, OwnerScenario: "another-scenario", EventPattern: DecisionAnswerEventPattern, DeliveryType: "webhook", DeliveryTarget: "http://foreign/hook", Enabled: true}})
		case http.MethodPost:
			var createdSubscription eventbus.Subscription
			if err := json.NewDecoder(r.Body).Decode(&createdSubscription); err != nil {
				t.Errorf("decode created subscription: %v", err)
			}
			if createdSubscription.OwnerScenario != "treasury" || createdSubscription.Name != DecisionAnswerSubscriptionName {
				t.Errorf("created subscription = %+v", createdSubscription)
			}
			created = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":2}`))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	if err := ReconcileDecisionEventSubscription(context.Background(), server.URL, "http://127.0.0.1"+DecisionWebhookPath); err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("Treasury route was not created beside the foreign row")
	}
}

func TestReconcileDecisionEventSubscriptionDisablesOwnedDuplicates(t *testing.T) {
	rows := []eventbus.Subscription{
		{ID: 1, Name: DecisionAnswerSubscriptionName, OwnerScenario: "treasury", EventPattern: "old.event", DeliveryType: "webhook", DeliveryTarget: "http://old/hook", Enabled: true},
		{ID: 2, Name: DecisionAnswerSubscriptionName, OwnerScenario: "treasury", EventPattern: DecisionAnswerEventPattern, DeliveryType: "webhook", DeliveryTarget: "http://new/hook", Enabled: true},
	}
	updated := map[int64]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPut:
			id, err := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/"), 10, 64)
			if err != nil {
				t.Errorf("parse update path %q: %v", r.URL.Path, err)
			}
			var replacement eventbus.Subscription
			if err := json.NewDecoder(r.Body).Decode(&replacement); err != nil {
				t.Errorf("decode replacement: %v", err)
			}
			replacement.ID = id
			updated[id] = replacement.Enabled
			for i := range rows {
				if rows[i].ID == id {
					rows[i] = replacement
				}
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	target := "http://new/hook"
	if err := ReconcileDecisionEventSubscription(context.Background(), server.URL, target); err != nil {
		t.Fatal(err)
	}
	if updated[2] || !updated[1] || rows[0].EventPattern != DecisionAnswerEventPattern || rows[0].DeliveryTarget != target || rows[0].Name != DecisionAnswerSubscriptionName || rows[1].Enabled {
		t.Fatalf("reconciled rows=%+v updates=%+v; want canonical id 1 repaired and duplicate id 2 disabled", rows, updated)
	}
}
