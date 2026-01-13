import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import type { ScenariosResponse } from "../components/scenario-inventory/types";
import type { TelemetryInsights, TelemetrySummary, TelemetryTailResponse } from "../domain/types";
import type {
  AgentManagerStatus,
  CreateTaskRequest,
  CreateTaskResponse,
  GetTaskResponse,
  Investigation,
  InvestigationSummary,
  ListTasksResponse,
  StopTaskResponse,
} from "../types/investigation";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });
export const getIconPreviewUrl = (path: string): string =>
  buildUrl(`/icons/preview?path=${encodeURIComponent(path)}`);

export async function fetchTelemetryInsights(scenarioName: string): Promise<TelemetryInsights> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/insights`)
  );
  if (!response.ok) {
    throw new Error(`Failed to fetch telemetry insights (${response.status})`);
  }
  return (await response.json()) as TelemetryInsights;
}

export interface DocsDocument {
  path: string;
  title: string;
  description?: string;
}

export interface DocsSection {
  id: string;
  title: string;
  icon?: string;
  description?: string;
  documents: DocsDocument[];
}

export interface DocsNavigation {
  primary?: string[];
  secondary?: string[];
}

export interface DocsManifest {
  version: string;
  title: string;
  description?: string;
  defaultDocument: string;
  sections: DocsSection[];
  navigation?: DocsNavigation;
}

export interface DocsContentResponse {
  path: string;
  content: string;
}

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

export interface SystemStatus {
  service: string;
  version: string;
  uptime: number;
  statistics: {
    total_builds: number;
    active_builds: number;
    completed_builds: number;
    failed_builds: number;
  };
}

export interface TemplateInfo {
  name: string;
  description: string;
  type: string;
  framework: string;
  use_cases: string[];
  features: string[];
  complexity: string;
}

export interface DesktopConfig {
  app_name: string;
  app_display_name: string;
  app_description: string;
  version: string;
  author: string;
  license: string;
  app_id: string;
  server_type: string;
  server_port: number;
  server_path: string;
  api_endpoint: string;
  framework: string;
  template_type: string;
  platforms: string[];
  output_path: string;
  location_mode?: "proper" | "temp" | "custom";
  icon?: string;
  features: Record<string, boolean>;
  window: Record<string, unknown>;
  deployment_mode?: string;
  auto_manage_vrooli?: boolean;
  vrooli_binary_path?: string;
  proxy_url?: string;
  external_server_url?: string;
  external_api_url?: string;
  bundle_manifest_path?: string;
  code_signing?: SigningConfig;
}

export interface BundleValidationError {
  code: string;
  service?: string;
  path?: string;
  message: string;
}

export interface BundleValidationWarning {
  code: string;
  service?: string;
  path?: string;
  message: string;
}

export interface BundleMissingBinary {
  service_id: string;
  platform: string;
  path: string;
}

export interface BundleMissingAsset {
  service_id: string;
  path: string;
}

export interface BundleInvalidChecksum {
  service_id: string;
  path: string;
  expected: string;
  actual: string;
}

export interface BundleValidationResult {
  valid: boolean;
  errors?: BundleValidationError[];
  warnings?: BundleValidationWarning[];
  missing_binaries?: BundleMissingBinary[];
  missing_assets?: BundleMissingAsset[];
  invalid_checksums?: BundleInvalidChecksum[];
}

export interface BundlePreflightRequest {
  bundle_manifest_path: string;
  bundle_root?: string;
  secrets?: Record<string, string>;
  timeout_seconds?: number;
  start_services?: boolean;
  log_tail_lines?: number;
  log_tail_services?: string[];
  status_only?: boolean;
  session_id?: string;
  session_ttl_seconds?: number;
  session_stop?: boolean;
}

export interface BundlePreflightSecret {
  id: string;
  class: string;
  required: boolean;
  has_value: boolean;
  description?: string;
  format?: string;
  prompt?: Record<string, string>;
}

export interface BundlePreflightReady {
  ready: boolean;
  details: Record<string, {
    ready: boolean;
    skipped?: boolean;
    message?: string;
    exit_code?: number;
    started_at?: string;
    ready_at?: string;
    updated_at?: string;
  }>;
  gpu?: {
    available: boolean;
    method?: string;
    reason?: string;
    requirements?: Record<string, string>;
  };
  snapshot_at?: string;
  waited_seconds?: number;
}

export interface BundlePreflightTelemetry {
  path: string;
  upload_url?: string;
}

export interface BundlePreflightRuntime {
  instance_id?: string;
  started_at?: string;
  app_data_dir?: string;
  bundle_root?: string;
  dry_run?: boolean;
  manifest_hash?: string;
  manifest_schema?: string;
  target?: string;
  app_name?: string;
  app_version?: string;
  ipc_host?: string;
  ipc_port?: number;
  runtime_version?: string;
  build_version?: string;
}

export interface BundlePreflightServiceFingerprint {
  service_id: string;
  platform?: string;
  binary_path?: string;
  binary_resolved_path?: string;
  binary_sha256?: string;
  binary_size_bytes?: number;
  binary_mtime?: string;
  error?: string;
}

export interface BundlePreflightLogTail {
  service_id: string;
  lines: number;
  content?: string;
  error?: string;
}

export interface BundlePreflightCheck {
  id: string;
  step: string;
  name: string;
  status: "pass" | "fail" | "warning" | "skipped";
  detail?: string;
}

export interface BundlePreflightResponse {
  status: string;
  validation?: BundleValidationResult;
  ready?: BundlePreflightReady;
  secrets?: BundlePreflightSecret[];
  ports?: Record<string, Record<string, number>>;
  telemetry?: BundlePreflightTelemetry;
  log_tails?: BundlePreflightLogTail[];
  checks?: BundlePreflightCheck[];
  runtime?: BundlePreflightRuntime;
  service_fingerprints?: BundlePreflightServiceFingerprint[];
  errors?: string[];
  session_id?: string;
  expires_at?: string;
}

export interface BundlePreflightStep {
  id: string;
  name: string;
  state: "pending" | "running" | "pass" | "fail" | "warning" | "skipped";
  detail?: string;
}

export interface BundlePreflightJobStartResponse {
  job_id: string;
}

export interface BundlePreflightJobStatusResponse {
  job_id: string;
  status: "running" | "completed" | "failed";
  steps?: BundlePreflightStep[];
  result?: BundlePreflightResponse;
  error?: string;
  started_at?: string;
  updated_at?: string;
}

export interface BundlePreflightHealthResponse {
  service_id: string;
  supported?: boolean;
  health_type?: string;
  message?: string;
  url?: string;
  status_code?: number;
  status?: string;
  body?: string;
  content_type?: string;
  truncated?: boolean;
  fetched_at?: string;
  elapsed_ms?: number;
}

export interface BundleManifestResponse {
  path: string;
  manifest: unknown;
}

export interface PlatformBuildResult {
  platform: string;
  status: "pending" | "building" | "ready" | "failed" | "skipped";
  started_at?: string;
  completed_at?: string;
  error_log?: string[];
  artifact?: string;
  file_size?: number;
  skip_reason?: string;
}

export interface BuildStatus {
  build_id: string;
  scenario_name: string;
  status: "building" | "ready" | "partial" | "failed";
  framework: string;
  template_type: string;
  platforms: string[]; // Legacy: platforms that were built
  requested_platforms?: string[]; // NEW: platforms that were requested to build
  platform_results?: Record<string, PlatformBuildResult>;
  output_path: string;
  created_at: string;
  completed_at?: string;
  error_log?: string[];
  build_log?: string[];
  artifacts?: Record<string, string>;
}

export interface SmokeTestStatus {
  smoke_test_id: string;
  scenario_name: string;
  platform: "win" | "mac" | "linux";
  status: "running" | "passed" | "failed";
  artifact_path?: string;
  started_at: string;
  completed_at?: string;
  logs?: string[];
  error?: string;
  telemetry_uploaded?: boolean;
  telemetry_upload_error?: string;
}

export interface ProbeResponse {
  server: {
    status: "ok" | "error" | "skipped";
    status_code?: number;
    message?: string;
  };
  api: {
    status: "ok" | "error" | "skipped";
    status_code?: number;
    message?: string;
  };
}

export interface ProxyHintsResponse {
  scenario: string;
  hints: Array<{
    url: string;
    source: string;
    confidence: string;
    message: string;
  }>;
}

export interface DesktopRecord {
  id: string;
  build_id: string;
  scenario_name: string;
  app_display_name?: string;
  template_type?: string;
  framework?: string;
  location_mode?: "proper" | "temp" | "custom" | string;
  output_path: string;
  destination_path?: string;
  staging_path?: string;
  custom_path?: string;
  deployment_mode?: string;
  icon?: string;
  created_at?: string;
  updated_at?: string;
}

export interface DesktopRecordResponse {
  records: Array<{
    record: DesktopRecord;
    build_status?: BuildStatus;
    has_build: boolean;
    build_state?: string;
  }>;
}

export interface TestArtifactSummary {
  count: number;
  total_bytes: number;
  paths?: string[];
}

export interface TestArtifactCleanupResult {
  removed_count: number;
  freed_bytes: number;
}

export interface WineInstallMethod {
  id: string;
  name: string;
  description: string;
  requires_sudo: boolean;
  steps: string[];
  estimated_time: string;
}

export interface WineCheckResponse {
  installed: boolean;
  version?: string;
  platform: string;
  required_for: string[];
  install_methods?: WineInstallMethod[];
  recommended_method?: string;
}

export interface WineInstallStatus {
  install_id: string;
  status: string;
  method: string;
  started_at: string;
  completed_at?: string;
  log: string[];
  error_log: string[];
}

export interface QuickGenerateRequest {
  scenario_name: string;
  template_type: string;
  deployment_mode?: string;
  proxy_url?: string;
  bundle_manifest_path?: string;
  auto_manage_vrooli?: boolean;
  vrooli_binary_path?: string;
}

export interface QuickGenerateResponse {
  build_id: string;
  status: string;
  output_path?: string;
  desktop_path?: string;
}

export interface TelemetryUploadRequest {
  scenario_name: string;
  deployment_mode?: string;
  source?: string;
  events: unknown[];
}

export async function fetchScenarioDesktopStatus(): Promise<ScenariosResponse> {
  const response = await fetch(buildUrl("/scenarios/desktop-status"));
  if (!response.ok) {
    throw new Error("Failed to fetch scenarios");
  }
  return response.json();
}

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(buildUrl("/health"));
  if (!response.ok) {
    throw new Error(`Health check failed: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchSystemStatus(): Promise<SystemStatus> {
  const response = await fetch(buildUrl("/status"));
  if (!response.ok) {
    throw new Error(`Failed to fetch system status: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchTemplates(): Promise<{ templates: TemplateInfo[] }> {
  const response = await fetch(buildUrl("/templates"));
  if (!response.ok) {
    throw new Error(`Failed to fetch templates: ${response.statusText}`);
  }
  return response.json();
}

export async function generateDesktop(config: DesktopConfig): Promise<{
  build_id: string;
  status: string;
  desktop_path: string;
  install_instructions: string;
}> {
  const response = await fetch(buildUrl("/desktop/generate"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify(config)
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || "Failed to generate desktop application");
  }

  return response.json();
}

export async function fetchBuildStatus(buildId: string): Promise<BuildStatus> {
  const response = await fetch(buildUrl(`/desktop/status/${buildId}`));
  if (!response.ok) {
    throw new Error(`Failed to fetch build status: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchDesktopRecords(): Promise<DesktopRecordResponse> {
  const response = await fetch(buildUrl("/desktop/records"));
  if (!response.ok) {
    throw new Error(`Failed to fetch desktop records: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchTestArtifacts(): Promise<TestArtifactSummary> {
  const response = await fetch(buildUrl("/desktop/test-artifacts"));
  if (!response.ok) {
    throw new Error(`Failed to fetch test artifact summary: ${response.statusText}`);
  }
  return response.json();
}

export async function cleanupTestArtifacts(): Promise<TestArtifactCleanupResult> {
  const response = await fetch(buildUrl("/desktop/test-artifacts/cleanup"), {
    method: "POST"
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.message || response.statusText);
  }
  return response.json();
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
  if (!response.ok) {
    const error = await response.json().catch(() => ({}));
    throw new Error(error.error || error.message || response.statusText);
  }
  return response.json();
}

export async function buildScenarioDesktop(
  scenarioName: string,
  platforms?: string[]
): Promise<{
  build_id: string;
  status: string;
  scenario: string;
  desktop_path: string;
  platforms: string[];
  status_url: string;
}> {
  const response = await fetch(buildUrl(`/desktop/build/${scenarioName}`), {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({
      platforms: platforms || ["win", "mac", "linux"]
    })
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || "Failed to build desktop application");
  }

  return response.json();
}

export function getDownloadUrl(scenarioName: string, platform: string): string {
  return buildUrl(`/desktop/download/${scenarioName}/${platform}`);
}

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

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || "Failed to probe URLs");
  }

  return response.json();
}

export async function runBundlePreflight(payload: BundlePreflightRequest): Promise<BundlePreflightResponse> {
  const response = await fetch(buildUrl("/desktop/preflight"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return response.json();
}

export async function startBundlePreflight(payload: BundlePreflightRequest): Promise<BundlePreflightJobStartResponse> {
  const response = await fetch(buildUrl("/desktop/preflight/start"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return response.json();
}

export async function fetchBundlePreflightStatus(payload: { job_id: string }): Promise<BundlePreflightJobStatusResponse> {
  const params = new URLSearchParams({ job_id: payload.job_id });
  const response = await fetch(buildUrl(`/desktop/preflight/status?${params.toString()}`));

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return response.json();
}

export async function fetchBundlePreflightHealth(payload: {
  session_id: string;
  service_id: string;
}): Promise<BundlePreflightHealthResponse> {
  const params = new URLSearchParams({
    session_id: payload.session_id,
    service_id: payload.service_id
  });
  const response = await fetch(buildUrl(`/desktop/preflight/health?${params.toString()}`));

  if (!response.ok) {
    const contentType = response.headers.get("content-type");
    if (contentType?.includes("application/json")) {
      const errorData = await response.json().catch(() => null);
      const errorMessage = errorData?.error || errorData?.message || response.statusText;
      throw new Error(errorMessage);
    }
    throw new Error(await response.text());
  }

  return response.json();
}

export async function fetchBundleManifest(payload: { bundle_manifest_path: string }): Promise<BundleManifestResponse> {
  const response = await fetch(buildUrl("/desktop/bundle-manifest"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    throw new Error(await response.text());
  }

  return response.json();
}

export async function fetchProxyHints(scenarioName: string): Promise<ProxyHintsResponse> {
  const response = await fetch(buildUrl(`/desktop/proxy-hints/${encodeURIComponent(scenarioName)}`));
  if (!response.ok) {
    throw new Error("Failed to load proxy hints");
  }
  return response.json();
}

export async function quickGenerateDesktop(payload: QuickGenerateRequest): Promise<QuickGenerateResponse> {
  const response = await fetch(buildUrl("/desktop/generate/quick"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });

  if (!response.ok) {
    const contentType = response.headers.get("content-type");
    let errorMessage = "Failed to generate desktop app";

    if (contentType?.includes("application/json")) {
      const errorData = await response.json().catch(() => null);
      errorMessage = errorData?.message || errorData?.error || errorMessage;
    } else {
      const textError = await response.text().catch(() => null);
      errorMessage = textError || errorMessage;
    }

    if (response.status === 404) {
      throw new Error("Scenario not found. Check that it exists in the scenarios directory.");
    } else if (response.status === 400) {
      throw new Error(`Invalid request: ${errorMessage}`);
    } else if (response.status === 500) {
      throw new Error(`Server error: ${errorMessage}. Check API logs for details.`);
    }
    throw new Error(`${errorMessage} (HTTP ${response.status})`);
  }

  return response.json();
}

export async function checkWineStatus(): Promise<WineCheckResponse> {
  const response = await fetch(buildUrl("/system/wine/check"));
  if (!response.ok) {
    throw new Error("Failed to check Wine status");
  }
  return response.json();
}

export async function startWineInstall(method: string): Promise<{ install_id: string }> {
  const response = await fetch(buildUrl("/system/wine/install"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ method })
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || "Failed to start installation");
  }
  return response.json();
}

export async function fetchWineInstallStatus(installId: string): Promise<WineInstallStatus> {
  const response = await fetch(buildUrl(`/system/wine/install/status/${installId}`));
  if (!response.ok) {
    throw new Error("Failed to fetch install status");
  }
  return response.json();
}

export async function deleteDesktopBuild(scenarioName: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/desktop/delete/${scenarioName}`), {
    method: "DELETE"
  });
  if (!response.ok) {
    const error = await response.text().catch(() => response.statusText);
    throw new Error(error || "Failed to delete desktop app");
  }
  return response.json();
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

  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText);
    throw new Error(text || "Failed to upload telemetry");
  }

  return response.json();
}

export async function deleteTelemetry(scenarioName: string): Promise<void> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}`),
    { method: "DELETE" }
  );
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to delete telemetry");
  }
}

export async function fetchTelemetrySummary(scenarioName: string): Promise<TelemetrySummary> {
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/summary`)
  );
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to fetch telemetry summary");
  }
  return response.json();
}

