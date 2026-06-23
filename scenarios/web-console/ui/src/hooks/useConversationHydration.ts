import { useEffect } from "react";
import { getConversationSession } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";

/**
 * useConversationHydration ensures every known session has its conversation
 * history loaded into the store ONCE, independent of whether its terminal pane
 * is currently mounted.
 *
 * Previously each pane hydrated its own conversation on mount. Once offscreen
 * terminals are unmounted (multi-session perf overhaul, Layer 2), that per-pane
 * hydrate no longer runs for backgrounded sessions — so their unread badges
 * would start empty until the next live event. Hydrating centrally keeps every
 * tab's badge correct from load. Live updates then arrive via the global SSE
 * channel (see useGlobalEventStream).
 *
 * Idempotent: skips any session already hydrated, and the store's hydrate/merge
 * dedupe by event id, so overlap with a pane's own load is harmless.
 */
export function useConversationHydration(sessionIds: string[]): void {
  const hydrateSession = useConversationStore((s) => s.hydrateSession);
  // Join into a stable key so the effect only re-runs when the set changes,
  // not on every parent render (the array identity is unstable).
  const key = sessionIds.join("|");

  useEffect(() => {
    let cancelled = false;
    const ids = key ? key.split("|") : [];
    for (const id of ids) {
      if (!id) continue;
      if (useConversationStore.getState().sessions[id]?.hydrated) continue;
      void (async () => {
        try {
          const data = await getConversationSession(id);
          if (!cancelled) hydrateSession(id, data.events, data.cursor);
        } catch {
          if (!cancelled) hydrateSession(id, [], { lastSeenSequence: 0, lastListenedSequence: 0 });
        }
      })();
    }
    return () => { cancelled = true; };
  }, [key, hydrateSession]);
}
