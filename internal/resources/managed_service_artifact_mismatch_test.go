package resources

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/binaryfetch"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// stageTree writes a small directory artifact and returns its path and digest.
func stageTree(t *testing.T, root, name string, files map[string]string) (string, string) {
	t.Helper()
	path := filepath.Join(root, name)
	for rel, body := range files {
		full := filepath.Join(path, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	digest, err := binaryfetch.TreeDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	return path, digest
}

func writeWitness(t *testing.T, artifactPath, declared, observed string) {
	t.Helper()
	record := InstallFacts{
		Resource:       "test-resource",
		RecordedAt:     time.Now().UTC(),
		ArtifactSHA256: declared,
		ObservedSHA256: observed,
		Layout:         "dir",
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installFactsPath(artifactPath), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestClassifyArtifactMismatchReturnsNilWhenVerified(t *testing.T) {
	root := t.TempDir()
	path, digest := stageTree(t, root, "artifact", map[string]string{"bin/run": "#!/bin/sh\n"})
	if got := classifyArtifactMismatch(path, "test-resource", digest, "dir"); got != nil {
		t.Fatalf("classifyArtifactMismatch() = %v, want nil", got)
	}
}

// The kyutai-stt case: the bytes are exactly what was staged and the manifest
// moved underneath them. Reporting this as "unavailable" is what left a healthy
// GPU-resident service reading as an uninstalled resource with no cure named.
func TestClassifyArtifactMismatchDetectsMovedDeclaration(t *testing.T) {
	root := t.TempDir()
	path, digest := stageTree(t, root, "artifact", map[string]string{"bin/run": "#!/bin/sh\n"})
	writeWitness(t, path, digest, digest)

	got := classifyArtifactMismatch(path, "test-resource", strings.Repeat("a", 64), "dir")
	if got == nil {
		t.Fatal("classifyArtifactMismatch() = nil, want a mismatch")
	}
	if got.Cause != MismatchDeclarationMoved {
		t.Fatalf("Cause = %q, want %q", got.Cause, MismatchDeclarationMoved)
	}
	if got.Actual != digest {
		t.Fatalf("Actual = %q, want the staged digest %q", got.Actual, digest)
	}
	for _, want := range []string{"the declaration moved", "vrooli resource install test-resource --reacquire"} {
		if !strings.Contains(got.Error(), want) {
			t.Fatalf("Error() = %q, want it to contain %q", got.Error(), want)
		}
	}
}

// The other half of the split: something wrote into the artifact store after
// installation (a Python interpreter dropping .pyc into its own pinned tree is
// the observed case).
func TestClassifyArtifactMismatchDetectsAlteredBytes(t *testing.T) {
	root := t.TempDir()
	path, staged := stageTree(t, root, "artifact", map[string]string{"bin/run": "#!/bin/sh\n"})
	writeWitness(t, path, staged, staged)

	// Simulate a runtime writing a derived file into the pinned tree.
	if err := os.WriteFile(filepath.Join(path, "bin", "run.pyc"), []byte("derived"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := classifyArtifactMismatch(path, "test-resource", staged, "dir")
	if got == nil {
		t.Fatal("classifyArtifactMismatch() = nil, want a mismatch")
	}
	if got.Cause != MismatchBytesChanged {
		t.Fatalf("Cause = %q, want %q", got.Cause, MismatchBytesChanged)
	}
	if !strings.Contains(got.Error(), "wrote into the artifact store after installation") {
		t.Fatalf("Error() = %q, want it to name the cause", got.Error())
	}
}

// Absence of a witness must be reported as absence, never guessed at.
func TestClassifyArtifactMismatchWithoutWitnessSaysSo(t *testing.T) {
	root := t.TempDir()
	path, _ := stageTree(t, root, "artifact", map[string]string{"bin/run": "#!/bin/sh\n"})
	got := classifyArtifactMismatch(path, "test-resource", strings.Repeat("b", 64), "dir")
	if got == nil {
		t.Fatal("classifyArtifactMismatch() = nil, want a mismatch")
	}
	if got.Cause != MismatchUnwitnessed {
		t.Fatalf("Cause = %q, want %q", got.Cause, MismatchUnwitnessed)
	}
	if !strings.Contains(got.Error(), "no staging witness was recorded") {
		t.Fatalf("Error() = %q, want it to admit the gap", got.Error())
	}
}

// writeInstallFacts must record what it SAW, not merely repeat what the
// manifest claimed. A record that only echoes the declaration cannot witness
// anything, which is exactly why the original outage was unattributable.
func TestWriteInstallFactsRecordsTheObservedDigest(t *testing.T) {
	root := t.TempDir()
	path, digest := stageTree(t, root, "artifact", map[string]string{"bin/run": "#!/bin/sh\n"})
	target := binaryfetch.AcquisitionTarget{ArtifactSHA256: digest, Layout: "dir"}
	if err := writeInstallFacts(path, "test-resource", binaryfetch.Facts{"os": "linux"}, target, resourcedeployment.ServiceArtifact{Path: "artifact", Version: "1.0.0", Layout: "dir", SHA256: digest}, time.Now()); err != nil {
		t.Fatalf("writeInstallFacts() error = %v", err)
	}
	record, ok, err := readInstallFacts(path)
	if err != nil || !ok {
		t.Fatalf("readInstallFacts() ok=%v err=%v", ok, err)
	}
	if record.ObservedSHA256 != digest {
		t.Fatalf("ObservedSHA256 = %q, want the digest of the staged bytes %q", record.ObservedSHA256, digest)
	}
}
