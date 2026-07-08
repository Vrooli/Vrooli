package main

import (
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"

	"cli-health/cli/domains"
)

// TestPrimitiveEvidenceArtifactCurrent keeps cli-health's committed static
// primitive-evidence artifact (the canonical .vrooli/generated location) in
// lockstep with the manifest and handlers. It assembles the real command tree —
// which wires handler closures but never executes them (a nil core is enough; the
// primitive builders record observed evidence at construction) — exports evidence,
// and either regenerates the artifact (UPDATE_CLI_EVIDENCE=1) or fails if the
// committed file is stale/missing.
//
// This is what makes cli-health a true verified-L4 reference adopter: CLI Health
// reads this exact committed artifact to prove each declared primitive, so the
// artifact must never drift from the CLI it describes. The test runs with the cli/
// directory as its working dir, so the scenario root is one level up.
func TestPrimitiveEvidenceArtifactCurrent(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(".."), cliapp.EvidenceExportInput{
		Scenario:    "cli-health",
		ManifestRaw: manifestBytes,
		Groups:      groups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

// TestEveryDeclaredPrimitiveHasEvidence proves cli-health has NO declared-only
// commands: every manifest command that declares an architecture.primitive is
// backed by matching observed cli-core evidence. A command declared but not
// primitive-built would leave the artifact without observed evidence for its
// path, failing here — the reference adopter is not allowed to claim L4 from
// manifest text alone.
func TestEveryDeclaredPrimitiveHasEvidence(t *testing.T) {
	groups, err := domains.SubcommandGroups(nil, manifestBytes)
	if err != nil {
		t.Fatalf("assemble command tree: %v", err)
	}
	artifact, err := cliapp.BuildPrimitiveEvidence(cliapp.EvidenceExportInput{
		Scenario:    "cli-health",
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
				continue // TestManifestDeclaresArchitectureOnEveryCommand covers this
			}
			path := g.Name + " " + c.Name
			got, ok := observed[path]
			if !ok {
				t.Errorf("command %q declares primitive %q but has no observed evidence (build the handler with the matching cli-core primitive)", path, arch.Primitive)
				continue
			}
			if cliapp.ClassifyPrimitiveEvidence(arch.Primitive, got) != cliapp.EvidenceVerified {
				t.Errorf("command %q declared %q but observed %q — not verified", path, arch.Primitive, got)
			}
		}
	}
}
