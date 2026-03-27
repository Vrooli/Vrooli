// DOC: docs/concepts/ARCHITECTURE.md#session-creation
// DOC: docs/internal/ERROR-SEMANTICS.md#structured-error-type-typescript
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

export interface SessionInfo {
  id: string;
  shell: string;
  created_at: string;
  cols: number;
  rows: number;
  policy: ExpirationPolicy;
  busy: boolean;
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

export async function createSession(opts?: { shell?: string; cols?: number; rows?: number }): Promise<SessionInfo> {
  const url = buildApiUrl("/sessions", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(opts ?? {}),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to create session");
  }
  return (await res.json()) as SessionInfo;
}

export async function listSessions(): Promise<SessionInfo[]> {
  const url = buildApiUrl("/sessions", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to list sessions");
  }
  return (await res.json()) as SessionInfo[];
}

export async function getSession(id: string): Promise<SessionInfo> {
  const url = buildApiUrl(`/sessions/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get session");
  }
  return (await res.json()) as SessionInfo;
}

export async function deleteSession(id: string): Promise<void> {
  const url = buildApiUrl(`/sessions/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to delete session");
  }
}

// [REQ:P1-001a] Session Policy API - client
export async function getSessionPolicy(id: string): Promise<PolicyResponse> {
  const url = buildApiUrl(`/sessions/${id}/policy`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get session policy");
  }
  return (await res.json()) as PolicyResponse;
}

export async function updateSessionPolicy(
  id: string,
  policy: { mode: string; duration?: string },
): Promise<PolicyResponse> {
  const url = buildApiUrl(`/sessions/${id}/policy`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(policy),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to update session policy");
  }
  return (await res.json()) as PolicyResponse;
}

// [REQ:P0-005a] AI Command Generation API - client
export interface AIGenerateResponse {
  command: string;
  provider: string;
}

export async function generateAICommand(prompt: string, context?: string): Promise<AIGenerateResponse> {
  const url = buildApiUrl("/ai/generate", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt, context }),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "AI command generation failed");
  }
  return (await res.json()) as AIGenerateResponse;
}

// [REQ:P0-005a] AI Suggestion API - client
export interface AISuggestResponse {
  commands: string[];
  provider: string;
}

export async function generateAISuggestions(prompt: string, context?: string): Promise<AISuggestResponse> {
  const url = buildApiUrl("/ai/suggest", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ prompt, context }),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "AI suggestion failed");
  }
  return (await res.json()) as AISuggestResponse;
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

export interface ConversationSessionResponse {
  sessionId: string;
  events: ConversationEvent[];
  cursor: ConversationCursor;
}

export async function reportTTSEvent(event: TTSPlaybackEvent): Promise<void> {
  const url = buildApiUrl("/tts/events", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(event),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to report TTS event");
  }
}

export async function getConversationSession(sessionId: string): Promise<ConversationSessionResponse> {
  const url = buildApiUrl(`/sessions/${sessionId}/conversation`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get conversation session");
  }
  return (await res.json()) as ConversationSessionResponse;
}

export async function updateConversationCursor(
  sessionId: string,
  patch: Partial<ConversationCursor>,
): Promise<ConversationCursor> {
  const url = buildApiUrl(`/sessions/${sessionId}/conversation/cursor`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to update conversation cursor");
  }
  return (await res.json()) as ConversationCursor;
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

export async function getEffectiveShortcuts(): Promise<ShortcutEntry[]> {
  const url = buildApiUrl("/shortcuts", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get shortcuts");
  }
  return (await res.json()) as ShortcutEntry[];
}

export async function listShortcutProfiles(): Promise<ShortcutProfile[]> {
  const url = buildApiUrl("/shortcuts/profiles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to list shortcut profiles");
  }
  return (await res.json()) as ShortcutProfile[];
}

export async function upsertShortcutProfile(profile: {
  id: string;
  scope: string;
  name: string;
  shortcuts: ShortcutEntry[];
}): Promise<ShortcutProfile> {
  const url = buildApiUrl("/shortcuts/profiles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(profile),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to save shortcut profile");
  }
  return (await res.json()) as ShortcutProfile;
}

