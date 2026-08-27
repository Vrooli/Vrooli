import { useEffect, useRef } from "react";
import type { ConversationEvent } from "../api/conversation";
import { API_BASE_WITH_SUFFIX } from "../api/client";
import { coerceOriginName, type BackendID, type SessionInfo } from "../api/sessions";
import { useConversationStore } from "../stores/useConversationStore";
import { useLiveStreamStore } from "../stores/useLiveStreamStore";
import { refreshConversationSession } from "./useConversationSession";

/**
 * useGlobalEventStream — the single SSE subscription for the whole app.
 *
 * Conversation events (assistant/user messages + async summarize updates) used
 * to ride the per-session terminal WebSocket, which coupled unread badges to
 * having a terminal mounted. They now flow over ONE process-wide SSE channel
 * (`GET /api/v1/events/stream`), so the UI subscribes once for ALL sessions and
 * badges keep updating even when their terminal is unmounted (the core enabler
 * for unmounting offscreen sessions — see the multi-session perf overhaul).
 *
 * Dispatch is idempotent by the monotonic global event id: the browser's native
 * EventSource reconnect replays missed events via `Last-Event-ID`, and the
 * server may overlap one event on reconnect, so we dedupe to never double-apply.
 * `appendEvent`/`updateEvent` also dedupe by event id as belt-and-suspenders.
 */

interface GlobalEventEnvelopeBase {
  id: number;
  session_id: string;
  sequence: number;
}

// Discriminated on `kind` so each variant carries its real payload shape: the
// conversation kinds a ConversationEventPayload, session_status a
// SessionStatusPayload. This lets the dispatch switch narrow `envelope.payload`
// per case without any cast.
type GlobalEventEnvelope =
  | (GlobalEventEnvelopeBase & {
      kind: "conversation_event" | "conversation_event_update" | "conversation_out_of_sync";
      payload: ConversationEventPayload;
    })
  | (GlobalEventEnvelopeBase & {
      kind: "session_status";
      payload: SessionStatusPayload;
    });

interface ConversationEventPayload {
  id?: string;
  source?: string;
  role?: string;
  text?: string;
  speechParagraphs?: string[];
  originalSpeechParagraphs?: string[];
  summarized?: boolean;
  summarizeError?: string;
  createdAt?: string;
  sequence?: number;
}

/**
 * Payload of a `session_status` envelope: a session existence transition
 * (created/deleted/terminated) fanned from the API's event logger onto the SSE
 * hub (see api/session_lifecycle_bridge.go). The "created" action carries enough
 * to build a full sidebar row without a follow-up fetch; delete/terminate carry
 * just the action (terminate adds a reason).
 */
interface SessionStatusPayload {
  action?: "created" | "deleted" | "terminated";
  shell?: string;
  cols?: number;
  rows?: number;
  backend?: string;
  origin?: string;
  owner?: string;
  display_label?: string;
  agent?: string;
  recovered?: boolean;
  created_at?: string;
  reason?: string;
}

/** Lifecycle callbacks for externally originated session existence changes. */
export interface SessionLifecycleHandlers {
  /** An externally created session (any origin) should be merged into state.
   *  `supportsMessagesView` is derived from the launch agent so the merged pane
   *  matches a locally launched one. */
  onSessionCreated?: (session: SessionInfo, supportsMessagesView: boolean) => void;
  /** An externally deleted/terminated session should be dropped from state. */
  onSessionEnded?: (sessionId: string, reason: "deleted" | "terminated") => void;
}

export interface UseGlobalEventStreamOptions extends SessionLifecycleHandlers {
  /** Surface an auto-summarize failure for the active pane (banner + retry). */
  onSummarizeError?: (sessionId: string, eventId: string, message: string) => void;
  /** Test seam: inject a fake EventSource factory. */
  createEventSource?: (url: string) => EventSource;
}

/** Build a full SessionInfo from a `created` lifecycle payload so the sidebar
 *  merge needs no follow-up fetch. Fields absent from the event (policy, busy)
 *  take neutral defaults; survives_restart is derived from the backend. */
