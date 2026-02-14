import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";
import { SSEParser } from "./sse";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Base URL without the /api/v1 suffix for resolving attachment paths
const ORIGIN_BASE = resolveApiBase({ appendSuffix: false });

/**
 * Type-safe wrapper around Response.json().
 * Casts the untyped Promise<any> from fetch to the expected type,
 * eliminating @typescript-eslint/no-unsafe-return warnings at each call-site.
 */
function jsonResponse<T>(res: Response): Promise<T> {
  return res.json() as Promise<T>;
}

/** Shape of error bodies returned by the API (used in error-handling blocks). */
interface ApiErrorBody {
  error?: {
    message?: string;
    code?: string;
    recovery?: string;
    details?: { user_message?: string };
  };
}

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

// =============================================================================
// Core Domain Types
// =============================================================================

/**
 * Chat represents a conversation thread with the AI.
 *
 * Chats support branching (ChatGPT-style regeneration) via the message tree
 * structure. The `active_leaf_message_id` tracks the current position in the tree.
 *
 * @example
 * // Create a new chat
 * const chat = await createChat({ name: "My Chat", model: "gpt-4" });
 *
 * @see Message - Messages within a chat
 * @see ChatWithMessages - Chat with its full message history
 */
export interface Chat {
  /** Unique identifier (UUID v4) */
  id: string;
  /** User-facing name (can be auto-generated via Ollama) */
  name: string;
  /** First ~100 chars of most recent message for list display */
  preview: string;
  /** AI model ID (e.g., "openai/gpt-4", "anthropic/claude-3-opus") */
  model: string;
  /** Display mode - currently only "bubble" is supported */
  view_mode: "bubble";
  /** Whether the chat has been read by the user */
  is_read: boolean;
  /** Whether the chat is in the archive view */
  is_archived: boolean;
  /** Whether the chat is in the starred view */
  is_starred: boolean;
  /** IDs of labels assigned to this chat */
  label_ids: string[];
  /** Whether AI can use tools in this chat (tool calling) */
  tools_enabled: boolean;
  /** Default web search setting for new messages in this chat */
  web_search_enabled: boolean;
  /** Current position in message tree (for branching/regeneration) */
  active_leaf_message_id?: string;
  /** Currently active template (suggests tools to use) */
  active_template_id?: string;
  /** Tool IDs suggested by the active template */
  active_template_tool_ids?: string[];
  /** Chat mode - "llm" for normal chat or "agent" for agent-manager integration */
  chat_mode: "llm" | "agent";
  /** Agent run ID when in agent mode */
  agent_run_id?: string;
  /** Agent task ID when in agent mode */
  agent_task_id?: string;
  /** ISO 8601 timestamp of creation */
  created_at: string;
  /** ISO 8601 timestamp of last modification */
  updated_at: string;
}

/**
 * ToolCall represents an AI-requested function call.
 *
 * When the AI model decides to use a tool, it returns a ToolCall structure
 * that specifies which function to invoke and with what arguments.
 *
 * @example
 * // ToolCall from AI response
 * {
 *   id: "call_abc123",
 *   type: "function",
 *   function: {
 *     name: "run-agent",
 *     arguments: '{"prompt": "Write a poem"}'
 *   }
 * }
 */
export interface ToolCall {
  /** Unique identifier for this tool call (format: "call_xxx") */
  id: string;
  /** Always "function" for OpenAI-compatible tool calls */
  type: string;
  /** The function being called */
  function: {
    /** Name of the tool/function to invoke */
    name: string;
    /** JSON-encoded arguments for the function */
    arguments: string;
  };
}

/**
 * Attachment represents a file attached to a message.
 *
 * Attachments can be images (for multimodal AI) or PDFs. They are stored
 * on the server and referenced by URL.
 *
 * @see uploadAttachment - Upload a new attachment
 * @see resolveAttachmentUrl - Convert storage path to full URL
 */
export interface Attachment {
  /** Unique identifier (UUID v4) */
  id: string;
  /** Message this attachment belongs to (set after linking) */
  message_id?: string;
  /** Original filename */
  file_name: string;
  /** MIME type (e.g., "image/png", "application/pdf") */
  content_type: string;
  /** File size in bytes */
  file_size: number;
  /** Server storage path (relative) */
  storage_path: string;
  /** Full URL for display (resolved via API base) */
  url?: string;
  /** Image width in pixels (images only) */
  width?: number;
  /** Image height in pixels (images only) */
  height?: number;
  /** ISO 8601 timestamp of upload */
  created_at: string;
}

