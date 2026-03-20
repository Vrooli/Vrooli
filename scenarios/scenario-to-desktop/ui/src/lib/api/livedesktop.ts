import { buildUrl, throwIfNotOk } from "./client";

// ============================================================================
// Types
// ============================================================================

export type SessionState = "creating" | "running" | "stopping" | "stopped" | "error";

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
}

export interface DesktopSessionConfig {
  width?: number;
  height?: number;
  scenario_name: string;
  app_path?: string;
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

export async function launchAppOnDesktop(id: string, appPath: string): Promise<void> {
  const res = await fetch(buildUrl(`/livedesktop/sessions/${encodeURIComponent(id)}/launch`), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ app_path: appPath }),
  });
  await throwIfNotOk(res);
}

/**
 * Build the WebSocket URL for the VNC proxy endpoint.
 * Uses the same origin as the API, switching protocol to ws/wss.
 */
export function buildVncWsUrl(sessionId: string): string {
  const apiUrl = buildUrl(`/livedesktop/sessions/${encodeURIComponent(sessionId)}/ws`);
  return apiUrl.replace(/^http/, "ws");
}
