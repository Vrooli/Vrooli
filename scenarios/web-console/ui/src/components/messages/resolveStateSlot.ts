import type { ActivitySource, PendingPromptView, SessionActivityView } from "../../api/sessionActivity";

// DOC: docs/internal/MESSAGES-VIEW-PROJECTION-UX.md#the-state-slot

/** Below this the server's reading is a guess, and a guess is not shown. */
export const STATE_SLOT_CONFIDENCE_FLOOR = 0.6;

interface WaitingSlotBase {
  harness: string;
  source: ActivitySource;
  since: string;
}

/** The one presentation Messages shows under its last row. */
export type StateSlot =
  | { kind: "working"; elapsedFrom: string; lastOutputAt?: string }
  | ({ kind: "waiting-detected" } & WaitingSlotBase)
  | ({ kind: "waiting-rendered"; prompt: PendingPromptView } & WaitingSlotBase)
  | ({ kind: "waiting-answerable"; prompt: PendingPromptView } & WaitingSlotBase);

/**
 * Maps a session's activity to the slot: a working strip, a waiting card at
 * the level the prompt could be read, or nothing (idle, unknown, a reading
 * below the confidence floor, or no activity).
 */
export function resolveStateSlot(activity: SessionActivityView | undefined): StateSlot | null {
  if (!activity || activity.confidence < STATE_SLOT_CONFIDENCE_FLOOR) return null;
  if (activity.state === "working") {
    return {
      kind: "working",
      elapsedFrom: activity.since,
      ...(activity.lastOutputAt ? { lastOutputAt: activity.lastOutputAt } : {}),
    };
  }
  if (activity.state !== "waiting") return null;
  const base: WaitingSlotBase = { harness: activity.harness, source: activity.source, since: activity.since };
  const prompt = activity.prompt;
  if (!prompt || prompt.text.trim() === "") return { kind: "waiting-detected", ...base };
  if (prompt.answerable) return { kind: "waiting-answerable", ...base, prompt };
  return { kind: "waiting-rendered", ...base, prompt };
}
