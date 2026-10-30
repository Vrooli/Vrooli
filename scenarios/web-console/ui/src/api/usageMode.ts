// Usage mode client. Talks only to Web Console's own API; it reads the shared
// mode file on the server and has no Agent Manager dependency.

import { buildApiUrl } from "@vrooli/api-base";
import { deviceIdentity } from "../lib/deviceIdentity";
import { getAccessToken } from "../lib/auth";
import { API_BASE, makeApiError } from "./client";

export interface UsageModeView {
  id: string;
  label: string;
  runner: string;
  startedAt: string;
  expiresAt?: string;
  untilKind: "time" | "window_reset" | "off" | string;
  startedBy?: string;
}

/** What a new session of one agent will use. `model` is empty when no mode applies. */
export interface UsageModeAgent {
  agentId: string;
  label: string;
  runner: string;
  lowerModel: string;
  usualModel: string;
  modeId?: string;
  model?: string;
  /** Arguments that pin the normal default for one launch; absent when pointless. */
  fullPowerArgs?: string;
}

export interface UsageModeState {
  modes: UsageModeView[];
  agents: UsageModeAgent[];
  canControl: boolean;
}

export type UsageModeStart =
  | { agentId: string; kind: "minutes"; minutes: number }
  | { agentId: string; kind: "off" }
  | { agentId: string; kind: "reset"; windowMinutes: number };

async function request(path: string, init?: RequestInit): Promise<UsageModeState> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Device-Id", deviceIdentity().id);
  const token = await getAccessToken();
  if (token && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${token}`);
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { ...init, headers, cache: "no-store" });
  if (!res.ok) throw makeApiError("usage_mode_error", `usage mode request failed (${String(res.status)})`, res.status);
  const body = (await res.json()) as Partial<UsageModeState>;
  return { modes: body.modes ?? [], agents: body.agents ?? [], canControl: Boolean(body.canControl) };
}

export const getUsageMode = (signal?: AbortSignal) => request("/api/v1/usage-mode", { method: "GET", signal });
export const startUsageMode = (body: UsageModeStart) =>
  request("/api/v1/usage-mode/start", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
export const endUsageMode = (id: string) =>
  request("/api/v1/usage-mode/end", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ id }) });

// ---- Context compaction ----------------------------------------------------
// Compacting a session, or watching its context, never changes a harness
// setting: the server types /compact only into an idle session with no goal.

export interface CompactCandidate {
  sessionId: string;
  label: string;
  harness: string;
  state: string;
  contextTokens: number;
  goalActive: boolean;
  eligible: boolean;
  skipReason?: string;
  armed: boolean;
  thresholdTokens: number;
  last?: CompactEvent;
}

export interface CompactEvent {
  at: string;
  trigger: "manual" | "armed" | "threshold" | "bulk" | string;
  outcome: "compacted" | "skipped" | "failed" | string;
  reason?: string;
  tokensBefore?: number;
  tokensAfter?: number;
}

export interface CompactResult extends CompactEvent {
  sessionId: string;
}

export interface SessionCompaction {
  watch: { armed: boolean; thresholdTokens: number };
  history: CompactEvent[];
  result?: CompactEvent;
  armed?: boolean;
}

async function compactionRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Device-Id", deviceIdentity().id);
  const token = await getAccessToken();
  if (token && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${token}`);
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { ...init, headers, cache: "no-store" });
  if (!res.ok) throw makeApiError("context_compaction_error", `context compaction request failed (${String(res.status)})`, res.status);
  return (await res.json()) as T;
}

const jsonPost = (body: unknown): RequestInit => ({ method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });

export const getCompactCandidates = (signal?: AbortSignal) =>
  compactionRequest<{ candidates: CompactCandidate[] }>("/api/v1/context-compaction/candidates", { method: "GET", signal }).then((body) => body.candidates ?? []);
export const compactIdleSessions = (sessionIds: string[]) =>
  compactionRequest<{ results: CompactResult[] }>("/api/v1/context-compaction/compact-idle", jsonPost({ sessionIds })).then((body) => body.results ?? []);
export const getSessionCompaction = (sessionId: string, signal?: AbortSignal) =>
  compactionRequest<SessionCompaction>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/context-compaction`, { method: "GET", signal });
export const setSessionCompaction = (sessionId: string, body: { action: "compact_when_idle" | "set_threshold" | "cancel"; thresholdTokens?: number }) =>
  compactionRequest<SessionCompaction>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/context-compaction`, jsonPost(body));
