// [REQ:GCT-PROPOSAL-CAPTURE]
package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"git-control-tower/internal/policygate"
	"git-control-tower/internal/proposals"
)

// These adapter fixtures follow the existing recovery tests: only t.TempDir
// repositories and in-memory SQL, never the operator repository or network.
func candidateFixture(t *testing.T) (*proposalGit, *proposals.Service, proposals.Repo, func(...string) string) {
	t.Helper()
	dir := t.TempDir()
	commands := safetyCommands{path: "git"}
	run := func(args ...string) string {
		t.Helper()
		out, err := commands.Run(context.Background(), dir, nil, args...)
		if err != nil {
			t.Fatalf("fixture %v: %v", args, err)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "--initial-branch=review", ".")
	run("config", "user.name", "Test Operator")
	run("config", "user.email", "operator@example.test")
	run("config", "commit.gpgsign", "false")
	run("config", "core.hooksPath", filepath.Join(dir, ".git", "hooks"))
	WriteTestFile(t, filepath.Join(dir, "a.txt"), "base\n")
	WriteTestFile(t, filepath.Join(dir, "keep.txt"), "keep\n")
	run("add", ".")
	run("commit", "-m", "Fixture base")
	g := newProposalGit(&ExecGitRunner{}, dir)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	store, err := proposals.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return g, proposals.NewService(store), proposals.Repo{ID: "fixture", Git: g}, run
}

func candidateProposal(t *testing.T, g *proposalGit, service *proposals.Service, repo proposals.Repo, paths ...string) proposals.View {
	t.Helper()
	result, err := service.Create(context.Background(), repo, proposals.CreateRequest{Subject: "Reviewed contents", Paths: paths, Actor: proposals.Actor{Subject: "fixture-agent", Kind: "vrooli-agent"}})
	if err != nil {
		t.Fatal(err)
	}
	return result.View
}

func candidateApplyContext() context.Context {
	now := time.Now()
	return policygate.WithIntent(authorizedHumanContextWithoutIntent(), policygate.HumanIntent{ID: "fixture-intent", PrincipalID: "test-operator", Operation: proposals.OperationApply, Consumed: true, ConsumedAt: &now, ExpiresAt: now.Add(time.Minute)})
}

func TestProposalCandidatePublishesExactCaptureAndPreservesOtherWork(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "staged\n")
	run("add", "a.txt")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "staged plus unstaged\n")
	WriteTestFile(t, filepath.Join(g.repoDir, "keep.txt"), "unrelated unfinished work\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	_, before, after, err := service.ReviewFile(context.Background(), repo, p.ID, p.Revision, "a.txt")
	if err != nil || string(before) != "base\n" || string(after) != "staged plus unstaged\n" {
		t.Fatalf("snapshot=%q -> %q err=%v", before, after, err)
	}
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
	})
	if err != nil || !result.Success || !result.Verified {
		t.Fatalf("apply=%+v err=%v", result, err)
	}
	if run("show", "HEAD:a.txt") != "staged plus unstaged" || run("show", "HEAD:keep.txt") != "keep" {
		t.Fatal("commit included unrelated work or omitted staged/unstaged contents")
	}
	bytes, err := os.ReadFile(filepath.Join(g.repoDir, "keep.txt"))
	if err != nil || string(bytes) != "unrelated unfinished work\n" {
		t.Fatal("publication rewrote operator files")
	}
}

func TestProposalCandidateHookRewriteCannotAdvanceSharedHead(t *testing.T) {
	for _, hook := range []string{"pre-commit", "commit-msg", "commit-msg-whitespace"} {
		t.Run(hook, func(t *testing.T) {
			g, service, repo, run := candidateFixture(t)
			WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "reviewed\n")
			p := candidateProposal(t, g, service, repo, "a.txt")
			head := run("rev-parse", "HEAD")
			body := "#!/bin/sh\nprintf 'hook rewrite\\n' > a.txt\ngit add a.txt\n"
			if hook == "commit-msg" {
				body = "#!/bin/sh\nprintf 'Changed message\\n' > \"$1\"\n"
			}
			hookPath := hook
			if hook == "commit-msg-whitespace" {
				hookPath = "commit-msg"
				body = "#!/bin/sh\nprintf '\\n' >> \"$1\"\n"
			}
			if err := os.WriteFile(filepath.Join(g.repoDir, ".git", "hooks", hookPath), []byte(body), 0700); err != nil {
				t.Fatal(err)
			}
			result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
				return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
			})
			if err != nil || result.Success || result.CommitOID != "" || run("rev-parse", "HEAD") != head || run("diff", "--cached", "--name-only") != "" {
				t.Fatalf("hook rewrite published or damaged index: %+v err=%v", result, err)
			}
		})
	}
}

