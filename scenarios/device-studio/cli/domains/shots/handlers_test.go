package shots

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	shotsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots"
	shotsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/shots/shots_v1connect"
)

func TestReadSnapshotUsesGeneratedProtoJSONSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "snapshot.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"motionMode":"MOTION_MODE_FIXED","scene":{"displayWidthPixels":400}}`), 0o600))
	snapshot, err := readSnapshot(path)
	require.NoError(t, err)
	require.Equal(t, shotsv1.MotionMode_MOTION_MODE_FIXED, snapshot.MotionMode)
	require.EqualValues(t, 400, snapshot.Scene.DisplayWidthPixels)

	require.NoError(t, os.WriteFile(path, []byte(`{"unknownRecipePayload":true}`), 0o600))
	_, err = readSnapshot(path)
	require.Error(t, err, "unknown fields must not become an arbitrary recipe escape hatch")
}

func TestShotConflictRequiresTypedDetail(t *testing.T) {
	plain := connect.NewError(connect.CodeAborted, errors.New("server busy"))
	_, ok := shotConflict(plain)
	require.False(t, ok)

	conflict := connect.NewError(connect.CodeAborted, errors.New("conflict"))
	detail, err := connect.NewErrorDetail(&shotsv1.ShotConflict{ShotId: "shot-1", ExpectedRevision: 2, CurrentRevision: 3})
	require.NoError(t, err)
	conflict.AddDetail(detail)
	message, ok := shotConflict(conflict)
	require.True(t, ok)
	require.ErrorContains(t, message, "shot-1 revision conflict: expected 2, current 3")
}

type shotContractClient struct {
	shotsconnect.ShotsServiceClient
	create    *shotsv1.CreateShotRequest
	save      *shotsv1.SaveShotRequest
	list      *shotsv1.ListShotsRequest
	get       *shotsv1.GetShotRequest
	revisions *shotsv1.ListShotRevisionsRequest
	revision  *shotsv1.GetShotRevisionRequest
	compile   *shotsv1.CompileShotRequest
	validate  *shotsv1.ValidateShotRequest
	err       error
}

func (c *shotContractClient) CreateShot(_ context.Context, r *connect.Request[shotsv1.CreateShotRequest]) (*connect.Response[shotsv1.CreateShotResponse], error) {
	c.create = r.Msg
	return connect.NewResponse(&shotsv1.CreateShotResponse{Shot: &shotsv1.Shot{Id: "shot-1", Name: "portrait", HeadRevision: 4}, Revision: &shotsv1.ShotRevision{Revision: 4, Name: "portrait"}}), c.err
}
func (c *shotContractClient) SaveShot(_ context.Context, r *connect.Request[shotsv1.SaveShotRequest]) (*connect.Response[shotsv1.SaveShotResponse], error) {
	c.save = r.Msg
	return connect.NewResponse(&shotsv1.SaveShotResponse{Shot: &shotsv1.Shot{Id: "shot-1"}, Revision: &shotsv1.ShotRevision{Revision: r.Msg.ExpectedRevision + 1}}), c.err
}
func (c *shotContractClient) ListShots(_ context.Context, r *connect.Request[shotsv1.ListShotsRequest]) (*connect.Response[shotsv1.ListShotsResponse], error) {
	c.list = r.Msg
	return connect.NewResponse(&shotsv1.ListShotsResponse{Shots: []*shotsv1.Shot{{Id: "shot-1", Name: "portrait", HeadRevision: 4}}, NextPageToken: "next"}), c.err
}
func (c *shotContractClient) GetShot(_ context.Context, r *connect.Request[shotsv1.GetShotRequest]) (*connect.Response[shotsv1.GetShotResponse], error) {
	c.get = r.Msg
	return connect.NewResponse(&shotsv1.GetShotResponse{Shot: &shotsv1.Shot{Id: r.Msg.Id, Name: "portrait"}, Head: &shotsv1.ShotRevision{Revision: 4}}), c.err
}
func (c *shotContractClient) ListShotRevisions(_ context.Context, r *connect.Request[shotsv1.ListShotRevisionsRequest]) (*connect.Response[shotsv1.ListShotRevisionsResponse], error) {
	c.revisions = r.Msg
	return connect.NewResponse(&shotsv1.ListShotRevisionsResponse{Revisions: []*shotsv1.ShotRevision{{ShotId: r.Msg.ShotId, Revision: 3, Name: "older"}}, NextPageToken: "older-page"}), c.err
}
func (c *shotContractClient) GetShotRevision(_ context.Context, r *connect.Request[shotsv1.GetShotRevisionRequest]) (*connect.Response[shotsv1.GetShotRevisionResponse], error) {
	c.revision = r.Msg
	return connect.NewResponse(&shotsv1.GetShotRevisionResponse{Revision: &shotsv1.ShotRevision{ShotId: r.Msg.ShotId, Revision: r.Msg.Revision, Name: "immutable"}}), c.err
}
func (c *shotContractClient) CompileShot(_ context.Context, r *connect.Request[shotsv1.CompileShotRequest]) (*connect.Response[shotsv1.CompileShotResponse], error) {
	c.compile = r.Msg
	return connect.NewResponse(&shotsv1.CompileShotResponse{Compiled: &shotsv1.CompiledShot{ShotId: r.Msg.ShotId, Revision: r.Msg.Revision, RenderEligible: false, Issues: []*shotsv1.ShotIssue{{FieldPath: "scene", Summary: "missing"}}}}), c.err
}
func (c *shotContractClient) ValidateShot(_ context.Context, r *connect.Request[shotsv1.ValidateShotRequest]) (*connect.Response[shotsv1.ValidateShotResponse], error) {
	c.validate = r.Msg
	return connect.NewResponse(&shotsv1.ValidateShotResponse{RenderEligible: false, Issues: []*shotsv1.ShotIssue{{FieldPath: "camera", Summary: "unsupported"}}}), c.err
}