export async function fetchTelemetryTail(
  scenarioName: string,
  limit = 200
): Promise<TelemetryTailResponse> {
  const params = new URLSearchParams({ limit: String(limit) });
  const response = await fetch(
    buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/tail?${params.toString()}`)
  );
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to fetch telemetry tail");
  }
  return response.json();
}

export const getTelemetryDownloadUrl = (scenarioName: string): string =>
  buildUrl(`/deployment/telemetry/${encodeURIComponent(scenarioName)}/download`);

export async function startSmokeTest(payload: {
  scenario_name: string;
  platform?: string;
}): Promise<SmokeTestStatus> {
  const response = await fetch(buildUrl("/desktop/smoke-test/start"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload)
  });
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to start smoke test");
  }
  return response.json();
}

export async function fetchSmokeTestStatus(smokeTestId: string): Promise<SmokeTestStatus> {
  const response = await fetch(buildUrl(`/desktop/smoke-test/status/${smokeTestId}`));
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to fetch smoke test status");
  }
  return response.json();
}

export async function cancelSmokeTest(smokeTestId: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/desktop/smoke-test/cancel/${smokeTestId}`), {
    method: "POST"
  });
  if (!response.ok) {
    const message = await response.text().catch(() => "");
    throw new Error(message || "Failed to cancel smoke test");
  }
  return response.json();
}

