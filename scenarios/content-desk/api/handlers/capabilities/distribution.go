package capabilities

import (
	"context"

	"content-desk/integrations/channelmanager"
	internalcapabilities "content-desk/internal/capabilities"
)

// channelManagerDistributionResolver reads current distribution-surface
// connectivity from Channel Manager through Content Desk's sole outbound
// boundary. It is read-only and never mutates the channel owner.
type channelManagerDistributionResolver struct {
	client *channelmanager.Client
}

func newChannelManagerDistributionResolver() internalcapabilities.DistributionResolver {
	return channelManagerDistributionResolver{client: channelmanager.NewClient()}
}

// ConnectedSurfaces maps the boundary projection to the capability domain. The
// integration keeps its own state type so the domain does not depend on the
// transport package.
func (r channelManagerDistributionResolver) ConnectedSurfaces(ctx context.Context) (internalcapabilities.DistributionState, error) {
	state, err := r.client.ConnectedSurfaces(ctx)
	if err != nil {
		return internalcapabilities.DistributionState{}, err
	}
	return internalcapabilities.DistributionState{
		Connected:  state.Connected,
		Source:     state.Source,
		ObservedAt: state.ObservedAt,
	}, nil
}
