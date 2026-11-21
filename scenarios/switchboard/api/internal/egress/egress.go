package egress

import (
	"context"
	"fmt"
	"switchboard/internal/channels"
)

type Router struct{ Registry *channels.Registry }

func (r Router) Send(ctx context.Context, out channels.Outbound) error {
	if r.Registry == nil {
		return fmt.Errorf("channel registry unavailable")
	}
	d, ok := r.Registry.Get(out.ChannelID)
	if !ok {
		return fmt.Errorf("channel %q not found", out.ChannelID)
	}
	if err := channels.ValidateOutbound(d, out); err != nil {
		return err
	}
	a, ok := r.Registry.Adapter(out.ChannelID)
	if !ok {
		return fmt.Errorf("channel %q has no adapter", out.ChannelID)
	}
	return a.Send(ctx, out)
}
