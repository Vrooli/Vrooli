/**
 * Zod schemas for scenario state management API response types.
 *
 * Covers scenario state load/save, staleness checks, log retrieval,
 * validation status, desktop status listings, and template queries.
 */

import { z } from "zod";
import { TemplateInfoSchema } from "./common";

// ==================== Form & Stage State ====================

export const FormStateSchema = z.object({
  selected_template: z.string().optional(),
  framework: z.string().optional(),
  app_display_name: z.string().optional(),
  app_description: z.string().optional(),
  icon_path: z.string().optional(),
  display_name_edited: z.boolean().optional(),
  description_edited: z.boolean().optional(),
  icon_path_edited: z.boolean().optional(),
  server_type: z.string().optional(),
  deployment_mode: z.string().optional(),
  proxy_url: z.string().optional(),
  server_port: z.number().optional(),
  local_server_path: z.string().optional(),
  local_api_endpoint: z.string().optional(),
  auto_manage_tier1: z.boolean().optional(),
  vrooli_binary_path: z.string().optional(),
  bundle_manifest_path: z.string().optional(),
  platforms: z
    .object({
      win: z.boolean().optional(),
      mac: z.boolean().optional(),
      linux: z.boolean().optional(),
    })
    .optional(),
  location_mode: z.string().optional(),
  output_path: z.string().optional(),
  connection_result: z.any().optional(),
  connection_error: z.string().nullable().optional(),
  // Generated preflight evidence is validated at its persistence boundary,
  // not here at the outer FormState boundary.
  preflight_result: z.any().nullable().optional(),
  preflight_error: z.string().nullable().optional(),
  preflight_override: z.boolean().optional(),
  preflight_secrets: z.record(z.string()).optional(),
  preflight_start_services: z.boolean().optional(),
  preflight_auto_refresh: z.boolean().optional(),
  preflight_session_id: z.string().nullable().optional(),
  preflight_session_expires_at: z.string().nullable().optional(),
  preflight_session_ttl: z.number().optional(),
  deployment_manager_url: z.string().nullable().optional(),
  signing_enabled_for_build: z.boolean().optional(),
  bundle_result: z.any().optional(),
  smoke_test_id: z.string().nullable().optional(),
  smoke_test_platform: z.enum(["win", "mac", "linux"]).nullable().optional(),
  smoke_test_status: z
    .enum(["running", "passed", "failed"])
    .nullable()
    .optional(),
  smoke_test_started_at: z.string().nullable().optional(),
  smoke_test_completed_at: z.string().nullable().optional(),
  smoke_test_logs: z.array(z.string()).nullable().optional(),
  smoke_test_error: z.string().nullable().optional(),
  smoke_test_telemetry_uploaded: z.boolean().optional(),
  wrapper_build_id: z.string().nullable().optional(),
  wrapper_build_status: z
    .enum(["building", "ready", "failed"])
    .nullable()
    .optional(),
  wrapper_output_path: z.string().nullable().optional(),
});

export const InputFingerprintSchema = z.object({
  manifest_path: z.string().optional(),
  manifest_hash: z.string().optional(),
  manifest_mtime: z.number().optional(),
  preflight_secret_keys: z.array(z.string()).optional(),
  preflight_timeout: z.number().optional(),
  start_services: z.boolean().optional(),
  template_type: z.string().optional(),
  framework: z.string().optional(),
  deployment_mode: z.string().optional(),
  app_display_name: z.string().optional(),
  app_description: z.string().optional(),
  icon_path: z.string().optional(),
  platforms: z.array(z.string()).optional(),
  signing_enabled: z.boolean().optional(),
  signing_config_hash: z.string().optional(),
  output_location: z.string().optional(),
  smoke_test_platform: z.string().optional(),
});

export const StageStateSchema = z.object({
  stage: z.string(),
  status: z.enum(["valid", "stale", "invalid", "none"]),
  input_fingerprint: InputFingerprintSchema.optional(),
  output_hash: z.string().optional(),
  validated_at: z.string().optional(),
  result: z.any().optional(),
  staleness_reason: z.string().optional(),
});

export const CompressedLogSchema = z.object({
  service_id: z.string(),
  compressed_data: z.string(),
  original_lines: z.number(),
  compressed_size: z.number(),
  captured_at: z.string(),
});

export const BuildArtifactSchema = z.object({
  platform: z.string(),
  status: z.enum(["pending", "building", "ready", "failed"]),
  file_path: z.string().optional(),
  file_name: z.string().optional(),
  file_size: z.number().optional(),
  build_id: z.string().optional(),
  built_at: z.string().optional(),
  error_message: z.string().optional(),
});

