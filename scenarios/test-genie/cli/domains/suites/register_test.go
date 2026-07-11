package suites

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"

	"test-genie/cli/internal/deps"
)

// TestExecuteCarriesDurableRunEvidence proves the execute command is registered as
// a verified durable_run exception: it declares the durable_run exception AND
// carries matching cli-core durable_run evidence (via WithLegacyPrimitive), so its
// server-owned start->follow lifecycle is proven rather than merely asserted.
// Building the group wires closures only — no command runs — so nil clients are
// safe here.
func TestExecuteCarriesDurableRunEvidence(t *testing.T) {
	group := Register(deps.Runtime{})

	var execute *cliapp.Command
	for i := range group.Commands {
		if group.Commands[i].Name == "execute" {
			execute = &group.Commands[i]
			break
		}
	}
	if execute == nil {
		t.Fatalf("suites group has no execute command")
	}

	if execute.PrimitiveEvidence() != cliapp.PrimitiveDurableRun {
		t.Fatalf("execute observed evidence = %q, want durable_run", execute.PrimitiveEvidence())
	}
	if execute.Architecture.Exception != cliapp.ExceptionDurableRun {
		t.Fatalf("execute declared exception = %q, want durable_run", execute.Architecture.Exception)
	}
	// The declared exception and observed primitive must not contradict.
	if execute.PrimitiveEvidence().SatisfiesException() != execute.Architecture.Exception {
		t.Fatalf("execute observed primitive %q does not satisfy declared exception %q",
			execute.PrimitiveEvidence(), execute.Architecture.Exception)
	}
	if execute.Run == nil {
		t.Fatalf("execute must keep its legacy Run handler")
	}
}

func TestRemediateCarriesActionEvidence(t *testing.T) {
	group := Register(deps.Runtime{})

	var remediate *cliapp.Command
	for i := range group.Commands {
		if group.Commands[i].Name == "remediate" {
			remediate = &group.Commands[i]
			break
		}
	}
	if remediate == nil {
		t.Fatalf("suites group has no remediate command")
	}

	if remediate.PrimitiveEvidence() != cliapp.PrimitiveAction {
		t.Fatalf("remediate observed evidence = %q, want action", remediate.PrimitiveEvidence())
	}
	if remediate.Architecture.Primitive != cliapp.PrimitiveAction {
		t.Fatalf("remediate declared primitive = %q, want action", remediate.Architecture.Primitive)
	}
	if remediate.RunCtx == nil {
		t.Fatalf("remediate must use the RunCtx primitive path")
	}
	if remediate.Run != nil {
		t.Fatalf("remediate should not keep a legacy Run handler")
	}
}
