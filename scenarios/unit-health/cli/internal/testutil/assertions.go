package testutil

import (
	"strings"
	"testing"
)

// RequireContains keeps repeated CLI surface assertions consistent without
// reimplementing cli-core's command or output fakes.
func RequireContains(t *testing.T, value string, wanted ...string) {
	t.Helper()
	for _, item := range wanted {
		if !strings.Contains(value, item) {
			t.Errorf("output %q does not contain %q", value, item)
		}
	}
}
