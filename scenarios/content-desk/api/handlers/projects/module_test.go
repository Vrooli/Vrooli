package projects

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	internalprojects "content-desk/internal/projects"
	"github.com/stretchr/testify/require"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/content-desk/v1/projects"
)

type fakeRepository struct {
	items     []internalprojects.Project
	createErr error
}

func (f *fakeRepository) ListActive(context.Context) ([]internalprojects.Project, error) {
	return f.items, nil
}
func (f *fakeRepository) Create(_ context.Context, name string) (internalprojects.Project, error) {
	if f.createErr != nil {
		return internalprojects.Project{}, f.createErr
	}
	p := internalprojects.Project{ID: "stable-id", Name: name, Status: internalprojects.StatusActive}
	f.items = append(f.items, p)
	return p, nil
}

func TestListProjectsReturnsOnlyRepositoryActiveItems(t *testing.T) {
	h := handler{repo: &fakeRepository{items: []internalprojects.Project{{ID: "p1", Name: "Notes", Status: "active"}}}}
	response, err := h.ListProjects(context.Background(), connect.NewRequest(&projectsv1.ListProjectsRequest{}))
	require.NoError(t, err)
	require.Equal(t, "p1", response.Msg.Projects[0].Id)
}

func TestCreateProjectMapsReservedNameToInvalidArgument(t *testing.T) {
	h := handler{repo: &fakeRepository{createErr: internalprojects.ErrInvalidName}}
	_, err := h.CreateProject(context.Background(), connect.NewRequest(&projectsv1.CreateProjectRequest{Name: "unassigned"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateProjectReturnsActiveProject(t *testing.T) {
	h := handler{repo: &fakeRepository{}}
	response, err := h.CreateProject(context.Background(), connect.NewRequest(&projectsv1.CreateProjectRequest{Name: "Notes"}))
	require.NoError(t, err)
	require.Equal(t, "stable-id", response.Msg.Project.Id)
	require.Equal(t, "active", response.Msg.Project.Status)
}

func TestCreateProjectMapsRepositoryFailure(t *testing.T) {
	h := handler{repo: &fakeRepository{createErr: errors.New("storage down")}}
	_, err := h.CreateProject(context.Background(), connect.NewRequest(&projectsv1.CreateProjectRequest{Name: "Notes"}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
