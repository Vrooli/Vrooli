package types

import (
	"time"

	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

// Manifest and its nested declarations are aliases of the control plane's
// canonical service-manifest model. The analyzer must never maintain a parser
// or a second structural interpretation of .vrooli/service.json.
type (
	Manifest                    = scenariomodel.ServiceManifest
	Resource                    = scenariomodel.Dependency
	ScenarioDependencySpec      = scenariomodel.Dependency
	TierFeasibility             = scenariomodel.TierFeasibility
	ServiceBuildConfig          = scenariomodel.ServiceBuildConfig
	DeploymentDependencyCatalog = scenariomodel.DeploymentDependencyCatalog
	DeploymentDependency        = scenariomodel.DeploymentDependency
	DeploymentRequirements      = scenariomodel.DeploymentRequirements
	DependencyTierSupport       = scenariomodel.DependencyTierSupport
	DependencySwap              = scenariomodel.DependencySwap
	DeploymentTier              = scenariomodel.DeploymentTier
	DeploymentAdaptation        = scenariomodel.DeploymentAdaptation
	DeploymentArtifact          = scenariomodel.DeploymentArtifact
	DeploymentOverride          = scenariomodel.DeploymentOverride
)

// ScenarioDependency represents a stored dependency edge.
type ScenarioDependency struct {
	ID             string                 `json:"id" db:"id"`
	ScenarioName   string                 `json:"scenario_name" db:"scenario_name"`
	DependencyType string                 `json:"dependency_type" db:"dependency_type"`
	DependencyName string                 `json:"dependency_name" db:"dependency_name"`
	Required       bool                   `json:"required" db:"required"`
	Purpose        string                 `json:"purpose" db:"purpose"`
	AccessMethod   string                 `json:"access_method" db:"access_method"`
	Configuration  map[string]interface{} `json:"configuration" db:"configuration"`
	DiscoveredAt   time.Time              `json:"discovered_at" db:"discovered_at"`
	LastVerified   time.Time              `json:"last_verified" db:"last_verified"`
}

// DependentScenario describes a scenario that relies on a given dependency.
type DependentScenario struct {
	ScenarioName string                 `json:"scenario_name"`
	Required     bool                   `json:"required"`
	Purpose      string                 `json:"purpose"`
	AccessMethod string                 `json:"access_method"`
	Alternatives []string               `json:"alternatives,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// DependencyImpactReport captures the impact of removing a dependency.
type DependencyImpactReport struct {
	DependencyName     string              `json:"dependency_name"`
	DependencyType     string              `json:"dependency_type"` // "resource", "scenario"
	DirectDependents   []DependentScenario `json:"direct_dependents"`
	IndirectDependents []DependentScenario `json:"indirect_dependents"`
	TotalAffected      int                 `json:"total_affected"`
	CriticalImpact     bool                `json:"critical_impact"` // true if any required dependency
	Severity           string              `json:"severity"`        // "none", "low", "medium", "high", "critical"
	ImpactSummary      string              `json:"impact_summary"`
	Recommendations    []string            `json:"recommendations"`
}

// DependencyGraph models persisted graph nodes/edges.
type DependencyGraph struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"graph_type"`
	Nodes    []GraphNode            `json:"nodes"`
	Edges    []GraphEdge            `json:"edges"`
	Metadata map[string]interface{} `json:"metadata"`
}

// GraphNode represents a node in dependency graph exports.
type GraphNode struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Group    string                 `json:"group"`
	Metadata map[string]interface{} `json:"metadata"`
}

// GraphEdge represents relationships between nodes.
type GraphEdge struct {
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Label    string                 `json:"label"`
	Type     string                 `json:"type"`
	Required bool                   `json:"required"`
	Weight   float64                `json:"weight"`
	Metadata map[string]interface{} `json:"metadata"`
}

// GraphCentralityReport summarizes scenario centrality over the combined graph.
type GraphCentralityReport struct {
	GraphType string                  `json:"graph_type"`
	Scenario  string                  `json:"scenario,omitempty"`
	Nodes     []GraphCentralityMetric `json:"nodes"`
	Metadata  map[string]interface{}  `json:"metadata,omitempty"`
}

