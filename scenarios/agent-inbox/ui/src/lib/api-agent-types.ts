/**
 * Agent mode types and helper functions.
 */

/**
 * Structured error for agent mode operations.
 * Carries the machine-readable error code and recovery hint from the API,
 * enabling the UI to show targeted recovery guidance.
 */
export class AgentModeError extends Error {
  /** Machine-readable error code (e.g., "D008", "V012") */
  code?: string;
  /** Suggested recovery action (e.g., "check_dependency", "correct_input") */
  recovery?: string;

  constructor(message: string, code?: string, recovery?: string) {
    super(message);
    this.name = "AgentModeError";
    this.code = code;
    this.recovery = recovery;
  }
}

/** Runner types available for agent mode */
export type RunnerType = "claude-code" | "codex" | "opencode";

/** Run status for agent runs */
export type AgentRunStatus =
  | "pending"
  | "starting"
  | "running"
  | "needs_review"
  | "complete"
  | "failed"
  | "cancelled";

/** Configuration for starting an agent chat session */
export interface AgentChatConfig {
  /** Initial message to send to the agent */
  message: string;
  /** Runner to use (claude-code, codex, opencode) */
  runner_type: RunnerType;
  /** Directory where the agent will operate */
  project_path: string;
  /** Optional model override */
  model?: string;
  /** Optional max turns limit */
  max_turns?: number;
}

/** Response from starting agent mode */
export interface AgentModeResponse {
  chat_id: string;
  task_id: string;
  run_id: string;
  session_id?: string;
}

/** Agent mode status response */
export interface AgentModeStatus {
  chat_mode: "llm" | "agent";
  is_agent: boolean;
  task_id?: string;
  run_id?: string;
  status?: AgentRunStatus;
  phase?: string;
  progress_percent?: number;
  session_id?: string;
  error_msg?: string;
  error?: string;
}

/** Translated event from agent-manager */
export interface AgentEvent {
  id: string;
  /** Known types: message, tool_call, tool_result, status, error, log, metric, artifact, message_deleted, compaction.
   *  Unknown types from agent-manager are also passed through. */
  type: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  timestamp: string;
  sequence: number;
  // Tool fields
  tool_name?: string;
  /** Correlation ID linking a tool_call event to its tool_result event. */
  tool_call_id?: string;
  tool_input?: string;
  tool_output?: string;
  tool_success?: boolean;
  // Status fields
  run_status?: AgentRunStatus;
  phase?: string;
  progress?: number;
  // Compaction fields
  compaction_trigger?: "manual" | "auto";
  compaction_focus?: string;
  compaction_messages_compacted?: number;
  compaction_tokens_before?: number;
  compaction_tokens_after?: number;
  compaction_original_command?: string;
  // Raw data for generic display of unrecognized or rich event types
  raw_data?: string;
}

/** Type guard for compaction events */
export function isCompactionEvent(event: AgentEvent): boolean {
  return event.type === "compaction";
}

/** Calculate token reduction percentage */
export function getCompactionReduction(event: AgentEvent): number | null {
  if (!isCompactionEvent(event)) return null;
  if (!event.compaction_tokens_before || event.compaction_tokens_before === 0) return null;

  const before = event.compaction_tokens_before;
  const after = event.compaction_tokens_after ?? 0;
  return Math.round(((before - after) / before) * 100);
}

/** Response from getting agent events */
export interface AgentEventsResponse {
  events: AgentEvent[];
  run_id: string;
}

/** Summary of an agent run for list views */
export interface AgentRunSummary {
  run_id: string;
  task_id: string;
  tag?: string;
  status: AgentRunStatus;
  phase?: string;
  progress_percent: number;
  created_at: string;
  updated_at: string;
}

/** Paginated response from listing agent runs */
export interface ListAgentRunsResponse {
  runs: AgentRunSummary[];
  total: number;
  has_more: boolean;
}

/** Options for filtering/paginating agent runs */
export interface ListAgentRunsOptions {
  status?: string;
  tag_prefix?: string;
  limit?: number;
  offset?: number;
}
