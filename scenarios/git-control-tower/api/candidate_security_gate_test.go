package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type candidateScannerStub struct {
	result CandidateSecurityResult
	err    error
	files  map[string][]byte
}

func (s *candidateScannerStub) Scan(_ context.Context, files map[string][]byte) (CandidateSecurityResult, error) {
	s.files = files
	return s.result, s.err
}

func TestScanStagedCandidateDetectsIndexBytesAgainstCleanWorktree(t *testing.T) {
	const secret = "staged-only-secret-value"
	git := NewFakeGitRunner().AddStagedFile("config/candidate.env")
	git.Staged["config/candidate.env"] = "TOKEN=" + secret + "\n"
	git.Unstaged["config/candidate.env"] = "TOKEN=safe-working-tree-value\n"
	scanner := &candidateScannerStub{result: CandidateSecurityResult{
		Complete: true,
		Findings: []CandidateSecurityFinding{{
			RuleID: "gitleaks.generic-api-key", Severity: "error", FilePath: "config/candidate.env:1",
			Description: "matched value is redacted",
		}},
	}}
	result := scanStagedCandidate(context.Background(), git, "/repo", scanner)
	if result.Status != "failed" || !strings.Contains(result.Stdout, "config/candidate.env:1") {
		t.Fatalf("scan result = %+v", result)
	}
	if strings.Contains(result.Stdout, secret) || string(scanner.files["config/candidate.env"]) != "TOKEN="+secret+"\n" {
		t.Fatalf("candidate bytes or redaction incorrect: result=%+v files=%q", result, scanner.files["config/candidate.env"])
	}
}

func TestScanStagedCandidateIgnoresSecretOnlyInWorkingTree(t *testing.T) {
	const secret = "working-tree-only-secret"
	git := NewFakeGitRunner().AddStagedFile("config/candidate.env")
	git.Staged["config/candidate.env"] = "TOKEN=clean-index-value\n"
	git.Unstaged["config/candidate.env"] = "TOKEN=" + secret + "\n"
	scanner := &candidateScannerStub{result: CandidateSecurityResult{Complete: true}}
	result := scanStagedCandidate(context.Background(), git, "/repo", scanner)
	if result.Status != "passed" {
		t.Fatalf("clean staged candidate result = %+v", result)
	}
	if string(scanner.files["config/candidate.env"]) != "TOKEN=clean-index-value\n" {
		t.Fatalf("scanner received working-tree content: %q", scanner.files["config/candidate.env"])
	}
	if strings.Contains(result.Stdout+result.Stderr, secret) {
		t.Fatal("working-tree-only secret leaked into result")
	}
}

