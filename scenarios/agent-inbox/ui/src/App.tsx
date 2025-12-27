import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import { Menu, X, ChevronLeft, Star } from "lucide-react";
import { useChats } from "./hooks/useChats";
import { useChatRoute, usePopStateListener } from "./hooks/useChatRoute";
import { useKeyboardShortcuts, type KeyboardShortcut } from "./hooks/useKeyboardShortcuts";
import { Sidebar } from "./components/layout/Sidebar";
import { ChatList } from "./components/chat/ChatList";
import { ChatView } from "./components/chat/ChatView";
import { EmptyState } from "./components/chat/EmptyState";
import { LabelManager } from "./components/labels/LabelManager";
import { Settings, getViewMode, setViewMode, type ViewMode } from "./components/settings/Settings";
import { KeyboardShortcuts } from "./components/settings/KeyboardShortcuts";
import { UsageStats } from "./components/settings/UsageStats";
import { Button } from "./components/ui/button";
import { ToastProvider } from "./components/ui/toast";

function AppContent() {
  const [showLabelManager, setShowLabelManager] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [showKeyboardShortcuts, setShowKeyboardShortcuts] = useState(false);
  const [showUsageStats, setShowUsageStats] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [chatListOpen, setChatListOpen] = useState(true);
  const [viewMode, setViewModeState] = useState<ViewMode>(getViewMode);
  const searchInputRef = useRef<HTMLInputElement>(null);
  // Focused chat index for j/k navigation (separate from selected chat)
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);

  // URL-based routing for chats
  const { initialChatId, setChatInUrl } = useChatRoute();

  // Handle URL sync when chat changes
  const handleChatChange = useCallback(
    (chatId: string | null) => {
      setChatInUrl(chatId || "");
    },
    [setChatInUrl]
  );

  const {
    // State
    selectedChatId,
    currentView,
    isGenerating,
    streamingContent,
    activeToolCalls,
    isRegenerating,

    // Data
    chats,
    chatData,
    models,
    labels,

    // Loading states
    loadingChats,
    loadingChat,

    // Actions
    setCurrentView,
    selectChat,
    sendMessage,
    createChatWithMessage,

    // Mutations
    createChat,
    deleteChat,
    deleteAllChats,
    updateChat,
    toggleRead,
    toggleArchive,
    toggleStar,
    createLabel,
    deleteLabel,
    assignLabel,
    removeLabel,

    // Branching operations
    regenerateMessage,
    selectBranch,

    // Mutation states
    isCreatingChat,
    isDeletingAllChats,
  } = useChats({
    initialChatId,
    onChatChange: handleChatChange,
  });

  // Handle browser back/forward navigation
  usePopStateListener(
    useCallback(
      (chatId: string) => {
        selectChat(chatId);
      },
      [selectChat]
    )
  );

  // Close mobile sidebar when a chat is selected
  useEffect(() => {
    if (selectedChatId && window.innerWidth < 1024) {
      setSidebarOpen(false);
      setChatListOpen(false);
    }
  }, [selectedChatId]);

  // Calculate unread counts for sidebar badges
  const chatCounts = {
    inbox: chats.filter((c) => !c.is_archived).length,
    starred: chats.filter((c) => c.is_starred).length,
    archived: chats.filter((c) => c.is_archived).length,
  };

  // Track which message to scroll to (from search results)
  const [scrollToMessageId, setScrollToMessageId] = useState<string | null>(null);

  const handleSelectChat = useCallback(
    (chatId: string, messageId?: string) => {
      selectChat(chatId);
      // Set message to scroll to if navigating from search
      setScrollToMessageId(messageId || null);
      // Close chat list on mobile when selecting
      if (window.innerWidth < 1024) {
        setChatListOpen(false);
      }
    },
    [selectChat]
  );

  const handleNewChat = useCallback(() => {
    createChat({});
    // Close sidebar on mobile when creating new chat
    if (window.innerWidth < 768) {
      setSidebarOpen(false);
    }
  }, [createChat]);

  const handleBackToList = useCallback(() => {
    setChatListOpen(true);
  }, []);

  const handleOpenSettings = useCallback(() => {
    setShowSettings(true);
  }, []);

  const handleShowKeyboardShortcuts = useCallback(() => {
    setShowKeyboardShortcuts(true);
  }, []);

  const handleShowUsageStats = useCallback(() => {
    setShowUsageStats(true);
  }, []);

  const handleViewModeChange = useCallback((mode: ViewMode) => {
    setViewModeState(mode);
    setViewMode(mode);
  }, []);

  const handleDeselectChat = useCallback(() => {
    selectChat("");
  }, [selectChat]);

  // Keyboard shortcuts
  const anyModalOpen = showLabelManager || showSettings || showKeyboardShortcuts || showUsageStats;

  // Get visible chats for navigation (filtered by current view)
  const visibleChats = useMemo(() => {
    return chats.filter((c) => {
      if (currentView === "inbox") return !c.is_archived;
      if (currentView === "starred") return c.is_starred;
      if (currentView === "archived") return c.is_archived;
      return true;
    });
  }, [chats, currentView]);

  // Reset focused index when view changes or chat list changes
  useEffect(() => {
    setFocusedIndex(-1);
  }, [currentView]);

  // Navigation handlers for j/k
  const handleNavigateDown = useCallback(() => {
    if (visibleChats.length === 0) return;
    setFocusedIndex((prev) => {
      if (prev < 0) return 0;
      return Math.min(prev + 1, visibleChats.length - 1);
    });
  }, [visibleChats.length]);

  const handleNavigateUp = useCallback(() => {
    if (visibleChats.length === 0) return;
    setFocusedIndex((prev) => {
      if (prev < 0) return visibleChats.length - 1;
      return Math.max(prev - 1, 0);
    });
  }, [visibleChats.length]);

  // Open focused chat with Enter
  const handleOpenFocused = useCallback(() => {
    if (focusedIndex >= 0 && focusedIndex < visibleChats.length) {
      const chat = visibleChats[focusedIndex];
      handleSelectChat(chat.id);
    }
  }, [focusedIndex, visibleChats, handleSelectChat]);

  const shortcuts: KeyboardShortcut[] = useMemo(
    () => [
      // J/K navigation (KEY-001)
      {
        key: "j",
        description: "Next chat",
        action: handleNavigateDown,
        category: "navigation",
      },
      {
        key: "k",
        description: "Previous chat",
        action: handleNavigateUp,
        category: "navigation",
      },
      // Enter to open (KEY-002)
      {
        key: "Enter",
        description: "Open focused chat",
        action: handleOpenFocused,
        category: "navigation",
      },
      {
        key: "n",
        ctrlKey: true,
        description: "New chat",
        action: () => createChat({}),
        category: "chat",
      },
      {
        key: "k",
        ctrlKey: true,
        description: "Focus search",
        action: () => searchInputRef.current?.focus(),
        category: "navigation",
      },
      // "/" also focuses search (KEY-005)
      {
        key: "/",
        description: "Focus search",
        action: () => searchInputRef.current?.focus(),
        category: "navigation",
      },
      {
        key: "1",
        ctrlKey: true,
        description: "Go to Inbox",
        action: () => setCurrentView("inbox"),
        category: "navigation",
      },
      {
        key: "2",
        ctrlKey: true,
        description: "Go to Starred",
        action: () => setCurrentView("starred"),
        category: "navigation",
      },
      {
        key: "3",
        ctrlKey: true,
        description: "Go to Archived",
        action: () => setCurrentView("archived"),
        category: "navigation",
      },
      {
        key: ",",
        ctrlKey: true,
        description: "Open settings",
        action: handleOpenSettings,
        category: "general",
      },
      {
        key: "?",
        description: "Show keyboard shortcuts",
        action: handleShowKeyboardShortcuts,
        category: "general",
      },
      {
        key: "Escape",
        description: "Close dialog / deselect chat",
        action: () => {
          if (showLabelManager) setShowLabelManager(false);
          else if (showSettings) setShowSettings(false);
          else if (showKeyboardShortcuts) setShowKeyboardShortcuts(false);
          else if (showUsageStats) setShowUsageStats(false);
          else if (selectedChatId) handleDeselectChat();
        },
        category: "navigation",
      },
      {
        key: "s",
        ctrlKey: true,
        description: "Toggle star on current chat",
        action: () => {
          if (selectedChatId) toggleStar({ chatId: selectedChatId });
        },
        category: "chat",
      },
      {
        key: "e",
        ctrlKey: true,
        description: "Archive current chat",
        action: () => {
          if (selectedChatId) toggleArchive({ chatId: selectedChatId });
        },
        category: "chat",
      },
    ],
    [
      handleNavigateDown,
      handleNavigateUp,
      handleOpenFocused,
      createChat,
      setCurrentView,
      handleOpenSettings,
      handleShowKeyboardShortcuts,
      showLabelManager,
      showSettings,
      showKeyboardShortcuts,
      showUsageStats,
      selectedChatId,
      handleDeselectChat,
      toggleStar,
      toggleArchive,
    ]
  );

  useKeyboardShortcuts(shortcuts, { disabled: anyModalOpen && shortcuts.every(s => s.key !== "Escape") });

  return (
    <div className="h-screen bg-slate-950 text-slate-50 flex overflow-hidden" data-testid="inbox-container">
      {/* Mobile Header */}
      <div className="lg:hidden fixed top-0 left-0 right-0 z-40 bg-slate-950 border-b border-white/10 px-2 py-2 flex items-center justify-between safe-top">
        <div className="flex items-center gap-1 min-w-0 flex-1">
          {selectedChatId && !chatListOpen ? (
            <Button
              variant="ghost"
              size="icon"
              onClick={handleBackToList}
              className="h-10 w-10 shrink-0"
              data-testid="mobile-back-button"
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setSidebarOpen(true)}
              className="h-10 w-10 shrink-0"
              data-testid="mobile-menu-button"
            >
              <Menu className="h-5 w-5" />
            </Button>
          )}
          <span className="text-sm font-medium truncate">
            {selectedChatId && !chatListOpen
              ? chatData?.chat.name || "Chat"
              : "Agent Inbox"}
          </span>
        </div>
        {/* Mobile header actions */}
        {selectedChatId && !chatListOpen && (
          <div className="flex items-center gap-1 shrink-0">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => toggleStar({ chatId: selectedChatId })}
              className="h-10 w-10"
              data-testid="mobile-star-button"
            >
              <Star className={`h-4 w-4 ${chatData?.chat.is_starred ? "fill-yellow-400 text-yellow-400" : ""}`} />
            </Button>
          </div>
        )}
      </div>

      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
          className="lg:hidden fixed inset-0 z-50 bg-black/60 backdrop-blur-sm"
          onClick={() => setSidebarOpen(false)}
          data-testid="mobile-sidebar-overlay"
        />
      )}

      {/* Sidebar - Desktop: always visible, Mobile: slide-in */}
      <div
        className={`fixed lg:relative inset-y-0 left-0 z-50 lg:z-auto transform transition-transform duration-200 ${
          sidebarOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        }`}
      >
        <div className="lg:hidden absolute top-3 right-3 z-10">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setSidebarOpen(false)}
            data-testid="close-sidebar-button"
          >
            <X className="h-5 w-5" />
          </Button>
        </div>
        <Sidebar
          currentView={currentView}
          onViewChange={(view) => {
            setCurrentView(view);
            if (window.innerWidth < 768) {
              setSidebarOpen(false);
            }
          }}
          onNewChat={handleNewChat}
          onManageLabels={() => setShowLabelManager(true)}
          onOpenSettings={handleOpenSettings}
          onShowKeyboardShortcuts={handleShowKeyboardShortcuts}
          isCreatingChat={isCreatingChat}
          labels={labels}
          chatCounts={chatCounts}
        />
      </div>

      {/* Chat List - Desktop: fixed width, Mobile: full width */}
      <div
        className={`${
          chatListOpen ? "flex" : "hidden lg:flex"
        } shrink-0 pt-14 lg:pt-0 w-full lg:w-auto`}
      >
        <ChatList
          ref={searchInputRef}
          chats={chats}
          labels={labels}
          selectedChatId={selectedChatId}
          focusedIndex={focusedIndex}
          currentView={currentView}
          isLoading={loadingChats}
          onSelectChat={handleSelectChat}
          onNewChat={handleNewChat}
          onRenameChat={(chatId, newName) => updateChat({ chatId, data: { name: newName } })}
        />
      </div>

      {/* Main Content - Chat View or Empty State */}
      <div
        className={`flex-1 flex flex-col pt-14 lg:pt-0 ${
          chatListOpen && selectedChatId ? "hidden lg:flex" : "flex"
        }`}
      >
        {selectedChatId ? (
          <ChatView
            chatData={chatData || null}
            models={models}
            labels={labels}
            isLoading={loadingChat}
            isGenerating={isGenerating}
            streamingContent={streamingContent}
            activeToolCalls={activeToolCalls}
            scrollToMessageId={scrollToMessageId}
            onScrollComplete={() => setScrollToMessageId(null)}
            onSendMessage={sendMessage}
            onUpdateChat={(data) => updateChat({ chatId: selectedChatId, data })}
            onToggleRead={() => toggleRead({ chatId: selectedChatId })}
            onToggleStar={() => toggleStar({ chatId: selectedChatId })}
            onToggleArchive={() => toggleArchive({ chatId: selectedChatId })}
            onDeleteChat={() => deleteChat(selectedChatId)}
            onAssignLabel={(labelId) => assignLabel({ chatId: selectedChatId, labelId })}
            onRemoveLabel={(labelId) => removeLabel({ chatId: selectedChatId, labelId })}
            viewMode={viewMode}
            onRegenerateMessage={(messageId) => regenerateMessage(selectedChatId, messageId)}
            onSelectBranch={selectBranch}
            isRegenerating={isRegenerating}
          />
        ) : (
          <EmptyState onStartChat={createChatWithMessage} isCreating={isCreatingChat} />
        )}
      </div>

      {/* Label Manager Dialog */}
      <LabelManager
        open={showLabelManager}
        onClose={() => setShowLabelManager(false)}
        labels={labels}
        onCreateLabel={createLabel}
        onDeleteLabel={deleteLabel}
      />

      {/* Settings Dialog */}
      <Settings
        open={showSettings}
        onClose={() => setShowSettings(false)}
        onDeleteAllChats={deleteAllChats}
        isDeletingAll={isDeletingAllChats}
        onShowKeyboardShortcuts={handleShowKeyboardShortcuts}
        onShowUsageStats={handleShowUsageStats}
        models={models}
        viewMode={viewMode}
        onViewModeChange={handleViewModeChange}
      />

      {/* Keyboard Shortcuts Dialog */}
      <KeyboardShortcuts
        open={showKeyboardShortcuts}
        onClose={() => setShowKeyboardShortcuts(false)}
      />

      {/* Usage Statistics Dialog */}
      <UsageStats
        isOpen={showUsageStats}
        onClose={() => setShowUsageStats(false)}
      />
    </div>
  );
}

export default function App() {
  return (
    <ToastProvider>
      <AppContent />
    </ToastProvider>
  );
}