func TestProposalCaptureBinaryDeletionModeSymlinkAndDriftRetention(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	if err := os.Remove(filepath.Join(g.repoDir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(g.repoDir, "keep.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(g.repoDir, "data.bin"), []byte{0, 1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("keep.txt", filepath.Join(g.repoDir, "link")); err != nil {
		t.Fatal(err)
	}
	p := candidateProposal(t, g, service, repo, "a.txt", "keep.txt", "data.bin", "link")
	for _, file := range p.Files {
		_, before, after, err := service.ReviewFile(context.Background(), repo, p.ID, p.Revision, file.Path)
		if err != nil {
			t.Fatal(err)
		}
		diff, err := capturedProposalDiff(context.Background(), file, before, after)
		if err != nil || diff.GetStats() == nil {
			t.Fatal(err)
		}
		switch file.Path {
		case "data.bin":
			if !diff.GetStats().GetIsBinary() {
				t.Fatal("binary diff misrepresented")
			}
		case "keep.txt":
			if file.Mode != "100755" || file.BaseMode != "100644" {
				t.Fatal("mode change lost")
			}
		case "link":
			if string(after) != "keep.txt" || file.Mode != "120000" {
				t.Fatal("symlink followed")
			}
		case "a.txt":
			if !diff.GetStats().GetIsDeletedFile() {
				t.Fatal("deletion lost")
			}
		}
	}
	WriteTestFile(t, filepath.Join(g.repoDir, "keep.txt"), "later change\n")
	_, _, after, err := service.ReviewFile(context.Background(), repo, p.ID, p.Revision, "keep.txt")
	if err != nil || string(after) != "keep\n" {
		t.Fatal("retained diff read live bytes")
	}
	if run("diff", "--cached", "--name-only") != "" {
		t.Fatal("capture staged files")
	}
}

func TestHistoryDetailsRetainBodyTrailersAndLiteralPaths(t *testing.T) {
	g, _, _, run := candidateFixture(t)
	path := "literal\nname.txt"
	WriteTestFile(t, filepath.Join(g.repoDir, path), "x\n")
	run("add", "--", path)
	message := "Work P2\n\nBody paragraph.\n\nVrooli-Effort: effort:example\nForeign-Key: preserved"
	run("commit", "-m", message)
	out, err := (&ExecGitRunner{}).LogDetails(context.Background(), g.repoDir, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	entries := parseHistoryDetails(out)
	if len(entries) != 1 || entries[0].Subject != "Work P2" || entries[0].Body != "Body paragraph." || len(entries[0].Trailers) != 2 || len(entries[0].Files) != 1 || entries[0].Files[0] != path {
		t.Fatalf("history lost metadata or path: %+v", entries)
	}
}

func TestProposalCandidateHonorsGitFilemodePolicy(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	run("config", "core.filemode", "false")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "reviewed\n")
	if err := os.Chmod(filepath.Join(g.repoDir, "a.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	p := candidateProposal(t, g, service, repo, "a.txt")
	if p.Files[0].Mode != "100644" {
		t.Fatal("capture ignored Git filemode policy")
	}
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
	})
	if err != nil || !result.Success {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !strings.HasPrefix(run("ls-tree", "HEAD", "a.txt"), "100644") {
		t.Fatal("published physical chmod instead of effective Git mode")
	}
}

func TestProposalExcludesIndexOnlyChangeWithoutAlteringStagedWork(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "staged work\n")
	run("add", "a.txt")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "base\n")
	WriteTestFile(t, filepath.Join(g.repoDir, "keep.txt"), "reviewed change\n")
	indexBefore := run("ls-files", "--stage")
	p := candidateProposal(t, g, service, repo, "a.txt", "keep.txt")
	if len(p.Files) != 1 || p.Files[0].Path != "keep.txt" || len(p.Excluded) != 1 || p.Excluded[0].Reason != "matches_commit_base" || indexBefore != run("ls-files", "--stage") {
		t.Fatalf("capture=%+v", p)
	}
	_, err := service.Create(context.Background(), repo, proposals.CreateRequest{Subject: "No contribution", Paths: []string{"a.txt"}, Actor: proposals.Actor{Kind: "vrooli-agent"}})
	if err == nil {
		t.Fatal("accepted a package that contributes no commit changes")
	}
	if indexBefore != run("ls-files", "--stage") {
		t.Fatal("no-op refusal changed staged work")
	}
}

func TestProposalRawByteDriftRefusedEvenWhenCanonicalBlobMatches(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, ".gitattributes"), "*.txt text eol=lf\n")
	run("add", ".gitattributes")
	run("commit", "-m", "Fixture text filter")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "reviewed\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	if err := os.WriteFile(filepath.Join(g.repoDir, "a.txt"), []byte("reviewed\r\n"), 0644); err != nil {
		t.Fatal(err)
	}
	hash, err := g.Hash(context.Background(), []string{"a.txt"})
	if err != nil || hash["a.txt"].BlobID != p.Files[0].BlobID {
		t.Fatalf("fixture not canonical-equal: %v", err)
	}
	view, err := service.Get(context.Background(), repo, p.ID)
	if err != nil || view.Freshness.State != proposals.FreshnessDrifted {
		t.Fatalf("freshness=%+v err=%v", view.Freshness, err)
	}
	head := run("rev-parse", "HEAD")
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(context.Context, string) (proposals.CommitOutcome, error) {
		t.Fatal("drift reached commit preparation")
		return proposals.CommitOutcome{}, nil
	})
	if err != nil || result.Success || run("rev-parse", "HEAD") != head {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestProposalAnchorRecognizesModeOnlyEpochContribution(t *testing.T) {
	g, service, repo, _ := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "older unfinished contents\n")
	anchor, err := service.Anchor(context.Background(), repo, proposals.AnchorRequest{EffortRef: "effort:fixture", Epoch: "E1", Scopes: []string{"*.txt"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(g.repoDir, "a.txt"), 0755); err != nil {
		t.Fatal(err)
	}
	p, err := service.Create(context.Background(), repo, proposals.CreateRequest{AnchorID: anchor.ID, Subject: "Executable entry", Paths: []string{"a.txt"}, Work: proposals.Work{EffortRef: "effort:fixture", Epoch: "E1"}, Actor: proposals.Actor{Kind: "vrooli-agent"}})
	if err != nil || len(p.View.Files) != 1 || p.View.Files[0].Mode != "100755" {
		t.Fatalf("mode-only contribution=%+v err=%v", p, err)
	}
	flagged := false
	for _, flag := range p.View.Files[0].Flags {
		if flag.Code == proposals.FlagMixedPrior {
			flagged = true
		}
	}
	if !flagged {
		t.Fatal("older text was not disclosed with mode-only change")
	}
}

func TestProposalConfiguredCheckRunsAgainstPrivateReviewedTree(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "reviewed\n")
	WriteTestFile(t, filepath.Join(g.repoDir, "keep.txt"), "unfinished unrelated work\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	checks := newTestPrecommitServiceWithRunner(t, ShellCommandRunner{})
	_, err := checks.Save(authorizedHumanContext(), g.repoDir, PrecommitConfig{Enabled: true, RunBeforeCommit: true, TimeoutSeconds: 30, Command: `test "$PWD" = "$VROOLI_ROOT" && test "$(cat a.txt)" = reviewed && test "$(cat keep.txt)" = keep && printf checked`})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_ROOT", g.repoDir)
	server := &Server{precommit: checks}
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, server.proposalCandidateCheck(g.repoDir, false), false)
	})
	if err != nil || !result.Success || result.Precommit == nil || result.Precommit.Status != "passed" || result.Precommit.Stdout != "checked" || result.Precommit.Timestamp.IsZero() {
		t.Fatalf("apply=%+v err=%v", result, err)
	}
	if run("show", "HEAD:keep.txt") != "keep" {
		t.Fatal("checks or commit adopted unrelated unfinished work")
	}
	data, err := os.ReadFile(filepath.Join(g.repoDir, "keep.txt"))
	if err != nil || string(data) != "unfinished unrelated work\n" {
		t.Fatal("checks rewrote operator files")
	}
}

