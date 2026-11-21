package eval_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	"search-hub/internal/control"
	"search-hub/internal/corpusgen"
	internaleval "search-hub/internal/eval"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/control"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
	evalconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval/eval_v1connect"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	handler "search-hub/handlers/eval"
)

// fakeResolver is a hand-written internaleval.ProviderResolver.
type fakeResolver struct {
	descs map[string]*registryv1.ProviderDescriptor
}

func (f fakeResolver) Get(_ context.Context, id string) (*registryv1.ProviderDescriptor, error) {
	if d, ok := f.descs[id]; ok {
		return d, nil
	}
	return nil, errors.New("no such provider")
}

// fakeGenerator is a hand-written handler.CorpusGenerator. The real generation is
// covered by internal/corpusgen; here we assert the handler's orchestration
// (preview vs apply, adequacy surfacing, error translation).
type fakeGenerator struct {
	res *corpusgen.Result
	err error
	got *evalv1.EvalSuite
}

func (f *fakeGenerator) Generate(_ context.Context, suite *evalv1.EvalSuite, _ *registryv1.ProviderDescriptor, _ corpusgen.Options) (*corpusgen.Result, error) {
	f.got = suite
	return f.res, f.err
}

// fakeCorpusControl is a hand-written handler.CorpusController. It records the
// corpus written back and (by default) echoes it as the effective corpus, the way
// a real provider does after a lossless file round-trip.
type fakeCorpusControl struct {
	gotCorpus *evalv1.EvalSuite
	gotToken  string
	written   bool
	err       error
}

func (f *fakeCorpusControl) WriteCorpus(_ context.Context, _ *registryv1.ProviderDescriptor, token string, corpus *evalv1.EvalSuite, _ bool) (*controlv1.WriteCorpusResponse, error) {
	f.gotCorpus = corpus
	f.gotToken = token
	if f.err != nil {
		return nil, f.err
	}
	return &controlv1.WriteCorpusResponse{Written: f.written, Effective: corpus}, nil
}

// fakeTokens is a hand-written handler.ControlTokenResolver.
type fakeTokens struct {
	token string
	err   error
}

func (f fakeTokens) Token(_ context.Context, _ string) (string, error) { return f.token, f.err }

func newGenerateClient(t *testing.T, store internaleval.Store, providers internaleval.ProviderResolver, gen handler.CorpusGenerator) evalconnect.EvalServiceClient {
	t.Helper()
	// Default apply wiring: a control plane that accepts + echoes the corpus and a
	// resolvable token, so `--apply` succeeds through WriteCorpus.
	return newGenerateClientFull(t, store, providers, gen, &fakeCorpusControl{written: true}, fakeTokens{token: "tok-123"})
}

