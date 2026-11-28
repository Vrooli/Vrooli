package validation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/metrics"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestValidateScenarioPopulatesMetricsStages(t *testing.T) {
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: cleanSurface()}})

	collector := metrics.Start()
	_, err := svc.ValidateScenario(WithMetrics(context.Background(), collector), "demo")
	require.NoError(t, err)
	out := collector.Stop()

	require.NotNil(t, out)
	require.GreaterOrEqual(t, len(out.GetStages()), 2, "expect at least discover + analyze stages")

	names := map[string]bool{}
	var analyze *commonv1.Stage
	for _, st := range out.GetStages() {
		names[st.GetName()] = true
		if st.GetName() == "analyze" {
			analyze = st
		}
	}
	require.True(t, names["discover"], "discover stage missing")
	require.True(t, names["analyze"], "analyze stage missing")
	require.NotNil(t, analyze, "analyze stage missing")
	require.GreaterOrEqual(t, len(analyze.GetChildren()), 1, "analyze should nest at least one child")

	childNames := map[string]bool{}
	for _, ch := range analyze.GetChildren() {
		childNames[ch.GetName()] = true
	}
	require.True(t, childNames["static-checks"], "static-checks nested child missing")
	require.NotNil(t, analyze.GetResources(), "hot stage should carry per-stage resources")
}

func TestValidateScenarioNilCollectorIsSafe(t *testing.T) {
	svc := newTestService(t, Deps{Loader: fakeLoader{surface: cleanSurface()}})
	// No collector in context: stage calls are no-ops, validation still succeeds.
	_, err := svc.ValidateScenario(context.Background(), "demo")
	require.NoError(t, err)
}
