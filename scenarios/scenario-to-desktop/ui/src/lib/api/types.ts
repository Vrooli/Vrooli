// Re-export client types
export type { RecoveryAction, ApiErrorResponse } from "./client";
export { ApiError } from "./client";

// ==================== Docs Types ====================

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

// ==================== System Types ====================

export interface HealthResponse {
  status: string;
  service: string;
  timestamp: string;
}

// ==================== Template Types ====================

export interface TemplateInfo {
  name: string;
  description: string;
  type: string;
  framework: string;
  use_cases: string[];
  features: string[];
  complexity: string;
}

// ==================== Desktop Config Types ====================

export interface SigningConfig {
  schema_version?: string;
  enabled: boolean;
  windows?: WindowsSigningConfig;
  macos?: MacOSSigningConfig;
  linux?: LinuxSigningConfig;
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

// ==================== Bundle Types ====================

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

// ==================== Preflight Types ====================

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

export interface BundleManifestResponse {
  path: string;
  manifest: unknown;
}

// ==================== Build Types ====================

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

// ==================== Smoke Test Types ====================

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

// ==================== Probe Types ====================

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

// ==================== Desktop Record Types ====================

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

// ==================== Test Artifact Types ====================

export interface TestArtifactSummary {
  count: number;
  total_bytes: number;
  paths?: string[];
}

export interface TestArtifactCleanupResult {
  removed_count: number;
  freed_bytes: number;
}

// ==================== Wine Types ====================

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

// ==================== Telemetry Types ====================

export interface TelemetryUploadRequest {
  scenario_name: string;
  deployment_mode?: string;
  source?: string;
  events: unknown[];
}

// ==================== Port Types ====================

export interface ScenarioPortResponse {
  scenario: string;
  port_name: string;
  host: string;
  port: number;
  url: string;
}

// ==================== Signing Types ====================

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

export interface SigningConfigResponse {
  scenario: string;
  config: SigningConfig | null;
  config_path: string;
}

/** Validation error for signing configuration validation. */
export interface SigningValidationError {
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

export interface SigningValidationResult {
  valid: boolean;
  platforms: Record<string, PlatformValidation>;
  errors: SigningValidationError[];
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
