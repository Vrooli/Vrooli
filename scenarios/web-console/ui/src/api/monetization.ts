export interface SubscriptionSessionStatus {
  configured: boolean;
  provider_state?: string;
  provider_detail?: string;
}
export interface SubscriptionSummary extends SubscriptionSessionStatus {
  status?: string;
  plan_tier?: string;
  credits?: unknown;
  pending_sync?: number;
  not_after?: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers({ "Content-Type": "application/json", ...(init?.headers ?? {}) });
  const accessToken = await getAccessToken();
  if (accessToken && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${accessToken}`);
  const response = await fetch(path, { ...init, headers });
  if (!response.ok) throw new Error(`Request failed (${response.status}): ${await response.text()}`);
  return response.status === 204 ? ({} as T) : response.json() as Promise<T>;
}

export const getSubscriptionSession = () => request<SubscriptionSessionStatus>("/api/v1/auth/subscription/session");
export const getSubscriptionSummary = async (): Promise<SubscriptionSummary> => {
  try {
    return await request<SubscriptionSummary>("/api/v1/auth/subscription/summary");
  } catch (error) {
    // The API intentionally refuses an unsigned summary request. The account
    // surface still needs to render a truthful signed-out state, so only that
    // specific response is normalized; authority and transport failures stay
    // visible to the caller.
    if (error instanceof Error && error.message.startsWith("Request failed (401)")) {
      return { configured: false, status: "signed_out", plan_tier: "free", pending_sync: 0 };
    }
    throw error;
  }
};
export const provisionSubscriptionSession = (refreshToken: string) => request("/api/v1/auth/subscription/session", { method: "POST", body: JSON.stringify({ refresh_token: refreshToken }) });
export const deleteSubscriptionSession = () => request("/api/v1/auth/subscription/session", { method: "DELETE" });
export const provisionOpenRouterKey = (value: string) => request("/api/v1/credentials/provision", { method: "POST", body: JSON.stringify({ identity: "vrooli/openrouter", field: "api-key", value }) });
export const removeOpenRouterKey = () => request("/api/v1/credentials/provision", { method: "DELETE", body: JSON.stringify({ identity: "vrooli/openrouter", field: "api-key" }) });
export const testOpenRouterKey = () => request<{ valid: boolean; source: string; checked_at: string }>("/api/v1/credentials/test", { method: "POST" });
import { getAccessToken } from "../lib/auth";
