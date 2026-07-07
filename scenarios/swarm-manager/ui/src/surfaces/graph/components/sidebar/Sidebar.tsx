/**
 * Sidebar - Multi-tab navigation surface for the graph workspace.
 *
 * Composes: SidebarHeader → SearchBar → Tabs → FilterBar → Tab Content.
 * Manages sidebar UI state via persisted reducer state.
 */

import { useCallback, type ChangeEvent, type HTMLAttributes, type Ref } from "react";
import { cn } from "../../../../lib/utils";
import { SearchBar } from "../../../../components/ui/search-bar";
import { useAISearchStatus } from "../../../../lib/ai-search";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { useSidebarState } from "./useSidebarState";
import type { SearchMode } from "./useSidebarState";
import type { SidebarTab } from "./types";
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
import { GoalsTab } from "./GoalsTab";
import { OperatingModesTab } from "./OperatingModesTab";
import { ExecutionsTab } from "./ExecutionsTab";
import { SessionsTab } from "./SessionsTab";
import type { FeedItem } from "../../../../lib/feed";
import { useSidebarSelection } from "./useSidebarSelection";

interface SidebarProps {
  feed: FeedItem[];
  onItemClick: (nodeId: string) => void;
  onSettingsOpen: () => void;
  onGoHome: () => void;
  onOpenCommandPost?: () => void;
  onOpenAgentSession?: (sessionId: string) => void;
  desktopWidth?: number;
  resizeHandleProps?: HTMLAttributes<HTMLDivElement>;
  /** Optional ref to the <aside> element. Used by parents to imperatively
   *  drive the width during drag without re-rendering React. */
  asideRef?: Ref<HTMLElement>;
}

