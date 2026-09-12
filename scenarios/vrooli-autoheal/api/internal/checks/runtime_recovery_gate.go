package checks

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// RuntimeRecoveryGate reads the runtime registry's durable pressure epoch. A
// read failure fails closed for restart actions: losing this coordination must
// not create two recovery controllers under host pressure.
type RuntimeRecoveryGate struct {
	HomeDir string
}

func (g RuntimeRecoveryGate) AllowsAutoHealRestart(ctx context.Context, _, _ string) (bool, string) {
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: g.HomeDir})
	if err != nil {
		return false, fmt.Sprintf("runtime recovery ownership unavailable: %v", err)
	}
	defer store.Close()
	epochs, err := store.ListPressureEpochs(ctx, 1)
	if err != nil {
		return false, fmt.Sprintf("runtime recovery ownership unavailable: %v", err)
	}
	if len(epochs) == 0 {
		return true, ""
	}
	epoch := epochs[0]
	if epoch.Status == scenarioruntime.PressureEpochDetected || epoch.Status == scenarioruntime.PressureEpochRegressed || epoch.Status == scenarioruntime.PressureEpochGated {
		return false, fmt.Sprintf("runtime pressure epoch %s owns recovery (%s)", epoch.EpochID, epoch.Status)
	}
	return true, ""
}
