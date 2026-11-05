package motions

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"device-studio/handlers/revisionapi"
	"device-studio/internal/composition"
	domain "device-studio/internal/motions"
	"device-studio/internal/physical"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	motionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions"
	motionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/motions/motions_v1connect"
)

func TestListMotionCandidatesRejectsMalformedCursor(t *testing.T) {
	_, err := NewConnectHandler(nil).ListMotionCandidates(context.Background(), connect.NewRequest(&motionsv1.ListMotionCandidatesRequest{AfterMotionId: "bad\nID"}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
	handler := NewConnectHandler(&cursorOnlyMotionService{})
	_, err = handler.ListMotionCandidates(context.Background(), connect.NewRequest(&motionsv1.ListMotionCandidatesRequest{AfterMotionId: "bad\nID"}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	_, err = handler.ListMotionCandidates(context.Background(), connect.NewRequest(&motionsv1.ListMotionCandidatesRequest{AfterMotionId: strings.Repeat("a", 129)}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

type cursorOnlyMotionService struct{ Service }

func (*cursorOnlyMotionService) ListHeads(context.Context, string, int) ([]domain.Revision, error) {
	panic("invalid cursor must be rejected before service access")
}

func TestMotionCandidateConnectRevisionContract(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(domain.Schema)))
	repo, err := domain.NewSQLiteRevisionRepository(db)
	require.NoError(t, err)
	service, err := domain.NewRevisionService(repo, nil)
	require.NoError(t, err)
	router := mux.NewRouter()
	ModuleWithService(service).Mount(router)
	server := httptest.NewServer(router)
	defer server.Close()
	client := motionsconnect.NewMotionsServiceClient(server.Client(), server.URL)
	candidate := validMotionCandidate()
	message := &motionsv1.MotionCandidate{}
	require.NoError(t, revisionapi.Encode(candidate, message))
	decoded, err := revisionapi.Decode[domain.MotionCandidate](message)
	require.NoError(t, err)
	_, _, _, err = domain.CanonicalizeCandidate(decoded)
	require.NoError(t, err)
	saved, err := client.SaveMotionCandidate(ctx, connect.NewRequest(&motionsv1.SaveMotionCandidateRequest{Candidate: message}))
	require.NoError(t, err)
	require.Equal(t, uint32(1), saved.Msg.Revision.Revision)
	require.Equal(t, "candidate", saved.Msg.Revision.State)
	got, err := client.GetMotionRevision(ctx, connect.NewRequest(&motionsv1.GetMotionRevisionRequest{MotionId: candidate.MotionID, Revision: 1}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, got.Msg.Revision.Sha256)
	head, err := client.GetMotionHead(ctx, connect.NewRequest(&motionsv1.GetMotionHeadRequest{MotionId: candidate.MotionID}))
	require.NoError(t, err)
	require.Equal(t, saved.Msg.Revision.Sha256, head.Msg.Revision.Sha256)
	page, err := client.ListMotionRevisions(ctx, connect.NewRequest(&motionsv1.ListMotionRevisionsRequest{MotionId: candidate.MotionID, PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, page.Msg.Revisions, 1)
	heads, err := client.ListMotionCandidates(ctx, connect.NewRequest(&motionsv1.ListMotionCandidatesRequest{PageSize: 1}))
	require.NoError(t, err)
	require.Len(t, heads.Msg.Candidates, 1)
	require.Equal(t, candidate.MotionID, heads.Msg.Candidates[0].MotionId)
	require.Equal(t, "candidate", heads.Msg.Candidates[0].State)
	_, err = client.SaveMotionCandidate(ctx, connect.NewRequest(&motionsv1.SaveMotionCandidateRequest{Candidate: message, ExpectedHead: 3}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeAborted, connectErr.Code())
	_, err = client.SaveMotionCandidate(ctx, connect.NewRequest(&motionsv1.SaveMotionCandidateRequest{Candidate: &motionsv1.MotionCandidate{Version: "unknown"}}))
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
}

func TestSaveMotionCandidateNilServiceIsInternal(t *testing.T) {
	_, err := NewConnectHandler(nil).SaveMotionCandidate(context.Background(), connect.NewRequest(&motionsv1.SaveMotionCandidateRequest{Candidate: &motionsv1.MotionCandidate{}}))
	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	require.Equal(t, connect.CodeInternal, connectErr.Code())
}

func validMotionCandidate() domain.MotionCandidate {
	identity := domain.TransformKey{Rotation: physical.QuatXYZW{W: 1}}
	return domain.MotionCandidate{Version: domain.CandidateVersion, MotionID: "api-motion", Timebase: composition.Rational{Num: 24, Den: 1}, DurationTicks: 10, Tracks: []domain.MotionTrack{{Target: domain.TrackTarget{DeviceRoot: true}, Interpolation: domain.Linear, Keys: []domain.MotionKey{{Tick: 0, Transform: &identity}, {Tick: 10, Transform: &identity}}}}}
}
