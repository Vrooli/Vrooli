// [REQ:GCT-OT-P1-002]
package stagedmessages

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func fixtureGit(t *testing.T) (string, func(string, ...string) string) {
	t.Helper()
	dir := t.TempDir()
	run := func(input string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null")
		cmd.Stdin = strings.NewReader(input)
		out, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("git %v: %v %s", args, e, out)
		}
		return string(out)
	}
	run("", "init", "-q")
	run("", "config", "user.name", "Fixture")
	run("", "config", "user.email", "fixture@example.invalid")
	writeFixture(t, dir, "file.txt", "base\n")
	run("", "add", "file.txt")
	run("", "commit", "-qm", "base")
	return dir, run
}

func writeFixture(t *testing.T, dir, path, data string) {
	t.Helper()
	if e := os.WriteFile(filepath.Join(dir, path), []byte(data), 0o600); e != nil {
		t.Fatal(e)
	}
}

func TestSnapshotReadsIndexNotWorkingTreeAndIgnoresWorktreeDrift(t *testing.T) {
	dir, git := fixtureGit(t)
	writeFixture(t, dir, "file.txt", "staged A\n")
	when := time.Unix(1700000100, 123456789)
	if e := os.Chtimes(filepath.Join(dir, "file.txt"), when, when); e != nil {
		t.Fatal(e)
	}
	git("", "add", "file.txt")
	writeFixture(t, dir, "file.txt", "unstaged B\n")
	g := GitSnapshotter{}
	c, e := g.Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(c.Bytes["file.txt"].After, []byte("staged A\n")) || strings.Contains(c.Snapshot.Files[0].Patch, "unstaged B") {
		t.Fatal("capture substituted working bytes")
	}
	hints := c.Snapshot.Files[0].TimingHints
	cached := false
	for _, h := range hints {
		if h.Source == "cached_index_stat_mtime" {
			cached = true
			if h.Timestamp != when.UTC().Format(time.RFC3339Nano) || h.ContentMatchesIndex {
				t.Fatalf("cached hint %#v", h)
			}
		}
	}
	if !cached {
		t.Fatal("missing cached modification-time hint")
	}
	writeFixture(t, dir, "file.txt", "another unstaged value\n")
	got, e := g.Identity(context.Background(), "repo", dir)
	if e != nil || got != c.Snapshot.Digest {
		t.Fatalf("worktree drift changed index identity %q %v", got, e)
	}
}

func TestSnapshotTouchRefreshDoesNotStaleContent(t *testing.T) {
	dir, git := fixtureGit(t)
	writeFixture(t, dir, "file.txt", "staged\n")
	git("", "add", "file.txt")
	g := GitSnapshotter{}
	first, e := g.Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	when := time.Unix(1700000300, 0)
	if e := os.Chtimes(filepath.Join(dir, "file.txt"), when, when); e != nil {
		t.Fatal(e)
	}
	git("", "update-index", "--refresh")
	second, e := g.Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	if first.Snapshot.Digest != second.Snapshot.Digest {
		t.Fatal("touch changed staged identity")
	}
	found := false
	for _, h := range second.Snapshot.Files[0].TimingHints {
		if h.Source == "cached_index_stat_mtime" {
			found = h.Timestamp == when.UTC().Format(time.RFC3339Nano)
		}
	}
	if !found {
		t.Fatal("refresh hint not labeled with new stat time")
	}
}

func TestPartialStagingHasMissingCachedMtime(t *testing.T) {
	dir, git := fixtureGit(t)
	var lines []string
	for i := 0; i < 30; i++ {
		lines = append(lines, string(rune('A'+i))+" base\n")
	}
	base := strings.Join(lines, "")
	writeFixture(t, dir, "file.txt", base)
	git("", "add", ".")
	git("", "commit", "-qm", "fixture lines")
	first := strings.Replace(base, lines[1], "selected\n", 1)
	both := strings.Replace(first, lines[22], "unselected\n", 1)
	writeFixture(t, dir, "file.txt", both)
	git("y\nn\n", "add", "-p", "file.txt")
	c, e := (GitSnapshotter{}).Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	if string(c.Bytes["file.txt"].After) != first {
		t.Fatal("partial index lost")
	}
	for _, h := range c.Snapshot.Files[0].TimingHints {
		if h.Source == "cached_index_stat_mtime" {
			t.Fatal("zero index cache time presented as known")
		}
	}
}

