package bundleruntime

import (
	"errors"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/scenarios/scenario-to-desktop/runtime/api"
)

func TestFirstAvailableFeatureProviderUsesStandaloneInstalledBundleOrder(t *testing.T) {
	tests := []struct {
		name       string
		available  []string
		wantSource string
		wantCalls  []string
	}{
		{name: "standalone wins", available: []string{"full_app", "component", "embedded"}, wantSource: "full_app", wantCalls: []string{"standalone"}},
		{name: "installed machine dependency precedes bundle", available: []string{"", "component", "embedded"}, wantSource: "component", wantCalls: []string{"standalone", "installed"}},
		{name: "bundle is removal fallback", available: []string{"", "", "embedded"}, wantSource: "embedded", wantCalls: []string{"standalone", "installed", "bundle"}},
		{name: "no provider", available: []string{"", "", ""}, wantCalls: []string{"standalone", "installed", "bundle"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls []string
			resolvers := make([]featureProviderResolver, 0, len(tt.available))
			labels := []string{"standalone", "installed", "bundle"}
			for i := range tt.available {
				source := tt.available[i]
				label := labels[i]
				resolvers = append(resolvers, func() (api.FeatureResolution, error) {
					calls = append(calls, label)
					return api.FeatureResolution{Source: source}, nil
				})
			}

			selected, err := firstAvailableFeatureProvider(resolvers...)
			if err != nil {
				t.Fatalf("select provider: %v", err)
			}
			if selected.Source != tt.wantSource {
				t.Fatalf("source = %q, want %q", selected.Source, tt.wantSource)
			}
			if !reflect.DeepEqual(calls, tt.wantCalls) {
				t.Fatalf("provider calls = %v, want %v", calls, tt.wantCalls)
			}
		})
	}
}

func TestFirstAvailableFeatureProviderFailsClosedBeforeLowerPriority(t *testing.T) {
	wantErr := errors.New("standalone provider is unhealthy")
	calls := []string{}
	selected, err := firstAvailableFeatureProvider(
		func() (api.FeatureResolution, error) {
			calls = append(calls, "standalone")
			return api.FeatureResolution{}, wantErr
		},
		func() (api.FeatureResolution, error) {
			calls = append(calls, "installed")
			return api.FeatureResolution{Source: "component"}, nil
		},
	)
	if !errors.Is(err, wantErr) || selected.Source != "" {
		t.Fatalf("selection = %+v, %v; want standalone error", selected, err)
	}
	if !reflect.DeepEqual(calls, []string{"standalone"}) {
		t.Fatalf("provider calls = %v, want standalone only", calls)
	}
}
