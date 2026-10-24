package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"secrets-manager/cli/domains"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the committed cli-core evidence
// artifact synchronized with the manifest and the actual registered handlers.
// Regeneration is explicit through UPDATE_CLI_EVIDENCE=1.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups := domains.SubcommandGroups(nil, manifestBytes)
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "secrets-manager",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
