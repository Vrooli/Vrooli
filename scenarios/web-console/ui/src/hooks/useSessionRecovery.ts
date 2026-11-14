import { useEffect, useRef, useState } from "react";
import { listSessionsWithRecovery } from "../api/sessions";

/**
 * Live view of startup persistent-session recovery. Recovery runs
 * asynchronously on the API (so the server is reachable immediately); this hook
 * polls the session list's recovery snapshot while recovery is in progress so
 * the UI can honestly show "sessions still recovering" instead of implying an
 * empty workspace. Polling stops as soon as recovery finishes.
 */
export interface SessionRecoveryState {
  inProgress: boolean;
  total: number;
  recovered: number;
  awaitingRecovery: number;
  adopted: number;
  /**
   * True once recovery has gone from in-progress to done during this mount AND
   * at least one session was recovered/adopted — i.e. the list the user is
   * looking at is now stale and worth refreshing.
   */
  justCompleted: boolean;
}

const POLL_MS = 1500;

const INITIAL: SessionRecoveryState = {
  inProgress: false,
  total: 0,
  recovered: 0,
  awaitingRecovery: 0,
  adopted: 0,
  justCompleted: false,
};

export function useSessionRecovery(): SessionRecoveryState {
  const [state, setState] = useState<SessionRecoveryState>(INITIAL);
  const sawInProgress = useRef(false);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | null = null;

    const poll = async () => {
      try {
        const { recovery } = await listSessionsWithRecovery();
        if (cancelled) return;
        if (recovery.in_progress) sawInProgress.current = true;
        const justCompleted =
          sawInProgress.current &&
          !recovery.in_progress &&
          recovery.recovered + recovery.adopted > 0;
        setState({
          inProgress: recovery.in_progress,
          total: recovery.total,
          recovered: recovery.recovered,
          awaitingRecovery: recovery.awaiting_recovery,
          adopted: recovery.adopted,
          justCompleted,
        });
        if (recovery.in_progress && !cancelled) {
          timer = setTimeout(() => void poll(), POLL_MS);
        }
      } catch {
        // Best-effort — a transient list failure must not wedge the banner.
        // Keep polling only while we still believe recovery is running.
        if (!cancelled && sawInProgress.current) {
          timer = setTimeout(() => void poll(), POLL_MS);
        }
      }
    };

    void poll();
    return () => {
      cancelled = true;
      if (timer) clearTimeout(timer);
    };
  }, []);

  return state;
}