func TestProposalPublicationSurvivesLostGitAcknowledgment(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "reviewed\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	wrapper := filepath.Join(t.TempDir(), "git-with-lost-ack")
	script := `#!/usr/bin/env python3
import os,sys,subprocess
if 'update-ref' not in sys.argv:
 os.execvp('git',['git']+sys.argv[1:])
child=subprocess.Popen(['git']+sys.argv[1:],stdin=subprocess.PIPE,stdout=subprocess.PIPE)
for line in sys.stdin.buffer:
 child.stdin.write(line);child.stdin.flush()
 if line.split()[0] in [b'start',b'prepare',b'commit']:
  reply=child.stdout.readline()
  if line.split()[0]==b'commit':
   child.stdin.close();child.wait();sys.exit(0)
  sys.stdout.buffer.write(reply);sys.stdout.buffer.flush()
`
	if err := os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	g.gitPath = wrapper
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
	})
	if err != nil || !result.Success || result.CommitOID == "" || run("show", "HEAD:a.txt") != "reviewed" || run("diff", "--cached", "--name-only") != "" {
		t.Fatalf("lost acknowledgment damaged committed index: %+v err=%v", result, err)
	}
	view, err := service.Get(context.Background(), repo, p.ID)
	if err != nil || view.State != proposals.StateCommitted {
		t.Fatalf("published commit not recorded: %+v err=%v", view, err)
	}
}

