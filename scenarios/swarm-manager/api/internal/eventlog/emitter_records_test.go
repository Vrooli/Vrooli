package eventlog_test

import (
	"context"
	"encoding/json"
	"testing"

	"swarm-manager/internal/eventlog"
)

func TestEmitRecordCreated_Filled(t *testing.T) {
	emitter, repo := setupEmitter(t)
	emitter.EmitRecordCreated("rec-1", "fix", "audio-tools", "fix/voice-auto-stop", false)

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityRecord {
		t.Errorf("entity_type = %q", e.EntityType)
	}
	if e.EntityID != "rec-1" {
		t.Errorf("entity_id = %q", e.EntityID)
	}
	if e.EventType != eventlog.EventRecordCreated {
		t.Errorf("event_type = %q", e.EventType)
	}

	var p eventlog.RecordCreatedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Kind != "fix" || p.Scenario != "audio-tools" || p.BacklogRef != "fix/voice-auto-stop" || p.Stub {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitRecordCreated_Stub(t *testing.T) {
	emitter, repo := setupEmitter(t)
	emitter.EmitRecordCreated("rec-2", "execute", "swarm-manager", "", true)

	e := lastEvent(t, repo)
	var p eventlog.RecordCreatedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !p.Stub {
		t.Errorf("expected stub=true, got %+v", p)
	}
	if p.BacklogRef != "" {
		t.Errorf("expected empty backlog_ref for stub-without-link, got %q", p.BacklogRef)
	}
}

func TestEmitRecordSupersededRejectsMissingActor(t *testing.T) {
	emitter, repo := setupEmitter(t)
	emitter.EmitRecordSuperseded(context.Background(), "rec-new", "rec-old", "regression")

	events, err := repo.All(context.Background())
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("record supersession without actor was persisted: %+v", events[0])
	}
}
