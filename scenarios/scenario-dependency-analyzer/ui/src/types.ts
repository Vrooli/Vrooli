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
