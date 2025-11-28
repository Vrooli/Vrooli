import { resolveApiBase, buildApiUrl } from "@vrooli/api-base";

const API_BASE = resolveApiBase({ appendSuffix: true });

// Types
export interface Chat {
  id: string;
  name: string;
  preview: string;
  model: string;
  view_mode: "bubble" | "terminal";
  is_read: boolean;
  is_archived: boolean;
  is_starred: boolean;
  label_ids: string[];
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: string;
  chat_id: string;
  role: "user" | "assistant" | "system";
  content: string;
  model?: string;
  token_count?: number;
  created_at: string;
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
