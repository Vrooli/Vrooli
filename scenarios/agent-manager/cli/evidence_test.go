package main

import (
	"os"
	"testing"

	"agent-manager/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("assemble Agent Manager CLI: %v", err)
	}
	manifest, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read CLI manifest: %v", err)
	}
	groups := domains.SubcommandGroups(app.dependencies())
	input := cliapp.EvidenceExportInput{
		Scenario:    "agent-manager",
		ManifestRaw: manifest,
		Groups:      groups,
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(input)
	if err != nil {
		t.Fatalf("build primitive evidence: %v", err)
	}
	if got := artifact.ObservedPrimitives()["run event-page"]; got != cliapp.PrimitiveProtoList {
		t.Fatalf("run event-page observed primitive = %q, want %q", got, cliapp.PrimitiveProtoList)
	}
	parsed, err := cliapp.ParseManifest(manifest)
	if err != nil {
		t.Fatalf("parse CLI manifest: %v", err)
	}
	declared := false
	var visit func([]cliapp.ManifestGroup)
	visit = func(groups []cliapp.ManifestGroup) {
		for _, group := range groups {
			for _, command := range group.Commands {
				if group.Name == "run" && command.Name == "event-page" &&
					command.Architecture != nil && command.Architecture.Primitive == string(cliapp.PrimitiveProtoList) &&
					command.Binding.Service == "AgentManagerService" && command.Binding.Method == "GetRunEvents" {
					declared = true
				}
			}
			visit(group.Groups)
		}
	}
	visit(parsed.Groups)
	if !declared {
		t.Fatal("run event-page manifest declaration must bind GetRunEvents as proto_list")
	}

	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), input, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}
