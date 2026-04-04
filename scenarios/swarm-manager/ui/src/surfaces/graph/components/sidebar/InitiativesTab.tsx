/**
 * InitiativesTab - Lists initiatives with rollup counts.
 */

import { useEffect } from "react";
import { FolderKanban } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useInitiativeStore } from "../../../../stores/initiative-store";
import { matchesSearch } from "./useSidebarSearch";
import type { InitiativeWithRollup } from "../../../../types";
import type { InitiativeFilters, SortConfig } from "./types";
import { NoteIndicator } from "../../../../components/ui/note-indicator";

interface InitiativesTabProps {
  searchQuery: string;
  filters: InitiativeFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  active: "bg-cyan-500/20 text-cyan-300",
  completed: "bg-green-500/20 text-green-300",
  archived: "bg-slate-700/40 text-slate-500",
};

function applyFilters(items: InitiativeWithRollup[], filters: InitiativeFilters): InitiativeWithRollup[] {
  if (filters.statuses.length === 0) return items;
  return items.filter((iwr) => filters.statuses.includes(iwr.initiative.status as "active" | "completed" | "archived"));
}

function applySort(items: InitiativeWithRollup[], sort: SortConfig): InitiativeWithRollup[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

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
  const sorted = applySort(filtered, sort);

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
        return (
          <button
            key={initiative.name}
            type="button"
            onClick={() => onItemClick(`initiative/${initiative.name}`)}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-initiative-item"
          >
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
            <div className="mt-1.5 flex flex-wrap gap-2 text-[11px]">
              <span className="text-green-400">{rollup.completed} done</span>
              <span className="text-cyan-400">{rollup.inProgress} active</span>
              {rollup.failed > 0 && <span className="text-red-400">{rollup.failed} failed</span>}
              <span className="text-slate-500">{rollup.pending} pending</span>
            </div>
            <p className="mt-1 text-[11px] text-slate-500">{formatRelativeTime(initiative.updated)}</p>
          </button>
        );
      })}
    </div>
  );
}
