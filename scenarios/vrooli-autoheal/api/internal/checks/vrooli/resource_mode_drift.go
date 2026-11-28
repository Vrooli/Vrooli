// [REQ:RESOURCE-CHECK-001] [REQ:HEAL-ACTION-001]
package vrooli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
)

// ModeDriftCheck asks one question the resource check cannot: is this resource
// on the accelerator backend it declared?
//
// It exists separately from ResourceCheck because the two have different
// remedies and different urgencies. A stopped resource needs a start; a
// resource serving from the CPU needs a restart only once the host can actually
// give it a device, and restarting it before then just costs a cold start for
// nothing.
type ModeDriftCheck struct {
	resourceName   string
	interval       int
	executor       checks.CommandExecutor
	client         *integration.Client
	statusProvider integration.ResourceStatusSnapshotProvider
}

// ModeDriftCheckOption configures a ModeDriftCheck.
type ModeDriftCheckOption func(*ModeDriftCheck)

// WithModeDriftExecutor sets the command executor (for testing).
func WithModeDriftExecutor(executor checks.CommandExecutor) ModeDriftCheckOption {
	return func(c *ModeDriftCheck) {
		c.executor = executor
		c.client = integration.NewClient(executor)
	}
}

// WithModeDriftStatusProvider shares the fleet snapshot with the base resource
// check. A mode-drift finding still escalates through the named typed status
// path when a fresh fast snapshot reports drift.
func WithModeDriftStatusProvider(provider integration.ResourceStatusSnapshotProvider) ModeDriftCheckOption {
	return func(c *ModeDriftCheck) { c.statusProvider = provider }
}

// NewModeDriftCheck creates the accelerator placement check for one resource.
func NewModeDriftCheck(resourceName string, opts ...ModeDriftCheckOption) *ModeDriftCheck {
	c := &ModeDriftCheck{
		resourceName: resourceName,
		interval:     120,
		executor:     checks.DefaultExecutor,
		client:       integration.NewClient(checks.DefaultExecutor),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ModeDriftCheck) ID() string    { return "resource-" + c.resourceName + "-mode-drift" }
func (c *ModeDriftCheck) Title() string { return c.resourceName + " Accelerator Placement" }

func (c *ModeDriftCheck) Description() string {
	return "Reports whether the " + c.resourceName + " resource is running on the accelerator backend it declared"
}

func (c *ModeDriftCheck) Importance() string {
	return "A resource serving from the CPU while it declared an accelerator answers every request correctly and does so 10x to 54x slower, with every other status surface reporting healthy"
}

func (c *ModeDriftCheck) Category() checks.Category  { return checks.CategoryResource }
func (c *ModeDriftCheck) IntervalSeconds() int       { return c.interval }
func (c *ModeDriftCheck) Platforms() []platform.Type { return nil }

// ResourceName returns the resource this check watches.
func (c *ModeDriftCheck) ResourceName() string { return c.resourceName }

func (c *ModeDriftCheck) HealTarget() checks.HealTarget {
	return checks.HealTarget{Kind: "resource", Name: c.resourceName}
}

func (c *ModeDriftCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{CheckID: c.ID(), Details: make(map[string]interface{})}

	var status integration.ResourceStatus
	var err error
	if c.statusProvider != nil {
		var snapshot integration.ResourceStatusSnapshot
		snapshot, err = c.statusProvider.Snapshot(ctx, []string{c.resourceName})
		now := snapshotNow(c.statusProvider)
		addSnapshotDetails(result.Details, snapshot, now)
		addSnapshotMetrics(result.Details, c.statusProvider)
		if err == nil && snapshot.FreshAt(now) {
			var ok bool
			status, ok = snapshot.Resources[c.resourceName]
			if !ok {
				err = fmt.Errorf("snapshot does not contain %s", c.resourceName)
			}
			if err == nil && status.ModeDrift {
				recordSnapshotEscalation(c.statusProvider, "mode_drift")
				deep, deepErr := c.statusProvider.DeepStatus(ctx, c.resourceName)
				result.Details["deepProbeAttempted"] = true
				result.Details["escalationReason"] = "mode_drift"
				if deepErr == nil {
					status = deep
					result.Details["snapshotSource"] = string(integration.SnapshotSourceNamedDeep)
					result.Details["snapshotProbeLevel"] = string(integration.ProbeLevelDeep)
				} else {
					result.Details["deepProbeError"] = deepErr.Error()
				}
				addSnapshotMetrics(result.Details, c.statusProvider)
			}
		} else if err == nil {
			err = fmt.Errorf("resource status snapshot is stale or degraded")
		}
	} else {
		status, _, err = c.client.ResourceStatus(ctx, c.resourceName)
	}
	if err != nil {
		// A status read that fails says nothing about placement. Report
		// undetermined rather than inventing a placement verdict.
		recordSnapshotUndetermined(c.statusProvider)
		result.Status = checks.StatusUndetermined
		result.Message = "could not read " + c.resourceName + " accelerator placement"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["running"] = status.Running
	result.Details["declaredMode"] = status.DeclaredMode
	result.Details["observedMode"] = status.ObservedMode
	result.Details["modeDrift"] = status.ModeDrift

	switch {
	case strings.TrimSpace(status.DeclaredMode) == "":
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " declares no accelerator backend"
	case !status.Running:
		// A stopped resource has no placement. The resource check owns
		// "stopped"; duplicating it here would double every incident.
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " is not running, so it has no placement to verify"
	case status.ModeDrift:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("%s declared %s but is running on %s", c.resourceName, status.DeclaredMode, observedOrUnknown(status.ObservedMode))
		if reason := strings.TrimSpace(status.ModeReason); reason != "" {
			result.Message += ": " + reason
		}
	case strings.TrimSpace(status.ObservedMode) == "":
		result.Status = checks.StatusWarning
		result.Message = c.resourceName + " placement could not be read from the host"
	default:
		result.Status = checks.StatusOK
		result.Message = c.resourceName + " is running on its declared " + status.DeclaredMode + " backend"
	}
	return result
}

func observedOrUnknown(mode string) string {
	if strings.TrimSpace(mode) == "" {
		return "an unknown backend"
	}
	return mode
}

// RecoveryActions offers the one action that can move a resource back onto its
// declared backend, and offers it only when the host can actually provide one.
func (c *ModeDriftCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	drifted := false
	if lastResult != nil {
		drifted, _ = lastResult.Details["modeDrift"].(bool)
	}
	return []checks.RecoveryAction{
		{
			ID:          "restart",
			Name:        "Restart",
			Description: "Restart " + c.resourceName + " so it re-selects its accelerator backend",
			Dangerous:   true,
			Available:   drifted,
		},
		{
			ID:          "explain",
			Name:        "Explain Placement",
			Description: "Show " + c.resourceName + " declared backends, host readiness and observed placement",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs a recovery action for this check.
func (c *ModeDriftCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: start}

	var args []string
	switch actionID {
	case "restart":
		args = []string{"resource", "restart", c.resourceName}
	case "explain":
		args = []string{"resource", "acceleration", "explain", c.resourceName}
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Message = "Action not recognized"
		result.Duration = time.Since(start)
		return result
	}

	output, err := c.executor.CombinedOutput(ctx, "vrooli", args...)
	result.Output = string(output)
	result.Duration = time.Since(start)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Action failed: " + actionID
		return result
	}
	result.Success = true
	switch actionID {
	case "restart":
		result.Message = c.resourceName + " restarted; its placement is re-verified on the next check"
	case "explain":
		result.Message = "Retrieved accelerator placement for " + c.resourceName
	}
	return result
}
