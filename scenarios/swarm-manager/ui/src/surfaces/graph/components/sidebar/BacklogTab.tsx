/**
 * BacklogTab - Lists backlog items with rich action cards.
 *
 * Uses the shared BacklogCard component, providing inline
 * decision answering, run/workshop/finalize actions, and follow-up/archive.
 */

import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ListTodo } from "lucide-react";
import { useBacklogStore } from "../../../../stores";
import { useSnoozedKeys } from "../../../../stores/snooze-store";
import { getItemActions } from "../../../../lib";
import { buildBacklogCompareFn, sortBacklogItems } from "../../../../lib/backlog-sort";
import { computeUnblockingMap } from "../../../../lib/dependency-sort";
import { filterSnoozed, snoozeKeyForBacklog } from "../../../../lib/snooze-utils";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import { BacklogCard } from "../../../../components/backlog/backlog-card";
import { RunBacklogModal } from "../../../../components/backlog/run-backlog-modal";
import type { RunBacklogTarget } from "../../../../components/backlog/run-backlog-modal";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { useCommandPostItemActions } from "../../../../hooks/useCommandPostItemActions";
import type { BacklogItem } from "../../../../types";
import type { BacklogFilters, SortConfig } from "./types";
import { backlogDetailPath } from "../../../../app/routes/route-paths";
import { SidebarEmptyState } from "./SidebarEmptyState";

interface PlanValidationSummary {
  passed: boolean;
}

function parsePlanValidationSummary(validationJson: string): PlanValidationSummary | null {
  try {
    const parsed: unknown = JSON.parse(validationJson);
    if (
      typeof parsed === "object"
      && parsed !== null
      && "passed" in parsed
      && typeof parsed.passed === "boolean"
    ) {
      return { passed: parsed.passed };
    }
  } catch {
    return null;
  }
  return null;
}

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
}

function applyFilters(items: BacklogItem[], filters: BacklogFilters): BacklogItem[] {
  return items.filter((item) => {
    // Hide archived items unless showArchived is on
    if (item.archivedAt != null && !filters.showArchived) return false;
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.priorityMin !== null && item.priority < filters.priorityMin) return false;
    if (filters.priorityMax !== null && item.priority > filters.priorityMax) return false;
    if (filters.validationStatus) {
      const json = item.planValidationJson;
      if (filters.validationStatus === "none") {
        if (json) return false;
      } else if (filters.validationStatus === "passed") {
        if (!json) return false;
        const validation = parsePlanValidationSummary(json);
        if (!validation?.passed) return false;
      } else if (filters.validationStatus === "failed") {
        if (!json) return false;
        const validation = parsePlanValidationSummary(json);
        if (validation?.passed !== false) return false;
      }
    }
    return true;
  });
}

function applySort(items: BacklogItem[], sort: SortConfig, allItems: BacklogItem[]): BacklogItem[] {
  const unblockingMap = computeUnblockingMap(allItems);
  return sortBacklogItems(items, buildBacklogCompareFn(sort, unblockingMap), allItems);
}

function hasActiveFilters(filters: BacklogFilters): boolean {
  return filters.statuses.length > 0 || filters.kinds.length > 0 || filters.priorityMin !== null || filters.priorityMax !== null || filters.showArchived || filters.validationStatus !== "";
}

export function BacklogTab({ searchQuery, filters, sort, onItemClick, onClearSearch }: BacklogTabProps) {
  const navigate = useNavigate();
  const items = useBacklogStore((s) => s.items);
  const blockingMap = useBacklogStore((s) => s.blockingMap);
  const fetchBacklog = useBacklogStore((s) => s.fetchBacklog);
  const snoozedKeys = useSnoozedKeys();

  const [runModalTarget, setRunModalTarget] = useState<RunBacklogTarget | undefined>();

  // ── Shared action wiring ────────────────────────────────────────────
  const {
    getItemCallbacks,
    activeRunKeys,
    readinessMap,
    pendingQuestionsMap,
    attentionReasonsMap,
    completedSteppers,
    transitionItems,
    handleStepperCompleted,
    workshopBlockingConfirm,
    setWorkshopBlockingConfirm,
    confirmWorkshopOverride,
  } = useCommandPostItemActions({
    onSelectBacklog: (kind, name) => navigate(backlogDetailPath(kind, name)),
    onRunItem: (kind, name, title) => setRunModalTarget({ kind, name, title }),
  });

  // ── Filter, search, snooze, sort ─────────────────────────────────────
  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((item) =>
      matchesSearch(searchQuery, item.title, item.name, item.description, ...(item.tags ?? [])),
    );
  }
  filtered = filterSnoozed(filtered, (item) => snoozeKeyForBacklog(item.kind, item.name), snoozedKeys);
  const sorted = applySort(filtered, sort, items);

  if (sorted.length === 0) {
    const filtersActive = hasActiveFilters(filters);
    const title = filtersActive
      ? "No backlog items match your filters."
      : "No backlog items yet.";
    return (
      <SidebarEmptyState
        icon={ListTodo}
        title={title}
        hint={filtersActive ? undefined : "Capture an idea or chore to get started."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <>
      <div className="space-y-2">
        {sorted.map((item) => {
          const nodeId = buildBacklogNodeId(item.kind, item.name);
          const itemKey = `${item.kind}/${item.name}`;
          const readiness = readinessMap.get(itemKey);
          const reasons = attentionReasonsMap.get(itemKey) ?? [];
          const callbacks = getItemCallbacks(item);

          return (
            <button
              key={nodeId}
              type="button"
              onClick={() => onItemClick(nodeId)}
              className="group w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
              data-testid="sidebar-backlog-item"
            >
              <BacklogCard
                item={item}
                allItems={items}
                readinessData={readiness}
                itemActions={getItemActions({
                  item,
                  blockingInfo: blockingMap[itemKey] ?? null,
                  readinessReady: readiness ? readiness.ready : null,
                  pendingSynthesis: readiness?.pendingSynthesis ?? false,
                  agentRunning: activeRunKeys.has(itemKey),
                  hasPendingDecisions: (pendingQuestionsMap.get(itemKey)?.length ?? 0) > 0,
                  hasExecutionHistory: item.status === "completed" || item.status === "failed",
                })}
                attentionReasons={reasons}
                pendingQuestions={pendingQuestionsMap.get(itemKey)}
                isStepperCompleted={completedSteppers.has(itemKey)}
                transitionResult={transitionItems.get(itemKey)}
                onStepperCompleted={(result) => handleStepperCompleted(itemKey, item, result)}
                batchMode={false}
                isSelected={false}
                onToggleSelection={() => {}}
                {...callbacks}
              />
            </button>
          );
        })}
      </div>

      {/* Run modal */}
      <RunBacklogModal
        isOpen={!!runModalTarget}
        onClose={() => setRunModalTarget(undefined)}
        target={runModalTarget}
        onSuccess={() => {
          setRunModalTarget(undefined);
          void fetchBacklog({ force: true });
        }}
      />

      {/* Workshop blocking override confirmation */}
      <ConfirmDialog
        isOpen={!!workshopBlockingConfirm}
        onClose={() => setWorkshopBlockingConfirm(null)}
        onConfirm={confirmWorkshopOverride}
        title="Dependencies Not Ready"
        description={
          workshopBlockingConfirm?.blockingDepKeys.length
            ? `This item is blocked by incomplete dependencies: ${workshopBlockingConfirm.blockingDepKeys.join(", ")}. Do you want to proceed anyway?`
            : "This item has incomplete dependencies. Do you want to proceed anyway?"
        }
        confirmLabel="Override and Proceed"
      />
    </>
  );
}
