package proposal

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"

	humancontrolv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control"
	humancontrolconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/human_control/human_control_v1connect"
	proposalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals"
	proposalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals/proposals_v1connect"
)

// recordingServer records every call. CLI tests never reach a repository.
type recordingServer struct {
	proposalsconnect.UnimplementedProposalServiceHandler
	humancontrolconnect.UnimplementedHumanControlServiceHandler
	calls       []string
	proposal    *proposalsv1.Proposal
	canMutate   bool
	applyResult *proposalsv1.ApplyProposalResponse
	create      *proposalsv1.CreateProposalRequest
	prepare     *humancontrolv1.PrepareMutationRequest
	apply       *proposalsv1.ApplyProposalRequest
}

func (s *recordingServer) GetAuthorityStatus(context.Context, *connect.Request[humancontrolv1.GetAuthorityStatusRequest]) (*connect.Response[humancontrolv1.AuthorityStatus], error) {
	s.calls = append(s.calls, "authority")
	return connect.NewResponse(&humancontrolv1.AuthorityStatus{CanMutate: s.canMutate, Reason: "agent callers may inspect and prepare changes"}), nil
}

func (s *recordingServer) PrepareMutation(_ context.Context, req *connect.Request[humancontrolv1.PrepareMutationRequest]) (*connect.Response[humancontrolv1.MutationPreview], error) {
	s.calls, s.prepare = append(s.calls, "prepare"), req.Msg
	return connect.NewResponse(&humancontrolv1.MutationPreview{RepositoryId: "1", Operation: req.Msg.GetOperation(), ExpectedRevision: "head1", SubjectDigest: "digest1", SubjectContext: req.Msg.GetSubjectContext()}), nil
}

func (s *recordingServer) ConfirmMutation(_ context.Context, req *connect.Request[humancontrolv1.ConfirmMutationRequest]) (*connect.Response[humancontrolv1.MutationIntent], error) {
	s.calls = append(s.calls, "confirm")
	if req.Msg.GetSubjectDigest() != "digest1" || req.Msg.GetOperation() != applyOperation {
		return nil, connect.NewError(connect.CodeAborted, errors.New("preview mismatch"))
	}
	return connect.NewResponse(&humancontrolv1.MutationIntent{IntentId: "intent-1"}), nil
}

func (s *recordingServer) GetProposal(context.Context, *connect.Request[proposalsv1.GetProposalRequest]) (*connect.Response[proposalsv1.GetProposalResponse], error) {
	s.calls = append(s.calls, "get")
	return connect.NewResponse(&proposalsv1.GetProposalResponse{Proposal: s.proposal}), nil
}

func (s *recordingServer) CreateProposal(_ context.Context, req *connect.Request[proposalsv1.CreateProposalRequest]) (*connect.Response[proposalsv1.CreateProposalResponse], error) {
	s.calls, s.create = append(s.calls, "create"), req.Msg
	return connect.NewResponse(&proposalsv1.CreateProposalResponse{Proposal: s.proposal, Stored: true}), nil
}

func (s *recordingServer) ApplyProposal(_ context.Context, req *connect.Request[proposalsv1.ApplyProposalRequest]) (*connect.Response[proposalsv1.ApplyProposalResponse], error) {
	s.calls, s.apply = append(s.calls, "apply"), req.Msg
	return connect.NewResponse(s.applyResult), nil
}

func openProposal() *proposalsv1.Proposal {
	return &proposalsv1.Proposal{
		Id: "gctp-0123456789ab", RepositoryId: "1", Revision: 3, State: proposalsv1.ProposalState_PROPOSAL_STATE_OPEN,
		Work:      &proposalsv1.ProposalWork{EffortRef: "effort:bas", Epoch: "E27"},
		Freshness: &proposalsv1.ProposalFreshness{State: proposalsv1.FreshnessState_FRESHNESS_STATE_FRESH},
		Files:     []*proposalsv1.ProposalFile{{Path: "a.go", Kind: proposalsv1.FileChangeKind_FILE_CHANGE_KIND_MODIFIED, Flags: []*proposalsv1.ProposalFlag{{Code: "mixed_prior_uncommitted"}}}, {Path: "gone.go", Kind: proposalsv1.FileChangeKind_FILE_CHANGE_KIND_DELETED, Deleted: true}},
		Message:   &proposalsv1.ProposalMessage{Subject: "bas: unify timeline (E27)", Rendered: "bas: unify timeline (E27)\n\nVrooli-Epoch: effort:bas#E27"},
	}
}

