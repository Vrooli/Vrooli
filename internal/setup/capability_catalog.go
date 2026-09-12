package setup

import (
	"context"

	"github.com/vrooli/vrooli/internal/credentialescrow"
	"github.com/vrooli/vrooli/internal/durablebackup"
	"github.com/vrooli/vrooli/internal/operatorcapability"
	"github.com/vrooli/vrooli/internal/storageapproval"
)

// NewCapabilityRegistry assembles the providers used by setup and capability
// workflows. Setup owns this wiring because it is the control-plane bootstrap
// composition root.
func NewCapabilityRegistry(root, home string) (*operatorcapability.Registry, error) {
	providers := credentialescrow.NewProviders(root, home)
	providers = append(providers, durablebackup.NewProvider())
	providers = append(providers, storageapproval.New())
	return operatorcapability.NewRegistry(providers...)
}

func DiscoverCapabilities(ctx context.Context, root, home string) ([]operatorcapability.Status, error) {
	registry, err := NewCapabilityRegistry(root, home)
	if err != nil {
		return nil, err
	}
	return registry.Discover(ctx)
}

// DiscoverCapabilityInventory produces the deterministic descriptor inventory
// used by catalogs and search. Registration remains local to this composition
// root; callers cannot inject an executable provider through the inventory
// surface.
func DiscoverCapabilityInventory(ctx context.Context, root, home string) (operatorcapability.Inventory, error) {
	registry, err := NewCapabilityRegistry(root, home)
	if err != nil {
		return operatorcapability.Inventory{}, err
	}
	return registry.Inventory(ctx)
}
