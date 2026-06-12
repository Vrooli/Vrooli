package aisearch

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/records"
)

func TestRecordPointID_Deterministic(t *testing.T) {
	a := recordPointID("rec-deadbeef")
	b := recordPointID("rec-deadbeef")
	if a != b {
		t.Errorf("expected deterministic UUIDv5, got %s vs %s", a, b)
	}
	if len(a) != 36 {
		t.Errorf("expected 36-char UUID, got %s (len=%d)", a, len(a))
	}
	if a[14] != '5' {
		t.Errorf("expected UUIDv5 version nibble '5', got %q", a[14])
	}
}

// Risk 4 mitigation: backlog and record point IDs must not collide even when
// the entity-suffix happens to be the same string ("x"). The namespace prefix
// discipline ("swarm-manager:record/" vs "swarm-manager:execute/") prevents it.
func TestRecordPointID_DiffersFromBacklogAndInitiative(t *testing.T) {
	r := recordPointID("x")
	b := backlogPointID(backlog.KindExecute, "x")
	i := initiativePointID("x")
	if r == b {
		t.Error("record and backlog point IDs collided for name 'x'")
	}
	if r == i {
		t.Error("record and initiative point IDs collided for name 'x'")
	}
}

func TestBuildRecordPayload_FieldsAndEntityDiscriminator(t *testing.T) {
	r := records.Record{
		ID:           "rec-abc",
		Kind:         records.KindFix,
		Scenario:     "audio-tools",
		BacklogRef:   "fix/voice-auto-stop",
		InitiativeID: "voice-reliability",
		Supersedes:   "rec-prev",
		SupersededBy: "",
		Outcome:      records.OutcomeShipped,
		Commit:       "deadbeef",
		FilesChanged: []string{"a.go", "b.go"},
		Stub:         false,
	}
	p := buildRecordPayload(r, "")
	if p["entity_type"] != "record" {
		t.Errorf("entity_type = %v, want 'record'", p["entity_type"])
	}
	if p["record_id"] != "rec-abc" {
		t.Errorf("record_id = %v", p["record_id"])
	}
	if p["kind"] != "fix" {
		t.Errorf("kind = %v", p["kind"])
	}
	if p["scenario"] != "audio-tools" {
		t.Errorf("scenario = %v", p["scenario"])
	}
	if p["backlog_ref"] != "fix/voice-auto-stop" {
		t.Errorf("backlog_ref = %v", p["backlog_ref"])
	}
	if p["initiative_id"] != "voice-reliability" {
		t.Errorf("initiative_id = %v", p["initiative_id"])
	}
	if p["supersedes"] != "rec-prev" {
		t.Errorf("supersedes = %v", p["supersedes"])
	}
	if p["outcome"] != "shipped" {
		t.Errorf("outcome = %v", p["outcome"])
	}
	if p["stub"] != false {
		t.Errorf("stub = %v", p["stub"])
	}
	files, ok := p["files_changed"].([]string)
	if !ok || len(files) != 2 {
		t.Errorf("files_changed = %v", p["files_changed"])
	}
	if _, present := p["payload_hash"]; present {
		t.Error("expected payload_hash absent when empty (clean-payload contract)")
	}
}

func TestBuildRecordPayload_IncludesPayloadHashWhenSet(t *testing.T) {
	r := records.Record{ID: "rec-1", Kind: records.KindFix, Scenario: "s"}
	p := buildRecordPayload(r, "sha256:cafe00000000beef")
	if p["payload_hash"] != "sha256:cafe00000000beef" {
		t.Errorf("payload_hash = %v", p["payload_hash"])
	}
}

func TestBuildRecordPayload_NilFilesNormalizedToEmptySlice(t *testing.T) {
	r := records.Record{ID: "rec-1", Kind: records.KindFix, Scenario: "s"}
	p := buildRecordPayload(r, "")
	files, ok := p["files_changed"].([]string)
	if !ok {
		t.Fatalf("expected files_changed as []string, got %T", p["files_changed"])
	}
	if len(files) != 0 {
		t.Errorf("expected empty files_changed slice, got %v", files)
	}
}

func TestComposeRecordText_UsesNarrativeFields(t *testing.T) {
	r := records.Record{
		Trigger:  "real-world verify scores 0.4-0.5 for self",
		Approach: "VAD trim asymmetry + centroid variance",
		RuledOut: []string{"missing enrollment", "wrong threshold"},
	}
	got := composeRecordText(r)
	for _, want := range []string{
		"real-world verify",
		"VAD trim asymmetry",
		"missing enrollment",
		"wrong threshold",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected composition to contain %q, got:\n%s", want, got)
		}
	}
}

