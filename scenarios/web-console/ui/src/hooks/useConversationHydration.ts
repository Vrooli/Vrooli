import { useEffect, useRef } from "react";
import { useConversationStore } from "../stores/useConversationStore";
import { refreshConversationSession } from "./useConversationSession";

const BACKGROUND_CONCURRENCY = 2;
const HYDRATION_RETRY_DELAYS_MS = [1000, 3000, 10 * 1000];

/**
 * Seed unread badges for every session, after prioritizing the visible session.
 * Background history is still windowed and eventually loaded for correct badges,
 * but two workers replace the startup request burst. Foreground pane loads share
 * refresh ownership with these workers, including loading/error/paging metadata.
 */
export function useConversationHydration(sessionIds: string[], activeSessionId: string | null = null): void {
  const key = sessionIds.join("|");
  const activeRef = useRef(activeSessionId);
  activeRef.current = activeSessionId;
  const running = useRef(0);
  const pumpRef = useRef<() => void>(() => {});

  useEffect(() => {
    let cancelled = false;
    const timers = new Set<ReturnType<typeof setTimeout>>();
    const pending = (key ? key.split("|") : []).map(id => ({ id, attempt: 0 }));
    const schedule = (callback: () => void, delay: number) => {
      const timer = setTimeout(() => { timers.delete(timer); callback(); }, delay);
      timers.add(timer);
    };
    const pump = () => {
      if (cancelled) return;
      while (running.current < BACKGROUND_CONCURRENCY && pending.length) {
        const priority = pending.findIndex(item => item.id === activeRef.current);
        const item = pending.splice(priority < 0 ? 0 : priority, 1)[0];
        if (!item) break;
        if (useConversationStore.getState().sessions[item.id]?.hydrated) continue;
        running.current++;
        void refreshConversationSession(item.id).then(result => {
          if (!cancelled && !result.ok && result.error.retryable) {
            const delay = HYDRATION_RETRY_DELAYS_MS[Math.min(item.attempt, HYDRATION_RETRY_DELAYS_MS.length - 1)] ?? 10_000;
            schedule(() => {
              pending.push({ id: item.id, attempt: item.attempt + 1 });
              pump();
            }, delay);
          }
        }).finally(() => {
          running.current--;
          // A changed session list may own the queue now. Yield between pages
          // so decoding/rendering can finish before another background batch.
          if (!cancelled) schedule(() => { pumpRef.current(); }, 50);
          else pumpRef.current();
        });
      }
    };
    pumpRef.current = pump;
    const active = activeRef.current;
    const foreground = active && !useConversationStore.getState().sessions[active]?.hydrated && pending.some(item => item.id === active)
      ? refreshConversationSession(active)
      : Promise.resolve();
    void foreground.finally(() => { if (!cancelled) schedule(pump, 50); });
    return () => {
      cancelled = true;
      for (const timer of timers) clearTimeout(timer);
    };
  }, [key]);

  // Switching to a queued session never waits behind background requests.
  useEffect(() => {
    if (activeSessionId && !useConversationStore.getState().sessions[activeSessionId]?.hydrated) {
      void refreshConversationSession(activeSessionId);
    }
  }, [activeSessionId]);
}
