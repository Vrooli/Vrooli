import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useChats } from "./hooks/useChats";
import { useAsyncStatus } from "./hooks/useAsyncStatus";
import { useTools } from "./hooks/useTools";
import { useActiveTemplate } from "./hooks/useActiveTemplate";
import { useChatRoute, usePopStateListener } from "./hooks/useChatRoute";
import { useResizableSidebar } from "./hooks/useResizableSidebar";
import { useIsMobile } from "./hooks/useIsMobile";
import { useSidebarState } from "./hooks/useSidebarState";
import { useAppModals } from "./hooks/useAppModals";
import { useAsyncReferences } from "./hooks/useAsyncReferences";
import { useChatViewCallbacks } from "./hooks/useChatViewCallbacks";
import { useAgentChatActions } from "./hooks/useAgentChatActions";
import { useAppKeyboardShortcuts } from "./hooks/useAppKeyboardShortcuts";
import { AppRouter } from "./components/AppRouter";
import { SidebarPanel } from "./components/layout/SidebarPanel";
import { AppDialogs } from "./components/layout/AppDialogs";
import { MainContent } from "./components/layout/MainContent";
import { ToastProvider } from "./components/ui/toast";
import { selectorsManifest } from "./consts/selectors";
import { getViewMode, setViewMode, type ViewMode } from "./components/settings/Settings";
import { deleteArchivedChats, markAllChatsAsRead } from "./lib/api";

