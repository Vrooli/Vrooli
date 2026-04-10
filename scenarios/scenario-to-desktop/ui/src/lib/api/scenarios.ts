import { buildUrl, fetchJson, mutateJson, mutateVoid, throwIfNotOk } from "./client";
import { parseOrThrow } from "./safeParse";
import {
  LoadStateResponseSchema,
  SaveStateResponseSchema,
  CheckStalenessResponseSchema,
  GetLogsResponseSchema,
  ValidationStatusSchema,
  ScenariosResponseSchema,
  TemplateListResponseSchema,
} from "./schemas/scenarios";
import type { BundlePreflightResponse } from "./types";

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

/** @deprecated Use ScenarioStageStatus instead */
export type StageStatus = ScenarioStageStatus;

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

// ==================== Scenario State Functions ====================

export async function fetchScenarioState(
  scenarioName: string,
  options?: LoadStateOptions,
) {
  const params = new URLSearchParams();
  if (options?.includeLogs) params.set("include_logs", "true");
  if (options?.validateManifest) params.set("validate_manifest", "true");
  if (options?.manifestPath) params.set("manifest_path", options.manifestPath);

  const url =
    buildUrl(`/scenarios/${encodeURIComponent(scenarioName)}/state`) +
    (params.toString() ? `?${params.toString()}` : "");

  const response = await fetch(url);
  // Special case: 404 means state doesn't exist yet (not an error)
  if (response.status === 404) {
    return { state: null, found: false } as const;
  }
  await throwIfNotOk(response);
  return parseOrThrow(LoadStateResponseSchema, await response.json());
}

export function saveScenarioState(
  scenarioName: string,
  formState: FormState,
  options?: SaveStateOptions,
) {
  return mutateJson(
    `/scenarios/${encodeURIComponent(scenarioName)}/state`,
    SaveStateResponseSchema,
    {
      method: "PUT",
      body: {
        form_state: formState,
        manifest_path: options?.manifestPath,
        compute_hash: options?.computeHash,
        log_tails: options?.logTails,
        build_artifacts: options?.buildArtifacts,
        stage_results: options?.stageResults,
        expected_hash: options?.expectedHash,
      },
    },
  );
}

export function deleteScenarioState(scenarioName: string) {
  return mutateVoid(
    `/scenarios/${encodeURIComponent(scenarioName)}/state`,
    { method: "DELETE" },
  );
}

export function checkStateStaleness(
  scenarioName: string,
  currentConfig: InputFingerprint,
) {
  return mutateJson(
    `/scenarios/${encodeURIComponent(scenarioName)}/state/check`,
    CheckStalenessResponseSchema,
    { method: "POST", body: { current_config: currentConfig } },
  );
}

export async function getScenarioLogs(
  scenarioName: string,
  serviceId: string,
) {
  const url = buildUrl(
    `/scenarios/${encodeURIComponent(scenarioName)}/state/logs/${encodeURIComponent(serviceId)}`
  );
  const response = await fetch(url);
  // Special case: 404 means logs don't exist (not an error)
  if (response.status === 404) {
    return null;
  }
  await throwIfNotOk(response);
  return parseOrThrow(GetLogsResponseSchema, await response.json());
}

export function invalidateScenarioStage(
  scenarioName: string,
  fromStage: string,
  reason?: string,
) {
  return mutateJson(
    `/scenarios/${encodeURIComponent(scenarioName)}/state/invalidate`,
    ValidationStatusSchema,
    { method: "POST", body: { from_stage: fromStage, reason } },
  );
}

// ==================== Other Scenario Functions ====================

export function fetchScenarioDesktopStatus() {
  return fetchJson("/scenarios/desktop-status", ScenariosResponseSchema);
}

export function fetchTemplates() {
  return fetchJson("/templates", TemplateListResponseSchema);
}
