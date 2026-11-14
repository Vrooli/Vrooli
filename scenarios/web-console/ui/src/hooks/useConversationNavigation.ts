import { useEffect, useMemo, useState } from "react";
import { useShallow } from "zustand/react/shallow";
import { useConversationStore } from "../stores/useConversationStore";
import type { ConversationEvent } from "../api/conversation";
import type { BuildWorkspaceNavigationItemsOptions } from "../lib/workspaceNavigation";

/** Subscribe to sidebar data, not loading state, old pages or speech updates. */
export function useConversationNavigation(sessionIds: string[]) {
  const values = useConversationStore(useShallow(state => sessionIds.flatMap(id => {
    const session = state.sessions[id];
    const latest = session?.events[session.events.length - 1];
    const unread = session?.unreadCount ?? session?.events.filter(event =>
      event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length ?? 0;
    return [latest?.id, latest?.text, latest?.createdAt, latest?.sequence, unread];
  })));
  const key = sessionIds.join("|");
  return useMemo(() => {
    const sessions: NonNullable<BuildWorkspaceNavigationItemsOptions["conversationSessions"]> = {};
    const ids = key ? key.split("|") : [];
    ids.forEach((id, index) => {
      const [eventId, text, createdAt, sequence, unread] = values.slice(index * 5, index * 5 + 5);
      sessions[id] = {
        events: eventId ? [{ id: eventId, text, createdAt, sequence } as ConversationEvent] : [],
        cursor: { lastSeenSequence: 0, lastListenedSequence: 0 },
        unreadCount: Number(unread),
      };
    });
    return sessions;
  }, [key, values]);
}

/** Relative activity labels must refresh even when there is no live traffic. */
export function useActivityTime() {
  const [minute, setMinute] = useState(() => new Date());
  useEffect(() => {
    const timer = setInterval(() => { setMinute(new Date()); }, 60_000);
    return () => { clearInterval(timer); };
  }, []);
  return minute;
}
