package onboard

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDigestFiles_DeterministicAndContentSensitive(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "alpha")
	writeFile(t, dir, "sub/b.txt", "beta")
	files := []string{"a.txt", "sub/b.txt"}

	d1, err := digestFiles(dir, files)
	if err != nil {
		t.Fatalf("digestFiles: %v", err)
	}
	// Order of the file list must not change the digest (we sort/normalise upstream,
	// but the digest itself absorbs entries in the given order — pass sorted).
	d2, err := digestFiles(dir, files)
	if err != nil {
		t.Fatalf("digestFiles: %v", err)
	}
	if d1 != d2 {
		t.Fatalf("digest not deterministic: %s vs %s", d1, d2)
	}

	// A content change must change the digest.
	writeFile(t, dir, "a.txt", "ALPHA-changed")
	d3, err := digestFiles(dir, files)
	if err != nil {
		t.Fatalf("digestFiles: %v", err)
	}
	if d3 == d1 {
		t.Fatalf("digest unchanged after content edit (%s)", d3)
	}
}

// The monorepo tracks symlinks — including symlinks to DIRECTORIES (e.g.
// scenario-local packages/iframe-bridge → ../../../packages/iframe-bridge).
// digestFiles must hash the link itself, exactly as writeTarStream ships it,
// never open through it (os.Open on a symlink-to-dir returns EISDIR).
func TestDigestFiles_HashesSymlinksWithoutFollowing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real/inner.txt", "content")
	writeFile(t, dir, "plain.txt", "plain")
	if err := os.Symlink("real", filepath.Join(dir, "dirlink")); err != nil {
		t.Fatalf("symlink dir: %v", err)
	}
	if err := os.Symlink("plain.txt", filepath.Join(dir, "filelink")); err != nil {
		t.Fatalf("symlink file: %v", err)
	}
	files := []string{"dirlink", "filelink", "plain.txt", "real/inner.txt"}

	d1, err := digestFiles(dir, files)
	if err != nil {
		t.Fatalf("digestFiles with symlinks: %v", err)
	}

	// A symlink digests its TARGET STRING (what tar ships), not the content
	// behind it: retargeting the link must change the digest even though the
	// reachable content is identical.
	if err := os.Remove(filepath.Join(dir, "filelink")); err != nil {
		t.Fatalf("remove filelink: %v", err)
	}
	if err := os.Symlink("./plain.txt", filepath.Join(dir, "filelink")); err != nil {
		t.Fatalf("retarget filelink: %v", err)
	}
	d2, err := digestFiles(dir, files)
	if err != nil {
		t.Fatalf("digestFiles after retarget: %v", err)
	}
	if d2 == d1 {
		t.Fatalf("digest unchanged after symlink retarget (%s)", d2)
	}

	// And a symlink must not collide with a regular file at the same path whose
	// content spells out the link encoding (the symlink hash is domain-separated
	// by construction, and this pins that property).
	linkRoot, fileRoot := t.TempDir(), t.TempDir()
	if err := os.Symlink("plain.txt", filepath.Join(linkRoot, "x")); err != nil {
		t.Fatalf("symlink x: %v", err)
	}
	writeFile(t, fileRoot, "x", "symlink\x00plain.txt")
	dLink, err := digestFiles(linkRoot, []string{"x"})
	if err != nil {
		t.Fatalf("digestFiles link root: %v", err)
	}
	dFile, err := digestFiles(fileRoot, []string{"x"})
	if err != nil {
		t.Fatalf("digestFiles file root: %v", err)
	}
	if dLink == dFile {
		t.Fatalf("symlink digest collides with same-encoding regular file")
	}
}

func TestBuildSyncRemoteCommand_DefaultAndExplicit(t *testing.T) {
	def := buildSyncRemoteCommand("")
	if !strings.Contains(def, `$HOME/vrooli`) {
		t.Fatalf("default sync command should resolve $HOME/vrooli: %s", def)
	}
	if !strings.Contains(def, syncDestMarker) {
		t.Fatalf("sync command must emit the dest marker: %s", def)
	}
	explicit := buildSyncRemoteCommand("/opt/checkout dir")
	if !strings.Contains(explicit, `'/opt/checkout dir'`) {
		t.Fatalf("explicit dest with a space must be shell-quoted: %s", explicit)
	}
}

func TestWriteTarStream_PreservesNamesWithSpaces(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "plain.txt", "one")
	writeFile(t, dir, "a dir/with space.txt", "two")

	var buf bytes.Buffer
	if err := writeTarStream(&buf, dir, []string{"plain.txt", "a dir/with space.txt"}); err != nil {
		t.Fatalf("writeTarStream: %v", err)
	}

	got := map[string]string{}
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read: %v", err)
		}
		b, _ := io.ReadAll(tr)
		got[hdr.Name] = string(b)
	}
	if got["plain.txt"] != "one" || got["a dir/with space.txt"] != "two" {
		t.Fatalf("tar did not preserve names/content with spaces: %#v", got)
	}
}

func TestGitWorkingTreeSource_Snapshot_UsesGitEnumeration(t *testing.T) {
	// Drive the source with a fake git so the enumeration + digest run without a
	// real repo: HEAD, toplevel, and an ls-files -z NUL list.
	dir := t.TempDir()
	writeFile(t, dir, "tracked.txt", "x")
	writeFile(t, dir, "untracked.txt", "y")

	src := &gitWorkingTreeSource{
		repoDir: dir,
		run: func(_ context.Context, _, _ string, args ...string) ([]byte, error) {
			switch {
			case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--show-toplevel":
				return []byte(dir + "\n"), nil
			case len(args) >= 2 && args[0] == "rev-parse" && args[1] == "HEAD":
				return []byte("deadbeefcafe\n"), nil
			case len(args) >= 1 && args[0] == "ls-files":
				return []byte("tracked.txt\x00untracked.txt\x00"), nil
			default:
				return nil, nil
			}
		},
	}
	snap, err := src.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.BaseHEAD != "deadbeefcafe" {
		t.Fatalf("BaseHEAD = %q", snap.BaseHEAD)
	}
	if len(snap.Files) != 2 || snap.Files[0] != "tracked.txt" {
		t.Fatalf("Files = %#v", snap.Files)
	}
	if snap.Digest == "" {
		t.Fatalf("Digest empty")
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
