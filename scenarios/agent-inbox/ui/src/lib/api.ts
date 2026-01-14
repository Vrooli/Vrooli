import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Base URL without the /api/v1 suffix for resolving attachment paths
const ORIGIN_BASE = resolveApiBase({ appendSuffix: false });

/**
 * Resolve an attachment URL to work in proxy contexts.
 * The API returns paths like "/api/v1/uploads/..." which need to be
 * resolved relative to the current origin/proxy base.
 */
export function resolveAttachmentUrl(url: string | undefined): string | undefined {
  if (!url) return undefined;
  // If already absolute URL, return as-is
  if (url.startsWith("http://") || url.startsWith("https://") || url.startsWith("data:")) {
    return url;
  }
  // Resolve relative path against the origin base
  return `${ORIGIN_BASE}${url}`;
}

// Types
export interface Chat {
  id: string;
  name: string;
  preview: string;
  model: string;
  view_mode: "bubble";
  is_read: boolean;
  is_archived: boolean;
  is_starred: boolean;
  label_ids: string[];
  tools_enabled: boolean; // Whether AI can use tools in this chat
  web_search_enabled: boolean; // Default web search setting for new messages
  active_leaf_message_id?: string; // Current branch leaf for message tree
  active_template_id?: string; // Currently active template (tools remain enabled until used)
  active_template_tool_ids?: string[]; // Tool IDs suggested by the active template
  created_at: string;
  updated_at: string;
}

export interface ToolCall {
  id: string;
  type: string;
  function: {
    name: string;
    arguments: string;
  };
}

export interface Attachment {
  id: string;
  message_id?: string;
  file_name: string;
  content_type: string;
  file_size: number;
  storage_path: string;
  url?: string; // Full URL for display
  width?: number;
  height?: number;
  created_at: string;
}

export interface Message {
  id: string;
  chat_id: string;
  role: "user" | "assistant" | "system" | "tool";
  content: string;
  model?: string;
  token_count?: number;
  tool_call_id?: string;
  tool_calls?: ToolCall[];
  response_id?: string;
  finish_reason?: string;
  parent_message_id?: string; // Parent message for branching
  sibling_index: number;      // Order among siblings (0 = first)
  attachments?: Attachment[]; // Attached images/PDFs
  web_search?: boolean;       // Per-message web search override
  created_at: string;
}

export interface ToolCallRecord {
  id: string;
  message_id: string;
  chat_id: string;
  tool_name: string;
  arguments: string;
  result?: string;
  status: "pending" | "pending_approval" | "approved" | "rejected" | "running" | "completed" | "failed" | "cancelled";
  scenario_name?: string;
  external_run_id?: string;
  started_at: string;
  completed_at?: string;
  error_message?: string;
}

// Approval override for tool configurations (three-state)
export type ApprovalOverride = "" | "require" | "skip";

export interface Label {
  id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface ChatWithMessages {
  chat: Chat;
  messages: Message[];
  tool_call_records?: ToolCallRecord[];
}

// Health
export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`API health check failed: ${res.status}`);
  }

  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

// Chats
export async function fetchChats(options?: { archived?: boolean; starred?: boolean }): Promise<Chat[]> {
  const params = new URLSearchParams();
  if (options?.archived) params.set("archived", "true");
  if (options?.starred) params.set("starred", "true");

  const queryString = params.toString();
  const endpoint = queryString ? `/chats?${queryString}` : "/chats";
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch chats: ${res.status}`);
  }

  return res.json();
}

export async function fetchChat(id: string): Promise<ChatWithMessages> {
  const url = buildApiUrl(`/chats/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch chat: ${res.status}`);
  }

  return res.json();
}

export async function createChat(data?: { name?: string; model?: string; view_mode?: string }): Promise<Chat> {
  const url = buildApiUrl("/chats", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data || {})
  });

  if (!res.ok) {
    throw new Error(`Failed to create chat: ${res.status}`);
  }

  return res.json();
}

export async function updateChat(id: string, data: { name?: string; model?: string; tools_enabled?: boolean }): Promise<Chat> {
  const url = buildApiUrl(`/chats/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });

  if (!res.ok) {
    throw new Error(`Failed to update chat: ${res.status}`);
  }

  return res.json();
}

