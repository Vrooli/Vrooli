/**
 * ActiveTab renders the operational sandbox list (creating, active,
 * stopped, error). Source: GET /sandboxes (filtered client-side by
 * tab status set + reducer filters).
 *
 * The Active tab keeps mount-health affordances since these sandboxes
 * still have live overlays.
 */

import { useMemo } from "react";
import { Box, Loader2 } from "lucide-react";

import type { Sandbox } from "../../lib/api";
import { ACTIVE_TAB_STATUSES, type ActiveFilters, type ActiveSortField, type SortConfig } from "./types";
import { sandboxDisplayName } from "../../lib/utils";
import { SandboxItem } from "./SandboxItem";
import { MountHealthBanner } from "./MountHealthBanner";
import { useBannerData } from "./useBannerData";
import { SELECTORS } from "../../consts/selectors";

interface ActiveTabProps {
  sandboxes: Sandbox[];
  selectedId?: string;
  onSelect: (sandbox: Sandbox) => void;
  isLoading: boolean;
  filters: ActiveFilters;
  sort: SortConfig<ActiveSortField>;
  searchQuery: string;
  onRestartSandbox?: (sandboxId: string) => void;
  onRestartUnhealthy?: () => void;
  restartingIds?: Set<string>;
}

function compareActive(a: Sandbox, b: Sandbox, field: ActiveSortField): number {
  switch (field) {
    case "createdAt":
      return new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
    case "lastUsedAt":
      return new Date(a.lastUsedAt).getTime() - new Date(b.lastUsedAt).getTime();
    case "fileCount":
      return (a.fileCount ?? 0) - (b.fileCount ?? 0);
    case "sizeBytes":
      return (a.sizeBytes ?? 0) - (b.sizeBytes ?? 0);
  }
}

function matchesActiveSearch(sb: Sandbox, query: string): boolean {
  if (!query) return true;
  const q = query.toLowerCase();
  if (sandboxDisplayName(sb).toLowerCase().includes(q)) return true;
  if (sb.owner?.toLowerCase().includes(q)) return true;
  if (sb.scopePath?.toLowerCase().includes(q)) return true;
  if (sb.id.toLowerCase().includes(q)) return true;
  return false;
}

export function ActiveTab({
  sandboxes,
  selectedId,
  onSelect,
  isLoading,
  filters,
  sort,
  searchQuery,
  onRestartSandbox,
  onRestartUnhealthy,
  restartingIds,
}: ActiveTabProps) {
  // Restrict to active-tab statuses, then apply filters and sort.
  const visible = useMemo(() => {
    const allowedStatuses =
      filters.statuses.length > 0 ? new Set(filters.statuses) : new Set(ACTIVE_TAB_STATUSES);

    const filtered = sandboxes.filter((sb) => {
      if (!allowedStatuses.has(sb.status)) return false;
      if (filters.owner && !sb.owner?.toLowerCase().includes(filters.owner.toLowerCase())) return false;
      if (
        filters.projectRoot &&
        !sb.projectRoot?.toLowerCase().includes(filters.projectRoot.toLowerCase())
      ) {
        return false;
      }
      if (!matchesActiveSearch(sb, searchQuery)) return false;
      return true;
    });

    const direction = sort.direction === "asc" ? 1 : -1;
    return [...filtered].sort((a, b) => direction * compareActive(a, b, sort.field));
  }, [sandboxes, filters, sort, searchQuery]);

  const activeOnlyForBanner = useMemo(
    () => sandboxes.filter((sb) => (ACTIVE_TAB_STATUSES as readonly string[]).includes(sb.status)),
    [sandboxes],
  );
  const { consolidatedMessages } = useBannerData(activeOnlyForBanner);

  return (
    <div className="px-2 py-2" data-testid="sidebar-active-tab">
      <MountHealthBanner
        sandboxes={activeOnlyForBanner}
        onRestartUnhealthy={onRestartUnhealthy}
        restartingIds={restartingIds}
      />

      {isLoading && visible.length === 0 ? (
        <div className="flex items-center justify-center py-8 text-slate-500">
          <Loader2 className="h-4 w-4 animate-spin mr-2" />
          <span className="text-xs">Loading...</span>
        </div>
      ) : visible.length === 0 ? (
        <div
          className="flex flex-col items-center justify-center py-12 text-center"
          data-testid={SELECTORS.emptyState}
        >
          <Box className="h-10 w-10 text-slate-700 mb-4" />
          <p className="text-sm text-slate-400">
            {sandboxes.length === 0 ? "No sandboxes yet" : "No matches for current filters"}
          </p>
          <p className="text-xs text-slate-500 mt-1">
            {sandboxes.length === 0
              ? "Create a sandbox to get started"
              : "Adjust filters or clear them to see more"}
          </p>
        </div>
      ) : (
        <ul className="divide-y divide-slate-800/30" role="list" data-testid="sidebar-active-list">
          {visible.map((sandbox) => (
            <SandboxItem
              key={sandbox.id}
              sandbox={sandbox}
              selected={selectedId === sandbox.id}
              onSelect={onSelect}
              showMountActions
              consolidatedMessages={consolidatedMessages}
              onRestartSandbox={onRestartSandbox}
              isRestarting={restartingIds?.has(sandbox.id) ?? false}
            />
          ))}
        </ul>
      )}
    </div>
  );
}
