package eventlog_test

import (
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

func TestEmitRecordSuperseded(t *testing.T) {
	emitter, repo := setupEmitter(t)
	emitter.EmitRecordSuperseded("rec-new", "rec-old", "regression")

	e := lastEvent(t, repo)
	if e.EntityID != "rec-new" {
		t.Errorf("entity_id should be the successor (rec-new), got %q", e.EntityID)
	}
	if e.EventType != eventlog.EventRecordSuperseded {
		t.Errorf("event_type = %q", e.EventType)
	}
	var p eventlog.RecordSupersededPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.SupersededID != "rec-old" || p.Reason != "regression" {
		t.Errorf("payload: %+v", p)
	}
}
