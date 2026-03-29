/**
 * CapturesTab - Lists captures with classification status.
 */

import { MessageSquare } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { formatRelativeTime } from "../../../../lib/format-utils";
import { useCaptureStore } from "../../../../stores";
import { matchesSearch } from "./useSidebarSearch";
import type { Capture } from "../../../../types";
import type { CaptureFilters, SortConfig } from "./types";

interface CapturesTabProps {
  searchQuery: string;
  filters: CaptureFilters;
  sort: SortConfig;
  onItemClick: (nodeId: string) => void;
}

const STATUS_COLORS: Record<string, string> = {
  classifying: "bg-blue-500/20 text-blue-300",
  classified: "bg-green-500/20 text-green-300",
  failed: "bg-red-500/20 text-red-300",
};

function applyFilters(items: Capture[], filters: CaptureFilters): Capture[] {
  if (filters.statuses.length === 0) return items;
  return items.filter((c) => filters.statuses.includes(c.status));
}

function applySort(items: Capture[], sort: SortConfig): Capture[] {
  const sorted = [...items];
  const dir = sort.direction === "asc" ? 1 : -1;

  sorted.sort((a, b) => {
    switch (sort.field) {
      case "recency":
        return (new Date(b.created).getTime() - new Date(a.created).getTime()) * dir;
      case "status":
        return a.status.localeCompare(b.status) * dir;
      case "alphabetical":
        return a.text.localeCompare(b.text) * dir;
      default:
        return (new Date(b.created).getTime() - new Date(a.created).getTime()) * dir;
    }
  });

  return sorted;
}

export function CapturesTab({ searchQuery, filters, sort, onItemClick }: CapturesTabProps) {
  const captures = useCaptureStore((s) => s.captures);

  let filtered = applyFilters(captures, filters);
  if (searchQuery) {
    filtered = filtered.filter((c) => matchesSearch(searchQuery, c.text));
  }
  const sorted = applySort(filtered, sort);

  if (sorted.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <MessageSquare className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery || filters.statuses.length > 0 ? "No captures match your filters." : "No captures yet."}</p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      {sorted.map((capture) => (
        <button
          key={capture.id}
          type="button"
          onClick={() => onItemClick(`capture/${capture.id}`)}
          className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
          data-testid="sidebar-capture-item"
        >
          <div className="flex items-start justify-between gap-2">
            <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
              {capture.text.slice(0, 80)}{capture.text.length > 80 ? "..." : ""}
            </p>
            <span className={cn("shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium", STATUS_COLORS[capture.status] ?? "bg-slate-700/60 text-slate-300")}>
              {capture.status}
            </span>
          </div>
          <p className="mt-1 text-[11px] text-slate-500">{formatRelativeTime(capture.created)}</p>
        </button>
      ))}
    </div>
  );
}