export async function deleteShortcutProfile(id: string): Promise<void> {
  const url = buildApiUrl(`/shortcuts/profiles/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to delete shortcut profile");
  }
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

export async function getAIConfig(): Promise<AIProviderConfigResponse> {
  const url = buildApiUrl("/ai/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get AI config");
  }
  return (await res.json()) as AIProviderConfigResponse;
}

export async function updateAIConfig(update: {
  name: string;
  enabled?: boolean;
  priority?: number;
  timeout_sec?: number;
  max_retries?: number;
}): Promise<AIProviderConfigResponse> {
  const url = buildApiUrl("/ai/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(update),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to update AI config");
  }
  return (await res.json()) as AIProviderConfigResponse;
}

export async function getAIHealth(): Promise<ProviderHealth[]> {
  const url = buildApiUrl("/ai/health", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get AI health");
  }
  return (await res.json()) as ProviderHealth[];
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
}

export async function fetchCapabilities(): Promise<CapabilitiesResponse> {
  const url = buildApiUrl("/capabilities", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to fetch capabilities");
  }
  return (await res.json()) as CapabilitiesResponse;
}

/** Fast liveness-only capability check (GET health only, no test transcription). */
export async function fetchCapabilitiesLiveness(): Promise<CapabilitiesResponse> {
  const url = buildApiUrl("/capabilities/liveness", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to fetch capabilities liveness");
  }
  return (await res.json()) as CapabilitiesResponse;
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

/** Reset the capabilities liveness cache. Exported for tests. */
export function _resetCapabilitiesCache(): void {
  _capCache = null;
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

export async function transcribeAudio(audioBlob: Blob, language?: string): Promise<string> {
  let path = "/voice/transcribe";
  if (language) path += `?language=${encodeURIComponent(language)}`;
  const url = buildApiUrl(path, { baseUrl: API_BASE });
  const formData = new FormData();
  formData.append("audio_file", audioBlob, "recording.webm");
  const res = await fetch(url, {
    method: "POST",
    body: formData,
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Voice transcription failed");
  }
  const data = (await res.json()) as { text: string };
  return data.text;
}

// Voice streaming configuration (server-side pipeline tuning)
export interface VoiceStreamConfig {
  flushIntervalMs: number;
  minDeltaBytes: number;
  overlapBytes: number;
  persistentMode: boolean;
  commandPrefix: string;
  segmentSilenceMs: number;
}

export async function getVoiceStreamConfig(): Promise<VoiceStreamConfig> {
  const url = buildApiUrl("/voice/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get voice config");
  return (await res.json()) as VoiceStreamConfig;
}

export async function updateVoiceStreamConfig(
  patch: Partial<VoiceStreamConfig>,
): Promise<VoiceStreamConfig> {
  const url = buildApiUrl("/voice/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to update voice config");
  return (await res.json()) as VoiceStreamConfig;
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

export async function getSpeakerVerificationConfig(): Promise<SpeakerVerificationConfig> {
  const url = buildApiUrl("/voice/speaker/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get speaker verification config");
  return (await res.json()) as SpeakerVerificationConfig;
}

export async function updateSpeakerVerificationConfig(
  patch: Partial<SpeakerVerificationConfig>,
): Promise<SpeakerVerificationConfig> {
  const url = buildApiUrl("/voice/speaker/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to update speaker verification config");
  return (await res.json()) as SpeakerVerificationConfig;
}

export async function getSpeakerVerificationStatus(): Promise<SpeakerVerificationStatusResponse> {
  const url = buildApiUrl("/voice/speaker/status", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get speaker verification status");
  return (await res.json()) as SpeakerVerificationStatusResponse;
}

export async function listSpeakerVerificationProfiles(): Promise<SpeakerVerificationProfile[]> {
  const url = buildApiUrl("/voice/speaker/profiles", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to list speaker verification profiles");
  const data = (await res.json()) as { profiles: SpeakerVerificationProfile[] };
  return data.profiles;
}

export async function enrollSpeakerVerificationProfile(args: {
  audioBlob: Blob;
  profileId?: string;
  displayName?: string;
  notes?: string;
  addToActive?: boolean;
  enable?: boolean;
}): Promise<SpeakerVerificationEnrollResult> {
  const url = buildApiUrl("/voice/speaker/enroll", { baseUrl: API_BASE });
  const formData = new FormData();
  formData.append("audio_file", args.audioBlob, "speaker-enrollment.webm");
  if (args.profileId) formData.append("profileId", args.profileId);
  if (args.displayName) formData.append("displayName", args.displayName);
  if (args.notes) formData.append("notes", args.notes);
  if (args.addToActive !== undefined) formData.append("addToActive", String(args.addToActive));
  if (args.enable !== undefined) formData.append("enable", String(args.enable));

  const res = await fetch(url, {
    method: "POST",
    body: formData,
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to enroll speaker verification profile");
  return (await res.json()) as SpeakerVerificationEnrollResult;
}

export async function clearSpeakerVerificationProfile(): Promise<SpeakerVerificationConfig> {
  const url = buildApiUrl("/voice/speaker/profile", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to clear active speaker profile");
  return (await res.json()) as SpeakerVerificationConfig;
}

export async function removeSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const url = buildApiUrl("/voice/speaker/profile/remove", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ profileId }),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to remove speaker profile");
  return (await res.json()) as SpeakerVerificationConfig;
}

export async function deleteSpeakerVerificationProfile(profileId: string): Promise<SpeakerVerificationConfig> {
  const url = buildApiUrl("/voice/speaker/profile/delete", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ profileId }),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to delete speaker profile");
  return (await res.json()) as SpeakerVerificationConfig;
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
  const url = buildApiUrl("/tts/synthesize", { baseUrl: API_BASE });
  // Combine caller-provided abort signal with a hard timeout so a stalled
  // Kokoro server doesn't leave speech hanging silently for 60+ seconds.
  const timeout = AbortSignal.timeout(TTS_SYNTHESIS_TIMEOUT_MS);
  const combined = signal ? AbortSignal.any([signal, timeout]) : timeout;
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      input,
      voice,
      speed,
      response_format: "mp3",
    }),
    signal: combined,
  });
  if (!res.ok) throw new Error("TTS synthesis failed");
  return res.blob();
}

/**
 * Fetch pre-cached TTS audio for an event. Returns the audio Blob on cache
 * hit, or null on cache miss (404). Used by the cache-first playback path
 * to eliminate synthesis latency on tab switch.
 */
export async function fetchCachedTTS(
  eventId: string,
  voice: string,
  speed: number,
  version: "active" | "original" = "active",
  signal?: AbortSignal,
): Promise<Blob | null> {
  const params = new URLSearchParams({ voice, speed: String(speed), version });
  const url = buildApiUrl(`/tts/cache/${eventId}?${params}`, { baseUrl: API_BASE });
  try {
    const resp = await fetch(url, { signal });
    if (resp.status === 404) return null;
    if (!resp.ok) return null;
    return resp.blob();
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
  const url = buildApiUrl("/tts/voices", { baseUrl: API_BASE });
  const res = await fetch(url, { cache: "no-store" });
  if (!res.ok) throw new Error("Failed to get TTS voices");
  return (await res.json()) as TTSVoiceInfo[];
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

export async function getTTSConfig(): Promise<TTSConfig> {
  const url = buildApiUrl("/tts/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get TTS config");
  return (await res.json()) as TTSConfig;
}

export async function updateTTSConfig(
  patch: Partial<TTSConfig>,
): Promise<TTSConfig> {
  const url = buildApiUrl("/tts/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to update TTS config");
  return (await res.json()) as TTSConfig;
}

// TTS summarization configuration
export interface TTSSummarizeConfig {
  enabled: boolean;
  charThreshold: number;
  level: "light" | "moderate" | "heavy";
  model: string;
  timeoutSeconds: number;
}

export async function getTTSSummarizeConfig(): Promise<TTSSummarizeConfig> {
  const url = buildApiUrl("/tts/summarize/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get TTS summarize config");
  return (await res.json()) as TTSSummarizeConfig;
}

export async function updateTTSSummarizeConfig(
  patch: Partial<TTSSummarizeConfig>,
): Promise<TTSSummarizeConfig> {
  const url = buildApiUrl("/tts/summarize/config", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to update TTS summarize config");
  return (await res.json()) as TTSSummarizeConfig;
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
  const url = buildApiUrl(`/sessions/${sessionId}/conversation/${eventId}/summarize`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    signal,
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to summarize event");
  return (await res.json()) as SummarizeEventResponse;
}

export async function getTTSStatus(): Promise<TTSStatus> {
  const url = buildApiUrl("/tts/status", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw await extractAPIError(res, "Failed to get TTS status");
  return (await res.json()) as TTSStatus;
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

export async function getWorkspaceLayout(): Promise<WorkspaceLayoutDTO> {
  const url = buildApiUrl("/workspace/layout", { baseUrl: API_BASE });
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to get workspace layout");
  }
  return (await res.json()) as WorkspaceLayoutDTO;
}

export async function saveWorkspaceLayout(req: {
  active_pane: string | null;
  pane_order: string[];
}): Promise<void> {
  const url = buildApiUrl("/workspace/layout", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to save workspace layout");
  }
}

export async function updateWorkspacePane(
  sessionId: string,
  update: Partial<Omit<WorkspacePaneDTO, "session_id">>,
): Promise<WorkspacePaneDTO> {
  const url = buildApiUrl(`/workspace/panes/${sessionId}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(update),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to update workspace pane");
  }
  return (await res.json()) as WorkspacePaneDTO;
}

export async function deleteWorkspacePane(sessionId: string): Promise<void> {
  const url = buildApiUrl(`/workspace/panes/${sessionId}`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to delete workspace pane");
  }
}

export async function createTabGroup(req: {
  name: string;
  color: string;
}): Promise<TabGroupDTO> {
  const url = buildApiUrl("/workspace/groups", { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to create tab group");
  }
  return (await res.json()) as TabGroupDTO;
}

export async function updateTabGroup(
  id: string,
  update: Partial<Omit<TabGroupDTO, "id">>,
): Promise<TabGroupDTO> {
  const url = buildApiUrl(`/workspace/groups/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(update),
  });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to update tab group");
  }
  return (await res.json()) as TabGroupDTO;
}

export async function deleteTabGroup(id: string): Promise<void> {
  const url = buildApiUrl(`/workspace/groups/${id}`, { baseUrl: API_BASE });
  const res = await fetch(url, { method: "DELETE" });
  if (!res.ok) {
    throw await extractAPIError(res, "Failed to delete tab group");
  }
}
