package adapters

import (
	"context"
	"fmt"

	"switchboard/internal/channels"
)

// Validate is shared by every adapter test and keeps transport behavior
// honest without importing any channel SDK into the core.
func Validate(ctx context.Context, adapter channels.Adapter, descriptor channels.Descriptor) error {
	if adapter.ID() != descriptor.ID {
		return fmt.Errorf("adapter id %q does not match descriptor %q", adapter.ID(), descriptor.ID)
	}
	if probe := adapter.Probe(ctx); !probe.Available {
		return fmt.Errorf("adapter unavailable: %s", probe.Reason)
	}
	return nil
}