/**
 * Message represents a single message in a chat conversation.
 *
 * Messages form a tree structure for branching support:
 * - `parent_message_id` links to the parent message
 * - `sibling_index` indicates order among alternatives (regenerations)
 * - The active branch is tracked via `Chat.active_leaf_message_id`
 *
 * Message roles:
 * - `user`: Human input
 * - `assistant`: AI response
 * - `system`: System prompts (usually first message)
 * - `tool`: Tool execution results
 *
 * @example
 * // Message tree structure (regeneration example)
 * // User message (parent)
 * //   ├── Assistant v1 (sibling_index: 0)
 * //   ├── Assistant v2 (sibling_index: 1) ← active_leaf
 * //   └── Assistant v3 (sibling_index: 2)
 *
 * @see regenerateMessage - Create alternative responses
 * @see selectBranch - Navigate between alternatives
 */
export interface Message {
  /** Unique identifier (UUID v4) */
  id: string;
  /** Parent chat ID */
  chat_id: string;
  /** Message author type */
  role: "user" | "assistant" | "system" | "tool";
  /** Message text content */
  content: string;
  /** AI model used (assistant messages only) */
  model?: string;
  /** Token count for context window management */
  token_count?: number;
  /** For tool messages: which tool call this responds to */
  tool_call_id?: string;
  /** For assistant messages: tool calls requested by AI */
  tool_calls?: ToolCall[];
  /** OpenRouter response ID for tracking */
  response_id?: string;
  /** Why the AI stopped: "stop", "tool_calls", "length" */
  finish_reason?: string;
  /** Parent message ID (null for root/system message) */
  parent_message_id?: string;
  /** Order among siblings (0 = original, 1+ = regenerations) */
  sibling_index: number;
  /** File attachments (images, PDFs) */
  attachments?: Attachment[];
  /** Per-message web search override (null = use chat default) */
  web_search?: boolean;
  /** ISO 8601 timestamp of creation */
  created_at: string;
}

/**
 * ToolCallRecord tracks the execution state of a tool call.
 *
 * Tool calls progress through states:
 * 1. `pending` → Initial state when AI requests tool
 * 2. `pending_approval` → Waiting for user to approve (YOLO mode off)
 * 3. `approved` → User approved, about to execute
 * 4. `running` → Currently executing
 * 5. Terminal: `completed` | `failed` | `cancelled` | `rejected`
 *
 * For async tools (long-running operations):
 * - `external_run_id` tracks the operation in the external service
 * - Use `useAsyncStatus` hook to poll for progress
 *
 * @see approveToolCall - Approve a pending tool call
 * @see rejectToolCall - Reject a pending tool call
 */
export interface ToolCallRecord {
  /** Tool call ID from AI (format: "call_xxx") */
  id: string;
  /** Assistant message that made this tool call */
  message_id: string;
  /** Parent chat ID */
  chat_id: string;
  /** Name of the tool being called */
  tool_name: string;
  /** JSON-encoded function arguments */
  arguments: string;
  /** JSON-encoded execution result */
  result?: string;
  /** Current execution state */
  status: "pending" | "pending_approval" | "approved" | "rejected" | "running" | "completed" | "failed" | "cancelled";
  /** Scenario that provides this tool */
  scenario_name?: string;
  /** External operation ID (for async tools) */
  external_run_id?: string;
  /** ISO 8601 timestamp of start */
  started_at: string;
  /** ISO 8601 timestamp of completion */
  completed_at?: string;
  /** Error message if failed */
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

  return jsonResponse<Chat[]>(res);
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

  return jsonResponse<ChatWithMessages>(res);
}

export async function createChat(data?: { name?: string; model?: string; view_mode?: string; chat_mode?: "llm" | "agent" }): Promise<Chat> {
  const url = buildApiUrl("/chats", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data || {})
  });

  if (!res.ok) {
    throw new Error(`Failed to create chat: ${res.status}`);
  }

  return jsonResponse<Chat>(res);
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

  return jsonResponse<Chat>(res);
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

  return jsonResponse<{ deleted: number }>(res);
}

