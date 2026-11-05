// DOC: docs/internal/SEAMS.md#ui-and-api-contract-seam
/**
 * Proto contract definitions for the System Monitor UI.
 *
 * Provides typed parse helpers that deserialize raw JSON into protobuf
 * message shapes using `fromJson` with `useProtoNames` so the API's
 * snake_case field names are handled correctly.
 */

import { fromJson, toJsonString, create, type DescMessage, type JsonValue, type MessageShape } from "@bufbuild/protobuf";

// Domain schemas — metrics
import { MetricsResponseSchema, MetricsTimelineResponseSchema, DetailedMetricsSchema, DiskDetailResponseSchema, ProcessMonitorDataSchema, InfrastructureMonitorDataSchema } from "@vrooli/proto-types/system-monitor/v1/metrics/metrics_pb";
import { InvestigationSchema, CooldownStatusSchema, TriggerInvestigationResponseSchema, GetCooldownStatusResponseSchema, ListInvestigationsResponseSchema, StopAgentResponseSchema } from "@vrooli/proto-types/system-monitor/v1/investigations/investigations_pb";
import { SystemSettingsSchema, GetSettingsResponseSchema, UpdateSettingsResponseSchema, ResetSettingsResponseSchema, GetMaintenanceStateResponseSchema, SetMaintenanceStateResponseSchema } from "@vrooli/proto-types/system-monitor/v1/settings/settings_pb";
import { ListScriptsResponseSchema, GetScriptResponseSchema, ExecuteScriptResponseSchema, InvestigationScriptSchema, ScriptExecutionSchema, ListRunsResponseSchema } from "@vrooli/proto-types/system-monitor/v1/scripts/scripts_pb";
import { EnhancedSystemReportSchema, ListReportsResponseSchema, GenerateReportResponseSchema } from "@vrooli/proto-types/system-monitor/v1/reports/reports_pb";
import { ListAlertsResponseSchema } from "@vrooli/proto-types/system-monitor/v1/alerts/alerts_pb";
import { GetCapacityOverviewResponseSchema, ListCapacityClaimsResponseSchema, ReconcileCapacityResponseSchema, GetCapacityPolicyResponseSchema, SetCapacityPolicyResponseSchema } from "@vrooli/proto-types/system-monitor/v1/capacity/capacity_pb";

// Domain schemas — investigations


// Domain schemas — settings


// API schemas — settings


// API schemas — investigations


// API schemas — scripts


// Domain schemas — reports


// API schemas — reports


// Domain schemas — scripts


// ---------------------------------------------------------------------------
// Shared options
// ---------------------------------------------------------------------------

const PROTO_JSON_OPTIONS = { ignoreUnknownFields: true } as const;

/** Validate values at the untyped API boundary before handing them to protobuf. */
function toProtoJsonValue(value: unknown, path = '$', ancestors = new WeakSet()): JsonValue {
  if (value === null || typeof value === 'string' || typeof value === 'boolean') return value;
  if (typeof value === 'number' && Number.isFinite(value)) return value;

  if (Array.isArray(value)) {
    if (ancestors.has(value)) throw new TypeError(`Cyclic API JSON value at ${path}`);
    ancestors.add(value);
    const result = Array.from(value, (item, index) => toProtoJsonValue(item, `${path}[${index}]`, ancestors));
    ancestors.delete(value);
    return result;
  }

  if (typeof value === 'object' && value !== null) {
    const prototype = Object.getPrototypeOf(value);
    if (prototype !== Object.prototype && prototype !== null) {
      throw new TypeError(`Unsupported API JSON object at ${path}`);
    }
    if (ancestors.has(value)) throw new TypeError(`Cyclic API JSON value at ${path}`);
    ancestors.add(value);
    const result: Record<string, JsonValue> = {};
    for (const [key, item] of Object.entries(value)) {
      Object.defineProperty(result, key, {
        value: toProtoJsonValue(item, `${path}.${key}`, ancestors),
        enumerable: true,
        configurable: true,
        writable: true,
      });
    }
    ancestors.delete(value);
    return result;
  }

  throw new TypeError(`Unsupported API JSON value at ${path}`);
}