function sessionFromStatusPayload(sessionId: string, p: SessionStatusPayload): SessionInfo {
  return {
    id: sessionId,
    shell: p.shell ?? "",
    created_at: p.created_at ?? new Date().toISOString(),
    cols: p.cols ?? 0,
    rows: p.rows ?? 0,
    backend: (p.backend as BackendID) || "standard",
    survives_restart: p.backend === "persistent",
    policy: { mode: "never" },
    ...(p.recovered ? { recovered: true } : {}),
    origin: coerceOriginName(p.origin),
    owner: p.owner ?? "",
    display_label: p.display_label ?? "",
  };
}

/** Builds the same-origin SSE endpoint URL (honors proxy/api-base resolution). */
export function buildEventStreamUrl(): string {
  return `${API_BASE_WITH_SUFFIX}/events/stream`;
}

const SEEN_ID_LIMIT = 4096;

// Backoff for the reconnects this hook owns. EventSource handles its own
// retries while CONNECTING; these cover the CLOSED case, where it does not.
const RECONNECT_BASE_DELAY_MS = 1000;
const RECONNECT_MAX_DELAY_MS = 30_000;

/**
 * dispatchGlobalEvent applies one parsed envelope to the conversation store.
 * Exported for unit tests (feed synthetic envelopes incl. replayed overlaps and
 * assert no duplicate appends / correct unread counts).
 */
export function dispatchGlobalEvent(
  envelope: GlobalEventEnvelope,
  onSummarizeError?: (sessionId: string, eventId: string, message: string) => void,
  lifecycle?: SessionLifecycleHandlers,
): void {
  const store = useConversationStore.getState();
  const { session_id: sessionId } = envelope;

  // Switch on envelope.kind (not a destructured copy) so TS narrows
  // envelope.payload to the per-kind shape inside each case — no casts.
  switch (envelope.kind) {
    case "conversation_event": {
      const event = toConversationEvent(sessionId, envelope.sequence, envelope.payload);
      if (event) store.appendEvent(event);
      break;
    }
    case "conversation_event_update": {
      const { payload } = envelope;
      const eventId = payload.id;
      if (!eventId) break;
      const patch: { speechParagraphs?: string[]; originalSpeechParagraphs?: string[]; summarized?: boolean } = {};
      if (payload.speechParagraphs !== undefined) patch.speechParagraphs = payload.speechParagraphs;
      if (payload.originalSpeechParagraphs !== undefined) patch.originalSpeechParagraphs = payload.originalSpeechParagraphs;
      if (payload.summarized !== undefined) patch.summarized = payload.summarized;
      if (Object.keys(patch).length > 0) store.updateEvent(sessionId, eventId, patch);
      if (payload.summarizeError) onSummarizeError?.(sessionId, eventId, payload.summarizeError);
      break;
    }
    case "conversation_out_of_sync": {
      if (sessionId) {
        void refreshConversationSession(sessionId);
      } else {
        // Whole-stream gap: refetch every session we currently track.
        for (const id of Object.keys(store.sessions)) void refreshConversationSession(id);
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
      } else if (p.action === "deleted" || p.action === "terminated") {
        lifecycle?.onSessionEnded?.(sessionId, p.action);
      }
      break;
    }
  }
}

