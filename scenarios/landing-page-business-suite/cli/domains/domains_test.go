package domains

import (
	"testing"

	"landing-page-business-suite/cli/internal/support"
	"landing-page-business-suite/cli/internal/testutil"
)

func TestCommandGroupsExposeValidRegistrationContracts(t *testing.T) {
	for _, group := range CommandGroups(support.Dependencies{}) {
		testutil.AssertCommandGroup(t, group)
	}
}
