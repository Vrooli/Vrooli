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
import { OperatingModeCard } from "../../../../components/initiative/operating-mode/operating-mode-card";

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
        Failed to load operating modes: {(error).message}
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
        <OperatingModeCard
          key={mode.mode}
          mode={mode}
          compact
          onClick={() => onItemClick(`operatingMode/${mode.mode}`)}
          data-testid="sidebar-operating-mode-item"
        />
      ))}
    </div>
  );
}
