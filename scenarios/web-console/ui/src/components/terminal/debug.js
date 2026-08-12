/**
 * Developer-observability probe for the terminal session layer.
 *
 * Populates a single `window.__wc_terminal_debug` object with live
 * snapshots of:
 *   - connection state (per session)
 *   - pending stdin queue
 *   - pending-ack map size
 *   - current alt-buffer state
 *   - last N coalesce events
 *
 * This is NOT part of any user-facing API. It exists so manual repro
 * and tests can introspect state without reaching into React internals.
 * Nothing in production code depends on this probe; it is a read-only
 * sink.
 */
const NOOP_PROBE = {
    sessions: {},
    update: () => { },
    remove: () => { },
};
function installProbe() {
    const sessions = {};
    const probe = {
        sessions,
        update: (snapshot) => {
            sessions[snapshot.sessionId] = snapshot;
        },
        remove: (sessionId) => {
            delete sessions[sessionId];
        },
    };
    return probe;
}
/**
 * Returns the live probe singleton. On non-browser environments or
 * during SSR the probe is a no-op that silently discards updates.
 */
export function getTerminalDebugProbe() {
    if (typeof window === "undefined")
        return NOOP_PROBE;
    const w = window;
    if (!w.__wc_terminal_debug) {
        w.__wc_terminal_debug = installProbe();
    }
    return w.__wc_terminal_debug;
}
