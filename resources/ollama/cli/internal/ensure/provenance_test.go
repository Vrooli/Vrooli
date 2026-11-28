package ensure

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordModelInstallProvenanceWritesExactVerifiedRegistryIdentity(t *testing.T) {
	dataRoot := t.TempDir()
	modelRoot := filepath.Join(dataRoot, "models")
	if err := os.Mkdir(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	pull := &PinnedPull{repository: "library/model", tag: "latest", digest: "sha256:manifest"}
	verifiedAt := time.Date(2026, 11, 11, 17, 30, 0, 0, time.UTC)
	model := ModelInfo{Name: "model:latest", Digest: "sha256:manifest", Size: 123}
	if err := recordModelInstallProvenance(modelRoot, pull, model, verifiedAt); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dataRoot, modelProvenanceFilename)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("provenance permissions = %o, want 600", info.Mode().Perm())
	}
	var doc modelProvenanceDocument
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SchemaVersion != 1 || len(doc.Models) != 1 {
		t.Fatalf("provenance document = %+v", doc)
	}
	entry := doc.Models[0]
	if entry.Name != model.Name || entry.Digest != model.Digest || entry.ManifestDigest != pull.digest || entry.Registry != RegistryHost || entry.Repository != "library/model" || entry.Tag != "latest" || !entry.VerifiedAt.Equal(verifiedAt) {
		t.Fatalf("provenance entry = %+v", entry)
	}
}

func TestRecordModelInstallProvenanceSurvivesWriterCrashAndReopen(t *testing.T) {
	const helperEnv = "VROOLI_PROVENANCE_CRASH_HELPER"
	const rootEnv = "VROOLI_PROVENANCE_DATA_ROOT"
	if os.Getenv(helperEnv) == "1" {
		dataRoot := os.Getenv(rootEnv)
		modelRoot := filepath.Join(dataRoot, "models")
		pull := &PinnedPull{repository: "library/team/model", tag: "v2", digest: "sha256:crash-reopen"}
		model := ModelInfo{Name: "team/model:v2", Digest: "sha256:crash-reopen", Size: 456}
		verifiedAt := time.Date(2026, 11, 11, 18, 45, 0, 0, time.UTC)
		if err := recordModelInstallProvenance(modelRoot, pull, model, verifiedAt); err != nil {
			t.Fatalf("write durable provenance before simulated process crash: %v", err)
		}
		// Kill the helper process immediately after the receipt's file and
		// directory fsyncs, leaving any caller continuation unexecuted.
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("find crash-helper process: %v", err)
		}
		if err := process.Kill(); err != nil {
			t.Fatalf("crash helper after durable receipt: %v", err)
		}
		select {}
	}

	parent := t.TempDir()
	dataRoot := filepath.Join(parent, "resource-data")
	if err := os.Mkdir(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dataRoot, "models"), 0o700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRecordModelInstallProvenanceSurvivesWriterCrashAndReopen$")
	command.Env = testEnvironmentWithValues(helperEnv+"=1", rootEnv+"="+dataRoot)
	output, err := command.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || len(output) != 0 {
		t.Fatalf("writer subprocess did not stop at the simulated crash boundary: err=%v output=%s", err, output)
	}

	// Reopen from a separate process after the writer exited without cleanup.
	path := filepath.Join(dataRoot, modelProvenanceFilename)
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("reopen provenance at exact data root: %v", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm() != 0o600 {
		t.Fatalf("reopened provenance file mode/type = %v, want regular mode 0600", info.Mode())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc modelProvenanceDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("reopened provenance is not valid JSON: %v", err)
	}
	wantTime := time.Date(2026, 11, 11, 18, 45, 0, 0, time.UTC)
	if doc.SchemaVersion != 1 || len(doc.Models) != 1 {
		t.Fatalf("reopened provenance document = %+v", doc)
	}
	entry := doc.Models[0]
	if entry.Name != "team/model:v2" || entry.Digest != "sha256:crash-reopen" || entry.ManifestDigest != "sha256:crash-reopen" || entry.Registry != RegistryHost || entry.Repository != "library/team/model" || entry.Tag != "v2" || !entry.VerifiedAt.Equal(wantTime) {
		t.Fatalf("reopened provenance identity = %+v", entry)
	}
	if _, err := os.Lstat(filepath.Join(dataRoot, "models", modelProvenanceFilename)); !os.IsNotExist(err) {
		t.Fatalf("receipt was written under the model directory instead of exact resource-data root: %v", err)
	}
}

func testEnvironmentWithValues(values ...string) []string {
	replaced := make(map[string]struct{}, len(values))
	for _, value := range values {
		key, _, _ := strings.Cut(value, "=")
		replaced[key] = struct{}{}
	}
	env := make([]string, 0, len(os.Environ())+len(values))
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		if _, replace := replaced[key]; !replace {
			env = append(env, value)
		}
	}
	return append(env, values...)
}

func TestRecordModelInstallProvenanceRejectsMismatchedOrCorruptEvidence(t *testing.T) {
	dataRoot := t.TempDir()
	modelRoot := filepath.Join(dataRoot, "models")
	if err := os.Mkdir(modelRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	pull := &PinnedPull{repository: "library/model", tag: "latest", digest: "sha256:pinned"}
	now := time.Date(2026, 11, 11, 17, 31, 0, 0, time.UTC)
	if err := recordModelInstallProvenance(modelRoot, pull, ModelInfo{Name: "model:latest", Digest: "sha256:other", Size: 123}, now); err == nil {
		t.Fatal("mismatched installed digest was accepted")
	}
	path := filepath.Join(dataRoot, modelProvenanceFilename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("mismatched evidence created a provenance file: %v", err)
	}
	if err := os.WriteFile(path, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recordModelInstallProvenance(modelRoot, pull, ModelInfo{Name: "model:latest", Digest: "sha256:pinned", Size: 123}, now); err == nil {
		t.Fatal("corrupt existing provenance was overwritten")
	}
}

func TestSyncProvenanceDirectoryRequiresDurableDirectoryHandle(t *testing.T) {
	dir := t.TempDir()
	if err := syncProvenanceDirectory(dir); err != nil {
		t.Fatalf("sync supported provenance directory: %v", err)
	}
	file := filepath.Join(dir, "regular-file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := syncProvenanceDirectory(file); err == nil {
		t.Fatal("regular file was accepted as a durable directory")
	}
}

func TestRecordModelInstallProvenanceRejectsLinkedDataRoot(t *testing.T) {
	parent := t.TempDir()
	dataRoot := filepath.Join(parent, "data")
	if err := os.Mkdir(dataRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	linkedRoot := filepath.Join(parent, "linked-data")
	if err := os.Symlink(dataRoot, linkedRoot); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	pull := &PinnedPull{repository: "library/model", tag: "latest", digest: "sha256:pinned"}
	err := recordModelInstallProvenance(filepath.Join(linkedRoot, "models"), pull, ModelInfo{Name: "model:latest", Digest: "sha256:pinned", Size: 1}, time.Now())
	if err == nil {
		t.Fatal("linked model provenance root was accepted")
	}
}
