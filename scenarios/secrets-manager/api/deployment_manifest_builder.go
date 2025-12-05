// Package main provides the ManifestBuilder for deployment manifest generation.
//
// ManifestBuilder orchestrates the generation of deployment manifests by
// coordinating multiple specialized components:
//   - SecretStore: Database access for secret retrieval
//   - AnalyzerClient: Integration with scenario-dependency-analyzer
//   - ResourceResolver: Resource discovery and merging
//   - SummaryBuilder: Deployment readiness computation
//   - BundlePlanBuilder: Bundle secret plan derivation
//
// This architecture enables:
//   - Clear separation of concerns
//   - Dependency injection for testing
//   - Consistent error handling and logging
package main

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ManifestBuilder orchestrates deployment manifest generation.
type ManifestBuilder struct {
	secretStore       SecretStore
	analyzerClient    AnalyzerClient
	resourceResolver  ResourceResolver
	summaryBuilder    *SummaryBuilder
	bundlePlanBuilder *BundlePlanBuilder
	logger            *Logger
}

// ManifestBuilderConfig configures ManifestBuilder construction.
type ManifestBuilderConfig struct {
	DB     *sql.DB
	Logger *Logger
}

// NewManifestBuilder creates a production ManifestBuilder with all dependencies.
func NewManifestBuilder(cfg ManifestBuilderConfig) *ManifestBuilder {
	var secretStore SecretStore
	if cfg.DB != nil {
		secretStore = NewPostgresSecretStore(cfg.DB, cfg.Logger)
	}

	analyzerClient := NewHTTPAnalyzerClient(cfg.Logger)
	resourceResolver := NewResourceResolver(analyzerClient, cfg.Logger)

	return &ManifestBuilder{
		secretStore:       secretStore,
		analyzerClient:    analyzerClient,
		resourceResolver:  resourceResolver,
		summaryBuilder:    NewSummaryBuilder(),
		bundlePlanBuilder: NewBundlePlanBuilder(),
		logger:            cfg.Logger,
	}
}

// NewManifestBuilderWithDeps creates a ManifestBuilder with explicit dependencies.
// This is primarily used for testing with mock implementations.
func NewManifestBuilderWithDeps(
	secretStore SecretStore,
	analyzerClient AnalyzerClient,
	resourceResolver ResourceResolver,
	logger *Logger,
) *ManifestBuilder {
	return &ManifestBuilder{
		secretStore:       secretStore,
		analyzerClient:    analyzerClient,
		resourceResolver:  resourceResolver,
		summaryBuilder:    NewSummaryBuilder(),
		bundlePlanBuilder: NewBundlePlanBuilder(),
		logger:            logger,
	}
}

// Build generates a deployment manifest for the given request.
//
// The build process:
//  1. Validates inputs (scenario and tier are required)
//  2. Resolves effective resources from service.json, analyzer, and request
//  3. Fetches secrets from database (or builds fallback if unavailable)
//  4. Computes deployment summary and readiness
//  5. Derives bundle secret plans
//  6. Enriches with analyzer insights if available
//  7. Persists telemetry for audit
func (b *ManifestBuilder) Build(ctx context.Context, req DeploymentManifestRequest) (*DeploymentManifest, error) {
	// Validate inputs
	scenario := strings.TrimSpace(req.Scenario)
	tier := strings.TrimSpace(strings.ToLower(req.Tier))
	if scenario == "" || tier == "" {
		return nil, fmt.Errorf("scenario and tier are required")
	}

	// Resolve effective resources
	resolved := b.resourceResolver.ResolveResources(ctx, scenario, req.Resources)

	// Handle no-database fallback
	if b.secretStore == nil {
		return b.buildFallbackManifest(scenario, tier, resolved.Effective), nil
	}

	// Fetch secrets from database (with scenario-specific overrides if any)
	entries, err := b.secretStore.FetchSecrets(ctx, scenario, tier, resolved.Effective, req.IncludeOptional)
	if err != nil {
		return nil, fmt.Errorf("fetch secrets: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no secrets discovered for manifest request")
	}

	// Build resource list from fetched entries
	resourceSet := make(map[string]struct{})
	for _, entry := range entries {
		resourceSet[entry.ResourceName] = struct{}{}
	}
	resourcesList := make([]string, 0, len(resourceSet))
	for resource := range resourceSet {
		resourcesList = append(resourcesList, resource)
	}

	// Merge with analyzer resources
	resourcesList = mergeResourceLists(resourcesList, resolved.FromAnalyzer)
	sort.Strings(resourcesList)

	// Fetch analyzer insights
	analyzerReport, _ := b.analyzerClient.FetchDeploymentReport(ctx, scenario)

	// Build summary
	summary := b.summaryBuilder.BuildSummary(SummaryInput{
		Entries:           entries,
		ScenarioResources: resolved.FromServiceJSON,
		AnalyzerResources: resolved.FromAnalyzer,
		Scenario:          scenario,
	})

	// Derive bundle plans
	bundlePlans := b.bundlePlanBuilder.DeriveBundlePlans(entries)

	// Assemble manifest
	manifest := &DeploymentManifest{
		Scenario:      scenario,
		Tier:          tier,
		GeneratedAt:   time.Now(),
		Resources:     resourcesList,
		Secrets:       entries,
		BundleSecrets: bundlePlans,
		Summary:       summary,
	}

	// Add analyzer insights if available
	if analyzerReport != nil {
		manifest.Dependencies = convertAnalyzerDependencies(analyzerReport)
		manifest.TierAggregates = convertAnalyzerAggregates(analyzerReport)
		reportTime := analyzerReport.GeneratedAt
		manifest.AnalyzerGeneratedAt = &reportTime
	}

	// Persist telemetry (non-blocking)
	if b.secretStore != nil {
		_ = b.secretStore.PersistManifest(ctx, scenario, tier, manifest)
	}

	return manifest, nil
}

