import { useEffect, useRef } from "react";
import { resolveApiBase } from "@vrooli/api-base";
import { coerceOriginName } from "../api/sessions";
import { useConversationStore } from "../stores/useConversationStore";
import { refreshConversationSession } from "./useConversationSession";
/** Build a full SessionInfo from a `created` lifecycle payload so the sidebar
 *  merge needs no follow-up fetch. Fields absent from the event (policy, busy)
 *  take neutral defaults; survives_restart is derived from the backend. */
function sessionFromStatusPayload(sessionId, p) {
    return {
        id: sessionId,
        shell: p.shell ?? "",
        created_at: p.created_at ?? new Date().toISOString(),
        cols: p.cols ?? 0,
        rows: p.rows ?? 0,
        backend: p.backend || "standard",
        survives_restart: p.backend === "persistent",
        policy: { mode: "never" },
        busy: false,
        ...(p.recovered ? { recovered: true } : {}),
        origin: coerceOriginName(p.origin),
        owner: p.owner ?? "",
        display_label: p.display_label ?? "",
    };
}
/** Builds the same-origin SSE endpoint URL (honors proxy/api-base resolution). */
export function buildEventStreamUrl() {
    const base = resolveApiBase({ appendSuffix: true });
    return `${base}/events/stream`;
}
const SEEN_ID_LIMIT = 4096;
/**
 * dispatchGlobalEvent applies one parsed envelope to the conversation store.
 * Exported for unit tests (feed synthetic envelopes incl. replayed overlaps and
 * assert no duplicate appends / correct unread counts).
 */
export function dispatchGlobalEvent(envelope, onSummarizeError, lifecycle) {
    const store = useConversationStore.getState();
    const { session_id: sessionId } = envelope;
    // Switch on envelope.kind (not a destructured copy) so TS narrows
    // envelope.payload to the per-kind shape inside each case — no casts.
    switch (envelope.kind) {
        case "conversation_event": {
            const event = toConversationEvent(sessionId, envelope.sequence, envelope.payload);
            if (event)
                store.appendEvent(event);
            break;
        }
        case "conversation_event_update": {
            const { payload } = envelope;
            const eventId = payload.id;
            if (!eventId)
                break;
            const patch = {};
            if (payload.speechParagraphs !== undefined)
                patch.speechParagraphs = payload.speechParagraphs;
            if (payload.originalSpeechParagraphs !== undefined)
                patch.originalSpeechParagraphs = payload.originalSpeechParagraphs;
            if (payload.summarized !== undefined)
                patch.summarized = payload.summarized;
            if (Object.keys(patch).length > 0)
                store.updateEvent(sessionId, eventId, patch);
            if (payload.summarizeError)
                onSummarizeError?.(sessionId, eventId, payload.summarizeError);
            break;
        }
        case "conversation_out_of_sync": {
            if (sessionId) {
                void refreshConversationSession(sessionId);
            }
            else {
                // Whole-stream gap: refetch every session we currently track.
                for (const id of Object.keys(store.sessions))
                    void refreshConversationSession(id);
            }
            break;
        }
        case "session_status": {
            // Session existence transitions from ANY origin (another tab, the CLI, an
            // expiry sweep). The bridge (api/session_lifecycle_bridge.go) fans these
            // onto the hub so the sidebar tracks sessions this client did not create.
            const p = envelope.payload;
            if (p.action === "created") {
                const supportsMessagesView = Boolean(p.agent) && p.agent !== "none";
                lifecycle?.onSessionCreated?.(sessionFromStatusPayload(sessionId, p), supportsMessagesView);
            }
            else if (p.action === "deleted" || p.action === "terminated") {
                lifecycle?.onSessionEnded?.(sessionId, p.action);
            }
            break;
        }
    }
}
function toConversationEvent(sessionId, fallbackSequence, payload) {
    if (!payload.id || !payload.text)
        return null;
    return {
        id: payload.id,
        sessionId,
        source: payload.source ?? "",
        role: payload.role === "user" ? "user" : "assistant",
        text: payload.text,
        speechParagraphs: payload.speechParagraphs ?? [payload.text],
        originalSpeechParagraphs: payload.originalSpeechParagraphs,
        summarized: payload.summarized ?? false,
        createdAt: payload.createdAt ?? new Date().toISOString(),
        sequence: payload.sequence ?? fallbackSequence,
        deliveryState: "received",
        ttsState: "idle",
        consumptionState: "unseen",
    };
}
export function useGlobalEventStream(options = {}) {
    const { onSummarizeError, onSessionCreated, onSessionEnded, createEventSource } = options;
    // Keep the latest callbacks in refs so re-creating them doesn't tear down and
    // re-open the SSE connection.
    const onSummarizeErrorRef = useRef(onSummarizeError);
    onSummarizeErrorRef.current = onSummarizeError;
    const lifecycleRef = useRef({ onSessionCreated, onSessionEnded });
    lifecycleRef.current = { onSessionCreated, onSessionEnded };
    useEffect(() => {
        if (typeof EventSource === "undefined" && !createEventSource)
            return;
        const url = buildEventStreamUrl();
        const source = createEventSource ? createEventSource(url) : new EventSource(url);
        // Dedupe by monotonic global id across reconnect replays.
        const seen = new Set();
        const order = [];
        const markSeen = (id) => {
            if (seen.has(id))
                return false;
            seen.add(id);
            order.push(id);
            if (order.length > SEEN_ID_LIMIT) {
                const evicted = order.shift();
                if (evicted !== undefined)
                    seen.delete(evicted);
            }
            return true;
        };
        const handle = (raw) => {
            let envelope;
            try {
                envelope = JSON.parse(raw.data);
            }
            catch {
                return;
            }
            // Dedupe only real (monotonic, >0) global ids across reconnect replays.
            // out-of-sync nudges are advisory and intentionally carry id:0 — they must
            // always be processed, never deduped against a prior nudge.
            if (typeof envelope.id === "number" && envelope.id > 0 && !markSeen(envelope.id))
                return;
            dispatchGlobalEvent(envelope, onSummarizeErrorRef.current, lifecycleRef.current);
        };
        const kinds = [
            "conversation_event",
            "conversation_event_update",
            "conversation_out_of_sync",
            "session_status",
        ];
        for (const kind of kinds)
            source.addEventListener(kind, handle);
        source.onerror = (event) => {
            console.warn("[web-console] global event stream error", { url, event });
        };
        return () => {
            for (const kind of kinds)
                source.removeEventListener(kind, handle);
            source.onerror = null;
            source.close();
        };
    }, [createEventSource]);
}
