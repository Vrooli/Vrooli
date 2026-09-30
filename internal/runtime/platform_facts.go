package runtime

import (
	"context"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func currentPlatformFacts() hostinventory.Snapshot {
	return hostinventory.CollectPlatformFacts(context.Background())
}

func currentPlatformOS() string {
	return hostinventory.CurrentPlatform()
}
