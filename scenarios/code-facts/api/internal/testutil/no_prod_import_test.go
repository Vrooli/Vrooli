package testutil_test

import (
	"testing"

	"github.com/vrooli/api-core/testguard"
)

func TestNoProductionImports(t *testing.T) {
	testguard.AssertNoProductionTestutilImports(t, "../..")
}
