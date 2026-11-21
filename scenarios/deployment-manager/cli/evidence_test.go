package main

import (
	"os"
	"testing"

	"deployment-manager/cli/domains"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups := domains.SubcommandGroups(nil, manifestBytes)
	artifactPath := cliapp.EvidenceArtifactPath("..")
	cliapptest.RequirePrimitiveEvidence(t, artifactPath, cliapp.EvidenceExportInput{
		Scenario:    "deployment-manager",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("primitive evidence artifact is unavailable after validation: %v", err)
	}
}

func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups := domains.SubcommandGroups(nil, manifestBytes)
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "deployment-manager",
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
