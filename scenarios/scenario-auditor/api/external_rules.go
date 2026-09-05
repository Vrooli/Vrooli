package main

import (
	"context"
	"fmt"
	"path/filepath"
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
	Run(ctx context.Context, target standardsScanTarget, ruleIDs []string) ([]StandardsViolation, error)
}

// externalRuleFixer is an optional interface for providers that support deterministic fixes.
type externalRuleFixer interface {
	Fix(ctx context.Context, scenarioNames []string, ruleIDs []string, dryRun bool) ([]ExternalFixResult, error)
}

type ExternalFixResult struct {
	ScenarioName string              `json:"scenario_name"`
	RuleID       string              `json:"rule_id"`
	Fixed        bool                `json:"fixed"`
	FilePath     string              `json:"file_path"`
	Changes      []ExternalFixChange `json:"changes"`
	Error        string              `json:"error,omitempty"`
}

type ExternalFixChange struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

var (
	externalProvidersMu   sync.RWMutex
	externalProviders     = make(map[string]externalRuleProvider)
	externalRulesIndex    = make(map[string]externalRuleDefinition)
	externalProvidersOnce sync.Once
)

func registerDefaultExternalProviders() {
	externalProvidersOnce.Do(func() {
		for _, provider := range []externalRuleProvider{
			// NOTE: the app-monitor interop provider was removed — the static
			// UI-interop rules moved to ui-health, the single UI-validation
			// authority. scenario-auditor no longer registers interop_* rules.
			// The test-genie structure provider was likewise removed — the
			// structure_* rules are owned by structure-health via test-genie's
			// structure phase, and the delegation endpoint no longer exists.
			// The PRD control-tower provider (prd_* rules) was removed —
			// the business contract is owned by business-health via
			// test-genie's business phase.
			newStackGovernorProvider(),
		} {
			registerExternalProvider(provider)
		}
	})
}

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
	registerDefaultExternalProviders()
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()

	infos := make(map[string]RuleInfo, len(externalRulesIndex))
	for id, def := range externalRulesIndex {
		infos[id] = def.rule
	}
	return infos
}

func isExternalRule(ruleID string) bool {
	registerDefaultExternalProviders()
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()
	_, ok := externalRulesIndex[ruleID]
	return ok
}

func externalRuleProviderFor(ruleID string) (externalRuleProvider, bool) {
	registerDefaultExternalProviders()
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()
	entry, ok := externalRulesIndex[ruleID]
	if !ok {
		return nil, false
	}
	return entry.provider, true
}

func externalFixerForRule(ruleID string) (externalRuleFixer, bool) {
	registerDefaultExternalProviders()
	externalProvidersMu.RLock()
	defer externalProvidersMu.RUnlock()
	info, ok := externalRulesIndex[ruleID]
	if !ok {
		return nil, false
	}
	fixer, ok := info.provider.(externalRuleFixer)
	return fixer, ok
}

func mergeWithExternalRules(ruleInfos map[string]RuleInfo) map[string]RuleInfo {
	registerDefaultExternalProviders()
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

func runExternalRuleChecks(ctx context.Context, target standardsScanTarget, requested map[string]struct{}, includeDisabled bool) ([]StandardsViolation, error) {
	registerDefaultExternalProviders()
	scenarioName := strings.TrimSpace(target.Name)
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
		providerViolations, err := provider.Run(ctx, target, ruleIDs)
		if err != nil {
			logger.Warn("External provider failed", map[string]any{
				"provider": provider.Name(),
				"scenario": scenarioName,
				"error":    err.Error(),
			})
			continue
		}
		for _, violation := range providerViolations {
			if shouldDropExternalViolationForTarget(target, violation) {
				continue
			}
			violation.FilePath = stableExternalViolationPath(target, violation.FilePath)
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

func shouldDropExternalViolationForTarget(target standardsScanTarget, violation StandardsViolation) bool {
	if strings.TrimSpace(target.Path) == "" || strings.TrimSpace(violation.FilePath) == "" {
		return false
	}
	filePath := filepath.Clean(violation.FilePath)
	if !filepath.IsAbs(filePath) {
		return false
	}
	return !pathWithinDir(filePath, filepath.Clean(target.Path))
}

func stableExternalViolationPath(target standardsScanTarget, filePath string) string {
	cleaned := strings.TrimSpace(filePath)
	if cleaned == "" || strings.TrimSpace(target.Path) == "" {
		return cleaned
	}
	abs := filepath.Clean(cleaned)
	if !filepath.IsAbs(abs) {
		return filepath.ToSlash(cleaned)
	}
	if rel, err := filepath.Rel(filepath.Clean(target.Path), abs); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		return filepath.ToSlash(rel)
	}
	return cleaned
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