export const ScenarioStateSchema = z.object({
  scenario_name: z.string(),
  schema_version: z.number(),
  created_at: z.string(),
  updated_at: z.string(),
  hash: z.string().optional(),
  form_state: FormStateSchema,
  stages: z.record(StageStateSchema).optional(),
  compressed_logs: z.array(CompressedLogSchema).optional(),
  build_artifacts: z.array(BuildArtifactSchema).optional(),
});

// ==================== Validation ====================

export const ScenarioStageStatusSchema = z.object({
  stage: z.string().optional(),
  status: z.enum(["valid", "stale", "invalid", "none"]),
  last_run: z.string().optional(),
  staleness_reason: z.string().optional(),
  can_reuse: z.boolean().optional(),
});

export const StateChangeSchema = z.object({
  change_type: z.string(),
  affected_stage: z.string(),
  reason: z.string(),
  old_value: z.string().optional(),
  new_value: z.string().optional(),
});

export const ValidationStatusSchema = z.object({
  scenario_name: z.string(),
  overall_status: z.enum(["valid", "partial", "stale", "none"]),
  stages: z.record(ScenarioStageStatusSchema),
  pending_changes: z.array(StateChangeSchema).optional(),
  last_validated: z.string().optional(),
});

// ==================== API Responses ====================

export const LoadStateResponseSchema = z.object({
  state: ScenarioStateSchema.nullable(),
  found: z.boolean(),
  manifest_changed: z.boolean().optional(),
  current_hash: z.string().optional(),
  stored_hash: z.string().optional(),
});

export const SaveStateResponseSchema = z.object({
  success: z.boolean(),
  updated_at: z.string(),
  hash: z.string().optional(),
  conflict: z.boolean().optional(),
  server_state: ScenarioStateSchema.optional(),
});

export const CheckStalenessResponseSchema = z.object({
  valid: z.boolean(),
  current_hash: z.string().optional(),
  stored_hash: z.string().optional(),
  changed: z.boolean(),
  pending_changes: z.array(StateChangeSchema).optional(),
  affected_stages: z.array(z.string()).optional(),
  status: ValidationStatusSchema.optional(),
});

export const GetLogsResponseSchema = z.object({
  service_id: z.string(),
  content: z.string(),
  lines: z.number(),
  captured_at: z.string(),
});

// ==================== Scenario Listing ====================

export const DesktopBuildArtifactSchema = z.object({
  platform: z.string().optional(),
  file_name: z.string(),
  size_bytes: z.number().optional(),
  modified_at: z.string().optional(),
  absolute_path: z.string().optional(),
  relative_path: z.string().optional(),
});

export const DesktopConnectionConfigSchema = z.object({
  proxy_url: z.string().optional(),
  server_url: z.string().optional(),
  api_url: z.string().optional(),
  app_display_name: z.string().optional(),
  app_description: z.string().optional(),
  icon: z.string().optional(),
  deployment_mode: z.string().optional(),
  auto_manage_vrooli: z.boolean().optional(),
  vrooli_binary_path: z.string().optional(),
  server_type: z.string().optional(),
  bundle_manifest_path: z.string().optional(),
  updated_at: z.string().optional(),
});

export const ScenarioDesktopStatusSchema = z.object({
  name: z.string(),
  display_name: z.string().optional(),
  service_display_name: z.string().optional(),
  service_description: z.string().optional(),
  service_icon_path: z.string().optional(),
  has_desktop: z.boolean(),
  desktop_path: z.string().optional(),
  version: z.string().optional(),
  platforms: z.array(z.string()).optional(),
  built: z.boolean().optional(),
  dist_path: z.string().optional(),
  last_modified: z.string().optional(),
  package_size: z.number().optional(),
  connection_config: DesktopConnectionConfigSchema.optional(),
  build_artifacts: z.array(DesktopBuildArtifactSchema).optional(),
  artifacts_source: z.string().optional(),
  artifacts_path: z.string().optional(),
  artifacts_expected_path: z.string().optional(),
  record_id: z.string().optional(),
  record_output_path: z.string().optional(),
  record_location_mode: z.string().optional(),
  record_updated_at: z.string().optional(),
});

export const ScenariosResponseSchema = z.object({
  scenarios: z.array(ScenarioDesktopStatusSchema),
  stats: z.object({
    total: z.number(),
    with_desktop: z.number(),
    built: z.number(),
    web_only: z.number(),
  }),
});

export const TemplateListResponseSchema = z.object({
  templates: z.array(TemplateInfoSchema),
});
