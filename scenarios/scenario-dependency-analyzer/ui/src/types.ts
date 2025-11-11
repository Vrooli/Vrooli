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
  scenario_name: string;
  dependency_type: string;
  dependency_name: string;
  required: boolean;
  purpose?: string;
  access_method?: string;
  configuration?: Record<string, unknown>;
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