export async function deleteArchivedChats(): Promise<{ deleted: number }> {
  const url = buildApiUrl("/chats/archived", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to delete archived chats: ${res.status}`);
  }

  return jsonResponse<{ deleted: number }>(res);
}

export async function markAllChatsAsRead(): Promise<{ updated: number }> {
  const url = buildApiUrl("/chats/mark-all-read", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to mark all chats as read: ${res.status}`);
  }

  return jsonResponse<{ updated: number }>(res);
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

  return jsonResponse<Chat>(res);
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

  return jsonResponse<BulkOperationResult>(res);
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

  return jsonResponse<Chat>(res);
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

  return jsonResponse<Message>(res);
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
    if (!reader) {
      throw new Error("Streaming not supported");
    }
    await processSSEStream(reader, options);
  } else {
    return jsonResponse<Message>(res);
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

  return jsonResponse<Message>(res);
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

  return jsonResponse<{ active_leaf_message_id: string }>(res);
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

  return jsonResponse<Message[]>(res);
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

  return jsonResponse<{ is_read: boolean }>(res);
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

  return jsonResponse<{ is_archived: boolean }>(res);
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

  return jsonResponse<{ is_starred: boolean }>(res);
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

  return jsonResponse<Label[]>(res);
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

  return jsonResponse<Label>(res);
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

  return jsonResponse<SearchResult[]>(res);
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

  return jsonResponse<Chat>(res);
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

  return jsonResponse<Model[]>(res);
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
// Streaming Event Types
// =============================================================================

/**
 * StreamingEvent represents a Server-Sent Event (SSE) from the completion endpoint.
 *
 * Events are received during streaming completions and provide real-time updates:
 * - `content` - Text chunks as they're generated
 * - `image_generated` - AI-generated images (multimodal models)
 * - `tool_call_start/result` - Tool execution lifecycle
 * - `tool_pending_approval` - Tool requires user approval
 * - `awaiting_approvals` - Stream paused waiting for approvals
 * - `error/warning` - Issues during completion
 * - `progress` - Status updates during long operations
 *
 * TEMPORAL FLOW: `completion_id` enables client-side correlation of events
 * from the same completion request, helping prevent stale event handling
 * when requests are cancelled or replaced.
 *
 * @see completeChat - Main streaming completion function
 * @see useCompletion - React hook for managing streaming state
 * @see docs/SEAMS.md - Full protocol specification
 */
const STREAMING_EVENT_TYPES = new Set([
  "content", "image_generated", "tool_call_start", "tool_call_result",
  "tool_calls_complete", "tool_pending_approval", "awaiting_approvals",
  "error", "warning", "progress",
]);

function isStreamingEvent(v: unknown): v is StreamingEvent {
  return typeof v === 'object' && v !== null
    && typeof (v as Record<string, unknown>).type === 'string'
    && STREAMING_EVENT_TYPES.has((v as Record<string, unknown>).type as string);
}

export interface StreamingEvent {
  /** Event type discriminator */
  type: "content" | "image_generated" | "tool_call_start" | "tool_call_result" | "tool_calls_complete" | "tool_pending_approval" | "awaiting_approvals" | "error" | "warning" | "progress";
  /** Unique ID for this completion request (for stale event filtering) */
  completion_id?: string;
  /** Text content chunk (type: "content") */
  content?: string;
  /** Generated image URL (type: "image_generated") */
  image_url?: string;
  /** Tool name (type: "tool_call_start", "tool_call_result", "tool_pending_approval") */
  tool_name?: string;
  /** Tool ID (legacy field, prefer tool_call_id) */
  tool_id?: string;
  /** Tool call ID (type: "tool_pending_approval", "tool_call_result") */
  tool_call_id?: string;
  /** JSON-encoded tool arguments */
  arguments?: string;
  /** JSON-encoded tool result */
  result?: string;
  /** Tool execution status ("completed", "failed") */
  status?: string;
  /** Error message (type: "error", "tool_call_result" with failure) */
  error?: string;
  /** Whether auto-continue is happening (type: "tool_calls_complete") */
  continuing?: boolean;
  /** Whether streaming is complete */
  done?: boolean;
  /** Current phase (type: "progress") */
  phase?: string;
  /** Status message (type: "progress", "warning") */
  message?: string;
  /** Error/warning code (type: "error", "warning") */
  code?: string;
  /** Server request ID for debugging */
  request_id?: string;
  /** Signal to deactivate active template (type: "tool_call_result") */
  deactivate_template?: boolean;
}

/**
 * Process an SSE stream with proper buffering.
 *
 * This uses SSEParser to handle events that may be split across chunk boundaries,
 * preventing data loss that occurs with naive line-based parsing.
 *
 * @param reader - ReadableStream reader
 * @param options - Callbacks for content chunks and events
 * @internal
 */
async function processSSEStream(
  reader: ReadableStreamDefaultReader<Uint8Array>,
  options?: {
    onChunk?: (content: string) => void;
    onEvent?: (event: StreamingEvent) => void;
    signal?: AbortSignal;
  }
): Promise<void> {
  const decoder = new TextDecoder();
  const parser = new SSEParser({
    onEvent: (sseEvent) => {
      // Skip [DONE] sentinel
      if (sseEvent.data === "[DONE]") return;

      try {
        const parsed: unknown = JSON.parse(sseEvent.data);
        if (!isStreamingEvent(parsed)) {
          console.warn("[SSE] Unexpected event shape:", JSON.stringify(parsed).slice(0, 200));
          return;
        }

        // Legacy callback for content chunks
        if (parsed.content && options?.onChunk) {
          options.onChunk(parsed.content);
        }

        // Event-based callback
        if (options?.onEvent) {
          options.onEvent(parsed);
        }
      } catch {
        // Log parse errors for debugging but don't crash the stream
        console.warn(`Failed to parse SSE event data: ${sseEvent.data.slice(0, 100)}...`);
      }
    },
    onError: (error, rawData) => {
      console.error("SSE parse error:", error.message, rawData.slice(0, 100));
    },
  });

  try {
    while (true) {
      // Check for abort before each read
      if (options?.signal?.aborted) {
        reader.cancel();
        throw new DOMException("Aborted", "AbortError");
      }

      const { done, value } = await reader.read();
      if (done) {
        // Flush any remaining buffered data
        parser.flush();
        break;
      }

      // Process chunk with buffered parser
      parser.processChunk(decoder.decode(value, { stream: true }));
    }
  } finally {
    // Ensure reader is released on any exit path
    try {
      reader.releaseLock();
    } catch {
      // Reader may already be released
    }
  }
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
    if (!reader) {
      throw new Error("Streaming not supported");
    }
    await processSSEStream(reader, options);
  } else {
    return jsonResponse<Message>(res);
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

  return jsonResponse<UsageStats>(res);
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

  return jsonResponse<UsageStats>(res);
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

  return jsonResponse<UsageRecord[]>(res);
}

// Export formats
export type ExportFormat = "markdown" | "json" | "txt";

// -----------------------------------------------------------------------------
// Tool Discovery Protocol Types (proto-derived, see proto-contracts.ts)
// -----------------------------------------------------------------------------

export type {
  ScenarioInfo,
  ToolParameters,
  ParameterSchema,
  ToolMetadata,
  ToolCategory,
  DiscoveredTool,
} from "./proto-contracts";

import type { ScenarioInfo, ToolCategory, DiscoveredTool } from "./proto-contracts";

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
    const extractedFilename = match?.[1];
    if (extractedFilename) {
      filename = extractedFilename;
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

  return jsonResponse<ManualToolExecuteResponse>(res);
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

  const data = await jsonResponse<{ enabled: boolean }>(res);
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

  const data = await jsonResponse<{ pending_approvals: PendingApproval[] }>(res);
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

  return jsonResponse<ApprovalResult>(res);
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

  return jsonResponse<UploadResponse>(res);
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

  const data = await jsonResponse<{ enabled: boolean }>(res);
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

  return jsonResponse<LinkPreviewData>(res);
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
  draft?: boolean;
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

  return jsonResponse<TemplateListResponse>(res);
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

  return jsonResponse<TemplateResponse>(res);
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

  return jsonResponse<TemplateResponse>(res);
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

  return jsonResponse<TemplateResponse>(res);
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

  return jsonResponse<TemplateResponse>(res);
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

  return jsonResponse<TemplateResponse>(res);
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

  return jsonResponse<{ imported: number }>(res);
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

  return jsonResponse<Template[]>(res);
}

// ============================================================================
// Skills API
// ============================================================================

import type { Skill } from "@/lib/types/templates";

export interface SkillResponse extends Skill {
  createdAt?: string;
  updatedAt?: string;
}

export interface SkillListResponse {
  skills: SkillResponse[];
  count: number;
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

  return jsonResponse<SkillListResponse>(res);
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

  return jsonResponse<SkillResponse>(res);
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

  return jsonResponse<SkillResponse>(res);
}

export type UpdateSkillInput = Partial<Omit<Skill, "id">>;

/**
 * Update an existing skill via prompt-manager.
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

  return jsonResponse<SkillResponse>(res);
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

  return jsonResponse<{ imported: number }>(res);
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

  return jsonResponse<Skill[]>(res);
}

// ============================================================================
// Skill Suggestion API
// ============================================================================

export interface SuggestedSkill {
  id: string;
  name: string;
  description: string;
  tags?: string[];
  modes?: string[];
  score: number;
  scorePercent: number;
}

export interface SkillSuggestResponse {
  suggestions: SuggestedSkill[];
  queryCount: number;
  method: string;
}

/**
 * Fetch AI-powered skill suggestions based on conversation context.
 * Returns empty suggestions on any error (graceful degradation).
 */
export async function fetchSkillSuggestions(params: {
  inputText?: string;
  chatId?: string;
  excludeSkillIds?: string[];
}): Promise<SkillSuggestResponse> {
  const url = buildApiUrl("/skills/suggest", { baseUrl: API_BASE });

  try {
    const res = await fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        inputText: params.inputText,
        chatId: params.chatId,
        excludeSkillIds: params.excludeSkillIds,
      }),
      signal: params.inputText ? AbortSignal.timeout(20000) : undefined,
    });

    if (!res.ok) {
      return { suggestions: [], queryCount: 0, method: "error" };
    }

    return jsonResponse<SkillSuggestResponse>(res);
  } catch {
    return { suggestions: [], queryCount: 0, method: "error" };
  }
}

