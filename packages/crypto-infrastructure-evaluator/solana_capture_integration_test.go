package evaluator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	chainforensics "github.com/vrooli/chain-forensics"
)

// This is a real retained finalized transaction body exercised through the
// same shared package and CI2 evaluator import used by the production consumer.
// It validates evidence serialization; the packet is not a rug attribution.
func TestCI2ConsumesRetainedSolanaRawCaptureWithoutCreatingBridgeEdges(t *testing.T) {
	const signature = "tC9VgKe8NNjaZPm5hQU9dqG16aF4zwe4PTyxUctQx7tLK9BoZnd11kTGzehzLa2JERbmzUbkALJFbQe2KL5rMwu"
	body, err := os.ReadFile(filepath.Join("..", "chain-forensics", "evidence", "cf6-solana-svs-20261116", "getTransaction-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		if request.Method != "getTransaction" {
			t.Errorf("CI2 capture called unexpected method %q", request.Method)
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		var got string
		if len(request.Params) > 0 {
			_ = json.Unmarshal(request.Params[0], &got)
		}
		if got != signature {
			t.Errorf("CI2 capture selected wrong signature %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	client, err := chainforensics.NewSolanaRPCClient(server.URL, server.Client(), nil)
	if err != nil {
		t.Fatal(err)
	}
	capture, err := client.FetchTransaction(context.Background(), signature)
	if err != nil {
		t.Fatal(err)
	}
	if len(capture.JupiterOrderEngineFills) != 1 || capture.JupiterOrderEngineFills[0].InputAmount != "500455096778" || capture.JupiterOrderEngineFills[0].OutputAmount != "3746294021993" {
		t.Fatalf("CI2 capture did not retain the decoded Jupiter RFQ rival flow: %+v", capture.JupiterOrderEngineFills)
	}
	casework := chainforensics.Case{
		ID: "cf6-solana-svs-capture", Title: "Private finalized Solana capture regression", Private: true,
		Sources: []chainforensics.Source{capture.Source}, Occurrences: capture.Occurrences,
	}
	report, err := Evaluate(casework)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid || !report.PrivatePacket || report.SourceCount != 1 || report.OccurrenceCount != len(capture.Occurrences) || report.EdgeCount != 0 || report.PacketSHA256 == "" {
		t.Fatalf("CI2 did not preserve source-backed observation semantics: %+v", report)
	}
	for _, occurrence := range capture.Occurrences {
		if occurrence.SourceID != capture.Source.ID || occurrence.Transaction != signature {
			t.Fatalf("captured occurrence lost source/transaction identity: %+v", occurrence)
		}
	}
}
