package skill_catalog

import (
	"os"
	"path/filepath"
	"testing"

	skillcatalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/skill_catalog"

	"github.com/vrooli/cli-core/cliapp"
)

func TestSkillCatalogManifestCoversSkillCatalogService(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	if err != nil {
		t.Fatalf("read cli/manifest.json: %v", err)
	}
	cliapp.RequireProtoServiceCoverage(t, raw, skillcatalogv1.File_development_toolchain_validator_v1_skill_catalog_skill_catalog_proto, "SkillCatalogService")
}
