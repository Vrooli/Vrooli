// Package capabilitycatalog assembles registered control-plane providers for
// generic setup, onboarding, and CLI consumers.
package capabilitycatalog

import (
	"context"

	"github.com/vrooli/vrooli/internal/credentialescrow"
	"github.com/vrooli/vrooli/internal/durablebackup"
	"github.com/vrooli/vrooli/internal/operatorcapability"
)

func New(root, home string) (*operatorcapability.Registry, error) {
	providers := credentialescrow.NewProviders(root, home)
	providers = append(providers, durablebackup.NewProvider())
	return operatorcapability.NewRegistry(providers...)
}

func Discover(ctx context.Context, root, home string) ([]operatorcapability.Status, error) {
	registry, err := New(root, home)
	if err != nil {
		return nil, err
	}
	return registry.Discover(ctx)
}