export async function fetchDocsManifest(): Promise<DocsManifest> {
  const response = await fetch(buildUrl("/docs/manifest"));
  if (!response.ok) {
    throw new Error("Failed to load docs manifest");
  }
  return response.json();
}

export async function fetchDocContent(path: string): Promise<DocsContentResponse> {
  const response = await fetch(buildUrl(`/docs/content?path=${encodeURIComponent(path)}`));
  if (!response.ok) {
    throw new Error("Failed to load document");
  }
  return response.json();
}

export interface BundleExportResponse {
  status: string;
  schema: string;
  manifest: unknown;
  checksum?: string;
  generated_at?: string;
  deployment_manager_url?: string;
  manifest_path?: string;
}

export interface AutoBuildPlatformStatus {
  name: string;
  status: string;
  output_path?: string;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface AutoBuildTargetStatus {
  id: string;
  folder: string;
  platforms: AutoBuildPlatformStatus[];
}

export interface AutoBuildStatusResponse {
  build_id: string;
  scenario: string;
  status: string;
  message?: string;
  created_at?: string;
  started_at?: string;
  completed_at?: string;
  targets?: AutoBuildTargetStatus[];
  build_log?: string[];
  error_log?: string[];
  status_url?: string;
  check_command?: string;
  poll_after_ms?: number;
}

export async function startDeploymentManagerAutoBuild(opts: {
  scenario: string;
  platforms?: string[];
  targets?: string[];
  dryRun?: boolean;
}): Promise<AutoBuildStatusResponse> {
  const response = await fetch(buildUrl("/deployment-manager/build/auto"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      scenario: opts.scenario,
      platforms: opts.platforms,
      targets: opts.targets,
      dry_run: opts.dryRun
    })
  });
  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      detail = body.details || body.error || body.message || detail;
    } catch {
      // ignore parse errors
    }
    throw new Error(`deployment-manager build failed: ${detail}`);
  }
  return response.json();
}

