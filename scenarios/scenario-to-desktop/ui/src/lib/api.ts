import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import type { ScenariosResponse } from "../components/scenario-inventory/types";

const API_BASE = resolveApiBase({ appendSuffix: true });
const buildUrl = (path: string) => buildApiUrl(path, { baseUrl: API_BASE });

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
}

export async function exportBundleFromDeploymentManager(opts: {
  baseUrl: string;
  scenario: string;
  tier?: string;
  includeSecrets?: boolean;
}): Promise<BundleExportResponse> {
  const tier = opts.tier || "tier-2-desktop";
  const includeSecrets = opts.includeSecrets ?? true;
  const base = opts.baseUrl.replace(/\/+$/, "");
  const response = await fetch(`${base}/api/v1/bundles/export`, {
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
