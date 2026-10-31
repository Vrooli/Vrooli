package main

import (
	"encoding/json"
	"testing"
)

// contractGoldenJSON is the frozen /detect wire contract. The CLI client
// (cli/internal/client/client.go) carries its own hand-kept copy of this shape
// in client.DetectResult / client.Span because the server and CLI are separate
// Go modules and cannot import a shared type. The identical literal is asserted
// in the client package's contract test, so a field rename/removal on either
// side of the module boundary breaks a committed test instead of silently
// drifting the wire format.
const contractGoldenJSON = `{"spans":[{"category":"EMAIL_ADDRESS","start":6,"end":21,"text":"bob@example.com","confidence":0.99,"source":"regex"}],"model_available":true,"model_id":"test/model"}`

func TestDetectResponseContractRoundTrip(t *testing.T) {
	want := detectResponse{
		Spans: []Span{{
			Category:   CategoryEmail,
			Start:      6,
			End:        21,
			Text:       "bob@example.com",
			Confidence: 0.99,
			Source:     "regex",
		}},
		ModelAvailable: true,
		ModelID:        "test/model",
	}

	// Marshal matches the frozen contract byte-for-byte.
	got, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal detectResponse: %v", err)
	}
	if string(got) != contractGoldenJSON {
		t.Fatalf("detectResponse wire format drifted:\n got  %s\n want %s", got, contractGoldenJSON)
	}

	// Unmarshal of the frozen contract reproduces the struct.
	var round detectResponse
	if err := json.Unmarshal([]byte(contractGoldenJSON), &round); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	if len(round.Spans) != 1 || round.Spans[0] != want.Spans[0] ||
		round.ModelAvailable != want.ModelAvailable || round.ModelID != want.ModelID {
		t.Fatalf("round-trip mismatch: got %+v want %+v", round, want)
	}
}
