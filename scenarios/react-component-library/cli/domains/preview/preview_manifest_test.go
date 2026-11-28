package preview

import (
	"os"
	"path/filepath"
	"testing"

	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"

	"github.com/vrooli/cli-core/cliapp"
)

func TestPreviewManifestCoversPreviewService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, previewv1.File_react_component_library_v1_preview_preview_proto, "PreviewService")
}
