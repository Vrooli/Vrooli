package lifecycle

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// stubFileInfo is a minimal os.FileInfo for exercising recognition logic without
// touching the filesystem.
type stubFileInfo struct {
	name string
	mode os.FileMode
}

func (s stubFileInfo) Name() string       { return s.name }
func (s stubFileInfo) Size() int64        { return 0 }
func (s stubFileInfo) Mode() os.FileMode  { return s.mode }
func (s stubFileInfo) ModTime() time.Time { return time.Unix(0, 0) }
func (s stubFileInfo) IsDir() bool        { return s.mode.IsDir() }
func (s stubFileInfo) Sys() any           { return nil }

// TestIsRunnableArtifactConsumesEvidence asserts the decision logic is a pure
// function of injected OS evidence — no runtime.GOOS branch. The same table runs
// identically regardless of host OS because the per-OS rule is in the recognizer
// seam, which the test supplies.
func TestIsRunnableArtifactConsumesEvidence(t *testing.T) {
	t.Parallel()
	regular := stubFileInfo{name: "scenario-api", mode: 0o644}

	cases := []struct {
		name      string
		checkType string
		info      os.FileInfo
		evidence  artifactEvidence
		want      bool
	}{
		{"binaries runnable", "binaries", regular, artifactEvidence{Known: true, Runnable: true}, true},
		{"binaries not runnable", "binaries", regular, artifactEvidence{Known: true, Runnable: false}, false},
		{"binaries probe unknown degrades to runnable", "binaries", regular, artifactEvidence{Known: false}, true},
		{"nil info is missing", "binaries", nil, artifactEvidence{Known: true, Runnable: true}, false},
		{"ui-bundle ignores recognizer (regular file)", "ui-bundle", regular, artifactEvidence{Known: true, Runnable: false}, true},
		{"ui-bundle directory is not an artifact", "ui-bundle", stubFileInfo{name: "dist", mode: os.ModeDir | 0o755}, artifactEvidence{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recognize := func(string, os.FileInfo) artifactEvidence { return tc.evidence }
			got := isRunnableArtifact(tc.checkType, "/tmp/"+tc.name, tc.info, recognize)
			if got != tc.want {
				t.Fatalf("isRunnableArtifact = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRecognizeArtifactDirectory confirms the default recognizer treats a
// directory or nil info as a known non-artifact regardless of OS.
func TestRecognizeArtifactDirectory(t *testing.T) {
	t.Parallel()
	if ev := recognizeArtifact("/tmp/x", nil); !ev.Known || ev.Runnable {
		t.Fatalf("nil info: got %+v, want {Known:true Runnable:false}", ev)
	}
	dir := stubFileInfo{name: "d", mode: os.ModeDir | 0o755}
	if ev := recognizeArtifact("/tmp/d", dir); !ev.Known || ev.Runnable {
		t.Fatalf("dir: got %+v, want {Known:true Runnable:false}", ev)
	}
}

// TestArtifactRecognizerDefaults proves the nil-seam accessor falls back to the
// host recognizer so a hand-built hostProbeDeps still decides correctly.
func TestArtifactRecognizerDefaults(t *testing.T) {
	t.Parallel()
	var deps hostProbeDeps
	if deps.artifactRecognizer() == nil {
		t.Fatal("artifactRecognizer() returned nil for an unwired hostProbeDeps")
	}
}

// TestVolumeCaseInsensitiveSeam covers the capability-flagged volume probe: an
// unknown verdict degrades to case-sensitive (false); an injected case-insensitive
// verdict propagates.
func TestVolumeCaseInsensitiveSeam(t *testing.T) {
	t.Parallel()
	unknown := hostProbeDeps{volumeCaseEvidence: func(string) caseEvidence { return caseEvidence{Known: false, Insensitive: true} }}
	if unknown.volumeCaseInsensitive("/x") {
		t.Fatal("unknown evidence must degrade to case-sensitive (false)")
	}
	insensitive := hostProbeDeps{volumeCaseEvidence: func(string) caseEvidence { return caseEvidence{Known: true, Insensitive: true} }}
	if !insensitive.volumeCaseInsensitive("/x") {
		t.Fatal("known case-insensitive evidence must propagate")
	}
}

// TestEvaluateFreshnessCaseFolding asserts that a manifest recorded with one
// casing matches the live input set under a different casing only when the spec
// is flagged case-insensitive; case-sensitive comparison reports stale.
func TestEvaluateFreshnessCaseFolding(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "Widget.go"), "package main\n")

	spec := cliutil.FreshnessSpec{SourceRoot: dir, ContextRoot: dir, Inputs: []string{"."}}
	manifest, err := cliutil.ComputeFreshnessManifest(spec, "binaries", nil, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("ComputeFreshnessManifest: %v", err)
	}
	// Rewrite the recorded rel to a different casing to simulate a manifest stamped
	// on a case-insensitive volume that reported a different case than the live FS.
	for i := range manifest.Files {
		if manifest.Files[i].Rel == "Widget.go" {
			manifest.Files[i].Rel = "widget.go"
		}
	}

	sensitive := spec
	sensitive.CaseInsensitive = false
	if v, err := cliutil.EvaluateFreshness(sensitive, manifest, nil); err != nil || !v.Stale {
		t.Fatalf("case-sensitive: got stale=%v err=%v, want stale", v.Stale, err)
	}

	insensitive := spec
	insensitive.CaseInsensitive = true
	if v, err := cliutil.EvaluateFreshness(insensitive, manifest, nil); err != nil || v.Stale {
		t.Fatalf("case-insensitive: got stale=%v (%s) err=%v, want fresh", v.Stale, v.Reason, err)
	}
}
