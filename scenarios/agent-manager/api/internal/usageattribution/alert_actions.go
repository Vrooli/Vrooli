package usageattribution

import (
	"context"
	"time"

	"agent-manager/internal/usageforecast"
)

// AlertActions adapts the recommendations service to the actions a usage
// notification names. It is the one place alerts read recommendations.
type AlertActions struct{ Service Service }

var _ usageforecast.ActionSource = AlertActions{}

// TopActions returns up to n quota levers for the window, evaluated for that
// window alone, highest score first: an alert is about quota, so tool and
// operator actions are left to the Usage page.
func (a AlertActions) TopActions(ctx context.Context, w *usageforecast.Window, n int, now time.Time) ([]usageforecast.Action, error) {
	key := WindowKey{Provider: w.Provider, Pool: w.Pool, Window: w.Window}
	report, _, err := a.Service.Recommendations(ctx, w.Provider, key, now)
	if err != nil {
		return nil, err
	}
	var actions []usageforecast.Action
	for _, action := range report.Actions {
		if action.Source != SourceLever || action.Effect.Unit != UnitPoints || action.Score <= 0 || action.Effect.Basis == BasisPerTrim || action.Provider != w.Provider {
			continue
		}
		actions = append(actions, usageforecast.Action{Text: action.Text, Saving: action.Effect.Value, Recoverable: action.Recoverable})
		if len(actions) == n {
			break
		}
	}
	return actions, nil
}
