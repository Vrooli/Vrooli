package session

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRegistrySnapshotExposesMetadataWithoutPayloads(t *testing.T) {
	registry := NewRegistry(64)
	ledger, resumed, err := registry.Open("session-metadata", "resume-token")
	if err != nil || resumed {
		t.Fatalf("open ledger = resumed %t, err %v", resumed, err)
	}
	if _, err := ledger.Receive(Chunk{
		Sequence: 0, StartSample: 0, EndSample: 4,
		Audio: []byte("private audio"),
	}); err != nil {
		t.Fatalf("receive audio: %v", err)
	}
	if _, err := ledger.Commit(Segment{ID: "segment-1", Text: "private transcript", EndSample: 4}); err != nil {
		t.Fatalf("commit transcript: %v", err)
	}

	snapshot, ok := registry.Snapshot("session-metadata")
	if !ok {
		t.Fatal("snapshot not found")
	}
	if snapshot.SessionID != "session-metadata" || snapshot.ReceivedSequence != 0 || snapshot.ProcessedSequence != -1 {
		t.Fatalf("snapshot sequence metadata = %#v", snapshot)
	}
	if snapshot.RetainedBytes != len("private audio") || snapshot.TerminalReason != TerminalNone {
		t.Fatalf("snapshot retained metadata = %#v", snapshot)
	}
	if snapshot.CommittedCount != 1 {
		t.Fatalf("snapshot committed count = %d, want 1", snapshot.CommittedCount)
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal metadata snapshot: %v", err)
	}
	for _, privatePayload := range []string{"private audio", "private transcript"} {
		if strings.Contains(string(encoded), privatePayload) {
			t.Fatalf("metadata snapshot exposed %q: %s", privatePayload, encoded)
		}
	}
}

func TestRegistryConformanceSnapshotIsExplicitPayloadBearingPath(t *testing.T) {
	registry := NewRegistry(64)
	ledger, _, err := registry.Open("session-conformance", "resume-token")
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	if _, err := ledger.Commit(Segment{ID: "segment-1", Text: "private transcript", EndSample: 4}); err != nil {
		t.Fatalf("commit transcript: %v", err)
	}

	snapshot, ok := registry.ConformanceSnapshot("session-conformance")
	if !ok || len(snapshot.Committed) != 1 || snapshot.Committed[0].Text != "private transcript" {
		t.Fatalf("conformance snapshot = %#v, %t", snapshot, ok)
	}
}

func TestRegistrySnapshotRejectsMissingRegistrySessionAndID(t *testing.T) {
	var nilRegistry *Registry
	if snapshot, ok := nilRegistry.Snapshot("session"); ok || snapshot != (MetadataSnapshot{}) {
		t.Fatalf("nil registry snapshot = %#v, %t", snapshot, ok)
	}
	registry := NewRegistry(64)
	if snapshot, ok := registry.Snapshot(""); ok || snapshot != (MetadataSnapshot{}) {
		t.Fatalf("empty ID snapshot = %#v, %t", snapshot, ok)
	}
	if snapshot, ok := registry.Snapshot("missing"); ok || snapshot != (MetadataSnapshot{}) {
		t.Fatalf("missing session snapshot = %#v, %t", snapshot, ok)
	}
}

func TestLedgerMetadataSnapshotNilReceiver(t *testing.T) {
	var ledger *Ledger
	if snapshot := ledger.MetadataSnapshot(); snapshot != (MetadataSnapshot{}) {
		t.Fatalf("nil ledger metadata snapshot = %#v", snapshot)
	}
}
