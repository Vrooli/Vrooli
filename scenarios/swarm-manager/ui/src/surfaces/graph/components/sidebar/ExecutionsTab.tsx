/**
 * ExecutionsTab - Lists execution records with status, mode, and timing.
 */

import { Play } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useExecutionStore } from "../../../../stores";
import { buildExecutionNodeId } from "../../lib/node-id-parser";
import { matchesSearch } from "./useSidebarSearch";
import type { ExecutionRecord } from "../../../../types";
import type { ExecutionFilters, SortConfig } from "./types";

interface ExecutionsTabProps {
  searchQuery: string;
  filters: ExecutionFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  pending: "bg-slate-700/60 text-slate-300",
  starting: "bg-blue-500/20 text-blue-300",
  running: "bg-cyan-500/20 text-cyan-300",
  needs_review: "bg-amber-500/20 text-amber-300",
  validating: "bg-blue-500/20 text-blue-300",
  needs_fixup: "bg-amber-500/20 text-amber-300",
  completed: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
  canceled: "bg-slate-700/40 text-slate-500",
};

const MODE_LABELS: Record<string, string> = {
  manual: "Manual",
  yolo: "YOLO",
};

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

export function ExecutionsTab({ searchQuery, filters, sort, onItemClick }: ExecutionsTabProps) {
  const items = useExecutionStore((s) => s.items);

  let filtered = applyFilters(items, filters);
  if (searchQuery) {
    filtered = filtered.filter((item) =>
      matchesSearch(searchQuery, item.backlogName, item.status, item.mode),
    );
  }
  const sorted = applySort(filtered, sort);

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Play className="mb-2 h-8 w-8" />
        <p className="text-sm">
          {searchQuery || filters.statuses.length > 0 || filters.modes.length > 0
            ? "No executions match your filters."
            : "No executions yet."}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      {sorted.map((item) => {
        const nodeId = buildExecutionNodeId(item.executionId);
        return (
          <button
            key={item.executionId}
            type="button"
            onClick={() => onItemClick(nodeId)}
            className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
            data-testid="sidebar-execution-item"
          >
            <div className="flex items-start justify-between gap-2">
              <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
                {item.backlogName}
              </p>
              <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[item.status] ?? "bg-slate-700/60 text-slate-300")}>
                {item.status.replace(/_/g, " ")}
              </span>
            </div>
            <div className="mt-1 flex items-center gap-2 text-[11px] text-slate-500">
              <span>{MODE_LABELS[item.mode] ?? item.mode}</span>
              <span>{formatRelativeTime(item.createdAt)}</span>
            </div>
          </button>
        );
      })}
    </div>
  );
}