func serve(t *testing.T, impl *recordingServer, input string) *bytes.Buffer {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(proposalsconnect.NewProposalServiceHandler(impl))
	mux.Handle(humancontrolconnect.NewHumanControlServiceHandler(impl))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	previousProposal, previousApply, previousHuman := proposalClientFactory, applyClientFactory, humanControlClientFactory
	previousInput, previousOutput, previousDetect := confirmInput, output, detectCaller
	t.Cleanup(func() {
		proposalClientFactory, applyClientFactory, humanControlClientFactory = previousProposal, previousApply, previousHuman
		confirmInput, output, detectCaller = previousInput, previousOutput, previousDetect
	})
	detectCaller = func() cliutil.CallerKind { return cliutil.CallerKindHuman }
	proposalClient := func(*cliapp.ScenarioApp) proposalsconnect.ProposalServiceClient {
		return proposalsconnect.NewProposalServiceClient(server.Client(), server.URL)
	}
	proposalClientFactory, applyClientFactory = proposalClient, proposalClient
	humanControlClientFactory = func(*cliapp.ScenarioApp) humancontrolconnect.HumanControlServiceClient {
		return humancontrolconnect.NewHumanControlServiceClient(server.Client(), server.URL)
	}
	var out bytes.Buffer
	confirmInput, output = strings.NewReader(input), &out
	return &out
}

func TestApproveShowsTheProposalAndRequiresConfirm(t *testing.T) {
	impl := &recordingServer{proposal: openProposal(), canMutate: true}
	out := serve(t, impl, "no\n")
	err := runApprove(nil, []string{"gctp-0123456789ab"})
	if err == nil || !strings.Contains(err.Error(), "not confirmed") {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(impl.calls, ",") != "authority,get,prepare" {
		t.Fatalf("calls = %v", impl.calls)
	}
	for _, want := range []string{"a.go", "mixed_prior_uncommitted", "D gone.go", "Vrooli-Epoch: effort:bas#E27"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("preview missing %q:\n%s", want, out.String())
		}
	}
}

