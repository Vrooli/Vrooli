/**
 * Proto contract definitions for the System Monitor UI.
 *
 * Provides typed parse helpers that deserialize raw JSON into protobuf
 * message shapes using `fromJson` with `useProtoNames` so the API's
 * snake_case field names are handled correctly.
 */

import { fromJson, toJsonString, create, type JsonValue, type MessageShape } from "@bufbuild/protobuf";

// Domain schemas — metrics
import {
  MetricsResponseSchema,
  MetricsTimelineResponseSchema,
  DetailedMetricsSchema,
  DiskDetailResponseSchema,
  ProcessMonitorDataSchema,
  InfrastructureMonitorDataSchema,
} from "@vrooli/proto-types/system-monitor/v1/domain/metrics_pb";

// Domain schemas — investigations
import {
  InvestigationSchema,
  TriggerConfigSchema,
  CooldownStatusSchema,
} from "@vrooli/proto-types/system-monitor/v1/domain/investigations_pb";

// Domain schemas — settings
import { SystemSettingsSchema } from "@vrooli/proto-types/system-monitor/v1/domain/settings_pb";

// API schemas — settings
import {
  GetSettingsResponseSchema,
  UpdateSettingsResponseSchema,
  ResetSettingsResponseSchema,
  GetMaintenanceStateResponseSchema,
  SetMaintenanceStateResponseSchema,
} from "@vrooli/proto-types/system-monitor/v1/api/settings_pb";

// API schemas — investigations
import {
  GetTriggersResponseSchema,
  TriggerInvestigationResponseSchema,
  GetCooldownStatusResponseSchema,
  ListInvestigationsResponseSchema,
} from "@vrooli/proto-types/system-monitor/v1/api/investigations_pb";

// API schemas — scripts
import {
  ListScriptsResponseSchema,
  GetScriptResponseSchema,
  ExecuteScriptResponseSchema,
} from "@vrooli/proto-types/system-monitor/v1/api/scripts_pb";

// Domain schemas — reports
import { EnhancedSystemReportSchema } from "@vrooli/proto-types/system-monitor/v1/domain/reports_pb";

// Domain schemas — scripts
import {
  InvestigationScriptSchema,
  ScriptExecutionSchema,
} from "@vrooli/proto-types/system-monitor/v1/domain/scripts_pb";

// ---------------------------------------------------------------------------
// Shared options
// ---------------------------------------------------------------------------

const PROTO_JSON_OPTIONS = { ignoreUnknownFields: true } as const;

// ---------------------------------------------------------------------------
// Parse helpers — metrics
// ---------------------------------------------------------------------------

