package projects_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"io"
	"testing"

	"connectrpc.com/connect"
	"device-studio/internal/jobs"

	handler "device-studio/handlers/projects"
	domain "device-studio/internal/projects"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	"github.com/vrooli/api-core/database"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
)

func client(t *testing.T, svc domain.Service) projectsconnect.ProjectsServiceClient {
	t.Helper()
	path, h := projectsconnect.NewProjectsServiceHandler(handler.NewConnectHandler(handler.Deps{Service: svc}))
	s := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return projectsconnect.NewProjectsServiceClient(s.Client(), s.URL)
}

func clientWithMedia(t *testing.T, svc domain.Service, media domain.SourceMediaService) projectsconnect.ProjectsServiceClient {
	t.Helper()
	path, h := projectsconnect.NewProjectsServiceHandler(handler.NewConnectHandler(handler.Deps{Service: svc, MediaService: media}))
	s := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return projectsconnect.NewProjectsServiceClient(s.Client(), s.URL)
}

func clientWithInspection(t *testing.T, svc domain.Service, media domain.SourceMediaService, inspection handler.SourceInspectionCoordinator) projectsconnect.ProjectsServiceClient {
	return clientWithInspectionAndActivator(t, svc, media, inspection, nil)
}

func clientWithInspectionAndActivator(t *testing.T, svc domain.Service, media domain.SourceMediaService, inspection handler.SourceInspectionCoordinator, activator handler.RouteActivator) projectsconnect.ProjectsServiceClient {
	t.Helper()
	path, h := projectsconnect.NewProjectsServiceHandler(handler.NewConnectHandler(handler.Deps{Service: svc, MediaService: media, Inspection: inspection, Activator: activator}))
	s := connectxtest.StartTestServer(t, connectx.ServiceMount{Path: path, Handler: h})
	return projectsconnect.NewProjectsServiceClient(s.Client(), s.URL)
}

type inspectionRPCStub struct {
	job         jobs.Job
	ensureCalls int
	statusCalls int
	retryCalls  int
	cancelCalls int
}

type routeActivatorStub struct{ calls int }

func (s *routeActivatorStub) ActivateTestRoute(context.Context) error { s.calls++; return nil }

func (s *inspectionRPCStub) EnsureSourceInspection(context.Context, string, string) (jobs.Job, bool, error) {
	s.ensureCalls++
	return s.job, true, nil
}
func (s *inspectionRPCStub) GetSourceInspectionStatus(context.Context, string, string) (jobs.Job, error) {
	s.statusCalls++
	return s.job, nil
}
func (s *inspectionRPCStub) RetrySourceInspection(_ context.Context, _, _ string, version int64) (jobs.Job, error) {
	s.retryCalls++
	if version != s.job.Version {
		return jobs.Job{}, jobs.ErrConcurrentTransition
	}
	return s.job, nil
}
func (s *inspectionRPCStub) CancelSourceInspection(_ context.Context, _, _ string, version int64) (jobs.Job, error) {
	s.cancelCalls++
	if version != s.job.Version {
		return jobs.Job{}, jobs.ErrConcurrentTransition
	}
	return s.job, nil
}

type mediaRPCStub struct{ row domain.SourceMedia }

type clockMediaRPCStub struct {
	mediaRPCStub
	receipt []byte
	clock   []byte
}

func (s clockMediaRPCStub) OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error) {
	return s.receipt, s.clock, nil
}

func (s mediaRPCStub) Import(context.Context, string, string, int64, io.Reader) (domain.SourceMedia, error) {
	return s.row, nil
}

func (s mediaRPCStub) BeginImport(context.Context, string, string, int64, io.Reader) (domain.StagedSourceMediaImport, error) {
	return nil, nil
}
func (s mediaRPCStub) BindInspectionJob(context.Context, string, string, string, string) (domain.SourceMedia, error) {
	return s.row, nil
}
func (s mediaRPCStub) BindInspectionAttempt(context.Context, string, string, string, string, string, string) (domain.SourceMedia, error) {
	return s.row, nil
}
func (s mediaRPCStub) StoreInspectionArtifacts(context.Context, string, string, string, string, string, string, []byte, []byte) (string, string, error) {
	return "", "", nil
}
func (s mediaRPCStub) OpenInspectionArtifacts(context.Context, string, string, string, string, string, string) ([]byte, []byte, error) {
	return nil, nil, nil
}
func (s mediaRPCStub) ProjectInspection(context.Context, domain.InspectionProjection) error {
	return nil
}

func (s mediaRPCStub) Get(context.Context, string, string) (domain.SourceMedia, error) {
	return s.row, nil
}

func (s mediaRPCStub) Open(context.Context, string, string) (domain.SourceMedia, io.ReadCloser, error) {
	return s.row, io.NopCloser(nil), nil
}

func (s mediaRPCStub) List(context.Context, string, int32, string) ([]domain.SourceMedia, string, error) {
	return []domain.SourceMedia{s.row}, "next", nil
}

type trackingMediaRPCStub struct {
	mediaRPCStub
	getCalls  int
	listCalls int
}

