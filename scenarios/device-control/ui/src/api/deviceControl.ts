import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE } from "./client";

export type Capability = { name: string; status: string; prerequisite?: string; next_action?: string; reason?: string; state_class?: string };
export type DeviceTransport = { strategy_id: string; name: string; endpoint?: string; health?: string; health_reason?: string; capabilities?: Record<string, Capability>; properties?: PropertyDescriptor[]; observed_at?: string };
export type PropertyDescriptor = { name: string; value_type: string; writable: boolean; minimum?: number; maximum?: number; enumeration?: string[] };
export type IdentityClaim = { kind: string; value: string; strategy_id: string; evidence: string };
export type Device = { id: string; name: string; kind: string; onboarding_kind?: string; identity_key?: string; claims?: IdentityClaim[]; identity_reason?: string; serial?: string; model?: string; endpoint?: string; os_version?: string; host_node_id?: string; transport?: string; strategy_id: string; status: string; health?: string; health_reason?: string; capabilities: Capability[]; properties?: PropertyDescriptor[]; transports?: DeviceTransport[] };
export type DeviceState = { foreground_package?: string; screen_state?: string; lock_state?: string; orientation?: string; auto_rotate?: boolean; battery_level?: number; charging?: boolean; thermal_status?: string; display_width?: number; display_height?: number; display_density?: number; properties?: Record<string, { value?: unknown; status: string; reason?: string; transport?: string }>; unavailable?: Record<string, string> };
export type Strategy = { id: string; description: string; status: string; reason?: string; supported_host_os?: string[]; tiers: string[]; executable_step_kinds: string[]; capabilities: Record<string, Capability>; next_actions?: string[]; promotable: boolean };
export type Session = { id: string; device_id: string; actor: string; state: string; lease_token?: string; expires_at: string; created_at: string };
export type OnboardingRung = { id: string; prerequisite?: string; owner?: string; status: string; next_action: string };
export type OnboardingReport = { kind?: string; rungs: OnboardingRung[]; first_next_action: string };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  const response = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { cache: "no-store", ...init, headers });
  const body: unknown = await response.json().catch(() => ({}));
  const error = typeof body === "object" && body !== null ? body as { message?: unknown; code?: unknown } : {};
  if (!response.ok) throw new Error(typeof error.message === "string" ? error.message : typeof error.code === "string" ? error.code : `Request failed (${response.status})`);
  return body as T;
}
export const listDevices = () => request<{ devices: Device[] }>("/api/v1/devices");
export const listStrategies = () => request<{ strategies: Strategy[] }>("/api/v1/strategies");
export const listSessions = () => request<{ sessions: Session[] }>("/api/v1/sessions");
export const killSession = (id: string) => request(`/api/v1/sessions/${encodeURIComponent(id)}/kill`, { method: "POST" });
export const releaseSession = (id: string) => request<{ session: Session }>(`/api/v1/sessions/${encodeURIComponent(id)}/release`, { method: "POST" });
export const acquireSession = (device_id: string, actor: string) => request<{ session: Session }>("/api/v1/sessions/acquire", { method: "POST", body: JSON.stringify({ device_id, actor, ttl_seconds: 300 }) });
export const connectDevice = (kind: string) => request<OnboardingReport>("/api/v1/devices/connect", { method: "POST", body: JSON.stringify({ kind }) });
export const describeDevice = (id: string) => request<{ device: Device }>(`/api/v1/devices/${encodeURIComponent(id)}`);
export const readDeviceState = (id: string) => request<DeviceState>(`/api/v1/devices/${encodeURIComponent(id)}/state`);
export const actuateDevice = (id: string, body: { actor: string; lease_token: string; key?: string; text?: string; media?: string; property?: string; value?: unknown; transport?: string; repeat?: number }) => request<{ audit: { causation_id: string }; interactive: boolean; evidence_backed: boolean }>(`/api/v1/devices/${encodeURIComponent(id)}/actuate`, { method: "POST", body: JSON.stringify(body) });
export const startPairing = (id: string) => request<{ pairing_started: boolean; pairing_id: string }>(`/api/v1/devices/${encodeURIComponent(id)}/pair/start`, { method: "POST", body: "{}" });
export const completePairing = (id: string, pairing_id: string, pin: string) => request<{ paired: boolean }>(`/api/v1/devices/${encodeURIComponent(id)}/pair/complete`, { method: "POST", body: JSON.stringify({ pairing_id, pin }) });
export const pairDevice = (id: string, pin: string) => request<{ paired: boolean }>(`/api/v1/devices/${encodeURIComponent(id)}/pair`, { method: "POST", body: JSON.stringify({ pin }) });
export const discoverDevices = () => request<{ services: Array<{ strategy_id: string; transport: string; service?: string; id: string; name: string; model?: string; endpoint: string; address?: string; port?: number; txt?: Record<string, string>; identity_key?: string; paired?: boolean; pairing_available?: boolean }>; health?: string; reason?: string }>("/api/v1/devices/discover");
export const watchDeviceEvents = (id: string, onEvent: (event: { device_id: string; attribute: string; old_value?: unknown; new_value?: unknown; causation_id: string }) => void) => { const source = new EventSource(buildApiUrl(`/api/v1/devices/${encodeURIComponent(id)}/events`, { baseUrl: API_BASE })); source.onmessage = (message) => { try { onEvent(JSON.parse(message.data) as { device_id: string; attribute: string; old_value?: unknown; new_value?: unknown; causation_id: string }); } catch { /* ignore malformed event frames */ } }; return () => source.close(); };
export const validateFlow = (strategy_id: string, flow: unknown) => request<{ runnable: boolean; gaps: string[]; warnings: string[] }>("/api/v1/flows/validate", { method: "POST", body: JSON.stringify({ strategy_id, flow }) });
export const runFlow = (device_id: string, actor: string, lease_token: string, flow: unknown) => request<{ run_id: string; disposition: string; chapters: Array<{ id: string; disposition: string; message: string }>; resolutions?: Array<{ target: string; rung: string; confidence: number }>; evidence: Array<{ id: string; kind: string; checksum?: string; sha256?: string; size_bytes: number; created_at: string; applied_rules?: string[]; redaction_verified: boolean; recording_method?: string; effective_fps?: number; disposition?: string; disposition_reason?: string }> }>("/api/v1/flows/run", { method: "POST", body: JSON.stringify({ device_id, actor, lease_token, flow }) });
export const auditRecords = () => request<{ records: Array<{ id: string; actor: string; device_id: string; verb: string; outcome: string; created_at: string }> }>("/api/v1/evidence/audit");
