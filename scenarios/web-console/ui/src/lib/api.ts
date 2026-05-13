// DOC: docs/concepts/ARCHITECTURE.md#session-creation
// DOC: docs/internal/ERROR_SEMANTICS.md#structured-error-type-typescript
import { resolveApiBase, buildApiUrl, buildWsUrl } from "@vrooli/api-base";
import type { ShortcutEntry } from "../consts/shortcuts";

// [REQ:P0-004a] api-base HTTP Integration
const API_BASE = resolveApiBase({ appendSuffix: true });

// [REQ:P1-001a] Expiration Policy types
export type PolicyMode = "never" | "preset" | "custom";

export interface ExpirationPolicy {
  mode: PolicyMode;
  duration?: string;
}

// CROSS-LANGUAGE COUPLING: Backend IDs must match BackendID constants in api/backend_registry.go
export type BackendID = "standard" | "persistent";

export interface BackendOption {
  id: BackendID;
  display_name: string;
  description: string;
  survives_restart: boolean;
  available: boolean;
  reason?: string;
}

export interface SessionInfo {
  id: string;
  shell: string;
  created_at: string;
  cols: number;
  rows: number;
  backend: BackendID;
  survives_restart: boolean;
  policy: ExpirationPolicy;
  busy: boolean;
  recovered?: boolean;
}

export interface PolicyResponse {
  session_id: string;
  policy: ExpirationPolicy;
  expires_at?: string;
  ttl_seconds?: number;
}

/** API error response shape from the backend. */
export interface APIErrorBody {
  error: string;
  code?: string;
  category?: "validation" | "resource_limit" | "dependency" | "internal";
  recovery?: string;
  retry?: boolean;
}

/**
 * An API error with structured fields for programmatic handling.
 * UI components can inspect `code`, `category`, and `recovery` to
 * choose the right recovery action instead of just showing a message.
 */
export class APIError extends Error {
  readonly code: string;
  readonly category: string;
  readonly recovery: string;
  readonly retry: boolean;
  readonly status: number;

  constructor(status: number, body: APIErrorBody) {
    super(body.error);
    this.name = "APIError";
    this.status = status;
    this.code = body.code ?? "unknown";
    this.category = body.category ?? "internal";
    this.recovery = body.recovery ?? "";
    this.retry = body.retry ?? false;
  }
}

/**
 * Extract a structured error from an API response.
 * Returns an APIError with category, recovery hints, and retry flag.
 */
async function extractAPIError(res: Response, fallback: string): Promise<APIError> {
  try {
    const body = (await res.json()) as APIErrorBody;
    if (body.error) return new APIError(res.status, body);
  } catch {
    // Response body was not JSON — use fallback
  }
  return new APIError(res.status, {
    error: `${fallback}: ${res.status}`,
    code: "unknown",
    category: "internal",
    recovery: "Try again. If the problem persists, check server logs.",
    retry: true,
  });
}

/**
 * Structured error info for display. Produced by `toErrorInfo` and
 * consumed by error display components (ErrorBanner, useSessionManager).
 */
export interface ErrorInfo {
  message: string;
  recovery?: string;
  retry?: boolean;
}

/**
 * Convert an unknown caught error into a plain object with message,
 * recovery hint, and retry flag. Extracts structured fields from
 * APIError; falls back to generic message for other error types.
 */
export function toErrorInfo(err: unknown): ErrorInfo {
  const message = err instanceof Error ? err.message : "Unknown error";
  if (err instanceof APIError) {
    return { message, recovery: err.recovery || undefined, retry: err.retry || undefined };
  }
  return { message };
}

export async function fetchHealth() {
  const url = buildApiUrl("/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store"
  });
  if (!res.ok) throw new Error(`API health check failed: ${res.status}`);
  return res.json() as Promise<{ status: string; service: string; timestamp: string }>;
}

export async function createSession(opts?: {
  shell?: string;
  cols?: number;
  rows?: number;
  backend?: BackendID;
  policy?: { mode: PolicyMode; duration?: string };
  launch_command?: string;
  agent_type?: "none" | "codex" | "claude";
}): Promise<SessionInfo> {
  const resp = await sessionsClient.create({
    shell: opts?.shell ?? "",
    cols: opts?.cols ?? 0,
    rows: opts?.rows ?? 0,
    backend: opts?.backend ?? "",
    launchCommand: opts?.launch_command ?? "",
    agentType: opts?.agent_type ?? "",
    ...(opts?.policy
      ? { policy: { mode: opts.policy.mode, duration: opts.policy.duration ?? "" }, hasPolicy: true }
      : {}),
  });
  return decodeSession(resp.session);
}

// Persistent-session-recovery surface. See
// scenarios/web-console/docs/guides/SESSION_RECOVERY.md for the contract.
export interface RecoverableSession {
  id: string;
  backend?: string;
  shell?: string;
  cols?: number;
  rows?: number;
  created_at?: string;
  orphaned_at?: string;
  last_activity_at?: string;
  agent_type?: "none" | "codex" | "claude";
  agent_session_id?: string;
  launch_command?: string;
  cwd?: string;
  last_rollout_path?: string;
  recoverable: boolean;
  not_recoverable_reason?: string;
}

export interface RecoverResult {
  old_session_id: string;
  new_session_id: string;
  agent_type?: string;
  command_sent?: string;
  codex_home_copied?: boolean;
}

export async function listRecoverableSessions(): Promise<RecoverableSession[]> {
  const resp = await sessionsClient.listRecoverable({});
  return resp.sessions.map(decodeRecoverable);
}

export async function recoverSession(oldId: string): Promise<RecoverResult> {
  const resp = await sessionsClient.recover({ id: oldId });
  return {
    old_session_id: resp.oldSessionId,
    new_session_id: resp.newSessionId,
    agent_type: resp.agentType || undefined,
    command_sent: resp.commandSent || undefined,
    codex_home_copied: resp.codexHomeCopied,
  };
}

export async function dismissRecoverableSession(oldId: string): Promise<void> {
  await sessionsClient.dismissRecoverable({ id: oldId });
}

export async function listSessions(): Promise<SessionInfo[]> {
  const resp = await sessionsClient.list({});
  return resp.sessions.map(decodeSession);
}

export async function getSession(id: string): Promise<SessionInfo> {
  const resp = await sessionsClient.get({ id });
  return decodeSession(resp.session);
}

export async function deleteSession(id: string): Promise<void> {
  await sessionsClient.delete({ id });
}

// [REQ:P1-001a] Session Policy API - client
export async function getSessionPolicy(id: string): Promise<PolicyResponse> {
  const resp = await sessionsClient.getPolicy({ id });
  return decodePolicyView(resp.policy);
}

export async function updateSessionPolicy(
  id: string,
  policy: { mode: string; duration?: string },
): Promise<PolicyResponse> {
  const resp = await sessionsClient.updatePolicy({
    id,
    policy: { mode: policy.mode, duration: policy.duration ?? "" },
  });
  return decodePolicyView(resp.policy);
}

