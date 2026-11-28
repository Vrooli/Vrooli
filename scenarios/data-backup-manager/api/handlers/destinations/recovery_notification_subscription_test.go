package destinations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecoverySubscriptionDoesNotReplaceUnrelatedSameTarget(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		switch r.Method {
		case http.MethodGet:
			if r.URL.Query().Get("owner") != "data-backup-manager" {
				t.Errorf("owner filter = %q", r.URL.Query().Get("owner"))
			}
			_, _ = w.Write([]byte(`[{"id":7,"name":"other-integration","owner_scenario":"data-backup-manager","event_pattern":"other.event","delivery_type":"webhook","delivery_target":"http://127.0.0.1/hook","enabled":true}]`))
		case http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Errorf("decode create payload: %v", err)
			}
			if payload["name"] != "data-backup-manager-volume-recovery-decisions" {
				t.Errorf("create payload = %#v", payload)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":8}`))
		case http.MethodPut, http.MethodDelete:
			t.Errorf("unrelated subscription was mutated with %s", r.Method)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	if err := reconcileRecoveryDecisionSubscriptionAt(context.Background(), server.URL, "http://127.0.0.1/hook"); err != nil {
		t.Fatal(err)
	}
	if len(methods) != 2 || methods[0] != http.MethodGet || methods[1] != http.MethodPost {
		t.Fatalf("methods = %v, want GET then POST", methods)
	}
}
