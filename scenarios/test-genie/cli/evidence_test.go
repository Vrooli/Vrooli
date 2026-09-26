package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"test-genie/cli/domains"
	"test-genie/cli/domains/suites"
	"test-genie/cli/internal/deps"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	group := suites.Register(deps.Runtime{})
	subgroups, err := domains.SubcommandGroups(manifestBytes, deps.Runtime{})
	if err != nil {
		t.Fatalf("register manifest-backed subcommands: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "test-genie",
		ManifestRaw: manifestBytes,
		TopLevel:    group.Commands,
		Groups:      subgroups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func TestExecuteExceptionHasObservedDurableEvidence(t *testing.T) {
	group := suites.Register(deps.Runtime{})
	subgroups, err := domains.SubcommandGroups(manifestBytes, deps.Runtime{})
	if err != nil {
		t.Fatalf("register manifest-backed subcommands: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "test-genie",
		ManifestRaw: manifestBytes,
		TopLevel:    group.Commands,
		Groups:      subgroups,
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	observed := artifact.ObservedPrimitives()["execute"]
	if observed != cliapp.PrimitiveDurableRun {
		t.Fatalf("execute observed primitive = %q, want %q", observed, cliapp.PrimitiveDurableRun)
	}
}
