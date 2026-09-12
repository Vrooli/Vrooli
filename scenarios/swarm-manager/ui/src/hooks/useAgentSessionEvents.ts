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

  // Only these three fields decide what this hook does. Depending on the whole
  // session object meant a new message — or, before the store learned to skip
  // unchanged polls, any poll at all — tore down the 2.5s events interval and
  // immediately refetched, so events were pulled about twice as often as
  // intended and never on the cadence the interval describes.
  const sessionID = session?.id;
  const sessionStatus = session?.status;
  const sessionRunID = session?.runId;
  const eventsActive = sessionStatus !== undefined && ACTIVE_EVENT_STATUSES.has(sessionStatus);

  const loadEvents = useCallback(async () => {
    if (!sessionID || sessionStatus === "draft" || !sessionRunID) {
      // Keep the existing empty array rather than allocating a fresh one: this
      // branch runs on every effect pass for a session with no run, and a new
      // array identity re-renders the whole inspector pane to display the same
      // nothing.
      setEvents((current) => (current.length === 0 ? current : []));
      setError(null);
      afterSequenceRef.current = 0n;
      return;
    }
    setIsLoading(true);
    try {
      const result = await listSessionEvents({
        sessionId: sessionID,
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
  }, [listSessionEvents, sessionID, sessionStatus, sessionRunID]);

  useEffect(() => {
    setEvents([]);
    setError(null);
    afterSequenceRef.current = 0n;
  }, [sessionID, sessionRunID]);

  useEffect(() => {
    void loadEvents();
    if (!eventsActive) return undefined;
    const interval = window.setInterval(() => {
      void loadEvents();
    }, 2500);
    return () => window.clearInterval(interval);
  }, [loadEvents, eventsActive]);

  return { events, isLoading, error, refreshEvents: loadEvents };
}

function mergeEvents(current: AgentSessionRunEvent[], incoming: AgentSessionRunEvent[]): AgentSessionRunEvent[] {
  // A poll that returns nothing new is the common case for a settled run.
  // Returning `current` unchanged keeps the array identity stable so the
  // inspector pane can skip re-rendering.
  if (incoming.length === 0) return current;
  const byKey = new Map<string, AgentSessionRunEvent>();
  for (const event of current) {
    byKey.set(event.id || event.sequence.toString(), event);
  }
  for (const event of incoming) {
    byKey.set(event.id || event.sequence.toString(), event);
  }
  return Array.from(byKey.values()).sort((a, b) => Number(a.sequence - b.sequence));
}
