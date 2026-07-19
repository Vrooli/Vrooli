package runtimesupervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// reconcileRecovery drives the durable recovery state machine. With no
// pressure provider (or unknown pressure evidence) it deliberately does
// nothing: missing telemetry is never interpreted as a pressure-clear signal.
func (s *Service) reconcileRecovery(ctx context.Context) (RecoveryReport, error) {
	if s.cfg.PressureProvider == nil {
		return RecoveryReport{}, nil
	}
	pressure := s.cfg.PressureProvider.Snapshot(ctx)
	if !pressure.Known {
		return RecoveryReport{}, nil
	}
	now := s.now()
	if pressure.ObservedAt.IsZero() {
		pressure.ObservedAt = now
	}
	latest, err := s.store.ListPressureEpochs(ctx, 1)
	if err != nil {
		return RecoveryReport{}, fmt.Errorf("list recovery pressure epochs: %w", err)
	}
	var epoch *scenarioruntime.PressureEpoch
	if len(latest) > 0 && isOpenPressureEpoch(latest[0].Status) {
		epoch = &latest[0]
	}
	if pressure.UnderPressure {
		if epoch == nil {
			created, err := s.store.CreatePressureEpoch(ctx, scenarioruntime.PressureEpoch{
				Status: scenarioruntime.PressureEpochDetected, Source: pressure.Source, DetectedAt: pressure.ObservedAt,
				DetailsJSON: pressure.Reason,
			})
			if err != nil {
				return RecoveryReport{}, fmt.Errorf("create pressure epoch: %w", err)
			}
			epoch = &created
		} else if epoch.Status != scenarioruntime.PressureEpochRegressed {
			epoch.Status = scenarioruntime.PressureEpochRegressed
			epoch.Source = pressure.Source
			epoch.ClearedAt = nil
			epoch.DetailsJSON = pressure.Reason
			updated, err := s.store.UpdatePressureEpoch(ctx, *epoch)
			if err != nil {
				return RecoveryReport{}, fmt.Errorf("mark pressure epoch regressed: %w", err)
			}
			epoch = &updated
		}
		return RecoveryReport{EpochID: epoch.EpochID, Gated: 1}, nil
	}
	if epoch == nil {
		return RecoveryReport{}, nil
	}
	if epoch.Status != scenarioruntime.PressureEpochGated && epoch.Status != scenarioruntime.PressureEpochCleared {
		cleared := pressure.ObservedAt
		// Gated is the explicit quiet-period state. External remediation must
		// defer restart actions while the runtime controller owns this epoch.
		epoch.Status = scenarioruntime.PressureEpochGated
		epoch.Source = pressure.Source
		epoch.ClearedAt = &cleared
		epoch.DetailsJSON = pressure.Reason
		updated, err := s.store.UpdatePressureEpoch(ctx, *epoch)
		if err != nil {
			return RecoveryReport{}, fmt.Errorf("mark pressure epoch cleared: %w", err)
		}
		epoch = &updated
	}
	if epoch.ClearedAt == nil || now.Before(epoch.ClearedAt.Add(s.recoveryQuietPeriod())) {
		return RecoveryReport{EpochID: epoch.EpochID, Gated: 1}, nil
	}
	report, err := s.dispatchRecovery(ctx, *epoch)
	if err != nil {
		return report, err
	}
	// Release the cross-service ownership gate once this bounded scheduling pass
	// has completed. Durable decisions prevent repeat dispatch in this epoch.
	epoch.Status = scenarioruntime.PressureEpochCleared
	updated, err := s.store.UpdatePressureEpoch(ctx, *epoch)
	if err != nil {
		return report, fmt.Errorf("complete pressure recovery gate: %w", err)
	}
	report.EpochID = updated.EpochID
	return report, nil
}