/**
 * Sync status response from the server.
 */
export interface SyncStatus {
  success: boolean;
  skillCount: number;
  localCount: number;
  hash: string;
  error?: string;
}

/**
 * Trigger an immediate sync of skills from prompt-manager.
 * This fetches the latest skills and updates the local cache.
 */
export async function syncSkills(): Promise<SyncStatus> {
  const url = buildApiUrl("/skills/sync", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" }
  });

  if (!res.ok) {
    throw new Error(`Failed to sync skills: ${res.status}`);
  }

  return jsonResponse<SyncStatus>(res);
}

// =============================================================================
// Agent Mode Types & API
// =============================================================================

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
  /** Known types: message, tool_call, tool_result, status, error, log, metric, artifact, message_deleted.
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
  // Raw data for generic display of unrecognized or rich event types
  raw_data?: string;
}

/** Response from getting agent events */
export interface AgentEventsResponse {
  events: AgentEvent[];
  run_id: string;
}

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
    const detail = body?.error?.details?.user_message;
    const message = detail || body?.error?.message || `Failed to start agent mode: ${res.status}`;
    throw new AgentModeError(message, body?.error?.code, body?.error?.recovery);
  }

  return jsonResponse<AgentModeResponse>(res);
}

/**
 * Send a message in agent mode.
 * Continues the existing agent run with a follow-up message.
 */
