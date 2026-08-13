package feedback

import (
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"
	"landing-page-business-suite/cli/internal/support"
)

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	group := Register(support.Dependencies{})
	if err := testutil.ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
}
