package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHardCutEnvelopeIngestQueryAndIdempotency(t *testing.T) {
	_, ts := newTestServer(t)
	data, err := structpb.NewStruct(map[string]any{"kind": "test"})
	if err != nil {
		t.Fatal(err)
	}
	env := &domain.EventEnvelope{
		EventId: "evt-hard-cut", EventType: "example.audit.v1", OccurredAt: timestamppb.New(time.Now().UTC()),
		Source: &domain.EventSource{Scenario: "test-source", ActorKind: "system"},
		Target: &domain.EventTarget{Scenario: "test-target", Operation: "POST /example", Protocol: "connect"},
		Data: func() *anypb.Any {
			packed, packErr := anypb.New(data)
			if packErr != nil {
				t.Fatal(packErr)
			}
			return packed
		}(),
	}
	body, err := protojson.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	for attempt, want := range []int{http.StatusAccepted, http.StatusConflict} {
		resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("attempt %d status=%d want %d", attempt, resp.StatusCode, want)
		}
	}
	resp, err := http.Get(ts.URL + "/api/v1/events?source=test-source&target=test-target&type=example.audit.v1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("query status=%d", resp.StatusCode)
	}
	var raw []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 1 {
		t.Fatalf("query returned %d envelopes", len(raw))
	}
	got := &domain.EventEnvelope{}
	if err := protojson.Unmarshal(raw[0], got); err != nil {
		t.Fatal(err)
	}
	if got.GetTarget().GetOperation() != "POST /example" || got.GetSource().GetScenario() != "test-source" {
		t.Fatalf("unexpected queried envelope: %#v", got)
	}
}