export async function deleteChat(id: string): Promise<void> {
  const url = buildApiUrl(`/chats/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete chat: ${res.status}`);
  }
}

export async function deleteAllChats(): Promise<{ deleted: number }> {
  const url = buildApiUrl("/chats", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete all chats: ${res.status}`);
  }

  return res.json();
}

// Active Template (template-to-tool linking)
export async function setActiveTemplate(
  chatId: string,
  templateId: string | null,
  toolIds: string[]
): Promise<Chat> {
  const url = buildApiUrl(`/chats/${chatId}/active-template`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      template_id: templateId ?? "",
      tool_ids: toolIds
    })
  });

  if (!res.ok) {
    throw new Error(`Failed to set active template: ${res.status}`);
  }

  return res.json();
}

// Bulk Operations
export type BulkOperation = "delete" | "archive" | "unarchive" | "mark_read" | "mark_unread" | "add_label" | "remove_label";

export interface BulkOperationResult {
  success_count: number;
  fail_count: number;
  total: number;
}

export async function bulkOperateChats(
  chatIds: string[],
  operation: BulkOperation,
  labelId?: string
): Promise<BulkOperationResult> {
  const url = buildApiUrl("/chats/bulk", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_ids: chatIds,
      operation,
      label_id: labelId
    })
  });

  if (!res.ok) {
    throw new Error(`Bulk operation failed: ${res.status}`);
  }

  return res.json();
}

// Fork conversation
export async function forkChat(chatId: string, messageId: string): Promise<Chat> {
  const url = buildApiUrl(`/chats/${chatId}/fork`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message_id: messageId })
  });

  if (!res.ok) {
    throw new Error(`Failed to fork chat: ${res.status}`);
  }

  return res.json();
}

export interface AddMessageData {
  role: string;
  content: string;
  model?: string;
  attachment_ids?: string[];
  web_search?: boolean;
  skill_ids?: string[];
}

export async function addMessage(chatId: string, data: AddMessageData): Promise<Message> {
  const url = buildApiUrl(`/chats/${chatId}/messages`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });

  if (!res.ok) {
    throw new Error(`Failed to add message: ${res.status}`);
  }

  return res.json();
}

// Message branching (ChatGPT-style regeneration)

/**
 * Regenerate an assistant message, creating a new sibling response.
 * The original response is preserved and a new alternative is generated.
 * Supports streaming via the same options as completeChat.
 */
export async function regenerateMessage(
  chatId: string,
  messageId: string,
  options?: {
    stream?: boolean;
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
  }
): Promise<Message | void> {
  const stream = options?.stream ?? true;
  const url = buildApiUrl(`/chats/${chatId}/messages/${messageId}/regenerate?stream=${stream}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    signal: options?.signal,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to regenerate message: ${errorText}`);
  }

  if (stream) {
    const reader = res.body?.getReader();
    const decoder = new TextDecoder();

    if (!reader) {
      throw new Error("Streaming not supported");
    }

    try {
      while (true) {
        if (options?.signal?.aborted) {
          reader.cancel();
          throw new DOMException("Aborted", "AbortError");
        }

        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value, { stream: true });
        const lines = text.split("\n");

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            if (data === "[DONE]") continue;

            try {
              const parsed = JSON.parse(data) as StreamingEvent;

              if (parsed.content && options?.onChunk) {
                options.onChunk(parsed.content);
              }

              if (options?.onEvent) {
                options.onEvent(parsed);
              }
            } catch {
              // Ignore parse errors for partial data
            }
          }
        }
      }
    } finally {
      try {
        reader.releaseLock();
      } catch {
        // Reader may already be released
      }
    }
  } else {
    return res.json();
  }
}

export interface EditMessageData {
  content: string;
  attachment_ids?: string[];
  web_search?: boolean;
}

/**
 * Edit a user message by creating a new sibling with updated content.
 * The original message is preserved (branch-based editing).
 * Returns the new message. The caller should trigger completion separately.
 */
export async function editMessage(
  chatId: string,
  messageId: string,
  data: EditMessageData
): Promise<Message> {
  const url = buildApiUrl(`/chats/${chatId}/messages/${messageId}/edit`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to edit message: ${errorText}`);
  }

  return res.json();
}

