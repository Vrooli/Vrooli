package eval_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	"search-hub/internal/corpusgen"
	internaleval "search-hub/internal/eval"

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

func newGenerateClient(t *testing.T, store internaleval.Store, providers internaleval.ProviderResolver, gen handler.CorpusGenerator) evalconnect.EvalServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, h := evalconnect.NewEvalServiceHandler(handler.NewConnectHandler(handler.Deps{
		Store: store, Providers: providers, Generator: gen, Logger: logger,
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

func TestGenerateApplyAppendsAndPersists(t *testing.T) {
	store := newFakeStore()
	_, _ = store.UpsertSuite(context.Background(), genSuite())
	client := newGenerateClient(t, store, genProviders(), &fakeGenerator{res: oneProposal()})

	resp, err := client.Generate(context.Background(), connect.NewRequest(&evalv1.GenerateRequest{
		SuiteId: "cli-health.commands.primary", Count: 5, Apply: true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetApplied())
	require.Len(t, resp.Msg.GetSuite().GetCases(), 2, "the proposal is appended to the original case")
	stored, _ := store.GetSuite(context.Background(), "cli-health.commands.primary")
	require.Len(t, stored.GetCases(), 2, "the merged suite is persisted")
	// The appended case carries the generated marker (so the sweep holds it out).
	var found bool
	for _, c := range stored.GetCases() {
		if c.GetCaseId() == "gen-abc" {
			found = true
			require.Contains(t, c.GetTags(), "generated")
		}
	}
	require.True(t, found)
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
