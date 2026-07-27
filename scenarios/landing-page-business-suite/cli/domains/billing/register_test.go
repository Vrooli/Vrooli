package billing

import (
	"testing"

	"landing-page-business-suite/cli/internal/support"
	"landing-page-business-suite/cli/internal/testutil"
)

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	testutil.AssertCommandGroup(t, Register(support.Dependencies{}))
}
