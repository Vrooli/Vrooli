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
	// GuardModes stores temporary, audited check-level log-only modes. When nil,
	// the runtime uses the store directory so every hook shares one mode set.
	GuardModes *GuardModeStore
	Now        func() time.Time
	// Removal resolves the host context for filesystem-removal decisions;
	// nil means HostRemovalContext.
	Removal func(workingDirectory string) RemovalContext
	// Creation resolves what the session created, so the guard can let it
	// delete its own files; nil means HostCreationEvidence for the session the
	// event and environment identify. A tool event with no creation source
	// never loosens anything.
	Creation func(event ToolEvent, now time.Time) CreationEvidence
}

func (r Runtime) now() time.Time {
	if r.Now != nil {
		return r.Now().UTC()
	}
	return time.Now().UTC()
}

func (r Runtime) Evaluate(event ToolEvent) (Decision, error) {
	started := time.Now()
	decision, err := r.evaluate(event)
	decision.EvaluationDurationMS = time.Since(started).Milliseconds()
	return decision, err
}

func (r Runtime) evaluate(event ToolEvent) (Decision, error) {
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
	if hardStop := hardStopForEvent(event); hardStop != "" {
		return r.hardStopDecision(event, risk, hardStop, now)
	}
	if hasOpaqueExecution(event) && r.relaxationMayBeActive(now) {
		return r.hardStopDecision(event, risk, HardStopUnknownEffect, now)
	}
	if reasons := operatorOnlyReasons(event, r.operatorEnv(event)); len(reasons) > 0 {
		return operatorOnlyDecision(event, reasons), nil
	}
	if risk == RiskFilesystemRemoval {
		return r.evaluateRemoval(event, now), nil
	}
	if r.Store == nil {
		decision := fallbackDecision(profile, event, risk, "snapshot bundle store is not configured")
		return decision, nil
	}
	bundle, err := r.Store.Load()
	if err != nil {
		decision := fallbackDecision(profile, event, risk, "snapshot bundle unavailable: "+err.Error())
		return decision, nil
	}
	decisions := make([]Decision, 0, len(bundle.Snapshots))
	for _, id := range snapshotIDs(bundle) {
		snapshot := bundle.Snapshots[id]
		if !scopeMatches(snapshot.Scope, event, risk) {
			continue
		}
		providerDecision := evaluateSnapshot(profile, snapshot, event, risk, now)
		r.applyEvaluatorLogOnly(event, now, &providerDecision)
		decisions = append(decisions, providerDecision)
	}
	if len(decisions) == 0 {
		decision := fallbackDecision(profile, event, risk, "no provider snapshot covers this event")
		return decision, nil
	}
	return combineDecisions(decisions), nil
}

// applyEvaluatorLogOnly applies a temporary mode to one provider's decision
// before provider decisions are combined. Decisions without a current,
// provider-owned evaluator ID cannot be suppressed.
func (r Runtime) applyEvaluatorLogOnly(event ToolEvent, now time.Time, decision *Decision) {
	if decision == nil || decision.Action == ActionAllow || !evaluatorIDOwnedBy(decision.EvaluatorID, decision.ProviderID) {
		return
	}
	store := r.guardModeStore()
	if store == nil {
		return
	}
	mode, active, err := store.RecordSuppressedAction(RiskClass(decision.EvaluatorID), event.EventID, decision.Action, now)
	if err != nil {
		decision.Degraded = true
		decision.Evidence = append(decision.Evidence, Evidence{Code: "CHECK_MODES_UNAVAILABLE", Message: "temporary check mode could not be confirmed and normal enforcement remains active", Source: "policy-runtime", Severity: "error"})
		return
	}
	if !active {
		return
	}
	original := decision.Action
	decision.Action = ActionAllow
	decision.Reason = "check is in audited log-only mode until " + mode.ExpiresAt.UTC().Format(time.RFC3339) + "; original action was " + string(original)
	decision.Evidence = append(decision.Evidence, Evidence{Code: "CHECK_LOG_ONLY", Message: "enforcement was observed but not applied under a temporary, audited check mode", Source: "policy-runtime", Facts: map[string]string{"check": decision.EvaluatorID, "mode": mode.Mode, "expires_at": mode.ExpiresAt.UTC().Format(time.RFC3339), "original_action": string(original), "event_id": event.EventID}, Severity: "warning"})
}

// evaluateRemoval decides a deletion from the floor plus the path rules of
// every usable provider snapshot. It does not consult the rollout profile.
func (r Runtime) evaluateRemoval(event ToolEvent, now time.Time) Decision {
	resolve := r.Removal
	if resolve == nil {
		resolve = HostRemovalContext
	}
	ctx := resolve(event.WorkingDirectory)
	targets := removalTargetsFor(event, removalEnv{WorkingDirectory: ctx.WorkingDirectory, Home: ctx.Home, Lookup: ctx.Lookup})
	var restored []RiskClass
	var restoreErr error
	if outsideRepositoryDeletion(ctx, targets) {
		restored, restoreErr = r.restoreGuardModes(HardStopOutsideDelete, event, now)
		// The filesystem floor still owns the deletion verdict and explanation.
		// The portfolio tripwire only withdraws temporary log-only modes.
	}
	rules, approval, degraded := r.pathRules(event, now)
	ctx.Rules, ctx.ApprovalCommand = rules, approval
	if ctx.Created == nil && r.creationEnabled() {
		resolve := r.Creation
		if resolve == nil {
			resolve = func(event ToolEvent, now time.Time) CreationEvidence {
				return HostCreationEvidence(RunIdentityFromEvent(event, ctx.Lookup), now)
			}
		}
		ctx.Created = func() CreationEvidence { return resolve(event, now) }
	}
	decision := ctx.EvaluateRemoval(targets)
	decision.EventID = event.EventID
	r.auditCreated(event, decision, now)
	if len(degraded) > 0 {
		decision.Degraded = true
		decision.Evidence = append(decision.Evidence, degraded...)
	}
	if outsideRepositoryDeletion(ctx, targets) {
		decision.Evidence = append(decision.Evidence, tripwireEvidence(HardStopOutsideDelete, restored, restoreErr))
		if restoreErr != nil {
			decision.Degraded = true
			decision.Action = ActionDeny
			decision.Reason = "outside-repository deletion was denied because temporary check restoration could not be confirmed"
		}
	}
	return decision
}

