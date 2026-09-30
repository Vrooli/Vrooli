import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export type ProviderStatus = { name: string; state: string; reason: string };
export type HealthFinding = { code: string; severity: string; count: number; message: string; action: string };
export type DiagnosticsReport = {
  workspaceId: string;
  generatedAt: string;
  schemaVersion: number;
  databaseOk: boolean;
  providers: ProviderStatus[];
  findings: HealthFinding[];
};

export async function fetchDiagnostics(workspaceId: string): Promise<DiagnosticsReport> {
  const query = encodeURIComponent(workspaceId);
  const res = await fetch(buildApiUrl(`/diagnostics?workspace_id=${query}`, { baseUrl: API_BASE }), { method: "GET", cache: "no-store" });
  if (!res.ok) throw await decodeApiError(res);
  return (await res.json()) as DiagnosticsReport;
}