export async function fetchDeploymentManagerAutoBuildStatus(buildId: string): Promise<AutoBuildStatusResponse> {
  const response = await fetch(buildUrl(`/deployment-manager/build/auto/${encodeURIComponent(buildId)}`));
  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      detail = body.details || body.error || body.message || detail;
    } catch {
      // ignore parse errors
    }
    throw new Error(`deployment-manager build status failed: ${detail}`);
  }
  return response.json();
}

export async function exportBundleFromDeploymentManager(opts: {
  scenario: string;
  tier?: string;
  includeSecrets?: boolean;
}): Promise<BundleExportResponse> {
  const tier = opts.tier || "tier-2-desktop";
  const includeSecrets = opts.includeSecrets ?? true;
  const response = await fetch(buildUrl("/deployment-manager/bundles/export"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      scenario: opts.scenario,
      tier,
      include_secrets: includeSecrets
    })
  });
  if (!response.ok) {
    let detail = `${response.status} ${response.statusText}`;
    try {
      const body = await response.json();
      detail = body.details || body.error || body.message || detail;
    } catch {
      // ignore parse errors
    }
    throw new Error(`deployment-manager export failed: ${detail}`);
  }
  return response.json();
}

export interface ScenarioPortResponse {
  scenario: string;
  port_name: string;
  host: string;
  port: number;
  url: string;
}

