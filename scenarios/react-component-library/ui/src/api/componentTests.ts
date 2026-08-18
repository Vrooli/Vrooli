import { API_BASE, decodeApiError } from "./client";

const procedure = "/vrooli.react_component_library.v1.componenttests.ComponentTestsService";
async function invoke<T>(method: string, body: Record<string, unknown>): Promise<T> {
  const response = await fetch(`${API_BASE}${procedure}/${method}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "Connect-Protocol-Version": "1" },
    body: JSON.stringify(body),
  });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<T>;
}

export interface ComponentTestResult {
  stage: string;
  assetLibraryId: string;
  version: string;
  subject: string;
  verdict: string;
  message: string;
  remediation: string;
}
export interface ComponentTestArtifact {
  kind: string;
  label: string;
  assetLibraryId: string;
  version: string;
  reference: string;
}
export interface ComponentTestReport {
  id: string;
  rootLibraryId: string;
  rootVersion: string;
  includeClosure: boolean;
  verdict: string;
  results: ComponentTestResult[];
  artifacts?: ComponentTestArtifact[];
  createdAt?: { toDate(): Date };
}

export async function runComponentTest(input: {
  componentId: string;
  version: string;
  includeClosure: boolean;
}): Promise<ComponentTestReport> {
  const response = await invoke<{ report?: ComponentTestReport }>("RunComponentTest", input);
  if (!response.report) throw new Error("The server returned no component test report.");
  return response.report;
}
export async function getComponentTestReport(id: string): Promise<ComponentTestReport> {
  const response = await invoke<{ report?: ComponentTestReport }>("GetComponentTestReport", { id });
  if (!response.report) throw new Error("The server returned no component test report.");
  return response.report;
}
export async function listComponentTestReports(input: {
  componentId: string;
  version?: string;
  limit?: number;
}): Promise<ComponentTestReport[]> {
  const response = await invoke<{ reports?: ComponentTestReport[] }>(
    "ListComponentTestReports",
    input,
  );
  return response.reports ?? [];
}
