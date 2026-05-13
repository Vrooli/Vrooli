import { createClient } from "@connectrpc/connect";
import { CapabilitiesService } from "@vrooli/proto-types/web-console/v1/capabilities/capabilities_pb";

import type { BackendID, BackendOption } from "./sessions";
import { transport } from "./client";

export const capabilitiesClient = createClient(CapabilitiesService, transport);

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

function decodeCapabilityStatus(s: string): CapabilityStatus {
  return s === "available" || s === "unavailable" ? s : "unknown";
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

export async function fetchCapabilities(): Promise<CapabilitiesResponse> {
  const resp = await capabilitiesClient.get({});
  return decodeCapabilitiesResponse(resp);
}

/** Fast liveness-only capability check (GET health only, no test transcription). */
export async function fetchCapabilitiesLiveness(): Promise<CapabilitiesResponse> {
  const resp = await capabilitiesClient.liveness({});
  return decodeCapabilitiesResponse(resp);
}

// Concurrent and near-simultaneous callers share a single in-flight request.
// Cache TTL matches the server-side capability cache (30 s).
let _capCache: { promise: Promise<CapabilitiesResponse>; at: number } | null = null;
const CAP_CACHE_TTL = 30_000;

export function fetchCapabilitiesLivenessCached(): Promise<CapabilitiesResponse> {
  const now = Date.now();
  if (_capCache && now - _capCache.at < CAP_CACHE_TTL) return _capCache.promise;
  const promise = fetchCapabilitiesLiveness();
  _capCache = { promise, at: now };
  promise.catch(() => {
    if (_capCache?.promise === promise) _capCache = null;
  });
  return promise;
}

let _capSnapshot: CapabilitiesResponse | null = null;

/**
 * Synchronous snapshot of the most recent capabilities liveness result.
 * Used by startRecording() to avoid blocking the mic activation hot path.
 * DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
 */
export function getCapabilitiesLivenessSnapshot(): CapabilitiesResponse | null {
  if (!_capCache) return null;
  if (Date.now() - _capCache.at >= CAP_CACHE_TTL) return null;
  return _capSnapshot;
}

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
