/**
 * buildBacklogActionMenuItems
 *
 * Builds the overflow-menu actions for a backlog item (secondary CTAs, edit,
 * follow-up, retry, agent, archive, reset workshop, delete) for the
 * DetailPageHeader ellipsis menu. The item's primary CTA is rendered by
 * `HeaderPrimaryAction` and excluded here; status changes go through the
 * header's StatusBadge.
 *
 * A pure builder (not a hook): the detail page assembles it from the same
 * values it feeds BacklogDetailContext, since the page renders the provider
 * and cannot read its own context.
 */

import {
  Archive,
  Edit,
  MessageSquare,
  MessageSquareText,
  Play,
  RefreshCw,
  RotateCcw,
  Sparkles,
  Trash2,
} from "lucide-react";
import { type ActionMenuItem } from "../ui/action-menu";
import { selectors } from "../../consts/selectors";
import type { ItemActions } from "../../lib/backlog-queue-utils";

export interface BacklogActionMenuDetail {
  itemActions: ItemActions | null;
  isLocked: boolean;
  isTerminal: boolean;
  agentRunningLabel: string;
  workshopActionLabel: string;
  deliverableLabel: string;
  agentLabel: string;
  isRunningAgent: boolean;
}

export interface BacklogActionMenuOptions {
  isUpdating: boolean;
  onFinalizeWorkshop: () => void;
  onStartRun: () => void;
  onRunWorkshop: () => void;
  onEdit: () => void;
  onFollowUp: () => void;
  onRetry: () => void;
  onOpenAgentDialog: () => void;
  onArchive: () => void;
  onResetWorkshop: () => void;
  hasWorkshopRounds: boolean;
  onDelete: () => void;
}

export function buildBacklogActionMenuItems(detail: BacklogActionMenuDetail, {
  isUpdating,
  onFinalizeWorkshop,
  onStartRun,
  onRunWorkshop,
  onEdit,
  onFollowUp,
  onRetry,
  onOpenAgentDialog,
  onArchive,
  onResetWorkshop,
  hasWorkshopRounds,
  onDelete,
}: BacklogActionMenuOptions): ActionMenuItem[] {
  const {
    itemActions, isLocked, isTerminal,
    agentRunningLabel, workshopActionLabel, deliverableLabel,
    agentLabel, isRunningAgent,
  } = detail;

  if (!itemActions) return [];

  const items: ActionMenuItem[] = [];

  // Secondary CTAs — every available CTA except the primary one, which
  // HeaderPrimaryAction already renders as the visible header button.
  if (itemActions.primaryCta !== "finalize" && (itemActions.canFinalize || itemActions.finalizeDisabled)) {
    items.push({
      label: itemActions.agentExecuting ? agentRunningLabel : isRunningAgent ? "Starting..." : `Finalize ${deliverableLabel}`,
      icon: <Sparkles />,
      onSelect: onFinalizeWorkshop,
      disabled: itemActions.finalizeDisabled || isRunningAgent,
      title: (itemActions.finalizeDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  if (itemActions.primaryCta !== "run" && (itemActions.canRun || itemActions.runDisabled)) {
    items.push({
      label: itemActions.agentExecuting ? agentRunningLabel : "Run",
      icon: <Play />,
      onSelect: onStartRun,
      disabled: itemActions.runDisabled,
      title: itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  if (itemActions.primaryCta !== "workshop" && (itemActions.canWorkshop || itemActions.workshopDisabled)) {
    items.push({
      label: itemActions.agentExecuting ? agentRunningLabel : isRunningAgent ? "Starting..." : workshopActionLabel,
      icon: <MessageSquareText />,
      onSelect: onRunWorkshop,
      disabled: itemActions.workshopDisabled || isRunningAgent,
      title: (itemActions.workshopDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  items.push({
    label: "Edit",
    icon: <Edit />,
    onSelect: onEdit,
    testId: selectors.backlogDetails.editButton,
  });
  if (itemActions.canFollowUp) {
    items.push({
      label: "Follow Up",
      icon: <MessageSquare />,
      onSelect: onFollowUp,
    });
  }
  if (itemActions.canRetry) {
    items.push({
      label: "Retry",
      icon: <RefreshCw />,
      onSelect: onRetry,
      title: "Re-run with the same scope. Use Follow-Up if the work needs to change.",
    });
  }
  if (!itemActions.canFollowUp && !itemActions.canRetry && !isTerminal) {
    items.push({
      label: agentLabel,
      icon: <Sparkles />,
      onSelect: onOpenAgentDialog,
      disabled: isLocked,
      testId: selectors.backlogDetails.agentButton,
    });
  }
  if (itemActions.canArchive) {
    items.push({
      label: isUpdating ? "Archiving..." : "Archive",
      icon: <Archive />,
      onSelect: onArchive,
      disabled: isUpdating,
    });
  }
  if (hasWorkshopRounds && !isLocked) {
    items.push({
      label: "Reset Workshop",
      icon: <RotateCcw />,
      onSelect: onResetWorkshop,
      destructive: true,
    });
  }
  items.push({
    label: "Delete",
    icon: <Trash2 />,
    onSelect: onDelete,
    destructive: true,
    testId: selectors.backlogDetails.deleteButton,
  });

  return items;
}
