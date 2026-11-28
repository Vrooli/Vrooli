package checks

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// RecoveryGateCheck is the detector for the restart gate itself (R6: a repair
// path that cannot read its coordination state must say so). A runtime registry
// newer than this binary is critical and names the fix, even when nothing is
// failing right now: the next failure would otherwise find its repair held.
type RecoveryGateCheck struct{ HomeDir string }

func (c RecoveryGateCheck) ID() string                 { return "vrooli-recovery-gate" }
func (c RecoveryGateCheck) Title() string              { return "Restart Gate Readable" }
func (c RecoveryGateCheck) Category() Category         { return CategorySystem }
func (c RecoveryGateCheck) IntervalSeconds() int       { return 60 }
func (c RecoveryGateCheck) Platforms() []platform.Type { return nil }
func (c RecoveryGateCheck) Description() string {
	return "Opens the runtime registry the restart gate reads, and reports a registry newer than this binary"
}
func (c RecoveryGateCheck) Importance() string {
	return "Guards the 2026-11-20 outage class where a control-plane schema bump silently skipped every restart until autoheal was rebuilt (RULES.md R6)"
}

func (c RecoveryGateCheck) Run(ctx context.Context) Result {
	result := Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: c.HomeDir})
	if err != nil {
		verdict := registryErrorVerdict(err)
		result.Status = StatusCritical
		result.Message = verdict.Reason
		result.Details["gateClass"] = verdict.Class
		return result
	}
	defer store.Close()
	epochs, err := store.ListPressureEpochs(ctx, 1)
	if err != nil {
		result.Status = StatusCritical
		result.Message = registryErrorVerdict(err).Reason
		return result
	}
	result.Status = StatusOK
	result.Message = "runtime registry readable by the restart gate"
	if len(epochs) > 0 {
		result.Details["pressureEpoch"] = fmt.Sprintf("%s (%s)", epochs[0].EpochID, epochs[0].Status)
	}
	return result
}
