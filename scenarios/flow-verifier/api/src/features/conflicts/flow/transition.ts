import type { ConflictResolutionState, ConflictResolutionEvent } from "./generated/runtime";
import { nextConflictResolutionStatus } from "./generated/runtime";

export const transitionConflictResolution = (state: ConflictResolutionState, event: ConflictResolutionEvent): ConflictResolutionState => {
  const next = nextConflictResolutionStatus(state.status, event.type);
  return { status: next } as ConflictResolutionState;
};