func (s *Service) dispatchRecovery(ctx context.Context, epoch scenarioruntime.PressureEpoch) (RecoveryReport, error) {
	report := RecoveryReport{EpochID: epoch.EpochID}
	policies, err := s.store.ListRecoveryPolicies(ctx, scenarioruntime.RecoveryPolicyFilter{})
	if err != nil {
		return report, fmt.Errorf("list recovery policies: %w", err)
	}
	instances, err := s.store.ListInstances(ctx, scenarioruntime.InstanceFilter{})
	if err != nil {
		return report, fmt.Errorf("list instances for recovery: %w", err)
	}
	latest := latestInstances(instances)
	decisions, err := s.store.ListRecoveryDecisions(ctx, scenarioruntime.RecoveryDecisionFilter{EpochID: epoch.EpochID, Limit: 1000})
	if err != nil {
		return report, fmt.Errorf("list recovery decisions: %w", err)
	}
	attempts := recoveryAttempts(decisions)
	cooldowns := recoveryCooldowns(decisions)
	terminal := recoveryTerminal(decisions)
	eligible := make([]scenarioruntime.RecoveryPolicy, 0, len(policies))
	for _, policy := range policies {
		if !policy.Enabled || !policy.Critical || policy.OptOut || policy.RetryBudget <= 0 {
			continue
		}
		key := policy.Scenario + "@" + policy.Variant
		// A queued or restored decision is a durable at-most-once handoff for
		// this epoch. The registry must provide new failure evidence in a later
		// epoch before the controller can attempt this workload again.
		if terminal[key] {
			continue
		}
		instance, ok := latest[key]
		if !ok || (instance.Status != scenarioruntime.StatusExpired && instance.Status != scenarioruntime.StatusFailed) {
			continue
		}
		if attempts[key] >= policy.RetryBudget {
			if err := s.recordDecision(ctx, epoch.EpochID, policy, scenarioruntime.RecoveryDecisionSkipped, "retry budget exhausted", attempts[key], nil); err != nil {
				return report, err
			}
			report.Skipped++
			continue
		}
		if until, ok := cooldowns[key]; ok && until.After(s.now()) {
			if err := s.recordDecision(ctx, epoch.EpochID, policy, scenarioruntime.RecoveryDecisionSkipped, "recovery cooldown active", attempts[key], &until); err != nil {
				return report, err
			}
			report.Skipped++
			continue
		}
		eligible = append(eligible, policy)
	}
	sort.SliceStable(eligible, func(i, j int) bool {
		if eligible[i].DependencyTier != eligible[j].DependencyTier {
			return eligible[i].DependencyTier < eligible[j].DependencyTier
		}
		if eligible[i].Scenario != eligible[j].Scenario {
			return eligible[i].Scenario < eligible[j].Scenario
		}
		return eligible[i].Variant < eligible[j].Variant
	})
	if len(eligible) == 0 {
		return report, nil
	}
	// Do not begin a later dependency tier in the same scheduling pass. A
	// prior tier must appear as a new healthy runtime on a later tick first.
	tier := eligible[0].DependencyTier
	limit := s.recoveryConcurrency()
	dispatched := 0
	for i, policy := range eligible {
		if policy.DependencyTier != tier || i >= limit || dispatched >= limit {
			break
		}
		attempt := attempts[policy.Scenario+"@"+policy.Variant] + 1
		if err := s.recordDecision(ctx, epoch.EpochID, policy, scenarioruntime.RecoveryDecisionQueued, "pressure clear gate satisfied", attempt, nil); err != nil {
			return report, err
		}
		report.Queued++
		if err := s.launchRecovery(ctx, RecoveryLaunchRequest{Scenario: policy.Scenario, Variant: policy.Variant, EpochID: epoch.EpochID, Attempt: attempt}); err != nil {
			cooldown := s.now().Add(s.recoveryCooldown())
			if recordErr := s.recordDecision(ctx, epoch.EpochID, policy, scenarioruntime.RecoveryDecisionFailed, err.Error(), attempt, &cooldown); recordErr != nil {
				return report, recordErr
			}
			report.Failed++
			// A failed prerequisite tier prevents later tiers from amplifying a
			// pressure incident. The next controller tick may retry after policy
			// cooldown handling is added with lifecycle result evidence.
			break
		}
		if err := s.recordDecision(ctx, epoch.EpochID, policy, scenarioruntime.RecoveryDecisionRestored, "lifecycle restart accepted", attempt, nil); err != nil {
			return report, err
		}
		report.Restored++
		dispatched++
	}
	return report, nil
}

func (s *Service) launchRecovery(ctx context.Context, request RecoveryLaunchRequest) error {
	if s.cfg.RecoveryLaunch != nil {
		return s.cfg.RecoveryLaunch(ctx, request)
	}
	return s.defaultRecoveryLaunch(ctx, request)
}

