import type { JsonObject } from "@bufbuild/protobuf";
import {
  LoadStateResponseSchema,
  SaveStateResponseSchema,
  CheckStalenessResponseSchema,
  GetLogsResponseSchema,
  ValidationStatusSchema,
  ScenariosResponseSchema,
} from "./schemas/scenarios";
import {
  operationsConnectClient,
  stateConnectClient,
  systemConnectClient,
} from "./connect";

// ==================== Scenario State Types ====================

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
  /** JSON persistence boundary for generated PreflightResponse evidence. */
  preflight_result?: unknown;
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
  /** Bundle result including auto-build status. Persisted for restoration on page load. */
  bundle_result?: unknown;
  /** Smoke test state - persisted for restoration on page load. */
  smoke_test_id?: string | null;
  smoke_test_platform?: "win" | "mac" | "linux" | null;
  smoke_test_status?: "running" | "passed" | "failed" | null;
  smoke_test_started_at?: string | null;
  smoke_test_completed_at?: string | null;
  smoke_test_logs?: string[] | null;
  smoke_test_error?: string | null;
  smoke_test_telemetry_uploaded?: boolean;
  /** Wrapper build state - persisted for restoration on page load. */
  wrapper_build_id?: string | null;
  wrapper_build_status?: "building" | "ready" | "failed" | null;
  wrapper_output_path?: string | null;
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

/**
 * Stage validation status for scenario state management.
 * NOTE: This is distinct from the StageStatus enum in common.ts which represents
 * pipeline execution status values (pending, running, completed, etc.)
 */
export interface ScenarioStageStatus {
  stage?: string;
  status: "valid" | "stale" | "invalid" | "none";
  last_run?: string;
  staleness_reason?: string;
  can_reuse?: boolean;
}

export interface ValidationStatus {
  scenario_name: string;
  overall_status: "valid" | "partial" | "stale" | "none";
  stages: Record<string, ScenarioStageStatus>;
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

const asJsonObject = (value: unknown): JsonObject =>
  JSON.parse(JSON.stringify(value)) as JsonObject;

const requiredPayload = (payload: JsonObject | undefined): JsonObject => {
  if (!payload) throw new Error("StateService returned an empty payload");
  return payload;
};

// ==================== Scenario State Functions ====================

export async function fetchScenarioState(
  scenarioName: string,
  options?: LoadStateOptions,
) {
  const response = await stateConnectClient.loadScenarioState({
    scenarioName,
    includeLogs: options?.includeLogs,
    validateManifest: options?.validateManifest,
    manifestPath: options?.manifestPath,
  });
  return LoadStateResponseSchema.parse(
    response.payload ?? { state: null, found: response.found },
  );
}

export async function saveScenarioState(
  scenarioName: string,
  formState: FormState,
  options?: SaveStateOptions,
) {
  const payload = {
    form_state: formState,
    manifest_path: options?.manifestPath,
    compute_hash: options?.computeHash,
    log_tails: options?.logTails,
    build_artifacts: options?.buildArtifacts,
    stage_results: options?.stageResults,
    expected_hash: options?.expectedHash,
  };
  const response = await stateConnectClient.saveScenarioState({
    scenarioName,
    payload: asJsonObject(payload),
  });
  return SaveStateResponseSchema.parse(requiredPayload(response.payload));
}

export async function deleteScenarioState(scenarioName: string) {
  await stateConnectClient.deleteScenarioState({ scenarioName });
}

export async function checkStateStaleness(
  scenarioName: string,
  currentConfig: InputFingerprint,
) {
  const response = await stateConnectClient.checkScenarioState({
    scenarioName,
    currentConfig: asJsonObject(currentConfig),
  });
  return CheckStalenessResponseSchema.parse(requiredPayload(response.payload));
}

export async function getScenarioLogs(scenarioName: string, serviceId: string) {
  const response = await stateConnectClient.getScenarioStateLog({
    scenarioName,
    serviceId,
  });
  return response.found && response.payload
    ? GetLogsResponseSchema.parse(response.payload)
    : null;
}

export async function invalidateScenarioStage(
  scenarioName: string,
  fromStage: string,
  reason?: string,
) {
  const response = await stateConnectClient.invalidateScenarioState({
    scenarioName,
    fromStage,
    reason,
  });
  return ValidationStatusSchema.parse(requiredPayload(response.payload));
}

// ==================== Other Scenario Functions ====================

export async function fetchScenarioDesktopStatus() {
  const response = await operationsConnectClient.listDesktopScenarioStatus({});
  return ScenariosResponseSchema.parse({
    scenarios: response.scenarios.map((scenario) => ({
      name: scenario.name,
      display_name: scenario.displayName,
      service_display_name: scenario.serviceDisplayName,
      service_description: scenario.serviceDescription,
      service_icon_path: scenario.serviceIconPath,
      has_desktop: scenario.hasDesktop,
      desktop_path: scenario.desktopPath,
      version: scenario.version,
      platforms: scenario.platforms,
      built: scenario.built,
      dist_path: scenario.distPath,
      last_modified: scenario.lastModified,
      package_size:
        scenario.packageSize === undefined
          ? undefined
          : Number(scenario.packageSize),
      connection_config: scenario.connectionConfig
        ? {
            deployment_mode: scenario.connectionConfig.mode,
            proxy_url: scenario.connectionConfig.endpoint,
          }
        : undefined,
      build_artifacts: scenario.buildArtifacts.map((artifact) => ({
        platform: artifact.platform,
        file_name: artifact.fileName,
        size_bytes: Number(artifact.sizeBytes),
        modified_at: artifact.modifiedAt,
        absolute_path: artifact.absolutePath,
        relative_path: artifact.relativePath,
      })),
      artifacts_source: scenario.artifactsSource,
      artifacts_path: scenario.artifactsPath,
      artifacts_expected_path: scenario.artifactsExpectedPath,
      record_id: scenario.recordId,
      record_output_path: scenario.recordOutputPath,
      record_location_mode: scenario.recordLocationMode,
      record_updated_at: scenario.recordUpdatedAt,
    })),
    stats: {
      total: response.stats?.total ?? 0,
      with_desktop: response.stats?.withDesktop ?? 0,
      built: response.stats?.built ?? 0,
      web_only: response.stats?.webOnly ?? 0,
    },
  });
}

export async function fetchTemplates() {
  return systemConnectClient.listTemplates({});
}
