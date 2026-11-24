package retention

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/services/evidence"
)

func TestInspectArtifactTreeUsesCanonicalEvidenceIntegrity(t *testing.T) {
	root := t.TempDir()
	payload := []byte("frame bytes")
	path := filepath.Join(root, "frame.png")
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatal(err)
	}
	integrity, err := evidence.InspectFileIntegrity(path)
	if err != nil {
		t.Fatal(err)
	}
	files, size, digest, exists, err := inspectArtifactTree(root)
	if err != nil {
		t.Fatal(err)
	}
	aggregate := sha256.Sum256([]byte("frame.png\x00" + fmt.Sprint(integrity.SizeBytes) + "\x00" + integrity.SHA256 + "\n"))
	if !exists || files != 1 || size != integrity.SizeBytes || digest != hex.EncodeToString(aggregate[:]) {
		t.Fatalf("artifact tree integrity = files:%d size:%d digest:%s exists:%v; canonical file integrity=%+v", files, size, digest, exists, integrity)
	}
}

type exactFS struct{ failDelete, retainOnDelete bool }

type incompleteExactStore struct{ *fakeStore }

func (s incompleteExactStore) ListExecutions(ctx context.Context, q database.ExecutionQuery) ([]*database.ExecutionIndex, int, error) {
	rows, total, err := s.fakeStore.ListExecutions(ctx, q)
	if err == nil {
		total++
	}
	return rows, total, err
}

func (f exactFS) DirSize(path string) (int64, bool, error) { return (OSFileSystem{}).DirSize(path) }
func (f exactFS) RemoveAll(path string) error              { return os.RemoveAll(path) }
func (f exactFS) DeleteContained(_ context.Context, root, path string) error {
	if f.failDelete {
		return errors.New("fixture delete failure")
	}
	if f.retainOnDelete {
		return nil
	}
	if !underRoot(root, path) {
		return errors.New("outside root")
	}
	return os.RemoveAll(path)
}

