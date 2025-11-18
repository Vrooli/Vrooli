
package services

import types "scenario-dependency-analyzer/internal/types"

// AnalysisService exposes dependency analysis operations.
type AnalysisService interface {
	AnalyzeScenario(name string) (*types.DependencyAnalysisResponse, error)
	AnalyzeAllScenarios() (map[string]*types.DependencyAnalysisResponse, error)
}

// ScanResult captures the outcome of a scan/apply operation.
type ScanResult struct {
	Analysis     *types.DependencyAnalysisResponse
	Applied      bool
	ApplySummary map[string]interface{}
}

// ScanService executes scan/apply workflows for scenarios.
type ScanService interface {
	ScanScenario(name string, req types.ScanRequest) (*ScanResult, error)
}

// GraphService generates dependency graph exports.
type GraphService interface {
	GenerateGraph(graphType string) (*types.DependencyGraph, error)
}

// OptimizationService coordinates optimization recommendation workflows.
type OptimizationService interface {
	RunOptimization(req types.OptimizationRequest) (map[string]*types.OptimizationResult, error)
}

// ScenarioService exposes catalog and detail operations.
type ScenarioService interface {
	ListScenarios() ([]types.ScenarioSummary, error)
	GetScenarioDetail(name string) (*types.ScenarioDetailResponse, error)
}

// DeploymentService provides access to computed deployment reports.
type DeploymentService interface {
	GetDeploymentReport(name string) (*types.DeploymentAnalysisReport, error)
}

// ProposalService handles proposed scenario analysis requests.
type ProposalService interface {
	AnalyzeProposedScenario(req types.ProposedScenarioRequest) (map[string]interface{}, error)
}

// Registry aggregates the services available to handlers and other consumers.
type Registry struct {
	Analysis     AnalysisService
	Scan         ScanService
	Graph        GraphService
	Optimization OptimizationService
	Scenarios    ScenarioService
	Deployment   DeploymentService
	Proposal     ProposalService
}
