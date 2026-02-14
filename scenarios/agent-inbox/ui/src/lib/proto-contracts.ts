/**
 * Proto-derived contracts for agent-inbox UI<->API communication.
 *
 * These interfaces match the snake_case JSON serialization used by the Go API
 * (protojson.MarshalOptions{UseProtoNames: true}). They are derived from the
 * canonical proto definitions for type-safety documentation and traceability.
 *
 * Source protos:
 *   - agent_inbox/v1/domain/tool.proto
 *   - agent_inbox/v1/domain/manifest.proto
 */

// Re-export proto types for reference/documentation
import type { ToolDefinition as ProtoToolDefinition } from "@vrooli/proto-types/agent-inbox/v1/domain/tool_pb";
import type { ToolParameters as ProtoToolParameters } from "@vrooli/proto-types/agent-inbox/v1/domain/tool_pb";
import type { ParameterSchema as ProtoParameterSchema } from "@vrooli/proto-types/agent-inbox/v1/domain/tool_pb";
import type { ToolMetadata as ProtoToolMetadata } from "@vrooli/proto-types/agent-inbox/v1/domain/tool_pb";
import type { ToolCategory as ProtoToolCategory } from "@vrooli/proto-types/agent-inbox/v1/domain/tool_pb";
import type { ScenarioInfo as ProtoScenarioInfo } from "@vrooli/proto-types/agent-inbox/v1/domain/manifest_pb";

// ---------------------------------------------------------------------------
// Snake_case interfaces matching API JSON serialization
// ---------------------------------------------------------------------------

/** Scenario metadata (proto: agent_inbox.v1.ScenarioInfo, UseProtoNames) */
export interface ScenarioInfo {
  name: string;
  version: string;
  description: string;
  base_url?: string;
}

/** Tool input parameter schema (proto: agent_inbox.v1.ParameterSchema, UseProtoNames) */
export interface ParameterSchema {
  type: string;
  description?: string;
  enum?: string[];
  default?: unknown;
  items?: ParameterSchema;
  properties?: Record<string, ParameterSchema>;
  format?: string;
}

/** Tool parameters (proto: agent_inbox.v1.ToolParameters, UseProtoNames) */
export interface ToolParameters {
  type: string;
  properties: Record<string, ParameterSchema>;
  required?: string[];
}

/** Tool orchestration metadata (proto: agent_inbox.v1.ToolMetadata, UseProtoNames) */
export interface ToolMetadata {
  enabled_by_default: boolean;
  requires_approval: boolean;
  timeout_seconds?: number;
  rate_limit_per_minute?: number;
  cost_estimate?: string;
  tags?: string[];
  long_running?: boolean;
  idempotent?: boolean;
}

/** Tool category for grouping (proto: agent_inbox.v1.ToolCategory, UseProtoNames) */
export interface ToolCategory {
  id: string;
  name: string;
  description?: string;
  icon?: string;
}

/** Discovered tool definition (proto: agent_inbox.v1.ToolDefinition, UseProtoNames) */
export interface DiscoveredTool {
  name: string;
  description: string;
  category?: string;
  parameters: ToolParameters;
  metadata: ToolMetadata;
}

// Proto type references (for documentation/traceability only)
export type {
  ProtoToolDefinition,
  ProtoToolParameters,
  ProtoParameterSchema,
  ProtoToolMetadata,
  ProtoToolCategory,
  ProtoScenarioInfo,
};
