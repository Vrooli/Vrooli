import { useCallback, useEffect } from "react";
import type { ConversationEvent, ConversationCursor } from "../api/conversation";
import { getConversationSession, updateConversationCursor } from "../api/conversation";
import { getSessionRefetchSinceSequence, useConversationStore } from "../stores/useConversationStore";
import { describeLoadFailure, PAGE_SIZE, type RefreshOutcome } from "../lib/conversationLoad";

export { PAGE_SIZE, describeLoadFailure } from "../lib/conversationLoad";
export type { RefreshOutcome } from "../lib/conversationLoad";

const olderPageLoads = new Set<string>();

/**
 * refreshConversationSession fetches the events the local store is missing
 * and merges them. Uses gap-aware since_sequence: if a sequence gap exists
 * (e.g., from a hydrate/append race or a dropped WS event), refetch starts
 * from before the gap so it gets backfilled — not just the tail. Safe to
 * call from any trigger — reconnect, view-open, visibility change, or a
 * server-sent conversation_out_of_sync notice.
 *
 * Both outcomes are recorded in the store: a success clears any prior error
 * and stores the server's capture diagnosis, and a failure is kept so the view
 * can explain itself rather than falling back to an empty state.
 */
export async function refreshConversationSession(sessionId: string): Promise<RefreshOutcome> {
  const state = useConversationStore.getState();
  const existing = state.sessions[sessionId];
  const since = getSessionRefetchSinceSequence(state, sessionId);
  const before = existing?.events.length ?? 0;
  state.beginLoad(sessionId);
  try {
    const data = await getConversationSession(
      sessionId,
      existing ? { sinceSequence: since } : { limit: PAGE_SIZE },
    );
    if (existing) {
      state.mergeEvents(sessionId, data.events, data.cursor, data.capture);
    } else {
      state.hydrateSession(sessionId, data.events, data.cursor, {
        oldestSequence: data.oldestSequence ?? data.events[0]?.sequence ?? 0,
        hasOlder: data.hasMore ?? false,
        totalCount: data.totalCount ?? data.events.length,
        capture: data.capture,
      });
    }
    const after = useConversationStore.getState().sessions[sessionId]?.events.length ?? 0;
    return { ok: true, addedEvents: Math.max(0, after - before) };
  } catch (error) {
    const described = describeLoadFailure(error);
    useConversationStore.getState().failLoad(sessionId, described);
    return { ok: false, error: described };
  }
}

export async function loadOlderConversationPage(sessionId: string): Promise<boolean> {
  const state = useConversationStore.getState();
  const session = state.sessions[sessionId];
  if (!session?.hasOlder || !session.windowOldestSequence || olderPageLoads.has(sessionId)) return false;
  olderPageLoads.add(sessionId);
  try {
    const data = await getConversationSession(sessionId, { limit: PAGE_SIZE, beforeSequence: session.windowOldestSequence });
    const knownEventIds = session.knownEventIds ?? new Set(session.events.map((event) => event.id));
    const added = data.events.some((event) => !knownEventIds.has(event.id));
    state.prependEvents(sessionId, data.events, { oldestSequence: data.oldestSequence ?? 0, hasOlder: data.hasMore ?? false, totalCount: data.totalCount ?? session.totalCount ?? session.events.length, capture: data.capture });
    return added;
  } catch {
    return false;
  } finally {
    olderPageLoads.delete(sessionId);
  }
}

/** Replaces the bounded window with the page that contains sequence. */
export async function loadConversationPageContaining(sessionId: string, sequence: number): Promise<boolean> {
  if (sequence <= 0) return false;
  const state = useConversationStore.getState();
  const beforeSequence = sequence + Math.floor(PAGE_SIZE / 2) + 1;
  try {
    const data = await getConversationSession(sessionId, { limit: PAGE_SIZE, beforeSequence });
    if (!data.events.some((event) => event.sequence === sequence)) return false;
    state.setSessionWindow(sessionId, data.events, data.cursor, {
      oldestSequence: data.oldestSequence ?? data.events[0]?.sequence ?? 0,
      hasOlder: data.hasMore ?? false,
      totalCount: data.totalCount ?? data.events.length,
      capture: data.capture,
    });
    return true;
  } catch {
    return false;
  }
}

export function useConversationSession(sessionId: string, options: { hydrate?: boolean } = {}) {
  const appendEvent = useConversationStore((state) => state.appendEvent);
  const setCursor = useConversationStore((state) => state.updateCursor);

  useEffect(() => {
    if (options.hydrate === false) return;
    // refreshConversationSession already owns the loading, success and failure
    // transitions, and writes them to the shared store. Hydrating through it
    // means a failed first load reports a failure instead of installing an
    // empty session, which is what previously made a server error look like an
    // empty conversation. No unmount guard is needed: the writes are keyed by
    // session id and idempotent, so a late resolution is still correct.
    void refreshConversationSession(sessionId);
  }, [options.hydrate, sessionId]);

  const appendConversationEvent = useCallback((event: ConversationEvent) => {
    appendEvent(event);
  }, [appendEvent]);

  const persistCursor = useCallback(async (cursor: Partial<ConversationCursor>) => {
    setCursor(sessionId, cursor);
    try {
      const updated = await updateConversationCursor(sessionId, cursor);
      setCursor(sessionId, updated);
    } catch {
      // Best effort: keep local optimistic state.
    }
  }, [sessionId, setCursor]);

  const refresh = useCallback(() => refreshConversationSession(sessionId), [sessionId]);

  return {
    appendConversationEvent,
    persistCursor,
    refresh,
  };
}