func (s *trackingMediaRPCStub) Get(ctx context.Context, projectID, id string) (domain.SourceMedia, error) {
	s.getCalls++
	return s.mediaRPCStub.Get(ctx, projectID, id)
}

func (s *trackingMediaRPCStub) List(ctx context.Context, projectID string, limit int32, token string) ([]domain.SourceMedia, string, error) {
	s.listCalls++
	return s.mediaRPCStub.List(ctx, projectID, limit, token)
}

func TestSourceMediaConnectListAndGetAreTyped(t *testing.T) {
	repo := &mem{p: domain.Project{ID: "p1", Name: "Project", Revision: 1}}
	row := domain.SourceMedia{ID: "m1", ProjectID: "p1", SHA256: "abc", MediaType: "image/png", Width: 3, Height: 2, RasterKnown: true, Kind: domain.MediaKindImage, Readiness: domain.ReadinessReady, OriginalFilename: "mask.png"}
	c := clientWithMedia(t, domain.NewService(repo, nil), mediaRPCStub{row})
	listed, err := c.ListSourceMedia(context.Background(), connect.NewRequest(&projectsv1.ListSourceMediaRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "m1", listed.Msg.SourceMedia[0].Id)
	require.Equal(t, "next", listed.Msg.NextPageToken)
	got, err := c.GetSourceMedia(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaRequest{ProjectId: "p1", Id: "m1"}))
	require.NoError(t, err)
	require.Equal(t, "abc", got.Msg.SourceMedia.Sha256)
	require.EqualValues(t, 3, *got.Msg.SourceMedia.Width)
}

func TestSourceMediaClockPagesAreBoundedAndDigestPinned(t *testing.T) {
	repo := &mem{p: domain.Project{ID: "p1", Name: "Project", Revision: 1}}
	clock := make([]byte, 12+3*12)
	copy(clock[:8], "DSCLK001")
	binary.BigEndian.PutUint32(clock[8:12], 3)
	for i, pts := range []int64{100, 103, 106} {
		pos := 12 + i*12
		binary.BigEndian.PutUint32(clock[pos:pos+4], uint32(i))
		binary.BigEndian.PutUint64(clock[pos+4:pos+12], uint64(pts))
	}
	clockSum := sha256.Sum256(clock)
	clockSHA := hex.EncodeToString(clockSum[:])
	receipt := &domain.InspectionReceipt{ID: "receipt", SHA256: "receipt-sha", ClockSHA256: clockSHA, TimeBaseNumerator: 1, TimeBaseDenominator: 30}
	row := domain.SourceMedia{ID: "m1", ProjectID: "p1", Kind: domain.MediaKindVideo, Readiness: domain.ReadinessReady, InspectionJobID: "job", InspectionAttemptID: "attempt", InspectionReceipt: receipt}
	client := clientWithMedia(t, domain.NewService(repo, nil), clockMediaRPCStub{mediaRPCStub: mediaRPCStub{row: row}, receipt: []byte(`{}`), clock: clock})
	first, err := client.GetSourceMediaClockPage(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaClockPageRequest{ProjectId: "p1", MediaId: "m1", ExpectedClockSha256: clockSHA, PageSize: 2}))
	require.NoError(t, err)
	require.Len(t, first.Msg.Points, 2)
	require.Equal(t, int64(100), first.Msg.Points[0].Pts)
	require.Equal(t, int64(103), first.Msg.Points[1].Pts)
	require.NotEmpty(t, first.Msg.NextPageToken)
	second, err := client.GetSourceMediaClockPage(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaClockPageRequest{ProjectId: "p1", MediaId: "m1", ExpectedClockSha256: clockSHA, PageSize: 2, PageToken: first.Msg.NextPageToken}))
	require.NoError(t, err)
	require.Len(t, second.Msg.Points, 1)
	require.Equal(t, uint32(2), second.Msg.Points[0].Ordinal)
	_, err = client.GetSourceMediaClockPage(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaClockPageRequest{ProjectId: "p1", MediaId: "m1", ExpectedClockSha256: base64.RawURLEncoding.EncodeToString([]byte("wrong")), PageSize: 1, PageToken: first.Msg.NextPageToken}))
	require.Error(t, err)
}

