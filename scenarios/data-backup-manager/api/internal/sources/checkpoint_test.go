//go:build linux

package sources

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func checkpointFixture(t *testing.T) (string, string) {
	t.Helper()
	src := t.TempDir()
	for p, body := range map[string]string{".git/index": "synthetic index bytes", ".git/HEAD": "ref: refs/heads/example\n", "working": "unstaged fixture", "untracked": "untracked fixture", ".ignored": "ignored fixture"} {
		path := filepath.Join(src, p)
		if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(path, []byte(body), 0751); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.Mkdir(filepath.Join(src, "empty"), 0710); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink("missing", filepath.Join(src, "link")); e != nil {
		t.Fatal(e)
	}
	return src, t.TempDir()
}

// [REQ:DBM-WORKSPACE-FIDELITY] No Git, Kopia, network, or catalog operations.
func TestCheckpointRoundTripAndTamper(t *testing.T) {
	src, stage := checkpointFixture(t)
	ctx := context.Background()
	c := &checkpointCapturer{}
	original, e := InspectCheckpoint(ctx, src)
	if e != nil {
		t.Fatal(e)
	}
	a, e := c.Capture(ctx, CaptureSpec{Locator: src, StageDir: stage})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = VerifyCheckpoint(ctx, a.Path); e != nil {
		t.Fatal(e)
	}
	dst := filepath.Join(t.TempDir(), "restored")
	if e = c.Restore(ctx, RestoreSpec{ArtifactPath: a.Path, Target: dst}); e != nil {
		t.Fatal(e)
	}
	got, e := InspectCheckpoint(ctx, dst)
	if e != nil || !reflect.DeepEqual(original, got) {
		t.Fatalf("fidelity: %v", e)
	}
	after, e := InspectCheckpoint(ctx, src)
	if e != nil || !reflect.DeepEqual(original, after) {
		t.Fatal("source changed")
	}
	if e = os.Chmod(filepath.Join(a.Path, "tree", "working"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = VerifyCheckpoint(ctx, a.Path); e == nil {
		t.Fatal("metadata corruption accepted")
	}
	if e = c.Restore(ctx, RestoreSpec{ArtifactPath: a.Path, Target: filepath.Join(t.TempDir(), "new")}); e == nil {
		t.Fatal("damaged checkpoint restored")
	}
}

func TestCheckpointRejectsMovementAndUnsafeRoots(t *testing.T) {
	src, stage := checkpointFixture(t)
	calls := 0
	inspect := func(ctx context.Context, p string) (CheckpointManifest, error) {
		calls++
		m, e := InspectCheckpoint(ctx, p)
		if calls == 2 {
			m.Entries[0].Modified++
		}
		return m, e
	}
	if _, e := captureCheckpoint(context.Background(), CaptureSpec{Locator: src, StageDir: stage}, inspect); e == nil {
		t.Fatal("drift accepted")
	}
	if _, e := (&checkpointCapturer{}).Capture(context.Background(), CaptureSpec{Locator: src, StageDir: src}); e == nil {
		t.Fatal("overlapping roots accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := InspectCheckpoint(ctx, src); e == nil {
		t.Fatal("cancellation ignored")
	}
}

func TestCheckpointRejectsExternalGitAndExtraEntries(t *testing.T) {
	src := t.TempDir()
	if e := os.WriteFile(filepath.Join(src, ".git"), []byte("gitdir: /outside"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := InspectCheckpoint(context.Background(), src); e == nil {
		t.Fatal("external Git accepted")
	}
	src, stage := checkpointFixture(t)
	a, e := (&checkpointCapturer{}).Capture(context.Background(), CaptureSpec{Locator: src, StageDir: stage})
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(filepath.Join(a.Path, "tree", "extra-empty"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e = VerifyCheckpoint(context.Background(), a.Path); e == nil {
		t.Fatal("unexpected empty directory accepted")
	}
}

func TestCheckpointRefusesUnsupportedMetadataAndDestination(t *testing.T) {
	src, stage := checkpointFixture(t)
	if e := os.Link(filepath.Join(src, "working"), filepath.Join(src, "hardlink")); e != nil {
		t.Fatal(e)
	}
	if _, e := InspectCheckpoint(context.Background(), src); e == nil {
		t.Fatal("unpreserved hard-link topology accepted")
	}
	if e := os.Remove(filepath.Join(src, "hardlink")); e != nil {
		t.Fatal(e)
	}
	a, e := (&checkpointCapturer{}).Capture(context.Background(), CaptureSpec{Locator: src, StageDir: stage})
	if e != nil {
		t.Fatal(e)
	}
	dst := t.TempDir()
	sentinel := filepath.Join(dst, "existing")
	if e = os.WriteFile(sentinel, []byte("preserve"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = (&checkpointCapturer{}).Restore(context.Background(), RestoreSpec{ArtifactPath: a.Path, Target: dst}); e == nil {
		t.Fatal("existing destination accepted")
	}
	b, e := os.ReadFile(sentinel)
	if e != nil || string(b) != "preserve" {
		t.Fatal("existing data changed")
	}
}
