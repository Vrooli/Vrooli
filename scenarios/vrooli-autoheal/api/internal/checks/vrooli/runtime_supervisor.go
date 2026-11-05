package vrooli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

type RuntimeSupervisorCheck struct {
	client *integration.Client
}

type RuntimeSupervisorCheckOption func(*RuntimeSupervisorCheck)

func WithRuntimeSupervisorClient(client *integration.Client) RuntimeSupervisorCheckOption {
	return func(c *RuntimeSupervisorCheck) {
		c.client = client
	}
}

func WithRuntimeSupervisorExecutor(executor checks.CommandExecutor) RuntimeSupervisorCheckOption {
	return func(c *RuntimeSupervisorCheck) {
		c.client = integration.NewClient(executor)
	}
}

func NewRuntimeSupervisorCheck(opts ...RuntimeSupervisorCheckOption) *RuntimeSupervisorCheck {
	c := &RuntimeSupervisorCheck{
		client: integration.NewClient(checks.DefaultExecutor),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *RuntimeSupervisorCheck) ID() string    { return "vrooli-runtime-supervisor" }
func (c *RuntimeSupervisorCheck) Title() string { return "Runtime Supervisor" }
func (c *RuntimeSupervisorCheck) Description() string {
	return "Verifies Vrooli's runtime supervisor process and heartbeat are live"
}

func (c *RuntimeSupervisorCheck) Importance() string {
	return "Scenario CLI port discovery depends on fresh runtime registry supervision"
}
func (c *RuntimeSupervisorCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *RuntimeSupervisorCheck) IntervalSeconds() int       { return 60 }
func (c *RuntimeSupervisorCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *RuntimeSupervisorCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: map[string]interface{}{},
	}
	if checkOS != "linux" {
		result.Status = checks.StatusNotApplicable
		result.Message = "Runtime supervisor check is not implemented on this platform"
		result.Details["platform"] = checkOS
		return result
	}
	status, _, err := c.client.RuntimeSupervisorStatus(ctx)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read runtime supervisor status"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["status"] = status.Status
	result.Details["supervisorId"] = status.SupervisorID
	result.Details["heartbeatDeadlineAt"] = status.HeartbeatDeadlineAt
	if status.PID != nil {
		result.Details["pid"] = *status.PID
	}
	if status.StatusReason != "" {
		result.Details["statusReason"] = status.StatusReason
	}

	switch status.Status {
	case "running":
		result.Status = checks.StatusOK
		result.Message = "Runtime supervisor is running with a fresh heartbeat"
	case "stale", "dead", "stopped", "":
		result.Status = checks.StatusCritical
		if status.Status == "" {
			result.Message = "Runtime supervisor status is unavailable"
		} else if status.StatusReason != "" {
			result.Message = fmt.Sprintf("Runtime supervisor is %s: %s", status.Status, status.StatusReason)
		} else {
			result.Message = fmt.Sprintf("Runtime supervisor is %s", status.Status)
		}
	default:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Runtime supervisor reported status %s", status.Status)
	}
	return result
}

func (c *RuntimeSupervisorCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	healthy := lastResult != nil && lastResult.Status == checks.StatusOK
	return []checks.RecoveryAction{
		{
			ID:          "install-start",
			Name:        "Install and Start Runtime Supervisor",
			Description: "Run `vrooli runtime supervisor install --user`, which installs/enables the user service and starts it",
			Dangerous:   false,
			Available:   !healthy,
		},
		{
			ID:          "status",
			Name:        "Show Runtime Supervisor Status",
			Description: "Read current runtime supervisor status from Vrooli",
			Dangerous:   false,
			Available:   true,
		},
	}
}

func (c *RuntimeSupervisorCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}
	switch actionID {
	case "install-start":
		output, err := c.client.InstallRuntimeSupervisor(ctx)
		result.Duration = time.Since(start)
		result.Output = strings.TrimSpace(string(output))
		if err != nil {
			result.Success = false
			result.Message = "Failed to install/start runtime supervisor"
			result.Error = err.Error()
			return result
		}
		result.Success = true
		result.Message = "Runtime supervisor service installed and started"
		return result
	case "status":
		_, output, err := c.client.RuntimeSupervisorStatus(ctx)
		result.Duration = time.Since(start)
		result.Output = strings.TrimSpace(string(output))
		result.Success = err == nil
		if err != nil {
			result.Error = err.Error()
		}
		return result
	default:
		result.Duration = time.Since(start)
		result.Success = false
		result.Error = "unknown action: " + actionID
		return result
	}
}

var _ checks.HealableCheck = (*RuntimeSupervisorCheck)(nil)
