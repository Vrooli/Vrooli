import type { ConversationEvent } from "../../api/conversation";
import type { PlaybackEventContext, PlaybackQueueEntry, PlaybackTarget, PlaybackVersion, SessionPlaybackControllerState } from "./types";

export function playbackEventKey(sessionId: string, eventId: string): string {
  return `${sessionId}:${eventId}`;
}

export function hasOriginalVersion(event: ConversationEvent | null | undefined): boolean {
  return Boolean(event?.summarized && (event.originalSpeechParagraphs?.length ?? 0) > 0);
}

export function defaultPlaybackVersion(event: ConversationEvent): PlaybackVersion {
  return hasOriginalVersion(event) ? "active" : "active";
}

export function resolvePlaybackVersion(
  selectedVersions: Record<string, PlaybackVersion>,
  preferredVersion: PlaybackVersion,
  sessionId: string,
  event: ConversationEvent,
): PlaybackVersion {
  const selected = selectedVersions[playbackEventKey(sessionId, event.id)];
  if (selected === "original" && hasOriginalVersion(event)) return "original";
  if (selected === "active") return "active";
  if (preferredVersion === "original" && hasOriginalVersion(event)) return "original";
  return selected ?? preferredVersion ?? defaultPlaybackVersion(event);
}

export function resolvePlaybackParagraphs(
  event: ConversationEvent,
  version: PlaybackVersion,
): string[] {
  if (version === "original" && hasOriginalVersion(event)) {
    return event.originalSpeechParagraphs ?? event.speechParagraphs;
  }
  return event.speechParagraphs?.length ? event.speechParagraphs : [event.text];
}

export function buildPlaybackQueue(
  sessionId: string,
  events: ConversationEvent[],
  selectedVersions: Record<string, PlaybackVersion>,
  preferredVersion: PlaybackVersion,
): PlaybackQueueEntry[] {
  return events.map((event) => ({
    eventId: event.id,
    sequence: event.sequence,
    version: resolvePlaybackVersion(selectedVersions, preferredVersion, sessionId, event),
  }));
}

export function findConversationEvent(
  sessions: Record<string, { events: ConversationEvent[] } | undefined>,
  target: PlaybackTarget | null,
): ConversationEvent | null {
  if (!target) return null;
  const session = sessions[target.sessionId];
  return session?.events.find((event) => event.id === target.eventId) ?? null;
}

export function buildQueueLabel(state: SessionPlaybackControllerState, event: ConversationEvent | null): string | null {
  if (!event) return null;
  if (state.queueEntries.length > 1 && state.queueIndex >= 0 && state.queueIndex < state.queueEntries.length) {
    return `${state.queueIndex + 1}/${state.queueEntries.length}`;
  }
  return `#${event.sequence}`;
}

export function buildPlaybackContext(
  sessions: Record<string, { events: ConversationEvent[] } | undefined>,
  state: SessionPlaybackControllerState,
  currentTarget: PlaybackTarget | null,
): PlaybackEventContext | null {
  const event = findConversationEvent(sessions, currentTarget);
  if (!currentTarget || !event) return null;
  const version = resolvePlaybackVersion(state.selectedVersions, state.preferredVersion, currentTarget.sessionId, event);
  return {
    event,
    sessionId: currentTarget.sessionId,
    version,
    queueLabel: buildQueueLabel(state, event),
    hasQueuedNext: state.queueEntries.length > 0 && state.queueIndex < state.queueEntries.length - 1,
  };
}