export async function fetchScenarioPort(scenario: string, portName: string): Promise<ScenarioPortResponse> {
  const response = await fetch(buildUrl(`/ports/${encodeURIComponent(scenario)}/${encodeURIComponent(portName)}`));
  if (!response.ok) {
    const text = await response.text().catch(() => response.statusText);
    throw new Error(text || "Failed to resolve scenario port");
  }
  return response.json();
}

// ==================== Code Signing API ====================

export interface WindowsSigningConfig {
  certificate_source: "file" | "store" | "azure_keyvault" | "aws_kms";
  certificate_file?: string;
  certificate_password_env?: string;
  certificate_thumbprint?: string;
  timestamp_server?: string;
  sign_algorithm?: "sha256" | "sha384" | "sha512";
  dual_sign?: boolean;
}

export interface MacOSSigningConfig {
  identity: string;
  team_id: string;
  hardened_runtime: boolean;
  notarize: boolean;
  entitlements_file?: string;
  provisioning_profile?: string;
  gatekeeper_assess?: boolean;
  apple_id_env?: string;
  apple_id_password_env?: string;
  apple_api_key_id?: string;
  apple_api_key_file?: string;
  apple_api_issuer_id?: string;
}

export interface LinuxSigningConfig {
  gpg_key_id?: string;
  gpg_passphrase_env?: string;
  gpg_homedir?: string;
  keyring_path?: string;
  deb_keyring_path?: string;
  rpm_keyring_path?: string;
}

export interface SigningConfig {
  schema_version?: string;
  enabled: boolean;
  windows?: WindowsSigningConfig;
  macos?: MacOSSigningConfig;
  linux?: LinuxSigningConfig;
}

export interface SigningConfigResponse {
  scenario: string;
  config: SigningConfig | null;
  config_path: string;
}

export interface ValidationError {
  code: string;
  platform?: string;
  field?: string;
  message: string;
  remediation?: string;
}

export interface ValidationWarning {
  code: string;
  platform?: string;
  message: string;
}

export interface PlatformValidation {
  configured: boolean;
  tool_installed?: boolean;
  tool_path?: string;
  tool_version?: string;
  errors: string[];
  warnings: string[];
}

export interface ValidationResult {
  valid: boolean;
  platforms: Record<string, PlatformValidation>;
  errors: ValidationError[];
  warnings: ValidationWarning[];
}

export interface PlatformStatus {
  ready: boolean;
  reason?: string;
}

export interface SigningReadinessResponse {
  ready: boolean;
  scenario: string;
  issues?: string[];
  platforms: Record<string, PlatformStatus>;
}

export interface ToolDetectionResult {
  platform: string;
  tool: string;
  installed: boolean;
  path?: string;
  version?: string;
  error?: string;
  remediation?: string;
}

export interface DiscoveredCertificate {
  id: string;
  name: string;
  subject?: string;
  issuer?: string;
  expires_at?: string;
  days_to_expiry: number;
  is_expired: boolean;
  is_code_sign: boolean;
  type?: string;
  platform: string;
  usage_hint?: string;
}

export interface GenerateKeyResponse {
  status: string;
  key_id: string;
  fingerprint: string;
  homedir: string;
  public_key?: string;
  config_path?: string;
  public_key_path?: string;
}

export async function fetchSigningConfig(scenario: string): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`));
  if (!response.ok) {
    throw new Error(`Failed to fetch signing config: ${response.statusText}`);
  }
  return response.json();
}

export async function saveSigningConfig(scenario: string, config: SigningConfig): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to save signing config");
  }
  return response.json();
}

export async function updatePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux",
  config: WindowsSigningConfig | MacOSSigningConfig | LinuxSigningConfig
): Promise<SigningConfigResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/${platform}`), {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to update platform signing config");
  }
  return response.json();
}

export async function deleteSigningConfig(scenario: string): Promise<{ status: string; scenario: string }> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}`), {
    method: "DELETE"
  });
  if (!response.ok) {
    throw new Error(`Failed to delete signing config: ${response.statusText}`);
  }
  return response.json();
}

export async function deletePlatformSigningConfig(
  scenario: string,
  platform: "windows" | "macos" | "linux"
): Promise<{ status: string; scenario: string; platform: string }> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/${platform}`), {
    method: "DELETE"
  });
  if (!response.ok) {
    throw new Error(`Failed to delete platform signing config: ${response.statusText}`);
  }
  return response.json();
}

