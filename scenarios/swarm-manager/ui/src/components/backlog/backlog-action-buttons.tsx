/**
 * BacklogActionButtons
 *
 * Renders the full set of action buttons for a backlog item (finalize, run,
 * workshop, edit, follow-up, agent, archive, status, delete).
 *
 * Reads shared state from BacklogDetailContext.
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
import { ActionMenuItemButton, type ActionMenuItem } from "../ui/action-menu";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import { USER_SETTABLE_STATUSES, formatBacklogStatus } from "../../types";
import type { BacklogStatus } from "../../types";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";

export interface BacklogActionButtonsProps {
  item: { title: string; description: string; status: BacklogStatus; priority: number; tags: string[] };
  isUpdating: boolean;
  onFinalizeWorkshop: () => void;
  onStartRun: () => void;
  onRunWorkshop: () => void;
  onEdit: () => void;
  onFollowUp: () => void;
  onRetry: () => void;
  onOpenAgentDialog: () => void;
  onArchive: () => void;
  onStatusChange: (newStatus: BacklogStatus) => void;
  onResetWorkshop: () => void;
  hasWorkshopRounds: boolean;
  onDelete: () => void;
}

export function BacklogActionButtons({
  item,
  isUpdating,
  onFinalizeWorkshop,
  onStartRun,
  onRunWorkshop,
  onEdit,
  onFollowUp,
  onRetry,
  onOpenAgentDialog,
  onArchive,
  onStatusChange,
  onResetWorkshop,
  hasWorkshopRounds,
  onDelete,
}: BacklogActionButtonsProps) {
  const {
    itemActions, isLocked, isTerminal,
    agentRunningLabel, workshopActionLabel, deliverableLabel,
    agentLabel, isRunningAgent,
  } = useBacklogDetail();

  if (!itemActions) return null;

  const primaryActionItems: ActionMenuItem[] = [];
  const secondaryActionItems: ActionMenuItem[] = [];
  const destructiveActionItems: ActionMenuItem[] = [];

  if (itemActions.canFinalize || itemActions.finalizeDisabled) {
    primaryActionItems.push({
      label: itemActions.agentExecuting ? agentRunningLabel : isRunningAgent ? "Starting..." : `Finalize ${deliverableLabel}`,
      icon: <Sparkles />,
      onSelect: onFinalizeWorkshop,
      disabled: itemActions.finalizeDisabled || isRunningAgent,
      title: (itemActions.finalizeDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  if (itemActions.canRun || itemActions.runDisabled) {
    primaryActionItems.push({
      label: itemActions.agentExecuting ? agentRunningLabel : "Run",
      icon: <Play />,
      onSelect: onStartRun,
      disabled: itemActions.runDisabled,
      title: itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  if (itemActions.canWorkshop || itemActions.workshopDisabled) {
    primaryActionItems.push({
      label: itemActions.agentExecuting ? agentRunningLabel : isRunningAgent ? "Starting..." : workshopActionLabel,
      icon: <MessageSquareText />,
      onSelect: onRunWorkshop,
      disabled: itemActions.workshopDisabled || isRunningAgent,
      title: (itemActions.workshopDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined,
    });
  }
  secondaryActionItems.push({
    label: "Edit",
    icon: <Edit />,
    onSelect: onEdit,
  });
  if (itemActions.canFollowUp) {
    secondaryActionItems.push({
      label: "Follow Up",
      icon: <MessageSquare />,
      onSelect: onFollowUp,
    });
  }
  if (itemActions.canRetry) {
    secondaryActionItems.push({
      label: "Retry",
      icon: <RefreshCw />,
      onSelect: onRetry,
      title: "Re-run with the same scope. Use Follow-Up if the work needs to change.",
    });
  }
  if (!itemActions.canFollowUp && !itemActions.canRetry && !isTerminal) {
    secondaryActionItems.push({
      label: agentLabel,
      icon: <Sparkles />,
      onSelect: onOpenAgentDialog,
      disabled: isLocked,
    });
  }
  if (itemActions.canArchive) {
    secondaryActionItems.push({
      label: isUpdating ? "Archiving..." : "Archive",
      icon: <Archive />,
      onSelect: onArchive,
      disabled: isUpdating,
    });
  }
  if (hasWorkshopRounds && !isLocked) {
    destructiveActionItems.push({
      label: "Reset Workshop",
      icon: <RotateCcw />,
      onSelect: onResetWorkshop,
      destructive: true,
    });
  }
  destructiveActionItems.push({
    label: "Delete",
    icon: <Trash2 />,
    onSelect: onDelete,
    destructive: true,
  });

  return (
    <div className="py-1">
      {primaryActionItems.map((actionItem) => (
        <ActionMenuItemButton key={actionItem.label} item={actionItem} />
      ))}
      {itemActions.disabledReason && (itemActions.runDisabled || itemActions.workshopDisabled || itemActions.finalizeDisabled) ? (
        <p className="px-3 py-2 text-xs text-amber-400/80">{itemActions.disabledReason}</p>
      ) : null}
      {itemActions.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.canRun && !itemActions.runDisabled && !itemActions.canWorkshop && !itemActions.workshopDisabled && !itemActions.canFinalize && !itemActions.finalizeDisabled ? (
        <p className="px-3 py-2 text-xs text-slate-500">{itemActions.notQueueableReason}</p>
      ) : null}
      {secondaryActionItems.map((actionItem) => (
        <ActionMenuItemButton key={actionItem.label} item={actionItem} />
      ))}
      {!isLocked && (
        <div className="px-3 py-2">
          <label htmlFor="action-status-select" className="text-xs text-slate-400">
            Status
          </label>
          <Select
            id="action-status-select"
            variant="filter"
            withChevron
            value={item.status}
            onChange={(e) => {
              const newStatus = e.target.value as BacklogStatus;
              if (newStatus !== item.status) onStatusChange(newStatus);
            }}
            data-testid={selectors.backlogDetails.statusSelect}
          >
            {USER_SETTABLE_STATUSES.map((s) => (
              <option key={s} value={s}>{formatBacklogStatus(s)}</option>
            ))}
          </Select>
        </div>
      )}
      {destructiveActionItems.map((actionItem) => (
        <ActionMenuItemButton key={actionItem.label} item={actionItem} />
      ))}
    </div>
  );
}
