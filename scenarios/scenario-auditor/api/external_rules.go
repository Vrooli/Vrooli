package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	re "scenario-auditor/internal/ruleengine"
	rulespkg "scenario-auditor/rules"
)

type externalRuleDefinition struct {
	rule     RuleInfo
	source   string
	provider externalRuleProvider
}

type externalRuleProvider interface {
	ID() string
	Name() string
	Rules() []rulespkg.Rule
	Run(ctx context.Context, scenarioName string, ruleIDs []string) ([]StandardsViolation, error)
}

var (
	externalProvidersMu sync.RWMutex
	externalProviders   = make(map[string]externalRuleProvider)
	externalRulesIndex  = make(map[string]externalRuleDefinition)
)

func registerExternalProvider(provider externalRuleProvider) {
	if provider == nil {
		return
	}

	externalProvidersMu.Lock()
	defer externalProvidersMu.Unlock()

	externalProviders[provider.ID()] = provider
	for _, rule := range provider.Rules() {
		info := RuleInfo{
			Rule: rulespkg.Rule{
				ID:          rule.ID,
				Name:        rule.Name,
				Description: rule.Description,
				Category:    rule.Category,
				Severity:    rule.Severity,
				Enabled:     true,
				Standard:    rule.Standard,
			},
			Targets: []string{"external"},
			Implementation: re.ImplementationStatus{
				Valid:   false,
				Error:   fmt.Sprintf("Managed by %s", provider.Name()),
				Details: provider.Name(),
			},
		}
		externalRulesIndex[rule.ID] = externalRuleDefinition{
			rule:     info,
			source:   provider.ID(),
			provider: provider,
		}
	}
}

func loadExternalRuleInfos() map[string]RuleInfo {
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()

	infos := make(map[string]RuleInfo, len(externalRulesIndex))
	for id, def := range externalRulesIndex {
		infos[id] = def.rule
	}
	return infos
}

func isExternalRule(ruleID string) bool {
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()
	_, ok := externalRulesIndex[ruleID]
	return ok
}

func externalRuleProviderFor(ruleID string) (externalRuleProvider, bool) {
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()
	entry, ok := externalRulesIndex[ruleID]
	if !ok {
		return nil, false
	}
	return entry.provider, true
}

func mergeWithExternalRules(ruleInfos map[string]RuleInfo) map[string]RuleInfo {
	merged := make(map[string]RuleInfo, len(ruleInfos)+len(externalRulesIndex))
	for id, info := range ruleInfos {
		merged[id] = info
	}
	externalProvidersMu.RLock()
	for id, def := range externalRulesIndex {
		merged[id] = def.rule
	}
	externalProvidersMu.RUnlock()
	return merged
}

func runExternalRuleChecks(ctx context.Context, scenarioName string, requested map[string]struct{}, includeDisabled bool) ([]StandardsViolation, error) {
	if strings.TrimSpace(scenarioName) == "" {
		return nil, nil
	}

	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()

	states := ruleStateStore.GetAllStates()
	byProvider := make(map[string][]string)
	for ruleID, entry := range externalRulesIndex {
		if !shouldEvaluateExternalRule(ruleID, requested, states, includeDisabled) {
			continue
		}
		byProvider[entry.source] = append(byProvider[entry.source], ruleID)
	}

	var violations []StandardsViolation
	for providerID, ruleIDs := range byProvider {
		provider := externalProviders[providerID]
		if provider == nil {
			continue
		}
		providerViolations, err := provider.Run(ctx, scenarioName, ruleIDs)
		if err != nil {
			logger.Warn("External provider failed", map[string]any{
				"provider": provider.Name(),
				"scenario": scenarioName,
				"error":    err.Error(),
			})
			continue
		}
		for _, violation := range providerViolations {
			violation.Source = providerID
			violation.ScenarioName = scenarioName
			if violation.ID == "" {
				violation.ID = uuid.New().String()
			}
			violations = append(violations, violation)
		}
	}

	return violations, nil
}

func shouldEvaluateExternalRule(ruleID string, requested map[string]struct{}, states map[string]bool, includeDisabled bool) bool {
	if len(requested) > 0 {
		if _, ok := requested[ruleID]; !ok {
			return false
		}
	}

	enabled := true
	if state, ok := states[ruleID]; ok {
		enabled = state
	}

	if !enabled && !includeDisabled {
		return false
	}

	return true
}
