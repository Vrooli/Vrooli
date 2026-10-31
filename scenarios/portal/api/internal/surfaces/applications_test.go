package surfaces

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplicationsResolveOnlyExistingRunningIdentities(t *testing.T) {
	resolutions := 0
	apps := &Applications{Run: func(context.Context) ([]byte, error) {
		return []byte(`{"success":true,"scenarios":[{"name":"nutrition-planner","status":"running","description":"Meals"},{"name":"portal","status":"available"},{"name":"../escape","status":"running"}]}`), nil
	}, ResolveURL: func(_ context.Context, name, host string) (string, error) {
		resolutions++
		require.Equal(t, "nutrition-planner", name)
		require.Equal(t, "192.168.1.173:16000", host)
		return "http://192.168.1.173:22000", nil
	}}
	list, err := apps.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, name := range []string{"portal", "unknown", "../escape"} {
		_, _, err := apps.Resolve(context.Background(), name, "192.168.1.173:16000")
		require.Error(t, err)
	}
	require.Zero(t, resolutions, "stopped and unknown apps must not trigger discovery or startup")
	_, address, err := apps.Resolve(context.Background(), "nutrition-planner", "192.168.1.173:16000")
	require.NoError(t, err)
	require.Equal(t, "http://192.168.1.173:22000", address)
}

func TestApplicationsGatewayRequiresKnownRunningTargetAndOwnerPorts(t *testing.T) {
	calls := 0
	apps := &Applications{Run: func(context.Context) ([]byte, error) {
		return []byte(`{"success":true,"scenarios":[{"name":"nutrition-planner","status":"running"},{"name":"portal","status":"running"},{"name":"stopped","status":"available"}]}`), nil
	}, ResolvePort: func(_ context.Context, name, key string) (int, error) {
		calls++
		require.Equal(t, "nutrition-planner", name)
		if key == "UI_PORT" {
			return 20002, nil
		}
		return 17000, nil
	}}
	for _, name := range []string{"portal", "../escape", "unknown", "stopped"} {
		_, _, err := apps.ResolveGateway(context.Background(), name)
		require.Error(t, err)
	}
	require.Zero(t, calls)
	app, path, err := apps.ResolveGateway(context.Background(), "nutrition-planner")
	require.NoError(t, err)
	require.Equal(t, "nutrition-planner", app.Scenario)
	require.Equal(t, "/apps/nutrition-planner/proxy/", path)
	target, err := apps.ProxyTarget(context.Background(), "nutrition-planner")
	require.NoError(t, err)
	require.Equal(t, map[string]int{"UI_PORT": 20002, "API_PORT": 17000}, target["port_mappings"])
	require.Equal(t, 2, calls)
}
