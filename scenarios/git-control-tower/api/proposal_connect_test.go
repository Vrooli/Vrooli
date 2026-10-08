package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"git-control-tower/internal/config"
	"git-control-tower/internal/policygate"
	"git-control-tower/internal/proposals"

	"github.com/vrooli/cli-core/cliutil"
	proposalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals"
	proposalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals/proposals_v1connect"
)

type proposalTransport struct {
	server  *Server
	sandbox *FakeWorkspaceSandboxAPI
	git     *proposals.MemoryGit
	client  proposalsconnect.ProposalServiceClient
	clock   *time.Time
}

// newProposalTransport serves ProposalService behind the production policy
// interceptor. The principal is injected only by this test server from a
// test header; production never trusts it. The repository is the in-memory
// git model, so no test mutates a real repository.
func newProposalTransport(t *testing.T) *proposalTransport {
	t.Helper()
	ctx := context.Background()
	store := newTestRepoStore(t)
	record, err := store.Upsert(ctx, RepoRecord{Path: "/fake/proposal-repo", Name: "proposal-repo"})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SetActive(ctx, record.ID); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", "file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	proposalStore, err := proposals.NewStore(db)
	if err != nil {
		t.Fatal(err)
	}
	memory := proposals.NewMemoryGit(map[string]string{"a.go": "a1", "b.go": "b1"})
	previous := proposalGitFor
	proposalGitFor = func(GitRunner, string) proposals.Git { return memory }
	t.Cleanup(func() { proposalGitFor = previous })

	clock := time.Now().UTC()
	fakeGit := NewFakeGitRunner()
	sandbox := NewFakeWorkspaceSandboxAPI()
	server := &Server{
		git: fakeGit, repos: NewRepoService(store, fakeGit), repoLock: NewRepoLock(), audit: &NoOpAuditLogger{}, sandbox: sandbox,
		intentService:   policygate.NewIntentService(policygate.NewMemoryIntentStore()).WithClock(func() time.Time { return clock }),
		proposalService: proposals.NewService(proposalStore),
	}
	path, handler := proposalsconnect.NewProposalServiceHandler(proposalConnectServer{server: server}, connect.WithInterceptors(policygate.NewInterceptor(config.PolicyConfig{}, nil)))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if kind := r.Header.Get("X-Test-Principal"); kind != "" {
			principal := policygate.Principal{Subject: "test-" + kind, Kind: cliutil.CallerKindHuman, Verified: true}
			if kind == "agent" {
				principal.Kind = cliutil.CallerKindVrooliAgent
			}
			ctx = policygate.WithPrincipal(ctx, principal)
		}
		mux.ServeHTTP(w, r.WithContext(ctx))
	}))
	t.Cleanup(httpServer.Close)
	return &proposalTransport{server: server, sandbox: sandbox, git: memory, client: proposalsconnect.NewProposalServiceClient(httpServer.Client(), httpServer.URL), clock: &clock}
}

func asPrincipal[T any](msg *T, kind string) *connect.Request[T] {
	req := connect.NewRequest(msg)
	if kind != "" {
		req.Header().Set("X-Test-Principal", kind)
	}
	return req
}

// createViaAgent creates a proposal the way an orchestrator does.
func (p *proposalTransport) createViaAgent(t *testing.T) *proposalsv1.Proposal {
	t.Helper()
	p.git.Write("a.go", "a2")
	resp, err := p.client.CreateProposal(context.Background(), asPrincipal(&proposalsv1.CreateProposalRequest{
		Work:    &proposalsv1.ProposalWork{EffortRef: "effort:bas", Epoch: "E27", RunIds: []string{"70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"}},
		Subject: "bas: unify timeline (E27)", Paths: []string{"a.go"}, CreatorRunId: "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb",
	}, "agent"))
	if err != nil {
		t.Fatalf("agent create: %v", err)
	}
	return resp.Msg.GetProposal()
}

