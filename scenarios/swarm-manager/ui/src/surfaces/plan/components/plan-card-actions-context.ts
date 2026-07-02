/**
 * PlanCardActionsContext — the seam between board cards and the action
 * layer (PlanBoardActions). Cards read mutation callbacks, snooze, and
 * the decision drawer opener without prop drilling.
 */

import { createContext, useContext } from "react";
import type { RunBacklogTarget } from "../../../components/backlog/run-backlog-modal";
import type { StableItemCallbacks } from "../../../hooks/useCommandPostItemActions";
import type { SnoozePreset } from "../../../lib/snooze-utils";
import type { BacklogItem } from "../../../types";
import type { PlanCardData } from "../types";

export interface PlanCardActions {
  /** Stable mutation callbacks for a backlog item, or undefined when the
   *  item is not in the loaded backlog. */
  getCallbacks: (kind: string, name: string) => StableItemCallbacks | undefined;
  /** Full backlog item lookup (menu labels need status/kind context). */
  getBacklogItem: (kind: string, name: string) => BacklogItem | undefined;
  /** Open the run modal for one or more targets (bulk-threshold confirmed). */
  runTargets: (targets: RunBacklogTarget[]) => void;
  /** Snooze a card with one of the shared presets. */
  snoozeCard: (card: PlanCardData, preset: SnoozePreset) => void;
  /** Open the decision drawer, optionally scoped to one item (kind/name). */
  openDecisions: (scopeItemKey?: string) => void;
}

export const PlanCardActionsContext = createContext<PlanCardActions | null>(null);

export function usePlanCardActions(): PlanCardActions | null {
  return useContext(PlanCardActionsContext);
}
