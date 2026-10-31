package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	attentionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/visited-tracker/v1/attention"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// [REQ:VT-REQ-025] Each command's declared flags reach its typed RPC without
// reading another command's schema; campaign identity survives CLI rebuilding.
func TestAttentionManifestCommands(t *testing.T) {
	for _, tc := range []struct {
		name, method string
		args         []string
	}{
		{"ensure-campaign", "EnsureCampaign", []string{"--location", "/tmp", "--tag", "review", "--patterns", "*.go"}},
		{"preview", "Preview", []string{"--limit", "2"}},
		{"claim", "Claim", []string{"--worker", "worker", "--request-id", "stable"}},
		{"complete", "Complete", []string{"--worker", "worker", "--claim-id", "claim", "--outcome", "incomplete"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("VISITED_TRACKER_CAMPAIGN_ID", "existing-campaign")
			var request map[string]any
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/health" {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{"status":"healthy"}`))
					return
				}
				if r.URL.Path != "/vrooli.visited_tracker.v1.attention.AttentionService/"+tc.method {
					t.Errorf("unexpected path %s", r.URL.Path)
				}
				var msg proto.Message
				switch tc.method {
				case "EnsureCampaign":
					msg = &attentionv1.EnsureCampaignRequest{}
				case "Preview":
					msg = &attentionv1.PreviewRequest{}
				case "Claim":
					msg = &attentionv1.ClaimRequest{}
				case "Complete":
					msg = &attentionv1.CompleteRequest{}
				}
				data, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
				}
				if err := proto.Unmarshal(data, msg); err != nil {
					t.Error(err)
				}
				encoded, err := protojson.Marshal(msg)
				if err != nil {
					t.Error(err)
				}
				if err := json.Unmarshal(encoded, &request); err != nil {
					t.Error(err)
				}
				w.Header().Set("Content-Type", "application/proto")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			app.core.APIOverride = server.URL
			args := append([]string{"attention", tc.name, "--json"}, tc.args...)
			var output bytes.Buffer
			if err := app.core.CLI.RunWithWriters(args, &output, &output); err != nil {
				t.Fatal(err)
			}
			if tc.name != "ensure-campaign" && request["campaignId"] != "existing-campaign" {
				t.Fatalf("lost campaign identity: %#v", request)
			}
			if tc.name == "ensure-campaign" && request["tag"] != "review" {
				t.Fatalf("lost scope: %#v", request)
			}
		})
	}
}