/**
 * Select a different branch by setting a message as the active leaf.
 * Used when navigating between alternative responses.
 */
export async function selectBranch(chatId: string, messageId: string): Promise<{ active_leaf_message_id: string }> {
  const url = buildApiUrl(`/chats/${chatId}/messages/${messageId}/select`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to select branch: ${res.status}`);
  }

  return res.json();
}

/**
 * Get all sibling messages (alternatives) for a given message.
 * Returns messages with the same parent, used for the version picker.
 */
export async function getMessageSiblings(chatId: string, messageId: string): Promise<Message[]> {
  const url = buildApiUrl(`/chats/${chatId}/messages/${messageId}/siblings`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get message siblings: ${res.status}`);
  }

  return res.json();
}

export async function toggleRead(chatId: string, value?: boolean): Promise<{ is_read: boolean }> {
  const url = buildApiUrl(`/chats/${chatId}/read`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: value !== undefined ? JSON.stringify({ value }) : "{}"
  });

  if (!res.ok) {
    throw new Error(`Failed to toggle read: ${res.status}`);
  }

  return res.json();
}

export async function toggleArchive(chatId: string, value?: boolean): Promise<{ is_archived: boolean }> {
  const url = buildApiUrl(`/chats/${chatId}/archive`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: value !== undefined ? JSON.stringify({ value }) : "{}"
  });

  if (!res.ok) {
    throw new Error(`Failed to toggle archive: ${res.status}`);
  }

  return res.json();
}

export async function toggleStar(chatId: string, value?: boolean): Promise<{ is_starred: boolean }> {
  const url = buildApiUrl(`/chats/${chatId}/star`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: value !== undefined ? JSON.stringify({ value }) : "{}"
  });

  if (!res.ok) {
    throw new Error(`Failed to toggle star: ${res.status}`);
  }

  return res.json();
}

// Labels
export async function fetchLabels(): Promise<Label[]> {
  const url = buildApiUrl("/labels", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch labels: ${res.status}`);
  }

  return res.json();
}

export async function createLabel(data: { name: string; color?: string }): Promise<Label> {
  const url = buildApiUrl("/labels", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });

  if (!res.ok) {
    throw new Error(`Failed to create label: ${res.status}`);
  }

  return res.json();
}

export async function deleteLabel(id: string): Promise<void> {
  const url = buildApiUrl(`/labels/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete label: ${res.status}`);
  }
}

export async function assignLabel(chatId: string, labelId: string): Promise<void> {
  const url = buildApiUrl(`/chats/${chatId}/labels/${labelId}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to assign label: ${res.status}`);
  }
}

export async function removeLabel(chatId: string, labelId: string): Promise<void> {
  const url = buildApiUrl(`/chats/${chatId}/labels/${labelId}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to remove label: ${res.status}`);
  }
}

// Search
export interface SearchResult {
  chat: Chat;
  message_id?: string;
  snippet?: string;
  rank: number;
  match_type: "chat_name" | "message_content";
}

export async function searchChats(query: string, limit?: number): Promise<SearchResult[]> {
  const params = new URLSearchParams({ q: query });
  if (limit) params.set("limit", limit.toString());

  const url = buildApiUrl(`/search?${params.toString()}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to search chats: ${res.status}`);
  }

  return res.json();
}

// Auto-naming
export async function autoNameChat(chatId: string): Promise<Chat> {
  const url = buildApiUrl(`/chats/${chatId}/auto-name`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to auto-name chat: ${res.status}`);
  }

  return res.json();
}

// Models
export interface ModelPricing {
  prompt: number;
  completion: number;
  request?: number;
  image?: number;
}

export interface ModelArchitecture {
  modality?: string;
  input?: string[];
  output?: string[];
}

