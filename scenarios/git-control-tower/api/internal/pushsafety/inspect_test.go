package pushsafety

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

const baseOID = "1111111111111111111111111111111111111111"
const headOID = "2222222222222222222222222222222222222222"
const blobOID = "3333333333333333333333333333333333333333"

type fakeCommands struct {
	calls   []string
	fail    string
	size    int64
	host    string
	deleted bool
	merged  bool
	moved   bool
	partial bool
	heads   int
	trees   int
}

func (f *fakeCommands) Run(ctx context.Context, dir string, input []byte, args ...string) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	key := strings.Join(args, " ")
	f.calls = append(f.calls, key)
	if f.fail != "" && strings.HasPrefix(key, f.fail) {
		return nil, errors.New("injected failure")
	}
	switch args[0] {
	case "check-ref-format", "merge-base":
		return nil, nil
	case "rev-parse":
		f.heads++
		if f.moved && f.heads > 1 {
			return []byte(baseOID), nil
		}
		return []byte(headOID), nil
	case "remote":
		if f.host != "" {
			return []byte(f.host), nil
		}
		return []byte("git@github.com:Vrooli/Vrooli.git"), nil
	case "ls-remote":
		return []byte(baseOID + "\trefs/heads/agi\n"), nil
	case "rev-list":
		if strings.Contains(key, "--min-parents") {
			if f.merged {
				return []byte(headOID), nil
			}
			return nil, nil
		}
		if strings.Contains(key, "--objects") {
			return []byte(blobOID + "\n"), nil
		}
		return []byte(baseOID + "\n" + headOID + "\n"), nil
	case "cat-file":
		if args[1] == "-e" {
			return nil, nil
		}
		if f.partial {
			return nil, nil
		}
		return []byte(fmt.Sprintf("%s blob %d\n", blobOID, f.size)), nil
	case "ls-tree":
		f.trees++
		if f.deleted && f.trees > 1 {
			return nil, nil
		}
		return []byte("100755 blob " + blobOID + "\tbundle/a\t[1]\n.bin\x00"), nil
	}
	return nil, fmt.Errorf("unexpected command %s", key)
}

// [REQ:GCT-OT-P0-006] Historical deletions must not hide an outgoing blob.
func TestInspectHistoryAndLimits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  int64
		state string
		files int
	}{{"below warning", WarningBytes, "clear", 0}, {"warning", WarningBytes + 1, "clear", 1}, {"exact limit", GitHubLimitBytes, "clear", 1}, {"over limit", GitHubLimitBytes + 1, "blocked", 1}} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeCommands{size: tc.size, deleted: true}
			r := Inspect(context.Background(), f, "/source", "origin", "agi")
			if !r.Complete || r.State != tc.state || len(r.Files) != tc.files {
				t.Fatalf("unexpected report: %+v", r)
			}
			if tc.files > 0 && (len(r.Files[0].Commits) != 1 || r.Files[0].Paths[0] != "bundle/a\t[1]\n.bin") {
				t.Fatalf("lost historical or literal path: %+v", r.Files)
			}
			if r.State == "blocked" && !r.CanPrepare {
				t.Fatal(r.RecoveryReason)
			}
		})
	}
}
func TestInspectFailuresAreUnknown(t *testing.T) {
	for _, step := range []string{"remote", "ls-remote", "rev-parse", "rev-list --reverse", "rev-list --objects", "cat-file --batch", "ls-tree"} {
		t.Run(step, func(t *testing.T) {
			f := &fakeCommands{size: GitHubLimitBytes + 1, fail: step}
			r := Inspect(context.Background(), f, "/source", "origin", "agi")
			if r.State != "unknown" || r.Complete || r.CanPrepare {
				t.Fatalf("failure became safety: %+v", r)
			}
		})
	}
}
func TestInspectRejectsIncompleteObjectMetadata(t *testing.T) {
	f := &fakeCommands{size: GitHubLimitBytes + 1, partial: true}
	r := Inspect(context.Background(), f, "/source", "origin", "agi")
	if r.Complete {
		t.Fatal("missing object metadata must not be clear")
	}
}
func TestInspectConcurrencyAndTopology(t *testing.T) {
	f := &fakeCommands{size: GitHubLimitBytes + 1, moved: true}
	r := Inspect(context.Background(), f, "/source", "origin", "agi")
	if r.Complete || r.CanPrepare {
		t.Fatal("source drift accepted")
	}
	f = &fakeCommands{size: GitHubLimitBytes + 1, merged: true}
	r = Inspect(context.Background(), f, "/source", "origin", "agi")
	if r.State != "blocked" || r.CanPrepare {
		t.Fatal("merge history accepted for rewrite")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r = Inspect(ctx, &fakeCommands{}, "/source", "origin", "agi")
	if r.Complete {
		t.Fatal("cancelled scan accepted")
	}
}
func TestHostPolicyAndDestinationIdentity(t *testing.T) {
	for _, host := range []string{"https://github.com.evil.test/a", "git@other:repo", "https://example.com/a"} {
		if IsGitHub(host) {
			t.Fatal(host)
		}
	}
	f := &fakeCommands{host: "https://example.com/a", size: GitHubLimitBytes + 1}
	r := Inspect(context.Background(), f, "/source", "origin", "agi")
	if r.State != "unknown" || !r.Complete || r.CanPrepare {
		t.Fatalf("unknown host: %+v", r)
	}
	a := Inspect(context.Background(), &fakeCommands{size: GitHubLimitBytes + 1}, "/one", "origin", "agi")
	b := Inspect(context.Background(), &fakeCommands{size: GitHubLimitBytes + 1}, "/two", "origin", "agi")
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("recovery identities must be repository-bound")
	}
}
func TestCommitPreservationAndTreeProof(t *testing.T) {
	raw := []byte("tree " + blobOID + "\nparent " + baseOID + "\nauthor A <a@b> 100 +0200\ncommitter C <c@d> 200 -0500\ngpgsig signature\n continued\nencoding UTF-8\n\nmessage\n\nbody\n")
	out, signed, e := rewriteCommit(raw, headOID, blobOID)
	if e != nil || !signed {
		t.Fatal(e)
	}
	for _, expected := range []string{"author A <a@b> 100 +0200", "committer C <c@d> 200 -0500", "encoding UTF-8", "\n\nmessage\n\nbody\n"} {
		if !strings.Contains(string(out), expected) {
			t.Fatalf("lost %q", expected)
		}
	}
	old := []byte("100755 blob " + blobOID + "\tkeep\x00100644 blob " + headOID + "\tdrop\x00")
	good := []byte("100755 blob " + blobOID + "\tkeep\x00")
	if e = verifyTrees(old, good, []string{"drop"}); e != nil {
		t.Fatal(e)
	}
	if e = verifyTrees(old, []byte("100644 blob "+blobOID+"\tkeep\x00"), []string{"drop"}); e == nil {
		t.Fatal("mode change went undetected")
	}
	if e = verifyTrees(old, old, []string{"drop"}); e == nil {
		t.Fatal("retained binary accepted")
	}
}