// TestLiveSecurityHealthScansRealVrooliWorktreeIndex is an opt-in integration
// check for the existing CreateCommit -> GCT candidate gate -> Security Health
// RPC path. All Git mutations, including the one benign commit, stay in a
// disposable linked worktree based on tracked Vrooli HEAD files.
func TestLiveSecurityHealthScansRealVrooliWorktreeIndex(t *testing.T) {
	if os.Getenv("GCT_RUN_LIVE_SECURITY_HEALTH_CANDIDATE") != "1" {
		t.Skip("set GCT_RUN_LIVE_SECURITY_HEALTH_CANDIDATE=1 to use the managed Security Health service")
	}
	rootOutput, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Fatal("resolve Vrooli repository root")
	}
	root := strings.TrimSpace(string(rootOutput))
	const candidatePath = "README.md"
	if err := exec.Command("git", "-C", root, "cat-file", "-e", "HEAD:"+candidatePath).Run(); err != nil {
		t.Fatal("candidate path is not a tracked Vrooli file at HEAD")
	}
	worktree := filepath.Join(t.TempDir(), "real Vrooli candidate worktree")
	if _, err := exec.Command("git", "-C", root, "worktree", "add", "--detach", worktree, "HEAD").CombinedOutput(); err != nil {
		t.Fatalf("create isolated Vrooli worktree (exit %v)", err)
	}
	t.Cleanup(func() {
		if _, err := exec.Command("git", "-C", root, "worktree", "remove", "--force", worktree).CombinedOutput(); err != nil {
			t.Errorf("remove isolated worktree (exit %v)", err)
		}
	})
	if mainIndex, fixtureIndex := gitPath(t, root, "index"), gitPath(t, worktree, "index"); mainIndex == fixtureIndex {
		t.Fatal("fixture worktree unexpectedly shares the main checkout index")
	}
	for _, setting := range [][2]string{{"user.name", "RH2 disposable showcase"}, {"user.email", "rh2-showcase@example.invalid"}} {
		if err := exec.Command("git", "-C", worktree, "config", setting[0], setting[1]).Run(); err != nil {
			t.Fatalf("configure disposable commit identity %s", setting[0])
		}
	}

	filePath := filepath.Join(worktree, filepath.FromSlash(candidatePath))
	original, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal("read tracked fixture candidate")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatal("stat tracked fixture candidate")
	}
	writeCandidate := func(content []byte) {
		t.Helper()
		if err := os.WriteFile(filePath, content, info.Mode().Perm()); err != nil {
			t.Fatal("write isolated fixture candidate")
		}
	}
	git := func(args ...string) error {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", worktree}, args...)...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %s failed", args[0])
		}
		return nil
	}
	headHash := func() string {
		t.Helper()
		output, err := exec.Command("git", "-C", worktree, "rev-parse", "HEAD").Output()
		if err != nil {
			t.Fatal("read disposable worktree HEAD")
		}
		return strings.TrimSpace(string(output))
	}
	gitDiffExit := func(args ...string) int {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", worktree, "diff"}, args...)...)
		err := cmd.Run()
		if err == nil {
			return 0
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		t.Fatalf("git diff %s could not verify candidate state", args[0])
		return -1
	}

	secretValue := []byte("AWS_ACCESS_KEY_ID=AKIA" + "1234567890ABCDEF\n")
	stagedSecret := append(append([]byte(nil), original...), []byte("\n"+string(secretValue))...)
	writeCandidate(stagedSecret)
	if err := git("add", "--", candidatePath); err != nil {
		t.Fatal(err)
	}
	initialHead := headHash()
	writeCandidate(original)
	if got := gitDiffExit("HEAD", "--quiet", "--", candidatePath); got != 0 {
		t.Fatalf("worktree should match tracked HEAD in staged-only case; diff exit %d", got)
	}

	files, err := (&ExecGitRunner{}).ReadStagedCandidate(context.Background(), worktree)
	if err != nil {
		t.Fatalf("read exact isolated index candidate: %v", err)
	}
	if !bytes.Equal(files[candidatePath], stagedSecret) {
		t.Fatal("candidate gate reader did not return the staged Vrooli blob")
	}
	baseURL := os.Getenv("SECURITY_HEALTH_API_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:17104"
	}
	scanner := NewCandidateSecurityClient(15 * time.Second)
	scanner.resolveURL = func(context.Context) (string, error) { return baseURL, nil }
	stagedResult := scanStagedCandidate(context.Background(), &ExecGitRunner{}, worktree, scanner)
	if stagedResult.Status != "failed" || !strings.Contains(stagedResult.Stdout, candidatePath) {
		t.Fatalf("Security Health did not detect the staged-only Vrooli candidate: status=%s", stagedResult.Status)
	}
	if strings.Contains(stagedResult.Stdout+stagedResult.Stderr+stagedResult.Summary, string(secretValue)) {
		t.Fatal("staged-only candidate value leaked from the redacted GCT result")
	}
	blockedCommit, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: &ExecGitRunner{}, RepoDir: worktree, CandidateSecurity: scanner,
	}, CommitRequest{Message: "fix: block staged secret in disposable candidate", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatalf("CreateCommit staged-secret admission: %v", err)
	}
	if blockedCommit.Success || blockedCommit.Precommit == nil || blockedCommit.Precommit.Status != "failed" {
		t.Fatalf("CreateCommit admitted staged-only secret: %+v", blockedCommit)
	}
	if strings.Contains(blockedCommit.Error+blockedCommit.Precommit.Stdout+blockedCommit.Precommit.Stderr+blockedCommit.Precommit.Summary, string(secretValue)) {
		t.Fatal("staged-only candidate value leaked from CreateCommit response")
	}
	if headHash() != initialHead {
		t.Fatal("blocked candidate unexpectedly changed fixture HEAD")
	}

	// Inverse mismatch: stage a benign tracked-file change, then place the
	// sample only in the working tree. CreateCommit must scan and commit the
	// staged bytes while ignoring the unstaged secret.
	benignStaged := append(append([]byte(nil), original...), []byte("\nRH2 disposable benign candidate\n")...)
	writeCandidate(benignStaged)
	if err := git("add", "--", candidatePath); err != nil {
		t.Fatal(err)
	}
	worktreeOnlySecret := append(append([]byte(nil), benignStaged...), secretValue...)
	writeCandidate(worktreeOnlySecret)
	if got := gitDiffExit("--cached", "--quiet", "--", candidatePath); got != 1 {
		t.Fatalf("inverse fixture should contain a staged benign change; cached diff exit %d", got)
	}
	if got := gitDiffExit("--quiet", "--", candidatePath); got != 1 {
		t.Fatalf("inverse sample should exist only in worktree; diff exit %d", got)
	}
	inverseFiles, err := (&ExecGitRunner{}).ReadStagedCandidate(context.Background(), worktree)
	if err != nil || !bytes.Equal(inverseFiles[candidatePath], benignStaged) {
		t.Fatalf("GCT reader included non-index candidate bytes (err=%v, files=%d, candidate bytes=%d)", err, len(inverseFiles), len(inverseFiles[candidatePath]))
	}
	inverse := scanStagedCandidate(context.Background(), &ExecGitRunner{}, worktree, scanner)
	if inverse.Status != "passed" || strings.Contains(inverse.Stdout+inverse.Stderr+inverse.Summary, string(secretValue)) {
		t.Fatalf("worktree-only sample affected exact staged scan: status=%s", inverse.Status)
	}
	benignCommit, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: &ExecGitRunner{}, RepoDir: worktree, CandidateSecurity: scanner,
	}, CommitRequest{Message: "fix: commit benign staged candidate", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatalf("CreateCommit inverse mismatch admission: %v", err)
	}
	if !benignCommit.Success || benignCommit.Hash == "" {
		t.Fatalf("CreateCommit rejected benign staged bytes with worktree-only secret: %+v", benignCommit)
	}
	committedBytes, err := (&ExecGitRunner{}).ShowFileAtCommit(context.Background(), worktree, benignCommit.Hash, candidatePath)
	if err != nil || !bytes.Equal(committedBytes, benignStaged) {
		t.Fatalf("benign CreateCommit did not write only staged bytes (err=%v)", err)
	}
	if !bytes.Contains(mustReadFile(t, filePath), secretValue) {
		t.Fatal("inverse mismatch fixture no longer has the secret only in the working tree")
	}

	// A transport outage is incomplete coverage, never a green result. The
	// Security Health handler's missing-binary case is covered in its owner tests.
	writeCandidate(append(append([]byte(nil), benignStaged...), []byte("another benign staged change\n")...))
	if err := git("add", "--", candidatePath); err != nil {
		t.Fatal(err)
	}
	writeCandidate(original)
	unavailable := NewCandidateSecurityClient(time.Second)
	unavailable.resolveURL = func(context.Context) (string, error) { return "http://127.0.0.1:0", nil }
	missing := scanStagedCandidate(context.Background(), &ExecGitRunner{}, worktree, unavailable)
	if missing.Status != "incomplete" || strings.Contains(missing.Stdout+missing.Stderr+missing.Summary, string(secretValue)) {
		t.Fatalf("unavailable Security Health coverage was not incomplete and redacted: status=%s", missing.Status)
	}
	blockedUnavailableCommit, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: &ExecGitRunner{}, RepoDir: worktree, CandidateSecurity: unavailable,
	}, CommitRequest{Message: "fix: refuse commit without candidate scanner", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatalf("CreateCommit unavailable scanner admission: %v", err)
	}
	if blockedUnavailableCommit.Success || blockedUnavailableCommit.Precommit == nil || blockedUnavailableCommit.Precommit.Status != "incomplete" {
		t.Fatalf("CreateCommit admitted candidate with unavailable scanner: %+v", blockedUnavailableCommit)
	}
	if headHash() != benignCommit.Hash {
		t.Fatal("unavailable scanner unexpectedly changed disposable fixture HEAD")
	}
	// Recovery: the same scanner client path is available again, and a new
	// exact staged candidate can be admitted after the incomplete result.
	recoveredStaged := append(append([]byte(nil), benignStaged...), []byte("RH2 scanner recovery candidate\n")...)
	writeCandidate(recoveredStaged)
	if err := git("add", "--", candidatePath); err != nil {
		t.Fatal(err)
	}
	writeCandidate(append(append([]byte(nil), recoveredStaged...), secretValue...))
	recoveredCommit, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: &ExecGitRunner{}, RepoDir: worktree, CandidateSecurity: scanner,
	}, CommitRequest{Message: "fix: recover candidate security admission", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatalf("CreateCommit after scanner recovery: %v", err)
	}
	if !recoveredCommit.Success || recoveredCommit.Hash == "" {
		t.Fatalf("candidate security did not recover after scanner outage: %+v", recoveredCommit)
	}
	recoveredBytes, err := (&ExecGitRunner{}).ShowFileAtCommit(context.Background(), worktree, recoveredCommit.Hash, candidatePath)
	if err != nil || !bytes.Equal(recoveredBytes, recoveredStaged) {
		t.Fatalf("recovered CreateCommit did not write only staged bytes (err=%v)", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read disposable candidate file: %v", err)
	}
	return content
}

