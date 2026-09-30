package content

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
	for _, name := range []string{"seo", "admin-variant-seo-update"} {
		found := false
		for _, command := range group.Commands {
			if command.Name != name {
				continue
			}
			found = true
			if command.PrimitiveEvidence() != cliapp.PrimitiveAction {
				t.Fatalf("%s primitive = %q, want %q", name, command.PrimitiveEvidence(), cliapp.PrimitiveAction)
			}
		}
		if !found {
			t.Fatalf("%s command was not registered", name)
		}
	}
}
