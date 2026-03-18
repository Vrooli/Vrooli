import { useCallback, useEffect } from "react";
import type { ConversationEvent, ConversationCursor } from "../lib/api";
import { getConversationSession, updateConversationCursor } from "../lib/api";
import { useConversationStore } from "../stores/useConversationStore";

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

  return {
    appendConversationEvent,
    persistCursor,
  };
}
