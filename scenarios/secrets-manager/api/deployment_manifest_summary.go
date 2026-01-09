// Package main provides summary computation for deployment manifests.
//
// The SummaryBuilder computes deployment readiness statistics by analyzing
// secret entries and categorizing them by classification, handling strategy,
// and blocking status.
package main

import "fmt"

// SummaryBuilder computes deployment summary statistics from secret entries.
type SummaryBuilder struct{}

// NewSummaryBuilder creates a new SummaryBuilder instance.
func NewSummaryBuilder() *SummaryBuilder {
	return &SummaryBuilder{}
}

// SummaryInput provides the data needed to build a deployment summary.
type SummaryInput struct {
	// Entries is the list of deployment secrets to summarize.
	Entries []DeploymentSecretEntry

	// ScenarioResources are resources declared in the scenario's service.json.
	ScenarioResources []string

	// AnalyzerResources are resources identified by the dependency analyzer.
	AnalyzerResources []string

	// Scenario is the name of the scenario being analyzed.
	Scenario string
}

// BuildSummary computes a DeploymentSummary from the provided entries.
// It categorizes secrets by classification and handling strategy,
// identifies blocking secrets, and calculates readiness per scope.
func (b *SummaryBuilder) BuildSummary(input SummaryInput) DeploymentSummary {
	classificationTotals := make(map[string]int)
	classificationReady := make(map[string]int)
	strategyBreakdown := make(map[string]int)
	var blockingSecrets []string
	var blockingDetails []BlockingSecretDetail

	for _, entry := range input.Entries {
		classificationTotals[entry.Classification]++

		if entry.HandlingStrategy != "unspecified" {
			classificationReady[entry.Classification]++
			strategyBreakdown[entry.HandlingStrategy]++
		} else {
			detail := b.buildBlockingDetail(entry, input)
			blockingSecrets = append(blockingSecrets, detail.Secret)
			blockingDetails = append(blockingDetails, detail)
		}
	}

	// Limit blocking secrets to prevent response bloat
	blockingSecrets, blockingDetails = b.truncateBlockingInfo(blockingSecrets, blockingDetails, 10)

	summary := DeploymentSummary{
		TotalSecrets:          len(input.Entries),
		StrategizedSecrets:    len(input.Entries) - len(blockingSecrets),
		RequiresAction:        len(blockingSecrets),
		BlockingSecrets:       blockingSecrets,
		BlockingSecretDetails: blockingDetails,
		ClassificationWeights: classificationTotals,
		StrategyBreakdown:     strategyBreakdown,
		ScopeReadiness:        b.computeScopeReadiness(classificationTotals, classificationReady),
	}

	// Ensure non-nil slices for consistent JSON output
	if summary.BlockingSecrets == nil {
		summary.BlockingSecrets = []string{}
	}

	return summary
}

// buildBlockingDetail creates a BlockingSecretDetail for an unstrategized secret.
func (b *SummaryBuilder) buildBlockingDetail(entry DeploymentSecretEntry, input SummaryInput) BlockingSecretDetail {
	secretKey := fmt.Sprintf("%s:%s", entry.ResourceName, entry.SecretKey)
	source := b.determineSource(entry.ResourceName, input.ScenarioResources, input.AnalyzerResources)

	return BlockingSecretDetail{
		Secret:         secretKey,
		Resource:       entry.ResourceName,
		Source:         source,
		DependencyPath: []string{input.Scenario, entry.ResourceName},
	}
}

// determineSource identifies where a resource was discovered.
func (b *SummaryBuilder) determineSource(resourceName string, scenarioResources, analyzerResources []string) string {
	inScenario := containsString(scenarioResources, resourceName)
	inAnalyzer := containsString(analyzerResources, resourceName)

	switch {
	case inScenario && inAnalyzer:
		return "service.json+analyzer"
	case inScenario:
		return "service.json"
	case inAnalyzer:
		return "analyzer"
	default:
		return "unknown"
	}
}

// truncateBlockingInfo limits blocking info to prevent response bloat.
func (b *SummaryBuilder) truncateBlockingInfo(secrets []string, details []BlockingSecretDetail, limit int) ([]string, []BlockingSecretDetail) {
	if len(secrets) > limit {
		secrets = secrets[:limit]
	}
	if len(details) > limit {
		details = details[:limit]
	}
	return secrets, details
}

// computeScopeReadiness calculates readiness strings per classification.
func (b *SummaryBuilder) computeScopeReadiness(totals, ready map[string]int) map[string]string {
	result := make(map[string]string, len(totals))
	for class, total := range totals {
		readyCount := ready[class]
		result[class] = fmt.Sprintf("%d/%d", readyCount, total)
	}
	return result
}
