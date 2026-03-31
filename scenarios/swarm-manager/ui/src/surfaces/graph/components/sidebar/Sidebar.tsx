/**
 * Sidebar - Multi-tab navigation surface for the graph workspace.
 *
 * Composes: SidebarHeader → SearchBar → Tabs → FilterBar → Tab Content.
 * Manages sidebar UI state via useReducer with URL sync.
 */

import { PanelLeft } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { SearchBar } from "../../../../components/ui/search-bar";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { useSidebarState } from "./useSidebarState";
import { useDebouncedValue } from "./useSidebarSearch";
import { useRestoreFromUrl, useSyncToUrl } from "./useSidebarUrlSync";
import { SidebarHeader } from "./SidebarHeader";
import { SidebarTabs } from "./SidebarTabs";
import { FilterBar } from "./FilterBar";
import { ActivityTab } from "./ActivityTab";
import { BacklogTab } from "./BacklogTab";
import { CapturesTab } from "./CapturesTab";
import { InitiativesTab } from "./InitiativesTab";
import { ExecutionsTab } from "./ExecutionsTab";
import type { FeedItem } from "../../../../lib/feed";

interface SidebarProps {
  feed: FeedItem[];
  onItemClick: (nodeId: string) => void;
  onSettingsOpen: () => void;
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
}

export function Sidebar({ feed, onItemClick, onSettingsOpen, onViewActivity, onViewBacklog }: SidebarProps) {
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const [state, dispatch] = useSidebarState();
  const debouncedSearch = useDebouncedValue(state.searchQuery);

  useRestoreFromUrl(dispatch);
  useSyncToUrl(state);

  if (sidebarCollapsed) {
    return (
      <button
        type="button"
        onClick={toggleSidebar}
        className="fixed left-3 top-[3.25rem] z-20 rounded-lg border border-slate-700/80 bg-slate-900/90 p-2 text-slate-400 shadow-lg backdrop-blur-sm transition-colors hover:bg-slate-800/70 hover:text-slate-200"
        aria-label="Open sidebar"
        data-testid="sidebar-toggle-open"
      >
        <PanelLeft className="h-4 w-4" />
      </button>
    );
  }

  const { activeTab } = state;

  return (
    <>
      {/* Mobile backdrop */}
      <button
        type="button"
        className="fixed inset-0 z-20 bg-black/40 backdrop-blur-[2px] md:hidden"
        aria-label="Close sidebar"
        onClick={toggleSidebar}
      />

      <aside
        className={cn(
          "fixed inset-0 z-40 flex w-full flex-col border-r border-slate-200/20 bg-slate-950 shadow-2xl",
          "md:relative md:w-80 md:shrink-0 md:shadow-none",
        )}
        style={{ touchAction: "manipulation" }}
        data-testid="sidebar"
      >
        {/* Header */}
        <SidebarHeader
          onSettingsOpen={onSettingsOpen}
          onCollapse={toggleSidebar}
          onViewActivity={onViewActivity}
          onViewBacklog={onViewBacklog}
        />

        {/* Search */}
        <div className="flex shrink-0 items-center border-b border-slate-200/20 px-3 py-2">
          <SearchBar
            placeholder="Search..."
            value={state.searchQuery}
            onChange={(e) => dispatch({ type: "SET_SEARCH", query: e.target.value })}
            widthClass="w-full"
            className="h-8 text-[16px] md:text-sm"
            data-testid="sidebar-search"
          />
        </div>

        {/* Tabs */}
        <SidebarTabs
          activeTab={activeTab}
          onTabChange={(tab) => dispatch({ type: "SET_TAB", tab })}
        />

        {/* Filters */}
        <FilterBar
          activeTab={activeTab}
          backlogFilters={state.filters.backlog}
          captureFilters={state.filters.captures}
          initiativeFilters={state.filters.initiatives}
          executionFilters={state.filters.executions}
          sort={state.sorts[activeTab]}
          dispatch={dispatch}
        />

        {/* Tab Content */}
        <div className="flex-1 overflow-y-auto p-2.5">
          {activeTab === "activity" && (
            <ActivityTab feed={feed} searchQuery={debouncedSearch} onItemClick={onItemClick} />
          )}
          {activeTab === "backlog" && (
            <BacklogTab
              searchQuery={debouncedSearch}
              filters={state.filters.backlog}
              sort={state.sorts.backlog}
              onItemClick={onItemClick}
            />
          )}
          {activeTab === "captures" && (
            <CapturesTab
              searchQuery={debouncedSearch}
              filters={state.filters.captures}
              sort={state.sorts.captures}
              onItemClick={onItemClick}
            />
          )}
          {activeTab === "initiatives" && (
            <InitiativesTab
              searchQuery={debouncedSearch}
              filters={state.filters.initiatives}
              sort={state.sorts.initiatives}
              onItemClick={onItemClick}
            />
          )}
          {activeTab === "executions" && (
            <ExecutionsTab
              searchQuery={debouncedSearch}
              filters={state.filters.executions}
              sort={state.sorts.executions}
              onItemClick={onItemClick}
            />
          )}
        </div>
      </aside>
    </>
  );
}
