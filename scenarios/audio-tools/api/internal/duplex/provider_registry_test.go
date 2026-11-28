package duplex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderRegistryRejectsInvalidAndDuplicateRegistrations(t *testing.T) {
	provider := &fakeProvider{}
	_, err := NewProviderRegistry(ProviderRegistration{ID: "", Kind: ProviderKindRealtimeDuplex, Provider: provider})
	require.Error(t, err)
	_, err = NewProviderRegistry(ProviderRegistration{ID: "live", Kind: "batch", Provider: provider})
	require.Error(t, err)
	_, err = NewProviderRegistry(
		ProviderRegistration{ID: "live", Kind: ProviderKindRealtimeDuplex, Provider: provider},
		ProviderRegistration{ID: "live", Kind: ProviderKindRealtimeDuplex, Provider: provider},
	)
	require.Error(t, err)
}

func TestRoutedCoordinatorRequiresExplicitKnownDuplexProviderBeforeAdmission(t *testing.T) {
	provider := &fakeProvider{}
	registry, err := NewProviderRegistry(ProviderRegistration{ID: "openai-gpt-live", Kind: ProviderKindRealtimeDuplex, Provider: provider})
	require.NoError(t, err)
	admission := &fakeAdmission{}
	coordinator, err := NewRoutedCoordinator(registry, admission)
	require.NoError(t, err)

	_, err = coordinator.Open(context.Background(), Request{ProviderID: "unknown"})
	require.ErrorIs(t, err, ErrRegisteredProviderUnavailable)
	require.Empty(t, admission.requests, "unknown routes must fail before budget reservation")
	require.False(t, provider.openSeen, "unknown routes must not contact a provider")

	ctx, cancel := context.WithCancel(context.Background())
	handle, err := coordinator.Open(ctx, Request{ProviderID: "openai-gpt-live"})
	require.NoError(t, err)
	require.Len(t, admission.requests, 1)
	require.True(t, provider.openSeen)
	require.NoError(t, handle.Close("test-complete"))
	cancel()
}
