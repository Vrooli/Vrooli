package projects

import (
	domain "device-studio/internal/projects"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func toProto(p domain.Project) *projectsv1.Project {
	return &projectsv1.Project{Id: p.ID, Name: p.Name, CreatedAt: timestamppb.New(p.CreatedAt.UTC()), UpdatedAt: timestamppb.New(p.UpdatedAt.UTC()), Revision: p.Revision}
}
