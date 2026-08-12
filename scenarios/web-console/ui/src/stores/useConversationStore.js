import { create } from "zustand";
const defaultCursor = () => ({
    lastSeenSequence: 0,
    lastListenedSequence: 0,
});
// Shared frozen empty array so selectors for sessions with no events return a
// referentially-stable value. Returning a fresh `[]` each call would defeat
// Zustand's Object.is short-circuit and re-render the subscriber on every
// unrelated store update.
const EMPTY_EVENTS = Object.freeze([]);
function sessionMetadata(events, cursor) {
    const knownEventIds = new Set(events.map((event) => event.id));
    let maxSequence = 0;
    let previous = 0;
    let hasSequenceGap = false;
    let refetchSinceSequence = 0;
    let unreadCount = 0;
    for (const event of events) {
        maxSequence = Math.max(maxSequence, event.sequence);
        if (previous > 0 && event.sequence !== previous + 1) {
            hasSequenceGap = true;
            if (refetchSinceSequence === 0)
                refetchSinceSequence = previous;
        }
        previous = event.sequence;
        if (event.role === "assistant" && event.sequence > cursor.lastSeenSequence)
            unreadCount += 1;
    }
    if ((events[0]?.sequence ?? 1) > 1)
        refetchSinceSequence = 0;
    return { knownEventIds, unreadCount, maxSequence, hasSequenceGap: hasSequenceGap || (events[0]?.sequence ?? 1) > 1, refetchSinceSequence };
}
export const useConversationStore = create((set) => ({
    sessions: {},
    viewModes: {},
    hydrateSession: (sessionId, events, cursor, page) => set((state) => {
        // Merge with anything appendEvent already added while the GET was in
        // flight — naively replacing would drop live WS events that arrived
        // after the request but before the response, leaving a permanent
        // sequence gap (refresh uses max seq as since_sequence and never
        // backfills lower events). See useConversationSession mount effect.
        const existing = state.sessions[sessionId];
        if (!existing || existing.events.length === 0) {
            return {
                sessions: {
                    ...state.sessions,
                    [sessionId]: { events, cursor, hydrated: true, ...sessionMetadata(events, cursor), windowOldestSequence: page?.oldestSequence, hasOlder: page?.hasOlder, totalCount: page?.totalCount },
                },
            };
        }
        const seen = new Set(events.map((e) => e.id));
        const extras = existing.events.filter((e) => !seen.has(e.id));
        const merged = extras.length > 0
            ? [...events, ...extras].sort((a, b) => a.sequence - b.sequence)
            : events;
        return {
            sessions: {
                ...state.sessions,
                [sessionId]: { events: merged, cursor, hydrated: true, ...sessionMetadata(merged, cursor), windowOldestSequence: page?.oldestSequence, hasOlder: page?.hasOlder, totalCount: page?.totalCount },
            },
        };
    }),
    setSessionWindow: (sessionId, events, cursor, page) => set((state) => ({
        sessions: {
            ...state.sessions,
            [sessionId]: {
                events,
                cursor,
                hydrated: true,
                ...sessionMetadata(events, cursor),
                windowOldestSequence: page.oldestSequence,
                hasOlder: page.hasOlder,
                totalCount: page.totalCount,
            },
        },
    })),
    prependEvents: (sessionId, incoming, page) => set((state) => {
        const existing = state.sessions[sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
        const known = existing.knownEventIds ?? new Set(existing.events.map((event) => event.id));
        const added = incoming.filter((event) => !known.has(event.id));
        const events = added.length > 0 ? [...added, ...existing.events].sort((a, b) => a.sequence - b.sequence) : existing.events;
        return { sessions: { ...state.sessions, [sessionId]: { ...existing, events, ...sessionMetadata(events, existing.cursor), windowOldestSequence: page.oldestSequence, hasOlder: page.hasOlder, totalCount: page.totalCount } } };
    }),
    appendEvent: (event) => set((state) => {
        const existing = state.sessions[event.sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
        const knownEventIds = existing.knownEventIds ?? new Set(existing.events.map((candidate) => candidate.id));
        if (knownEventIds.has(event.id)) {
            return state;
        }
        const nextKnownEventIds = new Set(knownEventIds).add(event.id);
        const hasSequenceGap = (existing.hasSequenceGap ?? false) || ((existing.maxSequence ?? 0) > 0 && event.sequence !== (existing.maxSequence ?? 0) + 1);
        const unreadCount = (existing.unreadCount ?? getSessionUnreadCount(state, event.sessionId)) + (event.role === "assistant" && event.sequence > existing.cursor.lastSeenSequence ? 1 : 0);
        return {
            sessions: {
                ...state.sessions,
                [event.sessionId]: {
                    ...existing,
                    events: [...existing.events, event],
                    knownEventIds: nextKnownEventIds,
                    unreadCount,
                    maxSequence: Math.max(existing.maxSequence ?? 0, event.sequence),
                    hasSequenceGap,
                },
            },
        };
    }),
    mergeEvents: (sessionId, incoming, cursor) => set((state) => {
        const existing = state.sessions[sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
        if (incoming.length === 0 && !cursor) {
            // Still mark hydrated so UI stops showing "not loaded" states.
            if (existing.hydrated)
                return state;
            return {
                sessions: {
                    ...state.sessions,
                    [sessionId]: { ...existing, hydrated: true },
                },
            };
        }
        const seen = existing.knownEventIds ?? new Set(existing.events.map((e) => e.id));
        const added = incoming.filter((e) => !seen.has(e.id));
        const merged = added.length > 0
            ? [...existing.events, ...added].sort((a, b) => a.sequence - b.sequence)
            : existing.events;
        return {
            sessions: {
                ...state.sessions,
                [sessionId]: {
                    events: merged,
                    cursor: cursor ?? existing.cursor,
                    hydrated: true,
                    ...sessionMetadata(merged, cursor ?? existing.cursor),
                },
            },
        };
    }),
    updateEvent: (sessionId, eventId, patch) => set((state) => {
        const session = state.sessions[sessionId];
        if (!session)
            return state;
        const idx = session.events.findIndex((e) => e.id === eventId);
        if (idx === -1)
            return state;
        const updatedEvents = [...session.events];
        const existing = updatedEvents[idx];
        const ev = { ...existing };
        if (patch.speechParagraphs != null)
            ev.speechParagraphs = patch.speechParagraphs;
        if (patch.originalSpeechParagraphs != null)
            ev.originalSpeechParagraphs = patch.originalSpeechParagraphs;
        if (patch.summarized != null)
            ev.summarized = patch.summarized;
        updatedEvents[idx] = ev;
        return {
            sessions: {
                ...state.sessions,
                [sessionId]: { ...session, events: updatedEvents },
            },
        };
    }),
    updateCursor: (sessionId, cursor) => set((state) => {
        const existing = state.sessions[sessionId] ?? { events: [], cursor: defaultCursor(), hydrated: true };
        return {
            sessions: {
                ...state.sessions,
                [sessionId]: {
                    ...existing,
                    cursor: {
                        lastSeenSequence: cursor.lastSeenSequence ?? existing.cursor.lastSeenSequence,
                        lastListenedSequence: cursor.lastListenedSequence ?? existing.cursor.lastListenedSequence,
                    },
                    ...sessionMetadata(existing.events, {
                        lastSeenSequence: cursor.lastSeenSequence ?? existing.cursor.lastSeenSequence,
                        lastListenedSequence: cursor.lastListenedSequence ?? existing.cursor.lastListenedSequence,
                    }),
                },
            },
        };
    }),
    setViewMode: (sessionId, mode) => set((state) => ({
        viewModes: {
            ...state.viewModes,
            [sessionId]: mode,
        },
    })),
    clearSession: (sessionId) => set((state) => {
        const sessions = { ...state.sessions };
        const viewModes = { ...state.viewModes };
        delete sessions[sessionId];
        delete viewModes[sessionId];
        return { sessions, viewModes };
    }),
}));
export function getSessionConversationEvents(state, sessionId) {
    return state.sessions[sessionId]?.events ?? EMPTY_EVENTS;
}
export function getSessionConversationCursor(state, sessionId) {
    return state.sessions[sessionId]?.cursor ?? defaultCursor();
}
export function getSessionUnreadCount(state, sessionId) {
    const session = state.sessions[sessionId];
    if (!session)
        return 0;
    return session.unreadCount ?? session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
}
export function getSessionUnlistenedEvents(state, sessionId) {
    const session = state.sessions[sessionId];
    if (!session)
        return [];
    return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastListenedSequence);
}
export function getSessionViewMode(state, sessionId) {
    return state.viewModes[sessionId] ?? "terminal";
}
/**
 * Returns the `since_sequence` value to use for a refetch that will both pull
 * any new tail events AND backfill any gap in the local sequence. Strategy:
 *   - empty store → 0 (full hydrate)
 *   - first event's sequence > 1 → 0 (we may be missing the prefix)
 *   - first internal gap at events[i] → events[i-1].sequence (refetch from before the gap)
 *   - no gap → max sequence (current tail-only behaviour)
 */
export function getSessionRefetchSinceSequence(state, sessionId) {
    const session = state.sessions[sessionId];
    const events = session?.events;
    if (!events || events.length === 0)
        return 0;
    if (session?.maxSequence != null && session.hasSequenceGap != null) {
        return session.hasSequenceGap ? (session.refetchSinceSequence ?? 0) : session.maxSequence;
    }
    const sorted = [...events].sort((a, b) => a.sequence - b.sequence);
    const first = sorted[0];
    if (!first || first.sequence > 1)
        return 0;
    for (let i = 1; i < sorted.length; i++) {
        const prev = sorted[i - 1];
        const cur = sorted[i];
        if (!prev || !cur)
            continue;
        if (cur.sequence !== prev.sequence + 1) {
            return prev.sequence;
        }
    }
    const last = sorted[sorted.length - 1];
    return last ? last.sequence : 0;
}
