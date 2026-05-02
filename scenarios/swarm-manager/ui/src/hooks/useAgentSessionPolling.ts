import { useCallback } from "react";
import {
  isActiveAgentSession,
  useAgentSessionStore,
} from "../stores/agent-session-store";
import { useStorePolling } from "./useStorePolling";

const SESSION_LIST_POLL_INTERVAL_MS = 4_000;
const ACTIVE_SESSION_POLL_INTERVAL_MS = 3_000;

export function useAgentSessionPolling(sessionId?: string): void {
  const sessions = useAgentSessionStore((state) => state.sessions);
  const fetchSessions = useAgentSessionStore((state) => state.fetchSessions);
  const refreshSession = useAgentSessionStore((state) => state.refreshSession);

  const hasActiveSessions = sessions.some(isActiveAgentSession);
  const detailSession = sessionId ? sessions.find((session) => session.id === sessionId) : undefined;
  const shouldPollDetail = !!sessionId && (!detailSession || isActiveAgentSession(detailSession));

  const pollSessionList = useCallback(async () => {
    await fetchSessions({ activeOnly: true }, { force: true });
  }, [fetchSessions]);

  const pollDetailSession = useCallback(async () => {
    if (!sessionId) return;
    try {
      await refreshSession(sessionId);
    } catch {
      // swallow — error surfaces via store.error
    }
  }, [sessionId, refreshSession]);

  useStorePolling({
    enabled: hasActiveSessions,
    intervalMs: SESSION_LIST_POLL_INTERVAL_MS,
    pollFn: pollSessionList,
  });

  useStorePolling({
    enabled: shouldPollDetail,
    intervalMs: ACTIVE_SESSION_POLL_INTERVAL_MS,
    pollFn: pollDetailSession,
  });
}
