/**
 * Agent mode API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { ApiErrorBody } from "./api-base";
import {
  AgentModeError,
  type AgentChatConfig,
  type AgentModeResponse,
  type AgentModeStatus,
  type AgentEventsResponse,
  type ListAgentRunsOptions,
  type ListAgentRunsResponse,
} from "./api-agent-types";

// Re-export all types and helpers from agent-types
export {
  AgentModeError,
  RUNNER_OPTIONS,
  SUPPORTED_RUNNER_TYPES,
  isCompactionEvent,
  getCompactionReduction,
} from "./api-agent-types";

export type {
  RunnerType,
  AgentRunStatus,
  AgentChatConfig,
  AgentModeResponse,
  AgentModeStatus,
  AgentEvent,
  AgentEventsResponse,
  AgentRunSummary,
  ListAgentRunsResponse,
  ListAgentRunsOptions,
} from "./api-agent-types";

// =============================================================================
// Agent Mode API Functions
// =============================================================================

/**
 * Start agent mode for a chat.
 * Creates a task and run in agent-manager, switches the chat to agent mode.
 */
export async function startAgentMode(
  chatId: string,
  config: AgentChatConfig
): Promise<AgentModeResponse> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/start`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config)
  });

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body.error?.details?.user_message;
    const message = detail || body.error?.message || `Failed to start agent mode: ${res.status}`;
    throw new AgentModeError(message, body.error?.code, body.error?.recovery);
  }

  return jsonResponse<AgentModeResponse>(res);
}

/**
 * Send a message in agent mode.
 * Continues the existing agent run with a follow-up message.
 */
export async function sendAgentMessage(
  chatId: string,
  message: string,
  attachmentIds?: string[]
): Promise<{ success: boolean; run_id: string }> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/message`, { baseUrl: API_BASE });

  const reqBody: Record<string, unknown> = { message };
  if (attachmentIds && attachmentIds.length > 0) {
    reqBody.attachment_ids = attachmentIds;
  }

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(reqBody)
  });

  if (!res.ok) {
    const errBody: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = errBody.error?.details?.user_message;
    const msg = detail || errBody.error?.message || `Failed to send agent message: ${res.status}`;
    throw new AgentModeError(msg, errBody.error?.code, errBody.error?.recovery);
  }

  return jsonResponse<{ success: boolean; run_id: string }>(res);
}

/**
 * Get events for an agent run.
 * @param chatId - The chat ID
 * @param afterSequence - Only return events after this sequence number
 */
export async function getAgentEvents(
  chatId: string,
  afterSequence: number = 0,
  signal?: AbortSignal
): Promise<AgentEventsResponse> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/events?after_sequence=${afterSequence}`, {
    baseUrl: API_BASE
  });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    signal
  });

  if (!res.ok) {
    throw new Error(`Failed to get agent events: ${res.status}`);
  }

  return jsonResponse<AgentEventsResponse>(res);
}

/**
 * Get the current status of an agent chat.
 */
export async function getAgentStatus(chatId: string, signal?: AbortSignal): Promise<AgentModeStatus> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/status`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
    signal
  });

  if (!res.ok) {
    throw new Error(`Failed to get agent status: ${res.status}`);
  }

  return jsonResponse<AgentModeStatus>(res);
}

/**
 * Stop an agent run.
 */
export async function stopAgentMode(
  chatId: string
): Promise<{ success: boolean; run_id: string }> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/stop`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body.error?.details?.user_message;
    const msg = detail || body.error?.message || `Failed to stop agent: ${res.status}`;
    throw new AgentModeError(msg, body.error?.code, body.error?.recovery);
  }

  return jsonResponse<{ success: boolean; run_id: string }>(res);
}

/**
 * Clear agent mode and return to LLM mode.
 * Stops any running agent and resets the chat state.
 */
export async function clearAgentMode(
  chatId: string
): Promise<{ success: boolean; chat_mode: "llm" }> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/clear`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body.error?.details?.user_message;
    const msg = detail || body.error?.message || `Failed to clear agent mode: ${res.status}`;
    throw new AgentModeError(msg, body.error?.code, body.error?.recovery);
  }

  return jsonResponse<{ success: boolean; chat_mode: "llm" }>(res);
}

// =============================================================================
// Agent Runs
// =============================================================================

/**
 * List runs from agent-manager.
 */
export async function listAgentRuns(
  options?: ListAgentRunsOptions
): Promise<ListAgentRunsResponse> {
  const params = new URLSearchParams();
  if (options?.status) params.set("status", options.status);
  if (options?.tag_prefix) params.set("tag_prefix", options.tag_prefix);
  if (options?.limit) params.set("limit", String(options.limit));
  if (options?.offset) params.set("offset", String(options.offset));

  const query = params.toString();
  const url = buildApiUrl(`/agent-runs${query ? `?${query}` : ""}`, { baseUrl: API_BASE });

  const res = await fetch(url);

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body.error?.details?.user_message;
    const msg = detail || body.error?.message || `Failed to list agent runs: ${res.status}`;
    throw new AgentModeError(msg, body.error?.code, body.error?.recovery);
  }

  return jsonResponse<ListAgentRunsResponse>(res);
}

/**
 * Get events for an agent-manager run directly by run ID (no chat required).
 * Used for previewing runs before attaching them.
 */
export async function getRunEvents(
  runId: string,
  afterSequence: number = 0
): Promise<AgentEventsResponse> {
  const url = buildApiUrl(`/agent-runs/${runId}/events?after_sequence=${afterSequence}`, {
    baseUrl: API_BASE
  });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get run events: ${res.status}`);
  }

  return jsonResponse<AgentEventsResponse>(res);
}

/**
 * Attach an existing agent-manager run to a chat.
 */
export async function attachAgentRun(
  chatId: string,
  runId: string,
  taskId: string
): Promise<AgentModeResponse> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/attach`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ run_id: runId, task_id: taskId })
  });

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body.error?.details?.user_message;
    const msg = detail || body.error?.message || `Failed to attach agent run: ${res.status}`;
    throw new AgentModeError(msg, body.error?.code, body.error?.recovery);
  }

  return jsonResponse<AgentModeResponse>(res);
}
