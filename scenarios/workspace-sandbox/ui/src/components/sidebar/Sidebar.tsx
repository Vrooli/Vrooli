/**
 * Sidebar — Active / History tabs for the workspace-sandbox UI.
 *
 * Composition (mirrors swarm-manager's `Sidebar.tsx`):
 *   Header → SearchBar → Tabs → FilterBar → tab content.
 *
 * Active tab is sourced from the existing `/sandboxes` listing
 * (filtered client-side). History tab is sourced from
 * `/sandboxes/history` with server-side filtering and pagination.
 *
 * Selection-on-transition UX: when the currently-selected sandbox
 * transitions to a History status mid-session (e.g. a run gets
 * approved), the parent (`App`) is responsible for switching the
 * active-tab pointer; the Sidebar exposes `setActiveTab` via the
 * controlled state hook.
 */

import { useMemo } from "react";
import { Box, Loader2 } from "lucide-react";

import {
  ACTIVE_STATUSES,
  type DiffArchive,
  type HistoryFilter,
  type Sandbox,
} from "../../lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "../ui/card";
import { ScrollArea } from "../ui/scroll-area";
import { useHistory } from "../../lib/hooks";
import { SELECTORS } from "../../consts/selectors";

import { ActiveTab } from "./ActiveTab";
import { FilterBar } from "./FilterBar";
import { HistoryTab } from "./HistoryTab";
import { SidebarSearchBar } from "./SidebarSearchBar";
import { SidebarTabs } from "./SidebarTabs";
import type { SidebarState } from "./useSidebarState";
import type { SidebarAction } from "./useSidebarState";

export interface SidebarProps {
  /** Active sandboxes from `useSandboxes`. */
  sandboxes: Sandbox[];
  /** ID of the currently-selected sandbox in either tab. */
  selectedId?: string;
  isLoading: boolean;
  /** Selecting an Active sandbox. */
  onSelectActive: (sandbox: Sandbox) => void;
  /** Selecting a History archive. The second arg is a Sandbox-shaped
   *  adapter for downstream components that expect a Sandbox. */
  onSelectHistory: (archive: DiffArchive, asSandbox: Sandbox) => void;
  /** Mount-health restart hooks (Active tab only). */
  onRestartSandbox?: (sandboxId: string) => void;
  onRestartUnhealthy?: () => void;
  restartingIds?: Set<string>;
  /** Sidebar state (controlled by App so selection-on-transition can
   *  programmatically switch tabs). */
  state: SidebarState;
  dispatch: React.Dispatch<SidebarAction>;
}

/** Default page size for the History listing. Page-size > total triggers
 *  a "showing N of M" footer in HistoryTab. */
const HISTORY_PAGE_SIZE = 100;

export function Sidebar({
  sandboxes,
  selectedId,
  isLoading,
  onSelectActive,
  onSelectHistory,
  onRestartSandbox,
  onRestartUnhealthy,
  restartingIds,
  state,
  dispatch,
}: SidebarProps) {
  const { activeTab, searchQuery, filters, sorts } = state;

  // Build the History query from filters + sort + search.
  const historyFilter = useMemo<HistoryFilter>(() => {
    const f = filters.history;
    const s = sorts.history;
    const sortBy: HistoryFilter["sortBy"] =
      s.field === "totalBlobBytes" ? "total_blob_bytes" : "snapshot_at";
    return {
      statuses: f.statuses.length > 0 ? f.statuses : undefined,
      owner: f.owner || undefined,
      projectRoot: f.projectRoot || undefined,
      agentManagerRunId: f.agentManagerRunId || undefined,
      search: (f.search || searchQuery.history) || undefined,
      snapshotAtFrom: f.snapshotAtFrom ? `${f.snapshotAtFrom}T00:00:00Z` : undefined,
      snapshotAtTo: f.snapshotAtTo ? `${f.snapshotAtTo}T23:59:59Z` : undefined,
      sortBy,
      sortDesc: s.direction === "desc",
      limit: HISTORY_PAGE_SIZE,
    };
  }, [filters.history, sorts.history, searchQuery.history]);

  const historyQuery = useHistory(historyFilter, { enabled: activeTab === "history" });

  const archives = historyQuery.data?.archives ?? [];
  const totalHistoryCount = historyQuery.data?.totalCount ?? 0;

  // Active-tab counts: how many sandboxes are still in active statuses.
  const activeCount = useMemo(
    () => sandboxes.filter((sb) => (ACTIVE_STATUSES as readonly string[]).includes(sb.status)).length,
    [sandboxes],
  );

  return (
    <Card className="h-full flex flex-col" data-testid={SELECTORS.sandboxList}>
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3">
        <CardTitle className="flex items-center gap-2">
          <Box className="h-4 w-4 text-slate-500" />
          Sandboxes
        </CardTitle>
        {(isLoading || (activeTab === "history" && historyQuery.isLoading)) && (
          <Loader2 className="h-4 w-4 animate-spin text-slate-500" />
        )}
      </CardHeader>

      <div className="px-3 pb-2">
        <SidebarSearchBar
          value={searchQuery[activeTab]}
          onChange={(query) => dispatch({ type: "SET_SEARCH", tab: activeTab, query })}
          placeholder={
            activeTab === "active" ? "Search active sandboxes..." : "Search archive history..."
          }
        />
      </div>

      <SidebarTabs
        activeTab={activeTab}
        onTabChange={(tab) => dispatch({ type: "SET_TAB", tab })}
        counts={{
          active: activeCount,
          history: activeTab === "history" ? totalHistoryCount : undefined,
        }}
      />

      <FilterBar activeTab={activeTab} filters={filters} sorts={sorts} dispatch={dispatch} />

      <CardContent className="flex-1 p-0 overflow-hidden">
        <ScrollArea className="h-full">
          {activeTab === "active" ? (
            <ActiveTab
              sandboxes={sandboxes}
              selectedId={selectedId}
              onSelect={onSelectActive}
              isLoading={isLoading}
              filters={filters.active}
              sort={sorts.active}
              searchQuery={searchQuery.active}
              onRestartSandbox={onRestartSandbox}
              onRestartUnhealthy={onRestartUnhealthy}
              restartingIds={restartingIds}
            />
          ) : (
            <HistoryTab
              archives={archives}
              selectedId={selectedId}
              onSelect={onSelectHistory}
              isLoading={historyQuery.isLoading}
              isError={historyQuery.isError}
              errorMessage={historyQuery.error?.message}
              filters={filters.history}
              sort={sorts.history}
              totalCount={totalHistoryCount}
            />
          )}
        </ScrollArea>
      </CardContent>
    </Card>
  );
}
