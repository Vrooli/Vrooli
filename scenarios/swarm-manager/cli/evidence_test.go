package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"swarm-manager/cli/domains"
	"swarm-manager/cli/internal/support"
)

// TestPrimitiveEvidenceArtifactCurrent keeps the generated evidence artifact
// that CLI Health consumes synchronized with the real registered command tree.
// Registration constructs primitive handlers but never invokes their operations,
// so a zero dependency set is sufficient and has no scenario side effects.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	groups := domains.SubcommandGroups(support.Dependencies{})
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "swarm-manager",
		ManifestRaw: manifest,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "swarm-manager",
		ManifestRaw: manifest,
		Groups:      domains.SubcommandGroups(support.Dependencies{}),
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	observed := artifact.ObservedPrimitives()
	for _, group := range parsed.Groups {
		for _, command := range group.Commands {
			declared := command.Architecture.CommandArchitecture().Primitive
			if declared == "" {
				continue
			}
			path := group.Name + " " + command.Name
			if got := observed[path]; cliapp.ClassifyPrimitiveEvidence(declared, got) != cliapp.EvidenceVerified {
				t.Errorf("%s declares %q but observed primitive is %q", path, declared, got)
			}
		}
	}
}
