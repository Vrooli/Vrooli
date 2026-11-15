package bindings

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/discovery"
)

func TestExecuteOwnsReachabilityAdmissionBeforeDispatch(t *testing.T) {
	registry := fixtureRegistry(t, `{"name":"program-runtime","groups":[{"name":"records","commands":[{"name":"list","binding":{"kind":"connect-rpc","service":"BindingRegistryService","method":"ListBindings"},"governance":{"effect":"read","run_eligible":true}}]}]}`)
	binding := registry.bindings[0]
	binding.Scenario = "fixture-target"
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) {
		return "", &discovery.Error{Kind: discovery.ErrScenarioNotRunning}
	})
	dispatched := false
	client := &http.Client{Transport: bindingRoundTripper(func(*http.Request) (*http.Response, error) {
		dispatched = true
		return nil, errors.New("unexpected dispatch")
	})}

	_, err := registry.Execute(context.Background(), binding.GetId(), map[string]any{}, nil, false, InvocationMetadata{}, client)
	require.ErrorContains(t, err, "is unreachable")
	require.False(t, dispatched, "Go admission must reject an unreachable target before transport")
}