// pathRules collects rules from fresh, ready, healthy, clean snapshots. The
// returned evidence explains any missing or unusable declarations; the floor
// then decides alone, which only ever makes decisions stricter. The approval
// command is the first usable snapshot's removal_approval, if any.
func (r Runtime) pathRules(event ToolEvent, now time.Time) ([]PathRule, string, []Evidence) {
	unavailable := func(message string) []Evidence {
		return []Evidence{{Code: "PATH_RULES_UNAVAILABLE", Message: message, Source: "policy-runtime", Severity: "warning"}}
	}
	if r.Store == nil {
		return nil, "", unavailable("no snapshot store is configured; only the built-in floor applies")
	}
	bundle, err := r.Store.Load()
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", unavailable("no provider has published path rules; only the built-in floor applies")
	}
	if err != nil {
		return nil, "", unavailable("snapshot bundle unavailable (" + err.Error() + "); only the built-in floor applies")
	}
	var rules []PathRule
	var evidence []Evidence
	approval := ""
	declared := false
	for _, id := range snapshotIDs(bundle) {
		snapshot := bundle.Snapshots[id]
		if len(snapshot.PathRules)+len(snapshot.NameRules) == 0 || !scopeMatches(snapshot.Scope, event, RiskFilesystemRemoval) {
			continue
		}
		declared = true
		if problem := unusableSnapshot(snapshot, now); problem != "" {
			evidence = append(evidence, Evidence{Code: "PATH_RULES_UNUSABLE", Message: problem + "; its path rules were ignored", Source: id, Severity: "warning"})
			continue
		}
		if approval == "" {
			approval = strings.TrimSpace(snapshot.RemovalApproval)
		}
		for _, rule := range append(append([]PathRule(nil), snapshot.PathRules...), snapshot.NameRules...) {
			rule.Provider = id
			if strings.TrimSpace(rule.Source) == "" {
				rule.Source = id
			}
			rules = append(rules, rule)
		}
	}
	if !declared {
		evidence = append(evidence, unavailable("no provider has published path rules; only the built-in floor applies")...)
	}
	return rules, approval, evidence
}

func unusableSnapshot(snapshot ProviderSnapshot, now time.Time) string {
	switch {
	case snapshot.ExpiresAt.Before(now) || snapshot.Health.ExpiresAt.Before(now):
		return "provider snapshot expired at " + snapshot.ExpiresAt.UTC().Format(time.RFC3339)
	case snapshot.CapturedAt.After(now.Add(MaxClockSkew)):
		return "provider snapshot is from the future beyond the allowed clock skew"
	case strings.ToLower(strings.TrimSpace(snapshot.Readiness.State)) != "ready":
		return "provider has not declared readiness"
	case snapshot.Health.State != HealthHealthy:
		return "provider health is " + string(snapshot.Health.State)
	case snapshot.Evidence != EvidenceClean:
		return "provider evidence is " + string(snapshot.Evidence)
	}
	return ""
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
		if evaluatorIDOwnedBy(rule.EvaluatorID, snapshot.ProviderID) {
			decision.EvaluatorID = rule.EvaluatorID
		}
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

func scopeMatches(scope ProviderScope, event ToolEvent, risk RiskClass) bool {
	if len(scope.Runners) > 0 && !containsFold(scope.Runners, event.Runner) {
		return false
	}
	if len(scope.Risks) > 0 {
		covered := false
		for _, candidate := range scope.Risks {
			covered = covered || candidate == risk
		}
		if !covered {
			return false
		}
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

// creationEnabled is false only when an operator switched the loosening off.
func (r Runtime) creationEnabled() bool {
	if r.Store == nil {
		return true
	}
	return LoadCreationSettings(r.Store.Dir).Enabled
}

// auditCreated records each path the guard allowed because the session created it.
func (r Runtime) auditCreated(event ToolEvent, decision Decision, now time.Time) {
	if r.Store == nil {
		return
	}
	for _, evidence := range decision.Evidence {
		if evidence.Facts["created_by"] == "" || evidence.Facts["path"] == "" {
			continue
		}
		AppendCreationAudit(r.Store.Dir, CreationAudit{
			At: now, Path: evidence.Facts["path"], Recursive: evidence.Facts["recursive"] == "true",
			Source: evidence.Facts["created_by"], Certainty: evidence.Facts["certainty"], Detail: evidence.Message,
			Session: RunIdentityFromEvent(event, nil).HarnessSession,
		})
	}
}
