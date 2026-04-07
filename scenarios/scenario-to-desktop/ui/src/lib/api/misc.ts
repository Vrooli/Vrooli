import { buildUrl, fetchJson, mutateJson, mutateVoid, throwIfNotOk } from "./client";
import { parseOrThrow } from "./safeParse";
import {
  DocsManifestSchema,
  DocsContentResponseSchema,
  ProxyHintsResponseSchema,
  BundleManifestResponseSchema,
  WineCheckResponseSchema,
  WineInstallStatusSchema,
  TelemetryInsightsSchema,
  TelemetrySummarySchema,
  TelemetryTailResponseSchema,
  ScenarioPortResponseSchema,
  MoveRecordResponseSchema,
  StatusResponseSchema,
  InstallIdResponseSchema,
  OutputPathResponseSchema,
} from "./schemas/misc";
import { HealthResponseSchema, ProbeResponseSchema, DesktopRecordResponseSchema } from "./schemas";
import type { DesktopRecordResponse, BundleManifestResponse } from "./types";

// ==================== Icon Functions ====================

export const getIconPreviewUrl = (path: string): string =>
  buildUrl(`/icons/preview?path=${encodeURIComponent(path)}`);

// ==================== Health & System Functions ====================

export function fetchHealth() {
  return fetchJson("/health", HealthResponseSchema);
}

// ==================== Docs Functions ====================

export function fetchDocsManifest() {
  return fetchJson("/docs/manifest", DocsManifestSchema);
}

export function fetchDocContent(path: string) {
  return fetchJson(`/docs/content?path=${encodeURIComponent(path)}`, DocsContentResponseSchema);
}

// ==================== Desktop Record Functions ====================

export async function fetchDesktopRecords(): Promise<DesktopRecordResponse> {
  const response = await fetch(buildUrl("/desktop/records"));
  await throwIfNotOk(response);
  // Schema validates shape; explicit return type bridges Zod's widened
  // union inference (e.g. `string` from z.union([enum, z.string()])) back
  // to the narrower hand-written interface.
  return parseOrThrow(DesktopRecordResponseSchema, await response.json()) as DesktopRecordResponse;
}

export function moveDesktopRecord(
  recordId: string,
  payload: { target?: "destination" | "custom"; destination_path?: string } = {}
) {
  return mutateJson(`/desktop/records/${recordId}/move`, MoveRecordResponseSchema, {
    method: "POST",
    body: payload,
  });
}

export function getDownloadUrl(scenarioName: string, platform: string): string {
  return buildUrl(`/desktop/download/${scenarioName}/${platform}`);
}

export function deleteDesktopBuild(scenarioName: string) {
  return mutateJson(`/desktop/delete/${scenarioName}`, StatusResponseSchema, {
    method: "DELETE",
  });
}

// ==================== Probe Functions ====================

export function probeEndpoints(payload: {
  proxy_url?: string;
  server_url?: string;
  api_url?: string;
  timeout_ms?: number;
}) {
  return mutateJson("/desktop/probe", ProbeResponseSchema, {
    method: "POST",
    body: payload,
  });
}

export function fetchProxyHints(scenarioName: string) {
  return fetchJson(
    `/desktop/proxy-hints/${encodeURIComponent(scenarioName)}`,
    ProxyHintsResponseSchema,
  );
}

// ==================== Bundle Manifest Functions ====================

export async function fetchBundleManifest(
  payload: { bundle_manifest_path: string },
): Promise<BundleManifestResponse> {
  const response = await fetch(buildUrl("/desktop/bundle-manifest"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  await throwIfNotOk(response);
  return parseOrThrow(BundleManifestResponseSchema, await response.json()) as BundleManifestResponse;
}

// ==================== Wine Functions ====================

export function checkWineStatus() {
  return fetchJson("/system/wine/check", WineCheckResponseSchema);
}

export function startWineInstall(method: string) {
  return mutateJson("/system/wine/install", InstallIdResponseSchema, {
    method: "POST",
    body: { method },
  });
}

export function fetchWineInstallStatus(installId: string) {
  return fetchJson(`/system/wine/install/status/${installId}`, WineInstallStatusSchema);
}

// ==================== Telemetry Functions ====================

export function fetchTelemetryInsights(scenarioName: string) {
  return fetchJson(
    `/deployment/telemetry/${encodeURIComponent(scenarioName)}/insights`,
    TelemetryInsightsSchema,
  );
}

export function uploadTelemetry(payload: {
  scenario_name: string;
  deployment_mode?: string;
  source?: string;
  events: unknown[];
}) {
  return mutateJson("/deployment/telemetry", OutputPathResponseSchema, {
    method: "POST",
    body: {
      deployment_mode: "external-server",
      source: "desktop-upload",
      ...payload,
    },
  });
}

export function deleteTelemetry(scenarioName: string) {
  return mutateVoid(
    `/deployment/telemetry/${encodeURIComponent(scenarioName)}`,
    { method: "DELETE" },
  );
}

export function fetchTelemetrySummary(scenarioName: string) {
  return fetchJson(
    `/deployment/telemetry/${encodeURIComponent(scenarioName)}/summary`,
    TelemetrySummarySchema,
  );
}

export function fetchTelemetryTail(scenarioName: string, limit = 200) {
  const params = new URLSearchParams({ limit: String(limit) });
  return fetchJson(
    `/deployment/telemetry/${encodeURIComponent(scenarioName)}/tail?${params.toString()}`,
    TelemetryTailResponseSchema,
  );
}

export const getTelemetryDownloadUrl = (scenarioName: string): string =>
  buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/download`);

// ==================== Port Functions ====================

export function fetchScenarioPort(scenario: string, portName: string) {
  return fetchJson(
    `/ports/${encodeURIComponent(scenario)}/${encodeURIComponent(portName)}`,
    ScenarioPortResponseSchema,
  );
}
