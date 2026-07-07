/**
 * InitiativesTab - Lists initiatives with rollup counts.
 */

import { memo, useEffect, useState } from "react";
import { Archive, Loader2 } from "lucide-react";
import { SIDEBAR_TAB_ICONS } from "../../../../types/constants";
import { useInitiativeStore } from "../../../../stores/initiative-store";
import { matchesSearch } from "./useSidebarSearch";
import type { InitiativeWithRollup } from "../../../../types";
import type { InitiativeFilters, SortConfig } from "./types";
import { InitiativeSummaryCard } from "../../../../components/initiative/initiative-summary-card";
import { ConfirmDialog } from "../../../../components/ui/confirm-dialog";
import { initiativeService } from "../../../../services";
import {
  computeEffectivePriority,
  computeUnblockingMap,
  dependencyAwareSort,
  type DepthItem,
} from "../../../../lib/dependency-sort";
import { SidebarEmptyState } from "./SidebarEmptyState";
import { runBulkAction, summarizeBulkOutcomes, type BulkOutcome } from "./bulk-actions";

// Constant namespace so initiative keys never collide with backlog keys
// in shared dependency-sort computations.
const INITIATIVE_KIND = "initiative";

function toDepthItem(iwr: InitiativeWithRollup): DepthItem {
  const init = iwr.initiative as InitiativeWithRollup["initiative"] & {
    priority?: number;
    dependsOn?: string[];
  };
  // Deps on disk are bare names; prefix with our synthetic kind for sort keys.
  const deps = (init.dependsOn ?? []).map((n) => `${INITIATIVE_KIND}/${n}`);
  return {
    kind: INITIATIVE_KIND,
    name: init.name,
    status: init.status,
    dependsOn: deps,
    archivedAt: init.archivedAt ?? null,
  };
}

interface InitiativesTabProps {
  searchQuery: string;
  filters: InitiativeFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
  onClearSearch?: () => void;
  selectionMode?: boolean;
  selectedIds?: Set<string>;
  onToggleSelection?: (id: string) => void;
  onVisibleIdsChange?: (ids: string[]) => void;
}

function applyFilters(items: InitiativeWithRollup[], filters: InitiativeFilters): InitiativeWithRollup[] {
  return items.filter((iwr) => {
    // Hide archived initiatives unless showArchived is on
    const initiative = iwr.initiative as { archivedAt?: string; status: string };
    if (initiative.archivedAt != null && !filters.showArchived) return false;
    if (filters.statuses.length > 0 && !filters.statuses.includes(iwr.initiative.status as "active" | "completed")) return false;
    return true;
  });
}

function applySort(
  items: InitiativeWithRollup[],
  sort: SortConfig,
  allItems: InitiativeWithRollup[],
): InitiativeWithRollup[] {
  const dir = sort.direction === "asc" ? 1 : -1;

  if (sort.field === "priority") {
    const allDepth = allItems.map(toDepthItem);
    const unblocking = computeUnblockingMap(allDepth);
    const compare = (a: InitiativeWithRollup, b: InitiativeWithRollup): number => {
      const ap = (a.initiative as { priority?: number }).priority ?? 0;
      const bp = (b.initiative as { priority?: number }).priority ?? 0;
      const effA = computeEffectivePriority(ap === 0 ? 99 : ap, unblocking.get(`${INITIATIVE_KIND}/${a.initiative.name}`) ?? 0);
      const effB = computeEffectivePriority(bp === 0 ? 99 : bp, unblocking.get(`${INITIATIVE_KIND}/${b.initiative.name}`) ?? 0);
      return (effA - effB) * dir;
    };
    // dependencyAwareSort respects topological depth then falls back to compare.
    const wrapped = items.map((iwr) => ({
      iwr,
      kind: INITIATIVE_KIND,
      name: iwr.initiative.name,
      dependsOn: toDepthItem(iwr).dependsOn,
    }));
    const sortedWrap = dependencyAwareSort(
      wrapped,
      (a, b) => compare(a.iwr, b.iwr),
      allDepth,
    );
    return sortedWrap.map((w) => w.iwr);
  }

  const sorted = [...items];
  sorted.sort((a, b) => {
    switch (sort.field) {
      case "recency":
        return (new Date(b.initiative.updated).getTime() - new Date(a.initiative.updated).getTime()) * dir;
      case "status":
        return a.initiative.status.localeCompare(b.initiative.status) * dir;
      case "alphabetical":
        return (a.initiative.title || a.initiative.name).localeCompare(b.initiative.title || b.initiative.name) * dir;
      default:
        return (a.initiative.title || a.initiative.name).localeCompare(b.initiative.title || b.initiative.name) * dir;
    }
  });

  return sorted;
}

function LoadingSkeleton() {
  return (
    <div className="space-y-1.5">
      {[1, 2, 3].map((i) => (
        <div key={i} className="animate-pulse rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5">
          <div className="h-4 w-3/4 rounded bg-slate-800" />
          <div className="mt-2 flex gap-3">
            <div className="h-3 w-10 rounded bg-slate-800" />
            <div className="h-3 w-10 rounded bg-slate-800" />
            <div className="h-3 w-10 rounded bg-slate-800" />
          </div>
        </div>
      ))}
    </div>
  );
}

function initiativeSelectionId(iwr: InitiativeWithRollup): string {
  return `initiative:${iwr.initiative.name}`;
}

