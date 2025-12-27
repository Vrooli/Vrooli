import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

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
  active_leaf_message_id?: string; // Current branch leaf for message tree
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
  created_at: string;
}

export interface ToolCallRecord {
  id: string;
  message_id: string;
  chat_id: string;
  tool_name: string;
  arguments: string;
  result?: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  scenario_name?: string;
  external_run_id?: string;
  started_at: string;
  completed_at?: string;
  error_message?: string;
}

export interface Label {
  id: string;
  name: string;
  color: string;
  created_at: string;
}

export interface ChatWithMessages {
  chat: Chat;
  messages: Message[];
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

export async function updateChat(id: string, data: { name?: string; model?: string }): Promise<Chat> {
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

export async function addMessage(chatId: string, data: { role: string; content: string; model?: string }): Promise<Message> {
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
  type: "content" | "tool_call_start" | "tool_call_result" | "tool_calls_complete" | "error" | "warning" | "progress";
  completion_id?: string;
  content?: string;
  tool_name?: string;
  tool_id?: string;
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
}

// Chat completion with streaming
// Supports AbortController signal for cancellation on unmount or new request
export async function completeChat(
  chatId: string,
  options?: {
    stream?: boolean;
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
  }
): Promise<Message | void> {
  const stream = options?.stream ?? true;
  const url = buildApiUrl(`/chats/${chatId}/complete?stream=${stream}`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
