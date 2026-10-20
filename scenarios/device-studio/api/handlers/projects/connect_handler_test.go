package projects_test

import (
	"connectrpc.com/connect"
	"context"
	handler "device-studio/handlers/projects"
	domain "device-studio/internal/projects"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	connectxtest "github.com/vrooli/api-core/connectxtest"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects/projects_v1connect"
	"io"
	"testing"
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

type mediaRPCStub struct{ row domain.SourceMedia }

func (s mediaRPCStub) Import(context.Context, string, string, int64, io.Reader) (domain.SourceMedia, error) {
	return s.row, nil
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

func TestSourceMediaConnectListAndGetAreTyped(t *testing.T) {
	repo := &mem{p: domain.Project{ID: "p1", Name: "Project", Revision: 1}}
	row := domain.SourceMedia{ID: "m1", ProjectID: "p1", SHA256: "abc", MediaType: "image/png", Width: 3, Height: 2, OriginalFilename: "mask.png"}
	c := clientWithMedia(t, domain.NewService(repo, nil), mediaRPCStub{row})
	listed, err := c.ListSourceMedia(context.Background(), connect.NewRequest(&projectsv1.ListSourceMediaRequest{ProjectId: "p1"}))
	require.NoError(t, err)
	require.Equal(t, "m1", listed.Msg.SourceMedia[0].Id)
	require.Equal(t, "next", listed.Msg.NextPageToken)
	got, err := c.GetSourceMedia(context.Background(), connect.NewRequest(&projectsv1.GetSourceMediaRequest{ProjectId: "p1", Id: "m1"}))
	require.NoError(t, err)
	require.Equal(t, "abc", got.Msg.SourceMedia.Sha256)
	require.EqualValues(t, 3, got.Msg.SourceMedia.Width)
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
func TestProjectConnectJourneyAndTypedRevisionConflict(t *testing.T) {
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
