import { buildApiUrl } from "@vrooli/api-base";

export type Capability = { name: string; status: string; prerequisite?: string; next_action?: string };
export type Device = { id: string; name: string; kind: string; strategy_id: string; status: string; health_reason?: string; capabilities: Capability[] };
export type Strategy = { id: string; description: string; status: string; tiers: string[]; executable_step_kinds: string[]; capabilities: Record<string, Capability>; next_actions?: string[]; promotable: boolean };
export type Session = { id: string; device_id: string; actor: string; state: string; expires_at: string; created_at: string };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  const response = await fetch(buildApiUrl(path), { cache: "no-store", ...init, headers });
  const body: unknown = await response.json().catch(() => ({}));
  const error = typeof body === "object" && body !== null ? body as { message?: unknown; code?: unknown } : {};
  if (!response.ok) throw new Error(typeof error.message === "string" ? error.message : typeof error.code === "string" ? error.code : `Request failed (${response.status})`);
  return body as T;
}
export const listDevices = () => request<{ devices: Device[] }>("/api/v1/devices");
export const listStrategies = () => request<{ strategies: Strategy[] }>("/api/v1/strategies");
export const listSessions = () => request<{ sessions: Session[] }>("/api/v1/sessions");
export const killSession = (id: string) => request(`/api/v1/sessions/${encodeURIComponent(id)}/kill`, { method: "POST" });
export const validateFlow = (strategy_id: string, flow: unknown) => request<{ runnable: boolean; gaps: string[]; warnings: string[] }>("/api/v1/flows/validate", { method: "POST", body: JSON.stringify({ strategy_id, flow }) });
export const auditRecords = () => request<{ records: Array<{ id: string; actor: string; device_id: string; verb: string; outcome: string; created_at: string }> }>("/api/v1/evidence/audit");
