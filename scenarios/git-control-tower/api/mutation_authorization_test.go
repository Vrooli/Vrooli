package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"git-control-tower/internal/policygate"
	"github.com/vrooli/cli-core/cliutil"
)

func TestRequireHumanMutationFailsClosedForDirectServiceCalls(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "missing principal", ctx: context.Background(), want: false},
		{
			name: "verified agent is not human authority",
			ctx: policygate.WithPrincipal(context.Background(), policygate.Principal{
				Kind: cliutil.CallerKindVrooliAgent, Subject: "agent-1", Verified: true,
			}),
			want: false,
		},
		{
			name: "verified human without consumed intent",
			ctx: policygate.WithPrincipal(context.Background(), policygate.Principal{
				Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true,
			}),
			want: false,
		},
		{name: "verified human with consumed intent", ctx: authorizedHumanContext(), want: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := requireHumanMutation(tc.ctx, "test writer")
			if (err == nil) != tc.want {
				t.Fatalf("requireHumanMutation error=%v, want authorized=%v", err, tc.want)
			}
		})
	}
}

func TestStageFilesDoesNotReachGitWithoutHumanPrincipal(t *testing.T) {
	fake := NewFakeGitRunner()
	_, err := StageFiles(context.Background(), StagingDeps{Git: fake, RepoDir: "/fake/repo"}, StageRequest{Paths: []string{"file.txt"}})
	if err == nil {
		t.Fatal("direct stage service call without a verified human must fail")
	}
	if fake.AssertCalled("Stage") {
		t.Fatal("denied direct service call reached the Git writer")
	}
}

func TestWriterIntentCannotAuthorizeDifferentOperation(t *testing.T) {
	issuedAt := time.Now().UTC()
	ctx := policygate.WithPrincipal(context.Background(), policygate.Principal{Kind: cliutil.CallerKindHuman, Subject: "operator-1", Verified: true})
	consumedAt := issuedAt
	ctx = policygate.WithIntent(ctx, policygate.HumanIntent{ID: "commit-intent", PrincipalID: "operator-1", Operation: mutationOperationCommit, ExpiresAt: issuedAt.Add(time.Minute), ConsumedAt: &consumedAt, Consumed: true})
	if err := requireHumanMutation(ctx, "stage files"); err == nil {
		t.Fatal("expected a commit intent to be rejected for stage")
	}
}

func TestCredentialAndRemoteWritersFailClosedWithoutHumanPrincipal(t *testing.T) {
	if _, err := SaveCredential(context.Background(), CredentialsDeps{}, CredentialSaveRequest{}); err == nil {
		t.Fatal("credential save without a verified human must fail")
	}
	if _, err := DeleteCredential(context.Background(), CredentialsDeps{}, CredentialDeleteRequest{ID: "credential-1"}); err == nil {
		t.Fatal("credential delete without a verified human must fail")
	}
	if _, err := UpdateRemoteURL(context.Background(), CredentialsDeps{}, RemoteURLUpdateRequest{}); err == nil {
		t.Fatal("remote URL update without a verified human must fail")
	}
}

func TestUntrackBinaryFailsClosedWithoutHumanPrincipal(t *testing.T) {
	_, err := UntrackBinary(context.Background(), HealthDeps{}, NewFakeGitRunner(), UntrackBinaryRequest{Path: "dist/app.bin"})
	if err == nil {
		t.Fatal("binary untrack without a verified human must fail before writing ignore state")
	}
}

func TestMutationPreviewSkipsPresentationWorkAndBindsFreshIndex(t *testing.T) {
	ctx := context.Background()
	fake := NewFakeGitRunner()
	fake.Branch.OID = strings.Repeat("a", 40)
	fake.Staged["source.go"] = "first staged content"
	fake.Unstaged["unrelated.go"] = "unrelated work"
	store := newTestRepoStore(t)
	record, err := store.Upsert(ctx, RepoRecord{Path: t.TempDir(), Name: "preview"})
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{git: fake, repos: NewRepoService(store, fake)}
	preview, err := s.prepareMutation(ctx, repositoryIDFor(&record), "repo.stage")
	if err != nil {
		t.Fatal(err)
	}
	if preview.ExpectedRevision != fake.Branch.OID || preview.FileCount != 1 || preview.StagedFiles[0] != "source.go" {
		t.Fatalf("lost authority subject: %+v", preview)
	}
	for _, method := range []string{"DiffNumstat", "ConfigGet", "LogFileFrequency"} {
		if fake.AssertCalled(method) {
			t.Errorf("authorization performed presentation-only work: %s", method)
		}
	}
	diff, err := fake.Diff(ctx, record.Path, "", true)
	if err != nil {
		t.Fatal(err)
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(append([]byte(fake.Branch.OID+"\x00\x00"), diff...)))
	if preview.SubjectDigest != expected {
		t.Fatal("authorization digest changed")
	}
	fake.Staged["source.go"] = "new staged content"
	changed, err := s.prepareMutation(ctx, repositoryIDFor(&record), "repo.stage")
	if err != nil {
		t.Fatal(err)
	}
	if preview.SubjectDigest == changed.SubjectDigest {
		t.Fatal("stale index was accepted")
	}
	fake.StatusError = fmt.Errorf("status unavailable")
	if _, err := s.prepareMutation(ctx, repositoryIDFor(&record), "repo.stage"); err == nil {
		t.Fatal("failed status became a preview")
	}
}

// Opt-in, read-only profiling of an existing checkout. Normal tests never read
// the live repository; callers must explicitly provide the measurement target.
func BenchmarkMutationStatusSnapshot(b *testing.B) {
	repo := os.Getenv("GCT_PERF_READONLY_REPO")
	if repo == "" {
		b.Skip("set GCT_PERF_READONLY_REPO for read-only profiling")
	}
	git := &ExecGitRunner{}
	b.Run("presentation-status", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := GetRepoStatus(context.Background(), RepoStatusDeps{Git: git, RepoDir: repo, StatusCache: NewRepoStatusCache(0)}); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("authorization-snapshot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := readRepoStatusSnapshot(context.Background(), git, repo); err != nil {
				b.Fatal(err)
			}
		}
	})
}