export function parseMetricsResponse(data: unknown): MessageShape<typeof MetricsResponseSchema> {
  return fromJson(MetricsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseDetailedMetrics(data: unknown): MessageShape<typeof DetailedMetricsSchema> {
  return fromJson(DetailedMetricsSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseProcessMonitorData(data: unknown): MessageShape<typeof ProcessMonitorDataSchema> {
  return fromJson(ProcessMonitorDataSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseInfrastructureMonitorData(data: unknown): MessageShape<typeof InfrastructureMonitorDataSchema> {
  return fromJson(InfrastructureMonitorDataSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseMetricsTimelineResponse(data: unknown): MessageShape<typeof MetricsTimelineResponseSchema> {
  return fromJson(MetricsTimelineResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseDiskDetailResponse(data: unknown): MessageShape<typeof DiskDetailResponseSchema> {
  return fromJson(DiskDetailResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — investigations
// ---------------------------------------------------------------------------

export function parseInvestigation(data: unknown): MessageShape<typeof InvestigationSchema> {
  return fromJson(InvestigationSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseInvestigations(data: unknown): MessageShape<typeof InvestigationSchema>[] {
  if (!Array.isArray(data)) return [];
  return data.map((item: unknown) => fromJson(InvestigationSchema, item as JsonValue, PROTO_JSON_OPTIONS));
}

export function parseTriggerConfig(data: unknown): MessageShape<typeof TriggerConfigSchema> {
  return fromJson(TriggerConfigSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseCooldownStatus(data: unknown): MessageShape<typeof CooldownStatusSchema> {
  return fromJson(CooldownStatusSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — settings
// ---------------------------------------------------------------------------

export function parseSystemSettings(data: unknown): MessageShape<typeof SystemSettingsSchema> {
  return fromJson(SystemSettingsSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — reports
// ---------------------------------------------------------------------------

export function parseEnhancedSystemReport(data: unknown): MessageShape<typeof EnhancedSystemReportSchema> {
  return fromJson(EnhancedSystemReportSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — scripts
// ---------------------------------------------------------------------------

export function parseInvestigationScript(data: unknown): MessageShape<typeof InvestigationScriptSchema> {
  return fromJson(InvestigationScriptSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseScriptExecution(data: unknown): MessageShape<typeof ScriptExecutionSchema> {
  return fromJson(ScriptExecutionSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (scripts)
// ---------------------------------------------------------------------------

export function parseListScriptsResponse(data: unknown): MessageShape<typeof ListScriptsResponseSchema> {
  return fromJson(ListScriptsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseGetScriptResponse(data: unknown): MessageShape<typeof GetScriptResponseSchema> {
  return fromJson(GetScriptResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseExecuteScriptResponse(data: unknown): MessageShape<typeof ExecuteScriptResponseSchema> {
  return fromJson(ExecuteScriptResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (settings)
// ---------------------------------------------------------------------------

export function parseGetSettingsResponse(data: unknown): MessageShape<typeof GetSettingsResponseSchema> {
  return fromJson(GetSettingsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseUpdateSettingsResponse(data: unknown): MessageShape<typeof UpdateSettingsResponseSchema> {
  return fromJson(UpdateSettingsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseResetSettingsResponse(data: unknown): MessageShape<typeof ResetSettingsResponseSchema> {
  return fromJson(ResetSettingsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseGetMaintenanceStateResponse(data: unknown): MessageShape<typeof GetMaintenanceStateResponseSchema> {
  return fromJson(GetMaintenanceStateResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseSetMaintenanceStateResponse(data: unknown): MessageShape<typeof SetMaintenanceStateResponseSchema> {
  return fromJson(SetMaintenanceStateResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (investigations)
// ---------------------------------------------------------------------------

export function parseGetTriggersResponse(data: unknown): MessageShape<typeof GetTriggersResponseSchema> {
  return fromJson(GetTriggersResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseTriggerInvestigationResponse(data: unknown): MessageShape<typeof TriggerInvestigationResponseSchema> {
  return fromJson(TriggerInvestigationResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseGetCooldownStatusResponse(data: unknown): MessageShape<typeof GetCooldownStatusResponseSchema> {
  return fromJson(GetCooldownStatusResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

export function parseListInvestigationsResponse(data: unknown): MessageShape<typeof ListInvestigationsResponseSchema> {
  return fromJson(ListInvestigationsResponseSchema, data as JsonValue, PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Re-export schema descriptors for request building
// ---------------------------------------------------------------------------

export {
  MetricsResponseSchema,
  MetricsTimelineResponseSchema,
  DetailedMetricsSchema,
  DiskDetailResponseSchema,
  ProcessMonitorDataSchema,
  InfrastructureMonitorDataSchema,
  InvestigationSchema,
  TriggerConfigSchema,
  CooldownStatusSchema,
  SystemSettingsSchema,
  EnhancedSystemReportSchema,
  InvestigationScriptSchema,
  ScriptExecutionSchema,
  GetSettingsResponseSchema,
  UpdateSettingsResponseSchema,
  ResetSettingsResponseSchema,
  GetMaintenanceStateResponseSchema,
  SetMaintenanceStateResponseSchema,
  GetTriggersResponseSchema,
  TriggerInvestigationResponseSchema,
  GetCooldownStatusResponseSchema,
  ListInvestigationsResponseSchema,
  ListScriptsResponseSchema,
  GetScriptResponseSchema,
  ExecuteScriptResponseSchema,
};

// Re-export utilities for consumers that need them
export { fromJson, toJsonString, create };