func TestProposalSavedVersionKeepsLaterWorkingChanges(t *testing.T) {
	for _, kind := range []string{"modify", "delete-then-recreate", "modify-then-delete", "add", "mode"} {
		t.Run(kind, func(t *testing.T) {
			g, service, repo, run := candidateFixture(t)
			path := "a.txt"
			if kind == "add" {
				path = "new.txt"
			}
			filePath := filepath.Join(g.repoDir, path)
			if kind == "delete-then-recreate" {
				if err := os.Remove(filePath); err != nil {
					t.Fatal(err)
				}
			} else {
				WriteTestFile(t, filePath, "saved proposal\n")
			}
			if kind == "mode" {
				if err := os.Chmod(filePath, 0755); err != nil {
					t.Fatal(err)
				}
			}
			p := candidateProposal(t, g, service, repo, path)
			if kind == "modify-then-delete" {
				if err := os.Remove(filePath); err != nil {
					t.Fatal(err)
				}
			} else {
				WriteTestFile(t, filePath, "later unfinished edits\n")
				if kind == "mode" {
					if err := os.Chmod(filePath, 0644); err != nil {
						t.Fatal(err)
					}
				}
			}
			result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true, KeepLaterChanges: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
				return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
			})
			if err != nil || !result.Success || !result.Verified {
				t.Fatalf("saved apply=%+v err=%v", result, err)
			}
			if kind == "delete-then-recreate" {
				if run("ls-tree", "HEAD", "--", path) != "" {
					t.Fatal("saved deletion not committed")
				}
			} else if run("show", "HEAD:"+path) != "saved proposal" {
				t.Fatal("later contents entered commit")
			}
			if kind == "mode" && !strings.HasPrefix(run("ls-tree", "HEAD", path), "100755") {
				t.Fatal("saved executable mode lost")
			}
			data, err := os.ReadFile(filePath)
			if kind == "modify-then-delete" {
				if !os.IsNotExist(err) {
					t.Fatal("later deletion reverted")
				}
			} else if err != nil || string(data) != "later unfinished edits\n" {
				t.Fatal("later working bytes were modified")
			}
			if kind == "mode" {
				info, err := os.Stat(filePath)
				if err != nil || info.Mode().Perm() != 0644 {
					t.Fatal("later mode reverted")
				}
			}
			if run("diff", "--cached", "--name-only") != "" {
				t.Fatal("candidate left staged changes")
			}
			if run("status", "--porcelain", "--", path) == "" {
				t.Fatal("later work no longer visible as uncommitted")
			}
		})
	}
}

