// The remote terminal REST/WS edge is an explicit REST exception: it bridges
// the browser JSON terminal protocol to the server-side Bridge binary session
// transport. The browser never receives Bridge credentials.
import { buildApiUrl, resolveApiBase } from "@vrooli/api-base";
import { decodeApiError } from "./client";
import type { BackendID, ExpirationPolicy, SessionInfo } from "./sessions";

export interface RemoteTerminalTarget {
  id: string;
  kind: "bridge-node" | "ssh" | "attached";
  label: string;
  available: boolean;
  readiness?: string[];
  failureRung?: string;
}

type RemoteSessionWire = {
  id: string;
  shell?: string;
  created_at?: string;
  cols?: number;
  rows?: number;
  backend?: string;
  survives_restart?: boolean;
  policy?: { mode?: string; duration?: string };
  busy?: boolean;
  origin?: string;
  owner?: string;
  display_label?: string;
};

function url(path: string): string {
  return buildApiUrl(path, { baseUrl: resolveApiBase({ appendSuffix: true }) });
}

function decodeRemoteSession(s: RemoteSessionWire): SessionInfo {
  const policy = s.policy;
  return {
    id: s.id,
    shell: s.shell ?? "",
    created_at: s.created_at ?? "",
    cols: s.cols ?? 0,
    rows: s.rows ?? 0,
    backend: (s.backend as BackendID) || "standard",
    survives_restart: Boolean(s.survives_restart),
    policy: {
      mode: (policy?.mode as ExpirationPolicy["mode"]) || "never",
      ...(policy?.duration ? { duration: policy.duration } : {}),
    },
    busy: Boolean(s.busy),
    origin: "remote",
    owner: s.owner ?? "",
    display_label: s.display_label ?? "Remote terminal",
  };
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url(path), { ...init, headers: { "Content-Type": "application/json", ...(init?.headers ?? {}) } });
  if (!response.ok) throw await decodeApiError(response);
  return (await response.json()) as T;
}

export async function listRemoteTerminalTargets(): Promise<RemoteTerminalTarget[]> {
  const data = await request<{ targets?: RemoteTerminalTarget[] }>("/remote-sessions/targets");
  return data.targets ?? [];
}

export async function createRemoteSession(opts: {
  target_id: string;
  shell?: string;
  working_dir?: string;
  launch_command?: string;
  cols?: number;
  rows?: number;
}): Promise<SessionInfo> {
  const data = await request<RemoteSessionWire>("/remote-sessions", { method: "POST", body: JSON.stringify(opts) });
  return decodeRemoteSession(data);
}

export async function listRemoteSessions(): Promise<SessionInfo[]> {
  const data = await request<{ sessions?: RemoteSessionWire[] }>("/remote-sessions");
  return (data.sessions ?? []).map(decodeRemoteSession);
}

export async function deleteRemoteSession(id: string): Promise<void> {
  const response = await fetch(url(`/remote-sessions/${encodeURIComponent(id)}`), { method: "DELETE" });
  if (!response.ok) throw await decodeApiError(response);
}