func newGenerateClientFull(t *testing.T, store internaleval.Store, providers internaleval.ProviderResolver, gen handler.CorpusGenerator, ctrl handler.CorpusController, tokens handler.ControlTokenResolver) evalconnect.EvalServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := evalconnect.NewEvalServiceHandler(handler.NewConnectHandler(handler.Deps{
		Store: store, Providers: providers, Generator: gen, Control: ctrl, Tokens: tokens, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return evalconnect.NewEvalServiceClient(server.Client(), server.URL)
}

func genSuite() *evalv1.EvalSuite {
	return &evalv1.EvalSuite{
		SuiteId: "cli-health.commands.primary", ProviderId: "cli-health.commands", Name: "primary",
		Cases: []*evalv1.EvalCase{{CaseId: "c1", Query: "restart api", ExpectIds: []string{"a"}, ExpectWithinTopK: 3}},
	}
}

func genSuiteWithCandidates() *evalv1.EvalSuite {
	s := genSuite()
	s.Cases = append(s.Cases,
		&evalv1.EvalCase{CaseId: "gen-1", Query: "candidate one", Status: "candidate", ExpectIds: []string{"x"}, ExpectWithinTopK: 5},
		&evalv1.EvalCase{CaseId: "gen-2", Query: "candidate two", Status: "candidate", ExpectIds: []string{"y"}, ExpectWithinTopK: 5},
	)
	return s
}

func genProviders() fakeResolver {
	return fakeResolver{descs: map[string]*registryv1.ProviderDescriptor{
		"cli-health.commands": {ProviderId: "cli-health.commands", Type: "command", ProviderGroup: "cli-health"},
	}}
}

func oneProposal() *corpusgen.Result {
	return &corpusgen.Result{
		Sampled: 3, Inverted: 3, Deduped: 1, Strata: []string{"type:command"},
		Proposed: []corpusgen.Proposal{{
			Case: &evalv1.EvalCase{
				CaseId: "gen-abc", Query: "how to restart the service", Tags: []string{"generated", "type:command"},
				ExpectIds: []string{"svc"}, ExpectWithinTopK: 5,
			},
			SourceID: "svc", Stratum: "type:command",
		}},
	}
}

func TestGeneratePreviewDoesNotPersist(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	client := newGenerateClient(t, store, genProviders(), &fakeGenerator{res: oneProposal()})

	resp, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Count: 5,
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetProposed(), 1)
	require.Equal(t, "gen-abc", resp.Msg.GetProposed()[0].GetCase().GetCaseId())
	require.False(t, resp.Msg.GetApplied(), "preview must not apply")
	require.Nil(t, resp.Msg.GetSuite(), "preview returns no persisted suite")
	require.NotEmpty(t, resp.Msg.GetSummary())
	// The stored suite still has just its original case.
	stored, _ := store.GetSuite(context.Background(), "cli-health.commands.primary")
	require.Len(t, stored.GetCases(), 1, "the stored suite is untouched on preview")
	// Adequacy reflects the would-be (thin) corpus.
	require.NotEmpty(t, resp.Msg.GetAdequacy())
}

func TestGenerateApplyWritesFileThenMirrorsStore(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	ctrl := &fakeCorpusControl{written: true}
	client := newGenerateClientFull(t, store, genProviders(), &fakeGenerator{res: oneProposal()}, ctrl, fakeTokens{token: "tok-123"})

	resp, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Count: 5, Apply: true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetApplied())
	require.Len(t, resp.Msg.GetSuite().GetCases(), 2, "the proposal is appended to the original case")

	// corpusMutationsGoThroughFile: the grown corpus was written back via the
	// control plane (WriteCorpus), authorized with the registry-minted token.
	require.NotNil(t, ctrl.gotCorpus, "apply must write back through WriteCorpus, not a direct store upsert")
	require.Len(t, ctrl.gotCorpus.GetCases(), 2)
	require.Equal(t, "tok-123", ctrl.gotToken, "the control token is presented on the write-back")

	// The store then mirrors the file's effective corpus.
	stored, _ := store.GetSuite(context.Background(), "cli-health.commands.primary")
	require.Len(t, stored.GetCases(), 2, "the store re-syncs to the file's effective corpus")
	var found bool
	for _, c := range stored.GetCases() {
		if c.GetCaseId() == "gen-abc" {
			found = true
			require.Contains(t, c.GetTags(), "generated")
		}
	}
	require.True(t, found)
}

func TestGenerateApplyNoControlPlaneIsPrecondition(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	// The provider declares no control endpoint → cannot write the corpus back.
	ctrl := &fakeCorpusControl{err: control.ErrNoControlPlane}
	client := newGenerateClientFull(t, store, genProviders(), &fakeGenerator{res: oneProposal()}, ctrl, fakeTokens{token: "tok-123"})

	_, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Apply: true,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	// The store was NOT mutated (no reverse drift to a cache the file can't back).
	stored, _ := store.GetSuite(context.Background(), "cli-health.commands.primary")
	require.Len(t, stored.GetCases(), 1)
}

