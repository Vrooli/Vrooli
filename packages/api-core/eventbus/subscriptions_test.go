package eventbus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestSubscriptionClientPerformsTypedCRUDAndDeliveryReconciliation(t *testing.T) {
	var requests []struct {
		Method string
		Path   string
		Query  string
		Body   map[string]any
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request := struct {
			Method string
			Path   string
			Query  string
			Body   map[string]any
		}{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&request.Body)
		}
		requests = append(requests, request)
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":7,"name":"answers","owner_scenario":"caller","event_pattern":"ask.resolved.v1","delivery_type":"webhook","delivery_target":"http://localhost/hook","enabled":true}]`))
		case http.MethodPost:
			if r.URL.Path == "/prefix/api/v1/subscriptions" {
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":9}`))
			} else {
				w.WriteHeader(http.StatusAccepted)
			}
		case http.MethodPut:
			_, _ = w.Write([]byte(`{"status":"updated"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client, err := NewSubscriptionClient(server.URL+"/prefix/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	subscriptions, err := client.ListSubscriptions(ctx, "caller/team")
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 || subscriptions[0].ID != 7 || subscriptions[0].EventPattern != "ask.resolved.v1" {
		t.Fatalf("subscriptions = %+v", subscriptions)
	}
	subscription := Subscription{
		Name: "answers", OwnerScenario: "caller", EventPattern: "ask.resolved.v1",
		DeliveryType: "webhook", DeliveryTarget: "http://localhost/hook", Enabled: true,
	}
	createdID, err := client.CreateSubscription(ctx, subscription)
	if err != nil || createdID != 9 {
		t.Fatalf("CreateSubscription() = %d, %v", createdID, err)
	}
	if err := client.UpdateSubscription(ctx, 7, subscription); err != nil {
		t.Fatal(err)
	}
	if err := client.ReconcileDelivery(ctx, 7, "event-1"); err != nil {
		t.Fatal(err)
	}
	if err := client.DeleteSubscription(ctx, 7); err != nil {
		t.Fatal(err)
	}
	wantMethods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPost, http.MethodDelete}
	gotMethods := make([]string, len(requests))
	for i, request := range requests {
		gotMethods[i] = request.Method
		if request.Path == "" || request.Path[:len("/prefix")] != "/prefix" {
			t.Errorf("request %d path = %q, expected deployment base path to be retained", i, request.Path)
		}
	}
	if !reflect.DeepEqual(gotMethods, wantMethods) {
		t.Fatalf("request methods = %v, want %v", gotMethods, wantMethods)
	}
	if requests[0].Query != "owner=caller%2Fteam" {
		t.Errorf("owner query = %q, want escaped path segment", requests[0].Query)
	}
	if requests[1].Body["owner_scenario"] != "caller" || requests[1].Body["enabled"] != true {
		t.Errorf("create payload = %#v", requests[1].Body)
	}
	if _, found := requests[1].Body["id"]; found {
		t.Errorf("create payload must not send server-owned id: %#v", requests[1].Body)
	}
	if requests[3].Body["event_id"] != "event-1" {
		t.Errorf("delivery reconciliation payload = %#v", requests[3].Body)
	}
}

func TestEnsureNamedSubscriptionRepairsProjectionAndDisablesDuplicates(t *testing.T) {
	rows := []Subscription{
		{ID: 1, Name: "unrelated", OwnerScenario: "caller", EventPattern: "other.event", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true},
		{ID: 8, Name: "answers", OwnerScenario: "caller", EventPattern: "ask.resolved.v1", SourceFilter: "wrong-source", DeliveryType: "webhook", DeliveryTarget: "http://old", Enabled: true},
		{ID: 3, Name: "answers", OwnerScenario: "caller", EventPattern: "old.event", DeliveryType: "webhook", DeliveryTarget: "http://old", Enabled: true},
	}
	var updatedIDs []int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("owner") != "caller" {
				t.Errorf("owner query = %q", r.URL.Query().Get("owner"))
			}
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPut:
			var replacement Subscription
			if err := json.NewDecoder(r.Body).Decode(&replacement); err != nil {
				t.Errorf("decode replacement: %v", err)
			}
			var id int64
			if _, err := fmt.Sscanf(r.URL.Path, "/api/v1/subscriptions/%d", &id); err != nil {
				t.Errorf("parse update path %q: %v", r.URL.Path, err)
			}
			updatedIDs = append(updatedIDs, id)
			replacement.ID = id
			for i := range rows {
				if rows[i].ID == id {
					rows[i] = replacement
				}
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected %s", r.Method)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := NewSubscriptionClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	desired := Subscription{Name: "answers", OwnerScenario: "caller", EventPattern: "ask.resolved.v1", DeliveryType: "webhook", DeliveryTarget: "http://new", Enabled: true}
	if err := client.EnsureNamedSubscription(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(updatedIDs, []int64{8, 3}) {
		t.Fatalf("updated row IDs = %v, want duplicate 8 disabled and lowest-ID row 3 repaired", updatedIDs)
	}
	if rows[0].Name != "unrelated" || !rows[0].Enabled || rows[0].DeliveryTarget != "http://hook" {
		t.Fatalf("unrelated same-target row changed: %+v", rows[0])
	}
	if rows[1].Enabled {
		t.Fatalf("duplicate remains enabled: %+v", rows[1])
	}
	if rows[2].SourceFilter != "" || rows[2].EventPattern != desired.EventPattern || rows[2].DeliveryTarget != desired.DeliveryTarget || !rows[2].Enabled {
		t.Fatalf("canonical row does not match full desired projection: %+v", rows[2])
	}
}

func TestEnsureNamedSubscriptionCreatesInsteadOfReusingSameTarget(t *testing.T) {
	var rows = []Subscription{{ID: 5, Name: "other", OwnerScenario: "caller", EventPattern: "ask.resolved.v1", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true}}
	created := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPost:
			var desired Subscription
			if err := json.NewDecoder(r.Body).Decode(&desired); err != nil {
				t.Errorf("decode create: %v", err)
			}
			if desired.Name != "answers" {
				t.Errorf("created wrong row: %+v", desired)
			}
			created = true
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":6}`))
		default:
			t.Errorf("unexpected %s", r.Method)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := NewSubscriptionClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	if err := client.EnsureNamedSubscription(context.Background(), Subscription{Name: "answers", OwnerScenario: "caller", EventPattern: "ask.resolved.v1", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if !created {
		t.Fatal("missing named subscription was not created")
	}
}

func TestEnsureNamedSubscriptionAdoptsExactUnnamedLegacyRow(t *testing.T) {
	var updated Subscription
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`[{"id":5,"name":"","owner_scenario":"caller","event_pattern":"ask.resolved.v1","source_filter":"","delivery_type":"webhook","delivery_target":"http://hook","enabled":true}]`))
		case http.MethodPut:
			if r.URL.Path != "/api/v1/subscriptions/5" {
				t.Errorf("update path = %q", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
				t.Errorf("decode legacy adoption: %v", err)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected %s", r.Method)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := NewSubscriptionClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	desired := Subscription{Name: "answers", OwnerScenario: "caller", EventPattern: "ask.resolved.v1", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true}
	if err := client.EnsureNamedSubscription(context.Background(), desired); err != nil {
		t.Fatal(err)
	}
	if updated.Name != desired.Name || updated.OwnerScenario != desired.OwnerScenario {
		t.Fatalf("legacy row was not adopted: %+v", updated)
	}
}

func TestEnsureNamedSubscriptionsUsesOneInventoryAndRetriesPartialWrites(t *testing.T) {
	rows := []Subscription{}
	var gets, posts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gets++
			_ = json.NewEncoder(w).Encode(rows)
		case http.MethodPost:
			posts++
			var created Subscription
			if err := json.NewDecoder(r.Body).Decode(&created); err != nil {
				t.Errorf("decode create: %v", err)
			}
			if posts == 2 {
				http.Error(w, `{"code":"temporary","error":"try again"}`, http.StatusServiceUnavailable)
				return
			}
			created.ID = int64(10 + posts)
			rows = append(rows, created)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":%d}`, created.ID)))
		default:
			t.Errorf("unexpected %s", r.Method)
			http.Error(w, "unexpected", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()
	client, err := NewSubscriptionClient(server.URL, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	desired := []Subscription{
		{Name: "first", OwnerScenario: "caller", EventPattern: "one.event", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true},
		{Name: "second", OwnerScenario: "caller", EventPattern: "two.event", DeliveryType: "webhook", DeliveryTarget: "http://hook", Enabled: true},
	}
	if inventory, err := client.EnsureNamedSubscriptions(context.Background(), desired); err == nil || inventory != nil {
		t.Fatalf("partial reconciliation = (%+v, %v), want an error after a durable first write", inventory, err)
	}
	if len(rows) != 1 || rows[0].Name != "first" || gets != 1 {
		t.Fatalf("first attempt state = rows:%+v gets:%d, want one durable first row and one inventory read", rows, gets)
	}
	inventory, err := client.EnsureNamedSubscriptions(context.Background(), desired)
	if err != nil {
		t.Fatal(err)
	}
	if len(inventory) != 2 || len(rows) != 2 || gets != 2 || posts != 3 {
		t.Fatalf("retry state = inventory:%+v rows:%+v gets:%d posts:%d, want convergence with one read per attempt", inventory, rows, gets, posts)
	}
}

func TestNewSubscriptionClientRejectsNonServiceBases(t *testing.T) {
	for _, base := range []string{"", "localhost:1234", "ftp://events.example", "http://user:secret@events.example", "http://events.example?owner=other", "http://events.example#fragment"} {
		if client, err := NewSubscriptionClient(base, nil); !errors.Is(err, ErrInvalidSubscriptionsURL) || client != nil {
			t.Errorf("NewSubscriptionClient(%q) = %v, %v; want invalid URL", base, client, err)
		}
	}
}

func TestSubscriptionClientReportsHTTPFailuresAndRejectsInvalidOperations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"code":"SUBSCRIPTION_READ_ERROR","error":"Events are restarting"}`))
	}))
	defer server.Close()
	client, err := NewSubscriptionClient(server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListSubscriptions(context.Background(), "owner")
	var statusErr *SubscriptionHTTPError
	if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusServiceUnavailable || statusErr.Code != "SUBSCRIPTION_READ_ERROR" || statusErr.Message != "Events are restarting" {
		t.Fatalf("ListSubscriptions error = %v, want typed 503 with Events error details", err)
	}
	if err := client.UpdateSubscription(context.Background(), 0, Subscription{}); err == nil {
		t.Fatal("UpdateSubscription accepted non-positive id")
	}
	if err := client.DeleteSubscription(context.Background(), 0); err == nil {
		t.Fatal("DeleteSubscription accepted non-positive id")
	}
	if err := client.ReconcileDelivery(context.Background(), 1, " event "); err == nil {
		t.Fatal("ReconcileDelivery accepted an untrimmed event identity")
	}
}
