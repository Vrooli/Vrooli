import { useCallback, useEffect } from "react";
import type { ConversationEvent, ConversationCursor } from "../api/conversation";
import { getConversationSession, updateConversationCursor } from "../api/conversation";
import { getSessionRefetchSinceSequence, useConversationStore } from "../stores/useConversationStore";

export const PAGE_SIZE = 500;
const olderPageLoads = new Set<string>();

/**
 * refreshConversationSession fetches the events the local store is missing
 * and merges them. Uses gap-aware since_sequence: if a sequence gap exists
 * (e.g., from a hydrate/append race or a dropped WS event), refetch starts
 * from before the gap so it gets backfilled — not just the tail. Safe to
 * call from any trigger — reconnect, view-open, visibility change, or a
 * server-sent conversation_out_of_sync notice. Returns true if the fetch
 * succeeded (even if zero events were added).
 */
export async function refreshConversationSession(sessionId: string): Promise<boolean> {
  const state = useConversationStore.getState();
  const existing = state.sessions[sessionId];
  const since = getSessionRefetchSinceSequence(state, sessionId);
  try {
    const data = await getConversationSession(sessionId, existing ? { sinceSequence: since } : { limit: PAGE_SIZE });
    if (existing) state.mergeEvents(sessionId, data.events, data.cursor);
    else state.hydrateSession(sessionId, data.events, data.cursor, { oldestSequence: data.oldestSequence ?? data.events[0]?.sequence ?? 0, hasOlder: data.hasMore ?? false, totalCount: data.totalCount ?? data.events.length });
    return true;
  } catch {
    return false;
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
    state.prependEvents(sessionId, data.events, { oldestSequence: data.oldestSequence ?? 0, hasOlder: data.hasMore ?? false, totalCount: data.totalCount ?? session.totalCount ?? session.events.length });
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
    });
    return true;
  } catch {
    return false;
  }
}

export function useConversationSession(sessionId: string, options: { hydrate?: boolean } = {}) {
  const hydrateSession = useConversationStore((state) => state.hydrateSession);
  const appendEvent = useConversationStore((state) => state.appendEvent);
  const setCursor = useConversationStore((state) => state.updateCursor);

  useEffect(() => {
    if (options.hydrate === false) return;
    let cancelled = false;
    const load = async () => {
      try {
        const data = await getConversationSession(sessionId, { limit: PAGE_SIZE });
        if (!cancelled) {
          hydrateSession(sessionId, data.events, data.cursor, { oldestSequence: data.oldestSequence ?? data.events[0]?.sequence ?? 0, hasOlder: data.hasMore ?? false, totalCount: data.totalCount ?? data.events.length });
        }
      } catch {
        if (!cancelled) {
          hydrateSession(sessionId, [], { lastSeenSequence: 0, lastListenedSequence: 0 });
        }
      }
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [hydrateSession, options.hydrate, sessionId]);

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