export async function validateSigningConfig(scenario: string): Promise<ValidationResult> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/validate`), {
    method: "POST"
  });
  if (!response.ok) {
    throw new Error(`Failed to validate signing config: ${response.statusText}`);
  }
  return response.json();
}

export async function checkSigningReadiness(scenario: string): Promise<SigningReadinessResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/ready`));
  if (!response.ok) {
    throw new Error(`Failed to check signing readiness: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchSigningPrerequisites(): Promise<{ tools: ToolDetectionResult[] }> {
  const response = await fetch(buildUrl("/signing/prerequisites"));
  if (!response.ok) {
    throw new Error(`Failed to fetch signing prerequisites: ${response.statusText}`);
  }
  return response.json();
}

export async function discoverCertificates(platform: "windows" | "macos" | "linux"): Promise<{
  platform: string;
  certificates: DiscoveredCertificate[];
}> {
  const response = await fetch(buildUrl(`/signing/discover/${platform}`));
  if (!response.ok) {
    throw new Error(`Failed to discover certificates: ${response.statusText}`);
  }
  return response.json();
}

export async function generateLinuxSigningKey(
  scenario: string,
  payload: {
    name?: string;
    email?: string;
    passphrase?: string;
    passphrase_env?: string;
    homedir?: string;
    expiry?: string;
    force?: boolean;
  }
): Promise<GenerateKeyResponse> {
  const response = await fetch(buildUrl(`/signing/${encodeURIComponent(scenario)}/linux/generate-key`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ...payload, export_public: true })
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(data.error || data.message || `Failed to generate key: ${response.statusText}`);
  }
  return data;
}

// ==================== Task / Investigation API ====================

export async function getAgentManagerStatus(): Promise<AgentManagerStatus> {
  const response = await fetch(buildUrl("/agent-manager/status"));
  if (!response.ok) {
    throw new Error(`Failed to fetch agent manager status: ${response.statusText}`);
  }
  return response.json();
}

export async function createTask(
  pipelineId: string,
  request: CreateTaskRequest
): Promise<Investigation> {
  const response = await fetch(buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || error.error || "Failed to create task");
  }
  const data: CreateTaskResponse = await response.json();
  return data.task;
}

export async function listTasks(
  pipelineId: string,
  limit?: number
): Promise<InvestigationSummary[]> {
  const params = new URLSearchParams();
  if (limit) {
    params.set("limit", String(limit));
  }
  const url = buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks`) +
    (params.toString() ? `?${params.toString()}` : "");
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`Failed to list tasks: ${response.statusText}`);
  }
  const data: ListTasksResponse = await response.json();
  return data.tasks;
}

