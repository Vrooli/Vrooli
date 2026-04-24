import { useCallback, useEffect } from "react";
import type { ConversationEvent, ConversationCursor } from "../lib/api";
import { getConversationSession, updateConversationCursor } from "../lib/api";
import { useConversationStore } from "../stores/useConversationStore";

/**
 * refreshConversationSession fetches only the events the local store is
 * missing (via ?since_sequence=<max local sequence>) and merges them. Safe to
 * call from any trigger — reconnect, view-open, visibility change, or a
 * server-sent conversation_out_of_sync notice. Returns true if the fetch
 * succeeded (even if zero events were added).
 */
export async function refreshConversationSession(sessionId: string): Promise<boolean> {
  const state = useConversationStore.getState();
  const session = state.sessions[sessionId];
  const maxSeq = session?.events.reduce((m, e) => (e.sequence > m ? e.sequence : m), 0) ?? 0;
  try {
    const data = await getConversationSession(sessionId, { sinceSequence: maxSeq });
    state.mergeEvents(sessionId, data.events, data.cursor);
    return true;
  } catch {
    return false;
  }
}

export function useConversationSession(sessionId: string) {
  const hydrateSession = useConversationStore((state) => state.hydrateSession);
  const appendEvent = useConversationStore((state) => state.appendEvent);
  const setCursor = useConversationStore((state) => state.updateCursor);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      try {
        const data = await getConversationSession(sessionId);
        if (!cancelled) {
          hydrateSession(sessionId, data.events, data.cursor);
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
  }, [hydrateSession, sessionId]);

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