export interface Model {
  id: string;
  name: string;
  description?: string;
  provider?: string;
  context_length?: number;
  max_completion_tokens?: number;
  pricing?: ModelPricing;
  architecture?: ModelArchitecture;
  supported_parameters?: string[];
}

export async function fetchModels(): Promise<Model[]> {
  const url = buildApiUrl("/models", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch models: ${res.status}`);
  }

  return res.json();
}

// Tools
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

  return res.json();
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

  return res.json();
}

// Streaming event types
//
// TEMPORAL FLOW: completion_id enables client-side correlation of events
// from the same completion request, helping prevent stale event handling
// when requests are cancelled or replaced.
export interface StreamingEvent {
  type: "content" | "image_generated" | "tool_call_start" | "tool_call_result" | "tool_calls_complete" | "tool_pending_approval" | "awaiting_approvals" | "error" | "warning" | "progress";
  completion_id?: string;
  content?: string;
  // Image generation event fields
  image_url?: string;
  tool_name?: string;
  tool_id?: string;
  tool_call_id?: string;
  arguments?: string;
  result?: string;
  status?: string;
  error?: string;
  continuing?: boolean;
  done?: boolean;
  // Progress event fields
  phase?: string;
  message?: string;
  // Warning/error event fields
  code?: string;
  request_id?: string;
  // Template deactivation (when a template's suggested tool is used)
  deactivate_template?: boolean;
}

// Chat completion with streaming
// Supports AbortController signal for cancellation on unmount or new request
export interface SkillPayloadForAPI {
  id: string;
  name: string;
  content: string;
  key: string;
  label: string;
  tags?: string[];
  targetToolId?: string;
}

export async function completeChat(
  chatId: string,
  options?: {
    stream?: boolean;
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
    forcedTool?: { scenario: string; toolName: string };
    skills?: SkillPayloadForAPI[];
  }
): Promise<Message | void> {
  const stream = options?.stream ?? true;
  const params = new URLSearchParams();
  params.set("stream", String(stream));
  if (options?.forcedTool) {
    params.set("force_tool", `${options.forcedTool.scenario}:${options.forcedTool.toolName}`);
  }
  const url = buildApiUrl(`/chats/${chatId}/complete?${params.toString()}`, { baseUrl: API_BASE });

  // Build request body with skills if provided
  const body = options?.skills && options.skills.length > 0
    ? JSON.stringify({ skills: options.skills })
    : undefined;

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body,
    signal: options?.signal,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Chat completion failed: ${errorText}`);
  }

  if (stream) {
    const reader = res.body?.getReader();
    const decoder = new TextDecoder();

    if (!reader) {
      throw new Error("Streaming not supported");
    }

    try {
      while (true) {
        // Check for abort before each read
        if (options?.signal?.aborted) {
          reader.cancel();
          throw new DOMException("Aborted", "AbortError");
        }

        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value, { stream: true });
        const lines = text.split("\n");

        for (const line of lines) {
          if (line.startsWith("data: ")) {
            const data = line.slice(6);
            if (data === "[DONE]") continue;

            try {
              const parsed = JSON.parse(data) as StreamingEvent;

              // Legacy callback for content chunks
              if (parsed.content && options?.onChunk) {
                options.onChunk(parsed.content);
              }

              // New event-based callback
              if (options?.onEvent) {
                options.onEvent(parsed);
              }
            } catch {
              // Ignore parse errors for partial data
            }
          }
        }
      }
    } finally {
      // Ensure reader is released on any exit path
      try {
        reader.releaseLock();
      } catch {
        // Reader may already be released
      }
    }
  } else {
    return res.json();
  }
}

// Usage tracking types
export interface UsageRecord {
  id: string;
  chat_id: string;
  message_id?: string;
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  prompt_cost: number;
  completion_cost: number;
  total_cost: number;
  created_at: string;
}

export interface ModelUsage {
  model: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  total_cost: number;
  request_count: number;
}

export interface DailyUsage {
  date: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  total_cost: number;
  request_count: number;
}