export async function getTask(
  pipelineId: string,
  taskId: string
): Promise<Investigation> {
  const response = await fetch(
    buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks/${encodeURIComponent(taskId)}`)
  );
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error("Task not found");
    }
    throw new Error(`Failed to fetch task: ${response.statusText}`);
  }
  const data: GetTaskResponse = await response.json();
  return data.task;
}

export async function stopTask(
  pipelineId: string,
  taskId: string
): Promise<void> {
  const response = await fetch(
    buildUrl(`/pipeline/${encodeURIComponent(pipelineId)}/tasks/${encodeURIComponent(taskId)}/stop`),
    { method: "POST" }
  );
  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new Error(error.message || error.error || "Failed to stop task");
  }
  const data: StopTaskResponse = await response.json();
  if (!data.success) {
    throw new Error(data.message || "Failed to stop task");
  }
}

// ========================
// Scenario State API
// ========================

export interface PlatformSelection {
  win?: boolean;
  mac?: boolean;
  linux?: boolean;
}

export interface FormState {
  selected_template?: string;
  framework?: string;
  app_display_name?: string;
  app_description?: string;
  icon_path?: string;
  display_name_edited?: boolean;
  description_edited?: boolean;
  icon_path_edited?: boolean;
  server_type?: string;
  deployment_mode?: string;
  proxy_url?: string;
  server_port?: number;
  local_server_path?: string;
  local_api_endpoint?: string;
  auto_manage_tier1?: boolean;
  vrooli_binary_path?: string;
  bundle_manifest_path?: string;
  platforms?: PlatformSelection;
  location_mode?: string;
  output_path?: string;
  connection_result?: unknown;
  connection_error?: string | null;
  preflight_result?: BundlePreflightResponse | null;
  preflight_error?: string | null;
  preflight_override?: boolean;
  preflight_secrets?: Record<string, string>;
  preflight_start_services?: boolean;
  preflight_auto_refresh?: boolean;
  preflight_session_id?: string | null;
  preflight_session_expires_at?: string | null;
  preflight_session_ttl?: number;
  deployment_manager_url?: string | null;
  signing_enabled_for_build?: boolean;
}

export interface InputFingerprint {
  manifest_path?: string;
  manifest_hash?: string;
  manifest_mtime?: number;
  preflight_secret_keys?: string[];
  preflight_timeout?: number;
  start_services?: boolean;
  template_type?: string;
  framework?: string;
  deployment_mode?: string;
  app_display_name?: string;
  app_description?: string;
  icon_path?: string;
  platforms?: string[];
  signing_enabled?: boolean;
  signing_config_hash?: string;
  output_location?: string;
  smoke_test_platform?: string;
}

export interface StageState {
  stage: string;
  status: "valid" | "stale" | "invalid" | "none";
  input_fingerprint?: InputFingerprint;
  output_hash?: string;
  validated_at?: string;
  result?: unknown;
  staleness_reason?: string;
}

export interface CompressedLog {
  service_id: string;
  compressed_data: string;
  original_lines: number;
  compressed_size: number;
  captured_at: string;
}

export interface BuildArtifact {
  platform: string;
  status: "pending" | "building" | "ready" | "failed";
  file_path?: string;
  file_name?: string;
  file_size?: number;
  build_id?: string;
  built_at?: string;
  error_message?: string;
}

export interface ScenarioState {
  scenario_name: string;
  schema_version: number;
  created_at: string;
  updated_at: string;
  hash?: string;
  form_state: FormState;
  stages?: Record<string, StageState>;
  compressed_logs?: CompressedLog[];
  build_artifacts?: BuildArtifact[];
}

export interface StateChange {
  change_type: string;
  affected_stage: string;
  reason: string;
  old_value?: string;
  new_value?: string;
}

export interface StageStatus {
  stage: string;
  status: "valid" | "stale" | "invalid" | "none";
  last_run?: string;
  staleness_reason?: string;
  can_reuse: boolean;
}

export interface ValidationStatus {
  scenario_name: string;
  overall_status: "valid" | "partial" | "stale" | "none";
  stages: Record<string, StageStatus>;
  pending_changes?: StateChange[];
  last_validated?: string;
}

export interface LoadStateResponse {
  state: ScenarioState | null;
  found: boolean;
  manifest_changed?: boolean;
  current_hash?: string;
  stored_hash?: string;
}

export interface SaveStateResponse {
  success: boolean;
  updated_at: string;
  hash?: string;
  conflict?: boolean;
  server_state?: ScenarioState;
}

export interface CheckStalenessResponse {
  valid: boolean;
  current_hash?: string;
  stored_hash?: string;
  changed: boolean;
  pending_changes?: StateChange[];
  affected_stages?: string[];
  status?: ValidationStatus;
}

export interface GetLogsResponse {
  service_id: string;
  content: string;
  lines: number;
  captured_at: string;
}

export interface LoadStateOptions {
  includeLogs?: boolean;
  validateManifest?: boolean;
  manifestPath?: string;
}

export interface SaveStateOptions {
  manifestPath?: string;
  computeHash?: boolean;
  logTails?: Array<{ service_id: string; content: string; lines: number }>;
  buildArtifacts?: BuildArtifact[];
  stageResults?: Record<string, unknown>;
  expectedHash?: string;
}

export async function fetchScenarioState(
  scenarioName: string,
  options?: LoadStateOptions
): Promise<LoadStateResponse> {
  const params = new URLSearchParams();
  if (options?.includeLogs) params.set("include_logs", "true");
  if (options?.validateManifest) params.set("validate_manifest", "true");
  if (options?.manifestPath) params.set("manifest_path", options.manifestPath);

  const url =
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state`) +
    (params.toString() ? `?${params.toString()}` : "");

  const response = await fetch(url);
  if (!response.ok) {
    if (response.status === 404) {
      return { state: null, found: false };
    }
    throw new Error(`Failed to fetch scenario state: ${response.statusText}`);
  }
  return response.json();
}

export async function saveScenarioState(
  scenarioName: string,
  formState: FormState,
  options?: SaveStateOptions
): Promise<SaveStateResponse> {
  const response = await fetch(
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state`),
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        form_state: formState,
        manifest_path: options?.manifestPath,
        compute_hash: options?.computeHash,
        log_tails: options?.logTails,
        build_artifacts: options?.buildArtifacts,
        stage_results: options?.stageResults,
        expected_hash: options?.expectedHash,
      }),
    }
  );
  if (!response.ok) {
    throw new Error(`Failed to save scenario state: ${response.statusText}`);
  }
  return response.json();
}

export async function deleteScenarioState(scenarioName: string): Promise<void> {
  const response = await fetch(
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state`),
    { method: "DELETE" }
  );
  if (!response.ok) {
    throw new Error(`Failed to delete scenario state: ${response.statusText}`);
  }
}

export async function checkStateStaleness(
  scenarioName: string,
  currentConfig: InputFingerprint
): Promise<CheckStalenessResponse> {
  const response = await fetch(
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state/check`),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ current_config: currentConfig }),
    }
  );
  if (!response.ok) {
    throw new Error(`Failed to check state staleness: ${response.statusText}`);
  }
  return response.json();
}

export async function getScenarioLogs(
  scenarioName: string,
  serviceId: string
): Promise<GetLogsResponse | null> {
  const response = await fetch(
    buildUrl(
      `/scenarios/${encodeURIComponent(scenarioName)}/state/logs/${encodeURIComponent(serviceId)}`
    )
  );
  if (!response.ok) {
    if (response.status === 404) {
      return null;
    }
    throw new Error(`Failed to fetch logs: ${response.statusText}`);
  }
  return response.json();
}

export async function invalidateScenarioStage(
  scenarioName: string,
  fromStage: string,
  reason?: string
): Promise<ValidationStatus> {
  const response = await fetch(
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state/invalidate`),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ from_stage: fromStage, reason }),
    }
  );
  if (!response.ok) {
    throw new Error(`Failed to invalidate stage: ${response.statusText}`);
  }
  return response.json();
}

// ==================== Distribution API ====================