// Connect-Web proto messages always materialize nested fields (no
// `undefined`); the helpers below normalize them to the legacy
// snake_case shape the rest of the UI consumes.
type ProtoSession = {
  id: string;
  shell: string;
  createdAt: string;
  cols: number;
  rows: number;
  backend: string;
  survivesRestart: boolean;
  policy?: { mode: string; duration: string };
  busy: boolean;
  recovered: boolean;
};

type ProtoRecoverable = {
  id: string;
  backend: string;
  shell: string;
  cols: number;
  rows: number;
  createdAt: string;
  orphanedAt: string;
  lastActivityAt: string;
  agentType: string;
  agentSessionId: string;
  launchCommand: string;
  cwd: string;
  lastRolloutPath: string;
  recoverable: boolean;
  notRecoverableReason: string;
};

type ProtoPolicyView = {
  sessionId: string;
  policy?: { mode: string; duration: string };
  expiresAt: string;
  ttlSeconds: number;
  hasExpiry: boolean;
};

function decodeSession(s: ProtoSession | undefined): SessionInfo {
  const policy = s?.policy;
  return {
    id: s?.id ?? "",
    shell: s?.shell ?? "",
    created_at: s?.createdAt ?? "",
    cols: s?.cols ?? 0,
    rows: s?.rows ?? 0,
    backend: (s?.backend as BackendID) ?? "standard",
    survives_restart: s?.survivesRestart ?? false,
    policy: {
      mode: (policy?.mode as PolicyMode) ?? "never",
      ...(policy?.duration ? { duration: policy.duration } : {}),
    },
    busy: s?.busy ?? false,
    ...(s?.recovered ? { recovered: true } : {}),
  };
}

function decodeRecoverable(r: ProtoRecoverable): RecoverableSession {
  return {
    id: r.id,
    backend: r.backend || undefined,
    shell: r.shell || undefined,
    cols: r.cols || undefined,
    rows: r.rows || undefined,
    created_at: r.createdAt || undefined,
    orphaned_at: r.orphanedAt || undefined,
    last_activity_at: r.lastActivityAt || undefined,
    agent_type: (r.agentType as "none" | "codex" | "claude") || undefined,
    agent_session_id: r.agentSessionId || undefined,
    launch_command: r.launchCommand || undefined,
    cwd: r.cwd || undefined,
    last_rollout_path: r.lastRolloutPath || undefined,
    recoverable: r.recoverable,
    not_recoverable_reason: r.notRecoverableReason || undefined,
  };
}

function decodePolicyView(v: ProtoPolicyView | undefined): PolicyResponse {
  return {
    session_id: v?.sessionId ?? "",
    policy: {
      mode: (v?.policy?.mode as PolicyMode) ?? "never",
      ...(v?.policy?.duration ? { duration: v.policy.duration } : {}),
    },
    ...(v?.hasExpiry
      ? {
          expires_at: v.expiresAt,
          ttl_seconds: v.ttlSeconds,
        }
      : {}),
  };
}

// [REQ:P0-005a] AI Command Generation API - client
export interface AIGenerateResponse {
  command: string;
  provider: string;
}

export async function generateAICommand(prompt: string, context?: string): Promise<AIGenerateResponse> {
  const resp = await aiClient.generate({ prompt, context: context ?? "" });
  return { command: resp.command, provider: resp.provider };
}

// [REQ:P0-005a] AI Suggestion API - client
export interface AISuggestResponse {
  commands: string[];
  provider: string;
}

export async function generateAISuggestions(prompt: string, context?: string): Promise<AISuggestResponse> {
  const resp = await aiClient.suggest({ prompt, context: context ?? "" });
  return { commands: resp.commands, provider: resp.provider };
}

export interface TTSPlaybackEvent {
  source: string;
  stage: string;
  backend?: string;
  sessionId?: string;
  message?: string;
}

export interface ConversationEvent {
  id: string;
  sessionId: string;
  source: string;
  role: "assistant" | "user";
  text: string;
  speechParagraphs: string[];
  originalSpeechParagraphs?: string[];
  summarized: boolean;
  createdAt: string;
  sequence: number;
  deliveryState: string;
  ttsState: string;
  consumptionState: string;
}

export interface ConversationCursor {
  lastSeenSequence: number;
  lastListenedSequence: number;
}

export interface FileReferenceResolveResponse {
  input_path: string;
  resolved_path: string;
  line?: number;
  exists: boolean;
  resolution_basis: "session_cwd" | "project_root" | "absolute_allowed" | "session_upload";
  category: "markdown" | "code" | "text" | "binary";
  can_preview: boolean;
}

export interface FileReferenceContentResponse {
  path: string;
  line?: number;
  category: "markdown" | "code" | "text" | "binary";
  content_type: string;
  content: string;
  truncated: boolean;
}

export interface ConversationSessionResponse {
  sessionId: string;
  events: ConversationEvent[];
  cursor: ConversationCursor;
}

export async function reportTTSEvent(event: TTSPlaybackEvent): Promise<void> {
  await ttsClient.recordPlaybackEvent({
    event: {
      source: event.source,
      stage: event.stage,
      backend: event.backend ?? "",
      sessionId: event.sessionId ?? "",
      message: event.message ?? "",
    },
  });
}

export async function getConversationSession(
  sessionId: string,
  opts?: { sinceSequence?: number },
): Promise<ConversationSessionResponse> {
  const resp = await conversationClient.get({
    sessionId,
    sinceSequence: opts?.sinceSequence && opts.sinceSequence > 0 ? BigInt(opts.sinceSequence) : 0n,
  });
  return {
    sessionId: resp.sessionId,
    events: resp.events.map(decodeConversationEvent),
    cursor: decodeConversationCursor(resp.cursor),
  };
}

export async function updateConversationCursor(
  sessionId: string,
  patch: Partial<ConversationCursor>,
): Promise<ConversationCursor> {
  const req: {
    sessionId: string;
    lastSeenSequence?: bigint;
    hasLastSeenSequence?: boolean;
    lastListenedSequence?: bigint;
    hasLastListenedSequence?: boolean;
  } = { sessionId };
  if (patch.lastSeenSequence !== undefined) {
    req.lastSeenSequence = BigInt(patch.lastSeenSequence);
    req.hasLastSeenSequence = true;
  }
  if (patch.lastListenedSequence !== undefined) {
    req.lastListenedSequence = BigInt(patch.lastListenedSequence);
    req.hasLastListenedSequence = true;
  }
  const resp = await conversationClient.updateCursor(req);
  return decodeConversationCursor(resp.cursor);
}

export async function resolveFileReference(
  sessionId: string,
  path: string,
): Promise<FileReferenceResolveResponse> {
  const resp = await conversationClient.resolveFileReference({ sessionId, path });
  return {
    input_path: resp.inputPath,
    resolved_path: resp.resolvedPath,
    line: resp.hasLine ? resp.line : undefined,
    exists: resp.exists,
    resolution_basis: resp.resolutionBasis as FileReferenceResolveResponse["resolution_basis"],
    category: resp.category as FileReferenceResolveResponse["category"],
    can_preview: resp.canPreview,
  };
}

