package testutil_test

import (
	"testing"

	"github.com/vrooli/cli-core/cliapptest"
)

func TestNoProductionImports(t *testing.T) {
	cliapptest.AssertNoProductionImports(t, "../../", "/internal/testutil")
}
