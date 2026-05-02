package repo

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseRepoFlags(t *testing.T) {
	diff := parseDiffFlags([]string{"--path=api/main.go", "--staged"})
	if diff.path != "api/main.go" || !diff.staged {
		t.Fatalf("unexpected diff flags: %#v", diff)
	}

	stage := parseStageFlags([]string{"--scope=scenario:git-control-tower", "ui/src/App.tsx"})
	if stage.scope != "scenario:git-control-tower" {
		t.Fatalf("unexpected stage scope: %#v", stage)
	}
	if len(stage.paths) != 1 || stage.paths[0] != "ui/src/App.tsx" {
		t.Fatalf("unexpected stage paths: %#v", stage.paths)
	}

	commit := parseCommitFlags([]string{"-m", "test: improve cli coverage", "--conventional", "--amend"})
	if commit.message != "test: improve cli coverage" || !commit.conventional || !commit.amend {
		t.Fatalf("unexpected commit flags: %#v", commit)
	}

	sync := parseSyncStatusFlags([]string{"--fetch", "--remote=origin"})
	if !sync.fetch || sync.remote != "origin" {
		t.Fatalf("unexpected sync flags: %#v", sync)
	}
}

func TestFormatDiffOutputIncludesStatsAndRawDiff(t *testing.T) {
	var resp diffResponse
	resp.Path = "api/main.go"
	resp.Staged = true
	resp.Stats.Additions = 12
	resp.Stats.Deletions = 3
	resp.Stats.NetLines = 9
	resp.Stats.HunkCount = 2
	resp.Stats.LargestHunk = 8
	resp.Raw = "@@ -1 +1 @@"

	output := captureStdout(t, func() {
		formatDiffOutput(&resp)
	})

	for _, want := range []string{
		"Diff for: api/main.go",
		"(staged changes)",
		"Stats: +12 -3 (net +9) | 2 hunks, largest: 8 lines",
		"@@ -1 +1 @@",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("diff output missing %q:\n%s", want, output)
		}
	}
}

func TestPrintStageAndCommitResults(t *testing.T) {
	output := captureStdout(t, func() {
		printStageResult(&stageResponse{
			Success: true,
			Staged:  []string{"api/main.go", "ui/src/App.tsx"},
		})
		printUnstageResult(&stageResponse{
			Success:  false,
			Errors:   []string{"nothing staged"},
			Unstaged: nil,
		})
		printCommitResult(&commitResponse{
			Success: true,
			Hash:    "abcdef1",
			Message: "test: improve cli coverage",
			Amended: true,
		})
		printCommitResult(&commitResponse{
			Success:          false,
			Error:            "empty commit",
			ValidationErrors: []string{"message must be conventional"},
		})
	})

	for _, want := range []string{
		"Staged 2 file(s)",
		"+ api/main.go",
		"Unstaging failed:",
		"! nothing staged",
		"Amended: abcdef1",
		"Message: test: improve cli coverage",
		"Commit failed:",
		"Error: empty commit",
		"! message must be conventional",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stage/commit output missing %q:\n%s", want, output)
		}
	}
}

func TestFormatSyncStatusSections(t *testing.T) {
	resp := &syncStatusResponse{
		Branch:                "feature/test-seams",
		Upstream:              "origin/feature/test-seams",
		RemoteURL:             "git@example.com:repo.git",
		Ahead:                 2,
		Behind:                1,
		HasUpstream:           true,
		CanPush:               true,
		CanPull:               true,
		HasUncommittedChanges: true,
		SafetyWarnings:        []string{"pull before pushing"},
		Recommendations:       []string{"run git-control-tower repo sync-status --fetch"},
		Fetched:               true,
		FetchError:            "temporary DNS failure",
	}

	output := captureStdout(t, func() {
		formatSyncBranchInfo(resp)
		formatSyncActions(resp)
		formatSyncWarnings(resp)
	})

	for _, want := range []string{
		"Branch: feature/test-seams",
		"Upstream: origin/feature/test-seams",
		"Remote: git@example.com:repo.git",
		"Ahead: 2  Behind: 1",
		"Status: can push, can pull, has uncommitted changes",
		"Warnings:",
		"! pull before pushing",
		"Recommendations:",
		"-> run git-control-tower repo sync-status --fetch",
		"(fetched fresh data from remote)",
		"! Fetch error: temporary DNS failure",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("sync output missing %q:\n%s", want, output)
		}
	}
}

func TestFormatSyncBranchInfoWithoutUpstream(t *testing.T) {
	output := captureStdout(t, func() {
		formatSyncBranchInfo(&syncStatusResponse{Branch: "main"})
	})

	if !strings.Contains(output, "No upstream configured") {
		t.Fatalf("expected no-upstream message:\n%s", output)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() failed: %v", err)
	}
	t.Cleanup(func() {
		os.Stdout = original
	})
	os.Stdout = writer

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("closing stdout pipe writer failed: %v", err)
	}
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatalf("reading stdout pipe failed: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("closing stdout pipe reader failed: %v", err)
	}
	return buf.String()
}
