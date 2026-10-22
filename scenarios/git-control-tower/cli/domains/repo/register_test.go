package repo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
	repoconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo/repo_v1connect"
)

type diffClientStub struct {
	repoconnect.RepoServiceClient
	request  *repov1.GetRepoDiffRequest
	response *repov1.GetRepoDiffResponse
	err      error
}

func (s *diffClientStub) GetRepoDiff(_ context.Context, req *connect.Request[repov1.GetRepoDiffRequest]) (*connect.Response[repov1.GetRepoDiffResponse], error) {
	s.request = req.Msg
	if s.err != nil {
		return nil, s.err
	}
	return connect.NewResponse(s.response), nil
}

func TestParseRepoFlags(t *testing.T) {
	diff, err := parseDiffFlags([]string{"--path=api/main.go", "--staged"})
	if err != nil {
		t.Fatal(err)
	}
	if diff.path != "api/main.go" || !diff.staged {
		t.Fatalf("unexpected diff flags: %#v", diff)
	}

	historical, err := parseDiffFlags([]string{"--path=src/App.tsx", "--commit=0123456789abcdef0123456789abcdef01234567", "--mode=source", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	if historical.path != "src/App.tsx" || historical.commit != "0123456789abcdef0123456789abcdef01234567" || historical.mode != "source" || !historical.json {
		t.Fatalf("unexpected source flags: %#v", historical)
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

func TestParseDiffFlagsRequiresExactSourceSelector(t *testing.T) {
	for _, args := range [][]string{
		{"--mode=source", "--commit=0123456789abcdef0123456789abcdef01234567"},
		{"--path=src/App.tsx", "--commit=main", "--mode=source"},
		{"--path=src/App.tsx", "--commit=0123456789abcdef0123456789abcdef01234567"},
		{"--path=src/App.tsx", "--commit=0123456789abcdef0123456789abcdef01234567", "--mode=source", "--staged"},
	} {
		if _, err := parseDiffFlags(args); err == nil {
			t.Fatalf("expected invalid source selector to fail: %v", args)
		}
	}
	if !isFullCommitID(strings.Repeat("a", 64)) {
		t.Fatal("expected full SHA-256 object IDs to be accepted")
	}
}

func TestRunDiffReadsExactCommitSourceAndReturnsHashAndByteLength(t *testing.T) {
	const commitID = "0123456789abcdef0123456789abcdef01234567"
	const content = "export const answer = 42;\n"
	client := &diffClientStub{response: &repov1.GetRepoDiffResponse{
		RepoDir: "/workspace/repo", Path: "src/App.tsx", Mode: "source", FullContent: content,
		ContentHash: strings.Repeat("a", 64),
	}}
	oldFactory := repoClientFactory
	repoClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient { return client }
	t.Cleanup(func() { repoClientFactory = oldFactory })

	output := captureStdout(t, func() {
		if err := runDiff(nil, []string{"--path=src/App.tsx", "--commit=" + commitID, "--mode=source", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	if client.request.GetCommit() != commitID || client.request.GetMode() != "source" || client.request.GetPath() != "src/App.tsx" || client.request.GetStaged() {
		t.Fatalf("unexpected historical source request: %#v", client.request)
	}
	var got sourceResponse
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("source output must be JSON: %v\n%s", err, output)
	}
	if got.RequestedCommit != commitID || got.Path != "src/App.tsx" || got.ContentHash != strings.Repeat("a", 64) || got.ContentBytes != len([]byte(content)) || got.Content != content {
		t.Fatalf("source output lost provenance/content: %#v", got)
	}
}

func TestRunDiffDefaultStillRequestsWorkingTreeDiff(t *testing.T) {
	client := &diffClientStub{response: &repov1.GetRepoDiffResponse{Path: "src/App.tsx", HasDiff: false}}
	oldFactory := repoClientFactory
	repoClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient { return client }
	t.Cleanup(func() { repoClientFactory = oldFactory })

	output := captureStdout(t, func() {
		if err := runDiff(nil, []string{"--path=src/App.tsx", "--staged"}); err != nil {
			t.Fatal(err)
		}
	})
	if client.request.GetPath() != "src/App.tsx" || !client.request.GetStaged() || client.request.GetCommit() != "" || client.request.GetMode() != "" {
		t.Fatalf("default diff semantics changed: %#v", client.request)
	}
	if output != "No changes\n" {
		t.Fatalf("unexpected default output %q", output)
	}
}

func TestRunDiffDefaultJSONRendersTypedResponse(t *testing.T) {
	client := &diffClientStub{response: &repov1.GetRepoDiffResponse{Path: "src/App.tsx", HasDiff: false}}
	oldFactory := repoClientFactory
	repoClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient { return client }
	t.Cleanup(func() { repoClientFactory = oldFactory })

	output := captureStdout(t, func() {
		if err := runDiff(nil, []string{"--path=src/App.tsx", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	var got map[string]any
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("diff --json must emit JSON: %v\n%s", err, output)
	}
	if got["path"] != "src/App.tsx" || got["has_diff"] != nil {
		t.Fatalf("unexpected typed JSON response: %#v", got)
	}
}

func TestRunDiffPropagatesSourceReadErrors(t *testing.T) {
	wantErr := errors.New("commit is unavailable")
	client := &diffClientStub{err: wantErr}
	oldFactory := repoClientFactory
	repoClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient { return client }
	t.Cleanup(func() { repoClientFactory = oldFactory })

	err := runDiff(nil, []string{"--path=src/App.tsx", "--commit=0123456789abcdef0123456789abcdef01234567", "--mode=source"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected source read error %v, got %v", wantErr, err)
	}
}

func TestParseBlameFlagsBuildsTypedRequest(t *testing.T) {
	request, jsonOutput, err := parseBlameFlags([]string{
		"--path=api/main.go", "--revision=HEAD", "--start=4", "--end=9", "--enrich", "--json",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !jsonOutput || request.GetRevision() != "HEAD" || request.GetStartLine() != 4 || request.GetEndLine() != 9 || !request.GetEnrich() {
		t.Fatalf("unexpected typed blame request: %#v", request)
	}
	if len(request.GetPaths()) != 1 || request.GetPaths()[0] != "api/main.go" {
		t.Fatalf("unexpected blame paths: %v", request.GetPaths())
	}
}

func TestParseBlameFlagsRejectsInvalidLineBounds(t *testing.T) {
	if _, _, err := parseBlameFlags([]string{"--start=not-a-line"}); err == nil {
		t.Fatal("expected invalid --start to be rejected")
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

type trackedBinariesClientStub struct {
	repoconnect.RepoServiceClient
	listResp   *repov1.TrackedBinariesResponse
	listErr    error
	gotUntrack *repov1.UntrackBinaryRequest
}

func (s *trackedBinariesClientStub) GetTrackedBinaries(_ context.Context, _ *connect.Request[repov1.GetTrackedBinariesRequest]) (*connect.Response[repov1.TrackedBinariesResponse], error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return connect.NewResponse(s.listResp), nil
}

func (s *trackedBinariesClientStub) UntrackBinary(_ context.Context, req *connect.Request[repov1.UntrackBinaryRequest]) (*connect.Response[repov1.UntrackBinaryResponse], error) {
	s.gotUntrack = req.Msg
	return connect.NewResponse(&repov1.UntrackBinaryResponse{Success: true, RemovedFromIndex: true}), nil
}

func withTrackedBinariesClient(t *testing.T, client repoconnect.RepoServiceClient) {
	t.Helper()
	oldFactory := repoClientFactory
	repoClientFactory = func(*cliapp.ScenarioApp) repoconnect.RepoServiceClient { return client }
	t.Cleanup(func() { repoClientFactory = oldFactory })
}

func TestRunUntrackedListRendersCandidates(t *testing.T) {
	client := &trackedBinariesClientStub{listResp: &repov1.TrackedBinariesResponse{
		Binaries: []*repov1.TrackedBinary{{
			Path: "scenarios/prompt-manager/api/objectives-import", Bytes: 18281673, Format: "elf",
			OwnerDir: "scenarios/prompt-manager", IgnorePattern: "/api/objectives-import",
		}},
		TotalBytes:     18281673,
		HistoryWarning: "Untracking does not reclaim history.",
	}}
	withTrackedBinariesClient(t, client)

	// JSON output preserves the per-candidate remediation fields.
	jsonOut := captureStdout(t, func() {
		if err := runUntrackedList(nil, []string{"--json"}); err != nil {
			t.Fatal(err)
		}
	})
	var got struct {
		Candidates []untrackedCandidate `json:"candidates"`
		TotalBytes int64                `json:"total_bytes"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &got); err != nil {
		t.Fatalf("list output must be JSON: %v\n%s", err, jsonOut)
	}
	if len(got.Candidates) != 1 || got.Candidates[0].Path != "scenarios/prompt-manager/api/objectives-import" || got.Candidates[0].IgnorePattern != "/api/objectives-import" {
		t.Fatalf("list lost candidate fields: %#v", got)
	}

	// Human-readable output names the file, its size, and the ignore target.
	textOut := captureStdout(t, func() {
		if err := runUntrackedList(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"objectives-import", "17.43 MiB", "/api/objectives-import", "scenarios/prompt-manager/.gitignore"} {
		if !strings.Contains(textOut, want) {
			t.Fatalf("human output missing %q:\n%s", want, textOut)
		}
	}
}

func TestRunUntrackedListReportsEmpty(t *testing.T) {
	withTrackedBinariesClient(t, &trackedBinariesClientStub{listResp: &repov1.TrackedBinariesResponse{}})
	out := captureStdout(t, func() {
		if err := runUntrackedList(nil, nil); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "No tracked files flagged") {
		t.Fatalf("expected empty message, got:\n%s", out)
	}
}

func TestRunUntrackRequiresPath(t *testing.T) {
	if err := runUntrack(nil, nil); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Fatalf("expected usage error, got %v", err)
	}
}

func TestRunUntrackRejectsNonCandidatePath(t *testing.T) {
	client := &trackedBinariesClientStub{listResp: &repov1.TrackedBinariesResponse{}}
	withTrackedBinariesClient(t, client)
	err := runUntrack(nil, []string{"some/tracked/source.go"})
	if err == nil || !strings.Contains(err.Error(), "not a tracked removal candidate") {
		t.Fatalf("expected non-candidate guard error, got %v", err)
	}
	if client.gotUntrack != nil {
		t.Fatalf("must not attempt a mutation for a non-candidate path")
	}
}
