import { buildApiUrl } from "@vrooli/api-base";
import type { DescribeInterfaceGraphResponse } from "@vrooli/proto-types/scenario-dependency-analyzer/v1/graph/graph_pb";

import { getApiBaseUrl } from "../lib/utils";
import type {
  DependencyGraph,
  DeploymentAnalysisReport,
  GraphType,
  ScenarioDetailResponse,
  ScenarioSummary
} from "../types";

interface ScanOptions {
  apply?: boolean;
  applyResources?: boolean;
  applyScenarios?: boolean;
}

interface OptimizeOptions {
  apply?: boolean;
}

interface AnalysisHealthResponse {
  status: string;
}

export function buildScenarioApiUrl(path: string): string {
  return buildApiUrl(path, { baseUrl: getApiBaseUrl() });
}

async function requestJson<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(buildScenarioApiUrl(path), init);
  if (!response.ok) {
    throw new Error(`Scenario API request failed (${response.status})`);
  }
  return (await response.json()) as T;
}

export function fetchDependencyGraph(type: GraphType, signal?: AbortSignal): Promise<DependencyGraph> {
  return requestJson<DependencyGraph>(`/graph/${type}`, { signal });
}

export function fetchAnalysisHealth(): Promise<AnalysisHealthResponse> {
  return requestJson<AnalysisHealthResponse>("/health/analysis");
}

export async function analyzeAllScenarios(): Promise<void> {
  await requestJson<unknown>("/analyze/all");
}

export function fetchScenarioSummaries(): Promise<ScenarioSummary[]> {
  return requestJson<ScenarioSummary[]>("/scenarios");
}

export function fetchScenarioDetail(name: string): Promise<ScenarioDetailResponse> {
  return requestJson<ScenarioDetailResponse>(`/scenarios/${name}`);
}

export async function scanScenario(name: string, options?: ScanOptions): Promise<void> {
  await requestJson<unknown>(`/scenarios/${name}/scan`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      apply: options?.apply ?? false,
      apply_resources: options?.applyResources ?? false,
      apply_scenarios: options?.applyScenarios ?? false
    })
  });
}

export async function optimizeScenario(name: string, options?: OptimizeOptions): Promise<void> {
  await requestJson<unknown>("/optimize", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      scenario: name,
      type: "all",
      apply: options?.apply ?? false
    })
  });
}

export async function fetchDeploymentReport(scenarioName: string): Promise<DeploymentAnalysisReport | null> {
  try {
    return await requestJson<DeploymentAnalysisReport>(`/scenarios/${scenarioName}/deployment`);
  } catch (error) {
    console.error(`Failed to fetch deployment report for ${scenarioName}:`, error);
    return null;
  }
}

export function fetchActualInterfaceGraphContract(
  signal?: AbortSignal
): Promise<DescribeInterfaceGraphResponse> {
  return requestJson<DescribeInterfaceGraphResponse>(
    "/vrooli.scenario_dependency_analyzer.v1.graph.InterfaceGraphService/DescribeInterfaceGraph",
    {
      body: "{}",
      headers: { "Content-Type": "application/json" },
      method: "POST",
      signal
    }
  );
}
