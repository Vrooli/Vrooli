/**
 * Tool discovery, configuration, and execution API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { ToolCallRecord } from "./api-types";
import type { ScenarioInfo, ToolCategory, DiscoveredTool } from "./proto-contracts";
import type { ApprovalOverride } from "./api-types";

// =============================================================================
// Tool Discovery Protocol Types (re-exported from proto-contracts)
// =============================================================================

export type {
  ScenarioInfo,
  ToolParameters,
  ParameterSchema,
  ToolMetadata,
  ToolCategory,
  DiscoveredTool,
} from "./proto-contracts";

// =============================================================================
// Tool Configuration Types
// =============================================================================

export type ToolConfigurationScope = "global" | "chat" | "";

export interface EffectiveTool {
  scenario: string;
  tool: DiscoveredTool;
  enabled: boolean;
  source: ToolConfigurationScope;
  requires_approval: boolean;
  approval_source?: ToolConfigurationScope;
  approval_override?: ApprovalOverride;
}

export interface ToolSet {
  scenarios: ScenarioInfo[];
  tools: EffectiveTool[];
  categories: ToolCategory[];
  generated_at: string;
}

export interface ScenarioStatus {
  scenario: string;
  available: boolean;
  last_checked: string;
  tool_count?: number;
  error?: string;
}

export interface ToolConfigUpdate {
  chat_id?: string;
  scenario: string;
  tool_name: string;
  enabled: boolean;
}

/**
 * Result of tool discovery sync operation.
 */
export interface DiscoveryResult {
  scenarios_with_tools: number;
  new_scenarios: string[];
  removed_scenarios: string[];
  total_tools: number;
}

// =============================================================================
// Basic Tool Definitions
// =============================================================================

export interface ToolDefinition {
  type: string;
  function: {
    name: string;
    description: string;
    parameters: Record<string, unknown>;
  };
}

export async function fetchTools(): Promise<ToolDefinition[]> {
  const url = buildApiUrl("/tools", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch tools: ${res.status}`);
  }

  return jsonResponse<ToolDefinition[]>(res);
}

export async function fetchChatToolCalls(chatId: string): Promise<ToolCallRecord[]> {
  const url = buildApiUrl(`/chats/${chatId}/tool-calls`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch tool calls: ${res.status}`);
  }

  return jsonResponse<ToolCallRecord[]>(res);
}

// =============================================================================
// Tool Configuration API Functions
// =============================================================================

/**
 * Fetch the complete tool set with effective enabled states.
 * @param chatId - Optional chat ID for chat-specific configurations
 */
export async function fetchToolSet(chatId?: string): Promise<ToolSet> {
  const params = new URLSearchParams();
  if (chatId) params.set("chat_id", chatId);

  const queryString = params.toString();
  const endpoint = queryString ? `/tools/set?${queryString}` : "/tools/set";
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch tool set: ${res.status}`);
  }

  return jsonResponse<ToolSet>(res);
}

/**
 * Fetch availability status of all configured scenarios.
 */
export async function fetchScenarioStatuses(): Promise<ScenarioStatus[]> {
  const url = buildApiUrl("/tools/scenarios", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch scenario statuses: ${res.status}`);
  }

  return jsonResponse<ScenarioStatus[]>(res);
}

/**
 * Update the enabled state for a tool.
 * @param config - Tool configuration update
 */
export async function setToolEnabled(config: ToolConfigUpdate): Promise<void> {
  const url = buildApiUrl("/tools/config", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });

  if (!res.ok) {
    throw new Error(`Failed to update tool configuration: ${res.status}`);
  }
}

/**
 * Reset a tool configuration to default.
 * @param scenario - Scenario name
 * @param toolName - Tool name
 * @param chatId - Optional chat ID (empty for global)
 */
export async function resetToolConfig(
  scenario: string,
  toolName: string,
  chatId?: string
): Promise<void> {
  const params = new URLSearchParams({
    scenario,
    tool_name: toolName
  });
  if (chatId) params.set("chat_id", chatId);

  const url = buildApiUrl(`/tools/config?${params.toString()}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to reset tool configuration: ${res.status}`);
  }
}

/**
 * Fetch information about a specific scenario.
 * @param name - Scenario name
 */
export async function fetchScenarioInfo(name: string): Promise<ScenarioInfo | null> {
  const url = buildApiUrl(`/scenarios/${encodeURIComponent(name)}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (res.status === 404) {
    return null;
  }

  if (!res.ok) {
    throw new Error(`Failed to fetch scenario info: ${res.status}`);
  }

  return jsonResponse<ScenarioInfo>(res);
}

/**
 * Perform full tool discovery from all running scenarios.
 * Discovers scenarios via vrooli CLI and probes each for /api/v1/tools.
 */
export async function syncTools(): Promise<DiscoveryResult> {
  const url = buildApiUrl("/tools/sync", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Tool discovery failed: ${errorText}`);
  }

  return jsonResponse<DiscoveryResult>(res);
}

// =============================================================================
// Manual Tool Execution
// =============================================================================

/**
 * Request for manual tool execution.
 */
export interface ManualToolExecuteRequest {
  scenario: string;
  tool_name: string;
  arguments: Record<string, unknown>;
  chat_id?: string;
}

/**
 * Response from manual tool execution.
 */
export interface ManualToolExecuteResponse {
  success: boolean;
  result: string | number | boolean | Record<string, unknown> | unknown[] | null;
  status: "completed" | "failed";
  error?: string;
  execution_time_ms: number;
  tool_call_record?: {
    id: string;
    message_id: string;
  };
}

/**
 * Execute a tool manually without going through AI.
 * @param request - The execution request with tool details and arguments
 */
export async function executeToolManually(
  request: ManualToolExecuteRequest
): Promise<ManualToolExecuteResponse> {
  const url = buildApiUrl("/tools/execute", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });

  if (!res.ok) {
    throw new Error(`Failed to execute tool: ${res.status}`);
  }

  return jsonResponse<ManualToolExecuteResponse>(res);
}
