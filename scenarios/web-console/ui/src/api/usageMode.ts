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
  /** "time", "window_reset" or "off"; open so an unknown kind still reads. */
  untilKind: string;
  startedBy?: string;
  /** "save" lowers the model; "learn" is a comparison with no settings, whose model depends on the arm. */
  kind: "save" | "learn";
  role?: string;
  splitPercent?: number;
  sessionsOptIn?: boolean;
  paused?: boolean;
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
  /** The compaction limit this agent's active mode applies, when the server reports it. */
  contextLimitTokens?: number;
  /** Set when a Learn comparison includes this agent's sessions: the model depends on the arm, so none is named. */
  learnModeId?: string;
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

/** Provenance-carrying context limit shared by launch settings and the live meter. */
export interface ContextLimitRecord {
  model?: string;
  modelSource?: string;
  modelWindowTokens: number | null;
  modelWindowSource: string;
  compactionPointTokens: number | null;
  compactionPointSource: string;
  /** Model-specific choices the server has already admitted for this harness. */
  offeredCompactionPoints?: number[];
}

export function contextLimitRecord(input: {
  model?: string;
  modelSource?: string;
  contextWindowTokens?: number | bigint | null;
  contextWindowSource?: string;
  contextLimitTokens?: number | bigint | null;
  contextLimitSource?: string;
  contextLimitOptions?: Array<number | bigint | null>;
}): ContextLimitRecord {
  const window = Number(input.contextWindowTokens ?? 0);
  const point = Number(input.contextLimitTokens ?? 0);
  const windowSource = input.contextWindowSource?.trim();
  // A positive number without a reader is not provenance-bearing. This also
  // makes an older proto consumer that drops the additive source field show an
  // unknown window instead of presenting an unverified value.
  const hasVersionedWindowSource = Boolean(windowSource && /^[a-z][a-z0-9-]*\/v[1-9]\d*$/.test(windowSource));
  const hasVerifiedWindow = window > 0 && hasVersionedWindowSource;
  const offeredCompactionPoints = [...new Set((input.contextLimitOptions ?? [])
    .map(Number)
    .filter((tokens) => Number.isFinite(tokens) && tokens > 0 && (!hasVerifiedWindow || tokens <= window)))].sort((a, b) => b - a);
  return {
    model: input.model,
    modelSource: input.modelSource,
    modelWindowTokens: hasVerifiedWindow ? window : null,
    modelWindowSource: hasVerifiedWindow ? windowSource! : "unknown",
    compactionPointTokens: point > 0 ? point : null,
    compactionPointSource: point > 0 ? input.contextLimitSource || "unknown" : "unknown",
    offeredCompactionPoints,
  };
}

async function request(path: string, init?: RequestInit): Promise<UsageModeState> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Device-Id", deviceIdentity().id);
  const token = await getAccessToken();
  if (token && !headers.has("Authorization")) headers.set("Authorization", `Bearer ${token}`);
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), { ...init, headers, cache: "no-store" });
  if (!res.ok) throw makeApiError("usage_mode_error", `usage mode request failed (${String(res.status)})`, res.status);
  const body = (await res.json()) as Partial<UsageModeState>;
  return { modes: (body.modes ?? []).map((mode) => ({ ...mode, kind: mode.kind === "learn" ? "learn" as const : "save" as const })), agents: body.agents ?? [], canControl: Boolean(body.canControl) };
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
  contextBudget?: ContextLimitRecord & {
    modelState: string;
    usedTokens: number | null;
    usedSource: string;
    tokensUntilCompaction: number | null;
    forecastState: string;
    tokensPerSecond?: number | null;
    secondsUntilCompaction?: number | null;
    progressPercent: number | null;
  };
  goalActive: boolean;
  eligible: boolean;
  skipReason?: string;
  armed: boolean;
  thresholdTokens: number;
  last?: CompactEvent;
}

export interface CompactEvent {
  at: string;
  /** "manual", "armed", "threshold" or "bulk"; open as above. */
  trigger: string;
  /** "compacted", "skipped" or "failed"; open as above. */
  outcome: string;
  reason?: string;
  tokensBefore?: number;
  tokensAfter?: number;
  pinnedStanceSent?: boolean;
}

export interface CompactResult extends CompactEvent {
  sessionId: string;
}

export interface SessionCompaction {
  watch: { armed: boolean; thresholdTokens: number };
  history: CompactEvent[];
  pinnedStanceMaxBytes: number;
  result?: CompactEvent;
  armed?: boolean;
  pinnedStance?: boolean;
  pinnedStanceText?: string;
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
  compactionRequest<{ candidates: CompactCandidate[] | null }>("/api/v1/context-compaction/candidates", { method: "GET", signal }).then((body) => body.candidates ?? []);
export const compactIdleSessions = (sessionIds: string[]) =>
  compactionRequest<{ results: CompactResult[] | null }>("/api/v1/context-compaction/compact-idle", jsonPost({ sessionIds })).then((body) => body.results ?? []);
export const getSessionCompaction = (sessionId: string, signal?: AbortSignal) =>
  compactionRequest<SessionCompaction>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/context-compaction`, { method: "GET", signal });
export const setSessionCompaction = (sessionId: string, body: { action: "compact_when_idle" | "set_threshold" | "cancel" | "pin_stance" | "unpin_stance"; thresholdTokens?: number; stance?: string }) =>
  compactionRequest<SessionCompaction>(`/api/v1/sessions/${encodeURIComponent(sessionId)}/context-compaction`, jsonPost(body));