// issueApplyIntent issues a human intent for the proposal's current
// revision, exactly as ConfirmMutation does.
func (p *proposalTransport) issueApplyIntent(t *testing.T, proposal *proposalsv1.Proposal) string {
	t.Helper()
	human := policygate.WithPrincipal(context.Background(), policygate.Principal{Subject: "test-human", Kind: cliutil.CallerKindHuman, Verified: true})
	preview, err := p.server.prepareMutationWithContext(human, "", mutationOperationApplyProposal, proposals.SubjectContext(proposal.GetId(), int(proposal.GetRevision())))
	if err != nil {
		t.Fatal(err)
	}
	principal, _ := policygate.PrincipalFromContext(human)
	intent, err := p.server.intentService.Issue(human, principal, policygate.IntentRequest{RepositoryID: preview.RepositoryID, Operation: mutationOperationApplyProposal, ExpectedRevision: preview.ExpectedRevision, SubjectDigest: preview.SubjectDigest})
	if err != nil {
		t.Fatal(err)
	}
	return intent.ID
}

func (p *proposalTransport) apply(kind, intentID string, proposal *proposalsv1.Proposal) (*proposalsv1.ApplyProposalResponse, error) {
	resp, err := p.client.ApplyProposal(context.Background(), asPrincipal(&proposalsv1.ApplyProposalRequest{IntentId: intentID, Id: proposal.GetId(), Revision: proposal.GetRevision()}, kind))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func TestProposalDraftsAreAgentCallableAndCarryAttribution(t *testing.T) {
	p := newProposalTransport(t)
	proposal := p.createViaAgent(t)
	if proposal.GetState() != proposalsv1.ProposalState_PROPOSAL_STATE_OPEN || proposal.GetCreatedBy().GetKind() != "vrooli-agent" || proposal.GetCreatedBy().GetRunId() == "" {
		t.Fatalf("proposal = %v", proposal)
	}
	var proposalTrailer *proposalsv1.ProposalTrailer
	for _, trailer := range proposal.GetMessage().GetTrailers() {
		if trailer.GetKey() == "Vrooli-Proposal" {
			proposalTrailer = trailer
		}
	}
	if proposalTrailer == nil || proposalTrailer.GetResolution() != proposalsv1.TrailerResolution_TRAILER_RESOLUTION_RESOLVED {
		t.Fatalf("proposal trailer = %v", proposalTrailer)
	}
	list, err := p.client.ListProposals(context.Background(), asPrincipal(&proposalsv1.ListProposalsRequest{}, ""))
	if err != nil || list.Msg.GetOpenCount() != 1 || list.Msg.GetProposals()[0].GetFreshness().GetState() != proposalsv1.FreshnessState_FRESHNESS_STATE_FRESH {
		t.Fatalf("list = %v err=%v", list, err)
	}
	// An agent driving a session verified as the OS user is still an agent
	// writer: its trailers get agent validation.
	selfDeclared := asPrincipal(&proposalsv1.CreateProposalRequest{
		Work: &proposalsv1.ProposalWork{EffortRef: "effort:bas", Epoch: "E28"}, Subject: "bas: next (E28)", Paths: []string{"a.go"},
		Trailers: []*proposalsv1.ProposalTrailer{{Key: "Co-Authored-By", Value: "Agent <agent@example.com>"}},
	}, "human")
	selfDeclared.Header().Set(cliutil.HeaderCaller, cliutil.CallerKindExternalAgent.String())
	if _, err := p.client.CreateProposal(context.Background(), selfDeclared); connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "Vrooli-Run") {
		t.Fatalf("self-declared agent co-author err = %v", err)
	}
	if _, err := p.client.WithdrawProposal(context.Background(), asPrincipal(&proposalsv1.WithdrawProposalRequest{Id: proposal.GetId(), Reason: "re-plan"}, "agent")); err != nil {
		t.Fatalf("agent withdraw: %v", err)
	}
}