func gitPath(t *testing.T, repo, path string) string {
	t.Helper()
	output, err := exec.Command("git", "-C", repo, "rev-parse", "--git-path", path).Output()
	if err != nil {
		t.Fatalf("resolve git path %s: %v", path, err)
	}
	resolved := strings.TrimSpace(string(output))
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(repo, resolved)
	}
	return filepath.Clean(resolved)
}

func TestScanStagedCandidateUnavailableIsIncomplete(t *testing.T) {
	git := NewFakeGitRunner().AddStagedFile("file.txt")
	git.Staged["file.txt"] = "candidate"
	for name, scanner := range map[string]CandidateSecurityScanner{
		"missing":    nil,
		"rpc error":  &candidateScannerStub{err: errors.New("scanner diagnostic may contain candidate bytes")},
		"incomplete": &candidateScannerStub{result: CandidateSecurityResult{FailureCode: "scanner_unavailable"}},
	} {
		t.Run(name, func(t *testing.T) {
			result := scanStagedCandidate(context.Background(), git, "/repo", scanner)
			if result.Status != "incomplete" || result.ExitCode == 0 {
				t.Fatalf("unavailable scanner result = %+v", result)
			}
			if strings.Contains(result.Stdout+result.Stderr+result.Summary, "candidate bytes") {
				t.Fatalf("scanner diagnostic leaked: %+v", result)
			}
		})
	}
	git.StagedCandidateError = errors.New("index read failed")
	result := scanStagedCandidate(context.Background(), git, "/repo", &candidateScannerStub{result: CandidateSecurityResult{Complete: true}})
	if result.Status != "incomplete" {
		t.Fatalf("index read error result = %+v", result)
	}
}

