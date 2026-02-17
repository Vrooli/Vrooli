import { Suspense, lazy, useState, useCallback, useEffect, useMemo, useRef, type RefObject } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  Menu,
  X,
  ChevronLeft,
  Tag,
  MoreVertical,
  Mail,
  MailOpen,
  Star,
  Archive,
  Edit3,
  FileText,
  FileJson,
  File,
  Trash2,
} from "lucide-react";
import { emitShortcutIntent, HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER } from "@vrooli/iframe-bridge";
import { useChats } from "./hooks/useChats";
import { useAsyncStatus, type AsyncStatusUpdate } from "./hooks/useAsyncStatus";
import { useTools } from "./hooks/useTools";
import { useActiveTemplate } from "./hooks/useActiveTemplate";
import { useChatRoute, usePopStateListener } from "./hooks/useChatRoute";
import { useKeyboardShortcuts, type KeyboardShortcut } from "./hooks/useKeyboardShortcuts";
import { useResizableSidebar } from "./hooks/useResizableSidebar";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Sidebar } from "./components/layout/Sidebar";
import { EmptyState } from "./components/chat/EmptyState";
import { LabelManager } from "./components/labels/LabelManager";
import { Settings, getViewMode, setViewMode, type ViewMode, type SettingsTab } from "./components/settings/Settings";
import { KeyboardShortcuts } from "./components/settings/KeyboardShortcuts";
import { UsageStats } from "./components/settings/UsageStats";
import { TemplateEditorModal } from "./components/chat/TemplateEditorModal";
import { ScenarioViewer, useScenarioViewerRoute } from "./components/scenarios/ScenarioViewer";
import { Button } from "./components/ui/button";
import { Dropdown, DropdownItem, DropdownSeparator } from "./components/ui/dropdown";
import { Dialog, DialogBody, DialogFooter, DialogHeader } from "./components/ui/dialog";
import { Input } from "./components/ui/input";
import { ToastProvider, useToast } from "./components/ui/toast";
import { selectorsManifest } from "./consts/selectors";
import { updateTemplate as updateTemplateAPI, updateDefaultTemplate as updateDefaultTemplateAPI } from "./data/templates";
import {
  deleteArchivedChats,
  markAllChatsAsRead,
  startAgentMode,
  deleteChat as deleteChatAPI,
  AgentModeError,
  createChat as createChatAPI,
  exportChat,
} from "./lib/api";
import type { AgentStartConfig } from "./components/chat/AgentStartModal";
import type { MessagePayload } from "./components/chat/MessageInput";
import { getDefaultModel } from "./components/settings/Settings";
import type { TemplateWithSource } from "./lib/types/templates";

// Sidebar collapsed state persistence (desktop)
const LazyChatView = lazy(async () => {
  const module = await import("./components/chat/ChatView");
  return { default: module.ChatView };
});

const SIDEBAR_COLLAPSED_KEY = "agent-inbox:sidebar-collapsed";

function getSidebarCollapsed(): boolean {
  if (typeof window !== "undefined") {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true";
  }
  return false;
}

function setSidebarCollapsed(collapsed: boolean): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(SIDEBAR_COLLAPSED_KEY, String(collapsed));
  }
}

// Chat list open state persistence (mobile)
const CHAT_LIST_OPEN_KEY = "agent-inbox:chat-list-open";

function getChatListOpen(): boolean {
  if (typeof window !== "undefined") {
    // Default to false (closed) so drawer doesn't cover main content
    return localStorage.getItem(CHAT_LIST_OPEN_KEY) === "true";
  }
  return false;
}

function setChatListOpenStorage(open: boolean): void {
  if (typeof window !== "undefined") {
    localStorage.setItem(CHAT_LIST_OPEN_KEY, String(open));
  }
}

// Hook to detect if screen is mobile (below lg breakpoint)
function useIsMobile(breakpoint = 1024): boolean {
  const [isMobile, setIsMobile] = useState(() => {
    if (typeof window !== "undefined") {
      return window.innerWidth < breakpoint;
    }
    return false;
  });

  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth < breakpoint);
    };
    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, [breakpoint]);

  return isMobile;
}