function InitiativesTabImpl({
  searchQuery,
  filters,
  sort,
  onItemClick,
  onClearSearch,
  selectionMode = false,
  selectedIds = new Set<string>(),
  onToggleSelection,
  onVisibleIdsChange,
}: InitiativesTabProps) {
  const items = useInitiativeStore((s) => s.items);
  const status = useInitiativeStore((s) => s.status);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);

  useEffect(() => {
    void fetchInitiatives();
  }, [fetchInitiatives]);

  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((iwr) =>
      matchesSearch(searchQuery, iwr.initiative.title, iwr.initiative.name, iwr.initiative.description),
    );
  }
  const sorted = applySort(filtered, sort, items);
  useEffect(() => {
    onVisibleIdsChange?.(sorted.map(initiativeSelectionId));
  }, [onVisibleIdsChange, sorted]);

  if (status === "loading") {
    return <LoadingSkeleton />;
  }

  if (sorted.length === 0) {
    const filtersActive = filters.statuses.length > 0;
    const title = filtersActive ? "No initiatives match your filters." : "No initiatives yet.";
    return (
      <SidebarEmptyState
        icon={SIDEBAR_TAB_ICONS.initiatives}
        title={title}
        hint={filtersActive ? undefined : "Group related backlog work into initiatives to track progress together."}
        query={searchQuery}
        onClearSearch={onClearSearch}
      />
    );
  }

  return (
    <div className="space-y-1.5">
      {selectionMode && <InitiativeBulkActions selectedInitiatives={sorted.filter((iwr) => selectedIds.has(initiativeSelectionId(iwr)))} />}
      {sorted.map((iwr) => {
        const { initiative } = iwr;
        const selectionId = initiativeSelectionId(iwr);
        return (
          <InitiativeSummaryCard
            key={initiative.name}
            item={iwr}
            onOpen={() => onItemClick(`initiative/${initiative.name}`)}
            batchMode={selectionMode}
            batchSelected={selectedIds.has(selectionId)}
            onBatchToggle={() => onToggleSelection?.(selectionId)}
          />
        );
      })}
    </div>
  );
}

export const InitiativesTab = memo(InitiativesTabImpl);

function InitiativeBulkActions({ selectedInitiatives }: { selectedInitiatives: InitiativeWithRollup[] }) {
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);
  const [action, setAction] = useState<"archive" | "unarchive">("archive");
  const [confirm, setConfirm] = useState<"archive" | "unarchive" | null>(null);
  const [running, setRunning] = useState(false);
  const [summary, setSummary] = useState<string | null>(null);
  const [outcomes, setOutcomes] = useState<BulkOutcome[]>([]);

  const eligible = selectedInitiatives.filter((iwr) => {
    const archived = iwr.initiative.archivedAt != null;
    return action === "archive" ? !archived : archived;
  });

  const execute = async () => {
    setRunning(true);
    setSummary(null);
    setOutcomes([]);
    try {
      const next = await runBulkAction(eligible, {
        getId: initiativeSelectionId,
        getLabel: (iwr) => iwr.initiative.title || iwr.initiative.name,
        run: (iwr) => action === "archive"
          ? initiativeService.archiveItem(iwr.initiative.name)
          : initiativeService.unarchiveItem(iwr.initiative.name),
      });
      setOutcomes(next);
      setSummary(summarizeBulkOutcomes(next));
      await fetchInitiatives({ force: true });
    } finally {
      setRunning(false);
      setConfirm(null);
    }
  };

  const failed = outcomes.filter((outcome) => outcome.status === "failed");

  return (
    <div className="rounded-lg border border-slate-800 bg-slate-900/70 p-2" data-testid="sidebar-initiative-bulk-actions">
      <div className="flex flex-wrap items-center gap-2">
        <select value={action} onChange={(event) => setAction(event.target.value as "archive" | "unarchive")} className="h-8 rounded border border-slate-700 bg-slate-950 px-2 text-xs text-slate-200" aria-label="Initiative bulk action">
          <option value="archive">Archive selected</option>
          <option value="unarchive">Unarchive selected</option>
        </select>
        <button
          type="button"
          disabled={selectedInitiatives.length === 0 || eligible.length === 0 || running}
          onClick={() => setConfirm(action)}
          className="inline-flex h-8 items-center gap-1.5 rounded border border-cyan-500/40 bg-cyan-500/10 px-2 text-xs font-medium text-cyan-200 hover:bg-cyan-500/20 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {running ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Archive className="h-3.5 w-3.5" />}
          Apply
        </button>
      </div>
      <div className="mt-1.5 text-[11px] text-slate-500">{eligible.length} eligible{summary ? ` - ${summary}` : ""}</div>
      {failed.length > 0 && <div className="mt-1 text-[11px] text-red-300">{failed.map((outcome) => <div key={outcome.id}>{outcome.label}: {outcome.message}</div>)}</div>}
      <ConfirmDialog
        isOpen={confirm !== null}
        onClose={() => setConfirm(null)}
        onConfirm={() => void execute()}
        title={confirm === "unarchive" ? "Unarchive selected initiatives" : "Archive selected initiatives"}
        description={`${confirm === "unarchive" ? "Unarchive" : "Archive"} ${eligible.length} selected initiative${eligible.length === 1 ? "" : "s"}?`}
        confirmLabel={confirm === "unarchive" ? "Unarchive selected" : "Archive selected"}
        isLoading={running}
      />
    </div>
  );
}
