/**
 * InitiativesTab - Lists initiatives with rollup counts.
 */

import { useEffect } from "react";
import { Archive, FolderKanban } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useInitiativeStore } from "../../../../stores/initiative-store";
import { matchesSearch } from "./useSidebarSearch";
import type { InitiativeWithRollup } from "../../../../types";
import type { InitiativeFilters, SortConfig } from "./types";
import { NoteIndicator } from "../../../../components/ui/note-indicator";
import { RollupProgressBar, rollupTotal } from "../../../../components/ui/rollup-progress-bar";
import {
  computeEffectivePriority,
  computeUnblockingMap,
  dependencyAwareSort,
  type DepthItem,
} from "../../../../lib/dependency-sort";

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
}

const STATUS_COLORS: Record<string, string> = {
  active: "bg-cyan-500/20 text-cyan-300",
  completed: "bg-green-500/20 text-green-300",
};

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

export function InitiativesTab({ searchQuery, filters, sort, onItemClick }: InitiativesTabProps) {
  const items = useInitiativeStore((s) => s.items);
  const status = useInitiativeStore((s) => s.status);
  const fetchInitiatives = useInitiativeStore((s) => s.fetchInitiatives);

  useEffect(() => {
    void fetchInitiatives();
  }, [fetchInitiatives]);

  if (status === "loading") {
    return <LoadingSkeleton />;
  }

  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((iwr) =>
      matchesSearch(searchQuery, iwr.initiative.title, iwr.initiative.name, iwr.initiative.description),
    );
  }
  const sorted = applySort(filtered, sort, items);

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <FolderKanban className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery || filters.statuses.length > 0 ? "No initiatives match your filters." : "No initiatives yet."}</p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      {sorted.map((iwr) => {
        const { initiative, rollup } = iwr;
        const deps = (initiative as { dependsOn?: string[] }).dependsOn ?? [];
        return (
          <button
            key={initiative.name}
            type="button"
            onClick={() => onItemClick(`initiative/${initiative.name}`)}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-initiative-item"
          >
            {(initiative as { archivedAt?: string }).archivedAt != null && (
              <div className="mb-1.5 flex items-center gap-1.5 rounded border border-amber-500/20 bg-amber-500/5 px-2 py-1 text-[11px] text-amber-400/80">
                <Archive className="h-3 w-3 shrink-0" />
                Archived
              </div>
            )}
            <div className="flex items-start justify-between gap-2">
              <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
                {initiative.title || initiative.name}
              </p>
              <div className="flex shrink-0 items-center gap-1.5">
                <NoteIndicator note={initiative.note} />
                <span className={cn("rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[initiative.status] ?? "bg-slate-700/60 text-slate-300")}>
                  {initiative.status}
                </span>
              </div>
            </div>
            {rollupTotal(rollup) > 0 && (
              <>
                <RollupProgressBar rollup={rollup} barHeight="h-1.5" className="mt-2" />
                <div className="mt-1 flex flex-wrap gap-2 text-[11px]">
                  <span className="text-emerald-400">{rollup.completed} done</span>
                  <span className="text-purple-400">{rollup.inProgress} active</span>
                  {rollup.failed > 0 && <span className="text-red-400">{rollup.failed} failed</span>}
                  <span className="text-slate-500">{rollup.pending} pending</span>
                </div>
              </>
            )}
            {deps.length > 0 && (
              <p className="mt-1 text-[11px] text-slate-400">
                <span className="text-slate-500">Depends on:</span> {deps.join(", ")}
              </p>
            )}
            <p className="mt-1 text-[11px] text-slate-500">{formatRelativeTime(initiative.updated)}</p>
          </button>
        );
      })}
    </div>
  );
}