func TestCreateCommitSkipPrecommitCannotBypassCandidateSecurity(t *testing.T) {
	git := NewFakeGitRunner().AddStagedFile("file.txt")
	git.Staged["file.txt"] = "candidate"
	result, err := CreateCommit(authorizedHumanContext(), CommitDeps{
		Git: git, RepoDir: "/repo",
		CandidateSecurity: &candidateScannerStub{result: CandidateSecurityResult{FailureCode: "scanner_unavailable"}},
	}, CommitRequest{Message: "fix: secure candidate", SkipPrecommitOnce: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Precommit == nil || result.Precommit.Status != "incomplete" {
		t.Fatalf("candidate security was bypassed: %+v", result)
	}
	if git.AssertCalled("Commit") {
		t.Fatal("commit writer ran despite incomplete candidate security coverage")
	}
}

func TestCommitGatesRequireCandidateSecurityAndWorkspaceHealthIndependently(t *testing.T) {
	t.Run("complete benign security does not cover failed workspace health", func(t *testing.T) {
		runner := fakeCommandRunner{stdout: "workspace health failed", exitCode: 2}
		precommit := newTestPrecommitServiceWithRunner(t, runner)
		repo := t.TempDir()
		if _, err := precommit.Save(authorizedHumanContext(), repo, PrecommitConfig{
			Enabled: true, Command: "workspace-health", RunBeforeCommit: true,
		}); err != nil {
			t.Fatal(err)
		}
		git := NewFakeGitRunner().AddStagedFile("file.txt")
		git.Staged["file.txt"] = "candidate"
		result, err := CreateCommit(authorizedHumanContext(), CommitDeps{
			Git: git, RepoDir: repo, Precommit: precommit,
			CandidateSecurity: &candidateScannerStub{result: CandidateSecurityResult{Complete: true}},
		}, CommitRequest{Message: "fix: validate both gates"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Success || result.Precommit == nil || result.Precommit.Status != "failed" {
			t.Fatalf("workspace health failure was hidden by benign candidate security: %+v", result)
		}
		if git.AssertCalled("Commit") {
			t.Fatal("commit writer ran after workspace health failed")
		}
	})

	t.Run("workspace health success does not cover incomplete candidate security", func(t *testing.T) {
		runner := &countingCandidateRunner{}
		precommit := newTestPrecommitServiceWithRunner(t, runner)
		repo := t.TempDir()
		if _, err := precommit.Save(authorizedHumanContext(), repo, PrecommitConfig{
			Enabled: true, Command: "workspace-health", RunBeforeCommit: true,
		}); err != nil {
			t.Fatal(err)
		}
		git := NewFakeGitRunner().AddStagedFile("file.txt")
		git.Staged["file.txt"] = "candidate"
		result, err := CreateCommit(authorizedHumanContext(), CommitDeps{
			Git: git, RepoDir: repo, Precommit: precommit,
			CandidateSecurity: &candidateScannerStub{result: CandidateSecurityResult{FailureCode: "scanner_unavailable"}},
		}, CommitRequest{Message: "fix: validate both gates"})
		if err != nil {
			t.Fatal(err)
		}
		if result.Success || result.Precommit == nil || result.Precommit.Status != "incomplete" {
			t.Fatalf("workspace health success substituted for candidate security coverage: %+v", result)
		}
		if runner.runs.Load() != 0 {
			t.Fatal("workspace health ran after candidate-security coverage was already incomplete")
		}
		if git.AssertCalled("Commit") {
			t.Fatal("commit writer ran without candidate-security coverage")
		}
	})
}
