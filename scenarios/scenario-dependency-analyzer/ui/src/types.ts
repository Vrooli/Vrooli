export type GraphType = "combined" | "resource" | "scenario";

export type LayoutMode = "force" | "radial" | "grid";

export interface GraphMetadata {
  complexity_score?: number;
  [key: string]: unknown;
}

export interface DependencyGraphNode {
  id: string;
  label: string;
  type: "scenario" | "resource" | string;
  group: string;
  metadata?: Record<string, unknown>;
}

export interface DependencyGraphEdge {
  source: string;
  target: string;
  label: string;
  type: string;
  required: boolean;
  weight: number;
  metadata?: Record<string, unknown>;
}

export type EdgeStatusFilter = "all" | "missing" | "declared-only";

export interface DependencyGraph {
  id: string;
  graph_type: GraphType;
  nodes: DependencyGraphNode[];
  edges: DependencyGraphEdge[];
  metadata?: GraphMetadata;
}

export interface DependencyDiffEntry {
  name: string;
  details?: Record<string, unknown>;
}

export interface DependencyDiffSummary {
  missing: DependencyDiffEntry[];
  extra: DependencyDiffEntry[];
}

export interface ScenarioSummary {
  name: string;
  display_name: string;
  description?: string;
  last_scanned?: string;
  tags?: string[];
}

export interface ScenarioDependencyRecord {
  id?: string;
  scenario_name: string;
  dependency_type: string;
  dependency_name: string;
  required: boolean;
  purpose?: string;
  access_method?: string;
  configuration?: Record<string, unknown>;
  discovered_at?: string;
  last_verified?: string;
}

export interface OptimizationRecommendation {
  id: string;
  scenario_name: string;
  recommendation_type: string;
  title: string;
  description?: string;
  priority?: string;
  confidence_score?: number;
  recommended_state?: Record<string, unknown>;
  current_state?: Record<string, unknown>;
  estimated_impact?: Record<string, unknown>;
}

export interface ScenarioDetailResponse {
  scenario: string;
  display_name: string;
  description?: string;
  last_scanned?: string;
  declared_resources: Record<string, unknown>;
  declared_scenarios: Record<string, unknown>;
  stored_dependencies: {
    resources: ScenarioDependencyRecord[];
    scenarios: ScenarioDependencyRecord[];
    shared_workflows: ScenarioDependencyRecord[];
  };
  resource_diff: DependencyDiffSummary;
  scenario_diff: DependencyDiffSummary;
  optimization_recommendations?: OptimizationRecommendation[];
  deployment_report?: DeploymentAnalysisReport;
}

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
  readiness: boolean;
  api_connectivity: {
    connected: boolean;
    api_url: string;
    error?: {
      code: string;
      message: string;
      category: string;
      retryable: boolean;
    } | null;
  };
}

export interface DeploymentAnalysisReport {
  scenario: string;
  report_version: number;
  generated_at: string;
  dependencies: DeploymentDependencyNode[];
  aggregates?: Record<string, DeploymentTierAggregate>;
  bundle_manifest?: BundleManifest;
}

export interface DeploymentDependencyNode {
  name: string;
  type: string;
  resource_type?: string;
  path?: string;
  requirements?: DeploymentRequirements;
  tier_support?: Record<string, TierSupportSummary>;
  alternatives?: string[];
  notes?: string;
  source?: string;
  children?: DeploymentDependencyNode[];
  metadata?: Record<string, unknown>;
}

export interface DeploymentRequirements {
  ram_mb?: number;
  disk_mb?: number;
  cpu_cores?: number;
  gpu?: boolean;
  network?: string;
  storage_mb_per_user?: number;
  startup_time_ms?: number;
  bucket?: string;
  source?: string;
  confidence?: string;
}

export interface TierSupportSummary {
  supported?: boolean;
  fitness_score?: number;
  reason?: string;
  notes?: string;
  requirements?: DeploymentRequirements;
  alternatives?: string[];
}

export interface DeploymentTierAggregate {
  fitness_score: number;
  dependency_count: number;
  blocking_dependencies?: string[];
  estimated_requirements: AggregatedRequirements;
}

export interface AggregatedRequirements {
  ram_mb: number;
  disk_mb: number;
  cpu_cores: number;
}

export interface BundleManifest {
  scenario: string;
  generated_at: string;
  files: BundleFileEntry[];
  dependencies: BundleDependencyEntry[];
}

export interface BundleFileEntry {
  path: string;
  type: string;
  exists: boolean;
  notes?: string;
}

export interface BundleDependencyEntry {
  name: string;
  type: string;
  resource_type?: string;
  tier_support?: Record<string, TierSupportSummary>;
  alternatives?: string[];
}
