package protocgenconnectgo

import (
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func TestConformance(t *testing.T) {
	hostreqkittest.RunSuite(t, hostreqkittest.Case{
		NewHandler:         func() hostreqkit.Handler { return NewHandler(testManifest) },
		Name:               testManifest.Name,
		Kind:               hostreqspec.KindTool,
		SupportedPlatforms: testManifest.Platforms,
		InstallCommand:     "go install connectrpc.com/connect/cmd/protoc-gen-connect-go",
		ManifestVersion:    testManifest.Version,
		DefaultVersion:     defaultVersion,
		Checks:             []string{"name_and_kind", "pinned_version_matches_manifest"},
	})
}