func exactFixture(t *testing.T, status string, fs exactFS) (*Service, *fakeStore, *database.ExecutionIndex, string) {
	t.Helper()
	root, err := os.MkdirTemp("/tmp", "bas-exact-purge-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	e := mkExec(t, status, 2, uuid.New())
	dir := filepath.Join(root, e.ID.String())
	e.ResultPath = filepath.Join(dir, "result.json")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.ResultPath, []byte("fixture page content must never be returned"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeExactManifest(t, dir, e.ID, e.WorkflowID)
	writeExactInventory(t, dir, e.ID, e.WorkflowID)
	store := &fakeStore{execs: []*database.ExecutionIndex{e}}
	return NewService(store, fs, root, nil), store, e, dir
}

func writeExactManifest(t *testing.T, dir string, id, workflowID uuid.UUID) {
	t.Helper()
	manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + id.String() + `","workflow_id":"` + workflowID.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + id.String() + `","artifacts":[]}}`
	if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeExactInventory(t *testing.T, dir string, id, workflowID uuid.UUID, bindings ...any) {
	t.Helper()
	type file struct {
		Path   string `json:"path"`
		Bytes  int64  `json:"bytes"`
		SHA256 string `json:"sha256"`
	}
	entries := []file{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "purge-inventory.v1.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, file{Path: filepath.ToSlash(rel), Bytes: int64(len(data)), SHA256: hex.EncodeToString(sum[:])})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	inventory := map[string]any{"schema_version": "bas-execution-directory/v1", "execution_id": id.String(), "workflow_id": workflowID.String(), "files": entries}
	if len(bindings) > 0 {
		inventory["artifact_bindings"] = bindings[0]
	}
	raw, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "purge-inventory.v1.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestExactPurgeRequiresBidirectionalTypedArtifactBindings(t *testing.T) {
	t.Run("disk-backed manifest entry must exist", func(t *testing.T) {
		svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		artifactID := uuid.NewString()
		sum := sha256.Sum256([]byte("not present"))
		digest := hex.EncodeToString(sum[:])
		manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + execution.ID.String() + `","workflow_id":"` + execution.WorkflowID.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + execution.ID.String() + `","artifacts":[{"id":"` + artifactID + `","kind":"ARTIFACT_KIND_SCREENSHOT","size_bytes":11,"sha256":"` + digest + `","execution_id":"` + execution.ID.String() + `"}]}}`
		if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		binding := map[string]any{"artifact_id": artifactID, "scope": "evidence", "storage": "execution_file", "path": "artifacts/screenshots/missing.png", "bytes": 11, "sha256": digest}
		writeExactInventory(t, dir, execution.ID, execution.WorkflowID, []map[string]any{binding})
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("unmatched disk-backed manifest item accepted: %v", err)
		}
	})

	t.Run("unknown storage mode fails closed", func(t *testing.T) {
		svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		artifactID := uuid.NewString()
		sum := sha256.Sum256([]byte("inline bytes"))
		digest := hex.EncodeToString(sum[:])
		manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + execution.ID.String() + `","workflow_id":"` + execution.WorkflowID.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + execution.ID.String() + `","artifacts":[{"id":"` + artifactID + `","kind":"ARTIFACT_KIND_SCREENSHOT","size_bytes":12,"sha256":"` + digest + `","execution_id":"` + execution.ID.String() + `"}]}}`
		if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		binding := map[string]any{"artifact_id": artifactID, "scope": "evidence", "storage": "unknown", "bytes": 12, "sha256": digest}
		writeExactInventory(t, dir, execution.ID, execution.WorkflowID, []map[string]any{binding})
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("unknown storage mode accepted: %v", err)
		}
	})
}

func TestExactPurgeKeepsExternalEvidenceOutsideExecutionDirectory(t *testing.T) {
	svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	artifactID := uuid.NewString()
	content := []byte("storage-owned bytes")
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + execution.ID.String() + `","workflow_id":"` + execution.WorkflowID.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + execution.ID.String() + `","artifacts":[{"id":"` + artifactID + `","kind":"ARTIFACT_KIND_TRACE","size_bytes":` + fmt.Sprint(len(content)) + `,"sha256":"` + digest + `","execution_id":"` + execution.ID.String() + `"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(dir)
	externalPath := filepath.Join(root, "artifacts", execution.ID.String(), "trace.zip")
	if err := os.MkdirAll(filepath.Dir(externalPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalPath, content, 0o600); err != nil {
		t.Fatal(err)
	}
	binding := map[string]any{"artifact_id": artifactID, "scope": "evidence", "storage": "external", "object_key": "artifacts/" + execution.ID.String() + "/trace.zip", "bytes": len(content), "sha256": digest}
	writeExactInventory(t, dir, execution.ID, execution.WorkflowID, []map[string]any{binding})
	preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
	if err != nil {
		t.Fatalf("external evidence outside execution directory should be identified without claiming local ownership: %v", err)
	}
	if preview.ExternalEvidence != 1 {
		t.Fatalf("preview did not disclose the external evidence count: %+v", preview)
	}
	if _, err := svc.PurgeExact(context.Background(), execution.ID, execution.ID.String(), preview); !errors.Is(err, ErrExactExternalArtifacts) {
		t.Fatalf("purge should refuse artifacts outside the proven execution directory: %v", err)
	}
	if _, err := os.Stat(externalPath); err != nil {
		t.Fatalf("exact directory inventory must not remove external storage objects: %v", err)
	}
}

func TestExactPurgeRefusesExternallyWritableRecordingsRoot(t *testing.T) {
	svc, _, execution, _ := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	if err := os.Chmod(svc.recordingsRoot, 0o777); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
		t.Fatalf("externally writable recordings root accepted: %v", err)
	}
}

func TestExactPurgeRefusesWritableAncestorWithoutStickyProtection(t *testing.T) {
	parent, err := os.MkdirTemp("/tmp", "bas-exact-parent-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(parent) })
	if err := os.Chmod(parent, 0o770); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(parent, "recordings")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := verifyPurgeAncestors(root); !errors.Is(err, ErrExactInventoryInvalid) {
		t.Fatalf("writable ancestor without sticky protection accepted: %v", err)
	}
}

func TestExactPurgePreviewRefusesMissingAndActiveRows(t *testing.T) {
	svc, _, _, _ := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	if _, err := svc.PreviewExactPurge(context.Background(), uuid.New()); !errors.Is(err, ErrExactExecutionMissing) {
		t.Fatalf("missing: %v", err)
	}
	svc, _, execution, _ := exactFixture(t, database.ExecutionStatusRunning, exactFS{})
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactExecutionActive) {
		t.Fatalf("active: %v", err)
	}
}

func TestExactPurgePreviewRefusesMissingArtifactAndSharedPath(t *testing.T) {
	svc, store, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactArtifactMissing) {
		t.Fatalf("missing artifact: %v", err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeExactManifest(t, dir, execution.ID, execution.WorkflowID)
	writeExactInventory(t, dir, execution.ID, execution.WorkflowID)
	shared := *execution
	shared.ID = uuid.New()
	store.execs = append(store.execs, &shared)
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactArtifactOverlap) {
		t.Fatalf("shared artifact: %v", err)
	}
}

func TestExactPurgeInventoryRejectsUnindexedAndMismatchedFiles(t *testing.T) {
	t.Run("extra file", func(t *testing.T) {
		svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		if err := os.WriteFile(filepath.Join(dir, "unexpected.txt"), []byte("extra"), 0o600); err != nil {
			t.Fatal(err)
		}
		// Simulate inventory refresh after an unrelated file arrived. The
		// ownership allowlist must prevent the refreshed inventory from blessing it.
		writeExactInventory(t, dir, execution.ID, execution.WorkflowID)
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("unindexed file accepted: %v", err)
		}
	})
	t.Run("changed file", func(t *testing.T) {
		svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte("changed"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("mismatched file accepted: %v", err)
		}
	})
	t.Run("artifact absent from evidence manifest", func(t *testing.T) {
		svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		artifactDir := filepath.Join(dir, "artifacts", "screenshots")
		if err := os.MkdirAll(artifactDir, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(artifactDir, "step.png"), []byte("artifact bytes"), 0o600); err != nil {
			t.Fatal(err)
		}
		writeExactInventory(t, dir, execution.ID, execution.WorkflowID)
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("unmanifested artifact accepted: %v", err)
		}
	})
}

func TestExactPurgeRejectsAncestorAndDescendantArtifactPaths(t *testing.T) {
	for _, tc := range []struct {
		name string
		path func(string) string
	}{
		{name: "descendant", path: func(target string) string { return filepath.Join(target, "nested", "result.json") }},
		{name: "ancestor", path: func(target string) string { return filepath.Join(filepath.Dir(target), "result.json") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, store, execution, target := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
			other := mkExec(t, database.ExecutionStatusCompleted, 1, execution.WorkflowID)
			other.ResultPath = tc.path(target)
			store.execs = append(store.execs, other)
			if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactArtifactOverlap) {
				t.Fatalf("overlap not rejected: %v", err)
			}
		})
	}
}

func TestExactPurgeRequiresVerifiedArtifactManifest(t *testing.T) {
	svc, _, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	if err := os.Remove(filepath.Join(dir, "evidence.proto.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
		t.Fatalf("missing manifest: %v", err)
	}
	writeExactManifest(t, dir, uuid.New(), execution.WorkflowID)
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
		t.Fatalf("wrong manifest execution: %v", err)
	}
	manifest := `{"id":"` + uuid.NewString() + `","schema_version":"bas-replay/v1","execution_id":"` + execution.ID.String() + `","workflow_id":"` + execution.WorkflowID.String() + `","evidence":{"id":"` + uuid.NewString() + `","schema_version":"bas-evidence/v1","execution_id":"` + execution.ID.String() + `","artifacts":[{"id":"` + uuid.NewString() + `","kind":"ARTIFACT_KIND_SCREENSHOT","size_bytes":2,"sha256":"bad","execution_id":"` + execution.ID.String() + `"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "evidence.proto.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
		t.Fatalf("invalid artifact inventory: %v", err)
	}
}

func TestExactPurgePreviewRefusesIncompleteOwnershipInventory(t *testing.T) {
	svc, store, execution, _ := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	svc.store = incompleteExactStore{fakeStore: store}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); err == nil {
		t.Fatal("incomplete execution inventory must not prove artifact exclusivity")
	}
}

func TestExactPurgePreviewRefusesNoncanonicalTargetAndDuplicateRows(t *testing.T) {
	t.Run("noncanonical result path", func(t *testing.T) {
		svc, _, execution, _ := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		execution.ResultPath = filepath.Join(svc.recordingsRoot, "elsewhere", "result.json")
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("noncanonical path: %v", err)
		}
	})
	t.Run("duplicate execution row", func(t *testing.T) {
		svc, store, execution, _ := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
		duplicate := *execution
		store.execs = append(store.execs, &duplicate)
		if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactInventoryInvalid) {
			t.Fatalf("duplicate row: %v", err)
		}
	})
}

func TestExactPurgeRequiresSameUUIDConfirmation(t *testing.T) {
	svc, store, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PurgeExact(context.Background(), execution.ID, uuid.NewString(), preview); !errors.Is(err, ErrExactConfirmation) {
		t.Fatalf("wrong confirmation: %v", err)
	}
	if len(store.deleted) != 0 {
		t.Fatalf("wrong confirmation deleted rows: %v", store.deleted)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("wrong confirmation touched artifacts: %v", err)
	}
}

func TestExactPurgeRejectsArtifactsChangedSincePreview(t *testing.T) {
	svc, store, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Model a legitimate writer updating both the artifact and its inventory
	// after an operator reviewed the preview. The stale confirmation must not
	// authorize deletion of the newer, still internally consistent directory.
	if err := os.WriteFile(execution.ResultPath, []byte("new result after review"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeExactInventory(t, dir, execution.ID, execution.WorkflowID)
	if _, err := svc.PurgeExact(context.Background(), execution.ID, execution.ID.String(), preview); !errors.Is(err, ErrExactPreviewChanged) {
		t.Fatalf("stale preview accepted after artifact update: %v", err)
	}
	if _, err := store.GetExecution(context.Background(), execution.ID); err != nil {
		t.Fatalf("stale preview deleted execution row: %v", err)
	}
	if content, err := os.ReadFile(execution.ResultPath); err != nil || string(content) != "new result after review" {
		t.Fatalf("stale preview changed updated artifact: %q %v", content, err)
	}
}

func TestExactPurgeDeletesOnlyExactSameWorkflowExecution(t *testing.T) {
	svc, store, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	unrelated := mkExec(t, database.ExecutionStatusCompleted, 2, execution.WorkflowID)
	unrelatedDir := filepath.Join(svc.recordingsRoot, unrelated.ID.String())
	unrelated.ResultPath = filepath.Join(unrelatedDir, "result.json")
	if err := os.MkdirAll(unrelatedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelated.ResultPath, []byte("untouched fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	store.execs = append(store.execs, unrelated)
	preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ArtifactFiles < 2 || preview.ArtifactBytes == 0 || preview.ArtifactSHA256 == "" || preview.ArtifactPathSHA256 == "" || !preview.Ready {
		t.Fatalf("incomplete safe metadata: %+v", preview)
	}
	result, err := svc.PurgeExact(context.Background(), execution.ID, execution.ID.String(), preview)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExecutionID != execution.ID.String() {
		t.Fatalf("wrong result: %+v", result)
	}
	if _, err := store.GetExecution(context.Background(), execution.ID); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("exact row remains: %v", err)
	}
	if _, err := store.GetExecution(context.Background(), unrelated.ID); err != nil {
		t.Fatalf("unrelated row changed: %v", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("target artifacts remain: %v", err)
	}
	if got, err := os.ReadFile(unrelated.ResultPath); err != nil || string(got) != "untouched fixture" {
		t.Fatalf("unrelated artifact changed: %q %v", got, err)
	}
}

func TestExactPurgeDeleteAndVerificationFailuresAreReported(t *testing.T) {
	for _, tc := range []struct {
		name string
		fs   exactFS
		want error
	}{
		{name: "delete", fs: exactFS{failDelete: true}},
		{name: "verification", fs: exactFS{retainOnDelete: true}, want: ErrExactVerificationFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, execution, _ := exactFixture(t, database.ExecutionStatusCompleted, tc.fs)
			preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
			if err != nil {
				t.Fatal(err)
			}
			_, err = svc.PurgeExact(context.Background(), execution.ID, execution.ID.String(), preview)
			if err == nil {
				t.Fatal("expected failure")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("error %v does not wrap %v", err, tc.want)
			}
		})
	}
}

func TestExactPurgeDatabaseFailureLeavesVisibleRowAndMissingDirectory(t *testing.T) {
	svc, store, execution, dir := exactFixture(t, database.ExecutionStatusCompleted, exactFS{})
	store.failDel = map[uuid.UUID]bool{execution.ID: true}
	preview, err := svc.PreviewExactPurge(context.Background(), execution.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.PurgeExact(context.Background(), execution.ID, execution.ID.String(), preview); err == nil {
		t.Fatal("database delete failure was hidden")
	}
	if _, err := store.GetExecution(context.Background(), execution.ID); err != nil {
		t.Fatalf("failed database deletion hid row: %v", err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("directory should be removed before row failure: %v", err)
	}
	if _, err := svc.PreviewExactPurge(context.Background(), execution.ID); !errors.Is(err, ErrExactArtifactMissing) {
		t.Fatalf("retry should fail closed on missing directory: %v", err)
	}
}