func (s *Service) defaultRecoveryLaunch(ctx context.Context, request RecoveryLaunchRequest) error {
	executable := strings.TrimSpace(s.cfg.Executable)
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return fmt.Errorf("resolve lifecycle executable: %w", err)
		}
	}
	root := strings.TrimSpace(os.Getenv("VROOLI_SOURCE_ROOT"))
	if root == "" {
		root = strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	}
	if root == "" {
		return fmt.Errorf("recovery lifecycle source root is not configured")
	}
	args := []string{"--no-stale-check", "scenario", "restart", request.Scenario, "--instance", request.Variant}
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Dir = root
	cmd.Env = supervisorCommandEnv(os.Environ(), s.cfg.HomeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("lifecycle restart %s@%s: %w: %s", request.Scenario, request.Variant, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *Service) recordDecision(ctx context.Context, epochID string, policy scenarioruntime.RecoveryPolicy, state, reason string, attempt int, cooldown *time.Time) error {
	_, err := s.store.RecordRecoveryDecision(ctx, scenarioruntime.RecoveryDecision{
		EpochID: epochID, Scenario: policy.Scenario, Variant: policy.Variant, State: state,
		Reason: reason, Attempt: attempt, CooldownUntil: cooldown,
		IdempotencyKey: fmt.Sprintf("%s/%s@%s/%s/%d", epochID, policy.Scenario, policy.Variant, state, attempt),
	})
	return err
}

func latestInstances(instances []scenarioruntime.Instance) map[string]scenarioruntime.Instance {
	latest := make(map[string]scenarioruntime.Instance, len(instances))
	for _, instance := range instances {
		key := instance.Scenario + "@" + instance.Variant
		if previous, ok := latest[key]; !ok || instance.Generation > previous.Generation {
			latest[key] = instance
		}
	}
	return latest
}

func recoveryAttempts(decisions []scenarioruntime.RecoveryDecision) map[string]int {
	attempts := map[string]int{}
	for _, decision := range decisions {
		if decision.State != scenarioruntime.RecoveryDecisionQueued && decision.State != scenarioruntime.RecoveryDecisionRestored && decision.State != scenarioruntime.RecoveryDecisionFailed {
			continue
		}
		key := decision.Scenario + "@" + decision.Variant
		if decision.Attempt > attempts[key] {
			attempts[key] = decision.Attempt
		}
	}
	return attempts
}

func recoveryCooldowns(decisions []scenarioruntime.RecoveryDecision) map[string]time.Time {
	cooldowns := map[string]time.Time{}
	for _, decision := range decisions {
		if decision.CooldownUntil == nil {
			continue
		}
		key := decision.Scenario + "@" + decision.Variant
		if current, ok := cooldowns[key]; !ok || decision.CooldownUntil.After(current) {
			cooldowns[key] = *decision.CooldownUntil
		}
	}
	return cooldowns
}

func recoveryTerminal(decisions []scenarioruntime.RecoveryDecision) map[string]bool {
	terminal := map[string]bool{}
	for _, decision := range decisions {
		if decision.State != scenarioruntime.RecoveryDecisionQueued && decision.State != scenarioruntime.RecoveryDecisionRestored {
			continue
		}
		terminal[decision.Scenario+"@"+decision.Variant] = true
	}
	return terminal
}

func isOpenPressureEpoch(status string) bool {
	return status == scenarioruntime.PressureEpochDetected || status == scenarioruntime.PressureEpochGated || status == scenarioruntime.PressureEpochRegressed || status == scenarioruntime.PressureEpochCleared
}

func (s *Service) recoveryQuietPeriod() time.Duration {
	if s.cfg.RecoveryQuietPeriod <= 0 {
		return DefaultRecoveryQuietPeriod
	}
	return s.cfg.RecoveryQuietPeriod
}

func (s *Service) recoveryCooldown() time.Duration {
	if s.cfg.RecoveryCooldown <= 0 {
		return DefaultRecoveryCooldown
	}
	return s.cfg.RecoveryCooldown
}

func (s *Service) recoveryConcurrency() int {
	if s.cfg.RecoveryConcurrency <= 0 {
		return DefaultRecoveryConcurrency
	}
	return s.cfg.RecoveryConcurrency
}
