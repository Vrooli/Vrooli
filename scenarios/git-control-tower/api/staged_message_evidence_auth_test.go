// [REQ:GCT-OT-P1-002]
package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"git-control-tower/internal/config"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/reviewjobs"
	"git-control-tower/internal/stagedmessages"
	"github.com/vrooli/api-core/authn"
	testdb "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	rpc "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory/advisory_v1connect"
)

type stagedEvidenceReadFixture struct{ reads atomic.Int32 }

func (*stagedEvidenceReadFixture) Capture(context.Context, string, string) (stagedmessages.Captured, error) {
	return stagedmessages.Captured{}, errors.New("evidence reads must not recollect the index")
}
func (*stagedEvidenceReadFixture) Identity(context.Context, string, string) (string, error) {
	return "", errors.New("evidence reads must not inspect the current index")
}
func (f *stagedEvidenceReadFixture) EvidenceFile(_ context.Context, _ string, file *v1.StagedMessageFile) (*v1.StagedMessageEvidenceFile, error) {
	f.reads.Add(1)
	return &v1.StagedMessageEvidenceFile{Path: file.Path, Patch: "@@ -1 +1 @@\n-original\n+immutable staged bytes\n", CompletePatch: true}, nil
}

// The mock credential authorities feed the canonical provenance/authentication
// middleware; request headers cannot directly create a principal or execution.
type stagedEvidenceHumanAuthority struct{}

func (stagedEvidenceHumanAuthority) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}
func (stagedEvidenceHumanAuthority) VerifyRequest(_ context.Context, req *http.Request) (identity.Principal, error) {
	switch req.Header.Get("Authorization") {
	case "Bearer verified-human":
		return identity.Principal{Kind: identity.ActorHuman, Subject: "fixture-operator", Source: identity.SourceScenarioAuthenticator, Verified: true, Scopes: []string{"git-control-tower:read"}, ExpiresAt: time.Now().Add(time.Hour)}, nil
	case "Bearer unverified-human":
		return identity.Principal{Kind: identity.ActorHuman, Subject: "unverified", Source: identity.SourceScenarioAuthenticator, Scopes: []string{"git-control-tower:read"}}, nil
	case "":
		return identity.Principal{}, identity.NewFailure(identity.FailureMissing, identity.SourceScenarioAuthenticator)
	default:
		return identity.Principal{}, identity.NewFailure(identity.FailureInvalid, identity.SourceScenarioAuthenticator)
	}
}

