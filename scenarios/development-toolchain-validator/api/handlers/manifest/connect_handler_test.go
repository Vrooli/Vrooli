package manifest_test

import (
	"context"
	"testing"
	"time"

	manifestH "development-toolchain-validator/handlers/manifest"
	manifest "development-toolchain-validator/internal/manifest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"
	manifestconnect "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest/manifest_v1connect"
)

type fakeService struct {
	ListOut    []manifest.Manifest
	ListErr    error
	GetOut     manifest.Manifest
	GetErr     error
	UpsertOut  manifest.Manifest
	UpsertErr  error
	ClearOut   time.Time
	ClearErr   error
	UpsertSeen manifest.UpsertInput
	GetSkillID string
	GetGolden  string
}

func (f *fakeService) List(context.Context) ([]manifest.Manifest, error) {
	return f.ListOut, f.ListErr
}

func (f *fakeService) Get(_ context.Context, skillID, goldenSlug string) (manifest.Manifest, error) {
	f.GetSkillID = skillID
	f.GetGolden = goldenSlug
	return f.GetOut, f.GetErr
}

func (f *fakeService) Upsert(_ context.Context, in manifest.UpsertInput) (manifest.Manifest, error) {
	f.UpsertSeen = in
	return f.UpsertOut, f.UpsertErr
}

func (f *fakeService) ClearStale(_ context.Context, _, _ string) (time.Time, error) {
	return f.ClearOut, f.ClearErr
}

var _ manifest.Service = (*fakeService)(nil)

func newClient(t *testing.T, svc manifest.Service) manifestconnect.ManifestServiceClient {
	t.Helper()
	logger, _ := connectxtest.NewLogger(t)
	path, handler := manifestconnect.NewManifestServiceHandler(manifestH.NewConnectHandler(manifestH.Deps{
		Service: svc,
		Logger:  logger,
	}))
	server := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: handler})
	return manifestconnect.NewManifestServiceClient(server.Client(), server.URL)
}

func TestList_PassesThrough(t *testing.T) {
	client := newClient(t, &fakeService{ListOut: []manifest.Manifest{
		{SkillID: "a", GoldenSlug: "g1"},
		{SkillID: "b", GoldenSlug: "g2"},
	}})
	resp, err := client.ListManifests(context.Background(), connect.NewRequest(&manifestv1.ListManifestsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Manifests, 2)
}

func TestGet_PassesSkillAndGolden(t *testing.T) {
	svc := &fakeService{GetOut: manifest.Manifest{SkillID: "implementation-plan-authoring", GoldenSlug: "reference-react-vite"}}
	client := newClient(t, svc)
	resp, err := client.GetManifest(context.Background(), connect.NewRequest(&manifestv1.GetManifestRequest{
		SkillId: "implementation-plan-authoring", GoldenSlug: "reference-react-vite",
	}))
	require.NoError(t, err)
	require.Equal(t, "implementation-plan-authoring", resp.Msg.Manifest.SkillId)
	require.Equal(t, "implementation-plan-authoring", svc.GetSkillID)
	require.Equal(t, "reference-react-vite", svc.GetGolden)
}

func TestGet_NotFoundMapsToNotFound(t *testing.T) {
	client := newClient(t, &fakeService{GetErr: manifest.ErrManifestNotFound{SkillID: "x", GoldenSlug: "y"}})
	_, err := client.GetManifest(context.Background(), connect.NewRequest(&manifestv1.GetManifestRequest{SkillId: "x", GoldenSlug: "y"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestUpsert_InvalidMapsToInvalidArgument(t *testing.T) {
	client := newClient(t, &fakeService{UpsertErr: manifest.ErrInvalidManifest{Field: "skill_id", Reason: "required"}})
	_, err := client.UpsertManifest(context.Background(), connect.NewRequest(&manifestv1.UpsertManifestRequest{
		Manifest: &manifestv1.Manifest{SkillId: "", GoldenSlug: "g", WildcardAllowed: true},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpsert_RequiresManifest(t *testing.T) {
	client := newClient(t, &fakeService{})
	_, err := client.UpsertManifest(context.Background(), connect.NewRequest(&manifestv1.UpsertManifestRequest{Manifest: nil}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestUpsert_RoundTripsConvergenceTarget(t *testing.T) {
	svc := &fakeService{UpsertOut: manifest.Manifest{
		SkillID: "implementation-plan-authoring", GoldenSlug: "reference-react-vite",
		ConvergenceTarget: manifest.ConvergenceTargetEmptyDiff, WildcardAllowed: true,
	}}
	client := newClient(t, svc)
	resp, err := client.UpsertManifest(context.Background(), connect.NewRequest(&manifestv1.UpsertManifestRequest{
		Manifest: &manifestv1.Manifest{
			SkillId: "implementation-plan-authoring", GoldenSlug: "reference-react-vite",
			WildcardAllowed:   true,
			ConvergenceTarget: manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF,
		},
	}))
	require.NoError(t, err)
	require.Equal(t, manifestv1.ConvergenceTarget_CONVERGENCE_TARGET_EMPTY_DIFF, resp.Msg.Manifest.ConvergenceTarget)
	require.Equal(t, manifest.ConvergenceTargetEmptyDiff, svc.UpsertSeen.ConvergenceTarget)
}

func TestClearStale_ReturnsTimestamp(t *testing.T) {
	now := time.Date(2026, 5, 18, 13, 0, 0, 0, time.UTC)
	client := newClient(t, &fakeService{ClearOut: now})
	resp, err := client.ClearStale(context.Background(), connect.NewRequest(&manifestv1.ClearStaleRequest{
		SkillId: "implementation-plan-authoring", GoldenSlug: "reference-react-vite",
	}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.ClearedAt)
	require.WithinDuration(t, now, resp.Msg.ClearedAt.AsTime(), time.Millisecond)
}
