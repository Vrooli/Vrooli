/**
 * ExecutionsTab - Lists execution records with status, mode, and timing.
 */

import { memo, useEffect, useMemo, useState } from "react";
import { Loader2, Square } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useExecutionStore } from "../../../../stores";
import { executionService } from "../../../../services";
import { buildExecutionNodeId } from "../../lib/node-id-parser";
import { ExecutionSummaryCard } from "../../../../components/execution/execution-summary-card";
import { matchesSearch } from "./useSidebarSearch";
import type { ExecutionRecord } from "../../../../types";
import type { ExecutionFilters, SortConfig } from "./types";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { runBulkAction, summarizeBulkOutcomes, type BulkOutcome } from "./bulk-actions";

interface ExecutionsTabProps {
  searchQuery: string;
  filters: ExecutionFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelection?: (id: string) => void;
  onVisibleIdsChange?: (ids: string[]) => void;
}

function applyFilters(items: ExecutionRecord[], filters: ExecutionFilters): ExecutionRecord[] {
  return items.filter((item) => {
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.modes.length > 0 && !filters.modes.includes(item.mode)) return false;
    return true;
  });
}

function applySort(items: ExecutionRecord[], sort: SortConfig): ExecutionRecord[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

  sorted.sort((a, b) => {
    switch (sort.field) {
      case "recency":
        return (new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return a.backlogName.localeCompare(b.backlogName) * dir;
      default:
        return (new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()) * dir;
    }
  });

  return sorted;
}

function executionSelectionId(item: ExecutionRecord): string {
  return `execution:${item.executionId}`;
}

function ExecutionsTabImpl({
  searchQuery,
  filters,
  sort,
  onItemClick,
  onClearSearch,
  selectionMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
  onVisibleIdsChange,
}: ExecutionsTabProps) {
  const items = useExecutionStore((s) => s.items);

  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((item) =>
      matchesSearch(searchQuery, item.backlogName, item.status, item.mode),
    );
  }
  const sorted = applySort(filtered, sort);
  useEffect(() => {
    onVisibleIdsChange?.(sorted.map(executionSelectionId));
  }, [onVisibleIdsChange, sorted]);
  const selectedExecutions = useMemo(
    () => sorted.filter((item) => selectedIds.has(executionSelectionId(item))),
    [selectedIds, sorted],
  );

  if (sorted.length === 0) {
    const filtersActive = filters.statuses.length > 0 || filters.modes.length > 0;
    const title = filtersActive ? "No executions match your filters." : "No executions yet.";
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.executions}
        title={title}
        hint={filtersActive ? undefined : "Runs and reviews appear here as agents start work."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <>
      {selectionMode && <ExecutionBulkActions selectedExecutions={selectedExecutions} />}
      <div className="space-y-1.5">
        {sorted.map((item) => {
          const nodeId = buildExecutionNodeId(item.executionId);
          const selectionId = executionSelectionId(item);
          return (
            <ExecutionSummaryCard
              key={item.executionId}
              item={item}
              onOpen={() => onItemClick(nodeId)}
              batchMode={selectionMode}
              batchSelected={selectedIds.has(selectionId)}
              onBatchToggle={() => onToggleSelection?.(selectionId)}
            />
          );
        })}
      </div>
    </>
  );
}

export const ExecutionsTab = memo(ExecutionsTabImpl);

function ExecutionBulkActions({ selectedExecutions }: { selectedExecutions: ExecutionRecord[] }) {
  const fetchExecutions = useExecutionStore((s) => s.fetchExecutions);
  const upsertExecution = useExecutionStore((s) => s.upsertExecution);
  const [action, setAction] = useState<"cancel" | "retry" | "review">("cancel");
  const [confirmCancel, setConfirmCancel] = useState(false);
  const [running, setRunning] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [outcomes, setOutcomes] = useState<BulkOutcome[]>([]);

  const eligible = selectedExecutions.filter((execution) => {
    if (action === "cancel") return ["pending", "starting", "running", "validating"].includes(execution.status);
    if (action === "retry") return execution.status === "failed" || execution.status === "canceled";
    return execution.status === "needs_review" || execution.status === "completed";
  });

  const execute = async () => {
    setRunning(true);
    setSummary(null);
    setOutcomes([]);
    try {
      const next = await runBulkAction(eligible, {
        getId: executionSelectionId,
        getLabel: (execution) => execution.backlogName,
        run: async (execution) => {
          const updated = action === "cancel"
            ? await executionService.cancel(execution.executionId)
            : action === "retry"
              ? await executionService.retry(execution.executionId)
              : await executionService.triggerReview(execution.executionId);
          upsertExecution(updated);
        },
      });
      setOutcomes(next);
      setSummary(summarizeBulkOutcomes(next));
      await fetchExecutions({ force: true });
    } finally {
      setRunning(false);
      setConfirmCancel(false);
    }
  };

  const failed = outcomes.filter((outcome) => outcome.status === "failed");

  return (
    <div className="mb-2 rounded-lg border border-slate-800 bg-slate-900/70 p-2" data-testid="sidebar-execution-bulk-actions">
      <div className="flex flex-wrap items-center gap-2">
        <select value={action} onChange={(event) => setAction(event.target.value as "cancel" | "retry" | "review")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Execution bulk action">
          <option value="cancel">Cancel active</option>
          <option value="retry">Retry failed</option>
          <option value="review">Trigger review</option>
        </select>
        <button
          type="button"
          disabled={selectedExecutions.length === 0 || eligible.length === 0 || running}
          onClick={() => {
            if (action === "cancel") setConfirmCancel(true);
            else void execute();
          }}
          className="inline-flex h-8 items-center gap-1.5 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : action === "cancel" ? <Square className="h-3.5 w-3.5" /> : null}
          Apply
        </button>
      </div>
      <div className="mt-1.5 text-[11px] text-slate-500">{eligible.length} eligible{summary ? ` - ${summary}` : ""}</div>
      {failed.length > 0 && <div className="mt-1 text-[11px] text-red-300">{failed.map((outcome) => <div key={outcome.id}>{outcome.label}: {outcome.message}</div>)}</div>}
      <ConfirmDialog
        isOpen={confirmCancel}
        onClose={() => setConfirmCancel(false)}
        onConfirm={() => void execute()}
        title="Cancel selected executions"
        description={`Cancel ${eligible.length} active execution${eligible.length === 1 ? "" : "s"}?`}
        confirmLabel="Cancel selected"
        isLoading={running}
      />
    </div>
  );
}
