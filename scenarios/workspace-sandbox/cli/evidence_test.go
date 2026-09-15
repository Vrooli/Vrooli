package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"workspace-sandbox/cli/domains"
	"workspace-sandbox/cli/internal/support"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the committed primitive-evidence
// artifact in lockstep with the manifest and the real command registration.
// CLI Health reads this artifact without executing scenario commands.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(support.Dependencies{}, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "workspace-sandbox",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(support.Dependencies{}, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "workspace-sandbox",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	observed := artifact.ObservedPrimitives()
	manifest, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			primitive := command.Architecture.CommandArchitecture().Primitive
			if primitive == "" {
				continue
			}
			path := group.Name + " " + command.Name
			got, ok := observed[path]
			if !ok {
				t.Errorf("command %q declares primitive %q but has no observed evidence", path, primitive)
				continue
			}
			if cliapp.ClassifyPrimitiveEvidence(primitive, got) != cliapp.EvidenceVerified {
				t.Errorf("command %q declared %q but observed %q", path, primitive, got)
			}
		}
	}
}
