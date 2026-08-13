package domains

import (
	"testing"

	testutil "github.com/vrooli/cli-core/cliapptest"
	"landing-page-business-suite/cli/internal/support"
)

func TestCommandGroupsExposeValidRegistrationContracts(t *testing.T) {
	for _, group := range CommandGroups(support.Dependencies{}) {
		if err := testutil.ValidateCommandGroup(group); err != nil {
			t.Fatalf("ValidateCommandGroup(%q) error = %v", group.Title, err)
		}
	}
}