export interface UsageStats {
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_tokens: number;
  total_cost: number;
  by_model: Record<string, ModelUsage>;
  by_day?: Record<string, DailyUsage>;
}

// Usage API functions
export async function fetchUsageStats(options?: { start?: string; end?: string }): Promise<UsageStats> {
  const params = new URLSearchParams();
  if (options?.start) params.set("start", options.start);
  if (options?.end) params.set("end", options.end);

  const queryString = params.toString();
  const endpoint = queryString ? `/usage?${queryString}` : "/usage";
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch usage stats: ${res.status}`);
  }

  return res.json();
}

export async function fetchChatUsageStats(chatId: string): Promise<UsageStats> {
  const url = buildApiUrl(`/chats/${chatId}/usage`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch chat usage stats: ${res.status}`);
  }

  return res.json();
}

export async function fetchUsageRecords(options?: { chatId?: string; limit?: number; offset?: number }): Promise<UsageRecord[]> {
  const params = new URLSearchParams();
  if (options?.chatId) params.set("chat_id", options.chatId);
  if (options?.limit) params.set("limit", options.limit.toString());
  if (options?.offset) params.set("offset", options.offset.toString());

  const queryString = params.toString();
  const endpoint = queryString ? `/usage/records?${queryString}` : "/usage/records";
  const url = buildApiUrl(endpoint, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch usage records: ${res.status}`);
  }

  return res.json();
}

// Export formats
export type ExportFormat = "markdown" | "json" | "txt";

// -----------------------------------------------------------------------------
// Tool Discovery Protocol Types
// -----------------------------------------------------------------------------

export interface ScenarioInfo {
  name: string;
  version: string;
  description: string;
  base_url?: string;
}

export interface ToolParameters {
  type: string;
  properties: Record<string, ParameterSchema>;
  required?: string[];
}

export interface ParameterSchema {
  type: string;
  description?: string;
  enum?: string[];
  default?: unknown;
  items?: ParameterSchema;
  properties?: Record<string, ParameterSchema>;
  format?: string;
}

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

export interface ToolCategory {
  id: string;
  name: string;
  description?: string;
  icon?: string;
}

export interface DiscoveredTool {
  name: string;
  description: string;
  category?: string;
  parameters: ToolParameters;
  metadata: ToolMetadata;
}

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

// Export chat to file
// Triggers browser download with the specified format
export async function exportChat(chatId: string, format: ExportFormat = "markdown"): Promise<void> {
  const url = buildApiUrl(`/chats/${chatId}/export?format=${format}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Accept": "*/*" }
  });

  if (!res.ok) {
    throw new Error(`Failed to export chat: ${res.status}`);
  }

  // Get filename from Content-Disposition header or generate one
  const contentDisposition = res.headers.get("Content-Disposition");
  let filename = `chat.${format === "markdown" ? "md" : format}`;
  if (contentDisposition) {
    const match = contentDisposition.match(/filename="([^"]+)"/);
    if (match) {
      filename = match[1];
    }
  }

  // Create blob and trigger download
  const blob = await res.blob();
  const downloadUrl = window.URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = downloadUrl;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(downloadUrl);
}

// -----------------------------------------------------------------------------
// Tool Configuration API Functions
// -----------------------------------------------------------------------------

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

  return res.json();
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

  return res.json();
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
 * @param chatId - Optional chat ID (empty for global)
 * @param scenario - Scenario name
 * @param toolName - Tool name
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
 * Trigger a refresh of the tool registry cache.
 */
export async function refreshTools(): Promise<{ success: boolean; scenarios_count: number; tools_count: number }> {
  const url = buildApiUrl("/tools/refresh", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to refresh tools: ${res.status}`);
  }

  return res.json();
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

  return res.json();
}

// -----------------------------------------------------------------------------
// Manual Tool Execution API Functions
// -----------------------------------------------------------------------------

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

  return res.json();
}

// -----------------------------------------------------------------------------
// YOLO Mode & Tool Approval API Functions
// -----------------------------------------------------------------------------

/**
 * Get the current YOLO mode setting.
 */
export async function getYoloMode(): Promise<boolean> {
  const url = buildApiUrl("/settings/yolo-mode", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get YOLO mode: ${res.status}`);
  }

  const data = await res.json();
  return data.enabled;
}

/**
 * Set the YOLO mode setting.
 * @param enabled - Whether to enable YOLO mode
 */
export async function setYoloMode(enabled: boolean): Promise<void> {
  const url = buildApiUrl("/settings/yolo-mode", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled })
  });

  if (!res.ok) {
    throw new Error(`Failed to set YOLO mode: ${res.status}`);
  }
}