func TestOmittedContentStillParticipatesInSnapshotIdentity(t *testing.T) {
	dir, git := fixtureGit(t)
	writeFixture(t, dir, "large.txt", strings.Repeat("x", maxFileBytes+1))
	git("", "add", "large.txt")
	g := GitSnapshotter{}
	a, e := g.Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(a.Snapshot.Files) != 1 || a.Snapshot.Files[0].OmissionReason == "" {
		t.Fatal("missing bounded omission")
	}
	writeFixture(t, dir, "large.txt", strings.Repeat("y", maxFileBytes+1))
	git("", "add", "large.txt")
	b, e := g.Identity(context.Background(), "repo", dir)
	if e != nil || a.Snapshot.Digest == b {
		t.Fatal("omitted content drift was missed", e)
	}
}

func TestSnapshotRejectsUnmergedIndex(t *testing.T) {
	dir, git := fixtureGit(t)
	oid := strings.TrimSpace(git("", "rev-parse", "HEAD:file.txt"))
	git("0 "+oid+"\tfile.txt\n100644 "+oid+" 1\tfile.txt\n100644 "+oid+" 2\tfile.txt\n100644 "+oid+" 3\tfile.txt\n", "update-index", "--index-info")
	_, e := (GitSnapshotter{}).Capture(context.Background(), "repo", dir)
	if e == nil || !strings.Contains(e.Error(), "unmerged") {
		t.Fatalf("want explicit unmerged standing, got %v", e)
	}
}

func largeIndexFixture(t *testing.T) (string, func(string, ...string) string) {
	t.Helper()
	dir, git := fixtureGit(t)
	blob := strings.TrimSpace(git("selected immutable bytes\n", "hash-object", "-w", "--stdin"))
	var entries strings.Builder
	for i := 0; i < MaxManifestFiles; i++ {
		path := fmt.Sprintf("scenarios/scope-%02d/src/%sfile-%04d.txt", i%32, strings.Repeat("long-directory-name/", 9), i)
		fmt.Fprintf(&entries, "100644 %s\t%s\n", blob, path)
	}
	git(entries.String(), "update-index", "--index-info")
	return dir, git
}

