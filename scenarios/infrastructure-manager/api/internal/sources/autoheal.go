package sources

import (
	"context"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	actionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions"
	actionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/actions/actions_v1connect"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
)

// AutohealReader reads only persisted, typed autoheal observations. It does
// not assign trust or band verdicts; the condition domain owns those rules.
type AutohealReader struct {
	Resolver    *discovery.Resolver
	HTTP        *http.Client
	Projection  string
	WindowHours int32
}

func (r AutohealReader) Read(ctx context.Context) ([]Observation, error) {
	if r.Projection != "availability" {
		return nil, nil
	}
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
	if err != nil {
		return nil, err
	}
	window := r.WindowHours
	if window <= 0 {
		window = 24
	}
	actionsClient := actionsconnect.NewActionsServiceClient(httpClient, base)
	trendResponse, err := actionsClient.GetPerCheckTrends(ctx, connect.NewRequest(&actionsv1.GetPerCheckTrendsRequest{WindowHours: window}))
	if err != nil {
		return nil, err
	}
	checksClient := checksconnect.NewChecksServiceClient(httpClient, base)
	reconcileResponse, reconcileErr := checksClient.GetReconcile(ctx, connect.NewRequest(&checksv1.GetReconcileRequest{}))
	shelvesResponse, shelvesErr := checksClient.ListShelves(ctx, connect.NewRequest(&checksv1.ListShelvesRequest{}))
	ghosts := map[string]bool{}
	if reconcileErr == nil && reconcileResponse.Msg.GetReconcile().GetAvailable() {
		for _, id := range reconcileResponse.Msg.GetReconcile().GetGhostCheckIds() {
			ghosts[id] = true
		}
	}
	shelves := map[string]bool{}
	if shelvesErr == nil {
		for _, shelf := range shelvesResponse.Msg.GetShelves() {
			shelves[shelf.GetCheckId()] = true
		}
	}
	checkedAt := time.Now().UTC()
	out := make([]Observation, 0, len(trendResponse.Msg.GetTrends()))
	for _, trend := range trendResponse.Msg.GetTrends() {
		hints := TrustHints{Ghost: ghosts[trend.GetCheckId()], Shelved: shelves[trend.GetCheckId()], UnitMatches: true}
		if reconcileErr != nil || shelvesErr != nil {
			hints.Untrusted = true
		}
		saturation, saturationErr := checksClient.GetSaturation(ctx, connect.NewRequest(&checksv1.GetSaturationRequest{CheckId: trend.GetCheckId(), WindowHours: window}))
		if saturationErr == nil {
			hints.Saturated = !saturation.Msg.GetTransitioned()
		} else {
			hints.Untrusted = true
		}
		observedAt := checkedAt
		if trend.GetLastChecked() != nil {
			observedAt = trend.GetLastChecked().AsTime()
		}
		out = append(out, Observation{
			ID: trend.GetCheckId(), CellRef: "availability/A1", Value: trend.GetUptimePercent(), Unit: "percent",
			Source: "vrooli-autoheal/actions.GetPerCheckTrends", ObservedAt: observedAt, TrustHints: hints,
		})
	}
	return out, nil
}