func TestSourceMediaRoutedReadsPinBeforeRepositoryAccess(t *testing.T) {
	media := &trackingMediaRPCStub{mediaRPCStub: mediaRPCStub{row: domain.SourceMedia{ID: "m1", ProjectID: "p1"}}}
	h := handler.NewConnectHandler(handler.Deps{Service: domain.NewService(&mem{p: domain.Project{ID: "p1"}}, nil), MediaService: media})
	_, err := h.GetSourceMedia(database.WithTestMode(context.Background()), connect.NewRequest(&projectsv1.GetSourceMediaRequest{ProjectId: "p1", Id: "m1"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	_, err = h.ListSourceMedia(database.WithTestMode(context.Background()), connect.NewRequest(&projectsv1.ListSourceMediaRequest{ProjectId: "p1"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	require.Zero(t, media.getCalls)
	require.Zero(t, media.listCalls)
}

func TestInspectionRPCsExposeReadOnlyStatusAndVersionedControls(t *testing.T) {
	repo := &mem{p: domain.Project{ID: "p1", Name: "Project", Revision: 1}}
	row := domain.SourceMedia{ID: "v1", ProjectID: "p1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SizeBytes: 64, MediaType: "video/mp4", Kind: domain.MediaKindVideo, Readiness: domain.ReadinessPending, InspectionJobID: "job1", InspectionAttemptID: "attempt1"}
	media := mediaRPCStub{row}
	job := jobs.Job{ID: "job1", Version: 7, State: jobs.Queued, Input: jobs.Input{ProjectID: "p1", SubjectKind: jobs.SubjectSourceInspection, SubjectID: "v1"}, Attempts: []jobs.Attempt{{ID: "attempt1"}}}
	inspection := &inspectionRPCStub{job: job}
	activator := &routeActivatorStub{}
	client := clientWithInspectionAndActivator(t, domain.NewService(repo, nil), media, inspection, activator)
	got, err := client.GetSourceMedia(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaRequest{ProjectId: "p1", Id: "v1"}))
	require.NoError(t, err)
	require.Equal(t, "job1", got.Msg.SourceMedia.InspectionJob.Id)
	require.Equal(t, int64(7), got.Msg.SourceMedia.InspectionJob.Version)
	require.Zero(t, inspection.ensureCalls, "read-only status must not submit work")
	require.Zero(t, inspection.retryCalls)
	ensured, err := client.EnsureSourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.EnsureSourceMediaInspectionRequest{ProjectId: "p1", MediaId: "v1"}))
	require.NoError(t, err)
	require.Equal(t, "attempt1", ensured.Msg.Job.CurrentAttemptId)
	_, err = client.RetrySourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.RetrySourceMediaInspectionRequest{ProjectId: "p1", MediaId: "v1", ExpectedJobVersion: 7}))
	require.NoError(t, err)
	_, err = client.CancelSourceMediaInspection(context.Background(), connect.NewRequest(&projectsv1.CancelSourceMediaInspectionRequest{ProjectId: "p1", MediaId: "v1", ExpectedJobVersion: 7}))
	require.NoError(t, err)
	require.Equal(t, 1, inspection.ensureCalls)
	require.Equal(t, 1, inspection.retryCalls)
	require.Equal(t, 1, inspection.cancelCalls)
	require.Equal(t, 3, activator.calls, "accepted ensure/retry/cancel must activate the shared pump")
}

type mem struct{ p domain.Project }

func (m *mem) Create(_ context.Context, p domain.Project) (domain.Project, error) {
	p.ID = "p1"
	m.p = p
	return p, nil
}

func (m *mem) Get(_ context.Context, id string) (domain.Project, error) {
	if id != m.p.ID {
		return domain.Project{}, domain.ErrProjectNotFound{ID: id}
	}
	return m.p, nil
}

func (m *mem) List(context.Context, int, *domain.Project) ([]domain.Project, error) {
	if m.p.ID == "" {
		return []domain.Project{}, nil
	}
	return []domain.Project{m.p}, nil
}

func (m *mem) Rename(_ context.Context, in domain.RenameInput, name string, rev int64) (domain.Project, error) {
	if rev != m.p.Revision {
		return domain.Project{}, domain.ErrRevisionConflict{ID: m.p.ID, Expected: rev, Current: m.p.Revision}
	}
	m.p.Name = name
	m.p.Revision++
	return m.p, nil
}

func TestProjectConnectJourneyAndTypedRevisionConflict(t *testing.T) { // [REQ:DS-INTEGRATION-OPERATIONS]
	repo := &mem{}
	svc := domain.NewService(repo, nil)
	c := client(t, svc)
	created, err := c.CreateProject(context.Background(), connect.NewRequest(&projectsv1.CreateProjectRequest{Name: "Example"}))
	require.NoError(t, err)
	p := created.Msg.Project
	got, err := c.GetProject(context.Background(), connect.NewRequest(&projectsv1.GetProjectRequest{Id: p.Id}))
	require.NoError(t, err)
	require.Equal(t, "Example", got.Msg.Project.Name)
	_, err = c.RenameProject(context.Background(), connect.NewRequest(&projectsv1.RenameProjectRequest{Id: p.Id, Name: "Updated", ExpectedRevision: p.Revision}))
	require.NoError(t, err)
	_, err = c.RenameProject(context.Background(), connect.NewRequest(&projectsv1.RenameProjectRequest{Id: p.Id, Name: "Attempted", ExpectedRevision: p.Revision}))
	require.Equal(t, connect.CodeAborted, connect.CodeOf(err))
	var decoded projectsv1.ProjectConflict
	details := err.(*connect.Error).Details()
	require.Len(t, details, 1)
	value, valueErr := details[0].Value()
	require.NoError(t, valueErr)
	decoded = *value.(*projectsv1.ProjectConflict)
	require.Equal(t, p.Id, decoded.ProjectId)
	require.Equal(t, p.Revision, decoded.ExpectedRevision)
	require.EqualValues(t, p.Revision+1, decoded.CurrentRevision)
}
