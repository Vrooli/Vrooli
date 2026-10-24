package main

import (
	"os"
	"testing"

	"react-component-library/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

// TestEveryDeclaredPrimitiveHasEvidence keeps the static architecture artifact
// synchronized with the assembled command tree. CLI Health consumes this file
// without executing scenario commands.
func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(&cliapp.ScenarioApp{}, manifestBytes)
	if err != nil {
		t.Fatalf("build subcommand groups: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "react-component-library",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