func TestProposalSavedVersionAllowsConcurrentWorkingEditsDuringChecks(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	path := filepath.Join(g.repoDir, "a.txt")
	WriteTestFile(t, path, "saved proposal\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	check := func(_ context.Context, candidate string) (*proposals.PrecommitOutcome, error) {
		data, err := os.ReadFile(filepath.Join(candidate, "a.txt"))
		if err != nil || string(data) != "saved proposal\n" {
			t.Fatal("check did not inspect saved version")
		}
		WriteTestFile(t, path, "edited while checking\n")
		return &proposals.PrecommitOutcome{Status: "passed"}, nil
	}
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true, KeepLaterChanges: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, check, false)
	})
	if err != nil || !result.Success || run("show", "HEAD:a.txt") != "saved proposal" {
		t.Fatalf("apply=%+v err=%v", result, err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "edited while checking\n" {
		t.Fatal("concurrent working edits overwritten")
	}
}

func TestProposalSavedVersionPreservesDifferentStagedContents(t *testing.T) {
	for _, staging := range []string{"different", "matching", "deletion"} {
		t.Run(staging, func(t *testing.T) {
			g, service, repo, run := candidateFixture(t)
			path := filepath.Join(g.repoDir, "a.txt")
			WriteTestFile(t, path, "saved proposal\n")
			p := candidateProposal(t, g, service, repo, "a.txt")
			if staging == "different" {
				WriteTestFile(t, path, "staged intermediate version\n")
			}
			if staging == "deletion" {
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			run("add", "--", "a.txt")
			WriteTestFile(t, path, "later working contents\n")
			head, index := run("rev-parse", "HEAD"), run("ls-files", "--stage")
			view, err := service.Get(context.Background(), repo, p.ID)
			if err != nil {
				t.Fatal(err)
			}
			differentFlag := false
			for _, flag := range view.Files[0].Flags {
				if flag.Code == proposals.FlagStagedVersionDiffers {
					differentFlag = true
				}
			}
			if differentFlag != (staging != "matching") {
				t.Fatalf("different staged disclosure=%v", differentFlag)
			}
			result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true, KeepLaterChanges: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
				return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
			})
			if err != nil {
				t.Fatal(err)
			}
			if staging == "matching" {
				if !result.Success || run("show", "HEAD:a.txt") != "saved proposal" {
					t.Fatalf("matching staged candidate=%+v", result)
				}
			} else if result.Success || result.Refusal == nil || result.Refusal.Code != "staged_version_differs" || run("rev-parse", "HEAD") != head || run("ls-files", "--stage") != index {
				t.Fatalf("different staged version damaged=%+v", result)
			}
			data, _ := os.ReadFile(path)
			if string(data) != "later working contents\n" {
				t.Fatal("working contents changed")
			}
		})
	}
}

func TestProposalSavedVersionRefusesAlreadyCommittedProposalPathChanges(t *testing.T) {
	g, service, repo, run := candidateFixture(t)
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "saved proposal\n")
	p := candidateProposal(t, g, service, repo, "a.txt")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "newer committed work\n")
	run("add", "a.txt")
	run("commit", "-m", "Newer work")
	head := run("rev-parse", "HEAD")
	WriteTestFile(t, filepath.Join(g.repoDir, "a.txt"), "still newer edits\n")
	result, err := service.Apply(candidateApplyContext(), repo, proposals.ApplyRequest{ID: p.ID, Revision: p.Revision, AcknowledgePriorWork: true, KeepLaterChanges: true}, func(ctx context.Context, _ string) (proposals.CommitOutcome, error) {
		return service.PrepareCommit(ctx, repo, p.ID, p.Revision, nil, false)
	})
	if err != nil || result.Success || result.Refusal == nil || result.Refusal.Code != proposals.RefuseBaseMoved || run("rev-parse", "HEAD") != head {
		t.Fatalf("committed base changed=%+v err=%v", result, err)
	}
}
