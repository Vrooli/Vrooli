package git

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func fakeRunner(responses map[string]string, fail map[string]bool) Runner {
	return func(_ context.Context, _ string, args ...string) ([]byte, error) {
		key := strings.Join(args, " ")
		if fail[key] {
			return nil, errors.New("boom")
		}
		v, ok := responses[key]
		if !ok {
			return nil, errors.New("unexpected git call: " + key)
		}
		return []byte(v), nil
	}
}

func TestCaptureParsesCleanBranch(t *testing.T) {
	run := fakeRunner(map[string]string{
		"rev-parse --is-inside-work-tree":          "true\n",
		"rev-parse HEAD":                           "abc123def456\n",
		"rev-parse --abbrev-ref HEAD":              "agi\n",
		"status --porcelain --untracked-files=all": "",
		"log -1 --pretty=format:%s%x00%an%x00%cI":  "fix the thing\x00Matt\x002026-05-26T10:00:00Z",
	}, nil)

	st, err := CaptureWith(context.Background(), "/repo", run)
	if err != nil {
		t.Fatalf("CaptureWith: %v", err)
	}
	if st.Sha != "abc123def456" || st.Branch != "agi" || st.Detached {
		t.Fatalf("unexpected head state: %+v", st)
	}
	if st.Dirty || st.DirtySummary != "" {
		t.Fatalf("expected clean tree, got dirty=%v summary=%q", st.Dirty, st.DirtySummary)
	}
	if st.CommitMessage != "fix the thing" || st.CommitAuthor != "Matt" {
		t.Fatalf("unexpected commit metadata: %+v", st)
	}
	if st.CommitDate.IsZero() {
		t.Fatalf("expected parsed commit date")
	}
}

func TestCaptureSummarizesDirtyTree(t *testing.T) {
	run := fakeRunner(map[string]string{
		"rev-parse --is-inside-work-tree":          "true\n",
		"rev-parse HEAD":                           "sha\n",
		"rev-parse --abbrev-ref HEAD":              "agi\n",
		"status --porcelain --untracked-files=all": " M a.go\nM  b.go\n?? c.txt\n?? d.txt\n",
		"log -1 --pretty=format:%s%x00%an%x00%cI":  "m\x00a\x002026-05-26T10:00:00Z",
	}, nil)

	st, err := CaptureWith(context.Background(), "/repo", run)
	if err != nil {
		t.Fatalf("CaptureWith: %v", err)
	}
	if !st.Dirty {
		t.Fatal("expected dirty")
	}
	if st.DirtySummary != "2 modified, 2 untracked" {
		t.Fatalf("dirty summary = %q", st.DirtySummary)
	}
}

func TestCaptureDetachedHead(t *testing.T) {
	run := fakeRunner(map[string]string{
		"rev-parse --is-inside-work-tree":          "true\n",
		"rev-parse HEAD":                           "deadbeefcafe\n",
		"rev-parse --abbrev-ref HEAD":              "HEAD\n",
		"status --porcelain --untracked-files=all": "",
		"log -1 --pretty=format:%s%x00%an%x00%cI":  "m\x00a\x002026-05-26T10:00:00Z",
	}, nil)

	st, err := CaptureWith(context.Background(), "/repo", run)
	if err != nil {
		t.Fatalf("CaptureWith: %v", err)
	}
	if !st.Detached || st.Branch != "" {
		t.Fatalf("expected detached HEAD, got %+v", st)
	}
}

func TestCaptureNonRepository(t *testing.T) {
	run := fakeRunner(nil, map[string]bool{"rev-parse --is-inside-work-tree": true})
	_, err := CaptureWith(context.Background(), "/notrepo", run)
	if !errors.Is(err, ErrNotARepository) {
		t.Fatalf("expected ErrNotARepository, got %v", err)
	}
}

func TestCaptureEmptyRepoNoCommits(t *testing.T) {
	// HEAD unresolved (fresh repo) — sha/commit empty but not an error.
	run := fakeRunner(map[string]string{
		"rev-parse --is-inside-work-tree":          "true\n",
		"rev-parse --abbrev-ref HEAD":              "agi\n",
		"status --porcelain --untracked-files=all": "?? new.txt\n",
	}, map[string]bool{
		"rev-parse HEAD": true,
	})
	st, err := CaptureWith(context.Background(), "/repo", run)
	if err != nil {
		t.Fatalf("CaptureWith: %v", err)
	}
	if st.Sha != "" || st.CommitMessage != "" {
		t.Fatalf("expected empty head on fresh repo, got %+v", st)
	}
	if !st.Dirty {
		t.Fatal("expected dirty (untracked file)")
	}
}
