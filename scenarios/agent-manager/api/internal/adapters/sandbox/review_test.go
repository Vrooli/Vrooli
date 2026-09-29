package sandbox

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestPrepareReviewRetainsIdentityOnReplayAndRefusesUnknownDrain(t *testing.T) {
	for _, mode := range []string{"fresh", "replay", "empty-inventory", "live-process", "wrong-digest", "read-unavailable"} {
		t.Run(mode, func(t *testing.T) {
			id, request := uuid.New(), uuid.New()
			digest := strings.Repeat("a", 64)
			var calls []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.Method+" "+r.URL.Path)
				base := "/api/v1/sandboxes/" + id.String()
				switch r.Method + " " + r.URL.Path {
				case "GET " + base + "/reviews/" + request.String():
					if mode == "read-unavailable" {
						w.WriteHeader(503)
						return
					}
					if mode != "replay" {
						w.WriteHeader(404)
						return
					}
					fmt.Fprint(w, `{}`)
				case "POST " + base + "/stop":
					fmt.Fprint(w, `{}`)
				case "GET " + base + "/processes":
					if mode == "empty-inventory" {
						fmt.Fprint(w, `{"processes":[]}`)
						return
					}
					if mode == "live-process" {
						fmt.Fprint(w, `{"processes":[{"pid":123}]}`)
						return
					}
					fmt.Fprint(w, `{"processes":[{"pid":123,"exitCode":0}]}`)
				case "POST " + base + "/reviews":
					var body struct {
						RequestID uuid.UUID `json:"requestId"`
						Paths     []string  `json:"paths"`
					}
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RequestID != request || strings.Join(body.Paths, ",") != "src" {
						t.Error("capture lost pinned selection", body, err)
					}
					_ = json.NewEncoder(w).Encode(ReviewInput{SandboxID: id, RequestID: request, SHA256: digest})
				case "POST " + base + "/reviews/" + request.String() + "/workspace":
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["expectedSha256"] != digest {
						t.Error("materialization lost digest", body, err)
					}
					returnedDigest := digest
					if mode == "wrong-digest" {
						returnedDigest = strings.Repeat("b", 64)
					}
					_ = json.NewEncoder(w).Encode(ReviewInput{Root: t.TempDir(), SHA256: returnedDigest})
				default:
					t.Errorf("unexpected owner call: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer server.Close()
			got, err := NewWorkspaceSandboxProvider(server.URL).PrepareReview(t.Context(), id, request, []string{"src"})
			if mode != "fresh" && mode != "replay" {
				if err == nil {
					t.Fatal("unverified input was admitted", got)
				}
				if (mode == "empty-inventory" || mode == "live-process" || mode == "read-unavailable") && strings.Contains(strings.Join(calls, "\n"), "POST /api/v1/sandboxes/"+id.String()+"/reviews") {
					t.Fatal("captured before drain evidence", calls)
				}
				return
			}
			if err != nil || got.SHA256 != digest || got.RequestID != request || got.SandboxID != id {
				t.Fatalf("prepare: %+v, %v, calls %v", got, err, calls)
			}
			if mode == "replay" && len(calls) != 3 {
				t.Fatal("replay touched live sandbox instead of retained input", calls)
			}
		})
	}
}
