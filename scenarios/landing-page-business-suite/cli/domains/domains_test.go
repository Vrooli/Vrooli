package domains

import (
	"testing"

	"landing-page-business-suite/cli/internal/support"
	"landing-page-business-suite/cli/internal/testutil"
)

func TestCommandGroupsExposeValidRegistrationContracts(t *testing.T) {
	for _, group := range CommandGroups(support.Dependencies{}) {
		if err := testutil.ValidateCommandGroup(group); err != nil {
			t.Fatalf("ValidateCommandGroup(%q) error = %v", group.Title, err)
		}
	}
}