export async function getFileReferenceContent(
  sessionId: string,
  path: string,
): Promise<FileReferenceContentResponse> {
  const resp = await conversationClient.getFileReferenceContent({ sessionId, path });
  return {
    path: resp.path,
    line: resp.hasLine ? resp.line : undefined,
    category: resp.category as FileReferenceContentResponse["category"],
    content_type: resp.contentType,
    content: resp.content,
    truncated: resp.truncated,
  };
}

interface ProtoConversationEvent {
  id: string;
  sessionId: string;
  source: string;
  role: string;
  text: string;
  speechParagraphs: string[];
  originalSpeechParagraphs: string[];
  summarized: boolean;
  createdAt: string;
  sequence: bigint;
  deliveryState: string;
  ttsState: string;
  consumptionState: string;
}

interface ProtoConversationCursor {
  lastSeenSequence: bigint;
  lastListenedSequence: bigint;
}

function decodeConversationEvent(e: ProtoConversationEvent): ConversationEvent {
  return {
    id: e.id,
    sessionId: e.sessionId,
    source: e.source,
    role: e.role as ConversationEvent["role"],
    text: e.text,
    speechParagraphs: e.speechParagraphs,
    originalSpeechParagraphs: e.originalSpeechParagraphs.length > 0 ? e.originalSpeechParagraphs : undefined,
    summarized: e.summarized,
    createdAt: e.createdAt,
    sequence: Number(e.sequence),
    deliveryState: e.deliveryState,
    ttsState: e.ttsState,
    consumptionState: e.consumptionState,
  };
}

function decodeConversationCursor(c: ProtoConversationCursor | undefined): ConversationCursor {
  return {
    lastSeenSequence: c ? Number(c.lastSeenSequence) : 0,
    lastListenedSequence: c ? Number(c.lastListenedSequence) : 0,
  };
}

// [REQ:P1-002a] Shortcut Profile API - client
// Re-export ShortcutEntry as the canonical shortcut type for both API wire
// format and local config. Previously duplicated as ShortcutEntry.
export type { ShortcutEntry } from "../consts/shortcuts";

export interface ShortcutProfile {
  id: string;
  scope: string;
  name: string;
  shortcuts: ShortcutEntry[];
  created_at: string;
  updated_at: string;
}

// Shortcut helpers are shims over the Connect-Web ShortcutsService client
// (src/api/shortcuts.ts). The legacy snake_case wire shape is preserved
// here so existing call-sites don't have to change.

function decodeShortcuts(in_: { label: string; command: string; description: string }[]): ShortcutEntry[] {
  return in_.map((s) => ({
    label: s.label,
    command: s.command,
    description: s.description || undefined,
  }));
}

function decodeProfile(p: {
  id: string;
  scope: string;
  name: string;
  shortcuts: { label: string; command: string; description: string }[];
  createdAt: string;
  updatedAt: string;
}): ShortcutProfile {
  return {
    id: p.id,
    scope: p.scope,
    name: p.name,
    shortcuts: decodeShortcuts(p.shortcuts),
    created_at: p.createdAt,
    updated_at: p.updatedAt,
  };
}

export async function getEffectiveShortcuts(): Promise<ShortcutEntry[]> {
  const resp = await shortcutsClient.getEffective({});
  return decodeShortcuts(resp.shortcuts);
}

export async function listShortcutProfiles(): Promise<ShortcutProfile[]> {
  const resp = await shortcutsClient.listProfiles({});
  return resp.profiles.map(decodeProfile);
}

export async function upsertShortcutProfile(profile: {
  id: string;
  scope: string;
  name: string;
  shortcuts: ShortcutEntry[];
}): Promise<ShortcutProfile> {
  const resp = await shortcutsClient.upsertProfile({
    id: profile.id,
    scope: profile.scope,
    name: profile.name,
    shortcuts: profile.shortcuts.map((s) => ({
      label: s.label,
      command: s.command,
      description: s.description ?? "",
    })),
  });
  if (!resp.profile) {
    throw new Error("upsertShortcutProfile: missing profile in response");
  }
  return decodeProfile(resp.profile);
}

export async function deleteShortcutProfile(id: string): Promise<void> {
  await shortcutsClient.deleteProfile({ id });
}

// [REQ:P1-003a] AI Provider Config API - client
export interface ProviderConfig {
  name: string;
  enabled: boolean;
  priority: number;
  timeout_sec: number;
  max_retries: number;
}

export interface ProviderHealth {
  name: string;
  available: boolean;
  last_check?: string;
  last_latency?: string;
  error_count: number;
  success_count: number;
  error_rate: number;
}

export interface AIProviderConfigResponse {
  providers: ProviderConfig[];
  health: ProviderHealth[];
}

type AIConfigClientReq = Parameters<typeof aiClient.getConfig>[0];
type AIConfigClientResp = Awaited<ReturnType<typeof aiClient.getConfig>>;
type AIProviderConfigProto = AIConfigClientResp["providers"][number];
type AIProviderHealthProto = AIConfigClientResp["health"][number];

function decodeProviderConfig(p: AIProviderConfigProto): ProviderConfig {
  return {
    name: p.name,
    enabled: p.enabled,
    priority: p.priority,
    timeout_sec: p.timeoutSec,
    max_retries: p.maxRetries,
  };
}

function decodeProviderHealth(h: AIProviderHealthProto): ProviderHealth {
  return {
    name: h.name,
    available: h.available,
    last_check: h.lastCheck || undefined,
    last_latency: h.lastLatency || undefined,
    error_count: Number(h.errorCount),
    success_count: Number(h.successCount),
    error_rate: h.errorRate,
  };
}

export async function getAIConfig(): Promise<AIProviderConfigResponse> {
  const req: AIConfigClientReq = {};
  const resp = await aiClient.getConfig(req);
  return {
    providers: resp.providers.map(decodeProviderConfig),
    health: resp.health.map(decodeProviderHealth),
  };
}

export async function updateAIConfig(update: {
  name: string;
  enabled?: boolean;
  priority?: number;
  timeout_sec?: number;
  max_retries?: number;
}): Promise<AIProviderConfigResponse> {
  const req: Parameters<typeof aiClient.updateConfig>[0] = { name: update.name };
  if (update.enabled !== undefined) {
    req.enabled = update.enabled;
    req.hasEnabled = true;
  }
  if (update.priority !== undefined) {
    req.priority = update.priority;
    req.hasPriority = true;
  }
  if (update.timeout_sec !== undefined) {
    req.timeoutSec = update.timeout_sec;
    req.hasTimeoutSec = true;
  }
  if (update.max_retries !== undefined) {
    req.maxRetries = update.max_retries;
    req.hasMaxRetries = true;
  }
  const resp = await aiClient.updateConfig(req);
  return {
    providers: resp.providers.map(decodeProviderConfig),
    health: resp.health.map(decodeProviderHealth),
  };
}

export async function getAIHealth(): Promise<ProviderHealth[]> {
  const resp = await aiClient.getHealth({});
  return resp.health.map(decodeProviderHealth);
}

