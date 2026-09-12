import { useEffect } from "react";
import { getConversationSession, type ConversationSessionResponse } from "../api/conversation";
import { useConversationStore } from "../stores/useConversationStore";
import { describeLoadFailure, PAGE_SIZE } from "../lib/conversationLoad";

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
 * The request is windowed to PAGE_SIZE like every other conversation fetch.
 * It used to ask for the whole history of every open pane at once, which on a
 * busy workspace meant pulling thousands of events per session into memory to
 * render a single unread badge — and, because it recorded no paging metadata,
 * it also left "load older messages" with nothing to page from.
 *
 * Idempotent: skips any session already hydrated, and the store's hydrate/merge
 * dedupe by event id, so overlap with a pane's own load is harmless.
 */
export function useConversationHydration(sessionIds: string[]): void {
  const hydrateSession = useConversationStore((s) => s.hydrateSession);
  const failLoad = useConversationStore((s) => s.failLoad);
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
          const data: ConversationSessionResponse = await getConversationSession(id, { limit: PAGE_SIZE });
          if (!cancelled) {
            hydrateSession(id, data.events, data.cursor, {
              oldestSequence: data.oldestSequence ?? data.events[0]?.sequence ?? 0,
              hasOlder: data.hasMore ?? false,
              totalCount: data.totalCount ?? data.events.length,
              capture: data.capture,
            });
          }
        } catch (error) {
          if (cancelled) return;
          const described = describeLoadFailure(error);
          // Record the failure rather than only logging it. A background
          // hydration that fails silently leaves the pane rendering an empty
          // session it never actually loaded.
          failLoad(id, described);
          if (!described.retryable) return;
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
  }, [key, hydrateSession, failLoad]);
}
