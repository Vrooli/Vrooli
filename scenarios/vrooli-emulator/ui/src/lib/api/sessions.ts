import { buildUrl, throwIfNotOk } from "./http";

export type SessionState = "creating" | "running" | "stopping" | "stopped" | "error";

export interface MetricsView {
  splash_duration_ms?: number;
  splash_detected: boolean;
  ready_duration_ms?: number;
  ready_detected: boolean;
  current_cpu_percent?: number;
  current_rss_mb?: number;
  peak_rss_mb?: number;
  sample_count: number;
}

export interface Session {
  id: string;
  scenario_name: string;
  state: SessionState;
  vnc_port: number;
  ws_port: number;
  width: number;
  height: number;
  created_at: string;
  last_heartbeat: string;
  error?: string;
  is_recording: boolean;
  network_mode: "normal" | "offline" | "slow";
  bandwidth_kbps?: number;
  dark_mode: boolean;
  locale?: string;
  app_running: boolean;
  platform?: string;
  metrics?: MetricsView;
}

export interface SessionConfig {
  width?: number;
  height?: number;
  scenario_name: string;
  app_path?: string;
  platform?: string;
}

export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "reconnecting" | "failed";

export async function createSession(config: SessionConfig): Promise<Session> {
  const res = await fetch(buildUrl("/sessions"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  await throwIfNotOk(res);
  return (await res.json()) as Session;
}

export async function destroySession(id: string): Promise<void> {
  const res = await fetch(buildUrl(`/sessions/${encodeURIComponent(id)}`), {
    method: "DELETE",
  });
  await throwIfNotOk(res);
}

export async function heartbeatSession(id: string): Promise<void> {
  const res = await fetch(buildUrl(`/sessions/${encodeURIComponent(id)}/heartbeat`), {
    method: "POST",
  });
  await throwIfNotOk(res);
}

export async function getSession(id: string): Promise<Session> {
  const res = await fetch(buildUrl(`/sessions/${encodeURIComponent(id)}`));
  await throwIfNotOk(res);
  return (await res.json()) as Session;
}

export async function listSessions(): Promise<Session[]> {
  const res = await fetch(buildUrl("/sessions"));
  await throwIfNotOk(res);
  return (await res.json()) as Session[];
}

export async function launchApp(id: string, appPath?: string): Promise<void> {
  const res = await fetch(buildUrl(`/sessions/${encodeURIComponent(id)}/launch`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(appPath ? { app_path: appPath } : {}),
  });
  await throwIfNotOk(res);
}

export interface ControlRequest {
  action: string;
  params?: Record<string, unknown>;
}

export interface ControlResult {
  status: string;
  data?: Record<string, unknown>;
  message?: string;
}

export async function executeSessionControl(id: string, req: ControlRequest): Promise<ControlResult> {
  const res = await fetch(buildUrl(`/sessions/${encodeURIComponent(id)}/control`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  await throwIfNotOk(res);
  return (await res.json()) as ControlResult;
}

export function buildVncWsUrl(sessionId: string): string {
  const apiUrl = buildUrl(`/sessions/${encodeURIComponent(sessionId)}/ws`);
  return apiUrl.replace(/^http/, "ws");
}