function apiBaseToWsBase(apiBase: string): string {
  if (apiBase.startsWith("https://")) return `wss://${apiBase.slice("https://".length)}`;
  if (apiBase.startsWith("http://")) return `ws://${apiBase.slice("http://".length)}`;
  return apiBase;
}

// [REQ:P0-004b] api-base WebSocket Integration
export function buildSessionWsUrl(sessionId: string): string {
  const wsBase = apiBaseToWsBase(API_BASE);
  return buildWsUrl(`/sessions/${sessionId}/ws`, { baseUrl: wsBase });
}

export function buildVoiceStreamWsUrl(language?: string): string {
  const wsBase = apiBaseToWsBase(API_BASE);
  const base = buildWsUrl("/voice/stream", { baseUrl: wsBase });
  if (language) return `${base}${base.includes("?") ? "&" : "?"}language=${encodeURIComponent(language)}`;
  return base;
}

// Voice input capabilities
export type CapabilityStatus = "available" | "unavailable" | "unknown";

export interface CapabilityState {
  id: string;
  name: string;
  description: string;
  dependencyKind: string;
  dependencySlug: string;
  features: string[];
  status: CapabilityStatus;
  message?: string;
  checkedAt?: string;
}

export interface CapabilitiesResponse {
  capabilities: CapabilityState[];
  timestamp: string;
  session_backends?: BackendOption[];
  default_backend?: string;
}

export async function fetchCapabilities(): Promise<CapabilitiesResponse> {
  const resp = await capabilitiesClient.get({});
  return decodeCapabilitiesResponse(resp);
}

/** Fast liveness-only capability check (GET health only, no test transcription). */
export async function fetchCapabilitiesLiveness(): Promise<CapabilitiesResponse> {
  const resp = await capabilitiesClient.liveness({});
  return decodeCapabilitiesResponse(resp);
}

interface ProtoCapabilityState {
  id: string;
  name: string;
  description: string;
  dependencyKind: string;
  dependencySlug: string;
  features: string[];
  status: string;
  message: string;
  checkedAt: string;
}

interface ProtoBackendOption {
  id: string;
  displayName: string;
  description: string;
  survivesRestart: boolean;
  available: boolean;
  reason: string;
}

interface ProtoCapabilitiesResponse {
  capabilities: ProtoCapabilityState[];
  timestamp: string;
  sessionBackends?: ProtoBackendOption[];
  defaultBackend?: string;
}

function decodeCapabilitiesResponse(resp: ProtoCapabilitiesResponse): CapabilitiesResponse {
  const out: CapabilitiesResponse = {
    capabilities: resp.capabilities.map((c) => ({
      id: c.id,
      name: c.name,
      description: c.description,
      dependencyKind: c.dependencyKind,
      dependencySlug: c.dependencySlug,
      features: c.features ?? [],
      status: decodeCapabilityStatus(c.status),
      message: c.message || undefined,
      checkedAt: c.checkedAt || undefined,
    })),
    timestamp: resp.timestamp,
  };
  if (resp.sessionBackends && resp.sessionBackends.length > 0) {
    out.session_backends = resp.sessionBackends.map((b) => ({
      id: b.id as BackendID,
      display_name: b.displayName,
      description: b.description,
      survives_restart: b.survivesRestart,
      available: b.available,
      reason: b.reason || undefined,
    }));
  }
  if (resp.defaultBackend) {
    out.default_backend = resp.defaultBackend;
  }
  return out;
}

function decodeCapabilityStatus(s: string): CapabilityStatus {
  return s === "available" || s === "unavailable" ? s : "unknown";
}

/**
 * Cached wrapper around fetchCapabilitiesLiveness.
 * Concurrent and near-simultaneous callers share a single in-flight request.
 * Cache TTL matches the server-side capability cache (30 s).
 */
let _capCache: { promise: Promise<CapabilitiesResponse>; at: number } | null = null;
const CAP_CACHE_TTL = 30_000;

export function fetchCapabilitiesLivenessCached(): Promise<CapabilitiesResponse> {
  const now = Date.now();
  if (_capCache && now - _capCache.at < CAP_CACHE_TTL) return _capCache.promise;
  const promise = fetchCapabilitiesLiveness();
  _capCache = { promise, at: now };
  // Clear cache on rejection so next caller retries instead of getting a stale error.
  promise.catch(() => {
    if (_capCache?.promise === promise) _capCache = null;
  });
  return promise;
}

/**
 * Synchronous snapshot of the most recent capabilities liveness result.
 * Returns null if no check has completed yet.
 *
 * Used by startRecording() to avoid blocking the mic activation hot path
 * on a network request. The background refresh (fetchCapabilitiesLivenessCached
 * on a 25s interval) keeps this warm.
 *
 * DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
 */
export function getCapabilitiesLivenessSnapshot(): CapabilitiesResponse | null {
  if (!_capCache) return null;
  // Return null if the cache is stale (older than TTL). The background
  // refresh should prevent this, but this guards against edge cases.
  if (Date.now() - _capCache.at >= CAP_CACHE_TTL) return null;
  // The promise may still be in-flight (not yet resolved). We return null
  // in that case since we can't synchronously extract the resolved value.
  // However, the snapshot is populated after the first successful resolve.
  return _capSnapshot;
}

/** Resolved snapshot of the last successful liveness check. Updated after
 *  each successful fetchCapabilitiesLivenessCached() call. */
let _capSnapshot: CapabilitiesResponse | null = null;

/**
 * Refresh the capabilities liveness cache and update the synchronous snapshot.
 * Designed to be called on an interval to keep the snapshot warm.
 */
export async function refreshCapabilitiesLiveness(): Promise<CapabilitiesResponse> {
  const result = await fetchCapabilitiesLivenessCached();
  _capSnapshot = result;
  return result;
}

/** Reset the capabilities liveness cache. Exported for tests. */
export function _resetCapabilitiesCache(): void {
  _capCache = null;
  _capSnapshot = null;
}

export async function uploadFile(sessionId: string, file: File | Blob, filename?: string): Promise<string> {
  const url = buildApiUrl(`/sessions/${sessionId}/upload`, { baseUrl: API_BASE });
  const formData = new FormData();
  formData.append("file", file, filename ?? (file instanceof File ? file.name : "image.png"));
  const res = await fetch(url, { method: "POST", body: formData });
  if (!res.ok) throw await extractAPIError(res, "File upload failed");
  const data = (await res.json()) as { path: string };
  return data.path;
}

async function blobToBytes(b: Blob): Promise<Uint8Array> {
  return new Uint8Array(await b.arrayBuffer());
}

export async function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
  const resp = await voiceClient.transcribe({
    audio: await blobToBytes(audioBlob),
    contentType: audioBlob.type,
    language: language ?? "",
    skipSpeakerVerification: false,
  });
  return resp.text;
}

/**
 * Transcribe audio while bypassing the server-side speaker-verification
 * filter. Used exclusively by the "Transcribe anyway" retry action after a
 * false rejection.
 */
