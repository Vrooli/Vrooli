import { createClient } from "@connectrpc/connect";
import {
  AnalysisOperationsService,
  AnalysisOperationKind,
  AnalysisOperationState
} from "@vrooli/proto-types/scenario-dependency-analyzer/v1/analysis_operations/analysis_operations_pb";
import { buildApiUrl, createScenarioConnectTransport, resolveApiBase } from "@vrooli/api-base";
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

export const API_BASE = resolveApiBase();
export const transport = createScenarioConnectTransport({ baseUrl: API_BASE });

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

export const analysisOperationsClient = createClient(AnalysisOperationsService, transport);
const pendingAnalysisKey = `sda:fleet-analysis:${API_BASE}`;
let pendingAnalysisCaller: string | null = null;

// Retain the admission key across lost responses and page reloads. Replaying it
// recovers the same durable receipt, including a terminal/interrupted outcome.
export async function analyzeAllScenarios(): Promise<void> {
  if (!pendingAnalysisCaller) {
    try { pendingAnalysisCaller = sessionStorage.getItem(pendingAnalysisKey); } catch { /* Session storage may be disabled. */ }
    pendingAnalysisCaller ??= crypto.randomUUID();
    try { sessionStorage.setItem(pendingAnalysisKey, pendingAnalysisCaller); } catch { /* In-memory replay still works. */ }
  }
  const caller = pendingAnalysisCaller;
  const started = await analysisOperationsClient.startOperation({
    clientOperationId: caller,
    kind: AnalysisOperationKind.FLEET_ANALYSIS
  });
  if (!started.operation?.id) throw new Error("Fleet analysis returned no operation receipt");
  const result = await analysisOperationsClient.waitOperation({ operationId: started.operation.id });
  const operation = result.operation;
  if (!operation || ![
    AnalysisOperationState.SUCCEEDED, AnalysisOperationState.PARTIAL,
    AnalysisOperationState.FAILED, AnalysisOperationState.INTERRUPTED
  ].includes(operation.state)) throw new Error("Fleet analysis returned no terminal outcome");
  if (pendingAnalysisCaller === caller) {
    pendingAnalysisCaller = null;
    try { if (sessionStorage.getItem(pendingAnalysisKey) === caller) sessionStorage.removeItem(pendingAnalysisKey); } catch { /* Optional browser storage. */ }
  }
  if (operation.state !== AnalysisOperationState.SUCCEEDED) {
    throw new Error(`Fleet analysis ${operation.id}: ${operation.error || "did not succeed"}`);
  }
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

export async function fetchDeploymentReport(scenarioName: string, signal?: AbortSignal): Promise<DeploymentAnalysisReport | null> {
  try {
    return await requestJson<DeploymentAnalysisReport>(`/scenarios/${scenarioName}/deployment`, { signal });
  } catch (error) {
    if (signal?.aborted) return null;
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
