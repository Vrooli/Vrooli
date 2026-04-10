/**
 * Models and usage tracking API functions.
 */
import { API_BASE, buildApiUrl, jsonResponse } from "./api-base";

// =============================================================================
// Models
// =============================================================================

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

// =============================================================================
// Usage Tracking
// =============================================================================

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
