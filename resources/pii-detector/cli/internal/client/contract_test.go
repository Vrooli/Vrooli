package client

import (
	"encoding/json"
	"testing"
)

// contractGoldenJSON is the frozen /detect wire contract, identical to the
// server module's copy (server/contract_test.go). The server and CLI are
// separate Go modules and cannot share a type, so DetectResult / Span are
// hand-kept copies of server detectResponse / Span. Asserting the same literal
// on both sides turns a cross-module field drift into a committed test failure
// instead of a silent wire mismatch.
const contractGoldenJSON = `{"spans":[{"category":"EMAIL_ADDRESS","start":6,"end":21,"text":"bob@example.com","confidence":0.99,"source":"regex"}],"model_available":true,"model_id":"test/model"}`

func TestDetectResultContractRoundTrip(t *testing.T) {
	want := DetectResult{
		Spans: []Span{{
			Category:   "EMAIL_ADDRESS",
			Start:      6,
			End:        21,
			Text:       "bob@example.com",
			Confidence: 0.99,
			Source:     "regex",
		}},
		ModelAvailable: true,
		ModelID:        "test/model",
	}

	got, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal DetectResult: %v", err)
	}
	if string(got) != contractGoldenJSON {
		t.Fatalf("DetectResult wire format drifted from the server contract:\n got  %s\n want %s", got, contractGoldenJSON)
	}

	var round DetectResult
	if err := json.Unmarshal([]byte(contractGoldenJSON), &round); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	if len(round.Spans) != 1 || round.Spans[0] != want.Spans[0] ||
		round.ModelAvailable != want.ModelAvailable || round.ModelID != want.ModelID {
		t.Fatalf("round-trip mismatch: got %+v want %+v", round, want)
	}
}
