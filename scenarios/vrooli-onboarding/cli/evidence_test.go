package main

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"vrooli-onboarding/cli/domains"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the committed primitive evidence
// beside the manifest and the command tree that produces it. The test builds
// handlers but never executes them; cli-core stamps the observed primitive at
// construction time.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	commandGroups, subcommandGroups, err := domains.ManifestCommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	groups := append([]cliapp.SubcommandGroup(nil), subcommandGroups...)
	for _, group := range commandGroups {
		groups = append(groups, cliapp.SubcommandGroup{Name: strings.ToLower(group.Title), Subcommands: group.Commands})
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    appName,
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
