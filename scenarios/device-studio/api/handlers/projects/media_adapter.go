package projects

import (
	"device-studio/internal/projects"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func sourceMediaToProto(m projects.SourceMedia) *projectsv1.SourceMedia {
	return &projectsv1.SourceMedia{Id: m.ID, ProjectId: m.ProjectID, Sha256: m.SHA256, SizeBytes: m.SizeBytes, MediaType: m.MediaType, Width: m.Width, Height: m.Height, Orientation: m.Orientation, ColorModel: m.ColorModel, ColorProfileStatus: m.ColorProfileStatus, OriginalFilename: m.OriginalFilename, ImportedAt: timestamppb.New(m.ImportedAt.UTC()), ImportMethod: m.ImportMethod, Decoder: m.Decoder}
}