export async function transcribeAudioBypassFilter(
  audioBlob: Blob,
  language?: string,
): Promise<string> {
  const resp = await voiceClient.transcribe({
    audio: await blobToBytes(audioBlob),
    contentType: audioBlob.type,
    language: language ?? "",
    skipSpeakerVerification: true,
  });
  return resp.text;
}

// Voice streaming configuration (server-side pipeline tuning)
export interface VoiceStreamConfig {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
}

function decodeStreamConfig(c: {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  wakeWordEnabled: boolean;
  wakeWordThreshold: number;
  segmentSilenceMs: number;
} | undefined): VoiceStreamConfig {
  return {
    flushIntervalMs: c?.flushIntervalMs ?? 0,
    minDeltaBytes: c?.minDeltaBytes ?? 0,
    overlapBytes: c?.overlapBytes ?? 0,
    persistentMode: c?.persistentMode ?? false,
    wakeWordEnabled: c?.wakeWordEnabled ?? false,
    wakeWordThreshold: c?.wakeWordThreshold ?? 0,
    segmentSilenceMs: c?.segmentSilenceMs ?? 0,
  };
}

export async function getVoiceStreamConfig(): Promise<VoiceStreamConfig> {
  const resp = await voiceClient.getStreamConfig({});
  return decodeStreamConfig(resp.config);
}

export async function updateVoiceStreamConfig(
  patch: Partial<VoiceStreamConfig>,
): Promise<VoiceStreamConfig> {
  const req: Record<string, unknown> = {};
  if (patch.flushIntervalMs !== undefined) {
    req.flushIntervalMs = patch.flushIntervalMs;
    req.hasFlushIntervalMs = true;
  }
  if (patch.minDeltaBytes !== undefined) {
    req.minDeltaBytes = patch.minDeltaBytes;
    req.hasMinDeltaBytes = true;
  }
  if (patch.overlapBytes !== undefined) {
    req.overlapBytes = patch.overlapBytes;
    req.hasOverlapBytes = true;
  }
  if (patch.persistentMode !== undefined) {
    req.persistentMode = patch.persistentMode;
    req.hasPersistentMode = true;
  }
  if (patch.wakeWordEnabled !== undefined) {
    req.wakeWordEnabled = patch.wakeWordEnabled;
    req.hasWakeWordEnabled = true;
  }
  if (patch.wakeWordThreshold !== undefined) {
    req.wakeWordThreshold = patch.wakeWordThreshold;
    req.hasWakeWordThreshold = true;
  }
  if (patch.segmentSilenceMs !== undefined) {
    req.segmentSilenceMs = patch.segmentSilenceMs;
    req.hasSegmentSilenceMs = true;
  }
  const resp = await voiceClient.updateStreamConfig(req as Parameters<typeof voiceClient.updateStreamConfig>[0]);
  return decodeStreamConfig(resp.config);
}

// Wake word template configuration
export interface WakeWordConfig {
  configured: boolean;
  template: import("../hooks/voice/wakeword/types").WakeWordTemplate | null;
}

function decodeWakeWord(cfg: { configured: boolean; templateJson: string } | undefined): WakeWordConfig {
  const configured = cfg?.configured ?? false;
  const tj = cfg?.templateJson ?? "";
  let template: import("../hooks/voice/wakeword/types").WakeWordTemplate | null = null;
  if (configured && tj) {
    try {
      template = JSON.parse(tj) as import("../hooks/voice/wakeword/types").WakeWordTemplate;
    } catch {
      template = null;
    }
  }
  return { configured, template };
}

export async function getWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await voiceClient.getWakeWordConfig({});
  return decodeWakeWord(resp.config);
}

export async function updateWakeWordConfig(
  template: import("../hooks/voice/wakeword/types").WakeWordTemplate,
): Promise<WakeWordConfig> {
  const resp = await voiceClient.updateWakeWordTemplate({ templateJson: JSON.stringify(template) });
  return decodeWakeWord(resp.config);
}

export async function deleteWakeWordConfig(): Promise<WakeWordConfig> {
  const resp = await voiceClient.deleteWakeWordTemplate({});
  return decodeWakeWord(resp.config);
}

export interface SpeakerVerificationConfig {
  enabled: boolean;
  profileIds: string[];
  threshold: number;
  mode: "off" | "filter" | "advisory";
  rejectBehavior: "drop" | "show-muted";
  fallbackWithoutVerification: boolean;
}

export interface SpeakerVerificationProfile {
  id: string;
  display_name: string;
  created_at: string;
  updated_at: string;
  model_name: string;
  embedding_dim: number;
  sample_rate: number;
  enrollment_audio_seconds: number;
  notes: string;
}

export interface SpeakerVerificationInfo {
  backend: string;
  model: string;
  device: string;
  sample_rate: number;
  version: string;
  embedding_dim: number;
}

export interface SpeakerVerificationStatusResponse {
  config: SpeakerVerificationConfig;
  capability: CapabilityStatus;
  capabilityLabel?: string;
  resourceReady: boolean;
  profileConfigured: boolean;
  profileExists: boolean;
  profileCount: number;
  profiles?: SpeakerVerificationProfile[];
  info?: SpeakerVerificationInfo;
  checkedAt: string;
}

export interface SpeakerVerificationEnrollmentResponse {
  profile_id: string;
  display_name: string;
  embedding_dim: number;
  sample_rate: number;
  enrollment_audio_seconds: number;
  model_name: string;
  created_at: string;
}

export interface SpeakerVerificationEnrollResult {
  enrollment: SpeakerVerificationEnrollmentResponse;
  config: SpeakerVerificationConfig;
}

function decodeSpeakerConfig(c: {
  enabled: boolean;
  profileIds: string[];
  threshold: number;
  mode: string;
  rejectBehavior: string;
  fallbackWithoutVerification: boolean;
} | undefined): SpeakerVerificationConfig {
  const mode = (c?.mode ?? "filter") as SpeakerVerificationConfig["mode"];
  const reject = (c?.rejectBehavior ?? "drop") as SpeakerVerificationConfig["rejectBehavior"];
  return {
    enabled: c?.enabled ?? false,
    profileIds: c?.profileIds ?? [],
    threshold: c?.threshold ?? 0,
    mode,
    rejectBehavior: reject,
    fallbackWithoutVerification: c?.fallbackWithoutVerification ?? false,
  };
}

function decodeSpeakerProfile(p: {
  id: string;
  displayName: string;
  createdAt: string;
  updatedAt: string;
  modelName: string;
  embeddingDim: number;
  sampleRate: number;
  enrollmentAudioSeconds: number;
  notes: string;
}): SpeakerVerificationProfile {
  return {
    id: p.id,
    display_name: p.displayName,
    created_at: p.createdAt,
    updated_at: p.updatedAt,
    model_name: p.modelName,
    embedding_dim: p.embeddingDim,
    sample_rate: p.sampleRate,
    enrollment_audio_seconds: p.enrollmentAudioSeconds,
    notes: p.notes,
  };
}

export async function getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.getSpeakerConfig({});
  return decodeSpeakerConfig(resp.config);
}

