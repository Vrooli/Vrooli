import { createClient } from "@connectrpc/connect";
import { SessionsService, SessionOrigin } from "@vrooli/proto-types/web-console/v1/sessions/sessions_pb";
import { resolveApiBase, buildWsUrl } from "@vrooli/api-base";
import { transport } from "./client";
// sessionsClient is the Connect-Web client for SessionsService. Consumers
// should prefer the typed wrappers below, which surface the snake_case
// shapes the UI components and stores expect.
export const sessionsClient = createClient(SessionsService, transport);
/** Map the SessionOrigin proto enum to the string form the sidebar buckets on. */
export function originName(o) {
    switch (o) {
        case SessionOrigin.UI:
            return "ui";
        case SessionOrigin.PROGRAMMATIC:
            return "programmatic";
        case SessionOrigin.REMOTE:
            return "remote";
        default:
            return "unspecified";
    }
}
/** Coerce a free-form origin string (e.g. from the SSE lifecycle payload) into a
 *  closed SessionOriginName, defaulting unknown/empty to "unspecified". */
export function coerceOriginName(s) {
    switch (s) {
        case "ui":
        case "programmatic":
        case "remote":
            return s;
        default:
            return "unspecified";
    }
}
function decodeSession(s) {
    const policy = s?.policy;
    return {
        id: s?.id ?? "",
        shell: s?.shell ?? "",
        created_at: s?.createdAt ?? "",
        cols: s?.cols ?? 0,
        rows: s?.rows ?? 0,
        backend: s?.backend ?? "standard",
        survives_restart: s?.survivesRestart ?? false,
        policy: {
            mode: policy?.mode ?? "never",
            ...(policy?.duration ? { duration: policy.duration } : {}),
        },
        busy: s?.busy ?? false,
        ...(s?.recovered ? { recovered: true } : {}),
        origin: originName(s?.origin),
        owner: s?.owner ?? "",
        display_label: s?.displayLabel ?? "",
        ...(s?.trackingDegraded ? { tracking_degraded: true } : {}),
    };
}
function decodeRecoverable(r) {
    return {
        id: r.id,
        backend: r.backend || undefined,
        shell: r.shell || undefined,
        cols: r.cols || undefined,
        rows: r.rows || undefined,
        created_at: r.createdAt || undefined,
        orphaned_at: r.orphanedAt || undefined,
        last_activity_at: r.lastActivityAt || undefined,
        agent_type: r.agentType || undefined,
        agent_session_id: r.agentSessionId || undefined,
        launch_command: r.launchCommand || undefined,
        cwd: r.cwd || undefined,
        last_rollout_path: r.lastRolloutPath || undefined,
        recoverable: r.recoverable,
        not_recoverable_reason: r.notRecoverableReason || undefined,
        pane_name: r.paneName || undefined,
        header_color: r.headerColor || undefined,
        group_name: r.groupName || undefined,
    };
}
function decodePolicyView(v) {
    return {
        session_id: v?.sessionId ?? "",
        policy: {
            mode: v?.policy?.mode ?? "never",
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
// ---------------------------------------------------------------------------
// Typed wrappers
// ---------------------------------------------------------------------------
export async function createSession(opts) {
    const resp = await sessionsClient.create({
        shell: opts?.shell ?? "",
        cols: opts?.cols ?? 0,
        rows: opts?.rows ?? 0,
        backend: opts?.backend ?? "",
        launchCommand: opts?.launch_command ?? "",
        executeLaunchCommand: opts?.execute_launch_command ?? false,
        agentType: opts?.agent_type ?? "",
        // First-party UI client: tag provenance explicitly so an origin-less
        // create (which the server normalizes to programmatic) can only come from
        // a non-UI caller.
        origin: SessionOrigin.UI,
        ...(opts?.policy
            ? { policy: { mode: opts.policy.mode, duration: opts.policy.duration ?? "" }, hasPolicy: true }
            : {}),
    });
    return decodeSession(resp.session);
}
export async function listSessionsWithRecovery() {
    const resp = await sessionsClient.list({});
    const r = resp.recovery;
    return {
        sessions: resp.sessions.map(decodeSession),
        recovery: {
            in_progress: r?.inProgress ?? false,
            total: r?.total ?? 0,
            recovered: r?.recovered ?? 0,
            awaiting_recovery: r?.awaitingRecovery ?? 0,
            adopted: r?.adopted ?? 0,
        },
    };
}
export async function listSessions() {
    return (await listSessionsWithRecovery()).sessions;
}
export async function getSession(id) {
    const resp = await sessionsClient.get({ id });
    return decodeSession(resp.session);
}
export async function deleteSession(id) {
    await sessionsClient.delete({ id });
}
export async function listRecoverableSessions() {
    const resp = await sessionsClient.listRecoverable({});
    return resp.sessions.map(decodeRecoverable);
}
export async function recoverSession(oldId, idempotencyKey) {
    const resp = await sessionsClient.recover({ id: oldId }, idempotencyKey ? { headers: { "X-Idempotency-Key": idempotencyKey } } : undefined);
    return {
        old_session_id: resp.oldSessionId,
        new_session_id: resp.newSessionId,
        agent_type: resp.agentType || undefined,
        command_sent: resp.commandSent || undefined,
        codex_home_copied: resp.codexHomeCopied,
    };
}
export async function dismissRecoverableSession(oldId) {
    await sessionsClient.dismissRecoverable({ id: oldId });
}
// [REQ:P1-001a] Session Policy API - client
export async function getSessionPolicy(id) {
    const resp = await sessionsClient.getPolicy({ id });
    return decodePolicyView(resp.policy);
}
export async function updateSessionPolicy(id, policy) {
    const resp = await sessionsClient.updatePolicy({
        id,
        policy: { mode: policy.mode, duration: policy.duration ?? "" },
    });
    return decodePolicyView(resp.policy);
}
// [REQ:P0-004b] api-base WebSocket Integration — session terminal WS URL.
// Kept in the sessions domain because the WS endpoint is part of the
// session lifecycle even though it bypasses Connect-RPC.
export function buildSessionWsUrl(sessionId, device) {
    const apiBase = resolveApiBase({ appendSuffix: true });
    const wsBase = apiBase.startsWith("https://")
        ? `wss://${apiBase.slice("https://".length)}`
        : apiBase.startsWith("http://")
            ? `ws://${apiBase.slice("http://".length)}`
            : apiBase;
    const url = buildWsUrl(`/sessions/${sessionId}/ws`, { baseUrl: wsBase });
    if (!device)
        return url;
    const query = new URLSearchParams({ deviceId: device.id, deviceLabel: device.label });
    return `${url}${url.includes("?") ? "&" : "?"}${query}`;
}
