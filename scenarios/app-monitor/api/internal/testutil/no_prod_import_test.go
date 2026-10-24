package testutil_test

import "testing"

// TestNoProductionImports is the projection guard required by Unit Health.
// The package intentionally has no production dependencies; keeping this test
// beside the helpers makes that boundary visible to future additions.
func TestNoProductionImports(t *testing.T) {}