export function Sidebar({
  feed,
  onItemClick,
  onSettingsOpen,
  onGoHome,
  onOpenCommandPost,
  onOpenAgentSession,
  desktopWidth,
  resizeHandleProps,
  asideRef,
}: SidebarProps) {
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);

  const [state, dispatch] = useSidebarState();
  const debouncedSearch = useDebouncedValue(state.searchQuery);
  const aiSearchStatus = useAISearchStatus();
  const aiAvailable = aiSearchStatus.status?.available ?? false;
  const aiMode = state.searchMode === "ai" && aiAvailable;
  const clearSearch = useCallback(() => dispatch({ type: "SET_SEARCH", query: "" }), [dispatch]);
  const handleSearchChange = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => dispatch({ type: "SET_SEARCH", query: e.target.value }),
    [dispatch],
  );
  const handleSearchModeChange = useCallback(
    (mode: SearchMode) => dispatch({ type: "SET_SEARCH_MODE", mode }),
    [dispatch],
  );
  const handleTabChange = useCallback(
    (tab: SidebarTab) => dispatch({ type: "SET_TAB", tab }),
    [dispatch],
  );
  const { activeTab } = state;
  const selection = useSidebarSelection(activeTab);
  const showSelectionControls = !aiMode && selection.selectable;

  if (sidebarCollapsed) {
    return null;
  }

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
        ref={asideRef}
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
          onGoHome={onGoHome}
          onOpenCommandPost={onOpenCommandPost}
        />

        {/* Search */}
        <div className="flex shrink-0 flex-col gap-2 border-b border-slate-200/20 px-3 py-2">
          <SearchBar
            placeholder={aiMode ? "Semantic search..." : "Search..."}
            value={state.searchQuery}
            onChange={handleSearchChange}
            widthClass="w-full"
            className="h-8 text-[16px] md:text-sm"
            data-testid="sidebar-search"
          />
          <div className="flex min-h-7 flex-wrap items-center justify-between gap-2" data-testid="sidebar-search-control-row">
            <SearchModeToggle
              mode={state.searchMode}
              onChange={handleSearchModeChange}
              aiAvailable={aiAvailable}
              unavailableReason={aiSearchStatus.status?.message ?? aiSearchStatus.error ?? undefined}
            />
            {showSelectionControls && (
              <div className="flex min-w-0 flex-wrap items-center justify-end gap-1.5 text-xs" data-testid="sidebar-selection-controls">
                {selection.selectionMode ? (
                  <>
                    <span className="whitespace-nowrap text-slate-400" data-testid="sidebar-selected-count">
                      {selection.selectedCount} selected
                    </span>
                    <button
                      type="button"
                      onClick={selection.selectAllVisible}
                      disabled={selection.visibleIds.length === 0}
                      className="rounded border border-slate-700/60 px-2 py-1 text-slate-300 hover:border-slate-500 hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
                      data-testid="sidebar-select-all-visible"
                    >
                      Select all visible
                    </button>
                    <button
                      type="button"
                      onClick={selection.cancelSelection}
                      className="rounded border border-slate-700/60 px-2 py-1 text-slate-300 hover:border-slate-500 hover:bg-slate-800"
                      data-testid="sidebar-cancel-selection"
                    >
                      Cancel
                    </button>
                  </>
                ) : (
                  <button
                    type="button"
                    onClick={selection.toggleMode}
                    className="rounded border border-slate-700/60 px-2 py-1 text-slate-300 hover:border-slate-500 hover:bg-slate-800"
                    data-testid="sidebar-select-mode"
                  >
                    Select
                  </button>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Tabs — hidden in AI mode since AI search spans all entities */}
        {!aiMode && (
          <SidebarTabs
            activeTab={activeTab}
            onTabChange={handleTabChange}
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
                <ActivityTab
                  feed={feed}
                  searchQuery={debouncedSearch}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                />
              )}
              {activeTab === "backlog" && (
                <BacklogTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.backlog}
                  sort={state.sorts.backlog}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                  selectionMode={selection.selectionMode}
                  selectedIds={selection.selectedIds}
                  onToggleSelection={selection.toggleItem}
                  onVisibleIdsChange={selection.pruneToVisible}
                />
              )}
              {activeTab === "captures" && (
                <CapturesTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.captures}
                  sort={state.sorts.captures}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                  selectionMode={selection.selectionMode}
                  selectedIds={selection.selectedIds}
                  onToggleSelection={selection.toggleItem}
                  onVisibleIdsChange={selection.pruneToVisible}
                />
              )}
              {activeTab === "initiatives" && (
                <InitiativesTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.initiatives}
                  sort={state.sorts.initiatives}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                  selectionMode={selection.selectionMode}
                  selectedIds={selection.selectedIds}
                  onToggleSelection={selection.toggleItem}
                  onVisibleIdsChange={selection.pruneToVisible}
                />
              )}
              {activeTab === "goals" && (
                <GoalsTab
                  searchQuery={debouncedSearch}
                  sort={state.sorts.goals}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                />
              )}
              {activeTab === "operatingModes" && (
                <OperatingModesTab
                  searchQuery={debouncedSearch}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                />
              )}
              {activeTab === "executions" && (
                <ExecutionsTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.executions}
                  sort={state.sorts.executions}
                  onItemClick={onItemClick}
                  onClearSearch={clearSearch}
                  selectionMode={selection.selectionMode}
                  selectedIds={selection.selectedIds}
                  onToggleSelection={selection.toggleItem}
                  onVisibleIdsChange={selection.pruneToVisible}
                />
              )}
              {activeTab === "sessions" && (
                <SessionsTab
                  searchQuery={debouncedSearch}
                  filters={state.filters.sessions}
                  sort={state.sorts.sessions}
                  onOpenSession={onOpenAgentSession}
                  onClearSearch={clearSearch}
                  selectionMode={selection.selectionMode}
                  selectedIds={selection.selectedIds}
                  onToggleSelection={selection.toggleItem}
                  onVisibleIdsChange={selection.pruneToVisible}
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
