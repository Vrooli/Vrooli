import type { ConversationEvent } from "../../api/conversation";
import type { PlaybackEventContext, PlaybackIntent, PlaybackQueueEntry, PlaybackTarget, PlaybackVersion, SessionPlaybackControllerState } from "./types";

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

export function buildPlaybackContext(
  sessions: Record<string, { events: ConversationEvent[] } | undefined>,
  state: SessionPlaybackControllerState,
  currentTarget: PlaybackTarget | null,
  intent: PlaybackIntent,
): PlaybackEventContext | null {
  const event = findConversationEvent(sessions, currentTarget);
  if (!currentTarget || !event) return null;
  const version = resolvePlaybackVersion(state.selectedVersions, state.preferredVersion, currentTarget.sessionId, event);
  return {
    event,
    sessionId: currentTarget.sessionId,
    version,
    queueIndex: state.queueIndex,
    queueLength: state.queueEntries.length,
    intent,
  };
}

/**
 * Whether an incoming assistant event is eligible for auto-play at all —
 * independent of whether playback is currently busy. When the transport is
 * idle this means "play now"; when it is busy this means "enqueue and play
 * when the current one ends" (Phase 7: mid-playback messages are queued, not
 * dropped). The `!isSpeaking` decision belongs only to the play-now variant.
 */
export function shouldQueueIncomingEvent(args: {
  autoTtsEnabled: boolean;
  playbackIntent: PlaybackIntent;
  activePaneId: string | null;
  sessionId: string;
  event: ConversationEvent;
}): boolean {
  return args.autoTtsEnabled
    && args.playbackIntent === "continuous"
    && args.activePaneId === args.sessionId
    && args.event.role === "assistant";
}

export function shouldAutoPlayIncomingEvent(args: {
  autoTtsEnabled: boolean;
  playbackIntent: PlaybackIntent;
  activePaneId: string | null;
  sessionId: string;
  event: ConversationEvent;
  isSpeaking: boolean;
}): boolean {
  return shouldQueueIncomingEvent(args) && !args.isSpeaking;
}

export function shouldShowPlayback(args: {
  autoTtsEnabled: boolean;
  activePaneId: string | null;
  context: PlaybackEventContext | null;
  isSpeaking: boolean;
  /** Playback the user started is loading or playing, before audio is heard. */
  isActive: boolean;
}): boolean {
  return Boolean(
    (args.autoTtsEnabled || args.isSpeaking || args.isActive)
    && args.context?.event
    && args.context.sessionId
    && args.activePaneId === args.context.sessionId,
  );
}

export function nextIntentAfterUserPlay(): PlaybackIntent {
  return "continuous";
}

export function nextIntentAfterUserPause(): PlaybackIntent {
  return "paused";
}

export function nextIntentAfterUserStop(): PlaybackIntent {
  return "stopped";
}

export function nextIntentAfterNaturalCompletion(intent: PlaybackIntent): PlaybackIntent {
  return intent === "continuous" ? "continuous" : intent;
}
