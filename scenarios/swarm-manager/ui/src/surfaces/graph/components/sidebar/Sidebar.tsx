/**
 * Sidebar - Multi-tab navigation surface for the graph workspace.
 *
 * Composes: SidebarHeader → SearchBar → Tabs → FilterBar → Tab Content.
 * Manages sidebar UI state via persisted reducer state.
 */

import type { HTMLAttributes } from "react";
import { cn } from "../../../../lib/utils";
import { SearchBar } from "../../../../components/ui/search-bar";
import { useAISearchStatus } from "../../../../lib/ai-search";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { useSidebarState } from "./useSidebarState";
import { useDebouncedValue } from "./useSidebarSearch";
import { SidebarHeader } from "./SidebarHeader";
import { SidebarTabs } from "./SidebarTabs";
import { SearchModeToggle } from "./SearchModeToggle";
import { AISearchResults } from "./AISearchResults";
import { FilterBar } from "./FilterBar";
import { ActivityTab } from "./ActivityTab";
import { BacklogTab } from "./BacklogTab";
import { CapturesTab } from "./CapturesTab";
import { InitiativesTab } from "./InitiativesTab";
import { OperatingModesTab } from "./OperatingModesTab";
import { ExecutionsTab } from "./ExecutionsTab";
import { SessionsTab } from "./SessionsTab";
import type { FeedItem } from "../../../../lib/feed";

interface SidebarProps {
  feed: FeedItem[];
  onItemClick: (nodeId: string) => void;
  onSettingsOpen: () => void;
  onViewActivity: (activityId: string) => void;
  onViewBacklog: (nodeId: string) => void;
  onGoHome: () => void;
  onOpenCommandPost?: () => void;
  onOpenAgentSession?: (sessionId: string) => void;
  desktopWidth?: number;
  resizeHandleProps?: HTMLAttributes<HTMLDivElement>;
}

export function Sidebar({
  feed,
  onItemClick,
  onSettingsOpen,
  onViewActivity,
  onViewBacklog,
  onGoHome,
  onOpenCommandPost,
  onOpenAgentSession,
  desktopWidth,
  resizeHandleProps,
}: SidebarProps) {
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const [state, dispatch] = useSidebarState();
  const debouncedSearch = useDebouncedValue(state.searchQuery);
  const aiSearchStatus = useAISearchStatus();
  const aiAvailable = aiSearchStatus.status?.available ?? false;
  const aiMode = state.searchMode === "ai" && aiAvailable;

  if (sidebarCollapsed) {
    return null;
  }

  const { activeTab } = state;

  return (
    <>
      {/* Mobile backdrop */}
      <button
        type="button"
        className="fixed inset-0 z-50 bg-black/40 backdrop-blur-[2px] md:hidden"
        aria-label="Close sidebar"
        onClick={toggleSidebar}
      />

      <aside
        className={cn(
          "fixed inset-0 z-50 flex w-full flex-col border-r border-slate-200/20 bg-slate-950 shadow-2xl",
          "md:relative md:z-auto md:w-auto md:shrink-0 md:shadow-none",
        )}
        style={{
          touchAction: "manipulation",
          width: desktopWidth,
        }}
        data-testid="sidebar"
      >
        {/* Header */}
        <SidebarHeader
          onSettingsOpen={onSettingsOpen}
          onCollapse={toggleSidebar}
          onViewActivity={onViewActivity}
          onViewBacklog={onViewBacklog}
          onGoHome={onGoHome}
          onOpenCommandPost={onOpenCommandPost}
        />

        {/* Search */}
        <div className="flex shrink-0 flex-col gap-2 border-b border-slate-200/20 px-3 py-2">
          <SearchBar
            placeholder={aiMode ? "Semantic search..." : "Search..."}
            value={state.searchQuery}
            onChange={(e) => dispatch({ type: "SET_SEARCH", query: e.target.value })}
            widthClass="w-full"
            className="h-8 text-[16px] md:text-sm"
            data-testid="sidebar-search"
          />
          <SearchModeToggle
            mode={state.searchMode}
            onChange={(mode) => dispatch({ type: "SET_SEARCH_MODE", mode })}
            aiAvailable={aiAvailable}
            unavailableReason={aiSearchStatus.status?.message ?? aiSearchStatus.error ?? undefined}
          />
        </div>

        {/* Tabs — hidden in AI mode since AI search spans all entities */}
        {!aiMode && (
          <SidebarTabs
            activeTab={activeTab}
            onTabChange={(tab) => dispatch({ type: "SET_TAB", tab })}
          />
        )}

        {/* Filters — hidden in AI mode */}
        {!aiMode && (
          <FilterBar
            activeTab={activeTab}
            backlogFilters={state.filters.backlog}
            captureFilters={state.filters.captures}
            initiativeFilters={state.filters.initiatives}
            executionFilters={state.filters.executions}
            sessionFilters={state.filters.sessions}
            sort={state.sorts[activeTab]}
            dispatch={dispatch}
          />
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-2.5">
          {aiMode ? (
            <AISearchResults query={debouncedSearch} onItemClick={onItemClick} />
          ) : (
            <>
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
              {activeTab === "operatingModes" && (
                <OperatingModesTab
                  searchQuery={debouncedSearch}
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
              {activeTab === "sessions" && (
                <SessionsTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.sessions}
                  sort={state.sorts.sessions}
                  onOpenSession={onOpenAgentSession}
                />
              )}
            </>
          )}
        </div>
      </aside>
      {resizeHandleProps && (
        <div
          {...resizeHandleProps}
          className={cn(
            "hidden w-1.5 shrink-0 cursor-col-resize border-r border-slate-800 bg-slate-950 transition-colors hover:bg-cyan-500/20 md:block",
            resizeHandleProps.className,
          )}
          data-testid="sidebar-resize-handle"
        />
      )}
    </>
  );
}
