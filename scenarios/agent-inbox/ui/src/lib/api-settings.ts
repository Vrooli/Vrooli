/**
 * Settings API: YOLO mode, web search, suggestions settings, and link preview.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";

// =============================================================================
// YOLO Mode
// =============================================================================

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

// =============================================================================
// Suggestions Settings
// =============================================================================

export interface SuggestionsAutoSuggestConfig {
  enabled: boolean;
  debounceMs: number;
  throttleMs: number;
  minInputLength: number;
  minScorePercent: number;
  maxSuggestions: number;
}

export interface SuggestionsSettingsResponse {
  autoSuggest: SuggestionsAutoSuggestConfig;
}

/**
 * Get server-backed suggestions settings.
 */
export async function getSuggestionsSettings(): Promise<SuggestionsSettingsResponse> {
  const url = buildApiUrl("/settings/suggestions", { baseUrl: API_BASE });

  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });

  if (!res.ok) {
    throw new Error(`Failed to get suggestions settings: ${res.status}`);
  }

  return jsonResponse<SuggestionsSettingsResponse>(res);
}

/**
 * Update server-backed suggestions settings.
 */
export async function setSuggestionsSettings(
  settings: SuggestionsSettingsResponse
): Promise<SuggestionsSettingsResponse> {
  const url = buildApiUrl("/settings/suggestions", { baseUrl: API_BASE });

  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings)
  });

  if (!res.ok) {
    const body = await res.text();
    throw new Error(body || `Failed to set suggestions settings: ${res.status}`);
  }

  return jsonResponse<SuggestionsSettingsResponse>(res);
}

// =============================================================================
// Web Search Settings
// =============================================================================

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

// =============================================================================
// Link Preview
// =============================================================================

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