function AppContent() {
  const appTestIds = {
    container: selectorsManifest.selectors["app.container"]?.testId ?? "inbox-container",
    mobileSidebarOverlay: selectorsManifest.selectors["app.mobileSidebarOverlay"]?.testId ?? "mobile-sidebar-overlay",
    closeSidebarButton: selectorsManifest.selectors["app.closeSidebarButton"]?.testId ?? "close-sidebar-button",
  };

  const [viewMode, setViewModeState] = useState<ViewMode>(getViewMode);
  const [isClearingArchived, setIsClearingArchived] = useState(false);
  const [isMarkingAllAsRead, setIsMarkingAllAsRead] = useState(false);
  const [scrollToMessageId, setScrollToMessageId] = useState<string | null>(null);
  const queryClient = useQueryClient();
  const isMobile = useIsMobile();
  const searchInputRef = useRef<HTMLInputElement>(null);

  const {
    sidebarOpen, setSidebarOpen, chatListOpen, setChatListOpen,
    sidebarCollapsed, handleToggleSidebarCollapsed,
  } = useSidebarState();

  const modals = useAppModals();

  const {
    width: sidebarWidth, isResizing: isSidebarResizing,
    panelRef: sidebarPanelRef, containerRef: sidebarContainerRef, handleResizeStart,
  } = useResizableSidebar({
    storageKey: "agent-inbox:sidebar-width",
    defaultWidth: 320, minWidth: 200, maxWidthRatio: 0.5,
  });

  const { initialChatId, setChatInUrl } = useChatRoute();
  const handleChatChange = useCallback(
    (chatId: string | null) => { setChatInUrl(chatId || ""); },
    [setChatInUrl]
  );

  const templateDeactivateRef = useRef<(() => void) | null>(null);
  const handleTemplateDeactivated = useCallback(() => { templateDeactivateRef.current?.(); }, []);

  const {
    selectedChatId, currentView, isGenerating, streamingContent,
    activeToolCalls, generatedImages, isRegenerating,
    chats, chatData, models, labels, loadingChats, loadingChat,
    setCurrentView, selectChat, sendMessage, createChatWithMessage,
    createChat, deleteChat, deleteAllChats, updateChat,
    toggleRead, toggleArchive, toggleStar,
    createLabel, deleteLabel, assignLabel, removeLabel,
    regenerateMessage, selectBranch,
    editingMessage, setEditingMessage, editMessageAndComplete, cancelEdit,
    bulkOperate, forkConversation,
    isCreatingChat, isDeletingAllChats, isBulkOperating, isForking,
  } = useChats({ initialChatId, onChatChange: handleChatChange, onTemplateDeactivated: handleTemplateDeactivated });

  const {
    operations: asyncOperations, activeOperations: activeAsyncOperations,
    completedOperations: completedAsyncOperations,
    cancelOperation: cancelAsyncOperation, refreshOperation: refreshAsyncOperation,
    fetchHistory: fetchAsyncHistory,
  } = useAsyncStatus(selectedChatId);

  const {
    asyncReferences, handleInsertAsyncReference, handleRemoveAsyncReference,
    hasMoreAsyncHistory, handleFetchAsyncHistory,
  } = useAsyncReferences(selectedChatId, fetchAsyncHistory);

  const { enableToolsByIds } = useTools({ chatId: selectedChatId ?? undefined });
  const activeTemplate = useActiveTemplate(selectedChatId ?? undefined, chatData?.chat);

  useEffect(() => {
    templateDeactivateRef.current = () => {
      if (selectedChatId && chatData?.chat.id === selectedChatId) void activeTemplate.deactivate();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTemplate.deactivate, selectedChatId, chatData?.chat.id]);

  const handleTemplateActivated = useCallback(
    async (templateId: string, toolIds: string[]) => {
      if (!selectedChatId) return;
      await enableToolsByIds(toolIds);
      await activeTemplate.activate(templateId, toolIds);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, enableToolsByIds, activeTemplate.activate]
  );

  usePopStateListener(useCallback((chatId: string) => { selectChat(chatId); }, [selectChat]));

  useEffect(() => {
    if (selectedChatId && window.innerWidth < 1024) { setSidebarOpen(false); setChatListOpen(false); }
  }, [selectedChatId, setChatListOpen, setSidebarOpen]);

  const chatCounts = useMemo(() => ({
    inbox: chats.filter((c) => !c.is_archived).length,
    starred: chats.filter((c) => c.is_starred).length,
    archived: chats.filter((c) => c.is_archived).length,
  }), [chats]);

  const handleSelectChat = useCallback(
    (chatId: string, messageId?: string) => {
      selectChat(chatId);
      setScrollToMessageId(null);
      requestAnimationFrame(() => { setScrollToMessageId(messageId || null); });
      if (window.innerWidth < 1024) setChatListOpen(false);
    },
    [selectChat, setChatListOpen]
  );

  const handleNewChat = useCallback(() => {
    createChat({}); if (window.innerWidth < 768) setSidebarOpen(false);
  }, [createChat, setSidebarOpen]);

  const handleNewAgentChat = useCallback(() => {
    createChat({ chat_mode: "agent" }); if (window.innerWidth < 768) setSidebarOpen(false);
  }, [createChat, setSidebarOpen]);

  const handleBackToList = useCallback(() => { setChatListOpen(true); }, [setChatListOpen]);
  const handleViewModeChange = useCallback((mode: ViewMode) => { setViewModeState(mode); setViewMode(mode); }, []);

  const handleClearArchived = useCallback(async () => {
    setIsClearingArchived(true);
    try { await deleteArchivedChats(); void queryClient.invalidateQueries({ queryKey: ["chats"] }); }
    finally { setIsClearingArchived(false); }
  }, [queryClient]);

  const handleMarkAllAsRead = useCallback(async () => {
    setIsMarkingAllAsRead(true);
    try { await markAllChatsAsRead(); void queryClient.invalidateQueries({ queryKey: ["chats"] }); }
    finally { setIsMarkingAllAsRead(false); }
  }, [queryClient]);

  const handleDeselectChat = useCallback(() => {
    selectChat("");
    if (window.innerWidth < 1024) { setSidebarOpen(false); setChatListOpen(false); }
  }, [selectChat, setChatListOpen, setSidebarOpen]);

  const { handleStartAgentChat, handleAttachRunFromEmpty } = useAgentChatActions({ selectChat });

  const chatViewCallbacks = useChatViewCallbacks({
    selectedChatId, updateChat, toggleRead, toggleStar, toggleArchive,
    deleteChat, assignLabel, removeLabel,
    regenerateMessage: (chatId, messageId) => { void regenerateMessage(chatId, messageId); },
    editingMessage,
    editMessageAndComplete: (messageId, payload) => { void editMessageAndComplete(messageId, payload); },
    setScrollToMessageId,
  });

  const handleManageLabels = useCallback(() => {
    modals.setShowLabelManager(true);
  }, [modals]);

  const handleRenameChat = useCallback(
    (chatId: string, newName: string) => {
      updateChat({ chatId, data: { name: newName } });
    },
    [updateChat],
  );

  const handleBulkOperate = useCallback(
    (chatIds: string[], operation: Parameters<typeof bulkOperate>[0]["operation"], labelId?: string) => {
      bulkOperate({ chatIds, operation, labelId });
    },
    [bulkOperate],
  );

  const handleSendMessage = useCallback(
    (payload: Parameters<typeof sendMessage>[0]) => {
      void sendMessage(payload);
    },
    [sendMessage],
  );

  const handleTemplateDeactivate = useCallback(() => {
    void activeTemplate.deactivate();
  }, [activeTemplate]);

  const handleCreateChatWithMessage = useCallback(
    (payload: Parameters<typeof createChatWithMessage>[0]) => {
      void createChatWithMessage(payload);
    },
    [createChatWithMessage],
  );

  const handleStartAgentChatFromEmpty = useCallback(
    (payload: Parameters<typeof handleStartAgentChat>[0], config: Parameters<typeof handleStartAgentChat>[1]) => {
      void handleStartAgentChat(payload, config);
    },
    [handleStartAgentChat],
  );

  const handleAttachRunFromEmptyState = useCallback(
    (run: Parameters<typeof handleAttachRunFromEmpty>[0]) => {
      void handleAttachRunFromEmpty(run);
    },
    [handleAttachRunFromEmpty],
  );

  const visibleChats = useMemo(() => {
    return chats.filter((c) => {
      if (currentView === "inbox") return !c.is_archived;
      if (currentView === "starred") return c.is_starred;
      return c.is_archived;
    });
  }, [chats, currentView]);

  const { focusedIndex } = useAppKeyboardShortcuts({
    searchInputRef, visibleChats, currentView, selectedChatId,
    createChat, setCurrentView,
    handleOpenSettings: modals.handleOpenSettings,
    handleShowKeyboardShortcuts: modals.handleShowKeyboardShortcuts,
    handleDeselectChat, handleSelectChat, toggleStar, toggleArchive,
    showLabelManager: modals.showLabelManager, showSettings: modals.showSettings,
    showKeyboardShortcuts: modals.showKeyboardShortcuts, showUsageStats: modals.showUsageStats,
    setShowLabelManager: modals.setShowLabelManager, setShowSettings: modals.setShowSettings,
    setShowKeyboardShortcuts: modals.setShowKeyboardShortcuts, setShowUsageStats: modals.setShowUsageStats,
    anyModalOpen: modals.anyModalOpen,
  });

  return (
    <div ref={sidebarContainerRef} className="h-screen bg-slate-950 text-slate-50 flex overflow-hidden" data-testid={appTestIds.container}>
      <SidebarPanel
        ref={searchInputRef}
        panelRef={sidebarPanelRef}
        sidebarOpen={sidebarOpen} chatListOpen={chatListOpen}
        sidebarCollapsed={sidebarCollapsed} sidebarWidth={sidebarWidth}
        isSidebarResizing={isSidebarResizing} isMobile={isMobile}
        handleResizeStart={handleResizeStart}
        setSidebarOpen={setSidebarOpen} setChatListOpen={setChatListOpen}
        handleToggleSidebarCollapsed={handleToggleSidebarCollapsed}
        currentView={currentView} setCurrentView={setCurrentView}
        handleNewChat={handleNewChat} handleNewAgentChat={handleNewAgentChat}
        onManageLabels={handleManageLabels}
        onOpenSettings={modals.handleOpenSettings}
        onShowKeyboardShortcuts={modals.handleShowKeyboardShortcuts}
        isCreatingChat={isCreatingChat} labels={labels} chatCounts={chatCounts}
        chats={chats} selectedChatId={selectedChatId} focusedIndex={focusedIndex}
        isLoadingChats={loadingChats} onSelectChat={handleSelectChat}
        onRenameChat={handleRenameChat}
        onBulkOperate={handleBulkOperate}
        isBulkOperating={isBulkOperating}
        onClearArchived={handleClearArchived} isClearingArchived={isClearingArchived}
        onDeselectChat={handleDeselectChat}
        mobileSidebarOverlayTestId={appTestIds.mobileSidebarOverlay}
        closeSidebarButtonTestId={appTestIds.closeSidebarButton}
      />

      <MainContent
        selectedChatId={selectedChatId} chatData={chatData} models={models} labels={labels}
        loadingChat={loadingChat} isGenerating={isGenerating}
        streamingContent={streamingContent} activeToolCalls={activeToolCalls}
        generatedImages={generatedImages} scrollToMessageId={scrollToMessageId}
        chatViewCallbacks={chatViewCallbacks}
        sendMessage={handleSendMessage}
        viewMode={viewMode}
        selectBranch={selectBranch} forkConversation={forkConversation}
        isRegenerating={isRegenerating} isForking={isForking}
        editingMessage={editingMessage} setEditingMessage={setEditingMessage} cancelEdit={cancelEdit}
        asyncOperations={asyncOperations} activeAsyncOperations={activeAsyncOperations}
        completedAsyncOperations={completedAsyncOperations}
        cancelAsyncOperation={cancelAsyncOperation} refreshAsyncOperation={refreshAsyncOperation}
        handleFetchAsyncHistory={handleFetchAsyncHistory} hasMoreAsyncHistory={hasMoreAsyncHistory}
        asyncReferences={asyncReferences}
        handleInsertAsyncReference={handleInsertAsyncReference}
        handleRemoveAsyncReference={handleRemoveAsyncReference}
        handleTemplateActivated={handleTemplateActivated}
        activeTemplateId={activeTemplate.activeTemplateId}
        onTemplateDeactivate={handleTemplateDeactivate}
        handleOpenAgentSettings={modals.handleOpenAgentSettings}
        handleBackToList={handleBackToList} isMobile={isMobile}
        setSidebarOpen={setSidebarOpen} chatListOpen={chatListOpen}
        createChatWithMessage={handleCreateChatWithMessage}
        handleStartAgentChat={handleStartAgentChatFromEmpty}
        handleAttachRunFromEmpty={handleAttachRunFromEmptyState}
        isCreatingChat={isCreatingChat}
      />

      <AppDialogs
        modals={modals} labels={labels} models={models}
        createLabel={createLabel} deleteLabel={deleteLabel}
        deleteAllChats={deleteAllChats} isDeletingAllChats={isDeletingAllChats}
        handleClearArchived={handleClearArchived} isClearingArchived={isClearingArchived}
        handleMarkAllAsRead={handleMarkAllAsRead} isMarkingAllAsRead={isMarkingAllAsRead}
        viewMode={viewMode} handleViewModeChange={handleViewModeChange}
      />
    </div>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <AppRouter>
        <AppContent />
      </AppRouter>
    </ToastProvider>
  );
}