export interface DistributionRetryConfig {
  max_attempts?: number;
  initial_backoff_ms?: number;
  max_backoff_ms?: number;
  backoff_multiplier?: number;
}

export interface DistributionTarget {
  name: string;
  enabled: boolean;
  provider: "s3" | "r2" | "s3-compatible";
  bucket: string;
  endpoint?: string;
  region?: string;
  path_prefix?: string;
  access_key_id_env: string;
  secret_access_key_env: string;
  acl?: string;
  cdn_url?: string;
  retry?: DistributionRetryConfig;
  created_at?: string;
  updated_at?: string;
}

export interface DistributionConfig {
  schema_version: string;
  targets: Record<string, DistributionTarget>;
  created_at?: string;
  updated_at?: string;
}

export interface DistributionConfigResponse {
  config: DistributionConfig | null;
  config_path: string;
}

export interface DistributionTargetResponse {
  target: DistributionTarget;
}

export interface DistributionTargetsResponse {
  targets: DistributionTarget[];
  count: number;
}

export interface DistributionTargetValidation {
  target_name: string;
  valid: boolean;
  errors: string[];
  warnings: string[];
}

export interface DistributionValidationResult {
  valid: boolean;
  targets: Record<string, DistributionTargetValidation>;
  error?: string;
}

export interface DistributionTestResult {
  target_name: string;
  success: boolean;
  message?: string;
  error?: string;
}

export interface DistributeRequest {
  scenario_name: string;
  version?: string;
  artifacts: Record<string, string>;
  target_names?: string[];
  parallel?: boolean;
}

export interface DistributeResponse {
  distribution_id: string;
  status_url: string;
}

export interface UploadStatus {
  platform: string;
  status: "pending" | "uploading" | "completed" | "failed";
  url?: string;
  error?: string;
  progress?: number;
  started_at?: string;
  completed_at?: string;
}

export interface TargetDistribution {
  target_name: string;
  status: "pending" | "running" | "completed" | "partial" | "failed";
  uploads: Record<string, UploadStatus>;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface DistributionStatus {
  distribution_id: string;
  scenario_name: string;
  version?: string;
  status: "pending" | "running" | "completed" | "partial" | "failed" | "cancelled";
  targets: Record<string, TargetDistribution>;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export async function fetchDistributionConfig(): Promise<DistributionConfigResponse> {
  const response = await fetch(buildUrl("/distribution/config"));
  if (!response.ok) {
    throw new Error(`Failed to fetch distribution config: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchDistributionConfigPath(): Promise<{ path: string }> {
  const response = await fetch(buildUrl("/distribution/config-path"));
  if (!response.ok) {
    throw new Error(`Failed to fetch distribution config path: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchDistributionTargets(): Promise<DistributionTargetsResponse> {
  const response = await fetch(buildUrl("/distribution/targets"));
  if (!response.ok) {
    throw new Error(`Failed to fetch distribution targets: ${response.statusText}`);
  }
  return response.json();
}

export async function fetchDistributionTarget(name: string): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`));
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error(`Target "${name}" not found`);
    }
    throw new Error(`Failed to fetch distribution target: ${response.statusText}`);
  }
  return response.json();
}

export async function createDistributionTarget(target: DistributionTarget): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl("/distribution/targets"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(target),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to create distribution target");
  }
  return response.json();
}

export async function updateDistributionTarget(
  name: string,
  target: Partial<DistributionTarget>
): Promise<DistributionTargetResponse> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`), {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(target),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to update distribution target");
  }
  return response.json();
}

export async function deleteDistributionTarget(name: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}`), {
    method: "DELETE",
  });
  if (!response.ok) {
    throw new Error(`Failed to delete distribution target: ${response.statusText}`);
  }
  return response.json();
}

export async function testDistributionTarget(name: string): Promise<DistributionTestResult> {
  const response = await fetch(buildUrl(`/distribution/targets/${encodeURIComponent(name)}/test`), {
    method: "POST",
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to test distribution target");
  }
  return response.json();
}

export async function validateDistributionTargets(
  targetNames?: string[]
): Promise<DistributionValidationResult> {
  const response = await fetch(buildUrl("/distribution/validate"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target_names: targetNames }),
  });
  if (!response.ok) {
    throw new Error(`Failed to validate distribution targets: ${response.statusText}`);
  }
  return response.json();
}

export async function startDistribution(request: DistributeRequest): Promise<DistributeResponse> {
  const response = await fetch(buildUrl("/distribution/distribute"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: response.statusText }));
    throw new Error(error.error || error.message || "Failed to start distribution");
  }
  return response.json();
}

export async function fetchDistributionStatus(distributionId: string): Promise<DistributionStatus> {
  const response = await fetch(buildUrl(`/distribution/status/${encodeURIComponent(distributionId)}`));
  if (!response.ok) {
    if (response.status === 404) {
      throw new Error(`Distribution "${distributionId}" not found`);
    }
    throw new Error(`Failed to fetch distribution status: ${response.statusText}`);
  }
  return response.json();
}

export async function cancelDistribution(distributionId: string): Promise<{ status: string }> {
  const response = await fetch(buildUrl(`/distribution/cancel/${encodeURIComponent(distributionId)}`), {
    method: "POST",
  });
  if (!response.ok) {
    throw new Error(`Failed to cancel distribution: ${response.statusText}`);
  }
  return response.json();
}
