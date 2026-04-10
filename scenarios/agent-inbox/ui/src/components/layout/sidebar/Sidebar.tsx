import { useState, useMemo, forwardRef } from "react";
import { useSearch } from "../../../hooks/useSearch";
import { selectorsManifest } from "../../../consts/selectors";
import { CollapsedSidebar } from "./CollapsedSidebar";
import { SidebarHeader } from "./SidebarHeader";
import { BulkActionsBar } from "./BulkActionsBar";
import { NavigationTabs } from "./NavigationTabs";
import { ClearArchivedSection } from "./ClearArchivedSection";
import { SearchPanel } from "./SearchPanel";
import type { ContentSearchOptions } from "./SearchPanel";
import { ChatList } from "./ChatList";
import { SidebarFooter } from "./SidebarFooter";
import { useBulkSelection } from "./useBulkSelection";
import { viewLabels } from "./types";
import type { SidebarProps, ChatSearchMode } from "./types";

export const Sidebar = forwardRef<HTMLInputElement, SidebarProps>(function Sidebar(
  {
    currentView,
    onViewChange,
    onNewChat,
    onNewAgentChat,
    onManageLabels,
    onOpenSettings,
    onShowKeyboardShortcuts,
    isCreatingChat,
    labels,
    chatCounts,
    chats,
    selectedChatId,
    focusedIndex = -1,
    isLoadingChats,
    onSelectChat,
    onRenameChat,
    onBulkOperate,
    isBulkOperating = false,
    isCollapsed = false,
    onToggleCollapsed,
    onClearArchived,
    isClearingArchived = false,
    onDeselectChat,
  },
  ref
) {
  const sidebarTestIds = {
    container: selectorsManifest.selectors["sidebar.container"]?.testId ?? "sidebar",
    newChatButton: selectorsManifest.selectors["sidebar.newChatButton"]?.testId ?? "new-chat-button",
    nav: selectorsManifest.selectors["sidebar.nav"]?.testId ?? "sidebar-nav",
    manageLabelsButton: selectorsManifest.selectors["sidebar.manageLabelsButton"]?.testId ?? "manage-labels-button",
    mobileActionsButton: selectorsManifest.selectors["sidebar.mobileActionsButton"]?.testId ?? "sidebar-mobile-actions",
  };
  const chatListPanelTestIds = {
    searchInput: selectorsManifest.selectors["chatListPanel.searchInput"]?.testId ?? "chat-search-input",
    clearSearchButton: selectorsManifest.selectors["chatListPanel.clearSearchButton"]?.testId ?? "clear-search-button",
    searchModeToggle: selectorsManifest.selectors["chatListPanel.searchModeToggle"]?.testId ?? "search-mode-toggle",
    searchModeQuick: selectorsManifest.selectors["chatListPanel.searchModeQuick"]?.testId ?? "search-mode-quick",
    searchModeContent: selectorsManifest.selectors["chatListPanel.searchModeContent"]?.testId ?? "search-mode-content",
    list: selectorsManifest.selectors["chatListPanel.list"]?.testId ?? "chat-list",
    switchToContentSearchButton: selectorsManifest.selectors["chatListPanel.switchToContentSearchButton"]?.testId ?? "switch-to-content-search",
  };

  // Search mode and options
  const [searchMode, setSearchMode] = useState<ChatSearchMode>("quick");
  const [contentSearchOptions, setContentSearchOptions] = useState<ContentSearchOptions>({
    caseSensitive: false,
    wholeWord: false,
    regex: false,
  });

  // Search hook
  const search = useSearch({
    debounceMs: 300,
    limit: 20,
    mode: searchMode,
    perChat: searchMode === "content" ? 5 : 1,
    ...(searchMode === "content" ? contentSearchOptions : {}),
  });

  // Compute display chats
  const displayChats = useMemo(() => {
    if (!search.isActive) return chats;
    if (searchMode === "quick") {
      const q = search.query.toLowerCase();
      return chats.filter((c) => c.name.toLowerCase().includes(q));
    }
    return search.results.map((r) => r.chat);
  }, [chats, search.isActive, search.query, search.results, searchMode]);

  // Bulk selection
  const bulk = useBulkSelection({
    displayChats,
    selectedChatId,
    currentView,
    onBulkOperate,
  });

  const { emptyMessage } = viewLabels[currentView];

  // Collapsed sidebar view (desktop only)
  if (isCollapsed) {
    return (
      <CollapsedSidebar
        onToggleCollapsed={onToggleCollapsed}
        onNewChat={onNewChat}
        isCreatingChat={isCreatingChat}
        currentView={currentView}
        onViewChange={onViewChange}
        chatCounts={chatCounts}
        onShowKeyboardShortcuts={onShowKeyboardShortcuts}
        onOpenSettings={onOpenSettings}
      />
    );
  }

  return (
    <aside
      className="w-full border-r border-white/10 flex flex-col bg-slate-950 shrink-0 h-full"
      data-testid={sidebarTestIds.container}
    >
      {/* Header with Logo + New Chat */}
      <SidebarHeader
        onDeselectChat={onDeselectChat}
        onToggleCollapsed={onToggleCollapsed}
        onBulkOperate={!!onBulkOperate}
        displayChats={displayChats}
        searchIsActive={search.isActive}
        selectionMode={bulk.selectionMode}
        exitSelectionMode={bulk.exitSelectionMode}
        setSelectionMode={bulk.setSelectionMode}
        onNewChat={onNewChat}
        onNewAgentChat={onNewAgentChat}
        isCreatingChat={isCreatingChat}
        onManageLabels={onManageLabels}
        onShowKeyboardShortcuts={onShowKeyboardShortcuts}
        onOpenSettings={onOpenSettings}
        sidebarTestIds={sidebarTestIds}
      />

      {/* Bulk Actions Bar */}
      {bulk.selectionMode && bulk.selectedChatIds.size > 0 && (
        <BulkActionsBar
          selectedCount={bulk.selectedChatIds.size}
          currentView={currentView}
          isBulkOperating={isBulkOperating}
          onSelectAll={bulk.selectAll}
          onDeselectAll={bulk.deselectAll}
          onBulkOperation={bulk.handleBulkOperation}
        />
      )}

      {/* Navigation Tabs */}
      <NavigationTabs
        currentView={currentView}
        onViewChange={onViewChange}
        chatCounts={chatCounts}
        testId={sidebarTestIds.nav}
      />

      {/* Clear All Archived */}
      {currentView === "archived" && onClearArchived && displayChats.length > 0 && !search.isActive && (
        <ClearArchivedSection
          chatCount={displayChats.length}
          onClearArchived={onClearArchived}
          isClearingArchived={isClearingArchived}
        />
      )}

      {/* Search */}
      <SearchPanel
        ref={ref}
        query={search.query}
        setQuery={search.setQuery}
        clear={search.clear}
        isSearching={search.isSearching}
        searchMode={searchMode}
        onSearchModeChange={setSearchMode}
        contentSearchOptions={contentSearchOptions}
        onContentSearchOptionsChange={setContentSearchOptions}
        testIds={chatListPanelTestIds}
      />

      {/* Chat List */}
      <ChatList
        chats={chats}
        displayChats={displayChats}
        isLoadingChats={isLoadingChats}
        selectedChatId={selectedChatId}
        focusedIndex={focusedIndex}
        onSelectChat={onSelectChat}
        onRenameChat={onRenameChat}
        labels={labels}
        emptyMessage={emptyMessage}
        searchIsActive={search.isActive}
        searchQuery={search.query}
        isSearching={search.isSearching}
        searchResults={search.results}
        searchMode={searchMode}
        onSearchModeChange={setSearchMode}
        clearSearch={search.clear}
        selectionMode={bulk.selectionMode}
        selectedChatIds={bulk.selectedChatIds}
        toggleChatSelection={bulk.toggleChatSelection}
        listTestId={chatListPanelTestIds.list}
        switchToContentSearchTestId={chatListPanelTestIds.switchToContentSearchButton}
      />

      {/* Footer */}
      <SidebarFooter
        labels={labels}
        onManageLabels={onManageLabels}
        onShowKeyboardShortcuts={onShowKeyboardShortcuts}
        onOpenSettings={onOpenSettings}
        manageLabelsTestId={sidebarTestIds.manageLabelsButton}
      />
    </aside>
  );
});
