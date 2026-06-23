import { useEffect, useRef } from "react";
import { resolveApiBase } from "@vrooli/api-base";
import type { ConversationEvent } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";
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

interface GlobalEventEnvelope {
  id: number;
  session_id: string;
  kind: "conversation_event" | "conversation_event_update" | "conversation_out_of_sync" | "session_status";
  sequence: number;
  payload: ConversationEventPayload;
}

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

export interface UseGlobalEventStreamOptions {
  /** Surface an auto-summarize failure for the active pane (banner + retry). */
  onSummarizeError?: (sessionId: string, eventId: string, message: string) => void;
  /** Test seam: inject a fake EventSource factory. */
  createEventSource?: (url: string) => EventSource;
}

/** Builds the same-origin SSE endpoint URL (honors proxy/api-base resolution). */
export function buildEventStreamUrl(): string {
  const base = resolveApiBase({ appendSuffix: true });
  return `${base}/events/stream`;
}

const SEEN_ID_LIMIT = 4096;

/**
 * dispatchGlobalEvent applies one parsed envelope to the conversation store.
 * Exported for unit tests (feed synthetic envelopes incl. replayed overlaps and
 * assert no duplicate appends / correct unread counts).
 */
export function dispatchGlobalEvent(
  envelope: GlobalEventEnvelope,
  onSummarizeError?: (sessionId: string, eventId: string, message: string) => void,
): void {
  const store = useConversationStore.getState();
  const { kind, session_id: sessionId, payload } = envelope;

  switch (kind) {
    case "conversation_event": {
      const event = toConversationEvent(sessionId, envelope.sequence, payload);
      if (event) store.appendEvent(event);
      break;
    }
    case "conversation_event_update": {
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
    case "session_status":
      // Reserved for forward compatibility (e.g. live exit status for
      // unmounted panes). No-op until the server emits it.
      break;
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
  const { onSummarizeError, createEventSource } = options;
  // Keep the latest onSummarizeError in a ref so re-creating the callback
  // doesn't tear down and re-open the SSE connection.
  const onSummarizeErrorRef = useRef(onSummarizeError);
  onSummarizeErrorRef.current = onSummarizeError;

  useEffect(() => {
    if (typeof EventSource === "undefined" && !createEventSource) return;

    const url = buildEventStreamUrl();
    const source = createEventSource ? createEventSource(url) : new EventSource(url);

    // Dedupe by monotonic global id across reconnect replays.
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
      dispatchGlobalEvent(envelope, onSummarizeErrorRef.current);
    };

    const kinds = [
      "conversation_event",
      "conversation_event_update",
      "conversation_out_of_sync",
      "session_status",
    ] as const;
    for (const kind of kinds) source.addEventListener(kind, handle as EventListener);

    return () => {
      for (const kind of kinds) source.removeEventListener(kind, handle as EventListener);
      source.close();
    };
  }, [createEventSource]);
}