// GraphCentralityMetric captures one scenario's centrality and core proximity.
type GraphCentralityMetric struct {
	Scenario                         string   `json:"scenario"`
	DirectReverseDependencyCount     int      `json:"direct_reverse_dependency_count"`
	TransitiveReverseDependencyCount int      `json:"transitive_reverse_dependency_count"`
	RequiredReverseDependencyCount   int      `json:"required_reverse_dependency_count"`
	RequiredEdgeWeightedScore        float64  `json:"required_edge_weighted_score"`
	DistanceToCoreSeed               int      `json:"distance_to_core_seed"`
	NearestCoreSeed                  string   `json:"nearest_core_seed,omitempty"`
	DirectDependents                 []string `json:"direct_dependents,omitempty"`
	TransitiveDependents             []string `json:"transitive_dependents,omitempty"`
}

// UnifiedGraphEdge is one merged, evidence-tagged edge in the persisted
// cross-scenario dependency graph. It is the single source of truth that powers
// `/graph/*` and centrality. A single (From,To) pair carries the union of every
// source that attests it, the highest-confidence source, and OR-ed required-ness.
type UnifiedGraphEdge struct {
	From         string                `json:"from"`
	To           string                `json:"to"`
	Kind         string                `json:"kind"`            // "scenario" | "resource"
	Source       string                `json:"evidence_source"` // highest-confidence attesting source
	Confidence   float64               `json:"confidence"`      // max confidence across sources, [0,1]
	Required     bool                  `json:"required"`
	Evidence     []UnifiedEdgeEvidence `json:"evidence"`
	Stale        bool                  `json:"stale"`         // last-good retained edge (source was unavailable)
	LastVerified time.Time             `json:"last_verified"` // last time a live source re-attested this edge
}

// UnifiedEdgeEvidence is one piece of provenance behind a UnifiedGraphEdge.
type UnifiedEdgeEvidence struct {
	Source     string `json:"source"`
	ImportPath string `json:"import_path,omitempty"`
	FromFile   string `json:"from_file,omitempty"`
	ToFile     string `json:"to_file,omitempty"`
	Path       string `json:"path,omitempty"`
	Analyzer   string `json:"analyzer,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

// AnalysisRequest wraps a request for dependency analysis.
type AnalysisRequest struct {
	ScenarioName      string `json:"scenario_name"`
	IncludeTransitive bool   `json:"include_transitive"`
}

// ProposedScenarioRequest captures an ad-hoc proposal for dependency inference.
type ProposedScenarioRequest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description"`
	Requirements     []string `json:"requirements"`
	SimilarScenarios []string `json:"similar_scenarios,omitempty"`
}

// DependencyAnalysisResponse aggregates scan results for a scenario.
type DependencyAnalysisResponse struct {
	Scenario              string                            `json:"scenario"`
	Resources             []ScenarioDependency              `json:"resources"`
	DetectedResources     []ScenarioDependency              `json:"detected_resources"`
	Scenarios             []ScenarioDependency              `json:"scenarios"`
	DeclaredScenarioSpecs map[string]ScenarioDependencySpec `json:"declared_scenarios"`
	SharedWorkflows       []ScenarioDependency              `json:"shared_workflows"`
	TransitiveDepth       int                               `json:"transitive_depth"`
	ResourceDiff          DependencyDiff                    `json:"resource_diff"`
	ScenarioDiff          DependencyDiff                    `json:"scenario_diff"`
	DeploymentReport      *DeploymentAnalysisReport         `json:"deployment_report,omitempty"`
}

// DependencyDiff highlights missing/extra dependencies after detection.
type DependencyDiff struct {
	Missing []DependencyDrift `json:"missing"`
	Extra   []DependencyDrift `json:"extra"`
}

