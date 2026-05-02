/**
 * OperatingModesTab — Lists registered operating modes with usage counts.
 *
 * Sources data from the catalog endpoint (which already includes per-mode
 * usage_count) so the sidebar issues a single network call regardless of
 * how many modes are registered.
 */

import { useQuery } from "@tanstack/react-query";
import { Layers } from "lucide-react";
import { matchesSearch } from "./useSidebarSearch";
import { initiativeModeService } from "../../../../services";
import type { OperatingModeCatalogEntry } from "../../../../types/operating-mode";

interface OperatingModesTabProps {
  searchQuery: string;
  onItemClick: (nodeId: string) => void;
}

function LoadingSkeleton() {
  return (
    <div className="space-y-1.5">
      {[1, 2, 3].map((i) => (
        <div key={i} className="animate-pulse rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5">
          <div className="h-4 w-3/4 rounded bg-slate-800" />
          <div className="mt-2 h-3 w-1/2 rounded bg-slate-800" />
        </div>
      ))}
    </div>
  );
}

export function OperatingModesTab({ searchQuery, onItemClick }: OperatingModesTabProps) {
  const { data, isLoading, error } = useQuery({
    queryKey: ["operating-modes", "catalog"],
    queryFn: () => initiativeModeService.catalog(),
  });

  if (isLoading) return <LoadingSkeleton />;
  if (error) {
    return (
      <div className="px-2 py-4 text-sm text-red-400">
        Failed to load operating modes: {(error as Error).message}
      </div>
    );
  }

  const modes: OperatingModeCatalogEntry[] = data?.modes ?? [];
  const filtered = searchQuery
    ? modes.filter((m) => matchesSearch(searchQuery, m.label, m.mode, m.description ?? ""))
    : modes;

  if (filtered.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-slate-500">
        <Layers className="mb-2 h-8 w-8" />
        <p className="text-sm">{searchQuery ? "No modes match your search." : "No operating modes registered."}</p>
      </div>
    );
  }

  return (
    <div className="space-y-1.5">
      {filtered.map((mode) => (
        <button
          key={mode.mode}
          type="button"
          onClick={() => onItemClick(`operatingMode/${mode.mode}`)}
          className="w-full rounded-lg border border-slate-800/80 bg-slate-900/50 p-2.5 text-left transition-colors hover:border-slate-700/80 hover:bg-slate-800/60"
          data-testid="sidebar-operating-mode-item"
        >
          <div className="flex items-start justify-between gap-2">
            <p className="line-clamp-2 text-[13px] font-medium leading-snug text-slate-100">
              {mode.label}
            </p>
            <span className="shrink-0 rounded-full bg-slate-700/60 px-2 py-0.5 text-[10px] font-medium text-slate-300">
              {mode.usageCount} init.
            </span>
          </div>
          {mode.description && (
            <p className="mt-1 line-clamp-2 text-[11px] text-slate-400">{mode.description}</p>
          )}
          <p className="mt-1 text-[11px] text-slate-500">
            {mode.scopeKind} · {mode.runStrategy}
            {mode.default ? " · default" : ""}
          </p>
        </button>
      ))}
    </div>
  );
}