export function parseProtoMessage<Desc extends DescMessage>(
  schema: Desc,
  data: unknown,
  path = '$',
): MessageShape<Desc> {
  return fromJson(schema, toProtoJsonValue(data, path), PROTO_JSON_OPTIONS);
}

// ---------------------------------------------------------------------------
// Parse helpers — metrics
// ---------------------------------------------------------------------------

export function parseMetricsResponse(data: unknown): MessageShape<typeof MetricsResponseSchema> {
  return parseProtoMessage(MetricsResponseSchema, data);
}

export function parseDetailedMetrics(data: unknown): MessageShape<typeof DetailedMetricsSchema> {
  return parseProtoMessage(DetailedMetricsSchema, data);
}

export function parseProcessMonitorData(data: unknown): MessageShape<typeof ProcessMonitorDataSchema> {
  return parseProtoMessage(ProcessMonitorDataSchema, data);
}

export function parseInfrastructureMonitorData(data: unknown): MessageShape<typeof InfrastructureMonitorDataSchema> {
  return parseProtoMessage(InfrastructureMonitorDataSchema, data);
}

export function parseMetricsTimelineResponse(data: unknown): MessageShape<typeof MetricsTimelineResponseSchema> {
  return parseProtoMessage(MetricsTimelineResponseSchema, data);
}

