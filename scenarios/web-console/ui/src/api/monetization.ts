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

export interface ManagedConnection {
  id: string;
  provider: string;
  connection_name: string;
  status: "connected" | "checking" | "needs_attention" | "disconnected" | "expired" | "insufficient_scope" | "provider_unavailable" | "revoked" | "offline" | "unknown";
  bindings?: string[];
  next_action?: string;
  supported_actions?: string[];
}

export interface ConnectionsResponse {
  connections: ManagedConnection[];
}

export const getConnections = () => request<ConnectionsResponse>("/api/v1/integrations/connections");

interface CommercialCacheEntry {
  token: string;
  value: CommercialContext;
  at: number;
}

const commercialCache = new Map<string, CommercialCacheEntry>();
const commercialInflight = new Map<string, Promise<CommercialContext>>();
const COMMERCIAL_TTL = 60_000;
const COMMERCIAL_CACHE_LIMIT = 8;

export async function getCommercialContext(placement = "integrations", capabilityId = ""): Promise<CommercialContext> {
  const token = await getAccessToken();
  if (!token) throw new Error("commercial context requires an account");
  const now = Date.now();
  const key = `${token}\u0000${placement}\u0000${capabilityId}`;
  const cached = commercialCache.get(key);
  if (cached?.token === token && now - cached.at < COMMERCIAL_TTL) return cached.value;
  const inflight = commercialInflight.get(key);
  if (inflight) return inflight;
  const params = new URLSearchParams({ placement });
  if (capabilityId) params.set("capability_id", capabilityId);
  const pending = request<CommercialContext>(`/api/v1/commercial-context?${params.toString()}`)
    .then((value) => {
      if (commercialCache.size >= COMMERCIAL_CACHE_LIMIT && !commercialCache.has(key)) {
        const oldest = commercialCache.keys().next().value;
        if (oldest) commercialCache.delete(oldest);
      }
      commercialCache.set(key, { token, value, at: Date.now() });
      return value;
    })
    .catch((error) => {
      if (cached?.token === token) return { ...cached.value, stale: true };
      throw error;
    })
    .finally(() => { commercialInflight.delete(key); });
  commercialInflight.set(key, pending);
  return pending;
}

export function _resetCommercialContextCache(): void {
  commercialCache.clear();
  commercialInflight.clear();
}
import { getAccessToken } from "../lib/auth";
import { LANDING_PAGE_URL } from "../shared/upgradeDestination";

export interface CommercialContent {
  content_id: string;
  placement: string;
  title: string;
  description: string;
  priority: string;
  eligible: boolean;
  cta_label: string;
  cta_destination: string;
  expires_at: string;
  dismissible: boolean;
  dismissed_until?: string;
}

export interface CommercialContext {
  account?: { subscription_status: string; plan_tier: string; credit_balance: number; entitlement_ids: string[]; evaluated_at: string };
  content: CommercialContent[];
  generated_at: string;
  stale_after: string;
  source: string;
  stale?: boolean;
}

/** Accept only destinations owned by the console or the configured LPBS site. */
export function safeCommercialDestination(destination: string): string | null {
  try {
    const parsed = new URL(destination, window.location.origin);
    const ownedOrigin = new URL(LANDING_PAGE_URL).origin;
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return null;
    if (parsed.origin !== window.location.origin && parsed.origin !== ownedOrigin) return null;
    return parsed.href;
  } catch {
    return null;
  }
}

/**
 * Presentation-only filtering for server-selected content. An invalid or
 * expired timestamp is hidden; this helper never grants or denies an
 * entitlement.
 */
export function isCommercialContentVisible(
  item: CommercialContent,
  placement: string,
  now = Date.now(),
): boolean {
  if (!item.eligible || item.placement !== placement) return false;
  const expiresAt = Date.parse(item.expires_at);
  if (!Number.isFinite(expiresAt) || expiresAt <= now) return false;
  if (item.dismissed_until) {
    const dismissedUntil = Date.parse(item.dismissed_until);
    if (Number.isFinite(dismissedUntil) && dismissedUntil > now) return false;
  }
  return true;
}
