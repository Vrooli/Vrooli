/**
 * Sidebar - Multi-tab navigation surface for the graph workspace.
 *
 * Composes: SidebarHeader → SearchBar → Tabs → FilterBar → Tab Content.
 * Manages sidebar UI state via persisted reducer state.
 */

import { useCallback, useState, type ChangeEvent, type HTMLAttributes, type Ref } from "react";
import { FilePlus2, ListChecks, Plus } from "lucide-react";
import { cn } from "../../../../lib/utils";
import { SearchBar } from "../../../../components/ui/search-bar";
import { CreateGoalDialog } from "../../../../components/goals/CreateGoalDialog";
import { BacklogFormDialog } from "../../../../components/backlog/backlog-form-dialog";
import { CreateWorkFromPlanDialog } from "../../../../components/plan/CreateWorkFromPlanDialog";
import { useAISearchStatus } from "../../../../lib/ai-search";
import { useGraphUIStore } from "../../stores/graph-ui-store";
import { useBacklogStore } from "../../../../stores";
import { backlogService } from "../../../../services";
import { useSidebarState } from "./useSidebarState";
import type { SearchMode } from "./useSidebarState";
import type { SidebarTab } from "./types";
import type { BacklogFormValues } from "../../../../types";
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
  onQuickCapture?: () => void;
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
  onQuickCapture,
  desktopWidth,
  resizeHandleProps,
  asideRef,
}: SidebarProps) {
  const sidebarCollapsed = useGraphUIStore((s) => s.sidebarCollapsed);
  const toggleSidebar = useGraphUIStore((s) => s.toggleSidebar);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

  const [state, dispatch] = useSidebarState();
  const [showCreateGoal, setShowCreateGoal] = useState(false);
  const [showCreateBacklog, setShowCreateBacklog] = useState(false);
  const [showCreateFromPlan, setShowCreateFromPlan] = useState(false);
  const [isCreatingBacklog, setIsCreatingBacklog] = useState(false);
  const [createBacklogError, setCreateBacklogError] = useState<string | null>(null);
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
  const createAction = !aiMode ? createActionForTab(activeTab, {
    onCreateGoal: () => setShowCreateGoal(true),
    onCreateBacklog: () => {
      setCreateBacklogError(null);
      setShowCreateBacklog(true);
    },
    onQuickCapture,
  }) : null;

  const handleCreateBacklog = useCallback(
    async (values: BacklogFormValues) => {
      setIsCreatingBacklog(true);
      setCreateBacklogError(null);
      try {
        const created = await backlogService.create({ ...values, suggestedSkills: [] });
        upsertBacklogItem(created);
        setShowCreateBacklog(false);
      } catch (err) {
        setCreateBacklogError(err instanceof Error ? err.message : "Failed to create backlog item");
      } finally {
        setIsCreatingBacklog(false);
      }
    },
    [upsertBacklogItem],
  );

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
          hideOpsTriggerOnDesktop
        />

        {/* Search */}
        <div className="flex shrink-0 flex-col gap-2 border-b border-slate-200/20 px-3 py-2">
          {showSelectionControls && selection.selectionMode ? (
            <div className="flex min-h-8 flex-wrap items-center justify-between gap-1.5 rounded-lg border border-slate-700/70 bg-slate-900/80 px-2 text-xs" data-testid="sidebar-selection-controls">
              <span className="whitespace-nowrap text-slate-300" data-testid="sidebar-selected-count">
                {selection.selectedCount} selected
              </span>
              <div className="flex items-center gap-1.5">
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
              </div>
            </div>
          ) : (
            <div className="relative">
              <SearchBar
                placeholder={aiMode ? "Semantic search..." : "Search..."}
                value={state.searchQuery}
                onChange={handleSearchChange}
                widthClass="w-full"
                className={cn(
                  "h-8 text-[16px] md:text-sm",
                  (aiAvailable || showSelectionControls || createAction || activeTab === "backlog") && "pr-36",
                )}
                data-testid="sidebar-search"
              />
              <div className="absolute right-2 top-1/2 flex -translate-y-1/2 items-center gap-1">
                {createAction && (
                  <button
                    type="button"
                    onClick={createAction.onClick}
                    className="inline-flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-700/70 hover:text-slate-100"
                    title={createAction.label}
                    aria-label={createAction.label}
                    data-testid="sidebar-create-current"
                  >
                    <Plus className="h-3.5 w-3.5" aria-hidden />
                  </button>
                )}
                {activeTab === "backlog" && (
                  <button
                    type="button"
                    onClick={() => setShowCreateFromPlan(true)}
                    className="inline-flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-700/70 hover:text-slate-100"
                    title="Create work from plan"
                    aria-label="Create work from plan"
                    data-testid="sidebar-create-from-plan"
                  >
                    <FilePlus2 className="h-3.5 w-3.5" aria-hidden />
                  </button>
                )}
                <SearchModeToggle
                  mode={state.searchMode}
                  onChange={handleSearchModeChange}
                  aiAvailable={aiAvailable}
                  unavailableReason={aiSearchStatus.status?.message ?? aiSearchStatus.error ?? undefined}
                />
                {showSelectionControls && (
                  <button
                    type="button"
                    onClick={selection.toggleMode}
                    className="inline-flex h-6 w-6 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-700/70 hover:text-slate-100"
                    title="Select visible items"
                    aria-label="Select visible items"
                    data-testid="sidebar-select-mode"
                  >
                    <ListChecks className="h-3.5 w-3.5" aria-hidden />
                  </button>
                )}
              </div>
            </div>
          )}
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
                  onCreateBacklog={createAction?.tab === "backlog" ? createAction.onClick : undefined}
                  onCreateFromPlan={() => setShowCreateFromPlan(true)}
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
                  onCreateCapture={createAction?.tab === "captures" ? createAction.onClick : undefined}
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
                  onCreateGoal={createAction?.tab === "goals" ? createAction.onClick : undefined}
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
      <CreateGoalDialog
        isOpen={showCreateGoal}
        onClose={() => setShowCreateGoal(false)}
      />
      <BacklogFormDialog
        isOpen={showCreateBacklog}
        mode="create"
        isSubmitting={isCreatingBacklog}
        submitError={createBacklogError}
        onClose={() => {
          setShowCreateBacklog(false);
          setCreateBacklogError(null);
        }}
        onSubmit={(values) => void handleCreateBacklog(values)}
      />
      <CreateWorkFromPlanDialog
        isOpen={showCreateFromPlan}
        onClose={() => setShowCreateFromPlan(false)}
        onImported={() => {
          void useBacklogStore.getState().fetchBacklog({ force: true });
        }}
      />
    </>
  );
}

function createActionForTab(
  tab: SidebarTab,
  handlers: {
    onCreateGoal: () => void;
    onCreateBacklog: () => void;
    onQuickCapture?: () => void;
  },
): { tab: "goals" | "backlog" | "captures"; label: string; onClick: () => void } | null {
  switch (tab) {
    case "goals":
      return { tab, label: "Create goal", onClick: handlers.onCreateGoal };
    case "backlog":
      return { tab, label: "Create backlog item", onClick: handlers.onCreateBacklog };
    case "captures":
      return handlers.onQuickCapture ? { tab, label: "Quick capture", onClick: handlers.onQuickCapture } : null;
    default:
      return null;
  }
}
