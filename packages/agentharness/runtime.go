package agentharness

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Runtime struct {
	Profile RolloutProfile
	Store   *BundleStore
	Now     func() time.Time
}

func (r Runtime) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r Runtime) Evaluate(event ToolEvent) (Decision, error) {
	now := r.now()
	if err := event.Normalize(now); err != nil {
		return Decision{}, err
	}
	profile := r.Profile
	if profile == "" {
		profile = ProfileAdvisory
	}
	if !validProfile(profile) {
		return Decision{}, fmt.Errorf("unsupported rollout profile %q", profile)
	}
	risk := ClassifyToolEvent(event)
	if r.Store == nil {
		return fallbackDecision(profile, event, risk, "snapshot bundle store is not configured"), nil
	}
	bundle, err := r.Store.Load()
	if err != nil {
		return fallbackDecision(profile, event, risk, "snapshot bundle unavailable: "+err.Error()), nil
	}
	decisions := make([]Decision, 0, len(bundle.Snapshots))
	for _, id := range snapshotIDs(bundle) {
		snapshot := bundle.Snapshots[id]
		if !scopeMatches(snapshot.Scope, event) {
			continue
		}
		decisions = append(decisions, evaluateSnapshot(profile, snapshot, event, risk, now))
	}
	if len(decisions) == 0 {
		return fallbackDecision(profile, event, risk, "no provider snapshot covers this event"), nil
	}
	return combineDecisions(decisions), nil
}

func evaluateSnapshot(profile RolloutProfile, snapshot ProviderSnapshot, event ToolEvent, risk RiskClass, now time.Time) Decision {
	decision := Decision{
		ContractVersion: ContractVersion,
		EventID:         event.EventID,
		Action:          ActionAllow,
		Risk:            risk,
		ProviderID:      snapshot.ProviderID,
		ProviderVersion: snapshot.Version,
		UsedSnapshot:    true,
		Maturity:        bestMaturity(snapshot.Capabilities),
		Health:          snapshot.Health.State,
		Evidence:        append([]Evidence(nil), snapshotEvidence(snapshot)...),
	}
	if snapshot.ExpiresAt.Before(now) || snapshot.Health.ExpiresAt.Before(now) {
		decision.Degraded = true
		decision.Health = HealthStale
		decision.Evidence = append(decision.Evidence, Evidence{Code: "PROVIDER_EVIDENCE_STALE", Message: "provider snapshot or health evidence has expired", Source: snapshot.ProviderID, Severity: "warning"})
	}
	if snapshot.CapturedAt.After(now.Add(MaxClockSkew)) {
		decision.Degraded = true
		decision.Health = HealthStale
		decision.Evidence = append(decision.Evidence, Evidence{Code: "PROVIDER_CLOCK_SKEW", Message: "provider snapshot is from the future beyond the allowed clock skew", Source: snapshot.ProviderID, Severity: "error"})
	}
	if strings.ToLower(strings.TrimSpace(snapshot.Readiness.State)) != "ready" {
		decision.Degraded = true
		decision.Health = HealthDegraded
		decision.Evidence = append(decision.Evidence, Evidence{Code: "PROVIDER_NOT_READY", Message: "provider has not declared readiness for policy evaluation", Source: snapshot.ProviderID, Severity: "warning"})
	}
	rule := matchingRule(snapshot.Rules, risk)
	if rule != nil {
		decision.Action = rule.Action
		decision.Reason = rule.Reason
		decision.Replacement = append([]string(nil), rule.Replacement...)
		decision.Repair = rule.Repair
		decision.Evidence = append(decision.Evidence, rule.Evidence...)
	}
	if decision.Reason == "" {
		decision.Reason = "provider snapshot evaluated the normalized event"
	}
	if decision.Action == ActionAllow && (snapshot.Evidence != EvidenceClean || decision.Health != HealthHealthy) {
		decision.Degraded = true
		decision.Action = degradedAction(profile, risk)
		decision.Reason = "provider evidence is not clean or healthy"
	}
	if highRisk(risk) {
		if decision.Health != HealthHealthy || snapshot.Evidence != EvidenceClean {
			decision.Degraded = true
			decision.Action = degradedAction(profile, risk)
			decision.Reason = "high-risk action requires healthy, fresh, clean provider evidence"
		}
		if profile == ProfileEnforcing && maturityRank(decision.Maturity) < maturityRank(MaturityEnforcing) {
			decision.Action = ActionDeny
			decision.Degraded = true
			decision.Reason = "enforcing profile requires an enforcing provider maturity"
			decision.Evidence = append(decision.Evidence, Evidence{Code: "MATURITY_BELOW_PROFILE", Message: "declared maturity is below the selected enforcing profile", Source: snapshot.ProviderID, Severity: "error"})
		}
		if profile == ProfileGuarded && maturityRank(decision.Maturity) < maturityRank(MaturityGuarded) && decision.Action == ActionAllow {
			decision.Action = ActionAsk
			decision.Reason = "guarded profile requires at least guarded provider maturity"
			decision.Evidence = append(decision.Evidence, Evidence{Code: "MATURITY_REQUIRES_CONFIRMATION", Message: "provider maturity requires explicit confirmation for this mutation", Source: snapshot.ProviderID, Severity: "warning"})
		}
		if profile == ProfileGuided && decision.Action == ActionAllow && maturityRank(decision.Maturity) < maturityRank(MaturityGuided) {
			decision.Action = ActionAsk
			decision.Reason = "guided profile requires confirmation below guided maturity"
		}
	}
	if decision.Action != ActionAllow && len(decision.Evidence) == 0 {
		decision.Evidence = append(decision.Evidence, Evidence{Code: "DECISION_REQUIRES_ATTENTION", Message: decision.Reason, Source: snapshot.ProviderID, Severity: "warning"})
	}
	return decision
}

