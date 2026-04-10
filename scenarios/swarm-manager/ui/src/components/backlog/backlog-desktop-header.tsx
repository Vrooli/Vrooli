/**
 * BacklogDesktopHeader
 *
 * Desktop-only header card for a backlog item showing status, priority,
 * title, and action buttons.
 *
 * Reads shared state from BacklogDetailContext.
 */

import type { ReactNode } from "react";
import { Archive, ArchiveRestore, Edit, RotateCcw, Sparkles, Trash2 } from "lucide-react";
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
import { formatRelativeTime } from "../../lib/format-utils";

export interface BacklogDesktopHeaderProps {
  item: BacklogItem;
  deleteError: string | null;
  archiveError: string | null;
  primaryAction: ReactNode;
  onEdit: () => void;
  onDelete: () => void;
  onArchive: () => void;
  onUnarchive: () => void;
  isArchiving: boolean;
  onResetWorkshop: () => void;
  hasWorkshopRounds: boolean;
  onOpenAgentDialog: () => void;
  onStatusChange: (newStatus: BacklogStatus) => void;
}

export function BacklogDesktopHeader({
  item,
  deleteError,
  archiveError,
  primaryAction,
  onEdit,
  onDelete,
  onArchive,
  onUnarchive,
  isArchiving,
  onResetWorkshop,
  hasWorkshopRounds,
  onOpenAgentDialog,
  onStatusChange,
}: BacklogDesktopHeaderProps) {
  const { isLocked, itemActions, agentLabel } = useBacklogDetail();
  const isArchived = item.archivedAt != null;

  return (
    <Card data-testid={selectors.backlogDetails.header}>
      {isArchived && (
        <div className="mb-3 flex items-center justify-between rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2">
          <div className="flex items-center gap-2 text-sm text-amber-300">
            <Archive className="h-4 w-4" />
            <span>Archived {formatRelativeTime(item.archivedAt ?? "")}</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            className="border-amber-500/30 text-amber-300 hover:bg-amber-500/10"
            onClick={onUnarchive}
            disabled={isArchiving}
          >
            <ArchiveRestore className="mr-2 h-4 w-4" />
            {isArchiving ? "Restoring..." : "Unarchive"}
          </Button>
        </div>
      )}
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
          {itemActions?.disabledReason && (itemActions.runDisabled || itemActions.workshopDisabled || itemActions.finalizeDisabled) ? (
            <span className="max-w-xs text-xs text-amber-400/80">{itemActions.disabledReason}</span>
          ) : null}
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
          {!isArchived && itemActions?.canArchive && (
            <Button
              variant="outline"
              size="sm"
              className="hidden lg:inline-flex"
              onClick={onArchive}
              disabled={isArchiving}
            >
              <Archive className="mr-2 h-4 w-4" />
              {isArchiving ? "Archiving..." : "Archive"}
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

      {(deleteError || archiveError) && (
        <div className="mt-4 space-y-2">
          {deleteError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {deleteError}
            </div>
          )}
          {archiveError && (
            <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {archiveError}
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
