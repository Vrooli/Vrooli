package main

import (
	"os"
	"testing"

	"search-hub/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(&cliapp.ScenarioApp{}, manifestBytes)
	if err != nil {
		t.Fatalf("build subcommand groups: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "search-hub",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
