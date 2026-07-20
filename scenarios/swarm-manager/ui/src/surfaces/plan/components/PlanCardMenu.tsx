/**
 * PlanCardMenu — per-card action menu mapped to real levers via the
 * Command Post's shared mutation hook (through PlanCardActionsContext).
 * No drag on this board (D7): every transition is an explicit action.
 */

import { useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  backlogDetailPath,
  captureDetailPath,
  executionDetailPath,
  graphPath,
  initiativeDetailPath,
} from "../../../app/routes/route-paths";
import { ActionMenu, type ActionMenuItem } from "../../../components/ui/action-menu";
import { SetAsGoalDialog } from "../../../components/goals/SetAsGoalDialog";
import { SNOOZE_PRESETS } from "../../../lib/snooze-utils";
import type { PlanCardData } from "../types";
import { usePlanCardActions } from "./plan-card-actions-context";

export interface PlanCardMenuProps {
  card: PlanCardData;
}

export function PlanCardMenu({ card }: PlanCardMenuProps) {
  const actions = usePlanCardActions();
  const navigate = useNavigate();
  const [goalDialogOpen, setGoalDialogOpen] = useState(false);

  if (!actions) return null;

  const items: ActionMenuItem[] = [];
  const itemKey = card.itemKind && card.itemName ? `${card.itemKind}/${card.itemName}` : null;
  const callbacks = itemKey
    ? actions.getCallbacks(card.itemKind, card.itemName)
    : undefined;

  // Open the owning detail surface.
  items.push({
    label: "Open",
    testId: "plan-card-menu-open",
    onSelect: () => {
      if (card.executionId) {
        navigate(executionDetailPath(card.executionId));
      } else if (card.id.startsWith("capture/")) {
        navigate(captureDetailPath(card.id.slice("capture/".length)));
      } else if (card.id.startsWith("initiative/")) {
        navigate(initiativeDetailPath(card.id.slice("initiative/".length)));
      } else if (itemKey) {
        navigate(backlogDetailPath(card.itemKind, card.itemName));
      }
    },
  });

  // Gate-specific primary action.
  if (card.cardType === "gate" && card.gate) {
    if ((card.gate.kind === "decide" || card.gate.kind === "proposal") && itemKey) {
      items.push({
        label: card.gate.kind === "proposal"
          ? `Review ${card.gate.count} proposal${card.gate.count === 1 ? "" : "s"}`
          : `Answer ${card.gate.count} question${card.gate.count === 1 ? "" : "s"}`,
        testId: "plan-card-menu-answer",
        onSelect: () => actions.openDecisions(itemKey),
      });
    }
  }

  // Item levers via the shared Command Post mutation hook.
  if (callbacks && card.cardType !== "outcome") {
    if (card.action === "run") {
      items.push({ label: "Run", testId: "plan-card-menu-run", onSelect: callbacks.onRun });
    }
    if (card.action === "workshop") {
      items.push({ label: "Workshop", testId: "plan-card-menu-workshop", onSelect: callbacks.onWorkshop });
    }
    if (card.action === "finalize") {
      items.push({ label: "Finalize", testId: "plan-card-menu-finalize", onSelect: callbacks.onFinalize });
    }
    if (card.status === "backlog") {
      items.push({
        label: "Mark ready",
        testId: "plan-card-menu-status-ready",
        onSelect: () => callbacks.onStatusChange("ready"),
      });
    } else if (card.status === "ready") {
      items.push({
        label: "Move to backlog",
        testId: "plan-card-menu-status-backlog",
        onSelect: () => callbacks.onStatusChange("backlog"),
      });
    }
  }

  // Archive: terminal outcomes and stuck items.
  if (callbacks) {
    items.push({
      label: "Archive",
      testId: "plan-card-menu-archive",
      destructive: true,
      onSelect: callbacks.onArchive,
    });
  }

  // Promote an item card to a goal (target of a new or existing goal).
  if (itemKey && card.cardType !== "outcome") {
    items.push({
      label: "Set as goal",
      testId: "plan-card-menu-set-goal",
      onSelect: () => setGoalDialogOpen(true),
    });
  }

  // Board → Focus bridge: view this card's neighborhood on the graph.
  items.push({
    label: "Focus on graph",
    testId: "plan-card-menu-focus",
    onSelect: () => navigate(graphPath({ lens: "focus", select: card.id })),
  });

  // Snooze presets (client-side, matching the Command Post popover).
  for (const preset of SNOOZE_PRESETS) {
    items.push({
      label: `Snooze ${preset.label.toLowerCase()}`,
      testId: `plan-card-menu-snooze-${preset.label.toLowerCase().replaceAll(" ", "-")}`,
      onSelect: () => actions.snoozeCard(card, preset),
    });
  }

  return (
    <>
      <ActionMenu
        items={items}
        label="Card actions"
        triggerTestId={`plan-card-menu-${card.id}`}
        menuTestId="plan-card-menu"
        triggerSize="icon"
      />
      {itemKey && (
        <SetAsGoalDialog
          isOpen={goalDialogOpen}
          onClose={() => setGoalDialogOpen(false)}
          targetRef={itemKey}
          targetTitle={card.title}
        />
      )}
    </>
  );
}