function toConversationEvent(
  sessionId: string,
  fallbackSequence: number,
  payload: ConversationEventPayload,
): ConversationEvent | null {
  if (!payload.id || !payload.text) return null;
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

export function useGlobalEventStream(options: UseGlobalEventStreamOptions = {}): void {
  const { onSummarizeError, onSessionCreated, onSessionEnded, createEventSource } = options;
  // Keep the latest callbacks in refs so re-creating them doesn't tear down and
  // re-open the SSE connection.
  const onSummarizeErrorRef = useRef(onSummarizeError);
  onSummarizeErrorRef.current = onSummarizeError;
  const lifecycleRef = useRef<SessionLifecycleHandlers>({ onSessionCreated, onSessionEnded });
  lifecycleRef.current = { onSessionCreated, onSessionEnded };

  useEffect(() => {
    if (typeof EventSource === "undefined" && !createEventSource) return;

    const url = buildEventStreamUrl();
    const { setStatus } = useLiveStreamStore.getState();

    // Dedupe by monotonic global id across reconnect replays. Shared across
    // reconnects so a replayed backlog is not re-applied after every drop.
    const seen = new Set<number>();
    const order: number[] = [];
    const markSeen = (id: number): boolean => {
      if (seen.has(id)) return false;
      seen.add(id);
      order.push(id);
      if (order.length > SEEN_ID_LIMIT) {
        const evicted = order.shift();
        if (evicted !== undefined) seen.delete(evicted);
      }
      return true;
    };

    const handle = (raw: MessageEvent) => {
      let envelope: GlobalEventEnvelope;
      try {
        envelope = JSON.parse(raw.data as string) as GlobalEventEnvelope;
      } catch {
        return;
      }
      // Dedupe only real (monotonic, >0) global ids across reconnect replays.
      // out-of-sync nudges are advisory and intentionally carry id:0 — they must
      // always be processed, never deduped against a prior nudge.
      if (typeof envelope.id === "number" && envelope.id > 0 && !markSeen(envelope.id)) return;
      dispatchGlobalEvent(envelope, onSummarizeErrorRef.current, lifecycleRef.current);
    };

    const kinds = [
      "conversation_event",
      "conversation_event_update",
      "conversation_out_of_sync",
      "session_status",
    ] as const;

    let source: EventSource | null = null;
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let attempt = 0;
    let disposed = false;

    const teardown = () => {
      if (!source) return;
      for (const kind of kinds) source.removeEventListener(kind, handle as EventListener);
      source.onopen = null;
      source.onerror = null;
      source.close();
      source = null;
    };

    const scheduleReconnect = () => {
      if (disposed || retryTimer) return;
      // EventSource retries on its own only while it is CONNECTING. Once it
      // reaches CLOSED it has given up permanently, and before this existed
      // nothing ever reopened it: live updates stayed dead until the page was
      // reloaded. That is the "I have to leave the view and come back" symptom.
      const delay = Math.min(RECONNECT_BASE_DELAY_MS * 2 ** attempt, RECONNECT_MAX_DELAY_MS);
      attempt += 1;
      retryTimer = setTimeout(() => {
        retryTimer = null;
        connect();
      }, delay);
    };

    function connect(): void {
      if (disposed) return;
      teardown();
      const next = createEventSource ? createEventSource(url) : new EventSource(url);
      source = next;
      for (const kind of kinds) next.addEventListener(kind, handle as EventListener);

      next.onopen = () => {
        attempt = 0;
        setStatus("open");
        // A reconnect may have missed events entirely. Refetching every tracked
        // session is the same repair the server's out-of-sync notice triggers,
        // applied to the case where the notice itself was what we missed. Each
        // refetch is gap-aware, so this is cheap when nothing was missed.
        for (const id of Object.keys(useConversationStore.getState().sessions)) {
          void refreshConversationSession(id);
        }
      };

      next.onerror = () => {
        if (disposed) return;
        // readyState CLOSED (2) means the browser has stopped trying; anything
        // else means it is still retrying on its own and we must not stack a
        // second connection on top of it.
        setStatus("reconnecting");
        if (next.readyState === 2) scheduleReconnect();
      };
    }

    // Mobile browsers suspend background connections aggressively and do not
    // always resume them. Re-checking on foreground turns a silently dead
    // stream into a reconnect the moment the user looks at the page again.
    const onVisible = () => {
      if (document.hidden || disposed) return;
      if (useLiveStreamStore.getState().status === "open") return;
      if (retryTimer) {
        clearTimeout(retryTimer);
        retryTimer = null;
      }
      attempt = 0;
      connect();
    };

    setStatus("connecting");
    connect();
    document.addEventListener("visibilitychange", onVisible);
    window.addEventListener("online", onVisible);

    return () => {
      disposed = true;
      document.removeEventListener("visibilitychange", onVisible);
      window.removeEventListener("online", onVisible);
      if (retryTimer) clearTimeout(retryTimer);
      teardown();
      useLiveStreamStore.getState().setStatus("closed");
    };
  }, [createEventSource]);
}
