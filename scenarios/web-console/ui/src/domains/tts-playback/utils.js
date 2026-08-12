export function playbackEventKey(sessionId, eventId) {
    return `${sessionId}:${eventId}`;
}
export function hasOriginalVersion(event) {
    return Boolean(event?.summarized && (event.originalSpeechParagraphs?.length ?? 0) > 0);
}
export function defaultPlaybackVersion(event) {
    return hasOriginalVersion(event) ? "active" : "active";
}
export function resolvePlaybackVersion(selectedVersions, preferredVersion, sessionId, event) {
    const selected = selectedVersions[playbackEventKey(sessionId, event.id)];
    if (selected === "original" && hasOriginalVersion(event))
        return "original";
    if (selected === "active")
        return "active";
    if (preferredVersion === "original" && hasOriginalVersion(event))
        return "original";
    return selected ?? preferredVersion ?? defaultPlaybackVersion(event);
}
export function resolvePlaybackParagraphs(event, version) {
    if (version === "original" && hasOriginalVersion(event)) {
        return event.originalSpeechParagraphs ?? event.speechParagraphs;
    }
    return event.speechParagraphs?.length ? event.speechParagraphs : [event.text];
}
export function buildPlaybackQueue(sessionId, events, selectedVersions, preferredVersion) {
    return events.map((event) => ({
        eventId: event.id,
        sequence: event.sequence,
        version: resolvePlaybackVersion(selectedVersions, preferredVersion, sessionId, event),
    }));
}
export function findConversationEvent(sessions, target) {
    if (!target)
        return null;
    const session = sessions[target.sessionId];
    return session?.events.find((event) => event.id === target.eventId) ?? null;
}
export function buildQueueLabel(state, event) {
    if (!event)
        return null;
    if (state.queueEntries.length > 1 && state.queueIndex >= 0 && state.queueIndex < state.queueEntries.length) {
        return `${state.queueIndex + 1}/${state.queueEntries.length}`;
    }
    return `#${event.sequence}`;
}
export function buildPlaybackContext(sessions, state, currentTarget, intent) {
    const event = findConversationEvent(sessions, currentTarget);
    if (!currentTarget || !event)
        return null;
    const version = resolvePlaybackVersion(state.selectedVersions, state.preferredVersion, currentTarget.sessionId, event);
    return {
        event,
        sessionId: currentTarget.sessionId,
        version,
        queueLabel: buildQueueLabel(state, event),
        hasQueuedNext: state.queueEntries.length > 0 && state.queueIndex < state.queueEntries.length - 1,
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
export function shouldQueueIncomingEvent(args) {
    return args.autoTtsEnabled
        && args.playbackIntent === "continuous"
        && args.activePaneId === args.sessionId
        && args.event.role === "assistant";
}
export function shouldAutoPlayIncomingEvent(args) {
    return shouldQueueIncomingEvent(args) && !args.isSpeaking;
}
export function shouldShowPlaybackBar(args) {
    return Boolean((args.autoTtsEnabled || args.isSpeaking)
        && args.context?.event
        && args.context.sessionId
        && args.activePaneId === args.context.sessionId);
}
export function nextIntentAfterUserPlay() {
    return "continuous";
}
export function nextIntentAfterUserPause() {
    return "paused";
}
export function nextIntentAfterUserStop() {
    return "stopped";
}
export function nextIntentAfterNaturalCompletion(intent) {
    return intent === "continuous" ? "continuous" : intent;
}