func TestApproveConfirmsOneExactIntentAndAppliesOnce(t *testing.T) {
	impl := &recordingServer{proposal: openProposal(), canMutate: true, applyResult: &proposalsv1.ApplyProposalResponse{Success: true, CommitOid: "abc123", CommitVerified: true}}
	out := serve(t, impl, "confirm\n")
	if err := runApprove(nil, []string{"gctp-0123456789ab"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(impl.calls, ",") != "authority,get,prepare,confirm,apply" {
		t.Fatalf("calls = %v", impl.calls)
	}
	if impl.prepare.GetOperation() != applyOperation || impl.prepare.GetSubjectContext() != "proposal:gctp-0123456789ab@3" {
		t.Fatalf("prepare = %v", impl.prepare)
	}
	if impl.apply.GetIntentId() != "intent-1" || impl.apply.GetRevision() != 3 || impl.apply.GetId() != "gctp-0123456789ab" {
		t.Fatalf("apply = %v", impl.apply)
	}
	if !strings.Contains(out.String(), "Committed abc123") {
		t.Fatalf("output = %s", out.String())
	}
}

func TestApproveDriftRefusalListsEveryFile(t *testing.T) {
	impl := &recordingServer{proposal: openProposal(), canMutate: true, applyResult: &proposalsv1.ApplyProposalResponse{
		Refusal: &proposalsv1.ApplyRefusal{Code: "content_drift", Paths: []string{"a.go", "gone.go"}, Detail: "content changed since the proposal; refresh it or remove the files"},
	}}
	serve(t, impl, "")
	err := runApprove(nil, []string{"gctp-0123456789ab", "--yes"})
	if err == nil || !strings.Contains(err.Error(), "content_drift") || !strings.Contains(err.Error(), "a.go") || !strings.Contains(err.Error(), "gone.go") {
		t.Fatalf("err = %v", err)
	}
}

func TestApproveIsRefusedForAgentsBeforeAnyIntent(t *testing.T) {
	impl := &recordingServer{proposal: openProposal(), canMutate: false}
	serve(t, impl, "confirm\n")
	err := runApprove(nil, []string{"gctp-0123456789ab", "--yes"})
	if err == nil || !strings.Contains(err.Error(), "only the operator") || strings.Join(impl.calls, ",") != "authority" {
		t.Fatalf("err = %v calls = %v", err, impl.calls)
	}
}

func TestApproveRefusesAgentProcessesBeforeAnyCall(t *testing.T) {
	impl := &recordingServer{proposal: openProposal(), canMutate: true}
	serve(t, impl, "confirm\n")
	detectCaller = func() cliutil.CallerKind { return cliutil.CallerKindVrooliAgent }
	err := runApprove(nil, []string{"gctp-0123456789ab", "--yes"})
	if err == nil || !strings.Contains(err.Error(), "only the operator") || len(impl.calls) != 0 {
		t.Fatalf("err = %v calls = %v", err, impl.calls)
	}
}

func TestApproveRefusesStaleProposalWithoutAnIntent(t *testing.T) {
	stale := openProposal()
	stale.Freshness = &proposalsv1.ProposalFreshness{State: proposalsv1.FreshnessState_FRESHNESS_STATE_DRIFTED, DriftedPaths: []string{"a.go"}}
	impl := &recordingServer{proposal: stale, canMutate: true}
	serve(t, impl, "confirm\n")
	if err := runApprove(nil, []string{"gctp-0123456789ab", "--yes"}); err == nil || !strings.Contains(err.Error(), "refresh") {
		t.Fatalf("err = %v", err)
	}
	if strings.Contains(strings.Join(impl.calls, ","), "prepare") {
		t.Fatalf("calls = %v", impl.calls)
	}
}

func TestCreateMergesRequestFileFlagsAndCreatorRun(t *testing.T) {
	impl := &recordingServer{proposal: openProposal()}
	serve(t, impl, "")
	dir := t.TempDir()
	requestFile := filepath.Join(dir, "p.json")
	if err := os.WriteFile(requestFile, []byte(`{"work":{"effort_ref":"effort:bas","epoch":"E27"},"subject":"from file","paths":["a.go"],"evidence":{"epoch_file":"docs/epochs/E27.md"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	bodyFile := filepath.Join(dir, "body.txt")
	if err := os.WriteFile(bodyFile, []byte("Outcome.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(creatorRunEnv, "70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb")
	err := runCreate(nil, []string{"--request-file", requestFile, "--subject", "bas: from flag (E27)", "--body-file", bodyFile, "--run", "e8c322ca-72e7-434c-9d37-1d2cf853961f", "--trailer", "Vrooli-Plan=plan-1", "b.go", "--validate-only"})
	if err != nil {
		t.Fatal(err)
	}
	req := impl.create
	if req.GetSubject() != "bas: from flag (E27)" || req.GetBody() != "Outcome.\n" || strings.Join(req.GetPaths(), ",") != "a.go,b.go" || !req.GetValidateOnly() {
		t.Fatalf("request = %v", req)
	}
	if req.GetWork().GetEffortRef() != "effort:bas" || len(req.GetWork().GetRunIds()) != 1 || req.GetCreatorRunId() == "" || req.GetEvidence().GetEpochFile() != "docs/epochs/E27.md" {
		t.Fatalf("request = %v", req)
	}
	if len(req.GetTrailers()) != 1 || req.GetTrailers()[0].GetKey() != "Vrooli-Plan" {
		t.Fatalf("trailers = %v", req.GetTrailers())
	}
}

func TestManifestCoversProposalService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, proposalsv1.File_git_control_tower_v1_proposals_proposals_proto, "ProposalService")
}