func TestStagedMessageEvidenceTransportRequiresVerifiedReaderAndOwningWorkflow(t *testing.T) {
	repos := newTestRepoStore(t)
	record, err := repos.Upsert(t.Context(), RepoRecord{Path: t.TempDir(), Name: "evidence-auth-fixture"})
	if err != nil {
		t.Fatal(err)
	}
	repo := repositoryIDFor(&record)
	store, err := reviewjobs.New(testdb.OpenSQLiteMemory(t))
	if err != nil {
		t.Fatal(err)
	}
	input := stagedmessages.DraftInput{FormatVersion: 3, JobID: "evidence-job", ContinuationOfJobID: "prior-success", Snapshot: &v1.StagedMessageSnapshot{RepositoryId: repo, Digest: "immutable-subject", Files: []*v1.StagedMessageFile{{Path: "already-covered.go"}, {Path: "remaining.go"}}}, PriorResult: &v1.StagedMessageSuggestion{Title: "Previous partial draft", Coverage: &v1.StagedMessageCoverage{Partial: true, CoveredPaths: []string{"already-covered.go"}, UncoveredPaths: []string{"remaining.go"}}}}
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	before, _, err := store.Create(t.Context(), reviewjobs.Job{ID: input.JobID, IdempotencyKey: "evidence-request", InputDigest: "immutable-input-digest", Operation: "staged-message", RepositoryID: repo, State: reviewjobs.Running, ExecutionID: "own-workflow", InputJSON: string(raw)})
	if err != nil {
		t.Fatal(err)
	}
	reader := &stagedEvidenceReadFixture{}
	s := &Server{repos: NewRepoService(repos, NewFakeGitRunner()), stagedMessages: stagedmessages.New(store, reader, nil, nil)}
	path, handler := rpc.NewAdvisoryServiceHandler(advisoryConnectServer{server: s}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{AgentAccess: config.AgentAccessDeny}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	verifier := provenance.VerifierFunc(func(token string) (*cliutil.VerifyResult, error) {
		claims := &cliutil.VerifiedClaims{RunID: "child-" + token, Subject: "fixture-operator", Scopes: []string{"git-control-tower:read"}, IssuedAt: time.Now().Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix(), Meta: map[string]string{"workflowExecutionId": "own-workflow"}}
		switch token {
		case "verified-own-child":
		case "verified-foreign-child":
			claims.Meta["workflowExecutionId"] = "foreign-workflow"
		case "verified-unbound-child":
			delete(claims.Meta, "workflowExecutionId")
		case "verified-without-read":
			claims.Scopes = nil
		default:
			return &cliutil.VerifyResult{Valid: false}, nil
		}
		return &cliutil.VerifyResult{Valid: true, Claims: claims}, nil
	})
	server := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(authn.Config{Providers: []authn.Provider{stagedEvidenceHumanAuthority{}}})(mux)))
	defer server.Close()
	client := rpc.NewAdvisoryServiceClient(server.Client(), server.URL)
	read := func(agent, human string) (*connect.Response[v1.ReadStagedMessageEvidenceResponse], error) {
		req := connect.NewRequest(&v1.ReadStagedMessageEvidenceRequest{RepositoryId: repo, JobId: input.JobID, SnapshotDigest: input.Snapshot.Digest, MaxBytes: 4096})
		if agent != "" {
			req.Header().Set(cliutil.HeaderAgentIdentityToken, agent)
		}
		if human != "" {
			req.Header().Set("Authorization", "Bearer "+human)
		}
		// Neither a caller label nor an unsigned execution coordinate is proof.
		req.Header().Set(cliutil.HeaderCaller, "human")
		req.Header().Set("X-Vrooli-Workflow-Execution-Id", "own-workflow")
		return client.ReadStagedMessageEvidence(t.Context(), req)
	}
	for _, test := range []struct{ name, agent, human string }{
		{"anonymous", "", ""}, {"invalid child token", "invalid-child", ""}, {"unverified human", "", "unverified-human"},
		{"foreign verified workflow", "verified-foreign-child", ""}, {"child without workflow binding", "verified-unbound-child", ""}, {"child without read capability", "verified-without-read", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := read(test.agent, test.human); connect.CodeOf(err) != connect.CodePermissionDenied {
				t.Fatalf("untrusted evidence reader reached domain: %v", err)
			}
		})
	}
	if reader.reads.Load() != 0 {
		t.Fatal("rejected reader reached immutable objects")
	}
	page, err := read("", "verified-human")
	if err != nil || len(page.Msg.Files) != 1 || page.Msg.Files[0].Path != "remaining.go" || !page.Msg.Files[0].CompletePatch {
		t.Fatalf("human read-only page: %v %v", page, err)
	}
	observed, err := store.Get(t.Context(), input.JobID)
	if err != nil || len(observed.Checks) != 0 {
		t.Fatalf("human read created agent coverage receipts: %v %v", observed.Checks, err)
	}
	page, err = read("verified-own-child", "")
	if err != nil || page.Msg.Receipt == "" || !page.Msg.Complete || len(page.Msg.Files) != 1 {
		t.Fatalf("own child read with only GCT read scope: %v %v", page, err)
	}
	observed, err = store.Get(t.Context(), input.JobID)
	if err != nil || len(observed.Checks) != 1 || observed.Checks[0].Name != page.Msg.Receipt || observed.Checks[0].Verdict != "immutable_staged_evidence" {
		t.Fatalf("own child receipt not retained: %v %v", observed.Checks, err)
	}
	if observed.State != before.State || observed.ExecutionID != before.ExecutionID || !observed.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("evidence receipt changed admission/cancellation revision")
	}
	eligible, err := s.stagedMessages.EvidenceEligibility(t.Context(), input.JobID, repo, input.Snapshot.Digest, "own-workflow")
	if err != nil || !eligible[2] || eligible[1] {
		t.Fatalf("wrong supplied-file coverage eligibility: %v %v", eligible, err)
	}
	foreign, err := s.stagedMessages.EvidenceEligibility(t.Context(), input.JobID, repo, input.Snapshot.Digest, "foreign-workflow")
	if err != nil || len(foreign) != 0 {
		t.Fatalf("receipt leaked across owner executions: %v %v", foreign, err)
	}
	if _, err = read("verified-own-child", ""); err != nil {
		t.Fatal(err)
	}
	observed, err = store.Get(t.Context(), input.JobID)
	if err != nil || len(observed.Checks) != 1 {
		t.Fatal("same page duplicated coverage receipts", err)
	}
	// Exercise the deployed personal-local provider chain, rather than only a
	// synthetic human provider: it must defer verified child authority to the
	// canonical provenance middleware without granting OS-human privileges.
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	t.Setenv("VROOLI_AUTH_PROVIDERS", "")
	t.Setenv("VROOLI_AUTH_OWNER_BINDINGS", "[]")
	deployed, err := buildAuthenticationConfig(nil)
	if err != nil {
		t.Fatal(err)
	}
	actual := httptest.NewServer(provenance.Middleware(verifier)(authn.Middleware(deployed)(mux)))
	defer actual.Close()
	client = rpc.NewAdvisoryServiceClient(actual.Client(), actual.URL)
	for _, token := range []string{"invalid-child", "verified-without-read", "verified-foreign-child", "verified-unbound-child"} {
		if _, err := read(token, ""); connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("personal-local promoted invalid or insufficient child %s: %v", token, err)
		}
	}
	if _, err := read("verified-own-child", ""); err != nil {
		t.Fatalf("deployed personal-local chain rejected the owning verified child: %v", err)
	}
	if _, err := read("verified-own-child", "invalid-browser-proof"); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("valid child bypassed invalid offered browser proof: %v", err)
	}

}
