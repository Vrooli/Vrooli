package types

import "time"

// ServiceConfig represents the full .vrooli/service.json structure for a scenario.
// Supports both v1.x (service block) and v2.0 (flat structure) formats.
type ServiceConfig struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`

	// v1.x format uses a nested "service" block
	Service struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"display_name"`
		Description string   `json:"description"`
		Version     string   `json:"version"`
		Tags        []string `json:"tags"`
	} `json:"service"`

	// v2.0 format uses flat fields
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags"`
	Ports       map[string]interface{} `json:"ports"`
	Resources   map[string]Resource    `json:"resources"`
	Scenarios   map[string]interface{} `json:"scenarios"`
	Lifecycle   interface{}            `json:"lifecycle"`

	// Common fields
	Dependencies struct {
		Resources map[string]Resource               `json:"resources"`
		Scenarios map[string]ScenarioDependencySpec `json:"scenarios"`
	} `json:"dependencies"`
	Deployment *ServiceDeployment `json:"deployment"`
}

// Resource describes a single resource dependency declaration.
type Resource struct {
	Type           string                   `json:"type"`
	Enabled        bool                     `json:"enabled"`
	Required       bool                     `json:"required"`
	Purpose        string                   `json:"purpose"`
	Initialization []map[string]interface{} `json:"initialization,omitempty"`
	Models         []string                 `json:"models,omitempty"`
}

// ScenarioDependencySpec captures declared scenario dependencies.
type ScenarioDependencySpec struct {
	Required     bool   `json:"required"`
	Version      string `json:"version"`
	VersionRange string `json:"versionRange"`
	Description  string `json:"description"`
}

// ServiceDeployment stores deployment metadata and tier readiness.
// Supports both v1.x (analyzer-generated) and v2.0 (user-defined) formats.
type ServiceDeployment struct {
	// v1.x analyzer-generated fields
	MetadataVersion       int                         `json:"metadata_version"`
	LastAnalyzedAt        string                      `json:"last_analyzed_at"`
	Analyzer              *DeploymentAnalyzerInfo     `json:"analyzer"`
	AggregateRequirements *DeploymentRequirements     `json:"aggregate_requirements"`
	Tiers                 map[string]DeploymentTier   `json:"tiers"`
	Dependencies          DeploymentDependencyCatalog `json:"dependencies"`
	Overrides             []DeploymentOverride        `json:"overrides"`

	// v2.0 user-defined fields
	SupportedTiers   []int                         `json:"supported_tiers"`
	Platforms        []string                      `json:"platforms"`
	DesktopReady     bool                          `json:"desktop_ready"`
	MinimalResources []string                      `json:"minimal_resources"`
	BuildConfigs     map[string]ServiceBuildConfig `json:"build_configs"`
}

// ServiceBuildConfig specifies how to build a service component.
type ServiceBuildConfig struct {
	Type          string `json:"type"`           // "go", "rust", "npm", "python", "custom"
	SourceDir     string `json:"source_dir"`     // relative path to source directory
	EntryPoint    string `json:"entry_point"`    // main file or package (e.g., "." for Go)
	OutputPattern string `json:"output_pattern"` // output path with {{platform}} and {{ext}} placeholders
}

// DeploymentAnalyzerInfo identifies the analyzer that produced the deployment block.
type DeploymentAnalyzerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// DeploymentDependencyCatalog captures curated metadata for dependencies.
type DeploymentDependencyCatalog struct {
	Resources map[string]DeploymentDependency `json:"resources"`
	Scenarios map[string]DeploymentDependency `json:"scenarios"`
}

// DeploymentDependency describes a single dependency's fitness metadata.
type DeploymentDependency struct {
	ResourceType    string                           `json:"resource_type"`
	Footprint       *DeploymentRequirements          `json:"footprint"`
	PlatformSupport map[string]DependencyTierSupport `json:"platform_support"`
	SwappableWith   []DependencySwap                 `json:"swappable_with"`
	PackagingHints  []string                         `json:"packaging_hints"`
}

