package main

import (
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Exercise the actual CLI RunService and shared HTTP transport, with fixture
// proofs only. Receiving authentication is covered separately by API tests.
func TestAuth01RunCreateTransportPreservesCallerAndRefusal(t *testing.T) {
	for _, kind := range []string{"run", "human", "both", "absent-refused"} {
		t.Run(kind, func(t *testing.T) {
			runProof, ownerProof := "", ""
			if kind == "run" || kind == "both" {
				runProof = "disposable-run-proof"
			}
			if kind == "human" || kind == "both" {
				ownerProof = "disposable-human-proof"
			}
			t.Setenv(cliutil.EnvIdentityToken, runProof)
			calls := 0
			parent := "43db704a-1098-491e-9bb7-03f5af50c75a"
			task := "3f825e71-4c8a-40f3-89ad-026664950313"
			key := "auth01-fixture"
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/runs" {
					t.Error("unexpected route")
				}
				if r.Header.Get(cliutil.HeaderAgentIdentityToken) != runProof {
					t.Error("run proof lost or substituted")
				}
				want := ""
				if ownerProof != "" {
					want = "Bearer " + ownerProof
				}
				if r.Header.Get("Authorization") != want {
					t.Error("owner proof lost or substituted")
				}
				body := readAll(t, r)
				var received apipb.CreateRunRequest
				if err := protojson.Unmarshal(body, &received); err != nil {
					t.Error(err)
				}
				if received.TaskId != task || received.GetParentRunId() != parent || received.GetIdempotencyKey() != key {
					t.Error("lineage or replay key changed")
				}
				if strings.Contains(string(body), "disposable-") {
					t.Error("credential copied into body")
				}
				w.Header().Set("Content-Type", "application/json")
				if kind == "absent-refused" {
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte(`{"error":"verified caller required"}`))
					return
				}
				_, _ = w.Write([]byte(`{"run":{"id":"f2aa1340-e5fd-44d3-b83b-9b6f9c31d34a"}}`))
			}))
			defer srv.Close()
			api := cliutil.NewAPIClient(cliutil.NewHTTPClient(cliutil.HTTPClientOptions{}), func() cliutil.APIBaseOptions { return cliutil.APIBaseOptions{DefaultBase: srv.URL} }, func() string { return ownerProof })
			_, run, err := NewServices(api).Runs.Create(&apipb.CreateRunRequest{TaskId: task, ParentRunId: &parent, IdempotencyKey: &key})
			if kind == "absent-refused" {
				if err == nil || run != nil || !strings.Contains(err.Error(), "verified caller required") {
					t.Fatal("receiving refusal lost")
				}
			} else if err != nil || run == nil {
				t.Fatalf("proof transport failed: %v", err)
			}
			if calls != 1 {
				t.Fatal("refusal retried or duplicate request")
			}
		})
	}
}
