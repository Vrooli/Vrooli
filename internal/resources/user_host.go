package resources

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/resources/securestore"
)

const userResourceHostID = "user-resource-host"

type managedSharedBootstrapper func(context.Context, *UserResourceHost, ManagedInstance, string) error

var managedSharedBootstrappers = map[string]managedSharedBootstrapper{}

func registerManagedSharedBootstrapper(resource string, bootstrap managedSharedBootstrapper) {
	if strings.TrimSpace(resource) == "" || bootstrap == nil {
		panic("managed shared bootstrapper requires a resource and implementation")
	}
	if _, exists := managedSharedBootstrappers[resource]; exists {
		panic("duplicate managed shared bootstrapper for " + resource)
	}
	managedSharedBootstrappers[resource] = bootstrap
}

func managedSharedBootstrapperFor(resource string) (managedSharedBootstrapper, bool) {
	bootstrap, ok := managedSharedBootstrappers[resource]
	return bootstrap, ok
}

func defaultManagedSharedSecureStore() securestore.Store { return securestore.Default() }

// UserResourceHost is the per-user resource authority. Its durable state is
// intentionally limited to broker identity and authorization metadata;
// resource-native recovery material resides solely in the operating-system
// secret store.
type UserResourceHost struct {
	Broker     *Broker
	Secrets    securestore.Store
	OwnerScope string
}

// SecureStorageReady proves store, read, and delete behavior before any
// service initialization changes durable resource state.
func (h *UserResourceHost) SecureStorageReady(instanceID string) error {
	if h == nil || h.Secrets == nil || strings.TrimSpace(instanceID) == "" {
		return fmt.Errorf("user resource host secure store is unavailable")
	}
	if err := securestore.Probe(h.Secrets); err != nil {
		return fmt.Errorf("verify secure storage: %w", err)
	}
	return nil
}

// OpenUserResourceHost restores the per-user, credential-free broker state.
// A caller may inject a Store in tests; production callers should use
// securestore.Default and fail closed when the OS facility is unavailable.
func OpenUserResourceHost(store securestore.Store, ownerScope string) (*UserResourceHost, error) {
	if store == nil {
		return nil, fmt.Errorf("user resource host secure store is required")
	}
	ownerScope = strings.TrimSpace(ownerScope)
	if ownerScope == "" {
		return nil, fmt.Errorf("user resource host owner scope is required")
	}
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{AppID: "vrooli"})
	if err != nil {
		return nil, err
	}
	paths, err := runtimestorage.EnsureAllDirs(resolver, runtimestorage.Options{ResourceID: userResourceHostID}, 0o700)
	if err != nil {
		return nil, err
	}
	broker, err := NewPersistentBroker(nil, FileBrokerStore{Path: filepath.Join(paths.StateDir, "broker.json")})
	if err != nil {
		return nil, fmt.Errorf("restore user resource host: %w", err)
	}
	return newUserResourceHost(broker, store, ownerScope)
}

func newUserResourceHost(broker *Broker, store securestore.Store, ownerScope string) (*UserResourceHost, error) {
	if broker == nil || store == nil || strings.TrimSpace(ownerScope) == "" {
		return nil, fmt.Errorf("user resource host requires broker, secure store, and owner scope")
	}
	return &UserResourceHost{Broker: broker, Secrets: store, OwnerScope: ownerScope}, nil
}
