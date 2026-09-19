package landing

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
testutil "landing-page-business-suite/cli/internal/testutil"
	"landing-page-business-suite/cli/internal/support"
)

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	group := Register(support.Dependencies{})
	if err := testutil.ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
	for _, command := range group.Commands {
		if command.Name == "variant-space" && command.PrimitiveEvidence() != cliapp.PrimitiveAction {
			t.Fatalf("variant-space primitive = %q, want %q", command.PrimitiveEvidence(), cliapp.PrimitiveAction)
		}
	}
}
