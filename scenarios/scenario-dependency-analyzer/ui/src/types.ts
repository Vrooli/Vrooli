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
  metadata_gaps?: DeploymentMetadataGaps;
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
  skeleton?: DesktopBundleSkeleton;
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

export interface DesktopBundleSkeleton {
  schema_version: string;
  target: string;
  app: BundleSkeletonApp;
  ipc: BundleSkeletonIPC;
  telemetry: BundleSkeletonTelemetry;
  ports: BundleSkeletonPorts;
  swaps?: BundleSkeletonSwap[];
  secrets?: BundleSkeletonSecret[];
  services: BundleSkeletonService[];
}

export interface BundleSkeletonApp {
  name: string;
  version: string;
  description?: string;
}

export interface BundleSkeletonIPC {
  mode: string;
  host: string;
  port: number;
  auth_token_path: string;
}

export interface BundleSkeletonTelemetry {
  file: string;
  upload_url?: string;
}

export interface BundleSkeletonPorts {
  default_range: BundleSkeletonPortRange;
  reserved?: number[];
}

export interface BundleSkeletonPortRange {
  min: number;
  max: number;
}

export interface BundleSkeletonSwap {
  original: string;
  replacement: string;
  reason?: string;
  limitations?: string;
}

export interface BundleSkeletonSecret {
  id: string;
  class: string;
  description?: string;
  format?: string;
  required?: boolean;
  prompt?: BundleSkeletonSecretPrompt;
  generator?: Record<string, unknown>;
  target: BundleSkeletonSecretTarget;
}

export interface BundleSkeletonSecretPrompt {
  label?: string;
  description?: string;
}

export interface BundleSkeletonSecretTarget {
  type: string;
  name: string;
}

export interface BundleSkeletonService {
  id: string;
  type: string;
  description?: string;
  binaries: Record<string, BundleSkeletonServiceBinary>;
  env?: Record<string, string>;
  secrets?: string[];
  data_dirs?: string[];
  log_dir?: string;
  ports?: BundleSkeletonServicePorts;
  health: BundleSkeletonHealth;
  readiness: BundleSkeletonReadiness;
  dependencies?: string[];
  migrations?: BundleSkeletonMigration[];
  assets?: BundleSkeletonAsset[];
  gpu?: BundleSkeletonGPU;
  critical?: boolean;
}

export interface BundleSkeletonServiceBinary {
  path: string;
  args?: string[];
  env?: Record<string, string>;
  cwd?: string;
}

export interface BundleSkeletonServicePorts {
  requested: BundleSkeletonRequestedPort[];
}

export interface BundleSkeletonRequestedPort {
  name: string;
  range: BundleSkeletonPortRange;
  requires_socket?: boolean;
}

export interface BundleSkeletonHealth {
  type: string;
  path?: string;
  port_name?: string;
  command?: string[];
  interval_ms?: number;
  timeout_ms?: number;
  retries?: number;
}

export interface BundleSkeletonReadiness {
  type: string;
  port_name?: string;
  pattern?: string;
  timeout_ms?: number;
}

export interface BundleSkeletonMigration {
  version: string;
  command: string[];
  env?: Record<string, string>;
  run_on?: string;
}

export interface BundleSkeletonAsset {
  path: string;
  sha256?: string;
  size_bytes?: number;
}

export interface BundleSkeletonGPU {
  requirement: string;
}

export interface ScenarioGapInfo {
  scenario_name: string;
  scenario_path: string;
  has_deployment_block: boolean;
  missing_dependency_catalog: boolean;
  missing_tier_definitions?: string[];
  missing_resource_metadata?: string[];
  missing_scenario_metadata?: string[];
  suggested_actions?: string[];
}

export interface DeploymentMetadataGaps {
  total_gaps: number;
  scenarios_missing_all: number;
  gaps_by_scenario?: Record<string, ScenarioGapInfo>;
  missing_tiers?: string[];
  recommendations?: string[];
}
