package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestReconcileDecisionEventSubscriptionAtPreservesOwnerPolicy(t *testing.T) {
	tests := []struct {
		name       string
		existing   string
		wantMethod []string
		wantUpdate bool
	}{
		{
			name:       "matching owned route is left alone",
			existing:   `[{"id":3,"name":"swarm-manager-decision-answers","owner_scenario":"swarm-manager","event_pattern":"` + askResolvedEventPattern + `","delivery_type":"webhook","delivery_target":"http://127.0.0.1/hook","enabled":true}]`,
			wantMethod: []string{http.MethodGet},
		},
		{
			name:       "owned route is repaired after event pattern changes",
			existing:   `[{"id":5,"name":"swarm-manager-decision-answers","owner_scenario":"swarm-manager","event_pattern":"old.event","delivery_type":"webhook","delivery_target":"http://127.0.0.1/old","enabled":true}]`,
			wantMethod: []string{http.MethodGet, http.MethodPut}, wantUpdate: true,
		},
		{
			name:       "unrelated same-target route is preserved",
			existing:   `[{"id":5,"name":"other-integration","owner_scenario":"swarm-manager","event_pattern":"old.event","delivery_type":"webhook","delivery_target":"http://127.0.0.1/hook","enabled":true}]`,
			wantMethod: []string{http.MethodGet, http.MethodPost},
		},
		{
			name:       "missing route is created",
			existing:   `[]`,
			wantMethod: []string{http.MethodGet, http.MethodPost},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				methods = append(methods, r.Method)
				if r.Method == http.MethodGet {
					if r.URL.Query().Get("owner") != "swarm-manager" {
						t.Errorf("owner filter = %q", r.URL.Query().Get("owner"))
					}
					_, _ = w.Write([]byte(test.existing))
					return
				}
				var payload map[string]any
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode %s body: %v", r.Method, err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if payload["name"] != "swarm-manager-decision-answers" || payload["event_pattern"] != askResolvedEventPattern || payload["delivery_target"] != "http://127.0.0.1/hook" {
					t.Errorf("%s payload = %#v", r.Method, payload)
				}
				if r.Method == http.MethodPost {
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":10}`))
				}
			}))
			t.Cleanup(server.Close)
			if err := reconcileDecisionEventSubscriptionAt(context.Background(), server.URL, "http://127.0.0.1/hook"); err != nil {
				t.Fatalf("reconcile subscription: %v", err)
			}
			if !reflect.DeepEqual(methods, test.wantMethod) {
				t.Errorf("methods = %v, want %v", methods, test.wantMethod)
			}
			if test.wantUpdate && len(methods) == 2 && methods[1] != http.MethodPut {
				t.Errorf("same-target replacement method = %q, want PUT", methods[1])
			}
		})
	}
}
