package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"music-tools/cli/domains"
)

// TestPrimitiveEvidenceArtifactCurrent keeps this scenario's committed static
// primitive-evidence artifact (the canonical .vrooli/generated location) in
// lockstep with the manifest and handlers. It assembles the real command tree —
// which wires handler closures but never executes them (a nil core suffices; the
// primitive builders record observed evidence at construction) — exports evidence,
// and either regenerates the artifact (UPDATE_CLI_EVIDENCE=1) or fails if the
// committed file is stale/missing.
//
// The scaffold generates the artifact once via a postHook; from then on this test
// is the guard that catches it drifting from the CLI it describes. CLI Health reads
// this exact committed artifact to award verified L4, so it must never go stale.
// The test runs with the cli/ directory as its working dir, so the scenario root
// is one level up.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "music-tools",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

// TestEveryDeclaredPrimitiveHasEvidence proves the scenario has NO declared-only
// commands: every manifest command that declares an architecture.primitive is
// backed by matching observed cli-core evidence. This is the reference-shape
// guard — a new command declared but not built with a cli-core primitive fails
// here rather than silently claiming verified L4 from manifest text.
func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "music-tools",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	})
	if err != nil {
		t.Fatalf("build evidence: %v", err)
	}
	observed := artifact.ObservedPrimitives()

	m, err := cliapp.ParseManifest(manifestBytes)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	for _, g := range m.Groups {
		for _, c := range g.Commands {
			arch := c.Architecture.CommandArchitecture()
			if arch.Primitive == "" {
				continue
			}
			path := g.Name + " " + c.Name
			got, ok := observed[path]
			if !ok {
				t.Errorf("command %q declares primitive %q but has no observed evidence (build the handler with the matching cli-core primitive and register via LoadFromManifestPrimitives)", path, arch.Primitive)
				continue
			}
			if cliapp.ClassifyPrimitiveEvidence(arch.Primitive, got) != cliapp.EvidenceVerified {
				t.Errorf("command %q declared %q but observed %q — not verified", path, arch.Primitive, got)
			}
		}
	}
}