export async function updateSpeakerVerificationConfig(
  patch: Partial<SpeakerVerificationConfig>,
): Promise<SpeakerVerificationConfig> {
  const req: Record<string, unknown> = {};
  if (patch.enabled !== undefined) {
    req.enabled = patch.enabled;
    req.hasEnabled = true;
  }
  if (patch.profileIds !== undefined) {
    req.profileIds = patch.profileIds;
    req.hasProfileIds = true;
  }
  if (patch.threshold !== undefined) {
    req.threshold = patch.threshold;
    req.hasThreshold = true;
  }
  if (patch.mode !== undefined) {
    req.mode = patch.mode;
    req.hasMode = true;
  }
  if (patch.rejectBehavior !== undefined) {
    req.rejectBehavior = patch.rejectBehavior;
    req.hasRejectBehavior = true;
  }
  if (patch.fallbackWithoutVerification !== undefined) {
    req.fallbackWithoutVerification = patch.fallbackWithoutVerification;
    req.hasFallbackWithoutVerification = true;
  }
  const resp = await voiceClient.updateSpeakerConfig(req as Parameters<typeof voiceClient.updateSpeakerConfig>[0]);
  return decodeSpeakerConfig(resp.config);
}

export async function getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
  const resp = await voiceClient.getSpeakerStatus({});
  const st = resp.status;
  if (!st) throw new Error("speaker status response missing status field");
  return {
    config: decodeSpeakerConfig(st.config),
    capability: st.capability as CapabilityStatus,
    capabilityLabel: st.capabilityLabel || undefined,
    resourceReady: st.resourceReady,
    profileConfigured: st.profileConfigured,
    profileExists: st.profileExists,
    profileCount: st.profileCount,
    profiles: st.profiles?.map(decodeSpeakerProfile),
    info: st.info
      ? {
          backend: st.info.backend,
          model: st.info.model,
          device: st.info.device,
          sample_rate: st.info.sampleRate,
          version: st.info.version,
          embedding_dim: st.info.embeddingDim,
        }
      : undefined,
    checkedAt: st.checkedAt,
  };
}

export async function listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
  const resp = await voiceClient.listSpeakerProfiles({});
  return resp.profiles.map(decodeSpeakerProfile);
}

export async function enrollSpeakerVerificationProfile(args: {
  audioBlob: Blob;
  profileId?: string;
  displayName?: string;
  notes?: string;
  addToActive?: boolean;
  enable?: boolean;
}): Promise<SpeakerVerificationEnrollResult> {
  const req: Record<string, unknown> = {
    audio: await blobToBytes(args.audioBlob),
    contentType: args.audioBlob.type,
    profileId: args.profileId ?? "",
    displayName: args.displayName ?? "",
    notes: args.notes ?? "",
  };
  if (args.addToActive !== undefined) {
    req.addToActive = args.addToActive;
    req.hasAddToActive = true;
  }
  if (args.enable !== undefined) {
    req.enable = args.enable;
    req.hasEnable = true;
  }
  const resp = await voiceClient.enrollSpeakerProfile(
    req as Parameters<typeof voiceClient.enrollSpeakerProfile>[0],
  );
  const en = resp.enrollment;
  return {
    enrollment: {
      profile_id: en?.profileId ?? "",
      display_name: en?.displayName ?? "",
      embedding_dim: en?.embeddingDim ?? 0,
      sample_rate: en?.sampleRate ?? 0,
      enrollment_audio_seconds: en?.enrollmentAudioSeconds ?? 0,
      model_name: en?.modelName ?? "",
      created_at: en?.createdAt ?? "",
    },
    config: decodeSpeakerConfig(resp.config),
  };
}

export async function clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.clearSpeakerProfileBinding({});
  return decodeSpeakerConfig(resp.config);
}

export async function removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.removeSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}

export async function deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const resp = await voiceClient.deleteSpeakerProfile({ profileId });
  return decodeSpeakerConfig(resp.config);
}

/** Maximum time (ms) to wait for Kokoro to return synthesized audio.
 * Bumped from 15 s to 30 s: with chunked input the payload is smaller,
 * but Kokoro can still be slow on cold-start or under load. */
const TTS_SYNTHESIS_TIMEOUT_MS = 30_000;

/** Synthesize text to audio via the Kokoro TTS backend. */
export async function synthesizeTTS(
  input: string,
  voice?: string,
  speed?: number,
  signal?: AbortSignal,
): Promise<Blob> {
  // Combine caller-provided abort signal with a hard timeout so a stalled
  // Kokoro server doesn't leave speech hanging silently for 60+ seconds.
  const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
  const resp = await ttsClient.synthesize(
    {
      input,
      voice: voice ?? "",
      responseFormat: "mp3",
      speed: speed ?? 0,
      eventId: "",
      version: "",
    },
    { signal: combined },
  );
  return new Blob([resp.audio as Uint8Array<ArrayBuffer>], {
    type: resp.contentType || "audio/mpeg",
  });
}

/**
 * Fetch pre-cached TTS audio for an event. Returns the audio Blob on cache
 * hit, or null on cache miss. Used by the cache-first playback path to
 * eliminate synthesis latency on tab switch.
 */
export async function fetchCachedTTS(
  eventId: string,
  voice: string,
  speed: number,
  version: "active" | "original" = "active",
  signal?: AbortSignal,
): Promise<Blob | null> {
  try {
    const resp = await ttsClient.getCache(
      { eventId, voice, speed, version },
      { signal },
    );
    if (resp.audio.byteLength === 0) return null;
    return new Blob([resp.audio as Uint8Array<ArrayBuffer>], {
      type: resp.contentType || "audio/mpeg",
    });
  } catch {
    return null;
  }
}

export interface TTSVoiceInfo {
  id: string;
  name: string;
}

/** Fetch available TTS voices from the Kokoro backend. */
export async function getTTSVoices(): Promise<TTSVoiceInfo[]> {
  const resp = await ttsClient.listVoices({});
  return resp.voices.map((v) => ({ id: v.id, name: v.name }));
}

// TTS auto-speak configuration (server-side)
export interface TTSConfig {
  autoEnabled: boolean;
  backend: "auto" | "kokoro" | "browser";
  kokoroVoice: string;
  kokoroSpeed: number;
}

export interface TTSRoutingResult {
  appended: boolean;
  code: string;
  reason: string;
  source: string;
  sessionId?: string;
  eventId?: string;
  sequence?: number;
  duplicate?: boolean;
}

export interface TTSClientAck {
  eventId: string;
  source: string;
  sessionId: string;
  stage: string;
  backend?: string;
  message?: string;
}

export interface TTSStatus {
  config: TTSConfig;
  hookRegistered: boolean;
  hookCode?: string;
  hookReason: string;
  hookSettingsPath?: string;
  lastRouting?: TTSRoutingResult;
  lastRoutingAt?: string;
  lastHookRouting?: TTSRoutingResult;
  lastHookRoutingAt?: string;
  lastTailerRouting?: TTSRoutingResult;
  lastTailerRoutingAt?: string;
  lastAck?: TTSClientAck;
  lastAckAt?: string;
  lastHookAck?: TTSClientAck;
  lastHookAckAt?: string;
  lastTailerAck?: TTSClientAck;
  lastTailerAckAt?: string;
  lastPlaybackEvent?: TTSPlaybackEvent;
  lastPlaybackAt?: string;
  kokoroCapability?: string;
  kokoroCapabilityLabel?: string;
}

