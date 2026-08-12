import { createClient } from "@connectrpc/connect";
import { CapabilitiesService } from "@vrooli/proto-types/web-console/v1/capabilities/capabilities_pb";
import { transport } from "./client";
export const capabilitiesClient = createClient(CapabilitiesService, transport);
function decodeCapabilityStatus(s) {
    return s === "available" || s === "unavailable" ? s : "unknown";
}
function decodeCapabilitiesResponse(resp) {
    const out = {
        capabilities: resp.capabilities.map((c) => ({
            id: c.id,
            name: c.name,
            description: c.description,
            dependencyKind: c.dependencyKind,
            dependencySlug: c.dependencySlug,
            features: c.features ?? [],
            status: decodeCapabilityStatus(c.status),
            message: c.message || undefined,
            reasonCode: c.reasonCode || undefined,
            actionKind: c.actionKind || undefined,
            actionLabel: c.actionLabel || undefined,
            operatorCommand: c.operatorCommand || undefined,
            checkedAt: c.checkedAt || undefined,
        })),
        timestamp: resp.timestamp,
    };
    if (resp.sessionBackends && resp.sessionBackends.length > 0) {
        out.session_backends = resp.sessionBackends.map((b) => ({
            id: b.id,
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
export async function fetchCapabilities() {
    const resp = await capabilitiesClient.get({});
    return decodeCapabilitiesResponse(resp);
}
export async function runCapabilityAction(capabilityId, actionKind) {
    const resp = await capabilitiesClient.runAction({ capabilityId, actionKind });
    const decoded = decodeCapabilitiesResponse({
        capabilities: resp.capabilities,
        timestamp: resp.timestamp,
    });
    return {
        success: resp.success,
        status: resp.status,
        message: resp.message || undefined,
        capabilityId: resp.capabilityId,
        actionKind: resp.actionKind,
        capabilities: decoded.capabilities,
        timestamp: decoded.timestamp,
    };
}
/** Fast liveness-only capability check (GET health only, no test transcription). */
export async function fetchCapabilitiesLiveness() {
    const resp = await capabilitiesClient.liveness({});
    return decodeCapabilitiesResponse(resp);
}
// Concurrent and near-simultaneous callers share a single in-flight request.
// Cache TTL matches the server-side capability cache (30 s).
let _capCache = null;
const CAP_CACHE_TTL = 30000;
export function fetchCapabilitiesLivenessCached() {
    const now = Date.now();
    if (_capCache && now - _capCache.at < CAP_CACHE_TTL)
        return _capCache.promise;
    const promise = fetchCapabilitiesLiveness();
    _capCache = { promise, at: now };
    promise.catch(() => {
        if (_capCache?.promise === promise)
            _capCache = null;
    });
    return promise;
}
let _capSnapshot = null;
/**
 * Synchronous snapshot of the most recent capabilities liveness result.
 * Used by startRecording() to avoid blocking the mic activation hot path.
 * DOC: docs/internal/VOICE-LATENCY.md#background-capability-check
 */
export function getCapabilitiesLivenessSnapshot() {
    if (!_capCache)
        return null;
    if (Date.now() - _capCache.at >= CAP_CACHE_TTL)
        return null;
    return _capSnapshot;
}
export async function refreshCapabilitiesLiveness() {
    const result = await fetchCapabilitiesLivenessCached();
    _capSnapshot = result;
    return result;
}
/** Reset the capabilities liveness cache. Exported for tests. */
export function _resetCapabilitiesCache() {
    _capCache = null;
    _capSnapshot = null;
}