// DeploymentRequirements defines the estimated requirement footprint.
type DeploymentRequirements struct {
	RAMMB            *float64 `json:"ram_mb,omitempty"`
	DiskMB           *float64 `json:"disk_mb,omitempty"`
	CPUCores         *float64 `json:"cpu_cores,omitempty"`
	GPU              *bool    `json:"gpu,omitempty"`
	Network          string   `json:"network,omitempty"`
	StorageMBPerUser *float64 `json:"storage_mb_per_user,omitempty"`
	StartupTimeMS    *float64 `json:"startup_time_ms,omitempty"`
	Bucket           string   `json:"bucket,omitempty"`
	Source           string   `json:"source,omitempty"`
	Confidence       string   `json:"confidence,omitempty"`
}

// DependencyTierSupport stores per-tier info for a dependency.
type DependencyTierSupport struct {
	Supported    *bool                   `json:"supported,omitempty"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Reason       string                  `json:"reason,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Alternatives []string                `json:"alternatives,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

// DependencySwap captures an alternative dependency option.
type DependencySwap struct {
	ID           string `json:"id"`
	Relationship string `json:"relationship"`
	Notes        string `json:"notes"`
}

// DeploymentTier holds tier readiness metadata.
type DeploymentTier struct {
	Status       string                  `json:"status"`
	FitnessScore *float64                `json:"fitness_score,omitempty"`
	Constraints  []string                `json:"constraints,omitempty"`
	Requirements *DeploymentRequirements `json:"requirements,omitempty"`
	Adaptations  []DeploymentAdaptation  `json:"adaptations,omitempty"`
	Secrets      []DeploymentTierSecret  `json:"secrets,omitempty"`
	Artifacts    []DeploymentArtifact    `json:"artifacts,omitempty"`
	Notes        string                  `json:"notes,omitempty"`
}

// DeploymentAdaptation describes a recommended swap for a tier.
type DeploymentAdaptation struct {
	Dependency string  `json:"dependency"`
	Swap       string  `json:"swap"`
	Impact     string  `json:"impact"`
	EffortDays float64 `json:"effort_days"`
	Notes      string  `json:"notes"`
}

// DeploymentTierSecret encodes secret requirements for a tier.
type DeploymentTierSecret struct {
	SecretID       string `json:"secret_id"`
	Classification string `json:"classification"`
	StrategyRef    string `json:"strategy_ref"`
	Notes          string `json:"notes"`
}

// DeploymentArtifact represents packaging artifacts per tier.
type DeploymentArtifact struct {
	Type     string `json:"type"`
	Producer string `json:"producer"`
	Status   string `json:"status"`
	Notes    string `json:"notes"`
}

// DeploymentOverride stores explicit overrides for deployment metadata.
type DeploymentOverride struct {
	Tier      string      `json:"tier"`
	Field     string      `json:"field"`
	Value     interface{} `json:"value"`
	Reason    string      `json:"reason"`
	ExpiresAt string      `json:"expires_at"`
}

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
	Scenario       string                             `json:"scenario"`
	ReportVersion  int                                `json:"report_version"`
	GeneratedAt    time.Time                          `json:"generated_at"`
	Dependencies   []DeploymentDependencyNode         `json:"dependencies"`
	Aggregates     map[string]DeploymentTierAggregate `json:"aggregates"`
	BundleManifest BundleManifest                     `json:"bundle_manifest"`
	MetadataGaps   *DeploymentMetadataGaps            `json:"metadata_gaps,omitempty"`
}

// DeploymentDependencyNode is a node in the recursive dependency DAG.
type DeploymentDependencyNode struct {
	Name         string                        `json:"name"`
	Type         string                        `json:"type"`
	ResourceType string                        `json:"resource_type,omitempty"`
	Path         string                        `json:"path,omitempty"`
	Requirements *DeploymentRequirements       `json:"requirements,omitempty"`
	TierSupport  map[string]TierSupportSummary `json:"tier_support,omitempty"`
	Alternatives []string                      `json:"alternatives,omitempty"`
	Notes        string                        `json:"notes,omitempty"`
	Source       string                        `json:"source,omitempty"`
	Children     []DeploymentDependencyNode    `json:"children,omitempty"`
	Metadata     map[string]interface{}        `json:"metadata,omitempty"`
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
	Secrets       []BundleSkeletonSecret  `json:"secrets,omitempty"`
	Services      []BundleSkeletonService `json:"services"`
}

type BundleSkeletonApp struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
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
	HasDeploymentBlock       bool     `json:"has_deployment_block"`
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
