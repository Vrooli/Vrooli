package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/routing"
)

// fakeCapacity is a programmable CapacityAdapter for routing tests.
type fakeCapacity struct {
	eval     routing.CapacityEvaluation
	err      error
	claims   []routing.CapacityRequest
	releases []string
}

func (f *fakeCapacity) Claim(_ context.Context, req routing.CapacityRequest) (routing.CapacityEvaluation, error) {
	f.claims = append(f.claims, req)
	if f.err != nil {
		return routing.CapacityEvaluation{Verdict: routing.CapacityUnknown, RequiredBytes: req.RequiredBytes}, f.err
	}
	e := f.eval
	e.RequiredBytes = req.RequiredBytes
	return e, nil
}

func (f *fakeCapacity) Release(_ context.Context, claimID string) {
	f.releases = append(f.releases, claimID)
}

func capReq(profile sharedv1.Profile) *sharedv1.GatewayRequest {
	req := baseRequest(profile)
	req.Metadata = map[string]string{"required_vram_bytes": "1000000"}
	return req
}

func newCapService(t *testing.T, runner providers.CommandRunner, cap routing.CapacityAdapter) *routing.Service {
	t.Helper()
	db := newSchemaDB(t)
	return routing.NewService(testAdapters(runner), routing.NewSQLRepository(db), routing.WithCapacity(cap))
}

func TestCapacitySufficientLocalRouteSelectedAndClaimReleased(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityFit, GrantedBytes: 2000000, ClaimID: "claim-1"}}
	svc := newCapService(t, runner, fake)

	resp, err := svc.Execute(context.Background(), capReq(sharedv1.Profile_PROFILE_LOCAL_FIRST), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "ollama", resp.GetEvidence().GetSelectedProvider())
	require.Equal(t, string(routing.CapacityFit), resp.GetEvidence().GetCapacityVerdict())
	require.Equal(t, "claim-1", resp.GetEvidence().GetCapacityClaimId())
	require.Equal(t, int64(1000000), resp.GetEvidence().GetCapacityRequiredBytes())
	// Every acquired claim is released (probe claim + held execution claim).
	require.Equal(t, len(fake.claims), len(fake.releases))
	require.NotEmpty(t, fake.releases)
}

func TestCapacityInsufficientLocalFirstFallsBackToRemote(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	runner.Results[openRouterGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityInsufficient}}
	svc := newCapService(t, runner, fake)

	resp, err := svc.Execute(context.Background(), capReq(sharedv1.Profile_PROFILE_LOCAL_FIRST), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Equal(t, "openrouter", resp.GetEvidence().GetSelectedProvider(), "local-first falls back to remote when local capacity is insufficient")
}

func TestCapacityInsufficientLocalOnlyFailsClosed(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityInsufficient}}
	svc := newCapService(t, runner, fake)

	resp, err := svc.Execute(context.Background(), capReq(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.Equal(t, "blocked", resp.GetEvidence().GetStatus())
	require.Equal(t, "insufficient_capacity", resp.GetEvidence().GetRejectionReason())
	require.Equal(t, string(routing.CapacityInsufficient), resp.GetEvidence().GetCapacityVerdict())
}

func TestCapacityInsufficientPrivacySensitiveFailsClosed(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityInsufficient}}
	svc := newCapService(t, runner, fake)

	req := capReq(sharedv1.Profile_PROFILE_PRIVACY_SENSITIVE)
	req.PrivacyClass = sharedv1.PrivacyClass_PRIVACY_CLASS_SECRET
	resp, err := svc.Execute(context.Background(), req, "hello")
	require.NoError(t, err)
	require.Equal(t, "blocked", resp.GetEvidence().GetStatus(), "secret requests never route remote just because local is constrained")
	require.Equal(t, "insufficient_capacity", resp.GetEvidence().GetRejectionReason())
}

func TestCapacityAdvisoryReclaimUnavailableTreatedAsUnavailable(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityAdvisoryReclaimUnavailable}}
	svc := newCapService(t, runner, fake)

	resp, err := svc.Preview(context.Background(), capReq(sharedv1.Profile_PROFILE_LOCAL_FIRST))
	require.NoError(t, err)
	require.Equal(t, "openrouter", resp.GetSelectedProvider())
	ollama := candidateByProvider(resp.GetCandidates(), "ollama")
	require.NotNil(t, ollama)
	require.Equal(t, string(routing.CapacityAdvisoryReclaimUnavailable), ollama.GetCapacityVerdict())
	require.Equal(t, "insufficient_capacity", ollama.GetRejectionReason())
}

func TestCapacityAdapterErrorDegradesToUnknownAndProceeds(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	fake := &fakeCapacity{err: errors.New("broker unavailable")}
	svc := newCapService(t, runner, fake)

	resp, err := svc.Execute(context.Background(), capReq(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus(), "broker error degrades to unknown and does not block local-only")
	require.Equal(t, string(routing.CapacityUnknown), resp.GetEvidence().GetCapacityVerdict())
}

func TestCapacityNotEvaluatedWithoutDeclaredFootprint(t *testing.T) { // [REQ:AIGW-CAPACITY-ROUTING]
	runner := roleRunner()
	runner.Results[ollamaGenerateCmd] = providers.Result{Stdout: `{"response":"ok"}`}
	fake := &fakeCapacity{eval: routing.CapacityEvaluation{Verdict: routing.CapacityInsufficient}}
	svc := newCapService(t, runner, fake)

	// No required_vram_bytes metadata → capacity is not consulted at all.
	resp, err := svc.Execute(context.Background(), baseRequest(sharedv1.Profile_PROFILE_LOCAL_ONLY), "hello")
	require.NoError(t, err)
	require.Equal(t, "succeeded", resp.GetEvidence().GetStatus())
	require.Empty(t, fake.claims, "the broker is never consulted without a declared footprint")
}
