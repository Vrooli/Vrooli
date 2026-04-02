/**
 * BacklogDesktopHeader
 *
 * Desktop-only header card for a backlog item showing status, priority,
 * timeline button, title, and action buttons.
 *
 * Reads shared state from BacklogDetailContext.
 */

import type { ReactNode } from "react";
import { Edit, History, RotateCcw, Sparkles, Trash2 } from "lucide-react";
import { Card } from "../ui/card";
import { Button } from "../ui/button";
import { Select } from "../ui/select";
import { selectors } from "../../consts/selectors";
import {
  BACKLOG_STATUS_COLORS,
  USER_SETTABLE_STATUSES,
  formatBacklogStatus,
} from "../../types";
import type { BacklogStatus } from "../../types";
import type { BacklogItem } from "../../types/domain";
import { useBacklogDetail } from "../../contexts/BacklogDetailContext";

export interface BacklogDesktopHeaderProps {
  item: BacklogItem;
  deleteError: string | null;
  primaryAction: ReactNode;
  onEdit: () => void;
  onDelete: () => void;
  onResetWorkshop: () => void;
  hasWorkshopRounds: boolean;
  onOpenAgentDialog: () => void;
  onOpenTimeline: () => void;
  onStatusChange: (newStatus: BacklogStatus) => void;
}

export function BacklogDesktopHeader({
  item,
  deleteError,
  primaryAction,
  onEdit,
  onDelete,
  onResetWorkshop,
  hasWorkshopRounds,
  onOpenAgentDialog,
  onOpenTimeline,
  onStatusChange,
}: BacklogDesktopHeaderProps) {
  const { isLocked, itemActions, agentLabel } = useBacklogDetail();

  return (
    <Card data-testid={selectors.backlogDetails.header}>
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            {isLocked ? (
              <>
                <span
                  className={`inline-block h-3 w-3 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                />
                <span className="text-xs uppercase tracking-wider text-slate-500 sm:text-sm">
                  {formatBacklogStatus(item.status)}
                </span>
              </>
            ) : (
              <div className="flex items-center gap-1.5">
                <span
                  className={`inline-block h-2.5 w-2.5 rounded-full ${BACKLOG_STATUS_COLORS[item.status] ?? "bg-slate-500"}`}
                />
                <Select
                  variant="compact"
                  withChevron
                  value={item.status}
                  onChange={(e) => {
                    const newStatus = e.target.value as BacklogStatus;
                    if (newStatus !== item.status) onStatusChange(newStatus);
                  }}
                  data-testid={selectors.backlogDetails.statusSelect}
                  className="w-auto uppercase tracking-wider"
                >
                  {USER_SETTABLE_STATUSES.map((s) => (
                    <option key={s} value={s}>{formatBacklogStatus(s)}</option>
                  ))}
                </Select>
              </div>
            )}
            <span className="rounded-full bg-slate-700 px-3 py-1 text-xs text-slate-300 sm:text-sm">
              Priority {item.priority}
            </span>
            <Button
              variant="outline"
              size="sm"
              className="h-7 w-7 rounded-md border-transparent bg-transparent p-0 hover:bg-slate-800/70"
              onClick={onOpenTimeline}
              aria-label="View activity timeline"
              data-testid={selectors.backlogDetails.timelineButton}
            >
              <History className="h-3.5 w-3.5" />
            </Button>
          </div>
          <h1
            className="text-xl font-bold text-slate-100 sm:text-2xl"
            data-testid={selectors.backlogDetails.title}
          >
            {item.title}
          </h1>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {primaryAction}
          {itemActions?.notQueueableReason && !itemActions.locked && !itemActions.terminal && !itemActions.canRun && !itemActions.runDisabled && !itemActions.canWorkshop && !itemActions.workshopDisabled && !itemActions.canFinalize && !itemActions.finalizeDisabled ? (
            <span className="max-w-xs text-xs text-slate-500">{itemActions.notQueueableReason}</span>
          ) : null}
          <Button
            variant="outline"
            size="sm"
            className="hidden lg:inline-flex"
            data-testid={selectors.backlogDetails.editButton}
            onClick={onEdit}
          >
            <Edit className="mr-2 h-4 w-4" />
            Edit
          </Button>
          {hasWorkshopRounds && !isLocked && (
            <Button
              variant="outline"
              size="sm"
              className="hidden border-red-500/30 text-red-300 hover:bg-red-500/10 lg:inline-flex"
              onClick={onResetWorkshop}
            >
              <RotateCcw className="mr-2 h-4 w-4" />
              Reset Workshop
            </Button>
          )}
          <Button
            variant="outline"
            size="sm"
            className="hidden lg:inline-flex"
            data-testid={selectors.backlogDetails.deleteButton}
            onClick={onDelete}
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="hidden lg:inline-flex"
            onClick={onOpenAgentDialog}
            disabled={isLocked}
            data-testid={selectors.backlogDetails.agentButton}
          >
            <Sparkles className="mr-2 h-4 w-4" />
            {agentLabel}
          </Button>
        </div>
      </div>

      {deleteError && (
        <div className="mt-4 space-y-2">
          <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
            {deleteError}
          </div>
        </div>
      )}
    </Card>
  );
}