// DependencyDrift represents a single missing/extra dependency entry.
type DependencyDrift struct {
	Name    string                 `json:"name"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// DeploymentAnalysisReport encapsulates deployment readiness output.
type DeploymentAnalysisReport struct {
	Scenario      string    `json:"scenario"`
	ReportVersion int       `json:"report_version"`
	GeneratedAt   time.Time `json:"generated_at"`
	// Stale is derived when a persisted report's input digest no longer
	// matches the manifests currently on disk. A stale report is evidence of
	// an older computation, never a current deployment answer.
	Stale          bool                               `json:"stale,omitempty"`
	Provenance     DeploymentVerdictProvenance        `json:"provenance"`
	Dependencies   []DeploymentDependencyNode         `json:"dependencies"`
	Aggregates     map[string]DeploymentTierAggregate `json:"aggregates"`
	BundleManifest BundleManifest                     `json:"bundle_manifest"`
	MetadataGaps   *DeploymentMetadataGaps            `json:"metadata_gaps,omitempty"`
}

// DeploymentVerdictProvenance identifies the computation and the authored
// manifests it consumed. The digest lets readers reject stale cached answers.
type DeploymentVerdictProvenance struct {
	Analyzer        string    `json:"analyzer"`
	AnalyzerVersion string    `json:"analyzer_version"`
	ComputedAt      time.Time `json:"computed_at"`
	InputDigest     string    `json:"input_digest"`
}

// DeploymentDependencyNode is a node in the recursive dependency DAG.
type DeploymentDependencyNode struct {
	Name         string                        `json:"name"`
	Type         string                        `json:"type"`
	ResourceType string                        `json:"resource_type,omitempty"`
	Path         string                        `json:"path,omitempty"`
	Required     *bool                         `json:"required,omitempty"`
	Enabled      *bool                         `json:"enabled,omitempty"`
	Requirements *DeploymentRequirements       `json:"requirements,omitempty"`
	TierSupport  map[string]TierSupportSummary `json:"tier_support,omitempty"`
	Alternatives []string                      `json:"alternatives,omitempty"`
	Notes        string                        `json:"notes,omitempty"`
	Source       string                        `json:"source,omitempty"`
	Children     []DeploymentDependencyNode    `json:"children,omitempty"`
	Metadata     map[string]interface{}        `json:"metadata,omitempty"`
}

// TargetDAGResponse is the target-aware dependency export. TargetKind is
// explicit so consumers cannot mistake package/resource graphs for scenario
// graphs.
type TargetDAGResponse struct {
	TargetKind  string                     `json:"target_kind"`
	TargetID    string                     `json:"target_id"`
	TargetRoot  string                     `json:"target_root"`
	Recursive   bool                       `json:"recursive"`
	GeneratedAt time.Time                  `json:"generated_at"`
	DAG         []DeploymentDependencyNode `json:"dag"`
}

// TierSupportSummary summarizes tier fitness info.
type TierSupportSummary struct {
	Supported    *bool                   `json:"supported,omitempty"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Reason       string                  `json:"reason,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Alternatives []string                `json:"alternatives,omitempty"`
}

// DeploymentTierAggregate rolls up dependency counts + requirements for a tier.
type DeploymentTierAggregate struct {
	FitnessScore          float64                `json:"fitness_score"`
	DependencyCount       int                    `json:"dependency_count"`
	BlockingDependencies  []string               `json:"blocking_dependencies,omitempty"`
	EstimatedRequirements AggregatedRequirements `json:"estimated_requirements"`
}

// AggregatedRequirements contains summed requirement estimates for a tier.
type AggregatedRequirements struct {
	RAMMB    float64 `json:"ram_mb"`
	DiskMB   float64 `json:"disk_mb"`
	CPUCores float64 `json:"cpu_cores"`
}

// BundleManifest lists files + dependencies needed for packaging.
type BundleManifest struct {
	Scenario     string                  `json:"scenario"`
	GeneratedAt  time.Time               `json:"generated_at"`
	Files        []BundleFileEntry       `json:"files"`
	Dependencies []BundleDependencyEntry `json:"dependencies"`
	Skeleton     *DesktopBundleSkeleton  `json:"skeleton,omitempty"`
}

// BundleFileEntry documents a file included in deployment bundle.
type BundleFileEntry struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Exists bool   `json:"exists"`
	Notes  string `json:"notes,omitempty"`
}

// BundleDependencyEntry documents a dependency needed for packaging.
type BundleDependencyEntry struct {
	Name         string                        `json:"name"`
	Type         string                        `json:"type"`
	ResourceType string                        `json:"resource_type,omitempty"`
	TierSupport  map[string]TierSupportSummary `json:"tier_support,omitempty"`
	Alternatives []string                      `json:"alternatives,omitempty"`
}

// DesktopBundleSkeleton approximates the full bundle.json consumed by the runtime.
// It is intentionally conservative: defaults are portable, and values are placeholders
// that deployment-manager can refine.
type DesktopBundleSkeleton struct {
	SchemaVersion string                  `json:"schema_version"`
	Target        string                  `json:"target"`
	App           BundleSkeletonApp       `json:"app"`
	IPC           BundleSkeletonIPC       `json:"ipc"`
	Telemetry     BundleSkeletonTelemetry `json:"telemetry"`
	Ports         BundleSkeletonPorts     `json:"ports"`
	Swaps         []BundleSkeletonSwap    `json:"swaps,omitempty"`
	Peers         []BundleSkeletonPeer    `json:"peers,omitempty"`
	Secrets       []BundleSkeletonSecret  `json:"secrets,omitempty"`
	Services      []BundleSkeletonService `json:"services"`
}

type BundleSkeletonApp struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Scenario    string `json:"scenario,omitempty"`
}

type BundleSkeletonPeer struct {
	Scenario         string              `json:"scenario"`
	BundlePolicy     string              `json:"bundle_policy"`
	StartupPolicy    string              `json:"startup_policy,omitempty"`
	DegradedBehavior string              `json:"degraded_behavior,omitempty"`
	Bindings         []BundlePeerBinding `json:"bindings"`
}

type BundlePeerBinding struct {
	EnvVar          string `json:"env_var"`
	Form            string `json:"form"`
	Port            string `json:"port"`
	WhenUnavailable string `json:"when_unavailable"`
}

type BundleSkeletonIPC struct {
	Mode          string `json:"mode"`
	Host          string `json:"host"`
	Port          int    `json:"port"`
	AuthTokenPath string `json:"auth_token_path"`
}

type BundleSkeletonTelemetry struct {
	File      string `json:"file"`
	UploadURL string `json:"upload_url,omitempty"`
}

type BundleSkeletonPorts struct {
	DefaultRange BundleSkeletonPortRange `json:"default_range"`
	Reserved     []int                   `json:"reserved,omitempty"`
}

type BundleSkeletonPortRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type BundleSkeletonSwap struct {
	Original    string `json:"original"`
	Replacement string `json:"replacement"`
	Reason      string `json:"reason,omitempty"`
	Limitations string `json:"limitations,omitempty"`
}

type BundleSkeletonSecret struct {
	ID          string                      `json:"id"`
	Class       string                      `json:"class"`
	Description string                      `json:"description,omitempty"`
	Format      string                      `json:"format,omitempty"`
	Required    *bool                       `json:"required,omitempty"`
	Prompt      *BundleSkeletonSecretPrompt `json:"prompt,omitempty"`
	Generator   map[string]interface{}      `json:"generator,omitempty"`
	Target      BundleSkeletonSecretTarget  `json:"target"`
}

type BundleSkeletonSecretPrompt struct {
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

type BundleSkeletonSecretTarget struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type BundleSkeletonService struct {
	ID           string                                 `json:"id"`
	Type         string                                 `json:"type"`
	Description  string                                 `json:"description,omitempty"`
	Binaries     map[string]BundleSkeletonServiceBinary `json:"binaries"`
	Build        *BundleSkeletonBuildConfig             `json:"build,omitempty"`
	Env          map[string]string                      `json:"env,omitempty"`
	Secrets      []string                               `json:"secrets,omitempty"`
	DataDirs     []string                               `json:"data_dirs,omitempty"`
	LogDir       string                                 `json:"log_dir,omitempty"`
	Ports        *BundleSkeletonServicePorts            `json:"ports,omitempty"`
	Health       BundleSkeletonHealth                   `json:"health"`
	Readiness    BundleSkeletonReadiness                `json:"readiness"`
	Dependencies []string                               `json:"dependencies,omitempty"`
	Migrations   []BundleSkeletonMigration              `json:"migrations,omitempty"`
	Assets       []BundleSkeletonAsset                  `json:"assets,omitempty"`
	DistRoot     string                                 `json:"dist_root,omitempty"`
	GPU          *BundleSkeletonGPU                     `json:"gpu,omitempty"`
	Critical     *bool                                  `json:"critical,omitempty"`
}

// BundleSkeletonBuildConfig specifies how to compile a service binary when not pre-built.
// This enables automatic cross-compilation during bundle packaging.
type BundleSkeletonBuildConfig struct {
	// Type is the build system: "go", "rust", "npm", "python", or "custom"
	Type string `json:"type"`
	// SourceDir is the relative path to the source code directory
	SourceDir string `json:"source_dir"`
	// EntryPoint is the main file or package (e.g., "." for Go, "src/main.rs" for Rust)
	EntryPoint string `json:"entry_point,omitempty"`
	// OutputPattern is the output path pattern with {{platform}} and {{ext}} placeholders
	OutputPattern string `json:"output_pattern,omitempty"`
	// BuildCommand is the custom build command (for type="custom")
	BuildCommand string `json:"build_command,omitempty"`
	// BuildArgs are additional arguments to pass to the build command
	BuildArgs []string `json:"build_args,omitempty"`
	// Env are environment variables to set during build
	Env map[string]string `json:"env,omitempty"`
}

type BundleSkeletonServiceBinary struct {
	Path string            `json:"path"`
	Args []string          `json:"args,omitempty"`
	Env  map[string]string `json:"env,omitempty"`
	Cwd  string            `json:"cwd,omitempty"`
}

type BundleSkeletonServicePorts struct {
	Requested []BundleSkeletonRequestedPort `json:"requested"`
}

type BundleSkeletonRequestedPort struct {
	Name           string                  `json:"name"`
	Range          BundleSkeletonPortRange `json:"range"`
	RequiresSocket bool                    `json:"requires_socket,omitempty"`
}

type BundleSkeletonHealth struct {
	Type     string   `json:"type"`
	Path     string   `json:"path,omitempty"`
	PortName string   `json:"port_name,omitempty"`
	Command  []string `json:"command,omitempty"`
	Interval int      `json:"interval_ms,omitempty"`
	Timeout  int      `json:"timeout_ms,omitempty"`
	Retries  int      `json:"retries,omitempty"`
}

type BundleSkeletonReadiness struct {
	Type     string `json:"type"`
	PortName string `json:"port_name,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Timeout  int    `json:"timeout_ms,omitempty"`
}