export async function sendAgentMessage(
  chatId: string,
  message: string
): Promise<{ success: boolean; run_id: string }> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/message`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ message })
  });

  if (!res.ok) {
    const body: ApiErrorBody = await (res.json() as Promise<ApiErrorBody>).catch(() => ({ error: { message: res.statusText } }));
    const detail = body?.error?.details?.user_message;
    const msg = detail || body?.error?.message || `Failed to send agent message: ${res.status}`;
    throw new AgentModeError(msg, body?.error?.code, body?.error?.recovery);
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
  afterSequence: number = 0
): Promise<AgentEventsResponse> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/events?after_sequence=${afterSequence}`, {
    baseUrl: API_BASE
  });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get agent events: ${res.status}`);
  }

  return jsonResponse<AgentEventsResponse>(res);
}

/**
 * Get the current status of an agent chat.
 */
export async function getAgentStatus(chatId: string): Promise<AgentModeStatus> {
  const url = buildApiUrl(`/chats/${chatId}/agent-mode/status`, { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
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
    const detail = body?.error?.details?.user_message;
    const msg = detail || body?.error?.message || `Failed to stop agent: ${res.status}`;
    throw new AgentModeError(msg, body?.error?.code, body?.error?.recovery);
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
    const detail = body?.error?.details?.user_message;
    const msg = detail || body?.error?.message || `Failed to clear agent mode: ${res.status}`;
    throw new AgentModeError(msg, body?.error?.code, body?.error?.recovery);
  }

  return jsonResponse<{ success: boolean; chat_mode: "llm" }>(res);
}
