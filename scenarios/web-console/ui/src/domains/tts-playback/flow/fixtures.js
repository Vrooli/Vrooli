const sessionId = "session-1";
const eventId = "event-1";
const loadId = "load-1";
const queue = [eventId];
const queueIndex = 0;
const message = "playback failed";
export const playbackTransportFormalFixtures = {
    stateFor: {
        idle: () => ({ status: "idle" }),
        loading: () => ({
            status: "loading",
            sessionId,
            eventId,
            loadId,
            queue,
            queueIndex,
        }),
        playing: () => ({
            status: "playing",
            sessionId,
            eventId,
            loadId,
            queue,
            queueIndex,
        }),
        paused: () => ({
            status: "paused",
            sessionId,
            eventId,
            loadId,
            queue,
            queueIndex,
        }),
        error: () => ({
            status: "error",
            sessionId,
            eventId,
            message,
            queue,
            queueIndex,
        }),
    },
    eventFor: {
        play: () => ({
            type: "play",
            sessionId,
            eventId,
            loadId,
            queue,
            queueIndex,
        }),
        loadResolved: () => ({ type: "loadResolved", loadId }),
        loadFailed: () => ({ type: "loadFailed", loadId, message }),
        trackStarted: () => ({ type: "trackStarted", eventId }),
        trackEnded: () => ({ type: "trackEnded", eventId }),
        pause: () => ({ type: "pause" }),
        resume: () => ({ type: "resume" }),
        stop: () => ({ type: "stop" }),
        playbackError: () => ({ type: "playbackError", message }),
    },
};
