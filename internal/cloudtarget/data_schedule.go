package cloudtarget

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/packages/recoverypoint"
)

// DBMRunner runs the DBM owner CLI on the protected target. The CLI talks to
// the target-local DBM service; it is never run by the scenario-to-cloud
// controller against remote source paths.
type DBMRunner func(context.Context, ...string) ([]byte, error)

// DataSchedulePolicy is the non-secret plan configuration passed through the
// reviewed scenario-to-cloud action graph to the target-local DBM owner.
type DataSchedulePolicy struct {
	DestinationName       string `json:"destination_name"`
	Schedule              string `json:"schedule"`
	KeepHourly            int32  `json:"keep_hourly"`
	KeepDaily             int32  `json:"keep_daily"`
	KeepWeekly            int32  `json:"keep_weekly"`
	KeepMonthly           int32  `json:"keep_monthly"`
	KeepAnnual            int32  `json:"keep_annual"`
	RecoveryDrillSchedule string `json:"recovery_drill_schedule"`
}

// DataScheduleRequest reconciles one deployment's registered source bindings
// to one existing off-host S3 destination. DBM owns execution and restore.
type DataScheduleRequest struct {
	Effect   EffectRequest
	Scenario string
	Policy   DataSchedulePolicy
	Bindings []recoverypoint.Binding
	Runner   DBMRunner
}

func dbmRunner(ctx context.Context, argv ...string) ([]byte, error) {
	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return exec.CommandContext(callCtx, "data-backup-manager", argv...).CombinedOutput() //nolint:gosec // typed argv only
}

// DataSchedule creates or updates a deployment-specific DBM plan. It refuses
// local destinations and ambiguous destination names so a healthy plan can
// never quietly be represented as off-host protection.
func (s *Store) DataSchedule(ctx context.Context, req DataScheduleRequest) (map[string]any, EffectResult, error) {
	if err := validIdentifier("scenario", req.Scenario); err != nil {
		return nil, EffectResult{}, err
	}
	if strings.TrimSpace(req.Policy.DestinationName) == "" {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "backup destination name is required")
	}
	interval, err := time.ParseDuration(req.Policy.Schedule)
	if err != nil || interval <= 0 {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "backup schedule must be a positive Go duration")
	}
	drill, err := time.ParseDuration(req.Policy.RecoveryDrillSchedule)
	if err != nil || drill <= 0 {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "recovery drill schedule must be a positive Go duration")
	}
	if req.Policy.KeepHourly < 0 || req.Policy.KeepDaily < 0 || req.Policy.KeepWeekly < 0 || req.Policy.KeepMonthly < 0 || req.Policy.KeepAnnual < 0 {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "backup retention values cannot be negative")
	}
	if req.Policy.KeepHourly+req.Policy.KeepDaily+req.Policy.KeepWeekly+req.Policy.KeepMonthly+req.Policy.KeepAnnual == 0 {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "at least one backup retention tier must be positive")
	}
	if len(req.Bindings) == 0 {
		return nil, EffectResult{}, refuse(CodeInvalidArgument, "scheduled backup requires declared data bindings")
	}
	runner := req.Runner
	if runner == nil {
		runner = dbmRunner
	}
	req.Effect.Verb = "data schedule"
	req.Effect.Input = map[string]any{"scenario": req.Scenario, "policy": req.Policy, "bindings": req.Bindings}
	var report map[string]any
	result, err := s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		id, err := ensureDBMPlan(ctx, runner, req.Effect.DeploymentID, req.Scenario, req.Policy, req.Bindings)
		if err != nil {
			return map[string]any{"plan_name": dbmPlanName(req.Effect.DeploymentID)}, OutcomeFailed, err
		}
		report = map[string]any{"plan_id": id, "plan_name": dbmPlanName(req.Effect.DeploymentID), "destination_name": req.Policy.DestinationName, "target_count": len(req.Bindings), "schedule": req.Policy.Schedule, "retention": req.Policy, "recovery_drill_schedule": req.Policy.RecoveryDrillSchedule, "owner": "data-backup-manager"}
		return report, OutcomeSucceeded, nil
	})
	if report == nil && result.Replayed && result.Receipt.Details != nil {
		report = result.Receipt.Details
	}
	return report, result, err
}

func dbmPlanName(deploymentID string) string { return "scenario-to-cloud-" + deploymentID }