func shotContext(flags, positionals map[string]string) cliapp.OperationContext {
	return cliapp.NewTestRunContext(cliapp.TestRunContextOptions{
		Schema: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "snapshot"}, {Name: "project-id"}, {Name: "name"}, {Name: "expected-revision"}, {Name: "page-size"}, {Name: "page-token"}, {Name: "revision"}}, Positionals: []cliapp.Positional{{Name: "id"}}},
		Flags:  flags, Positionals: positionals,
	})
}

func TestShotHandlersForwardOperationsAndRenderServerResults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shot.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"motionMode":"MOTION_MODE_FIXED"}`), 0o600))
	client := &shotContractClient{}
	h := &handlers{client: client}
	ctx := shotContext(map[string]string{"snapshot": path, "project-id": "project-9", "name": "portrait", "expected-revision": "4", "page-size": "12", "page-token": "cursor", "revision": "3"}, map[string]string{"id": "shot-1"})

	created, err := h.createCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "project-9", client.create.ProjectId)
	require.Equal(t, "portrait", client.create.Name)
	require.NotNil(t, client.create.Snapshot)
	require.Contains(t, h.createReport(ctx, created).Result[0], "Created shot shot-1 at revision 4")

	saved, err := h.saveCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "shot-1", client.save.ShotId)
	require.EqualValues(t, 4, client.save.ExpectedRevision)
	require.Contains(t, h.saveReport(ctx, saved).Result[0], "Saved shot shot-1 at revision 5")

	listed, err := h.listCall(ctx)
	require.NoError(t, err)
	require.Equal(t, &shotsv1.ListShotsRequest{ProjectId: "project-9", PageSize: 12, PageToken: "cursor"}, client.list)
	require.Contains(t, h.listReport(ctx, listed).Results[0], "portrait")
	require.Contains(t, h.listReport(ctx, listed).RetrievalHints[0], "next")

	opened, err := h.getCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "shot-1", client.get.Id)
	require.Contains(t, h.getReport(ctx, opened).Results[0], "portrait")

	pages, err := h.revisionsCall(ctx)
	require.NoError(t, err)
	require.Equal(t, &shotsv1.ListShotRevisionsRequest{ShotId: "shot-1", PageSize: 12, PageToken: "cursor"}, client.revisions)
	require.Contains(t, h.revisionsReport(ctx, pages).RetrievalHints[0], "older-page")

	loaded, err := h.revisionCall(ctx)
	require.NoError(t, err)
	require.Equal(t, &shotsv1.GetShotRevisionRequest{ShotId: "shot-1", Revision: 3}, client.revision)
	require.Contains(t, h.revisionReport(ctx, loaded).Summary[0], "immutable")

	compiled, err := h.compileCall(ctx)
	require.NoError(t, err)
	require.Equal(t, &shotsv1.CompileShotRequest{ShotId: "shot-1", Revision: 3}, client.compile)
	require.Contains(t, h.compileReport(ctx, compiled).Summary[0], "render blocked")
	require.Contains(t, h.compileReport(ctx, compiled).Results[2], "scene: missing")

	validated, err := h.validateCall(ctx)
	require.NoError(t, err)
	require.Equal(t, "project-9", client.validate.ProjectId)
	require.NotNil(t, client.validate.Snapshot)
	require.Contains(t, h.validateReport(ctx, validated).Results[0], "camera: unsupported")
}

func TestShotHandlersRejectInvalidInputsBeforeRPCAndPropagateRPCFailure(t *testing.T) {
	client := &shotContractClient{}
	h := &handlers{client: client}
	for _, raw := range []string{"abc", "2147483648"} {
		ctx := shotContext(map[string]string{"page-size": raw}, nil)
		_, listErr := h.listCall(ctx)
		_, revisionsErr := h.revisionsCall(shotContext(map[string]string{"page-size": raw}, map[string]string{"id": "shot-1"}))
		require.ErrorContains(t, listErr, "invalid page-size")
		require.ErrorContains(t, revisionsErr, "invalid page-size")
		require.Nil(t, client.list)
		require.Nil(t, client.revisions)
	}
	_, err := h.saveCall(shotContext(map[string]string{"expected-revision": "0"}, map[string]string{"id": "shot-1"}))
	require.ErrorContains(t, err, "expected-revision must be a positive integer")
	require.Nil(t, client.save)

	client.err = connect.NewError(connect.CodeUnavailable, errors.New("backend offline"))
	_, err = h.listCall(shotContext(nil, nil))
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "list shots"), fmt.Sprintf("unexpected error: %v", err))
	if client.list.PageSize != 50 {
		t.Fatalf("default page size = %d, want 50", client.list.PageSize)
	}
}

func TestRevisionReportHandlesEmptyPageWithContinuation(t *testing.T) {
	report := (&handlers{}).revisionsReport(shotContext(nil, map[string]string{"id": "shot-1"}), &shotsv1.ListShotRevisionsResponse{NextPageToken: "next"})
	require.Equal(t, []string{"Found 0 revision(s)."}, report.Summary)
	require.Contains(t, report.RetrievalHints[0], "next")
}