/**
 * Set the approval override for a tool.
 * @param scenario - Scenario name
 * @param toolName - Tool name
 * @param approvalOverride - Approval override value ("" | "require" | "skip")
 * @param chatId - Optional chat ID for chat-specific configuration
 */
export async function setToolApproval(
  scenario: string,
  toolName: string,
  approvalOverride: ApprovalOverride,
  chatId?: string
): Promise<void> {
  const url = buildApiUrl("/tools/config/approval", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      chat_id: chatId,
      scenario,
      tool_name: toolName,
      approval_override: approvalOverride
    })
  });

  if (!res.ok) {
    throw new Error(`Failed to set tool approval: ${res.status}`);
  }
}

// -----------------------------------------------------------------------------
// Pending Approvals API Functions
// -----------------------------------------------------------------------------

export interface PendingApproval {
  id: string;
  tool_name: string;
  arguments: string;
  status: string;
  started_at: string;
}

export interface ApprovalResult {
  success: boolean;
  tool_result: {
    id: string;
    tool_name: string;
    status: string;
    result?: string;
  };
  pending_approvals: PendingApproval[];
  auto_continued: boolean;
}

/**
 * Get pending tool call approvals for a chat.
 * @param chatId - Chat ID
 */
export async function getPendingApprovals(chatId: string): Promise<PendingApproval[]> {
  const url = buildApiUrl(`/chats/${chatId}/pending-approvals`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get pending approvals: ${res.status}`);
  }

  const data = await res.json();
  return data.pending_approvals;
}

/**
 * Approve a pending tool call.
 * @param toolCallId - Tool call ID
 * @param chatId - Chat ID for validation
 */
export async function approveToolCall(toolCallId: string, chatId: string): Promise<ApprovalResult> {
  const url = buildApiUrl(`/tool-calls/${toolCallId}/approve?chat_id=${chatId}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to approve tool call: ${errorText}`);
  }

  return res.json();
}

/**
 * Reject a pending tool call.
 * @param toolCallId - Tool call ID
 * @param chatId - Chat ID for validation
 * @param reason - Optional rejection reason
 */
export async function rejectToolCall(toolCallId: string, chatId: string, reason?: string): Promise<void> {
  const url = buildApiUrl(`/tool-calls/${toolCallId}/reject?chat_id=${chatId}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason })
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(`Failed to reject tool call: ${errorText}`);
  }
}

// -----------------------------------------------------------------------------
// Attachment Upload API Functions
// -----------------------------------------------------------------------------

export interface UploadResponse {
  id: string;
  file_name: string;
  content_type: string;
  file_size: number;
  storage_path: string;
  url: string;
}

/**
 * Upload a file attachment.
 * @param file - The file to upload
 * @returns Upload response with file metadata and URL
 */
export async function uploadAttachment(file: File): Promise<UploadResponse> {
  const url = buildApiUrl("/attachments/upload", { baseUrl: API_BASE });

  const formData = new FormData();
  formData.append("file", file);

  const res = await fetch(url, {
    method: "POST",
    body: formData,
  });

  if (!res.ok) {
    if (res.status === 413) {
      throw new Error("File is too large");
    }
    if (res.status === 415) {
      throw new Error("File type not supported");
    }
    throw new Error(`Failed to upload file: ${res.status}`);
  }

  return res.json();
}

// -----------------------------------------------------------------------------
// Web Search Settings API Functions
// -----------------------------------------------------------------------------

/**
 * Get the web search enabled setting for a chat.
 * @param chatId - Chat ID
 */
