/**
 * useSetAsGoalMenu — packages the "Set as goal" context-menu item together with
 * the SetAsGoalDialog it opens, so any list row (sidebar backlog items,
 * initiatives, …) can offer "Add to a goal" by reusing the exact flow behind
 * the graph node inspector's and plan card menu's "Set as goal" button.
 */

import { useState } from "react";
import type { ReactNode } from "react";
import { ENTITY_TYPE_ICONS } from "../../types/constants";
import type { ActionMenuItem } from "../ui/action-menu";
import { SetAsGoalDialog } from "./SetAsGoalDialog";
import type { GoalTarget } from "./goal-target";

export interface SetAsGoalMenu {
  /** Menu item to include in a context menu (null when the target is null). */
  item: ActionMenuItem | null;
  /** The dialog element to render alongside the row (null when unavailable). */
  dialog: ReactNode;
}

/**
 * Returns a "Set as goal" menu item plus the dialog it controls for the given
 * goal target. Pass `null` when the entity cannot be a goal target.
 */
export function useSetAsGoalMenu(target: GoalTarget | null): SetAsGoalMenu {
  const [open, setOpen] = useState(false);

  const item: ActionMenuItem | null = target
    ? {
        label: "Set as goal",
        icon: <ENTITY_TYPE_ICONS.goal className="text-cyan-400" aria-hidden />,
        onSelect: () => setOpen(true),
        testId: "context-menu-set-goal",
      }
    : null;

  const dialog: ReactNode = target ? (
    <SetAsGoalDialog
      isOpen={open}
      onClose={() => setOpen(false)}
      targetRef={target.ref}
      targetTitle={target.title}
    />
  ) : null;

  return { item, dialog };
}
