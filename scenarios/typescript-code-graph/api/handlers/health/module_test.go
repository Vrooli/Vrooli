package health_test

import (
	"testing"

	"github.com/vrooli/api-core/databasetest"
	"typescript-code-graph/handlers/health"

	"github.com/stretchr/testify/require"
)

// TestModule_Shape pins the public contract.
func TestModule_Shape(t *testing.T) {
	m := health.Module(&databasetest.FakePinger{}, "react-vite-test", "1.0.0")

	require.Equal(t, "health", m.Name)
	require.NotNil(t, m.Mount)
	require.Same(t, &health.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice")
	require.Len(t, health.Endpoints, 1, "health ships a single endpoint with /api/v1/health as a path alias")
}