export function parseDiskDetailResponse(data: unknown): MessageShape<typeof DiskDetailResponseSchema> {
  return parseProtoMessage(DiskDetailResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — alerts
// ---------------------------------------------------------------------------

export function parseListAlertsResponse(data: unknown): MessageShape<typeof ListAlertsResponseSchema> {
  return parseProtoMessage(ListAlertsResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — investigations
// ---------------------------------------------------------------------------

export function parseInvestigation(data: unknown): MessageShape<typeof InvestigationSchema> {
  return parseProtoMessage(InvestigationSchema, data);
}

export function parseInvestigations(data: unknown): MessageShape<typeof InvestigationSchema>[] {
  if (!Array.isArray(data)) return [];
  return data.map((item: unknown, index: number) => parseProtoMessage(InvestigationSchema, item, `$[${index}]`));
}

export function parseCooldownStatus(data: unknown): MessageShape<typeof CooldownStatusSchema> {
  return parseProtoMessage(CooldownStatusSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — settings
// ---------------------------------------------------------------------------

export function parseSystemSettings(data: unknown): MessageShape<typeof SystemSettingsSchema> {
  return parseProtoMessage(SystemSettingsSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — reports
// ---------------------------------------------------------------------------

export function parseEnhancedSystemReport(data: unknown): MessageShape<typeof EnhancedSystemReportSchema> {
  return parseProtoMessage(EnhancedSystemReportSchema, data);
}

export function parseListReportsResponse(data: unknown): MessageShape<typeof ListReportsResponseSchema> {
  return parseProtoMessage(ListReportsResponseSchema, data);
}

export function parseGenerateReportResponse(data: unknown): MessageShape<typeof GenerateReportResponseSchema> {
  return parseProtoMessage(GenerateReportResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — scripts
// ---------------------------------------------------------------------------

export function parseInvestigationScript(data: unknown): MessageShape<typeof InvestigationScriptSchema> {
  return parseProtoMessage(InvestigationScriptSchema, data);
}

export function parseScriptExecution(data: unknown): MessageShape<typeof ScriptExecutionSchema> {
  return parseProtoMessage(ScriptExecutionSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (scripts)
// ---------------------------------------------------------------------------

export function parseListScriptsResponse(data: unknown): MessageShape<typeof ListScriptsResponseSchema> {
  return parseProtoMessage(ListScriptsResponseSchema, data);
}

export function parseGetScriptResponse(data: unknown): MessageShape<typeof GetScriptResponseSchema> {
  return parseProtoMessage(GetScriptResponseSchema, data);
}

export function parseExecuteScriptResponse(data: unknown): MessageShape<typeof ExecuteScriptResponseSchema> {
  return parseProtoMessage(ExecuteScriptResponseSchema, data);
}

export function parseListRunsResponse(data: unknown): MessageShape<typeof ListRunsResponseSchema> {
  return parseProtoMessage(ListRunsResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (settings)
// ---------------------------------------------------------------------------

export function parseGetSettingsResponse(data: unknown): MessageShape<typeof GetSettingsResponseSchema> {
  return parseProtoMessage(GetSettingsResponseSchema, data);
}

export function parseUpdateSettingsResponse(data: unknown): MessageShape<typeof UpdateSettingsResponseSchema> {
  return parseProtoMessage(UpdateSettingsResponseSchema, data);
}

export function parseResetSettingsResponse(data: unknown): MessageShape<typeof ResetSettingsResponseSchema> {
  return parseProtoMessage(ResetSettingsResponseSchema, data);
}

export function parseGetMaintenanceStateResponse(data: unknown): MessageShape<typeof GetMaintenanceStateResponseSchema> {
  return parseProtoMessage(GetMaintenanceStateResponseSchema, data);
}

export function parseSetMaintenanceStateResponse(data: unknown): MessageShape<typeof SetMaintenanceStateResponseSchema> {
  return parseProtoMessage(SetMaintenanceStateResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — capacity
// ---------------------------------------------------------------------------

export function parseGetCapacityOverviewResponse(data: unknown): MessageShape<typeof GetCapacityOverviewResponseSchema> {
  return parseProtoMessage(GetCapacityOverviewResponseSchema, data);
}

export function parseListCapacityClaimsResponse(data: unknown): MessageShape<typeof ListCapacityClaimsResponseSchema> {
  return parseProtoMessage(ListCapacityClaimsResponseSchema, data);
}

export function parseReconcileCapacityResponse(data: unknown): MessageShape<typeof ReconcileCapacityResponseSchema> {
  return parseProtoMessage(ReconcileCapacityResponseSchema, data);
}

export function parseGetCapacityPolicyResponse(data: unknown): MessageShape<typeof GetCapacityPolicyResponseSchema> {
  return parseProtoMessage(GetCapacityPolicyResponseSchema, data);
}

export function parseSetCapacityPolicyResponse(data: unknown): MessageShape<typeof SetCapacityPolicyResponseSchema> {
  return parseProtoMessage(SetCapacityPolicyResponseSchema, data);
}

// ---------------------------------------------------------------------------
// Parse helpers — API wrapper types (investigations)
// ---------------------------------------------------------------------------

export function parseTriggerInvestigationResponse(data: unknown): MessageShape<typeof TriggerInvestigationResponseSchema> {
  return parseProtoMessage(TriggerInvestigationResponseSchema, data);
}

export function parseGetCooldownStatusResponse(data: unknown): MessageShape<typeof GetCooldownStatusResponseSchema> {
  return parseProtoMessage(GetCooldownStatusResponseSchema, data);
}

export function parseListInvestigationsResponse(data: unknown): MessageShape<typeof ListInvestigationsResponseSchema> {
  return parseProtoMessage(ListInvestigationsResponseSchema, data);
}

export function parseStopAgentResponse(data: unknown): MessageShape<typeof StopAgentResponseSchema> {
  return parseProtoMessage(StopAgentResponseSchema, data);
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
  TriggerInvestigationResponseSchema,
  GetCooldownStatusResponseSchema,
  ListInvestigationsResponseSchema,
  ListScriptsResponseSchema,
  GetScriptResponseSchema,
  ExecuteScriptResponseSchema,
  ListReportsResponseSchema,
  GenerateReportResponseSchema,
  GetCapacityOverviewResponseSchema,
  ListCapacityClaimsResponseSchema,
  ReconcileCapacityResponseSchema,
  GetCapacityPolicyResponseSchema,
  SetCapacityPolicyResponseSchema,
};

// Re-export utilities for consumers that need them
export { fromJson, toJsonString, create };
