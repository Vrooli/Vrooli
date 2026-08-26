package scenarioprimitives

import (
	"fmt"
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
	groups := []cliapp.SubcommandGroup{group}
	for _, path := range []string{
		"break-glass",
		"runtime/supervisor",
		"runtime/recovery/policy",
		"host",
		"capability",
		"credentials",
		"credentials/store",
		"credentials/keyring",
		"credentials/recovery",
	} {
		manifestGroup, loadErr := loadManifestGroupForEvidence(manifestRaw, path)
		if loadErr != nil {
			t.Fatalf("assemble manifest group %q: %v", path, loadErr)
		}
		groups = append(groups, manifestGroup)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(repoRoot), cliapp.EvidenceExportInput{
		Scenario:    projectCLIName,
		ManifestRaw: manifestRaw,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func loadManifestGroupForEvidence(raw []byte, path string) (cliapp.SubcommandGroup, error) {
	manifest, err := cliapp.ParseManifest(raw)
	if err != nil {
		return cliapp.SubcommandGroup{}, err
	}
	group := manifest.FindGroup(path)
	if group == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("group %q not found", path)
	}
	bindings := make(map[string]func(cliapp.RunContext) error, len(group.Commands))
	for _, command := range group.Commands {
		key := command.Binding.Handler
		bindings[key] = func(cliapp.RunContext) error { return nil }
	}
	return cliapp.LoadFromManifest(raw, path, bindings)
}