func TestIndexRecord_SkipsStub(t *testing.T) {
	emb := &fakeEmbedder{}
	vs := &fakeVectorStore{}
	svc := NewService(emb, nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	stub := records.Record{
		ID:       "rec-stub",
		Kind:     records.KindFix,
		Scenario: "s",
		Stub:     true,
	}
	if err := svc.IndexRecord(context.Background(), stub); err != nil {
		t.Fatalf("unexpected error indexing stub: %v", err)
	}
	if emb.callCount() != 0 {
		t.Errorf("expected embedder not called for stub, got %d calls", emb.callCount())
	}
	if len(vs.points) != 0 {
		t.Errorf("expected no upsert for stub, got %d points", len(vs.points))
	}
}

func TestIndexRecord_EmbedsAndUpserts(t *testing.T) {
	emb := &fakeEmbedder{}
	vs := &fakeVectorStore{}
	svc := NewService(emb, nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	r := records.Record{
		ID:       "rec-1",
		Kind:     records.KindFix,
		Scenario: "audio-tools",
		Trigger:  "wedged at silence",
		Approach: "server flag + client latch",
		Outcome:  records.OutcomeShipped,
	}
	if err := svc.IndexRecord(context.Background(), r); err != nil {
		t.Fatalf("IndexRecord: %v", err)
	}
	if emb.callCount() != 1 {
		t.Errorf("expected 1 embed call, got %d", emb.callCount())
	}
	// Records have no reconciler coverage, so the write-through MUST ensure its
	// own collection before upserting — otherwise the first upsert 404s on a
	// non-existent collection and records search silently breaks. Guard the fix.
	if vs.ensureCalls < 1 {
		t.Errorf("expected IndexRecord to EnsureCollection before upsert, got %d ensure calls", vs.ensureCalls)
	}
	expectedID := recordPointID("rec-1")
	if _, ok := vs.points[expectedID]; !ok {
		t.Errorf("expected upsert at point %s, got points: %v", expectedID, keysOf(vs.points))
	}
	payload := vs.points[expectedID]
	if payload["entity_type"] != "record" {
		t.Errorf("entity_type = %v, want 'record'", payload["entity_type"])
	}
	if payload["record_id"] != "rec-1" {
		t.Errorf("record_id = %v", payload["record_id"])
	}
	if _, present := payload["payload_hash"]; !present {
		t.Error("expected payload_hash on upserted payload")
	}
}

func TestIndexRecord_ErrorsWhenNotConfigured(t *testing.T) {
	svc := NewService(&fakeEmbedder{}, nil, nil, nil, nil, 0)
	// recordStore not set
	r := records.Record{ID: "x", Kind: records.KindFix, Scenario: "s", Trigger: "t"}
	if err := svc.IndexRecord(context.Background(), r); err == nil {
		t.Error("expected error when record store unconfigured")
	}
}

func TestDeleteRecord_CallsStore(t *testing.T) {
	vs := &fakeVectorStore{}
	vs.seed(recordPointID("rec-1"), map[string]interface{}{"x": 1})
	svc := NewService(&fakeEmbedder{}, nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	if err := svc.DeleteRecord(context.Background(), "rec-1"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
	if vs.deleteCalls != 1 {
		t.Errorf("expected 1 delete call, got %d", vs.deleteCalls)
	}
}

func TestRecordIndexerAdapter_DelegatesToService(t *testing.T) {
	emb := &fakeEmbedder{}
	vs := &fakeVectorStore{}
	svc := NewService(emb, nil, nil, nil, nil, 0)
	svc.SetRecordStore(vs)

	adapter := NewRecordIndexerAdapter(svc)
	r := records.Record{
		ID: "rec-adapter", Kind: records.KindFix, Scenario: "s",
		Trigger: "t",
	}
	if err := adapter.IndexRecord(context.Background(), r); err != nil {
		t.Fatalf("adapter IndexRecord: %v", err)
	}
	if emb.callCount() != 1 {
		t.Errorf("expected adapter to delegate to service.IndexRecord (1 embed), got %d", emb.callCount())
	}
}

func TestRecordIndexerAdapter_NilSafe(t *testing.T) {
	var adapter *RecordIndexerAdapter
	// Calling IndexRecord on a nil adapter must not panic — the noop indexer
	// fallback in records.NewService relies on it being safe.
	if err := adapter.IndexRecord(context.Background(), records.Record{}); err != nil {
		t.Errorf("nil adapter should no-op, got %v", err)
	}
}

func keysOf(m map[string]map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
