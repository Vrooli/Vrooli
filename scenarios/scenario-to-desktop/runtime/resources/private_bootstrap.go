package resources

import (
	"context"
	"fmt"
)

// privateServiceBootstrapper is deliberately resource-native. The generic
// supervisor owns artifact verification and process lifetime; it must not
// acquire resource-specific bootstrap, recovery, or credential knowledge.
type privateServiceBootstrapper func(context.Context, Item, map[string]int, string) (map[string]string, error)

var privateServiceBootstrappers = map[string]privateServiceBootstrapper{}

func registerPrivateServiceBootstrapper(resource string, bootstrap privateServiceBootstrapper) {
	if resource == "" || bootstrap == nil {
		panic("private service bootstrapper requires resource and implementation")
	}
	if _, exists := privateServiceBootstrappers[resource]; exists {
		panic("duplicate private service bootstrapper for " + resource)
	}
	privateServiceBootstrappers[resource] = bootstrap
}

func bootstrapPrivateService(ctx context.Context, item Item, ports map[string]int, appDataDir string) (map[string]string, error) {
	bootstrap, ok := privateServiceBootstrappers[item.Resource]
	if !ok {
		return nil, nil
	}
	if item.Service == nil {
		return nil, fmt.Errorf("missing managed service declaration")
	}
	return bootstrap(ctx, item, ports, appDataDir)
}
