import type { ConflictResolutionFormalReplayFixtures } from "./generated/replay.helper";

export const conflictResolutionFormalFixtures = {
  stateFor: {
    idle: () => ({ status: "idle" as const }),
    ready: () => ({ status: "ready" as const }),
  },
  eventFor: {
    start: () => ({ type: "start" as const }),
    reset: () => ({ type: "reset" as const }),
  },
} satisfies ConflictResolutionFormalReplayFixtures;
