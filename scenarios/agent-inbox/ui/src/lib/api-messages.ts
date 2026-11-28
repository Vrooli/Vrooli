/**
 * Message operations: add, edit, regenerate, branch selection, siblings.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { Message } from "./api-types";
import type { StreamingEvent } from "./api-completion";
import { processSSEStream } from "./api-completion";

// =============================================================================
// Add Message
// =============================================================================

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

// =============================================================================
// Message Branching (ChatGPT-style regeneration)
// =============================================================================

/**
 * Streaming path return type - callbacks deliver content, no message value.
 */
type StreamCompletionResult = ReturnType<() => void>;

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
): Promise<Message | StreamCompletionResult> {
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

// =============================================================================
// Edit Message
// =============================================================================

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

// =============================================================================
// Branch Selection & Siblings
// =============================================================================

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