type BundleSkeletonMigration struct {
	Version string            `json:"version"`
	Command []string          `json:"command"`
	Env     map[string]string `json:"env,omitempty"`
	RunOn   string            `json:"run_on,omitempty"`
}

type BundleSkeletonAsset struct {
	Path      string `json:"path"`
	SHA256    string `json:"sha256,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type BundleSkeletonGPU struct {
	Requirement string `json:"requirement"`
}

// DeploymentMetadataGaps reports missing deployment metadata across the dependency tree.
type DeploymentMetadataGaps struct {
	TotalGaps               int                        `json:"total_gaps"`
	ScenariosMissingAll     int                        `json:"scenarios_missing_all"`
	GapsByScenario          map[string]ScenarioGapInfo `json:"gaps_by_scenario"`
	MissingTiers            []string                   `json:"missing_tiers"`
	SecretRequirements      []SecretRequirement        `json:"secret_requirements,omitempty"`
	ResourceSwapSuggestions []ResourceSwapSuggestion   `json:"resource_swap_suggestions,omitempty"`
	Recommendations         []string                   `json:"recommendations"`
}

// ScenarioGapInfo describes metadata gaps for a single scenario in the tree.
type ScenarioGapInfo struct {
	ScenarioName             string   `json:"scenario_name"`
	ScenarioPath             string   `json:"scenario_path,omitempty"`
	HasTierFeasibility       bool     `json:"has_tier_feasibility"`
	MissingDependencyCatalog bool     `json:"missing_dependency_catalog"`
	MissingTierDefinitions   []string `json:"missing_tier_definitions,omitempty"`
	MissingResourceMetadata  []string `json:"missing_resource_metadata,omitempty"`
	MissingScenarioMetadata  []string `json:"missing_scenario_metadata,omitempty"`
	SuggestedActions         []string `json:"suggested_actions,omitempty"`
}

// OptimizationRequest represents a CLI/API optimization request payload.
type OptimizationRequest struct {
	Scenario string `json:"scenario"`
	Type     string `json:"type"`
	Apply    bool   `json:"apply"`
}

// OptimizationResult is returned after running optimizations for a scenario.
type OptimizationResult struct {
	Scenario          string                       `json:"scenario"`
	Recommendations   []OptimizationRecommendation `json:"recommendations"`
	Summary           OptimizationSummary          `json:"summary"`
	Applied           bool                         `json:"applied"`
	ApplySummary      map[string]interface{}       `json:"apply_summary,omitempty"`
	AnalysisTimestamp time.Time                    `json:"analysis_timestamp"`
	Error             string                       `json:"error,omitempty"`
}

// OptimizationSummary holds aggregate stats for recommendations.
type OptimizationSummary struct {
	RecommendationCount int            `json:"recommendation_count"`
	ByType              map[string]int `json:"by_type"`
	HighPriority        int            `json:"high_priority"`
	PotentialImpact     map[string]int `json:"potential_impact"`
}

// OptimizationRecommendation describes an actionable optimization proposal.
type OptimizationRecommendation struct {
	ID                 string                 `json:"id"`
	ScenarioName       string                 `json:"scenario_name"`
	RecommendationType string                 `json:"recommendation_type"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	CurrentState       map[string]interface{} `json:"current_state"`
	RecommendedState   map[string]interface{} `json:"recommended_state"`
	EstimatedImpact    map[string]interface{} `json:"estimated_impact"`
	ConfidenceScore    float64                `json:"confidence_score"`
	Priority           string                 `json:"priority"`
	Status             string                 `json:"status"`
	CreatedAt          time.Time              `json:"created_at"`
}