func ensureDBMPlan(ctx context.Context, runner DBMRunner, deploymentID, scenario string, policy DataSchedulePolicy, bindings []recoverypoint.Binding) (string, error) {
	owner := scenario + "-" + deploymentID
	var targetIDs []string
	for _, binding := range bindings {
		kind, err := dbmSourceKind(binding)
		if err != nil {
			return "", err
		}
		out, err := runner(ctx, "targets", "register", "--owner", owner, "--name", binding.ID, "--kind", kind, "--locator", binding.Locator, "--json")
		if err != nil {
			return "", fmt.Errorf("register DBM target %s: %w: %s", binding.ID, err, strings.TrimSpace(string(out)))
		}
		var reply struct {
			ID     string `json:"id"`
			Target struct {
				ID string `json:"id"`
			} `json:"target"`
		}
		if err := json.Unmarshal(out, &reply); err != nil {
			return "", fmt.Errorf("decode DBM target %s: %w", binding.ID, err)
		}
		id := reply.Target.ID
		if id == "" {
			id = reply.ID
		}
		if id == "" {
			return "", fmt.Errorf("DBM target %s registration returned no id", binding.ID)
		}
		targetIDs = append(targetIDs, id)
	}
	sort.Strings(targetIDs)
	var dests struct {
		Destinations []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			BackendKind string `json:"backendKind"`
		} `json:"destinations"`
	}
	out, err := runner(ctx, "destinations", "list", "--json")
	if err != nil {
		return "", fmt.Errorf("list DBM destinations: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := json.Unmarshal(out, &dests); err != nil {
		return "", fmt.Errorf("decode DBM destinations: %w", err)
	}
	destinationID := ""
	for _, dest := range dests.Destinations {
		if dest.Name != policy.DestinationName {
			continue
		}
		if destinationID != "" {
			return "", fmt.Errorf("DBM destination name %q is ambiguous", policy.DestinationName)
		}
		if !strings.Contains(strings.ToUpper(dest.BackendKind), "S3") {
			return "", refuse(CodeInvalidArgument, "DBM destination %q is not S3-backed", policy.DestinationName)
		}
		destinationID = dest.ID
	}
	if destinationID == "" {
		return "", fmt.Errorf("operator-provisioned S3 destination %q is not configured in target-local DBM", policy.DestinationName)
	}
	var plans struct {
		Plans []struct {
			ID                    string   `json:"id"`
			Name                  string   `json:"name"`
			TargetIDs             []string `json:"targetIds"`
			DestinationIDs        []string `json:"destinationIds"`
			Schedule              string   `json:"schedule"`
			Enabled               bool     `json:"enabled"`
			RecoveryDrillSchedule string   `json:"recoveryDrillSchedule"`
			Retention             struct {
				KeepHourly  int32 `json:"keepHourly"`
				KeepDaily   int32 `json:"keepDaily"`
				KeepWeekly  int32 `json:"keepWeekly"`
				KeepMonthly int32 `json:"keepMonthly"`
				KeepAnnual  int32 `json:"keepAnnual"`
			} `json:"retention"`
		} `json:"plans"`
	}
	out, err = runner(ctx, "plans", "list", "--json")
	if err != nil {
		return "", fmt.Errorf("list DBM plans: %w: %s", err, strings.TrimSpace(string(out)))
	}
	if err := json.Unmarshal(out, &plans); err != nil {
		return "", fmt.Errorf("decode DBM plans: %w", err)
	}
	name := dbmPlanName(deploymentID)
	var current *struct {
		ID                    string   `json:"id"`
		Name                  string   `json:"name"`
		TargetIDs             []string `json:"targetIds"`
		DestinationIDs        []string `json:"destinationIds"`
		Schedule              string   `json:"schedule"`
		Enabled               bool     `json:"enabled"`
		RecoveryDrillSchedule string   `json:"recoveryDrillSchedule"`
		Retention             struct {
			KeepHourly  int32 `json:"keepHourly"`
			KeepDaily   int32 `json:"keepDaily"`
			KeepWeekly  int32 `json:"keepWeekly"`
			KeepMonthly int32 `json:"keepMonthly"`
			KeepAnnual  int32 `json:"keepAnnual"`
		} `json:"retention"`
	}
	for i := range plans.Plans {
		if plans.Plans[i].Name == name {
			if current != nil {
				return "", fmt.Errorf("DBM plan name %q is ambiguous", name)
			}
			current = &plans.Plans[i]
		}
	}
	destIDs := []string{destinationID}
	if current != nil && sameStrings(sortedStrings(current.TargetIDs), targetIDs) && sameStrings(sortedStrings(current.DestinationIDs), destIDs) && current.Schedule == policy.Schedule && current.Enabled && current.RecoveryDrillSchedule == policy.RecoveryDrillSchedule && current.Retention.KeepHourly == policy.KeepHourly && current.Retention.KeepDaily == policy.KeepDaily && current.Retention.KeepWeekly == policy.KeepWeekly && current.Retention.KeepMonthly == policy.KeepMonthly && current.Retention.KeepAnnual == policy.KeepAnnual {
		return current.ID, nil
	}
	args := []string{"--targets", strings.Join(targetIDs, ","), "--destinations", destinationID, "--schedule", policy.Schedule, "--keep-hourly", fmt.Sprint(policy.KeepHourly), "--keep-daily", fmt.Sprint(policy.KeepDaily), "--keep-weekly", fmt.Sprint(policy.KeepWeekly), "--keep-monthly", fmt.Sprint(policy.KeepMonthly), "--keep-annual", fmt.Sprint(policy.KeepAnnual), "--recovery-drill-schedule", policy.RecoveryDrillSchedule, "--protection-tier", "critical-primary", "--allow-incomplete-coverage", "--json"}
	if current == nil {
		args = append([]string{"plans", "create", "--name", name}, args...)
		out, err = runner(ctx, args...)
		if err != nil {
			return "", fmt.Errorf("create DBM plan: %w: %s", err, strings.TrimSpace(string(out)))
		}
		var reply struct {
			Plan struct {
				ID string `json:"id"`
			} `json:"plan"`
		}
		if err := json.Unmarshal(out, &reply); err != nil || reply.Plan.ID == "" {
			return "", fmt.Errorf("DBM create plan reply is invalid: %s", strings.TrimSpace(string(out)))
		}
		return verifyDBMPlan(ctx, runner, reply.Plan.ID, name, targetIDs, destIDs, policy)
	}
	args = append([]string{"plans", "update", "--id", current.ID, "--name", name}, args...)
	args = append(args, "--enabled", "true")
	out, err = runner(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("update DBM plan: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return verifyDBMPlan(ctx, runner, current.ID, name, targetIDs, destIDs, policy)
}

// verifyDBMPlan reads the target-local owner state back after a create/update.
// A successful CLI exit alone is not evidence that the requested schedule is
// enabled with the expected sources, destination, retention and drill cadence.
func verifyDBMPlan(ctx context.Context, runner DBMRunner, id, name string, targetIDs, destIDs []string, policy DataSchedulePolicy) (string, error) {
	out, err := runner(ctx, "plans", "list", "--json")
	if err != nil {
		return "", fmt.Errorf("verify DBM plan: %w: %s", err, strings.TrimSpace(string(out)))
	}
	var plans struct {
		Plans []struct {
			ID                    string   `json:"id"`
			Name                  string   `json:"name"`
			TargetIDs             []string `json:"targetIds"`
			DestinationIDs        []string `json:"destinationIds"`
			Schedule              string   `json:"schedule"`
			Enabled               bool     `json:"enabled"`
			RecoveryDrillSchedule string   `json:"recoveryDrillSchedule"`
			Retention             struct {
				KeepHourly  int32 `json:"keepHourly"`
				KeepDaily   int32 `json:"keepDaily"`
				KeepWeekly  int32 `json:"keepWeekly"`
				KeepMonthly int32 `json:"keepMonthly"`
				KeepAnnual  int32 `json:"keepAnnual"`
			} `json:"retention"`
		} `json:"plans"`
	}
	if err := json.Unmarshal(out, &plans); err != nil {
		return "", fmt.Errorf("decode DBM plan verification: %w", err)
	}
	for _, plan := range plans.Plans {
		if plan.Name != name {
			continue
		}
		valid := plan.ID == id && plan.Enabled && plan.Schedule == policy.Schedule && plan.RecoveryDrillSchedule == policy.RecoveryDrillSchedule &&
			sameStrings(sortedStrings(plan.TargetIDs), targetIDs) && sameStrings(sortedStrings(plan.DestinationIDs), destIDs) &&
			plan.Retention.KeepHourly == policy.KeepHourly && plan.Retention.KeepDaily == policy.KeepDaily && plan.Retention.KeepWeekly == policy.KeepWeekly &&
			plan.Retention.KeepMonthly == policy.KeepMonthly && plan.Retention.KeepAnnual == policy.KeepAnnual
		if !valid {
			return "", fmt.Errorf("DBM plan %q did not persist the requested enabled schedule and recovery policy", name)
		}
		return id, nil
	}
	return "", fmt.Errorf("DBM plan %q missing after create/update", name)
}

func dbmSourceKind(binding recoverypoint.Binding) (string, error) {
	switch {
	case binding.Kind == recoverypoint.KindSQL && binding.Provider == recoverypoint.ProviderPostgres:
		return "postgres", nil
	case binding.Kind == recoverypoint.KindSQL && binding.Provider == "sqlite":
		return "sqlite", nil
	case binding.Kind == recoverypoint.KindFiles:
		return "filesystem", nil
	case binding.Kind == recoverypoint.KindApplication:
		return "workspace-checkpoint", nil
	default:
		return "", refuse(CodeInvalidArgument, "DBM has no source kind for binding %s (%s/%s)", binding.ID, binding.Kind, binding.Provider)
	}
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
