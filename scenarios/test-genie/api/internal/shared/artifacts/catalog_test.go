package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestArtifactCatalogDiscoversTypedEvidenceWithoutPhaseFiltering(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	scenarioDir := t.TempDir()
	runID := "run-catalog"
	pageDir := filepath.Join(RunUISmokePagesDir(scenarioDir, runID), "home")
	videoDir := filepath.Join(RunAutomationDir(scenarioDir, runID), "login-flow", "video")
	phaseDir := RunPhaseResultsDir(scenarioDir, runID)
	logDir := RunLogsDir(scenarioDir, runID)
	for _, dir := range []string{pageDir, videoDir, phaseDir, logDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	largeLabel := strings.Repeat("evidence-", 2048)
	if err := os.WriteFile(filepath.Join(pageDir, visualPageFile), []byte(`{"page":"/home","label":"`+largeLabel+`"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestArtifact(t, filepath.Join(pageDir, visualScreenshotFile), "png")
	writeTestArtifact(t, filepath.Join(videoDir, "recording-webm"), "video")
	writeTestArtifact(t, filepath.Join(phaseDir, "future-phase.json"), `{}`)
	writeTestArtifact(t, filepath.Join(logDir, "future-phase.log"), "command output")

	catalog, err := DiscoverArtifactCatalog(scenarioDir, runID, []ArtifactPhaseDeclaration{
		{Phase: "visual-provider", EvidenceKinds: []string{ArtifactKindScreenshot}},
		{Phase: "automation-provider", EvidenceKinds: []string{ArtifactKindWorkflowVideo}},
	}, time.Unix(100, 0), false)
	if err != nil {
		t.Fatalf("DiscoverArtifactCatalog: %v", err)
	}
	if err := WriteArtifactCatalog(scenarioDir, catalog); err != nil {
		t.Fatalf("WriteArtifactCatalog: %v", err)
	}
	loaded, err := ReadArtifactCatalog(scenarioDir, runID)
	if err != nil {
		t.Fatalf("ReadArtifactCatalog: %v", err)
	}
	if loaded.Digest == "" || loaded.SchemaVersion != ArtifactCatalogSchemaVersion {
		t.Fatalf("catalog metadata = %+v", loaded)
	}
	byKind := map[string]ArtifactRef{}
	for _, artifact := range loaded.Artifacts {
		byKind[artifact.Kind] = artifact
		if strings.Contains(artifact.ID, "/") || strings.Contains(artifact.ID, "home") {
			t.Fatalf("artifact ID leaks a storage locator: %q", artifact.ID)
		}
	}
	if got := byKind[ArtifactKindScreenshot].ProducingPhase; got != "visual-provider" {
		t.Fatalf("screenshot producer = %q", got)
	}
	if got := byKind[ArtifactKindScreenshot].Metadata["page"]; got != "/home" {
		t.Fatalf("screenshot page metadata = %q", got)
	}
	if got := byKind[ArtifactKindWorkflowVideo].ProducingPhase; got != "automation-provider" {
		t.Fatalf("video producer = %q", got)
	}
	if got := byKind[ArtifactKindPhaseResult].ProducingPhase; got != "future-phase" {
		t.Fatalf("phase-result producer = %q", got)
	}
	if got := byKind[ArtifactKindCommandLog].ProducingPhase; got != "future-phase" {
		t.Fatalf("command-log producer = %q", got)
	}
}

func TestArtifactCatalogUnknownKindRoundTripsAndResolvesGenerically(t *testing.T) {
	scenarioDir := t.TempDir()
	runID := "run-future"
	path := filepath.Join(RunDir(scenarioDir, runID), "future", "bundle.bin")
	writeTestArtifact(t, path, "future bytes")
	ref, err := RegisterArtifact(scenarioDir, runID, ArtifactRegistration{
		Kind: "future.bundle", MediaType: "application/octet-stream", Label: "Future bundle",
		ProducingPhase: "future-provider", Metadata: map[string]string{"future": "preserved"},
		StorageRoot: "run", StoragePath: "future/bundle.bin",
	})
	if err != nil {
		t.Fatalf("RegisterArtifact: %v", err)
	}
	if _, err := RefreshArtifactCatalog(scenarioDir, runID, nil, time.Unix(200, 0)); err != nil {
		t.Fatalf("RefreshArtifactCatalog: %v", err)
	}
	got, resolved, err := ResolveCatalogArtifact(scenarioDir, runID, ref.ID, nil)
	if err != nil {
		t.Fatalf("ResolveCatalogArtifact: %v", err)
	}
	if got.Kind != "future.bundle" || got.ProducingPhase != "future-provider" || got.Metadata["future"] != "preserved" || resolved != path {
		t.Fatalf("resolved = %+v %q", got, resolved)
	}
}

func TestArtifactCatalogRejectsCrossRunReuseMissingUnsafeAndDuplicateIDs(t *testing.T) { // [REQ:TESTGENIE-TYPED-EVIDENCE-P0]
	scenarioDir := t.TempDir()
	for _, runID := range []string{"run-a", "run-b"} {
		writeTestArtifact(t, filepath.Join(RunDir(scenarioDir, runID), "evidence.txt"), runID)
		catalog, err := DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Unix(100, 0), false)
		if err != nil {
			t.Fatal(err)
		}
		if err := WriteArtifactCatalog(scenarioDir, catalog); err != nil {
			t.Fatal(err)
		}
	}
	catA, _ := ReadArtifactCatalog(scenarioDir, "run-a")
	catB, _ := ReadArtifactCatalog(scenarioDir, "run-b")
	if catA.Artifacts[0].ID == catB.Artifacts[0].ID {
		t.Fatal("opaque IDs must be run-scoped")
	}
	if _, _, err := ResolveCatalogArtifact(scenarioDir, "run-a", catB.Artifacts[0].ID, nil); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("foreign ID error = %v, want ErrArtifactNotFound", err)
	}
	if err := os.Remove(filepath.Join(RunDir(scenarioDir, "run-a"), "evidence.txt")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveCatalogArtifact(scenarioDir, "run-a", catA.Artifacts[0].ID, nil); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("missing byte error = %v, want ErrArtifactNotFound", err)
	}

	runID := "run-link"
	runRoot := RunDir(scenarioDir, runID)
	if err := os.MkdirAll(runRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(runRoot, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	unsafe := ArtifactRef{
		ID: artifactID(runID, "run", "escape.txt"), Kind: ArtifactKindGenericFile,
		MediaType: "text/plain", Label: "escape", AccessCapability: ArtifactAccessStream,
		CreatedAt:  time.Unix(100, 0).UTC().Format(time.RFC3339Nano),
		Provenance: ArtifactProvenanceCatalog, StorageRoot: "run", StoragePath: "escape.txt",
	}
	if err := WriteArtifactCatalog(scenarioDir, ArtifactCatalog{SchemaVersion: 1, RunID: runID, GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Artifacts: []ArtifactRef{unsafe}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ResolveCatalogArtifact(scenarioDir, runID, unsafe.ID, nil); !errors.Is(err, ErrUnsafeArtifact) {
		t.Fatalf("symlink escape error = %v, want ErrUnsafeArtifact", err)
	}

	duplicate := ArtifactCatalog{SchemaVersion: 1, RunID: "run-dup", GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Artifacts: []ArtifactRef{unsafe, unsafe}}
	duplicate.Artifacts[0].ID = artifactID("run-dup", "run", "escape.txt")
	duplicate.Artifacts[1].ID = duplicate.Artifacts[0].ID
	if err := WriteArtifactCatalog(scenarioDir, duplicate); !errors.Is(err, ErrInvalidArtifactCatalog) {
		t.Fatalf("duplicate ID error = %v", err)
	}
}

func TestArtifactCatalogLegacyDiscoveryIsStableAndExplicit(t *testing.T) {
	scenarioDir := t.TempDir()
	runID := "legacy"
	writeTestArtifact(t, filepath.Join(RunDir(scenarioDir, runID), FindingsArtifactFile), `{}`)
	catalog, err := DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Unix(100, 0), true)
	if err != nil {
		t.Fatal(err)
	}
	if !catalog.LegacyDiscovered || len(catalog.Artifacts) != 1 || catalog.Artifacts[0].Provenance != ArtifactProvenanceLegacy {
		t.Fatalf("legacy catalog = %+v", catalog)
	}
	repeated, err := DiscoverArtifactCatalog(scenarioDir, runID, nil, time.Unix(200, 0), true)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.Digest != repeated.Digest || catalog.GeneratedAt != repeated.GeneratedAt {
		t.Fatalf("unchanged legacy projection is unstable: first=%+v repeated=%+v", catalog, repeated)
	}
}

func TestReadArtifactCatalogRejectsMismatchedRunIdentity(t *testing.T) {
	scenarioDir := t.TempDir()
	writeTestArtifact(t, filepath.Join(RunDir(scenarioDir, "run-a"), "artifact.txt"), "evidence")
	catalog, err := DiscoverArtifactCatalog(scenarioDir, "run-a", nil, time.Unix(100, 0), false)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteArtifactCatalog(scenarioDir, catalog); err != nil {
		t.Fatal(err)
	}
	pathA := RunArtifactCatalogPath(scenarioDir, "run-a")
	pathB := RunArtifactCatalogPath(scenarioDir, "run-b")
	raw, err := os.ReadFile(pathA)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(pathB), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pathB, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadArtifactCatalog(scenarioDir, "run-b"); !errors.Is(err, ErrInvalidArtifactCatalog) {
		t.Fatalf("mismatched catalog error = %v, want ErrInvalidArtifactCatalog", err)
	}
}

func writeTestArtifact(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