function decodeTTSConfig(c: {
  autoEnabled: boolean;
  backend: string;
  kokoroVoice: string;
  kokoroSpeed: number;
} | undefined): TTSConfig {
  return {
    autoEnabled: c?.autoEnabled ?? false,
    backend: ((c?.backend || "auto") as TTSConfig["backend"]),
    kokoroVoice: c?.kokoroVoice ?? "",
    kokoroSpeed: c?.kokoroSpeed ?? 1,
  };
}

export async function getTTSConfig(): Promise<TTSConfig> {
  const resp = await ttsClient.getConfig({});
  return decodeTTSConfig(resp.config);
}

export async function updateTTSConfig(
  patch: Partial<TTSConfig>,
): Promise<TTSConfig> {
  const req = {
    autoEnabled: patch.autoEnabled ?? false,
    hasAutoEnabled: patch.autoEnabled !== undefined,
    backend: patch.backend ?? "",
    hasBackend: patch.backend !== undefined,
    kokoroVoice: patch.kokoroVoice ?? "",
    hasKokoroVoice: patch.kokoroVoice !== undefined,
    kokoroSpeed: patch.kokoroSpeed ?? 0,
    hasKokoroSpeed: patch.kokoroSpeed !== undefined,
  };
  const resp = await ttsClient.updateConfig(req);
  return decodeTTSConfig(resp.config);
}

// TTS summarization configuration
export interface TTSSummarizeConfig {
  enabled: boolean;
  charThreshold: number;
  level: "light" | "moderate" | "heavy";
  model: string;
  timeoutSeconds: number;
}

function decodeSummarizeConfig(c: {
  enabled: boolean;
  charThreshold: number;
  level: string;
  model: string;
  timeoutSeconds: number;
} | undefined): TTSSummarizeConfig {
  return {
    enabled: c?.enabled ?? false,
    charThreshold: c?.charThreshold ?? 0,
    level: ((c?.level || "moderate") as TTSSummarizeConfig["level"]),
    model: c?.model ?? "",
    timeoutSeconds: c?.timeoutSeconds ?? 0,
  };
}

export async function getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
  const resp = await ttsClient.getSummarizeConfig({});
  return decodeSummarizeConfig(resp.config);
}

export async function updateTTSSummarizeConfig(
  patch: Partial<TTSSummarizeConfig>,
): Promise<TTSSummarizeConfig> {
  const req = {
    enabled: patch.enabled ?? false,
    hasEnabled: patch.enabled !== undefined,
    charThreshold: patch.charThreshold ?? 0,
    hasCharThreshold: patch.charThreshold !== undefined,
    level: patch.level ?? "",
    hasLevel: patch.level !== undefined,
    model: patch.model ?? "",
    hasModel: patch.model !== undefined,
    timeoutSeconds: patch.timeoutSeconds ?? 0,
    hasTimeoutSeconds: patch.timeoutSeconds !== undefined,
  };
  const resp = await ttsClient.updateSummarizeConfig(req);
  return decodeSummarizeConfig(resp.config);
}

export interface SummarizeEventResponse {
  summarized: boolean;
  speechParagraphs?: string[];
  error?: string;
}

export async function summarizeEvent(
  sessionId: string,
  eventId: string,
  signal?: AbortSignal,
): Promise<SummarizeEventResponse> {
  const resp = await conversationClient.summarizeEvent({ sessionId, eventId }, { signal });
  return {
    summarized: resp.summarized,
    speechParagraphs: resp.speechParagraphs.length > 0 ? resp.speechParagraphs : undefined,
    error: resp.error || undefined,
  };
}

function decodeRouting(r: {
  appended: boolean;
  code: string;
  reason: string;
  source: string;
  sessionId: string;
  eventId: string;
  sequence: bigint;
  duplicate: boolean;
} | undefined): TTSRoutingResult | undefined {
  if (!r || (!r.appended && !r.code && !r.source)) return undefined;
  return {
    appended: r.appended,
    code: r.code,
    reason: r.reason,
    source: r.source,
    sessionId: r.sessionId || undefined,
    eventId: r.eventId || undefined,
    sequence: r.sequence !== 0n ? Number(r.sequence) : undefined,
    duplicate: r.duplicate || undefined,
  };
}

function decodeAck(a: {
  eventId: string;
  source: string;
  sessionId: string;
  stage: string;
  backend: string;
  message: string;
} | undefined): TTSClientAck | undefined {
  if (!a || (!a.eventId && !a.source && !a.stage)) return undefined;
  return {
    eventId: a.eventId,
    source: a.source,
    sessionId: a.sessionId,
    stage: a.stage,
    backend: a.backend || undefined,
    message: a.message || undefined,
  };
}

function decodePlayback(p: {
  source: string;
  stage: string;
  backend: string;
  sessionId: string;
  message: string;
} | undefined): TTSPlaybackEvent | undefined {
  if (!p || (!p.source && !p.stage)) return undefined;
  return {
    source: p.source,
    stage: p.stage,
    backend: p.backend || undefined,
    sessionId: p.sessionId || undefined,
    message: p.message || undefined,
  };
}

export async function getTTSStatus(): Promise<TTSStatus> {
  const resp = await ttsClient.getStatus({});
  const st = resp.status;
  if (!st) {
    throw new Error("TTS status response missing payload");
  }
  return {
    config: decodeTTSConfig(st.config),
    hookRegistered: st.hookRegistered,
    hookCode: st.hookCode || undefined,
    hookReason: st.hookReason,
    hookSettingsPath: st.hookSettingsPath || undefined,
    lastRouting: decodeRouting(st.lastRouting),
    lastRoutingAt: st.lastRoutingAt || undefined,
    lastHookRouting: decodeRouting(st.lastHookRouting),
    lastHookRoutingAt: st.lastHookRoutingAt || undefined,
    lastTailerRouting: decodeRouting(st.lastTailerRouting),
    lastTailerRoutingAt: st.lastTailerRoutingAt || undefined,
    lastAck: decodeAck(st.lastAck),
    lastAckAt: st.lastAckAt || undefined,
    lastHookAck: decodeAck(st.lastHookAck),
    lastHookAckAt: st.lastHookAckAt || undefined,
    lastTailerAck: decodeAck(st.lastTailerAck),
    lastTailerAckAt: st.lastTailerAckAt || undefined,
    lastPlaybackEvent: decodePlayback(st.lastPlaybackEvent),
    lastPlaybackAt: st.lastPlaybackAt || undefined,
    kokoroCapability: st.kokoroCapability || undefined,
    kokoroCapabilityLabel: st.kokoroCapabilityLabel || undefined,
  };
}