export async function getWebSearchEnabled(chatId: string): Promise<boolean> {
  const url = buildApiUrl(`/chats/${chatId}/settings/web-search`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get web search setting: ${res.status}`);
  }

  const data = await res.json();
  return data.enabled;
}

/**
 * Set the web search enabled setting for a chat.
 * @param chatId - Chat ID
 * @param enabled - Whether to enable web search by default
 */
export async function setWebSearchEnabled(chatId: string, enabled: boolean): Promise<void> {
  const url = buildApiUrl(`/chats/${chatId}/settings/web-search`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ enabled })
  });

  if (!res.ok) {
    throw new Error(`Failed to set web search setting: ${res.status}`);
  }
}

// -----------------------------------------------------------------------------
// Link Preview API Functions
// -----------------------------------------------------------------------------

export interface LinkPreviewData {
  title?: string;
  description?: string;
  image?: string;
  favicon?: string;
  site_name?: string;
}

/**
 * Fetch OpenGraph metadata preview for a URL.
 * @param url - The URL to fetch preview for
 * @returns Preview data or null if unavailable
 */
export async function fetchLinkPreview(url: string): Promise<LinkPreviewData | null> {
  const apiUrl = buildApiUrl(`/link-preview?url=${encodeURIComponent(url)}`, { baseUrl: API_BASE });

  const res = await fetch(apiUrl, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (res.status === 204) {
    // No content - preview unavailable
    return null;
  }

  if (!res.ok) {
    throw new Error(`Failed to fetch link preview: ${res.status}`);
  }

  return res.json();
}

// -----------------------------------------------------------------------------
// Templates API Functions
// -----------------------------------------------------------------------------

export interface TemplateVariable {
  name: string;
  label: string;
  type: "text" | "textarea" | "select";
  placeholder?: string;
  options?: string[];
  required?: boolean;
  defaultValue?: string;
}

export interface Template {
  id: string;
  name: string;
  description: string;
  icon?: string;
  modes?: string[];
  content: string;
  variables: TemplateVariable[];
  suggestedSkillIds?: string[];
  suggestedToolIds?: string[];
}

export type TemplateSource = "default" | "user" | "modified";

export interface TemplateResponse extends Template {
  source: TemplateSource;
  hasDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface TemplateListResponse {
  templates: TemplateResponse[];
  defaults_count: number;
  user_count: number;
  modified_defaults_count: number;
}

/**
 * Fetch all templates (defaults merged with user overrides).
 */
export async function fetchTemplates(): Promise<TemplateListResponse> {
  const url = buildApiUrl("/templates", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch templates: ${res.status}`);
  }

  return res.json();
}

/**
 * Fetch a single template by ID.
 * @param id - Template ID
 */
export async function fetchTemplate(id: string): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch template: ${res.status}`);
  }

  return res.json();
}

export type CreateTemplateInput = Omit<Template, "id">;

/**
 * Create a new user template.
 * @param template - Template data (id will be generated)
 */
export async function createTemplate(template: CreateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl("/templates", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(template)
  });

  if (!res.ok) {
    throw new Error(`Failed to create template: ${res.status}`);
  }

  return res.json();
}

export type UpdateTemplateInput = Partial<Omit<Template, "id">>;

/**
 * Update an existing template.
 * If it's a default template, creates a user override.
 * @param id - Template ID
 * @param updates - Fields to update
 */
export async function updateTemplate(id: string, updates: UpdateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update template: ${res.status}`);
  }

  return res.json();
}

/**
 * Delete a user template or user override.
 * @param id - Template ID
 */
export async function deleteTemplate(id: string): Promise<void> {
  const url = buildApiUrl(`/templates/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete template: ${res.status}`);
  }
}

/**
 * Reset a modified default template to its original state.
 * Deletes the user override to reveal the default.
 * @param id - Template ID
 */
export async function resetTemplate(id: string): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}/reset`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to reset template: ${res.status}`);
  }

  return res.json();
}

/**
 * Update the actual default template (not a user override).
 * This modifies the shipped default template files directly.
 * @param id - Template ID
 * @param updates - Template fields to update
 */