// ScenarioSummary provides high-level information for the catalog.
type ScenarioSummary struct {
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	LastScanned *time.Time `json:"last_scanned,omitempty"`
	Tags        []string   `json:"tags"`
}

// ScenarioDetailResponse powers the scenario detail panel response.
type ScenarioDetailResponse struct {
	Scenario                    string                            `json:"scenario"`
	DisplayName                 string                            `json:"display_name"`
	Description                 string                            `json:"description"`
	LastScanned                 *time.Time                        `json:"last_scanned,omitempty"`
	DeclaredResources           map[string]Resource               `json:"declared_resources"`
	DeclaredScenarios           map[string]ScenarioDependencySpec `json:"declared_scenarios"`
	StoredDependencies          map[string][]ScenarioDependency   `json:"stored_dependencies"`
	ResourceDiff                DependencyDiff                    `json:"resource_diff"`
	ScenarioDiff                DependencyDiff                    `json:"scenario_diff"`
	OptimizationRecommendations []OptimizationRecommendation      `json:"optimization_recommendations"`
	DeploymentReport            *DeploymentAnalysisReport         `json:"deployment_report,omitempty"`
}

// ScanRequest controls scan/apply behavior for a scenario.
type ScanRequest struct {
	Apply          bool `json:"apply"`
	ApplyResources bool `json:"apply_resources"`
	ApplyScenarios bool `json:"apply_scenarios"`
}

// SecretRequirement identifies a dependency that needs secret configuration
type SecretRequirement struct {
	DependencyName    string   `json:"dependency_name"`
	DependencyType    string   `json:"dependency_type"`
	SecretType        string   `json:"secret_type"`
	RequiredSecrets   []string `json:"required_secrets"`
	PlaybookReference string   `json:"playbook_reference"`
	Priority          string   `json:"priority"`
}

// ResourceSwapSuggestion recommends a lighter alternative for specific deployment tiers
type ResourceSwapSuggestion struct {
	OriginalResource    string   `json:"original_resource"`
	AlternativeResource string   `json:"alternative_resource"`
	Reason              string   `json:"reason"`
	ApplicableTiers     []string `json:"applicable_tiers"`
	Relationship        string   `json:"relationship"`
	ImpactDescription   string   `json:"impact_description"`
}
