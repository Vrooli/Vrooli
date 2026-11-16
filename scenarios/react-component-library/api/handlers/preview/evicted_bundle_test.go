package preview

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	"react-component-library/internal/components"
	internalpreview "react-component-library/internal/preview"
)

type evictedBundleService struct{}

func (evictedBundleService) GetBundle(context.Context, string) (internalpreview.Bundle, error) {
	return internalpreview.Bundle{}, internalpreview.ErrEvictedDependency{LibraryID: "react-component-library:Child", Version: "1.2.3"}
}
func (evictedBundleService) GetBundleVersion(context.Context, string, string) (internalpreview.Bundle, error) {
	return internalpreview.Bundle{}, nil
}
func (evictedBundleService) GetBundleVersionWithFrame(context.Context, string, string, *components.StoryFrame) (internalpreview.Bundle, error) {
	return internalpreview.Bundle{}, nil
}

func TestGetPreviewBundleMapsEvictedDependencyToFailedPrecondition(t *testing.T) {
	h := NewConnectHandler(Deps{Service: evictedBundleService{}})
	_, err := h.GetPreviewBundle(context.Background(), connect.NewRequest(&previewv1.GetPreviewBundleRequest{Id: "root"}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "react-component-library:Child@1.2.3 is evicted")
	require.Contains(t, err.Error(), `react-component-library versions materialize "react-component-library:Child" 1.2.3`)
}