export async function updateDefaultTemplate(id: string, updates: UpdateTemplateInput): Promise<TemplateResponse> {
  const url = buildApiUrl(`/templates/${id}/update-default`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update default template: ${res.status}`);
  }

  return res.json();
}

/**
 * Import multiple templates from a JSON array.
 * @param templates - Array of templates to import
 */
export async function importTemplates(templates: Template[]): Promise<{ imported: number }> {
  const url = buildApiUrl("/templates/import", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(templates)
  });

  if (!res.ok) {
    throw new Error(`Failed to import templates: ${res.status}`);
  }

  return res.json();
}

/**
 * Export all user templates.
 */
export async function exportTemplates(): Promise<Template[]> {
  const url = buildApiUrl("/templates/export", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to export templates: ${res.status}`);
  }

  return res.json();
}

// ============================================================================
// Skills API
// ============================================================================

import type { Skill, SkillSource, SkillWithSource } from "@/lib/types/templates";

export interface SkillResponse extends Skill {
  source: SkillSource;
  hasDefault: boolean;
  createdAt?: string;
  updatedAt?: string;
}

export interface SkillListResponse {
  skills: SkillResponse[];
  defaults_count: number;
  user_count: number;
  modified_defaults_count: number;
}

/**
 * Fetch all skills (defaults merged with user overrides).
 */
export async function fetchSkills(): Promise<SkillListResponse> {
  const url = buildApiUrl("/skills", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch skills: ${res.status}`);
  }

  return res.json();
}

/**
 * Fetch a single skill by ID.
 * @param id - Skill ID
 */
export async function fetchSkill(id: string): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch skill: ${res.status}`);
  }

  return res.json();
}

export type CreateSkillInput = Omit<Skill, "id">;

/**
 * Create a new user skill.
 * @param skill - Skill data (id will be generated)
 */
export async function createSkill(skill: CreateSkillInput): Promise<SkillResponse> {
  const url = buildApiUrl("/skills", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(skill)
  });

  if (!res.ok) {
    throw new Error(`Failed to create skill: ${res.status}`);
  }

  return res.json();
}

export type UpdateSkillInput = Partial<Omit<Skill, "id">>;

/**
 * Update an existing skill.
 * If it's a default skill, creates a user override.
 * @param id - Skill ID
 * @param updates - Fields to update
 */
export async function updateSkill(id: string, updates: UpdateSkillInput): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update skill: ${res.status}`);
  }

  return res.json();
}

/**
 * Delete a user skill or user override.
 * @param id - Skill ID
 */
export async function deleteSkill(id: string): Promise<void> {
  const url = buildApiUrl(`/skills/${id}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete skill: ${res.status}`);
  }
}

/**
 * Reset a modified default skill to its original state.
 * Deletes the user override to reveal the default.
 * @param id - Skill ID
 */
export async function resetSkill(id: string): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}/reset`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to reset skill: ${res.status}`);
  }

  return res.json();
}

/**
 * Update the actual default skill (not a user override).
 * This modifies the shipped default skill files directly.
 * @param id - Skill ID
 * @param updates - Skill fields to update
 */
export async function updateDefaultSkill(id: string, updates: UpdateSkillInput): Promise<SkillResponse> {
  const url = buildApiUrl(`/skills/${id}/update-default`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(updates)
  });

  if (!res.ok) {
    throw new Error(`Failed to update default skill: ${res.status}`);
  }

  return res.json();
}

/**
 * Import multiple skills from a JSON array.
 * @param skills - Array of skills to import
 */
export async function importSkills(skills: Skill[]): Promise<{ imported: number }> {
  const url = buildApiUrl("/skills/import", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(skills)
  });

  if (!res.ok) {
    throw new Error(`Failed to import skills: ${res.status}`);
  }

  return res.json();
}

/**
 * Export all user skills.
 */
export async function exportSkills(): Promise<Skill[]> {
  const url = buildApiUrl("/skills/export", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to export skills: ${res.status}`);
  }

  return res.json();
}