func TestGenerateApplyWithoutControlIsUnimplemented(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	logger, _ := connectxtest.NewLogger(t)
	// Generator wired but NO Control/Tokens → preview works, apply is Unimplemented.
	path, h := evalconnect.NewEvalServiceHandler(handler.NewConnectHandler(handler.Deps{
		Store: store, Providers: genProviders(), Generator: &fakeGenerator{res: oneProposal()}, Logger: logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	client := evalconnect.NewEvalServiceClient(server.Client(), server.URL)
	_, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Apply: true,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestGenerateApplyWithNoProposalsDoesNotPersist(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	empty := &corpusgen.Result{Sampled: 2, Strata: []string{"type:command"}}
	client := newGenerateClient(t, store, genProviders(), &fakeGenerator{res: empty})

	resp, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Apply: true,
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetApplied(), "apply with zero proposals is a no-op")
}

func TestGenerateUnknownSuiteNotFound(t *testing.T) {
	client := newGenerateClient(t, newFakeStore(), genProviders(), &fakeGenerator{res: oneProposal()})
	_, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{SuiteId: "ghost"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGenerateUnregisteredProviderPrecondition(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	// Empty resolver → provider lookup fails.
	client := newGenerateClient(t, store, fakeResolver{descs: map[string]*registryv1.ProviderDescriptor{}}, &fakeGenerator{res: oneProposal()})
	_, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{SuiteId: "cli-health.commands.primary"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}

func TestGenerateUnconfiguredUnimplemented(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	logger, _ := connectxtest.NewLogger(t)
	// No Generator wired → Unimplemented, not a panic.
	path, h := evalconnect.NewEvalServiceHandler(handler.NewConnectHandler(handler.Deps{Store: store, Providers: genProviders(), Logger: logger}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	client := evalconnect.NewEvalServiceClient(server.Client(), server.URL)
	_, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{SuiteId: "cli-health.commands.primary"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestPromoteCasesWritesFileThenMirrorsStore(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuiteWithCandidates())
	ctrl := &fakeCorpusControl{written: true}
	client := newGenerateClientFull(t, store, genProviders(), &fakeGenerator{res: oneProposal()}, ctrl, fakeTokens{token: "tok-123"})

	resp, err := client.PromoteCases(context.Background(), connect.NewRequest(&evalv1.PromoteCasesRequest{
		SuiteId: "cli-health.commands.primary",
		CaseIds: []string{"gen-1"},
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetApplied())
	require.Equal(t, []string{"gen-1"}, resp.Msg.GetPromotedCaseIds())
	require.Empty(t, resp.Msg.GetAlreadyReviewedCaseIds())

	require.NotNil(t, ctrl.gotCorpus, "promotion must write through WriteCorpus")
	require.Equal(t, "tok-123", ctrl.gotToken)
	require.Equal(t, "reviewed", statusOf(ctrl.gotCorpus, "gen-1"))
	require.Equal(t, "candidate", statusOf(ctrl.gotCorpus, "gen-2"))

	stored, _ := store.GetSuite(context.Background(), "cli-health.commands.primary")
	require.Equal(t, "reviewed", statusOf(stored, "gen-1"), "store mirrors the provider's effective corpus")
}

func TestPromoteCasesAllAndIdempotentReplay(t *testing.T) {
	store := newFakeStore()
	s := genSuiteWithCandidates()
	s.Cases[1].Status = "reviewed"
	_, _ = store.UpsertSuite(context.Background(), s)
	ctrl := &fakeCorpusControl{written: true}
	client := newGenerateClientFull(t, store, genProviders(), &fakeGenerator{res: oneProposal()}, ctrl, fakeTokens{token: "tok-123"})

	resp, err := client.PromoteCases(context.Background(), connect.NewRequest(&evalv1.PromoteCasesRequest{
		SuiteId: "cli-health.commands.primary",
		All:     true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetApplied())
	require.Equal(t, []string{"gen-2"}, resp.Msg.GetPromotedCaseIds())
	require.Empty(t, resp.Msg.GetAlreadyReviewedCaseIds())

	resp, err = client.PromoteCases(context.Background(), connect.NewRequest(&evalv1.PromoteCasesRequest{
		SuiteId: "cli-health.commands.primary",
		All:     true,
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.GetApplied(), "second replay has no candidates left to mutate")
	require.Empty(t, resp.Msg.GetPromotedCaseIds())
	require.Empty(t, resp.Msg.GetAlreadyReviewedCaseIds())
}

func TestPromoteCasesRejectsAmbiguousSelection(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuiteWithCandidates())
	client := newGenerateClient(t, store, genProviders(), &fakeGenerator{res: oneProposal()})

	_, err := client.PromoteCases(context.Background(), connect.NewRequest(&evalv1.PromoteCasesRequest{
		SuiteId: "cli-health.commands.primary",
		CaseIds: []string{"gen-1"},
		All:     true,
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestPromoteCasesRejectsUnknownCase(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuiteWithCandidates())
	client := newGenerateClient(t, store, genProviders(), &fakeGenerator{res: oneProposal()})

	_, err := client.PromoteCases(context.Background(), connect.NewRequest(&evalv1.PromoteCasesRequest{
		SuiteId: "cli-health.commands.primary",
		CaseIds: []string{"missing"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func statusOf(s *evalv1.EvalSuite, id string) string {
	for _, c := range s.GetCases() {
		if c.GetCaseId() == id {
			return c.GetStatus()
		}
	}
	return ""
}
