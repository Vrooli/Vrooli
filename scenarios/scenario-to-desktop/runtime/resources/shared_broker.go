package resources

import (
	"context"
	"fmt"
	"strings"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// SharedServiceBinding is an ephemeral, app-scoped connection handed to the
// desktop runtime after the broker has verified the shared instance. Its
// environment is intentionally not exposed through status or telemetry.
type SharedServiceBinding struct {
	Endpoint    string
	Environment map[string]string
	ExpiresAt   time.Time
}

// SharedServiceResolver is supplied by an embedding application only after a
// user has explicitly consented to shared-resource reuse. A nil resolver keeps
// the bundle fully private.
type SharedServiceResolver interface {
	ResolveSharedService(context.Context, Item) (SharedServiceBinding, error)
}

// SharedBrokerGrant is the narrow result a control-plane broker adapter may
// return after it has acquired and authorized a scoped use lease.
type SharedBrokerGrant struct {
	Endpoint   string
	Credential string
	ExpiresAt  time.Time
}

// SharedBrokerClient is implemented by the Vrooli control-plane adapter. The
// desktop runtime deliberately cannot pass a raw endpoint or choose a scope;
// both remain bound to the authenticated broker credential held by that
// adapter.
type SharedBrokerClient interface {
	GrantSharedService(context.Context, string, time.Duration) (SharedBrokerGrant, error)
}

// BrokerSharedServiceConfig binds one resource to an already scoped broker
// client. Environment values may reference ${RESOURCE_ENDPOINT} and
// ${RESOURCE_CREDENTIAL}; the latter is never persisted or reported.
type BrokerSharedServiceConfig struct {
	Client      SharedBrokerClient
	Environment map[string]string
}

// BrokerSharedServiceResolver obtains a use lease and resource-native scoped
// credential from the loopback Vrooli broker. It has no lifecycle authority.
type BrokerSharedServiceResolver struct {
	Resources            map[string]BrokerSharedServiceConfig
	LeaseTTL             time.Duration
	SharedReuseConsented bool
}

func (r BrokerSharedServiceResolver) ResolveSharedService(ctx context.Context, item Item) (SharedServiceBinding, error) {
	if item.Service == nil {
		return SharedServiceBinding{}, fmt.Errorf("resource %s has no managed service declaration", item.Resource)
	}
	if _, err := serviceProviderPolicy(item.Service).ResolveProvider(resourcedeployment.ProviderRequest{
		Mode: resourcedeployment.ProviderManagedShared, SharedConsented: r.SharedReuseConsented,
	}); err != nil {
		return SharedServiceBinding{}, err
	}
	configuration, ok := r.Resources[item.Resource]
	if !ok {
		return SharedServiceBinding{}, fmt.Errorf("no shared broker configuration for %s", item.Resource)
	}
	if configuration.Client == nil {
		return SharedServiceBinding{}, fmt.Errorf("shared broker client is required for %s", item.Resource)
	}
	ttl := r.LeaseTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	grant, err := configuration.Client.GrantSharedService(ctx, item.Resource, ttl)
	if err != nil {
		return SharedServiceBinding{}, fmt.Errorf("obtain shared %s broker grant: %w", item.Resource, err)
	}
	if strings.TrimSpace(grant.Endpoint) == "" || strings.TrimSpace(grant.Credential) == "" || grant.ExpiresAt.IsZero() {
		return SharedServiceBinding{}, fmt.Errorf("broker returned an invalid scoped %s grant", item.Resource)
	}
	environment := make(map[string]string, len(configuration.Environment))
	for key, value := range configuration.Environment {
		key = strings.TrimSpace(key)
		if key == "" {
			return SharedServiceBinding{}, fmt.Errorf("shared %s environment contains an empty key", item.Resource)
		}
		value = strings.ReplaceAll(value, "${RESOURCE_ENDPOINT}", grant.Endpoint)
		value = strings.ReplaceAll(value, "${RESOURCE_CREDENTIAL}", grant.Credential)
		environment[key] = value
	}
	return SharedServiceBinding{Endpoint: grant.Endpoint, Environment: environment, ExpiresAt: grant.ExpiresAt}, nil
}
