package scenarioprimitives

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	manifestPath := filepath.Join(repoRoot, "cli", "manifest.json")
	manifestRaw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read project manifest: %v", err)
	}
	group, err := BuildScenarioPrimitiveGroup(manifestRaw, nil)
	if err != nil {
		t.Fatalf("assemble project scenario primitive group: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(repoRoot), cliapp.EvidenceExportInput{
		Scenario:    projectCLIName,
		ManifestRaw: manifestRaw,
		Groups:      []cliapp.SubcommandGroup{group},
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
