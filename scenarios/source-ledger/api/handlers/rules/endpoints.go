package rules

import (
	rulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/rules/rules_v1connect"
	"source-ledger/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "rules_list", Path: rulesconnect.ClassificationRulesServiceListRulesProcedure, Method: "POST", Summary: "List classification rules", Category: "rules"},
	{ID: "rules_create", Path: rulesconnect.ClassificationRulesServiceCreateRuleProcedure, Method: "POST", Summary: "Create classification rule", Category: "rules"},
	{ID: "rules_dry_run", Path: rulesconnect.ClassificationRulesServiceDryRunRuleProcedure, Method: "POST", Summary: "Dry-run classification rule", Category: "rules"},
	{ID: "rules_enable", Path: rulesconnect.ClassificationRulesServiceEnableRuleProcedure, Method: "POST", Summary: "Enable classification rule", Category: "rules"},
	{ID: "rules_revert", Path: rulesconnect.ClassificationRulesServiceRevertRuleProcedure, Method: "POST", Summary: "Revert rule assignments", Category: "rules"},
	{ID: "rules_refacet", Path: rulesconnect.ClassificationRulesServiceRefacetCorpusProcedure, Method: "POST", Summary: "Re-facet the immutable corpus", Category: "rules", Request: &module.Schema{Type: "RefacetCorpusRequest"}, Response: &module.Schema{Type: "RefacetCorpusResponse"}},
	{ID: "rules_measure_distribution", Path: rulesconnect.ClassificationRulesServiceMeasureDistributionProcedure, Method: "POST", Summary: "Measure rule coverage and classifier tail", Category: "rules", Request: &module.Schema{Type: "MeasureDistributionRequest"}, Response: &module.Schema{Type: "MeasureDistributionResponse"}},
}