func TestLargeManifestRetains4096IdentitiesBoundsGitWorkAndDetectsDistantDrift(t *testing.T) {
	dir, git := largeIndexFixture(t)
	// Count real Git subprocesses, including patch rendering, rather than using a
	// timing assertion tied to this host. All index writes remain in the fixture.
	realGit, e := exec.LookPath("git")
	if e != nil {
		t.Fatal(e)
	}
	bin := t.TempDir()
	logPath := filepath.Join(bin, "calls")
	wrapper := "#!/bin/sh\nprintf 'call\\n' >> \"$GCT_FIXTURE_GIT_CALL_LOG\"\nexec \"$GCT_FIXTURE_REAL_GIT\" \"$@\"\n"
	if e := os.WriteFile(filepath.Join(bin, "git"), []byte(wrapper), 0o700); e != nil {
		t.Fatal(e)
	}
	t.Setenv("GCT_FIXTURE_REAL_GIT", realGit)
	t.Setenv("GCT_FIXTURE_GIT_CALL_LOG", logPath)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	g := GitSnapshotter{}
	c, e := g.Capture(context.Background(), "repo", dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(c.Snapshot.Files) != MaxManifestFiles || len(c.Bytes) > maxDetailFiles {
		t.Fatalf("manifest=%d details=%d", len(c.Snapshot.Files), len(c.Bytes))
	}
	patchBytes, omissions := 0, 0
	scopes := map[string]bool{}
	for _, f := range c.Snapshot.Files {
		patchBytes += len(f.Patch)
		if f.OmissionReason != "" {
			omissions++
		} else {
			scopes[strings.Split(f.Path, "/")[1]] = true
		}
	}
	if patchBytes > maxPatchBytes || omissions < MaxManifestFiles-maxDetailFiles || len(scopes) != 32 {
		t.Fatalf("patch=%d omissions=%d source scopes=%d", patchBytes, omissions, len(scopes))
	}
	calls, e := os.ReadFile(logPath)
	if e != nil {
		t.Fatal(e)
	}
	processes := bytes.Count(calls, []byte("call\n"))
	if processes > maxDetailFiles*5+10 {
		t.Fatalf("%d Git processes exceeds detail-bound work", processes)
	}
	last := c.Snapshot.Files[len(c.Snapshot.Files)-2]
	if last.OmissionReason == "" {
		t.Fatal("fixture must drift a distant omitted file")
	}
	changed := strings.TrimSpace(git("different immutable bytes\n", "hash-object", "-w", "--stdin"))
	git("100644 "+changed+"\t"+last.Path+"\n", "update-index", "--index-info")
	got, e := g.Identity(context.Background(), "repo", dir)
	if e != nil || got == c.Snapshot.Digest {
		t.Fatalf("distant omitted drift lost: %q %v", got, e)
	}
	git("100644 "+changed+"\textra-file.txt\n", "update-index", "--index-info")
	if _, e = g.Identity(context.Background(), "repo", dir); e == nil || !strings.Contains(e.Error(), "4096") {
		t.Fatalf("4097th file must be explicit error: %v", e)
	}
	t.Logf("complete manifest=%d; detail files=%d; omissions=%d; source scopes=%d; patch bytes=%d; Git processes=%d", len(c.Snapshot.Files), len(c.Bytes), omissions, len(scopes), patchBytes, processes)
}

func TestRepresentativeDetailsBalanceScopesAndPreferSourceOverGeneratedBulk(t *testing.T) {
	var files []*v1.StagedMessageFile
	for i := 0; i < 4000; i++ {
		files = append(files, &v1.StagedMessageFile{Path: fmt.Sprintf("packages/proto/gen/go/file-%04d.pb.go", i)})
	}
	for i := 0; i < 80; i++ {
		files = append(files, &v1.StagedMessageFile{Path: fmt.Sprintf("scenarios/source-%02d/api/main.go", i)})
	}
	selected := representativeDetails(files)
	if len(selected) != maxDetailFiles {
		t.Fatalf("sample size=%d", len(selected))
	}
	for i := range selected {
		if bulkDetailPath(files[i].Path) {
			t.Fatal("generated bulk displaced available source scopes")
		}
	}
	// Two source scopes get equal quotas despite a 100:1 file-count difference.
	files = nil
	for i := 0; i < 1000; i++ {
		files = append(files, &v1.StagedMessageFile{Path: fmt.Sprintf("scenarios/bulk-source/api/%04d.go", i)})
	}
	for i := 0; i < 10; i++ {
		files = append(files, &v1.StagedMessageFile{Path: fmt.Sprintf("scenarios/small-source/api/%02d.go", i)})
	}
	selected = representativeDetails(files)
	for i := 1000; i < 1010; i++ {
		if !selected[i] {
			t.Fatal("small source scope disappeared under bulk")
		}
	}
}

func TestRepresentativeWholePatchPackingRedistributesUnusedAllowance(t *testing.T) {
	var files []*v1.StagedMessageFile
	for i := 0; i < maxDetailFiles; i++ {
		files = append(files, &v1.StagedMessageFile{Path: fmt.Sprintf("scenarios/scope-%02d/api/change.go", i), Patch: strings.Repeat("p", 2500)})
	}
	// One larger candidate must not consume the budget before small scopes.
	files[0].Patch = strings.Repeat("large", 10000)
	selected := representativePatches(files)
	bytes := 0
	for i := range selected {
		bytes += len(files[i].Patch)
	}
	if len(selected) < 32 || bytes > maxPatchBytes || selected[0] {
		t.Fatalf("whole-patch sample did not redistribute: files=%d bytes=%d large=%v", len(selected), bytes, selected[0])
	}
}

func TestRepresentativeRawContentBudgetReachesLaterScopes(t *testing.T) {
	dir, git := fixtureGit(t)
	base := strings.Repeat("unchanged context line\n", 3000)
	changed := strings.Replace(base, "unchanged context line", "selected context line", 1)
	old := strings.TrimSpace(git(base, "hash-object", "-w", "--stdin"))
	newBlob := strings.TrimSpace(git(changed, "hash-object", "-w", "--stdin"))
	entries := func(blob string) string {
		var out strings.Builder
		for _, scope := range []string{"early", "later"} {
			for i := 0; i < 40; i++ {
				fmt.Fprintf(&out, "100644 %s\tscenarios/%s/api/%03d.go\n", blob, scope, i)
			}
		}
		return out.String()
	}
	git(entries(old), "update-index", "--index-info")
	git("", "commit", "-qm", "base scopes")
	git(entries(newBlob), "update-index", "--index-info")
	c, err := (GitSnapshotter{}).Capture(context.Background(), "repo", dir)
	if err != nil {
		t.Fatal(err)
	}
	counts, retainedBytes := map[string]int{}, 0
	for path, fb := range c.Bytes {
		counts[detailScope(path)]++
		retainedBytes += len(fb.Before) + len(fb.After)
	}
	if len(c.Bytes) >= maxDetailFiles || counts["scenarios/early"] < 2 || counts["scenarios/later"] < 2 || retainedBytes > maxTotalBytes {
		t.Fatalf("raw-byte fairness lost: counts=%v bytes=%d", counts, retainedBytes)
	}
	t.Logf("bounded raw bytes=%d; retained scope details=%v", retainedBytes, counts)
}
