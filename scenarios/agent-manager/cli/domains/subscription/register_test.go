package subscription

import (
	"testing"

	"agent-manager/cli/internal/support"
)

func TestRegister(t *testing.T) {
	group := Register(support.Dependencies{})
	if group.Title != "Subscription" || len(group.Commands) != 1 {
		t.Fatalf("unexpected subscription command group: %+v", group)
	}
}
