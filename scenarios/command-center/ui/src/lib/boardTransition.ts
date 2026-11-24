export type BoardTransition = "crossfade" | "cut";

/** Keep navigation timing and the room-motion token in step with saved board settings. */
export function boardTransitionDuration(transition: BoardTransition): number {
  return transition === "cut" ? 0 : 900;
}
