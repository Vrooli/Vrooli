/**
 * Chat CRUD, toggles, and bulk operations.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type { Chat, ChatWithMessages } from "./api-types";

// =============================================================================
// Chat CRUD
// =============================================================================

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

// =============================================================================
// Bulk Operations
// =============================================================================

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

// =============================================================================
// Chat Toggles
// =============================================================================

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
