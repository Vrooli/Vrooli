package main

import (
	"context"
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
