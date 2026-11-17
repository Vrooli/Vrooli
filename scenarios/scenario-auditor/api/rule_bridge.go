package main

import (
	re "scenario-auditor/internal/ruleengine"
)

// Compatibility aliases for legacy code paths.
type (
	RuleInfo                 = re.Info
	RuleImplementationStatus = re.ImplementationStatus
	RuleExecutionInfo        = re.ExecutionInfo
	RuleArgumentInfo         = re.ArgumentInfo
	RuleExecutionCall        = re.ExecutionCall
)

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

func runRuleTests(ruleID string, info RuleInfo) ([]re.TestResult, error) {
	svc, err := ruleService()
	if err != nil {
		return nil, err
	}
	return svc.RunTestsForInfo(ruleID, info)
}
