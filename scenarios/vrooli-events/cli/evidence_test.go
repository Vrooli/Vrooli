package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"vrooli-events/cli/domains"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "vrooli-events",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
