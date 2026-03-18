/**
 * Miscellaneous API: labels, search, export, health, path validation, auto-naming.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";
import type {
  Chat,
  Label,
  ExportFormat,
  ValidatePathResult,
} from "./api-types";

// =============================================================================
// Labels
// =============================================================================

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

// =============================================================================
// Search
// =============================================================================

export interface SearchResult {
  chat: Chat;
  message_id?: string;
  snippet?: string;
  match_start?: number;
  match_end?: number;
  rank: number;
  match_type: "chat_name" | "message_content";
}

export interface ContentSearchOptions {
  caseSensitive?: boolean;
  wholeWord?: boolean;
  regex?: boolean;
}

export async function searchChats(
  query: string,
  limit?: number,
  perChat?: number,
  options?: ContentSearchOptions,
): Promise<SearchResult[]> {
  const params = new URLSearchParams({ q: query });
  if (limit) params.set("limit", limit.toString());
  if (perChat && perChat > 1) params.set("per_chat", perChat.toString());
  if (options?.caseSensitive) params.set("case_sensitive", "true");
  if (options?.wholeWord) params.set("whole_word", "true");
  if (options?.regex) params.set("regex", "true");

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

// =============================================================================
// Auto-naming
// =============================================================================

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

// =============================================================================
// Export
// =============================================================================

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

// =============================================================================
// Health & Path Validation
// =============================================================================

export async function validatePath(path: string): Promise<ValidatePathResult> {
  const url = buildApiUrl("/validate-path", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ path }),
  });

  if (!res.ok) {
    throw new Error(`Path validation failed: ${res.status}`);
  }

  return jsonResponse<ValidatePathResult>(res);
}

export async function fetchProjectRoot(): Promise<string> {
  const url = buildApiUrl("/project-root", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`Failed to fetch project root: ${res.status}`);
  }

  const data = await jsonResponse<{ project_root: string }>(res);
  return data.project_root;
}

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
