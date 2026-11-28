package preview

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"react-component-library/internal/components"
)

func TestPreviewImagePathUsesManifestCategory(t *testing.T) {
	root := filepath.Join("/repo", "scenarios", "react-component-library", "library")
	cases := []struct {
		name      string
		component components.Component
		want      string
	}{
		{
			name: "component",
			component: components.Component{
				Slug:         "Button",
				ManifestPath: "components/Button/component.json",
			},
			want: filepath.Join(root, "components", "Button", "versions", "2.2.14", "preview.png"),
		},
		{
			name: "primitive",
			component: components.Component{
				Slug:         "Text",
				ManifestPath: "primitives/Text/component.json",
			},
			want: filepath.Join(root, "primitives", "Text", "versions", "2.2.14", "preview.png"),
		},
		{
			name: "foundation",
			component: components.Component{
				Slug:         "BaseStyles",
				ManifestPath: "foundations/BaseStyles/component.json",
			},
			want: filepath.Join(root, "foundations", "BaseStyles", "versions", "2.2.14", "preview.png"),
		},
		{
			name: "path traversal cannot escape the category",
			component: components.Component{
				Slug:         "Button",
				ManifestPath: "../outside/Button/component.json",
			},
			want: filepath.Join(root, "../outside/Button", "versions", "2.2.14", "preview.png"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, previewImagePath(root, tc.component, "2.2.14"))
		})
	}
}
