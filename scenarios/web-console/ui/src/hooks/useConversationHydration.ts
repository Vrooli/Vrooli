import { useEffect } from "react";
import { getConversationSession } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";

const HYDRATION_RETRY_DELAYS_MS = [1000, 3000, 10 * 1000];

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
    const timers: ReturnType<typeof setTimeout>[] = [];
    const ids = key ? key.split("|") : [];
    const hydrate = (id: string, attempt: number) => {
      if (!id) return;
      if (useConversationStore.getState().sessions[id]?.hydrated) return;
      void (async () => {
        try {
          const data = await getConversationSession(id);
          if (!cancelled) hydrateSession(id, data.events, data.cursor);
        } catch (error) {
          if (cancelled) return;
          console.warn("[web-console] conversation hydration failed", { sessionId: id, error });
          const delay = HYDRATION_RETRY_DELAYS_MS[Math.min(attempt, HYDRATION_RETRY_DELAYS_MS.length - 1)];
          timers.push(setTimeout(() => hydrate(id, attempt + 1), delay));
        }
      })();
    };
    for (const id of ids) {
      hydrate(id, 0);
    }
    return () => {
      cancelled = true;
      for (const timer of timers) clearTimeout(timer);
    };
  }, [key, hydrateSession]);
}
