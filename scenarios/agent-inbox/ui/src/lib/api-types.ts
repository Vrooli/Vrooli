/**
 * Core domain types shared across API modules.
 */

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
 * //   +-- Assistant v1 (sibling_index: 0)
 * //   +-- Assistant v2 (sibling_index: 1) <- active_leaf
 * //   +-- Assistant v3 (sibling_index: 2)
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
 * 1. `pending` -> Initial state when AI requests tool
 * 2. `pending_approval` -> Waiting for user to approve (YOLO mode off)
 * 3. `approved` -> User approved, about to execute
 * 4. `running` -> Currently executing
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

// Path validation
export interface ValidatePathResult {
  valid: boolean;
  message?: string;
}

// Export formats
export type ExportFormat = "markdown" | "json" | "txt";
