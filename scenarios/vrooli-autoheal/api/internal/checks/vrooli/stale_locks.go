// Package vrooli provides Vrooli-specific health checks
// [REQ:STALE-LOCK-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package vrooli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"

	integration "github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/integrations/vrooli"
)

// StaleLockCheck summarizes core-maintained port lock state.
type StaleLockCheck struct {
	warningThreshold  int
	criticalThreshold int
	executor          checks.CommandExecutor
	client            *integration.Client
}

type StaleLockCheckOption func(*StaleLockCheck)

func WithStaleLockThresholds(warning, critical int) StaleLockCheckOption {
	return func(c *StaleLockCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

func WithStaleLockExecutor(executor checks.CommandExecutor) StaleLockCheckOption {
	return func(c *StaleLockCheck) {
		c.executor = executor
		c.client = integration.NewClient(executor)
	}
}

func WithStaleLockClient(client *integration.Client) StaleLockCheckOption {
	return func(c *StaleLockCheck) {
		c.client = client
	}
}

func NewStaleLockCheck(opts ...StaleLockCheckOption) *StaleLockCheck {
	c := &StaleLockCheck{
		warningThreshold:  3,
		criticalThreshold: 10,
		executor:          checks.DefaultExecutor,
		client:            integration.NewClient(checks.DefaultExecutor),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *StaleLockCheck) ID() string    { return "vrooli-stale-locks" }
func (c *StaleLockCheck) Title() string { return "Registry Claim Hygiene" }
func (c *StaleLockCheck) Description() string {
	return "Summarizes Vrooli core registry claim hygiene and legacy port-lock artifact buildup"
}

func (c *StaleLockCheck) Importance() string {
	return "Stale registry claims or leftover legacy lock files can confuse diagnostics; allocation authority lives in the registry"
}

func (c *StaleLockCheck) Category() checks.Category  { return checks.CategoryInfrastructure }
func (c *StaleLockCheck) IntervalSeconds() int       { return 60 }
func (c *StaleLockCheck) Platforms() []platform.Type { return nil }

func (c *StaleLockCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	claims, _, err := c.client.ListRegistryClaims(ctx)
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to read registry claims"
		result.Details["error"] = err.Error()
		return result
	}

	// Allocation authority lives in the registry. A claim is "stale" when core
	// reconciliation recommends expiring it (non-authoritative); orphan-listener
	// claims are a distinct problem that `vrooli cleanup locks` does not resolve,
	// so they are surfaced separately rather than counted as cleanable.
	staleClaims := make([]integration.RegistryClaim, 0)
	orphanListeners := 0
	for _, claim := range claims {
		switch {
		case claim.StaleExpireRecommended():
			staleClaims = append(staleClaims, claim)
		case claim.OrphanListenerSuspected():
			orphanListeners++
		}
	}

	staleCount := len(staleClaims)
	result.Details["staleCount"] = staleCount
	result.Details["totalClaims"] = len(claims)
	result.Details["orphanListenerCount"] = orphanListeners
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold
	if staleCount > 0 {
		limit := 10
		if staleCount < limit {
			limit = staleCount
		}
		result.Details["staleClaims"] = staleClaims[:limit]
	}

	score := 100
	if staleCount > 0 {
		score = 100 - (staleCount * 10)
		if score < 0 {
			score = 0
		}
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "stale-claim-count",
				Passed: staleCount < c.criticalThreshold,
				Detail: fmt.Sprintf("%d registry claims recommended for expiry", staleCount),
			},
		},
	}

	switch {
	case staleCount >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Critical: %d stale registry claims detected", staleCount)
	case staleCount >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Warning: %d stale registry claims detected", staleCount)
	case staleCount > 0:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("%d stale registry claims (below threshold)", staleCount)
	default:
		result.Status = checks.StatusOK
		result.Message = "No stale registry claims detected"
	}

	return result
}

func (c *StaleLockCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasStale := false
	if lastResult != nil {
		if count, ok := lastResult.Details["staleCount"].(int); ok {
			hasStale = count > 0
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "clean",
			Name:        "Clean Stale Registry Claims and Legacy Artifacts",
			Description: "Delegate cleanup to `vrooli cleanup locks` — expires non-authoritative registry claims and prunes leftover legacy lock files",
			Dangerous:   false,
			Available:   hasStale,
		},
		{
			ID:          "list",
			Name:        "List Lock Diagnostics",
			Description: "Show registry claim and legacy artifact state reported by core Vrooli maintenance commands",
			Dangerous:   false,
			Available:   true,
		},
	}
}

func (c *StaleLockCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "list":
		return c.executeList(ctx, start)
	case "clean":
		return c.executeClean(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

func (c *StaleLockCheck) executeList(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "list",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	claims, output, err := c.client.ListRegistryClaims(ctx)
	result.Duration = time.Since(start)
	result.Output = string(output)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result
	}

	staleCount := 0
	for _, claim := range claims {
		if claim.StaleExpireRecommended() {
			staleCount++
		}
	}
	result.Success = true
	result.Message = fmt.Sprintf("Found %d stale registry claims out of %d total", staleCount, len(claims))
	return result
}

func (c *StaleLockCheck) executeClean(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "clean",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	report, output, err := c.client.CleanupLocks(ctx)
	result.Duration = time.Since(start)
	result.Output = strings.TrimSpace(string(output))
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Message = "Failed to clean stale locks"
		return result
	}

	result.Success = len(report.Failed) == 0
	result.Message = report.Message
	if !result.Success {
		errors := make([]string, 0, len(report.Failed))
		for _, item := range report.Failed {
			errors = append(errors, item.Name+": "+item.Error)
		}
		result.Error = strings.Join(errors, "; ")
	}
	return result
}

type PortDiagnostics = integration.PortDiagnostic

func DiagnosePort(port int, scenario string, executor checks.CommandExecutor) (*PortDiagnostics, error) {
	client := integration.NewClient(executor)
	diagnostic, _, err := client.DiagnosePort(context.Background(), strconv.Itoa(port), scenario)
	if err != nil {
		return nil, err
	}
	return &diagnostic, nil
}

func AutoRecoverPort(port int, scenario string, executor checks.CommandExecutor) (bool, string, error) {
	client := integration.NewClient(executor)

	lockReport, _, lockErr := client.CleanupLocks(context.Background())
	orphanReport, _, orphanErr := client.CleanupOrphans(context.Background())
	if lockErr != nil {
		return false, "", lockErr
	}
	if orphanErr != nil {
		return false, "", orphanErr
	}

	changes := make([]string, 0, 2)
	if len(lockReport.Stopped) > 0 {
		changes = append(changes, lockReport.Message)
	}
	if len(orphanReport.Stopped) > 0 {
		changes = append(changes, orphanReport.Message)
	}
	if len(changes) == 0 {
		diagnostic, _, err := client.DiagnosePort(context.Background(), strconv.Itoa(port), scenario)
		if err != nil {
			return false, "", err
		}
		if len(diagnostic.Recommendations) > 0 {
			return false, strings.Join(diagnostic.Recommendations, "; "), nil
		}
		return false, "No automated recovery action was available", nil
	}

	return true, strings.Join(changes, "; "), nil
}

var _ checks.HealableCheck = (*StaleLockCheck)(nil)
