import { useCallback, useEffect, useRef, useState } from "react";
import { useAgentSessionStore } from "../stores";
import type { AgentSession, AgentSessionRunEvent } from "../types";

const ACTIVE_EVENT_STATUSES = new Set<AgentSession["status"]>(["starting", "running"]);

export function useAgentSessionEvents(session: AgentSession | undefined) {
  const listSessionEvents = useAgentSessionStore((s) => s.listSessionEvents);
  const [events, setEvents] = useState<AgentSessionRunEvent[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const afterSequenceRef = useRef<bigint>(0n);

  const loadEvents = useCallback(async () => {
    if (!session || session.status === "draft" || !session.runId) {
      setEvents([]);
      setError(null);
      afterSequenceRef.current = 0n;
      return;
    }
    setIsLoading(true);
    try {
      const result = await listSessionEvents({
        sessionId: session.id,
        afterSequence: afterSequenceRef.current > 0n ? afterSequenceRef.current : undefined,
        limit: 100,
      });
      setEvents((current) => mergeEvents(current, result.events));
      afterSequenceRef.current = result.nextAfterSequence > afterSequenceRef.current
        ? result.nextAfterSequence
        : afterSequenceRef.current;
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load session events.");
    } finally {
      setIsLoading(false);
    }
  }, [listSessionEvents, session]);

  useEffect(() => {
    setEvents([]);
    setError(null);
    afterSequenceRef.current = 0n;
  }, [session?.id, session?.runId]);

  useEffect(() => {
    void loadEvents();
    if (!session || !ACTIVE_EVENT_STATUSES.has(session.status)) return undefined;
    const interval = window.setInterval(() => {
      void loadEvents();
    }, 2500);
    return () => window.clearInterval(interval);
  }, [loadEvents, session]);

  return { events, isLoading, error, refreshEvents: loadEvents };
}

function mergeEvents(current: AgentSessionRunEvent[], incoming: AgentSessionRunEvent[]): AgentSessionRunEvent[] {
  const byKey = new Map<string, AgentSessionRunEvent>();
  for (const event of current) {
    byKey.set(event.id || event.sequence.toString(), event);
  }
  for (const event of incoming) {
    byKey.set(event.id || event.sequence.toString(), event);
  }
  return Array.from(byKey.values()).sort((a, b) => Number(a.sequence - b.sequence));
}
