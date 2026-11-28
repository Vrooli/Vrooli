package stagedmessages

import (
	"context"
	"strings"
	"testing"
	"time"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
)

func stagedFiles(paths ...string) *v1.StagedMessageSnapshot {
	snapshot := &v1.StagedMessageSnapshot{}
	for _, path := range paths {
		snapshot.Files = append(snapshot.Files, &v1.StagedMessageFile{Path: path})
	}
	return snapshot
}

func TestGitContinuationsSuggestsTheRelatedSeriesHeadWithCarriedWorkLinks(t *testing.T) {
	dir, run := fixtureGit(t)
	writeFixture(t, dir, "a.go", "a1\n")
	writeFixture(t, dir, "b.go", "b1\n")
	run("", "add", "a.go", "b.go")
	run("", "commit", "-qm", "feat: start series p1\n\nVrooli-Effort: effort:gct-polish\nSigned-off-by: Someone <s@example.invalid>")
	first := strings.TrimSpace(run("", "rev-parse", "HEAD"))
	writeFixture(t, dir, "a.go", "a2\n")
	run("", "add", "a.go")
	run("", "commit", "-qm", "feat: start series p2\n\nVrooli-Effort: effort:gct-polish\nVrooli-Continues: "+first)
	head := strings.TrimSpace(run("", "rev-parse", "HEAD"))
	writeFixture(t, dir, "unrelated.txt", "x\n")
	run("", "add", "unrelated.txt")
	run("", "commit", "-qm", "chore: unrelated")

	got, err := GitContinuations{}.Continuations(context.Background(), dir, stagedFiles("a.go", "b.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Commit != head {
		t.Fatalf("want only the series head %s (p1 is already continued; unrelated shares nothing), got %+v", head, got)
	}
	if got[0].Subject != "feat: start series p2" || got[0].SharedPaths != 1 || got[0].Reason != "Shares 1 of 2 staged files" {
		t.Fatalf("unexpected candidate facts: %+v", got[0])
	}
	keys := []string{}
	for _, trailer := range got[0].Trailers {
		keys = append(keys, trailer.Key+": "+trailer.Value)
	}
	if strings.Join(keys, "|") != "Vrooli-Effort: effort:gct-polish|Vrooli-Continues: "+head {
		t.Fatalf("want carried work links then the predecessor link, got %v", keys)
	}
}

func TestRankContinuationsIgnoresWeakStaleAndUnrelatedCommits(t *testing.T) {
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	commit := func(hash string, age time.Duration, paths ...string) historyCommit {
		return historyCommit{Hash: hash, Time: now.Add(-age), Paths: paths}
	}
	staged := stagedFiles("a.go", "b.go", "c.go", "d.go")
	got := rankContinuations(staged, []historyCommit{
		commit("1111111111", time.Hour, "a.go", "x.go", "y.go", "z.go"),         // one shared path in a broad commit
		commit("2222222222", 20*24*time.Hour, "a.go", "b.go", "c.go", "d.go"),   // too old
		commit("3333333333", 2*time.Hour, "a.go", "b.go"),                       // focused on staged paths
		commit("4444444444", 3*time.Hour, "a.go", "b.go", "c.go", "other/e.go"), // broad overlap
	}, now)
	if len(got) != 2 || got[0].Commit != "3333333333" && got[0].Commit != "4444444444" {
		t.Fatalf("want the two strong recent candidates, got %+v", got)
	}
	for _, candidate := range got {
		if candidate.Commit == "1111111111" || candidate.Commit == "2222222222" {
			t.Fatalf("weak or stale commit suggested: %+v", candidate)
		}
	}
	if none := rankContinuations(staged, nil, now); len(none) != 0 {
		t.Fatalf("empty history must suggest nothing, got %+v", none)
	}
}