func combineDecisions(decisions []Decision) Decision {
	sortDecisions(decisions)
	for _, action := range []DecisionAction{ActionDeny, ActionAsk, ActionRepair, ActionRewrite, ActionRoute, ActionUnavailable} {
		for _, decision := range decisions {
			if decision.Action == action {
				return decision
			}
		}
	}
	return decisions[0]
}

func sortDecisions(decisions []Decision) {
	for i := 1; i < len(decisions); i++ {
		for j := i; j > 0 && decisions[j].ProviderID < decisions[j-1].ProviderID; j-- {
			decisions[j], decisions[j-1] = decisions[j-1], decisions[j]
		}
	}
}

func fallbackDecision(profile RolloutProfile, event ToolEvent, risk RiskClass, reason string) Decision {
	decision := Decision{ContractVersion: ContractVersion, EventID: event.EventID, Action: ActionAllow, Risk: risk, Degraded: true, Reason: reason, Evidence: []Evidence{{Code: "PROVIDER_UNAVAILABLE", Message: reason, Source: "policy-runtime", Severity: "warning"}}}
	if highRisk(risk) {
		switch profile {
		case ProfileGuided:
			decision.Action = ActionAsk
		case ProfileGuarded, ProfileEnforcing:
			decision.Action = ActionDeny
			decision.Evidence[0].Severity = "error"
		}
	}
	return decision
}

func degradedAction(profile RolloutProfile, risk RiskClass) DecisionAction {
	if !highRisk(risk) {
		return ActionAllow
	}
	if profile == ProfileGuarded || profile == ProfileEnforcing {
		return ActionDeny
	}
	if profile == ProfileGuided {
		return ActionAsk
	}
	return ActionAllow
}

func matchingRule(rules []PolicyRule, risk RiskClass) *PolicyRule {
	for i := range rules {
		if rules[i].Risk == risk {
			return &rules[i]
		}
	}
	for i := range rules {
		if rules[i].Risk == RiskUnknown {
			return &rules[i]
		}
	}
	return nil
}

func snapshotEvidence(snapshot ProviderSnapshot) []Evidence {
	if snapshot.Evidence == EvidenceClean && snapshot.Health.State == HealthHealthy {
		return nil
	}
	message := snapshot.Health.Message
	if message == "" {
		message = "provider snapshot reports degraded evidence"
	}
	return []Evidence{{Code: "PROVIDER_EVIDENCE_STATE", Message: message, Source: snapshot.ProviderID, Severity: "warning"}}
}

func bestMaturity(capabilities []ProviderCapability) Maturity {
	best := MaturityExperimental
	for _, capability := range capabilities {
		if maturityRank(capability.DeclaredMaturity) > maturityRank(best) {
			best = capability.DeclaredMaturity
		}
	}
	return best
}

func scopeMatches(scope ProviderScope, event ToolEvent) bool {
	if len(scope.Runners) > 0 && !containsFold(scope.Runners, event.Runner) {
		return false
	}
	if len(scope.Ecosystems) > 0 {
		ecosystem := ""
		if event.Context != nil {
			ecosystem = event.Context["ecosystem"]
		}
		if ecosystem == "" || !containsFold(scope.Ecosystems, ecosystem) {
			return false
		}
	}
	if len(scope.Roots) > 0 {
		matched := false
		for _, root := range scope.Roots {
			cleanRoot := filepath.Clean(root)
			cleanDir := filepath.Clean(event.WorkingDirectory)
			if cleanDir == cleanRoot || strings.HasPrefix(cleanDir, cleanRoot+string(os.PathSeparator)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func containsFold(values []string, value string) bool {
	for _, candidate := range values {
		if strings.EqualFold(strings.TrimSpace(candidate), strings.TrimSpace(value)) {
			return true
		}
	}
	return false
}

func validProfile(profile RolloutProfile) bool {
	return profile == ProfileAdvisory || profile == ProfileGuided || profile == ProfileGuarded || profile == ProfileEnforcing
}

func ExitCode(action DecisionAction) int {
	switch action {
	case ActionAllow, ActionRoute, ActionRewrite, ActionRepair:
		return 0
	case ActionAsk:
		return 10
	case ActionDeny:
		return 20
	case ActionUnavailable:
		return 30
	default:
		return 30
	}
}

func IsUnavailable(err error) bool {
	return errors.Is(err, os.ErrNotExist) || strings.Contains(strings.ToLower(err.Error()), "unavailable")
}