// buildFallbackManifest creates a manifest when database is unavailable.
// This ensures bundle consumers always receive a valid manifest structure
// with prompt-based secrets that can be filled in manually.
func (b *ManifestBuilder) buildFallbackManifest(scenario, tier string, resources []string) *DeploymentManifest {
	defaultResources := resources
	if len(defaultResources) == 0 {
		defaultResources = []string{"core-platform"}
	}

	entries := make([]DeploymentSecretEntry, 0, len(defaultResources))
	for idx, resource := range defaultResources {
		entries = append(entries, DeploymentSecretEntry{
			ResourceName:      resource,
			SecretKey:         fmt.Sprintf("PLACEHOLDER_SECRET_%d", idx+1),
			SecretType:        "token",
			Required:          true,
			Classification:    "service",
			Description:       "Fallback manifest entry (no database connection)",
			HandlingStrategy:  "prompt",
			RequiresUserInput: true,
			Prompt: &PromptMetadata{
				Label:       "Provide secret",
				Description: "Enter the secure value manually",
			},
			TierStrategies: map[string]string{
				tier: "prompt",
			},
		})
	}

	summary := DeploymentSummary{
		TotalSecrets:          len(entries),
		StrategizedSecrets:    len(entries),
		RequiresAction:        0,
		BlockingSecrets:       []string{},
		ClassificationWeights: map[string]int{"service": len(entries)},
		StrategyBreakdown:     map[string]int{"prompt": len(entries)},
		ScopeReadiness:        map[string]string{"service": fmt.Sprintf("%d/%d", len(entries), len(entries))},
	}

	manifest := &DeploymentManifest{
		Scenario:    scenario,
		Tier:        tier,
		GeneratedAt: time.Now(),
		Resources:   defaultResources,
		Secrets:     entries,
		Summary:     summary,
	}

	manifest.BundleSecrets = b.bundlePlanBuilder.DeriveBundlePlans(entries)
	return manifest
}

// convertAnalyzerDependencies transforms analyzer dependencies to manifest format.
func convertAnalyzerDependencies(report *analyzerDeploymentReport) []DependencyInsight {
	if report == nil {
		return nil
	}

	insights := make([]DependencyInsight, 0, len(report.Dependencies))
	for _, dep := range report.Dependencies {
		insight := DependencyInsight{
			Name:         dep.Name,
			Kind:         dep.Type,
			ResourceType: dep.ResourceType,
			Source:       dep.Source,
			Alternatives: dep.Alternatives,
		}

		// Include requirements if any are non-zero
		if dep.Requirements.RAMMB != 0 || dep.Requirements.DiskMB != 0 ||
			dep.Requirements.CPUCores != 0 || dep.Requirements.Network != "" {
			insight.Requirements = &DependencyRequirementSummary{
				RAMMB:      dep.Requirements.RAMMB,
				DiskMB:     dep.Requirements.DiskMB,
				CPUCores:   dep.Requirements.CPUCores,
				Network:    dep.Requirements.Network,
				Source:     dep.Requirements.Source,
				Confidence: dep.Requirements.Confidence,
			}
		}

		// Include tier support information
		if len(dep.TierSupport) > 0 {
			insight.TierSupport = make(map[string]DependencyTierSupportView, len(dep.TierSupport))
			for tier, support := range dep.TierSupport {
				insight.TierSupport[tier] = DependencyTierSupportView{
					Supported:    support.Supported,
					FitnessScore: support.FitnessScore,
					Notes:        support.Notes,
					Reason:       support.Reason,
					Alternatives: support.Alternatives,
				}
			}
		}

		insights = append(insights, insight)
	}
	return insights
}

// convertAnalyzerAggregates transforms analyzer aggregates to manifest format.
func convertAnalyzerAggregates(report *analyzerDeploymentReport) map[string]TierAggregateView {
	if report == nil || len(report.Aggregates) == 0 {
		return nil
	}

	result := make(map[string]TierAggregateView, len(report.Aggregates))
	for tier, aggregate := range report.Aggregates {
		view := TierAggregateView{
			FitnessScore:         aggregate.FitnessScore,
			DependencyCount:      aggregate.DependencyCount,
			BlockingDependencies: aggregate.BlockingDependencies,
		}

		// Include requirements if any are non-zero
		if aggregate.EstimatedRequirements.RAMMB != 0 ||
			aggregate.EstimatedRequirements.DiskMB != 0 ||
			aggregate.EstimatedRequirements.CPUCores != 0 {
			view.EstimatedRequirements = &DependencyRequirementSummary{
				RAMMB:    aggregate.EstimatedRequirements.RAMMB,
				DiskMB:   aggregate.EstimatedRequirements.DiskMB,
				CPUCores: aggregate.EstimatedRequirements.CPUCores,
			}
		}

		result[tier] = view
	}
	return result
}
