package main

import (
	"fmt"

	re "scenario-auditor/internal/ruleengine"
)

// Compatibility aliases for legacy code paths.
type RuleInfo = re.Info
type RuleImplementationStatus = re.ImplementationStatus
type RuleExecutionInfo = re.ExecutionInfo
type RuleArgumentInfo = re.ArgumentInfo
type RuleExecutionCall = re.ExecutionCall

func LoadRulesFromFiles() (map[string]RuleInfo, error) {
	svc, err := ruleService()
	if err != nil {
		return nil, err
	}
	rules, err := svc.Load()
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func ConvertRuleInfoToRule(info RuleInfo) Rule {
	return convertInfoToRule(info)
}

func buildRuleExecutionInfo(info RuleInfo) re.ExecutionInfo {
	return re.BuildExecutionInfo(info)
}

func runRuleTests(ruleID string, info RuleInfo) ([]re.TestResult, error) {
	svc, err := ruleService()
	if err != nil {
		return nil, err
	}
	return svc.RunTestsForInfo(ruleID, info)
}

func validateRuleInput(ruleID, code, language string) (re.TestResult, error) {
	svc, err := ruleService()
	if err != nil {
		return re.TestResult{}, err
	}
	return svc.Validate(ruleID, code, language)
}

func clearRuleTestCache(ruleID string) error {
	svc, err := ruleService()
	if err != nil {
		return err
	}
	svc.ClearTestCache(ruleID)
	return nil
}

func ruleTestCacheInfo(ruleID string, info RuleInfo) (string, bool, error) {
	svc, err := ruleService()
	if err != nil {
		return "", false, err
	}
	return svc.TestCacheInfo(ruleID, info)
}

func extractRuleTestCases(info RuleInfo) ([]re.TestCase, error) {
	svc, err := ruleService()
	if err != nil {
		return nil, fmt.Errorf("rule service unavailable: %w", err)
	}
	return svc.ExtractTestCases(info)
}