function AppContent() {
  const appTestIds = {
    container: selectorsManifest.selectors["app.container"]?.testId ?? "inbox-container",
    mobileBackButton: selectorsManifest.selectors["app.mobileBackButton"]?.testId ?? "mobile-back-button",
    mobileMenuButton: selectorsManifest.selectors["app.mobileMenuButton"]?.testId ?? "mobile-menu-button",
    mobileSidebarOverlay: selectorsManifest.selectors["app.mobileSidebarOverlay"]?.testId ?? "mobile-sidebar-overlay",
    closeSidebarButton: selectorsManifest.selectors["app.closeSidebarButton"]?.testId ?? "close-sidebar-button",
  };
  const [showLabelManager, setShowLabelManager] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = useState<SettingsTab>("general");
  const [showKeyboardShortcuts, setShowKeyboardShortcuts] = useState(false);
  const [showUsageStats, setShowUsageStats] = useState(false);
  const [settingsEditingTemplate, setSettingsEditingTemplate] = useState<TemplateWithSource | null>(null);
  const [settingsAllTemplates, setSettingsAllTemplates] = useState<TemplateWithSource[]>([]);
  const [showMobileRenameDialog, setShowMobileRenameDialog] = useState(false);
  const [mobileChatName, setMobileChatName] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [chatListOpen, setChatListOpenState] = useState(getChatListOpen);

  // Wrapper to persist chat list state to localStorage
  const setChatListOpen = useCallback((open: boolean) => {
    setChatListOpenState(open);
    setChatListOpenStorage(open);
  }, []);
  const [sidebarCollapsed, setSidebarCollapsedState] = useState(getSidebarCollapsed);
  const [viewMode, setViewModeState] = useState<ViewMode>(getViewMode);
  const [isClearingArchived, setIsClearingArchived] = useState(false);
  const [isMarkingAllAsRead, setIsMarkingAllAsRead] = useState(false);
  const queryClient = useQueryClient();
  const { addToast } = useToast();
  const isMobile = useIsMobile();
  const searchInputRef = useRef<HTMLInputElement>(null);

  // Resizable sidebar (desktop only)
  const {
    width: sidebarWidth,
    isResizing: isSidebarResizing,
    containerRef: sidebarContainerRef,
    handleResizeStart,
  } = useResizableSidebar({
    storageKey: "agent-inbox:sidebar-width",
    defaultWidth: 320,
    minWidth: 200,
    maxWidthRatio: 0.5,
  });
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

  // Template deactivation callback (uses ref to break circular dependency with useChats/useActiveTemplate)
  const templateDeactivateRef = useRef<(() => void) | null>(null);
  const handleTemplateDeactivated = useCallback(() => {
    templateDeactivateRef.current?.();
  }, []);

  const {
    // State
    selectedChatId,
    currentView,
    isGenerating,
    streamingContent,
    activeToolCalls,
    generatedImages,
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

    // Edit operations
    editingMessage,
    setEditingMessage,
    editMessageAndComplete,
    cancelEdit,

    // Bulk operations
    bulkOperate,

    // Fork conversation
    forkConversation,

    // Mutation states
    isCreatingChat,
    isDeletingAllChats,
    isBulkOperating,
    isForking,
  } = useChats({
    initialChatId,
    onChatChange: handleChatChange,
    onTemplateDeactivated: handleTemplateDeactivated,
  });

  // Track async operations for the selected chat
  const {
    operations: asyncOperations,
    activeOperations: activeAsyncOperations,
    completedOperations: completedAsyncOperations,
    cancelOperation: cancelAsyncOperation,
    refreshOperation: refreshAsyncOperation,
    fetchHistory: fetchAsyncHistory,
  } = useAsyncStatus(selectedChatId);

  // Async result references for including in follow-up messages
  const [asyncReferences, setAsyncReferences] = useState<Array<{
    tool_call_id: string;
    tool_name: string;
    status: string;
    summary: string;
  }>>([]);

  // Track history pagination
  const [asyncHistoryOffset, setAsyncHistoryOffset] = useState(0);
  const [hasMoreAsyncHistory, setHasMoreAsyncHistory] = useState(true);

  // Handle inserting an async result reference
  const handleInsertAsyncReference = useCallback((op: AsyncStatusUpdate) => {
    const summarizeResult = (result: unknown, maxLength = 100): string => {
      if (result === null || result === undefined) return "No result data";
      if (typeof result === "string") {
        return result.length > maxLength ? result.slice(0, maxLength - 3) + "..." : result;
      }
      if (typeof result === "object") {
        const obj = result as Record<string, unknown>;
        if (typeof obj.message === "string") {
          return obj.message.length > maxLength ? obj.message.slice(0, maxLength - 3) + "..." : obj.message;
        }
        if (typeof obj.summary === "string") {
          return obj.summary.length > maxLength ? obj.summary.slice(0, maxLength - 3) + "..." : obj.summary;
        }
        if (Array.isArray(obj.files)) {
          return `Created ${obj.files.length} file${obj.files.length !== 1 ? "s" : ""}`;
        }
      }
      return "Result available";
    };

    setAsyncReferences((prev) => [
      ...prev.filter((r) => r.tool_call_id !== op.tool_call_id),
      {
        tool_call_id: op.tool_call_id,
        tool_name: op.tool_name,
        status: op.status,
        summary: summarizeResult(op.result, 100),
      },
    ]);
  }, []);

  // Handle removing an async result reference
  const handleRemoveAsyncReference = useCallback((toolCallId: string) => {
    setAsyncReferences((prev) => prev.filter((r) => r.tool_call_id !== toolCallId));
  }, []);

  // Handle fetching more async history
  const handleFetchAsyncHistory = useCallback(async () => {
    const result = await fetchAsyncHistory(20, asyncHistoryOffset);
    setAsyncHistoryOffset((prev) => prev + result.operations.length);
    setHasMoreAsyncHistory(result.hasMore);
  }, [fetchAsyncHistory, asyncHistoryOffset]);

  // Reset async references when chat changes
  useEffect(() => {
    setAsyncReferences([]);
    setAsyncHistoryOffset(0);
    setHasMoreAsyncHistory(true);
  }, [selectedChatId]);

  // Template-to-tool linking: manage active template state and tool enablement
  const { enableToolsByIds } = useTools({ chatId: selectedChatId ?? undefined });
  const activeTemplate = useActiveTemplate(selectedChatId ?? undefined, chatData?.chat);

  // Update the ref so the deactivation callback can use the activeTemplate
  // Guard: Only deactivate if chatData matches selectedChatId to prevent
  // deactivating the wrong chat during transitions
  //
  // CRITICAL: Depend on activeTemplate.deactivate specifically, NOT the whole
  // activeTemplate object. The object has properties like isUpdating that change,
  // which would cause this effect to run unnecessarily during critical transitions.
  useEffect(() => {
    templateDeactivateRef.current = () => {
      // Safety check: Only proceed if we have valid, matching chat data
      if (selectedChatId && chatData?.chat?.id === selectedChatId) {
        activeTemplate.deactivate();
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Depend on .deactivate specifically, NOT the whole activeTemplate object (see comment above)
  }, [activeTemplate.deactivate, selectedChatId, chatData?.chat?.id]);

  // Handle template activation (when user selects a template with suggested tools)
  // CRITICAL: Depend on activeTemplate.activate specifically, NOT the whole activeTemplate
  // object. The object has properties like isUpdating that change frequently, which would
  // cause this callback to get a new reference and cascade re-renders through ChatView.
  const handleTemplateActivated = useCallback(
    async (templateId: string, toolIds: string[]) => {
      if (!selectedChatId) return;
      // First enable the suggested tools
      await enableToolsByIds(toolIds);
      // Then activate the template at the chat level
      await activeTemplate.activate(templateId, toolIds);
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- Depend on .activate specifically, NOT the whole activeTemplate object (see comment above)
    [selectedChatId, enableToolsByIds, activeTemplate.activate]
  );

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
  }, [selectedChatId, setChatListOpen]);

  // Calculate unread counts for sidebar badges
  // Memoized to prevent creating new object on every render
  const chatCounts = useMemo(() => ({
    inbox: chats.filter((c) => !c.is_archived).length,
    starred: chats.filter((c) => c.is_starred).length,
    archived: chats.filter((c) => c.is_archived).length,
  }), [chats]);

  // Track which message to scroll to (from search results)
  const [scrollToMessageId, setScrollToMessageId] = useState<string | null>(null);

  const handleSelectChat = useCallback(
    (chatId: string, messageId?: string) => {
      selectChat(chatId);
      // Clear first, then set in next frame to force re-trigger even if same messageId
      setScrollToMessageId(null);
      requestAnimationFrame(() => {
        setScrollToMessageId(messageId || null);
      });
      // Close chat list on mobile when selecting
      if (window.innerWidth < 1024) {
        setChatListOpen(false);
      }
    },
    [selectChat, setChatListOpen]
  );

  const handleNewChat = useCallback(() => {
    createChat({});
    // Close sidebar on mobile when creating new chat
    if (window.innerWidth < 768) {
      setSidebarOpen(false);
    }
  }, [createChat]);

  const handleNewAgentChat = useCallback(() => {
    createChat({ chat_mode: "agent" });
    // Close sidebar on mobile when creating new chat
    if (window.innerWidth < 768) {
      setSidebarOpen(false);
    }
  }, [createChat]);

  // Handle starting an agent chat with message and config from EmptyState
  const handleStartAgentChat = useCallback(
    async (payload: MessagePayload, config: AgentStartConfig) => {
      const hasContent = payload.content.trim();
      if (!hasContent) return;

      let chatId: string | undefined;
      try {
        // Create chat in agent mode with default model
        const defaultModel = getDefaultModel();
        const newChat = await createChatAPI({ model: defaultModel, chat_mode: "agent" });
        chatId = newChat.id;

        // Select the new chat
        selectChat(chatId);

        // Start agent mode with the first message
        await startAgentMode(chatId, {
          message: payload.content.trim(),
          runner_type: config.runner_type,
          project_path: config.project_path,
          model: config.model || undefined,
          max_turns: config.max_turns || undefined,
        });

        // Refresh chat data to get updated agent state
        queryClient.invalidateQueries({ queryKey: ["chats"] });
        queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      } catch (error) {
        console.error("Failed to create agent chat:", error);
        // Clean up the partially-created chat so the user isn't left with a broken empty chat
        if (chatId) {
          try { await deleteChatAPI(chatId); } catch { /* best effort */ }
          selectChat("");
          queryClient.invalidateQueries({ queryKey: ["chats"] });
        }
        // Surface the error to the user
        const msg = error instanceof AgentModeError
          ? error.message
          : error instanceof Error ? error.message : "Failed to start agent chat";
        addToast(msg, "error", 8000);
      }
    },
    [selectChat, queryClient, addToast]
  );

  const handleBackToList = useCallback(() => {
    setChatListOpen(true);
  }, [setChatListOpen]);

  const handleOpenSettings = useCallback((tab: SettingsTab = "general") => {
    setSettingsInitialTab(tab);
    setShowSettings(true);
  }, []);

  const handleOpenAgentSettings = useCallback(() => {
    handleOpenSettings("agent");
  }, [handleOpenSettings]);

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

  const handleToggleSidebarCollapsed = useCallback(() => {
    setSidebarCollapsedState((prev) => {
      const newValue = !prev;
      setSidebarCollapsed(newValue);
      return newValue;
    });
  }, []);

  const handleClearArchived = useCallback(async () => {
    setIsClearingArchived(true);
    try {
      await deleteArchivedChats();
      // Invalidate chat queries to refresh the list
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    } finally {
      setIsClearingArchived(false);
    }
  }, [queryClient]);

  const handleMarkAllAsRead = useCallback(async () => {
    setIsMarkingAllAsRead(true);
    try {
      await markAllChatsAsRead();
      // Invalidate chat queries to refresh the read status
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    } finally {
      setIsMarkingAllAsRead(false);
    }
  }, [queryClient]);

  const handleEditTemplateFromSettings = useCallback((template: TemplateWithSource, allTemplates: TemplateWithSource[]) => {
    setSettingsEditingTemplate(template);
    setSettingsAllTemplates(allTemplates);
  }, []);

  const handleSaveTemplateFromSettings = useCallback(async (
    templateData: Omit<TemplateWithSource, "id" | "createdAt" | "updatedAt" | "isBuiltIn" | "source" | "hasDefault">,
    options?: { applyToDefault?: boolean }
  ) => {
    if (!settingsEditingTemplate) return;
    if (options?.applyToDefault) {
      await updateDefaultTemplateAPI(settingsEditingTemplate.id, templateData);
    } else {
      await updateTemplateAPI(settingsEditingTemplate.id, templateData);
    }
    setSettingsEditingTemplate(null);
  }, [settingsEditingTemplate]);

  const handleDeselectChat = useCallback(() => {
    selectChat("");
    if (window.innerWidth < 1024) {
      setSidebarOpen(false);
      setChatListOpen(false);
    }
  }, [selectChat, setChatListOpen]);

  // CRITICAL: Memoize ALL callback props passed to ChatView to prevent
  // creating new function references on every render. New references cause
  // unnecessary child re-renders during critical transitions (like message send)
  // which can contribute to "too many re-renders" errors.
  const handleScrollComplete = useCallback(() => {
    setScrollToMessageId(null);
  }, []);

  const handleUpdateChatFromView = useCallback(
    (data: Parameters<typeof updateChat>[0]["data"]) => {
      if (selectedChatId) {
        updateChat({ chatId: selectedChatId, data });
      }
    },
    [selectedChatId, updateChat]
  );

  const handleToggleReadFromView = useCallback(() => {
    if (selectedChatId) {
      toggleRead({ chatId: selectedChatId });
    }
  }, [selectedChatId, toggleRead]);

  const handleToggleStarFromView = useCallback(() => {
    if (selectedChatId) {
      toggleStar({ chatId: selectedChatId });
    }
  }, [selectedChatId, toggleStar]);

  const handleToggleArchiveFromView = useCallback(() => {
    if (selectedChatId) {
      toggleArchive({ chatId: selectedChatId });
    }
  }, [selectedChatId, toggleArchive]);

  const handleDeleteChatFromView = useCallback(() => {
    if (selectedChatId) {
      deleteChat(selectedChatId);
    }
  }, [selectedChatId, deleteChat]);

  const handleAssignLabelFromView = useCallback(
    (labelId: string) => {
      if (selectedChatId) {
        assignLabel({ chatId: selectedChatId, labelId });
      }
    },
    [selectedChatId, assignLabel]
  );

  const handleRemoveLabelFromView = useCallback(
    (labelId: string) => {
      if (selectedChatId) {
        removeLabel({ chatId: selectedChatId, labelId });
      }
    },
    [selectedChatId, removeLabel]
  );

  const handleRegenerateMessageFromView = useCallback(
    (messageId: string) => {
      if (selectedChatId) {
        regenerateMessage(selectedChatId, messageId);
      }
    },
    [selectedChatId, regenerateMessage]
  );

  const handleSubmitEditFromView = useCallback(
    (payload: Parameters<typeof editMessageAndComplete>[1]) => {
      if (editingMessage) {
        editMessageAndComplete(editingMessage.id, payload);
      }
    },
    [editingMessage, editMessageAndComplete]
  );

  // Handle refresh chat (used after agent mode changes)
  const handleRefreshChat = useCallback(() => {
    if (selectedChatId) {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    }
  }, [selectedChatId, queryClient]);

  const handleMobileExport = useCallback(async (format: "markdown" | "json" | "txt") => {
    if (!selectedChatId) return;
    try {
      await exportChat(selectedChatId, format);
    } catch (error) {
      console.error("Export failed:", error);
      addToast("Failed to export chat", "error");
    }
  }, [selectedChatId, addToast]);

  // Keyboard shortcuts
  const anyModalOpen = showLabelManager || showSettings || showKeyboardShortcuts || showUsageStats || !!settingsEditingTemplate;

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
      if (chat) {
        handleSelectChat(chat.id);
      }
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
        action: () => {
          const searchInput = searchInputRef.current;
          if (!searchInput) return false;
          if (document.activeElement === searchInput) {
            return false;
          }
          searchInput.focus();
          return true;
        },
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

  const handleUnhandledShortcut = useCallback((shortcut: KeyboardShortcut, event: KeyboardEvent) => {
    if (!shortcut.ctrlKey || shortcut.key.toLowerCase() !== "k") {
      return;
    }

    emitShortcutIntent({
      action: HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER,
      outcome: "noop",
      chord: "mod+k",
      source: "keyboard",
      detail: {
        key: event.key,
      },
    });
  }, []);

  useKeyboardShortcuts(shortcuts, {
    disabled: anyModalOpen && shortcuts.every(s => s.key !== "Escape"),
    onUnhandledShortcut: handleUnhandledShortcut,
  });

  const currentMobileChat = selectedChatId && !chatListOpen && chatData?.chat?.id === selectedChatId
    ? chatData.chat
    : null;
  const currentMobileLabelIds = currentMobileChat?.label_ids || [];
  const mobileAvailableLabels = labels.filter((label) => !currentMobileLabelIds.includes(label.id));

  return (
    <div ref={sidebarContainerRef as RefObject<HTMLDivElement>} className="h-screen bg-slate-950 text-slate-50 flex overflow-hidden" data-testid={appTestIds.container}>
      {/* Mobile Header */}
      <div className="lg:hidden fixed top-0 left-0 right-0 z-40 bg-slate-950 border-b border-white/10 px-2 py-1.5 flex items-center justify-between safe-top">
        <div className="flex items-center gap-1 min-w-0 flex-1">
          {selectedChatId && !chatListOpen ? (
            <Button
              variant="ghost"
              size="icon"
              onClick={handleBackToList}
              className="h-10 w-10 shrink-0"
              data-testid={appTestIds.mobileBackButton}
            >
              <ChevronLeft className="h-5 w-5" />
            </Button>
          ) : (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setSidebarOpen(true)}
              className="h-10 w-10 shrink-0"
              data-testid={appTestIds.mobileMenuButton}
            >
              <Menu className="h-5 w-5" />
            </Button>
          )}
          <span className="text-sm font-medium truncate">
            {selectedChatId && !chatListOpen
              ? chatData?.chat.name || "Chat"
              : "Agent Inbox"}
          </span>
          {currentMobileChat && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => {
                setMobileChatName(currentMobileChat.name || "");
                setShowMobileRenameDialog(true);
              }}
              className="h-8 w-8 shrink-0"
              aria-label="Rename chat"
            >
              <Edit3 className="h-3.5 w-3.5" />
            </Button>
          )}
        </div>
        {currentMobileChat && (
          <div className="flex items-center gap-1 shrink-0">
            <Dropdown
              trigger={
                <Button variant="ghost" size="icon" aria-label="Manage labels">
                  <Tag className="h-4 w-4" />
                </Button>
              }
              align="right"
            >
              <div className="p-2">
                {mobileAvailableLabels.length > 0 ? (
                  <>
                    <p className="text-xs text-slate-500 px-2 mb-1">Add a label</p>
                    {mobileAvailableLabels.map((label) => (
                      <DropdownItem key={label.id} onClick={() => handleAssignLabelFromView(label.id)}>
                        <span
                          className="w-2.5 h-2.5 rounded-full shrink-0"
                          style={{ backgroundColor: label.color }}
                        />
                        {label.name}
                      </DropdownItem>
                    ))}
                  </>
                ) : labels.length === 0 ? (
                  <p className="text-xs text-slate-500 px-2 py-2">No labels yet. Create labels from the sidebar.</p>
                ) : (
                  <p className="text-xs text-slate-500 px-2 py-2">All labels assigned</p>
                )}
              </div>
            </Dropdown>

            <Dropdown
              trigger={
                <Button variant="ghost" size="icon" aria-label="Chat actions">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              }
              align="right"
            >
              <DropdownItem onClick={handleToggleReadFromView}>
                {currentMobileChat.is_read ? <Mail className="h-4 w-4" /> : <MailOpen className="h-4 w-4" />}
                {currentMobileChat.is_read ? "Mark as unread" : "Mark as read"}
              </DropdownItem>
              <DropdownItem onClick={handleToggleStarFromView}>
                <Star className={`h-4 w-4 ${currentMobileChat.is_starred ? "text-yellow-500 fill-yellow-500" : ""}`} />
                {currentMobileChat.is_starred ? "Remove star" : "Star chat"}
              </DropdownItem>
              <DropdownItem onClick={handleToggleArchiveFromView}>
                <Archive className="h-4 w-4" />
                {currentMobileChat.is_archived ? "Unarchive" : "Archive"}
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem
                onClick={() => {
                  setMobileChatName(currentMobileChat.name || "");
                  setShowMobileRenameDialog(true);
                }}
              >
                <Edit3 className="h-4 w-4" />
                Rename chat
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem onClick={() => handleMobileExport("markdown")}>
                <FileText className="h-4 w-4 text-indigo-400" />
                Export as Markdown
              </DropdownItem>
              <DropdownItem onClick={() => handleMobileExport("json")}>
                <FileJson className="h-4 w-4 text-emerald-400" />
                Export as JSON
              </DropdownItem>
              <DropdownItem onClick={() => handleMobileExport("txt")}>
                <File className="h-4 w-4 text-slate-400" />
                Export as Text
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem destructive onClick={handleDeleteChatFromView}>
                <Trash2 className="h-4 w-4" />
                Delete chat
              </DropdownItem>
            </Dropdown>
          </div>
        )}
      </div>

      {/* Mobile Sidebar Overlay */}
      {sidebarOpen && (
        <div
          className="lg:hidden fixed inset-0 z-50 bg-black/60 backdrop-blur-sm"
          onClick={() => setSidebarOpen(false)}
          data-testid={appTestIds.mobileSidebarOverlay}
        />
      )}

      {/* Unified Sidebar - Desktop: always visible, Mobile: slide-in */}
      <div
        className={`fixed lg:relative inset-y-0 left-0 z-50 lg:z-auto transform transition-transform duration-200 ${
          sidebarOpen || chatListOpen ? "translate-x-0" : "-translate-x-full lg:translate-x-0"
        } pt-14 lg:pt-0`}
        style={
          !isMobile && !(sidebarCollapsed && !isMobile)
            ? { width: sidebarWidth }
            : undefined
        }
      >
        <div className="lg:hidden absolute top-3 right-3 z-10">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => {
              setSidebarOpen(false);
              setChatListOpen(false);
            }}
            data-testid={appTestIds.closeSidebarButton}
          >
            <X className="h-5 w-5" />
          </Button>
        </div>
        <ErrorBoundary name="Sidebar">
          <Sidebar
            ref={searchInputRef}
            currentView={currentView}
            onViewChange={(view) => {
              setCurrentView(view);
              if (window.innerWidth < 768) {
                setSidebarOpen(false);
              }
            }}
            onNewChat={handleNewChat}
            onNewAgentChat={handleNewAgentChat}
            onManageLabels={() => setShowLabelManager(true)}
            onOpenSettings={handleOpenSettings}
            onShowKeyboardShortcuts={handleShowKeyboardShortcuts}
            isCreatingChat={isCreatingChat}
            labels={labels}
            chatCounts={chatCounts}
            chats={chats}
            selectedChatId={selectedChatId}
            focusedIndex={focusedIndex}
            isLoadingChats={loadingChats}
            onSelectChat={handleSelectChat}
            onRenameChat={(chatId, newName) => updateChat({ chatId, data: { name: newName } })}
            onBulkOperate={(chatIds, operation, labelId) => bulkOperate({ chatIds, operation, labelId })}
            isBulkOperating={isBulkOperating}
            isCollapsed={sidebarCollapsed && !isMobile}
            onToggleCollapsed={handleToggleSidebarCollapsed}
            onClearArchived={handleClearArchived}
            isClearingArchived={isClearingArchived}
            onDeselectChat={handleDeselectChat}
          />
        </ErrorBoundary>
      </div>

      {/* Resize Handle - Desktop only, hidden when sidebar is collapsed */}
      {!isMobile && !(sidebarCollapsed && !isMobile) && (
        <div
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize sidebar"
          className={`hidden lg:flex items-center justify-center w-3 cursor-col-resize group shrink-0 ${
            isSidebarResizing ? "bg-indigo-500/10" : "hover:bg-white/5"
          }`}
          onMouseDown={handleResizeStart}
        >
          <div
            className={`w-px h-8 rounded-full transition-colors ${
              isSidebarResizing ? "bg-indigo-500" : "bg-white/20 group-hover:bg-white/40"
            }`}
          />
        </div>
      )}

      {/* Main Content - Chat View or Empty State */}
      <div
        className={`flex-1 flex flex-col min-h-0 min-w-0 pt-14 lg:pt-0 ${
          chatListOpen && selectedChatId ? "hidden lg:flex" : "flex"
        }`}
      >
        <ErrorBoundary name="ChatContent">
          {/* Guard: Only render ChatView when chatData matches selectedChatId to prevent stale data issues */}
          {selectedChatId ? (
            <Suspense fallback={(
              <div className="flex-1 flex items-center justify-center bg-slate-950">
                <span className="text-sm text-slate-400">Loading chat...</span>
              </div>
            )}>
              <LazyChatView
              key={selectedChatId}
              chatData={chatData?.chat?.id === selectedChatId ? chatData : null}
              models={models}
              labels={labels}
              isLoading={loadingChat || (!!selectedChatId && chatData?.chat?.id !== selectedChatId)}
              isGenerating={isGenerating}
              streamingContent={streamingContent}
              activeToolCalls={activeToolCalls}
              generatedImages={generatedImages}
              scrollToMessageId={scrollToMessageId}
              onScrollComplete={handleScrollComplete}
              onSendMessage={sendMessage}
              onUpdateChat={handleUpdateChatFromView}
              onToggleRead={handleToggleReadFromView}
              onToggleStar={handleToggleStarFromView}
              onToggleArchive={handleToggleArchiveFromView}
              onDeleteChat={handleDeleteChatFromView}
              onAssignLabel={handleAssignLabelFromView}
              onRemoveLabel={handleRemoveLabelFromView}
              viewMode={viewMode}
              onRegenerateMessage={handleRegenerateMessageFromView}
              onSelectBranch={selectBranch}
              onForkConversation={forkConversation}
              isRegenerating={isRegenerating}
              isForking={isForking}
              editingMessage={editingMessage}
              onEditMessage={setEditingMessage}
              onCancelEdit={cancelEdit}
              onSubmitEdit={handleSubmitEditFromView}
              asyncOperations={asyncOperations}
              activeAsyncOperations={activeAsyncOperations}
              completedAsyncOperations={completedAsyncOperations}
              onCancelAsyncOperation={cancelAsyncOperation}
              onRefreshAsyncOperation={refreshAsyncOperation}
              onFetchAsyncHistory={handleFetchAsyncHistory}
              hasMoreAsyncHistory={hasMoreAsyncHistory}
              asyncReferences={asyncReferences}
              onInsertAsyncReference={handleInsertAsyncReference}
              onRemoveAsyncReference={handleRemoveAsyncReference}
              onTemplateActivated={handleTemplateActivated}
              activeTemplateId={activeTemplate.activeTemplateId}
              onTemplateDeactivate={activeTemplate.deactivate}
              onRefreshChat={handleRefreshChat}
              onOpenAgentSettings={handleOpenAgentSettings}
            />
            </Suspense>
          ) : (
            <EmptyState
              onStartChat={createChatWithMessage}
              onStartAgentChat={handleStartAgentChat}
              onOpenAgentSettings={handleOpenAgentSettings}
              isCreating={isCreatingChat}
              models={models}
            />
          )}
        </ErrorBoundary>
      </div>

      {/* Label Manager Dialog */}
      <ErrorBoundary name="LabelManager">
        <LabelManager
          open={showLabelManager}
          onClose={() => setShowLabelManager(false)}
          labels={labels}
          onCreateLabel={createLabel}
          onDeleteLabel={deleteLabel}
        />
      </ErrorBoundary>

      <Dialog open={showMobileRenameDialog} onClose={() => setShowMobileRenameDialog(false)}>
        <DialogHeader onClose={() => setShowMobileRenameDialog(false)}>Rename Chat</DialogHeader>
        <DialogBody>
          <Input
            value={mobileChatName}
            onChange={(e) => setMobileChatName(e.target.value)}
            placeholder="Enter chat name..."
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter" && selectedChatId && mobileChatName.trim()) {
                updateChat({ chatId: selectedChatId, data: { name: mobileChatName.trim() } });
                setShowMobileRenameDialog(false);
              }
            }}
          />
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setShowMobileRenameDialog(false)}>
            Cancel
          </Button>
          <Button
            onClick={() => {
              if (!selectedChatId || !mobileChatName.trim()) return;
              updateChat({ chatId: selectedChatId, data: { name: mobileChatName.trim() } });
              setShowMobileRenameDialog(false);
            }}
            disabled={!mobileChatName.trim() || !selectedChatId}
          >
            Save
          </Button>
        </DialogFooter>
      </Dialog>

      {/* Settings Dialog */}
      <ErrorBoundary name="Settings">
        <Settings
          open={showSettings}
          initialTab={settingsInitialTab}
          onClose={() => setShowSettings(false)}
          onDeleteAllChats={deleteAllChats}
          isDeletingAll={isDeletingAllChats}
          onClearArchived={handleClearArchived}
          isClearingArchived={isClearingArchived}
          onMarkAllAsRead={handleMarkAllAsRead}
          isMarkingAllAsRead={isMarkingAllAsRead}
          onShowKeyboardShortcuts={handleShowKeyboardShortcuts}
          onShowUsageStats={handleShowUsageStats}
          models={models}
          viewMode={viewMode}
          onViewModeChange={handleViewModeChange}
          onEditTemplate={handleEditTemplateFromSettings}
        />
      </ErrorBoundary>

      {/* Template Editor from Settings - Only render when open to avoid useTools cascade */}
      {!!settingsEditingTemplate && (
        <ErrorBoundary name="TemplateEditor">
          <TemplateEditorModal
            open={!!settingsEditingTemplate}
            onClose={() => {
              setSettingsEditingTemplate(null);
              setSettingsAllTemplates([]);
            }}
            onSave={handleSaveTemplateFromSettings}
            template={settingsEditingTemplate || undefined}
            templateSource={settingsEditingTemplate?.source}
            allTemplates={settingsAllTemplates}
            onSelectTemplate={(template) => {
              setSettingsEditingTemplate(template);
            }}
            onSaveAll={async (updates) => {
              const { updateTemplates, getAllTemplates } = await import("./data/templates");
              await updateTemplates(updates);
              const updated = await getAllTemplates();
              setSettingsAllTemplates(updated);
            }}
          />
        </ErrorBoundary>
      )}

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

/** Scenario viewer wrapper component */
function ScenarioViewerWrapper() {
  const { scenarioName, path } = useScenarioViewerRoute();

  const handleBack = useCallback(() => {
    window.history.back();
  }, []);

  if (!scenarioName) {
    return null;
  }

  return (
    <ScenarioViewer
      scenarioName={scenarioName}
      path={path ?? undefined}
      onBack={handleBack}
    />
  );
}

/** App router - decides which view to render based on URL */
function AppRouter() {
  const { isScenarioViewer } = useScenarioViewerRoute();

  if (isScenarioViewer) {
    return (
      <ErrorBoundary name="ScenarioViewer">
        <ScenarioViewerWrapper />
      </ErrorBoundary>
    );
  }

  return <AppContent />;
}

export default function App() {
  return (
    <ToastProvider>
      <AppRouter />
    </ToastProvider>
  );
}
