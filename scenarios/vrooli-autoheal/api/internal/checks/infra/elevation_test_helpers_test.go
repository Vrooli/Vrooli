package infra

import (
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/elevation"
)

func allowRecoveryGrant(t *testing.T) {
	t.Helper()
	restore := elevation.SetGrantPathForTest("test-vrooli-autoheal-grant", func() bool { return true })
	t.Cleanup(restore)
}
