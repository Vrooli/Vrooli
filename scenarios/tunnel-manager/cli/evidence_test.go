package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"tunnel-manager/cli/domains"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the committed primitive evidence
// artifact aligned with the manifest-backed command tree. It never executes an
// API call; cli-core primitives record their implementation evidence at build
// time.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "tunnel-manager",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
