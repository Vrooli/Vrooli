package projects

import (
	domain "device-studio/internal/projects"
	"github.com/stretchr/testify/require"
	projectsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/projects"
	"testing"
)

func TestSourceMediaProtoPreservesUnknownVideoFactsAsAbsent(t *testing.T) {
	got := sourceMediaToProto(domain.SourceMedia{ID: "video", ProjectID: "p", Kind: domain.MediaKindVideo, Readiness: domain.ReadinessPending, MediaType: "video/mp4"})
	require.Equal(t, projectsv1.SourceMediaKind_SOURCE_MEDIA_KIND_VIDEO, got.Kind)
	require.Equal(t, projectsv1.MediaReadiness_MEDIA_READINESS_PENDING, got.Readiness)
	require.Nil(t, got.Width)
	require.Nil(t, got.Height)
	require.Nil(t, got.Orientation)
}
