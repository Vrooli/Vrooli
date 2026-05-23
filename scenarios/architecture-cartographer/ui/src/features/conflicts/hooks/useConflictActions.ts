import * as React from "react";

import {
  legalEventsFor,
  statusToState,
  type ConflictEvent,
  type ConflictState,
} from "../flow/transition";
import type { Conflict } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

/**
 * Read the legal flow events for the given conflict. UI buttons must call
 * this rather than rendering every event — the flow's contract says
 * disallowed transitions are not surfaced.
 */
export function useConflictActions(conflict: Conflict | undefined): {
  state: ConflictState;
  legalEvents: readonly ConflictEvent[];
} {
  return React.useMemo(() => {
    const state = conflict ? statusToState(conflict.status) : "detected";
    return { state, legalEvents: legalEventsFor(state) };
  }, [conflict]);
}
