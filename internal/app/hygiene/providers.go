package hygiene

import (
	"context"
	"fmt"
	"time"
)

// Provider is one registered hygiene check surface. Providers append their
// checks/findings/actions into the shared report so the root hygiene command can
// aggregate scenario- or domain-specific hygiene without hardcoding every rule
// in Service.Run.
type Provider interface {
	ID() string
	Run(ctx context.Context, req Request, report *Report) error
}

// BudgetedProvider makes a provider's lane budget explicit. Providers without
// a budget are still usable for tests and extensions, but production providers
// must implement this interface so a slow check cannot silently expand hygiene.
type BudgetedProvider interface {
	Provider
	Budget() time.Duration
}

// Registry owns the ordered hygiene provider set.
type Registry struct {
	order    []string
	provider map[string]Provider
}

func NewRegistry(providers ...Provider) Registry {
	r := Registry{provider: map[string]Provider{}}
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

func (r *Registry) Register(p Provider) {
	if p == nil || p.ID() == "" {
		return
	}
	id := p.ID()
	if _, exists := r.provider[id]; !exists {
		r.order = append(r.order, id)
	}
	r.provider[id] = p
}

func (r Registry) Run(ctx context.Context, req Request, report *Report, ids ...string) error {
	if len(ids) == 0 {
		ids = r.order
	}
	for _, id := range ids {
		p, ok := r.provider[id]
		if !ok {
			return fmt.Errorf("hygiene provider %q is not registered", id)
		}
		providerCtx := ctx
		var cancel context.CancelFunc
		budgeted, hasBudget := p.(BudgetedProvider)
		if hasBudget && budgeted.Budget() > 0 {
			providerCtx, cancel = context.WithTimeout(ctx, budgeted.Budget())
		}
		started := time.Now()
		err := p.Run(providerCtx, req, report)
		if cancel != nil {
			cancel()
		}
		if hasBudget && budgeted.Budget() > 0 && time.Since(started) > budgeted.Budget() {
			elapsed := time.Since(started).Round(time.Millisecond)
			report.addCheck("hygiene_provider_budget_"+id, true, SeverityWarning, fmt.Sprintf("Hygiene provider %s exceeded its %s budget (%s)", id, budgeted.Budget(), elapsed))
			report.addFinding(Finding{Severity: SeverityWarning, Code: "hygiene_provider_budget", Message: fmt.Sprintf("Provider %s exceeded its declared %s budget; measured %s", id, budgeted.Budget(), elapsed), Why: "Hygiene lane budgets keep slow providers observable without turning timing drift into a commit failure."})
			if providerCtx.Err() == context.DeadlineExceeded {
				err = nil
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