export async function transcribeAudioWithRetry(audioBlob: Blob, maxAttempts = 2, language?: string): Promise<string> {
  let lastError: unknown;
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    try {
      return await transcribeAudio(audioBlob, language);
    } catch (err) {
      lastError = err;
      if (attempt < maxAttempts - 1) {
        await new Promise((r) => setTimeout(r, 500 * (attempt + 1)));
      }
    }
  }
  throw lastError;
}

// Workspace layout (cross-device sync)
export interface WorkspacePaneDTO {
  session_id: string;
  name: string;
  header_color: string;
  theme_id: string;
  font_size: number;
  sort_order: number;
  group_id: string | null;
  supports_messages_view: boolean;
}

export interface TabGroupDTO {
  id: string;
  name: string;
  color: string;
  sort_order: number;
  is_collapsed: boolean;
}

export interface WorkspaceLayoutDTO {
  active_pane: string;
  panes: WorkspacePaneDTO[];
  groups: TabGroupDTO[];
}

// Workspace functions below are shims over Connect-RPC WorkspaceService.
// The legacy snake_case DTOs are preserved so existing call-sites don't
// have to change; the Connect-Web client lives in ../api/workspace.

function decodePane(p: {
  sessionId: string;
  name: string;
  headerColor: string;
  themeId: string;
  fontSize: number;
  sortOrder: number;
  groupId: string;
  supportsMessagesView: boolean;
}): WorkspacePaneDTO {
  return {
    session_id: p.sessionId,
    name: p.name,
    header_color: p.headerColor,
    theme_id: p.themeId,
    font_size: p.fontSize,
    sort_order: p.sortOrder,
    group_id: p.groupId === "" ? null : p.groupId,
    supports_messages_view: p.supportsMessagesView,
  };
}

function decodeGroup(g: {
  id: string;
  name: string;
  color: string;
  sortOrder: number;
  isCollapsed: boolean;
}): TabGroupDTO {
  return {
    id: g.id,
    name: g.name,
    color: g.color,
    sort_order: g.sortOrder,
    is_collapsed: g.isCollapsed,
  };
}

export async function getWorkspaceLayout(): Promise<WorkspaceLayoutDTO> {
  const resp = await workspaceClient.getLayout({});
  return {
    active_pane: resp.activePane,
    panes: resp.panes.map(decodePane),
    groups: resp.groups.map(decodeGroup),
  };
}

export async function saveWorkspaceLayout(req: {
  active_pane: string | null;
  pane_order: string[];
}): Promise<void> {
  await workspaceClient.saveLayout({
    activePane: req.active_pane ?? "",
    paneOrder: req.pane_order,
  });
}

export async function updateWorkspacePane(
  sessionId: string,
  update: Partial<Omit<WorkspacePaneDTO, "session_id">>,
): Promise<WorkspacePaneDTO> {
  const req: Parameters<typeof workspaceClient.updatePane>[0] = { sessionId };
  if (update.name !== undefined) {
    req.name = update.name;
    req.hasName = true;
  }
  if (update.header_color !== undefined) {
    req.headerColor = update.header_color;
    req.hasHeaderColor = true;
  }
  if (update.theme_id !== undefined) {
    req.themeId = update.theme_id;
    req.hasThemeId = true;
  }
  if (update.font_size !== undefined) {
    req.fontSize = update.font_size;
    req.hasFontSize = true;
  }
  if (update.sort_order !== undefined) {
    req.sortOrder = update.sort_order;
    req.hasSortOrder = true;
  }
  if (update.group_id !== undefined) {
    req.groupId = update.group_id ?? "";
    req.hasGroupId = true;
  }
  if (update.supports_messages_view !== undefined) {
    req.supportsMessagesView = update.supports_messages_view;
    req.hasSupportsMessagesView = true;
  }
  const resp = await workspaceClient.updatePane(req);
  if (!resp.pane) {
    throw new Error("workspace.updatePane: missing pane in response");
  }
  return decodePane(resp.pane);
}

export async function deleteWorkspacePane(sessionId: string): Promise<void> {
  await workspaceClient.deletePane({ sessionId });
}

export async function createTabGroup(req: {
  name: string;
  color: string;
}): Promise<TabGroupDTO> {
  const resp = await workspaceClient.createGroup(req);
  if (!resp.group) {
    throw new Error("workspace.createGroup: missing group in response");
  }
  return decodeGroup(resp.group);
}

export async function updateTabGroup(
  id: string,
  update: Partial<Omit<TabGroupDTO, "id">>,
): Promise<TabGroupDTO> {
  const req: Parameters<typeof workspaceClient.updateGroup>[0] = { id };
  if (update.name !== undefined) {
    req.name = update.name;
    req.hasName = true;
  }
  if (update.color !== undefined) {
    req.color = update.color;
    req.hasColor = true;
  }
  if (update.is_collapsed !== undefined) {
    req.isCollapsed = update.is_collapsed;
    req.hasIsCollapsed = true;
  }
  const resp = await workspaceClient.updateGroup(req);
  if (!resp.group) {
    throw new Error("workspace.updateGroup: missing group in response");
  }
  return decodeGroup(resp.group);
}

export async function deleteTabGroup(id: string): Promise<void> {
  await workspaceClient.deleteGroup({ id });
}

// Session defaults settings — backed by Connect-RPC SettingsService.
// The legacy snake_case shape is preserved here so existing call-sites
// don't have to change; the Connect-Web client lives in
// src/api/settings.ts.
import { aiClient } from "../api/ai";
import { conversationClient } from "../api/conversation";
import { sessionsClient } from "../api/sessions";
import { capabilitiesClient } from "../api/capabilities";
import { settingsClient } from "../api/settings";
import { shortcutsClient } from "../api/shortcuts";
import { ttsClient } from "../api/tts";
import { voiceClient } from "../api/voice";
import { workspaceClient } from "../api/workspace";

export interface SessionDefaultsResponse {
  default_backend: string;
  default_policy: ExpirationPolicy;
}

function decodePolicy(p: { mode?: string; duration?: string } | undefined): ExpirationPolicy {
  return {
    mode: ((p?.mode as ExpirationPolicy["mode"]) || "never"),
    duration: p?.duration || "",
  };
}

export async function getSessionDefaults(): Promise<SessionDefaultsResponse> {
  const resp = await settingsClient.getSessionDefaults({});
  const d = resp.defaults;
  return {
    default_backend: d?.defaultBackend ?? "",
    default_policy: decodePolicy(d?.defaultPolicy),
  };
}

export async function updateSessionDefaults(update: {
  default_backend?: string;
  default_policy?: ExpirationPolicy;
}): Promise<SessionDefaultsResponse> {
  const req: {
    defaultBackend?: string;
    defaultPolicy?: { mode: string; duration: string };
  } = {};
  if (update.default_backend !== undefined) {
    req.defaultBackend = update.default_backend;
  }
  if (update.default_policy !== undefined) {
    req.defaultPolicy = {
      mode: update.default_policy.mode,
      duration: update.default_policy.duration ?? "",
    };
  }
  const resp = await settingsClient.updateSessionDefaults(req);
  const d = resp.defaults;
  return {
    default_backend: d?.defaultBackend ?? "",
    default_policy: decodePolicy(d?.defaultPolicy),
  };
}