// GCT-001 and GCT-003: no principal, a forged human caller header and an
// agent principal never reach the writer, even with a valid human intent.
func TestApplyProposalRefusesNonHumansBeforeTheWriter(t *testing.T) {
	p := newProposalTransport(t)
	proposal := p.createViaAgent(t)
	intentID := p.issueApplyIntent(t, proposal)

	if _, err := p.apply("", intentID, proposal); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous apply err = %v", err)
	}
	forged := asPrincipal(&proposalsv1.ApplyProposalRequest{IntentId: intentID, Id: proposal.GetId(), Revision: proposal.GetRevision()}, "")
	forged.Header().Set(cliutil.HeaderCaller, "human")
	forged.Header().Set(policygate.HeaderAuthorized, "true")
	if _, err := p.client.ApplyProposal(context.Background(), forged); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("forged header apply err = %v", err)
	}
	if _, err := p.apply("agent", intentID, proposal); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent apply err = %v", err)
	}
	// A session personal-local authentication verifies as the OS user, but
	// whose client identifies as an agent, cannot approve either.
	selfDeclaredAgent := asPrincipal(&proposalsv1.ApplyProposalRequest{IntentId: intentID, Id: proposal.GetId(), Revision: proposal.GetRevision()}, "human")
	selfDeclaredAgent.Header().Set(cliutil.HeaderCaller, cliutil.CallerKindVrooliAgent.String())
	if _, err := p.client.ApplyProposal(context.Background(), selfDeclaredAgent); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent-driven human session apply err = %v", err)
	}
	if _, err := p.apply("human", "", proposal); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("intent-less apply err = %v", err)
	}
	if len(p.git.StageCalls()) != 0 {
		t.Fatal("a refused caller reached the writer")
	}
	// The intent was never consumed, so the human can still use it. The
	// recording commit writer needs a staged entry to report a commit.
	p.server.git.(*FakeGitRunner).AddStagedFile("a.go")
	intentID = p.issueApplyIntent(t, proposal)
	result, err := p.apply("human", intentID, proposal)
	if err != nil || !result.GetSuccess() || result.GetProposal().GetState() != proposalsv1.ProposalState_PROPOSAL_STATE_COMMITTED {
		t.Fatalf("human apply = %v err=%v", result, err)
	}
	// Workspace Sandbox learns the exact committed paths (asynchronously).
	deadline := time.Now().Add(2 * time.Second)
	for len(p.sandbox.MarkCommittedCalls()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if calls := p.sandbox.MarkCommittedCalls(); len(calls) != 1 || strings.Join(calls[0].FilePaths, ",") != "a.go" || calls[0].CommitHash != result.GetCommitOid() {
		t.Fatalf("mark committed calls = %#v", calls)
	}
}

// GCT-004 and GCT-005: an expired intent is refused, a consumed intent is
// never a second logical action, and an intent for other content is stale.
func TestApplyProposalIntentIsExactSingleUseAndExpires(t *testing.T) {
	p := newProposalTransport(t)
	proposal := p.createViaAgent(t)

	expired := p.issueApplyIntent(t, proposal)
	*p.clock = p.clock.Add(3 * time.Minute)
	if _, err := p.apply("human", expired, proposal); connect.CodeOf(err) != connect.CodeAborted || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired intent err = %v", err)
	}

	stale := p.issueApplyIntent(t, proposal)
	p.git.Write("a.go", "changed after review")
	if _, err := p.apply("human", stale, proposal); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("stale intent err = %v", err)
	}
	p.git.Write("a.go", "a2")

	p.git.Write("b.go", "operator work")
	p.git.StageDirect("b.go")
	once := p.issueApplyIntent(t, proposal)
	first, err := p.apply("human", once, proposal)
	if err != nil || first.GetRefusal().GetCode() != proposals.RefuseForeignStaged || first.GetRefusal().GetPaths()[0] != "b.go" {
		t.Fatalf("first use = %v err=%v", first, err)
	}
	if _, err := p.apply("human", once, proposal); connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("replayed intent err = %v", err)
	}
	if len(p.git.StageCalls()) != 0 || p.git.CommitCount() != 1 {
		t.Fatal("refused applies changed the repository")
	}
}
