import { buildUrl, throwIfNotOk } from "./client";
import type {
  HealthResponse,
  DocsManifest,
  DocsContentResponse,
  DesktopRecordResponse,
  TestArtifactSummary,
  TestArtifactCleanupResult,
  ProbeResponse,
  ProxyHintsResponse,
  BundleManifestResponse,
  WineCheckResponse,
  WineInstallStatus,
  TelemetryUploadRequest,
  ScenarioPortResponse,
} from "./types";
import type { TelemetryInsights, TelemetrySummary, TelemetryTailResponse } from "../../domain/types";

// ==================== Icon Functions ====================

export const getIconPreviewUrl = (path: string): string =>
  buildUrl(`/icons/preview?path=${encodeURIComponent(path)}`);

// ==================== Health & System Functions ====================

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(buildUrl("/health"));
  await throwIfNotOk(response);
  return await response.json() as HealthResponse;
}

// ==================== Docs Functions ====================

export async function fetchDocsManifest(): Promise<DocsManifest> {
  const response = await fetch(buildUrl("/docs/manifest"));
  await throwIfNotOk(response);
  return await response.json() as DocsManifest;
}

export async function fetchDocContent(path: string): Promise<DocsContentResponse> {
  const response = await fetch(buildUrl(`/docs/content?path=${encodeURIComponent(path)}`));
  await throwIfNotOk(response);
  return await response.json() as DocsContentResponse;
}

// ==================== Desktop Record Functions ====================

export async function fetchDesktopRecords(): Promise<DesktopRecordResponse> {
  const response = await fetch(buildUrl("/desktop/records"));
  await throwIfNotOk(response);
  return await response.json() as DesktopRecordResponse;
}

export async function moveDesktopRecord(
  recordId: string,
  payload: { target?: "destination" | "custom"; destination_path?: string } = {}
): Promise<{ record_id: string; from: string; to: string; status: string }> {
  const response = await fetch(buildUrl(`/desktop/records/${recordId}/move`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  await throwIfNotOk(response);
  return await response.json() as { record_id: string; from: string; to: string; status: string };
}

export function getDownloadUrl(scenarioName: string, platform: string): string {
  return buildUrl(`/desktop/download/${scenarioName}/${platform}`);
}

export async function deleteDesktopBuild(scenarioName: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/desktop/delete/${scenarioName}`), {
    method: "DELETE"
  });
  await throwIfNotOk(response);
  return await response.json() as { status: string };
}

// ==================== Test Artifact Functions ====================

export async function fetchTestArtifacts(): Promise<TestArtifactSummary> {
  const response = await fetch(buildUrl("/desktop/test-artifacts"));
  await throwIfNotOk(response);
  return await response.json() as TestArtifactSummary;
}

export async function cleanupTestArtifacts(): Promise<TestArtifactCleanupResult> {
  const response = await fetch(buildUrl("/desktop/test-artifacts/cleanup"), {
    method: "POST"
  });
  await throwIfNotOk(response);
  return await response.json() as TestArtifactCleanupResult;
}

// ==================== Probe Functions ====================

export async function probeEndpoints(payload: {
  proxy_url?: string;
  server_url?: string;
  api_url?: string;
  timeout_ms?: number;
}): Promise<ProbeResponse> {
  const response = await fetch(buildUrl("/desktop/probe"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  await throwIfNotOk(response);
  return await response.json() as ProbeResponse;
}

export async function fetchProxyHints(scenarioName: string): Promise<ProxyHintsResponse> {
  const response = await fetch(buildUrl(`/desktop/proxy-hints/${encodeURIComponent(scenarioName)}`));
  await throwIfNotOk(response);
  return await response.json() as ProxyHintsResponse;
}

// ==================== Bundle Manifest Functions ====================

export async function fetchBundleManifest(payload: { bundle_manifest_path: string }): Promise<BundleManifestResponse> {
  const response = await fetch(buildUrl("/desktop/bundle-manifest"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  await throwIfNotOk(response);
  return await response.json() as BundleManifestResponse;
}

// ==================== Wine Functions ====================

export async function checkWineStatus(): Promise<WineCheckResponse> {
  const response = await fetch(buildUrl("/system/wine/check"));
  await throwIfNotOk(response);
  return await response.json() as WineCheckResponse;
}

export async function startWineInstall(method: string): Promise<{ install_id: string }> {
  const response = await fetch(buildUrl("/system/wine/install"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ method })
  });
  await throwIfNotOk(response);
  return await response.json() as { install_id: string };
}

export async function fetchWineInstallStatus(installId: string): Promise<WineInstallStatus> {
  const response = await fetch(buildUrl(`/system/wine/install/status/${installId}`));
  await throwIfNotOk(response);
  return await response.json() as WineInstallStatus;
}

// ==================== Telemetry Functions ====================

export async function fetchTelemetryInsights(scenarioName: string): Promise<TelemetryInsights> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/insights`)
  );
  await throwIfNotOk(response);
  return (await response.json()) as TelemetryInsights;
}

export async function uploadTelemetry(payload: TelemetryUploadRequest): Promise<{ output_path: string }> {
  const response = await fetch(buildUrl("/deployment/telemetry"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      deployment_mode: "external-server",
      source: "desktop-upload",
      ...payload
    })
  });
  await throwIfNotOk(response);
  return await response.json() as { output_path: string };
}

export async function deleteTelemetry(scenarioName: string): Promise<void> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}`),
    { method: "DELETE" }
  );
  await throwIfNotOk(response);
}

export async function fetchTelemetrySummary(scenarioName: string): Promise<TelemetrySummary> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/summary`)
  );
  await throwIfNotOk(response);
  return await response.json() as TelemetrySummary;
}

export async function fetchTelemetryTail(
  scenarioName: string,
  limit = 200
): Promise<TelemetryTailResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/tail?${params.toString()}`)
  );
  await throwIfNotOk(response);
  return await response.json() as TelemetryTailResponse;
}

export const getTelemetryDownloadUrl = (scenarioName: string): string =>
  buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/download`);

// ==================== Port Functions ====================

export async function fetchScenarioPort(scenario: string, portName: string): Promise<ScenarioPortResponse> {
  const response = await fetch(buildUrl(`/ports/${encodeURIComponent(scenario)}/${encodeURIComponent(portName)}`));
  await throwIfNotOk(response);
  return await response.json() as ScenarioPortResponse;
}
