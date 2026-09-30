package cliapptest

import (
	"bytes"
	"os"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// RequirePrimitiveEvidence keeps a scenario's committed static primitive-evidence
// artifact honest. It builds fresh evidence from the scenario's assembled command
// tree (which never executes a handler) and either (re)writes the artifact or
// asserts the committed file already matches:
//
//   - update=true  → (re)write path from freshly built evidence. Wire this to an
//     env toggle (e.g. UPDATE_CLI_EVIDENCE=1) so a maintainer can regenerate.
//   - update=false → fail if the committed artifact is missing or stale. This is
//     the CI guard that catches an artifact drifting from the manifest/handlers.
//
// Because the artifact is committed, CLI Health can read it statically to award
// verified L4 without ever running the scenario's commands (plan decision D1).
func RequirePrimitiveEvidence(tb testing.TB, path string, input cliapp.EvidenceExportInput, update bool) {
	tb.Helper()

	artifact, err := cliapp.BuildPrimitiveEvidence(input)
	if err != nil {
		tb.Fatalf("build primitive evidence for %q: %v", input.Scenario, err)
		return
	}
	want, err := cliapp.MarshalPrimitiveEvidence(artifact)
	if err != nil {
		tb.Fatalf("marshal primitive evidence for %q: %v", input.Scenario, err)
		return
	}

	if update {
		if err := cliapp.WritePrimitiveEvidence(path, artifact); err != nil {
			tb.Fatalf("write primitive evidence %q: %v", path, err)
		}
		return
	}

	got, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("read committed primitive evidence %q: %v (regenerate by running this test with UPDATE_CLI_EVIDENCE=1)", path, err)
		return
	}
	if !bytes.Equal(got, want) {
		tb.Fatalf("committed primitive evidence %q is stale — regenerate by running this test with UPDATE_CLI_EVIDENCE=1", path)
	}
}
