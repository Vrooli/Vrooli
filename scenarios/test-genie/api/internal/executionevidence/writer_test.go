package executionevidence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWriterStreamsArtifactAndPublishesValidatedManifest(t *testing.T) {
	root := t.TempDir()
	writer, err := NewWriter(root)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	findings, err := writer.StreamArtifact(context.Background(), "findings", "findings.document", "details/findings.ndjson", "application/x-ndjson", "security", strings.NewReader("one\ntwo\n"), 1024)
	if err != nil {
		t.Fatalf("StreamArtifact: %v", err)
	}
	if findings.SizeBytes != int64(len("one\ntwo\n")) || findings.SHA256 == "" {
		t.Fatalf("artifact reference = %#v", findings)
	}
	manifest := Manifest{
		SchemaVersion: SchemaVersion,
		RunID:         "run-1",
		Scenario:      "demo",
		CreatedAt:     time.Now().UTC(),
		Verdict:       "PASS",
		Findings:      findings,
		Phases: []PhaseSummary{{
			Name: "security", Status: "failed", FindingCount: 2, ObservationCount: 2, Findings: &findings,
		}},
	}
	if err := writer.WriteManifest(manifest); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ManifestFile)); err != nil {
		t.Fatalf("manifest was not published: %v", err)
	}
}

func TestReadManifestDoesNotOpenDetailedArtifact(t *testing.T) {
	root := t.TempDir()
	writer, err := NewWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	findings, err := writer.StreamArtifact(context.Background(), "findings", "findings.document", "findings.json", "application/json", "", strings.NewReader(`{"phases":[]}`), 1024)
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{SchemaVersion: SchemaVersion, RunID: "run-1", Scenario: "demo", Verdict: "PASS", CreatedAt: time.Now().UTC(), Findings: findings, Phases: []PhaseSummary{{Name: "unit", Status: "passed"}}}
	if err := writer.WriteManifest(want); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, "findings.json")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadManifest(root)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.RunID != want.RunID || got.Verdict != want.Verdict || len(got.Phases) != 1 {
		t.Fatalf("manifest = %#v", got)
	}
}

func TestWriterRejectsOversizedAndEscapingArtifactsWithoutPublishing(t *testing.T) {
	root := t.TempDir()
	writer, err := NewWriter(root)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	_, err = writer.StreamArtifact(context.Background(), "video", "workflow.video", "media/video.webm", "video/webm", "workflow", strings.NewReader("too-large"), 3)
	if !errors.Is(err, ErrArtifactTooLarge) {
		t.Fatalf("oversized error = %v, want ErrArtifactTooLarge", err)
	}
	if _, err := os.Stat(filepath.Join(root, "media", "video.webm")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("oversized artifact published unexpectedly: %v", err)
	}
	_, err = writer.StreamArtifact(context.Background(), "escape", "workflow.video", "../escape.webm", "video/webm", "workflow", strings.NewReader("x"), 10)
	if !errors.Is(err, ErrCorruptEvidence) {
		t.Fatalf("escape error = %v, want ErrCorruptEvidence", err)
	}
}

func TestManifestRejectsCompetingFindingsOwners(t *testing.T) {
	canonical := ArtifactRef{ID: "findings", Kind: "findings.document", RelativePath: "details/findings.ndjson", SizeBytes: 1, SHA256: "abc"}
	other := ArtifactRef{ID: "other", Kind: "findings.document", RelativePath: "details/other.ndjson", SizeBytes: 1, SHA256: "def"}
	err := (Manifest{
		SchemaVersion: SchemaVersion, RunID: "run", Scenario: "demo", CreatedAt: time.Now(), Verdict: "FAIL", Findings: canonical,
		Phases: []PhaseSummary{{Name: "security", Status: "failed", Findings: &other}},
	}).Validate()
	if !errors.Is(err, ErrCorruptEvidence) {
		t.Fatalf("Validate error = %v, want ErrCorruptEvidence", err)
	}
}
