/**
 * BacklogTab - Lists backlog items with kind icons, status badges, and priority.
 */

import { Bug, Cog, FlaskConical, Lightbulb, ListTodo, Wrench } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useBacklogStore } from "../../../../stores";
import { buildBacklogNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import type { BacklogItem, BacklogKind } from "../../../../types";
import type { BacklogFilters, SortConfig } from "./types";

interface BacklogTabProps {
  searchQuery: string;
  filters: BacklogFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
}

const KIND_ICONS: Record<BacklogKind, React.ReactNode> = {
  idea: <Lightbulb className="h-3.5 w-3.5 text-amber-400" />,
  research: <FlaskConical className="h-3.5 w-3.5 text-purple-400" />,
  fix: <Bug className="h-3.5 w-3.5 text-red-400" />,
  execute: <Cog className="h-3.5 w-3.5 text-cyan-400" />,
  chore: <Wrench className="h-3.5 w-3.5 text-slate-400" />,
};

const STATUS_COLORS: Record<string, string> = {
  backlog: "bg-slate-700/60 text-slate-300",
  researching: "bg-blue-500/20 text-blue-300",
  ready: "bg-emerald-500/20 text-emerald-300",
  queued: "bg-amber-500/20 text-amber-300",
  in_progress: "bg-cyan-500/20 text-cyan-300",
  completed: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  archived: "bg-slate-700/40 text-slate-500",
};

function applyFilters(items: BacklogItem[], filters: BacklogFilters): BacklogItem[] {
  return items.filter((item) => {
    if (filters.statuses.length > 0 && !filters.statuses.includes(item.status)) return false;
    if (filters.kinds.length > 0 && !filters.kinds.includes(item.kind)) return false;
    if (filters.priorityMin !== null && item.priority < filters.priorityMin) return false;
    if (filters.priorityMax !== null && item.priority > filters.priorityMax) return false;
    return true;
  });
}

function applySort(items: BacklogItem[], sort: SortConfig): BacklogItem[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

  sorted.sort((a, b) => {
    switch (sort.field) {
      case "priority":
        return (a.priority - b.priority) * dir;
      case "recency":
        return (new Date(b.updated).getTime() - new Date(a.updated).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return (a.title || a.name).localeCompare(b.title || b.name) * dir;
    }
  });

  return sorted;
}

export function BacklogTab({ searchQuery, filters, sort, onItemClick }: BacklogTabProps) {
  const items = useBacklogStore((s) => s.items);

  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((item) =>
      matchesSearch(searchQuery, item.title, item.name, item.description, ...(item.tags ?? [])),
    );
  }
  const sorted = applySort(filtered, sort);

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <ListTodo className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery || hasActiveFilters(filters) ? "No backlog items match your filters." : "No backlog items."}</p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      {sorted.map((item) => {
        const nodeId = buildBacklogNodeId(item.kind, item.name);
        return (
          <button
            key={nodeId}
            type="button"
            onClick={() => onItemClick(nodeId)}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-backlog-item"
          >
            <div className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0">{KIND_ICONS[item.kind]}</span>
              <div className="min-w-0 flex-1">
                <div className="flex items-start justify-between gap-2">
                  <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
                    {item.title || item.name}
                  </p>
                  <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[item.status] ?? "bg-slate-700/60 text-slate-300")}>
                    {item.status.replace(/_/g, " ")}
                  </span>
                </div>
                <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500">
                  <span>P{item.priority}</span>
                  <span>{formatRelativeTime(item.updated)}</span>
                </div>
              </div>
            </div>
          </button>
        );
      })}
    </div>
  );
}

function hasActiveFilters(filters: BacklogFilters): boolean {
  return filters.statuses.length > 0 || filters.kinds.length > 0 || filters.priorityMin !== null || filters.priorityMax !== null;
}
