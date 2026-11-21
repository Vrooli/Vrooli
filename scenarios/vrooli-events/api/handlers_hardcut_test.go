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
		Correlation: &domain.EventCorrelation{AgentRunId: "run-query-owner"},
		Source:      &domain.EventSource{Scenario: "test-source", ActorKind: "system"},
		Target:      &domain.EventTarget{Scenario: "test-target", Operation: "POST /example", Protocol: "connect"},
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
	resp, err := http.Get(ts.URL + "/api/v1/events?source=test-source&target=test-target&type=example.audit.v1&agent_run_id=run-query-owner")
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

func TestQueryNewestFirstCursorIsBoundedAndUnknownCursorRefuses(t *testing.T) {
	_, ts := newTestServer(t)
	for _, id := range []string{"evt-cursor-a", "evt-cursor-b", "evt-cursor-c"} {
		data, err := structpb.NewStruct(map[string]any{"id": id})
		if err != nil {
			t.Fatal(err)
		}
		packed, err := anypb.New(data)
		if err != nil {
			t.Fatal(err)
		}
		env := &domain.EventEnvelope{
			EventId: id, EventType: "example.cursor.v1", OccurredAt: timestamppb.New(time.Now().UTC()),
			Source: &domain.EventSource{Scenario: "test-source", ActorKind: "system"}, Data: packed,
		}
		body, err := protojson.Marshal(env)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.Post(ts.URL+"/api/v1/events", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("ingest %s status=%d", id, resp.StatusCode)
		}
	}

	resp, err := http.Get(ts.URL + "/api/v1/events?type=example.cursor.v1&source=test-source&limit=2")
	if err != nil {
		t.Fatal(err)
	}
	var defaultOrder []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&defaultOrder); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(defaultOrder) != 2 {
		t.Fatalf("default query status=%d rows=%d", resp.StatusCode, len(defaultOrder))
	}
	for i, want := range []string{"evt-cursor-a", "evt-cursor-b"} {
		got := &domain.EventEnvelope{}
		if err := protojson.Unmarshal(defaultOrder[i], got); err != nil || got.GetEventId() != want {
			t.Fatalf("default row %d = %q err=%v; want %s", i, got.GetEventId(), err, want)
		}
	}

	resp, err = http.Get(ts.URL + "/api/v1/events?type=example.cursor.v1&source=test-source&order=desc&limit=2")
	if err != nil {
		t.Fatal(err)
	}
	var newest []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&newest); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(newest) != 2 {
		t.Fatalf("newest query status=%d rows=%d", resp.StatusCode, len(newest))
	}
	first := &domain.EventEnvelope{}
	if err := protojson.Unmarshal(newest[0], first); err != nil || first.GetEventId() != "evt-cursor-c" {
		t.Fatalf("first newest row = %q err=%v; want evt-cursor-c", first.GetEventId(), err)
	}
	last := &domain.EventEnvelope{}
	if err := protojson.Unmarshal(newest[1], last); err != nil || last.GetEventId() != "evt-cursor-b" {
		t.Fatalf("second newest row = %q err=%v; want evt-cursor-b", last.GetEventId(), err)
	}
	resp, err = http.Get(ts.URL + "/api/v1/events?type=example.cursor.v1&source=test-source&order=desc&before_event_id=" + last.GetEventId() + "&limit=2")
	if err != nil {
		t.Fatal(err)
	}
	var older []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&older); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || len(older) != 1 {
		t.Fatalf("cursor query status=%d rows=%d", resp.StatusCode, len(older))
	}
	oldest := &domain.EventEnvelope{}
	if err := protojson.Unmarshal(older[0], oldest); err != nil || oldest.GetEventId() != "evt-cursor-a" {
		t.Fatalf("cursor row = %q err=%v", oldest.GetEventId(), err)
	}

	resp, err = http.Get(ts.URL + "/api/v1/events?order=desc&before_event_id=missing&limit=1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown cursor status=%d", resp.StatusCode)
	}
}
