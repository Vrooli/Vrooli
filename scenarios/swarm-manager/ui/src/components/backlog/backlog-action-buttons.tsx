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
import { Button } from "../ui/button";
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

  const rowButtonClass =
    "h-10 w-full justify-start rounded-lg border-slate-700/80 bg-slate-900/40 px-3 text-sm text-slate-100 hover:bg-slate-800/70";
  const primaryRowButtonClass =
    "h-10 w-full justify-start rounded-lg border-transparent bg-slate-100 px-3 text-sm text-slate-900 hover:bg-white";
  const destructiveRowButtonClass =
    "h-10 w-full justify-start rounded-lg border-red-500/30 bg-red-500/10 px-3 text-sm text-red-200 hover:bg-red-500/20";

  return (
    <div className="space-y-2">
      {(itemActions.canFinalize || itemActions.finalizeDisabled) && (
        <Button
          variant="default"
          size="sm"
          className={itemActions.primaryCta === "finalize" ? primaryRowButtonClass : rowButtonClass}
          onClick={onFinalizeWorkshop}
          disabled={itemActions.finalizeDisabled || isRunningAgent}
          title={(itemActions.finalizeDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined}
        >
          <Sparkles className="mr-2 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : isRunningAgent ? "Starting..." : `Finalize ${deliverableLabel}`}
        </Button>
      )}
      {(itemActions.canRun || itemActions.runDisabled) && (
        <Button
          variant="default"
          size="sm"
          className={itemActions.primaryCta === "run" ? primaryRowButtonClass : rowButtonClass}
          onClick={onStartRun}
          disabled={itemActions.runDisabled}
          title={itemActions.runDisabled && itemActions.disabledReason ? itemActions.disabledReason : undefined}
        >
          <Play className="mr-2 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : "Run"}
        </Button>
      )}
      {(itemActions.canWorkshop || itemActions.workshopDisabled) && (
        <Button
          variant="default"
          size="sm"
          className={itemActions.primaryCta === "workshop" ? primaryRowButtonClass : rowButtonClass}
          onClick={onRunWorkshop}
          disabled={itemActions.workshopDisabled || isRunningAgent}
          title={(itemActions.workshopDisabled || isRunningAgent) && itemActions.disabledReason ? itemActions.disabledReason : undefined}
        >
          <MessageSquareText className="mr-2 h-4 w-4" />
          {itemActions.agentRunning ? agentRunningLabel : isRunningAgent ? "Starting..." : workshopActionLabel}
        </Button>
      )}
      {itemActions.disabledReason && (itemActions.runDisabled || itemActions.workshopDisabled || itemActions.finalizeDisabled) ? (
        <p className="text-xs text-amber-400/80">{itemActions.disabledReason}</p>
      ) : null}
      {itemActions.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.canRun && !itemActions.runDisabled && !itemActions.canWorkshop && !itemActions.workshopDisabled && !itemActions.canFinalize && !itemActions.finalizeDisabled ? (
        <p className="text-xs text-slate-500">{itemActions.notQueueableReason}</p>
      ) : null}
      <Button variant="outline" size="sm" className={rowButtonClass} onClick={onEdit}>
        <Edit className="mr-2 h-4 w-4" />
        Edit
      </Button>
      {itemActions.canFollowUp ? (
        <Button variant="outline" size="sm" className={rowButtonClass} onClick={onFollowUp}>
          <MessageSquare className="mr-2 h-4 w-4" />
          Follow Up
        </Button>
      ) : null}
      {itemActions.canRetry ? (
        <Button
          variant="outline"
          size="sm"
          className={rowButtonClass}
          onClick={onRetry}
          title="Re-run with the same scope. Use Follow-Up if the work needs to change."
        >
          <RefreshCw className="mr-2 h-4 w-4" />
          Retry
        </Button>
      ) : null}
      {!itemActions.canFollowUp && !itemActions.canRetry && !isTerminal ? (
        <Button variant="outline" size="sm" className={rowButtonClass} onClick={onOpenAgentDialog} disabled={isLocked}>
          <Sparkles className="mr-2 h-4 w-4" />
          {agentLabel}
        </Button>
      ) : null}
      {itemActions.canArchive && (
        <Button variant="outline" size="sm" className={rowButtonClass} onClick={onArchive} disabled={isUpdating}>
          <Archive className="mr-2 h-4 w-4" />
          {isUpdating ? "Archiving..." : "Archive"}
        </Button>
      )}
      {!isLocked && (
        <div className="space-y-1">
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
      {hasWorkshopRounds && !isLocked && (
        <Button variant="outline" size="sm" className={destructiveRowButtonClass} onClick={onResetWorkshop}>
          <RotateCcw className="mr-2 h-4 w-4" />
          Reset Workshop
        </Button>
      )}
      <Button variant="outline" size="sm" className={destructiveRowButtonClass} onClick={onDelete}>
        <Trash2 className="mr-2 h-4 w-4" />
        Delete
      </Button>
    </div>
  );
}
