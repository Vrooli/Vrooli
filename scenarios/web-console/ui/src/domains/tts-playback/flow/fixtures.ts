import type { PlaybackTransportFormalReplayFixtures } from "./generated/replay.helper";

const sessionId = "session-1";
const eventId = "event-1";
const loadId = "load-1";
const queue = [eventId] as const;
const queueIndex = 0;
const message = "playback failed";

export const playbackTransportFormalFixtures = {
  stateFor: {
    idle: () => ({ status: "idle" as const }),
    loading: () => ({
      status: "loading" as const,
      sessionId,
      eventId,
      loadId,
      queue,
      queueIndex,
    }),
    playing: () => ({
      status: "playing" as const,
      sessionId,
      eventId,
      loadId,
      queue,
      queueIndex,
    }),
    paused: () => ({
      status: "paused" as const,
      sessionId,
      eventId,
      loadId,
      queue,
      queueIndex,
    }),
    error: () => ({
      status: "error" as const,
      sessionId,
      eventId,
      message,
      queue,
      queueIndex,
    }),
  },
  eventFor: {
    play: () => ({
      type: "play" as const,
      sessionId,
      eventId,
      loadId,
      queue,
      queueIndex,
    }),
    loadResolved: () => ({ type: "loadResolved" as const, loadId }),
    loadFailed: () => ({ type: "loadFailed" as const, loadId, message }),
    trackStarted: () => ({ type: "trackStarted" as const, eventId }),
    trackEnded: () => ({ type: "trackEnded" as const, eventId }),
    pause: () => ({ type: "pause" as const }),
    resume: () => ({ type: "resume" as const }),
    stop: () => ({ type: "stop" as const }),
    playbackError: () => ({ type: "playbackError" as const, message }),
  },
} satisfies PlaybackTransportFormalReplayFixtures;
