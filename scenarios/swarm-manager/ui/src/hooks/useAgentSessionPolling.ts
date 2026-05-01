import { useCallback } from "react";
import {
  isActiveAgentSession,
  useAgentSessionStore,
} from "../stores/agent-session-store";
import { useStorePolling } from "./useStorePolling";

const SESSION_LIST_POLL_INTERVAL_MS = 4_000;
const ACTIVE_SESSION_POLL_INTERVAL_MS = 3_000;

export function useAgentSessionPolling(): void {
  const sessions = useAgentSessionStore((state) => state.sessions);
  const activeSession = useAgentSessionStore((state) => state.activeSession);
  const fetchSessions = useAgentSessionStore((state) => state.fetchSessions);
  const refreshSession = useAgentSessionStore((state) => state.refreshSession);

  const hasActiveSessions = sessions.some(isActiveAgentSession);
  const shouldPollActiveSession = activeSession ? isActiveAgentSession(activeSession) : false;

  const pollSessionList = useCallback(async () => {
    await fetchSessions({ activeOnly: true }, { force: true });
  }, [fetchSessions]);

  const pollActiveSession = useCallback(async () => {
    if (!activeSession) return;
    await refreshSession(activeSession.id);
  }, [activeSession, refreshSession]);

  useStorePolling({
    enabled: hasActiveSessions,
    intervalMs: SESSION_LIST_POLL_INTERVAL_MS,
    pollFn: pollSessionList,
  });

  useStorePolling({
    enabled: shouldPollActiveSession,
    intervalMs: ACTIVE_SESSION_POLL_INTERVAL_MS,
    pollFn: pollActiveSession,
  });
}
