/**
 * buildBacklogActionMenuItems
 *
 * Builds the overflow-menu actions for a backlog item (secondary CTAs, edit,
 * follow-up, retry, archive, and delete) for the
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
	Play,
	RefreshCw,
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
}

export interface BacklogActionMenuOptions {
  isUpdating: boolean;
  onStartRun: () => void;
  onEdit: () => void;
  onFollowUp: () => void;
  onRetry: () => void;
  onArchive: () => void;
  onDelete: () => void;
}

export function buildBacklogActionMenuItems(detail: BacklogActionMenuDetail, {
  isUpdating,
	onStartRun,
  onEdit,
  onFollowUp,
  onRetry,
  onArchive,
  onDelete,
}: BacklogActionMenuOptions): ActionMenuItem[] {
  const { itemActions, agentRunningLabel } = detail;

  if (!itemActions) return [];

  const items: ActionMenuItem[] = [];

  // Secondary CTAs — every available CTA except the primary one, which
  // HeaderPrimaryAction already renders as the visible header button.
  if (itemActions.primaryCta !== "run" && (itemActions.canRun || itemActions.runDisabled)) {
    items.push({
      label: itemActions.agentExecuting ? agentRunningLabel : "Run",
      icon: <Play />,
      description: "Run this item through its next execution step.",
      onSelect: onStartRun,
      disabled: itemActions.runDisabled,
      title: itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  items.push({
    label: "Edit",
    icon: <Edit />,
    description: "Change the canonical title, scope, status, or priority.",
    onSelect: onEdit,
    testId: selectors.backlogDetails.editButton,
  });
  if (itemActions.canFollowUp) {
    items.push({
      label: "Follow Up",
      icon: <MessageSquare />,
      description: "Give the most recent run additional direction.",
      onSelect: onFollowUp,
    });
  }
  if (itemActions.canRetry) {
    items.push({
      label: "Retry",
      icon: <RefreshCw />,
      description: "Repeat the last run with the same scope.",
      onSelect: onRetry,
      title: "Re-run with the same scope. Use Follow-Up if the work needs to change.",
    });
  }
  if (itemActions.canArchive) {
    items.push({
      label: isUpdating ? "Archiving..." : "Archive",
      icon: <Archive />,
      description: "Hide this item from active work while retaining history.",
      onSelect: onArchive,
      disabled: isUpdating,
    });
  }
  items.push({
    label: "Delete",
    icon: <Trash2 />,
    description: "Permanently remove this backlog item.",
    onSelect: onDelete,
    destructive: true,
    testId: selectors.backlogDetails.deleteButton,
  });

  return items;
}
