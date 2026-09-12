import { buildUrl } from "./client";
import {
  TelemetryInsightsSchema,
  TelemetrySummarySchema,
  TelemetryTailResponseSchema,
  MoveRecordResponseSchema,
  StatusResponseSchema,
  InstallIdResponseSchema,
  OutputPathResponseSchema,
} from "./schemas/misc";
import type {
  ProbeEndpointsResponse,
  ProxyHintsResponse,
  ScenarioPortResponse,
} from "@vrooli/proto-types/scenario-to-desktop/v1/domain/operations_pb";
import type { ManifestResponse } from "@vrooli/proto-types/scenario-to-desktop/v1/domain/preflight_pb";
import type { JsonObject } from "@bufbuild/protobuf";
import {
  desktopRecordsConnectClient,
  documentationConnectClient,
  operationsConnectClient,
  preflightConnectClient,
  systemConnectClient,
  telemetryConnectClient,
} from "./connect";

// ==================== Icon Functions ====================

export const getIconPreviewUrl = (path: string): string =>
  buildUrl(`/icons/preview?path=${encodeURIComponent(path)}`);

// ==================== Docs Functions ====================

export async function fetchDocsManifest() {
  return documentationConnectClient.getDocumentationManifest({});
}

export async function fetchDocContent(path: string) {
  return documentationConnectClient.getDocumentationContent({ path });
}

// ==================== Desktop Record Functions ====================

export async function fetchDesktopRecords() {
  return desktopRecordsConnectClient.listDesktopRecords({});
}

export async function moveDesktopRecord(
  recordId: string,
  payload: {
    target?: "destination" | "custom";
    destination_path?: string;
  } = {},
) {
  const response = await desktopRecordsConnectClient.moveDesktopRecord({
    recordId,
    target: payload.target,
    destinationPath: payload.destination_path,
  });
  return MoveRecordResponseSchema.parse({
    record_id: response.recordId,
    from: response.from,
    to: response.to,
    status: response.status,
  });
}

export function getDownloadUrl(scenarioName: string, platform: string): string {
  return buildUrl(`/desktop/download/${scenarioName}/${platform}`);
}

export async function deleteDesktopBuild(scenarioName: string) {
  const response = await desktopRecordsConnectClient.deleteDesktopScenario({
    scenarioName,
  });
  return StatusResponseSchema.parse({
    status: response.status,
    message: response.message,
  });
}

// ==================== Probe Functions ====================

export function probeEndpoints(payload: {
  proxy_url?: string;
  server_url?: string;
  api_url?: string;
  timeout_ms?: number;
}): Promise<ProbeEndpointsResponse> {
  return operationsConnectClient.probeEndpoints({
    proxyUrl: payload.proxy_url,
    serverUrl: payload.server_url,
    apiUrl: payload.api_url,
    timeoutMs: payload.timeout_ms,
  });
}

export function fetchProxyHints(
  scenarioName: string,
): Promise<ProxyHintsResponse> {
  return operationsConnectClient.getProxyHints({ scenarioName });
}

// ==================== Bundle Manifest Functions ====================

export async function fetchBundleManifest(payload: {
  bundle_manifest_path: string;
}): Promise<ManifestResponse> {
  const response = await preflightConnectClient.inspectManifest({
    manifestPath: payload.bundle_manifest_path,
  });
  if (response.errors.length > 0) {
    throw new Error(response.errors.map((error) => error.message).join("; "));
  }
  return response;
}

// ==================== Wine Functions ====================

export async function checkWineStatus() {
  return systemConnectClient.checkWine({});
}

export async function startWineInstall(method: string) {
  const response = await systemConnectClient.installWine({ method });
  return InstallIdResponseSchema.parse({ install_id: response.installId });
}

export async function fetchWineInstallStatus(installId: string) {
  return systemConnectClient.getWineInstallStatus({
    installId,
  });
}

// ==================== Telemetry Functions ====================

const asJsonObject = (value: unknown): JsonObject =>
  JSON.parse(JSON.stringify(value)) as JsonObject;

const requiredStructPayload = (payload: JsonObject | undefined): JsonObject => {
  if (!payload) throw new Error("Connect service returned an empty payload");
  return payload;
};

export async function fetchTelemetryInsights(scenarioName: string) {
  const response = await telemetryConnectClient.getTelemetryInsights({
    scenarioName,
  });
  return TelemetryInsightsSchema.parse(requiredStructPayload(response.payload));
}

export async function uploadTelemetry(payload: {
  scenario_name: string;
  deployment_mode?: string;
  source?: string;
  events: unknown[];
}) {
  const response = await telemetryConnectClient.ingestTelemetry({
    scenarioName: payload.scenario_name,
    deploymentMode: payload.deployment_mode ?? "external-server",
    source: payload.source ?? "desktop-upload",
    events: payload.events.map(asJsonObject),
  });
  return OutputPathResponseSchema.parse({ output_path: response.outputPath });
}

export async function deleteTelemetry(scenarioName: string) {
  await telemetryConnectClient.deleteTelemetry({ scenarioName });
}

export async function fetchTelemetrySummary(scenarioName: string) {
  const response = await telemetryConnectClient.getTelemetrySummary({
    scenarioName,
  });
  return TelemetrySummarySchema.parse(requiredStructPayload(response.payload));
}

export async function fetchTelemetryTail(scenarioName: string, limit = 200) {
  const response = await telemetryConnectClient.getTelemetryTail({
    scenarioName,
    limit,
  });
  return TelemetryTailResponseSchema.parse(
    requiredStructPayload(response.payload),
  );
}

export const getTelemetryDownloadUrl = (scenarioName: string): string =>
  buildUrl(
    `/deployment/telemetry/${encodeURIComponent(scenarioName)}/download`,
  );

// ==================== Port Functions ====================

export async function fetchScenarioPort(
  scenario: string,
  portName: string,
): Promise<ScenarioPortResponse> {
  return operationsConnectClient.resolveScenarioPort({
    scenarioName: scenario,
    portName,
  });
}
