import { buildUrl, throwIfNotOk } from "./client";

// ============================================================================
// Types
// ============================================================================

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

export interface DesktopSession {
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

export interface DesktopSessionConfig {
  width?: number;
  height?: number;
  scenario_name: string;
  app_path?: string;
  platform?: string;
}

export type ConnectionStatus = "disconnected" | "connecting" | "connected" | "error";

// ============================================================================
// API Functions
// ============================================================================

export async function startDesktopSession(config: DesktopSessionConfig): Promise<DesktopSession> {
  const res = await fetch(buildUrl("/livedesktop/sessions"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(config),
  });
  await throwIfNotOk(res);
  return res.json();
}

export async function stopDesktopSession(id: string): Promise<void> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}`), {
    method: "DELETE",
  });
  await throwIfNotOk(res);
}

export async function heartbeatSession(id: string): Promise<void> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}/heartbeat`), {
    method: "POST",
  });
  await throwIfNotOk(res);
}

export async function getDesktopSession(id: string): Promise<DesktopSession> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}`));
  await throwIfNotOk(res);
  return res.json();
}

export async function listDesktopSessions(): Promise<DesktopSession[]> {
  const res = await fetch(buildUrl("/livedesktop/sessions"));
  await throwIfNotOk(res);
  return res.json();
}

export async function launchAppOnDesktop(id: string, appPath?: string): Promise<void> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}/launch`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(appPath ? { app_path: appPath } : {}),
  });
  await throwIfNotOk(res);
}

export async function findArtifact(id: string): Promise<string> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}/artifact`));
  await throwIfNotOk(res);
  const data: { artifact_path: string } = await res.json();
  return data.artifact_path;
}

// ============================================================================
// Control Actions
// ============================================================================

export interface ControlRequest {
  action: string;
  params?: Record<string, unknown>;
}

export interface ControlResult {
  status: string;
  data?: Record<string, unknown>;
  message?: string;
}

export async function executeDesktopControl(id: string, req: ControlRequest): Promise<ControlResult> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}/control`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  await throwIfNotOk(res);
  return res.json();
}

/**
 * Build the WebSocket URL for the VNC proxy endpoint.
 * Uses the same origin as the API, switching protocol to ws/wss.
 */
export function buildVncWsUrl(sessionId: string): string {
  const apiUrl = buildUrl(`/livedesktop/sessions/${encodeURIComponent(sessionId)}/ws`);
  return apiUrl.replace(/^http/, "ws");
}
