package findings

import "context"

// notifyingService decorates a Service so notify fires after every successful
// content mutation. main.go points notify at the aisearch sync loop's Kick, so
// a fresh capture becomes searchable within the kick debounce instead of
// waiting out the periodic sync interval (the interval remains the repair
// path). Read paths and usage telemetry pass through untouched — they never
// change indexable content.
type notifyingService struct {
	Service
	notify func()
}

// WithMutationNotify wraps svc so notify runs after each successful mutation
// (Add/Edit/Supersede/Flag/ResolveDispute/Prune). notify must be non-blocking
// (SyncLoop.Kick is). A nil notify returns svc unchanged.
func WithMutationNotify(svc Service, notify func()) Service {
	if notify == nil {
		return svc
	}
	return &notifyingService{Service: svc, notify: notify}
}

func (n *notifyingService) Add(ctx context.Context, in NewFinding) (Finding, error) {
	f, err := n.Service.Add(ctx, in)
	if err == nil {
		n.notify()
	}
	return f, err
}

func (n *notifyingService) Edit(ctx context.Context, id string, in EditInput) (Finding, error) {
	f, err := n.Service.Edit(ctx, id, in)
	if err == nil {
		n.notify()
	}
	return f, err
}

func (n *notifyingService) Supersede(ctx context.Context, id, replacement, reason string) (Finding, error) {
	f, err := n.Service.Supersede(ctx, id, replacement, reason)
	if err == nil {
		n.notify()
	}
	return f, err
}

func (n *notifyingService) Flag(ctx context.Context, id, reason string) (Finding, error) {
	f, err := n.Service.Flag(ctx, id, reason)
	if err == nil {
		n.notify()
	}
	return f, err
}

func (n *notifyingService) ResolveDispute(ctx context.Context, id, resolution, replacement, reason string) (Finding, error) {
	f, err := n.Service.ResolveDispute(ctx, id, resolution, replacement, reason)
	if err == nil {
		n.notify()
	}
	return f, err
}

func (n *notifyingService) Prune(ctx context.Context, dryRun bool) ([]string, error) {
	ids, err := n.Service.Prune(ctx, dryRun)
	if err == nil && !dryRun && len(ids) > 0 {
		n.notify()
	}
	return ids, err
}
