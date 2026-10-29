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

// TopActions returns up to n plans for the window, evaluated for that window
// alone, largest combined saving first. Each action is the plan's largest
// lever with that lever's own saving and recoverable figure, so the text and
// the numbers describe the same change.
func (a AlertActions) TopActions(ctx context.Context, w *usageforecast.Window, n int, now time.Time) ([]usageforecast.Action, error) {
	key := WindowKey{Provider: w.Provider, Pool: w.Pool, Window: w.Window}
	report, _, err := a.Service.Recommendations(ctx, w.Provider, key, now)
	if err != nil {
		return nil, err
	}
	var actions []usageforecast.Action
	for _, plan := range report.Plans {
		lever := plan.Levers[0]
		actions = append(actions, usageforecast.Action{Text: lever.Action, Saving: lever.Saved, Recoverable: lever.Recoverable})
		if len(actions) == n {
			break
		}
	}
	return actions, nil
}
