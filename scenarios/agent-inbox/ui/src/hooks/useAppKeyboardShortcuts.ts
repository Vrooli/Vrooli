import { useMemo, useCallback, useState, useEffect, type RefObject } from "react";
import { emitShortcutIntent, HOST_SHORTCUT_ACTION_OPEN_GLOBAL_SWITCHER } from "@vrooli/iframe-bridge";
import { useKeyboardShortcuts, type KeyboardShortcut } from "./useKeyboardShortcuts";
import type { View } from "./useChats";
import type { SettingsTab } from "../components/settings/settingsTypes";

interface UseAppKeyboardShortcutsParams {
  searchInputRef: RefObject<HTMLInputElement | null>;
  visibleChats: Array<{ id: string }>;
  currentView: string;
  selectedChatId: string | null;
  createChat: (opts: Record<string, unknown>) => void;
  setCurrentView: (view: View) => void;
  handleOpenSettings: (tab?: SettingsTab) => void;
  handleShowKeyboardShortcuts: () => void;
  handleDeselectChat: () => void;
  handleSelectChat: (chatId: string) => void;
  toggleStar: (args: { chatId: string }) => void;
  toggleArchive: (args: { chatId: string }) => void;
  showLabelManager: boolean;
  showSettings: boolean;
  showKeyboardShortcuts: boolean;
  showUsageStats: boolean;
  setShowLabelManager: (show: boolean) => void;
  setShowSettings: (show: boolean) => void;
  setShowKeyboardShortcuts: (show: boolean) => void;
  setShowUsageStats: (show: boolean) => void;
  anyModalOpen: boolean;
}

export function useAppKeyboardShortcuts({
  searchInputRef,
  visibleChats,
  currentView,
  selectedChatId,
  createChat,
  setCurrentView,
  handleOpenSettings,
  handleShowKeyboardShortcuts,
  handleDeselectChat,
  handleSelectChat,
  toggleStar,
  toggleArchive,
  showLabelManager,
  showSettings,
  showKeyboardShortcuts,
  showUsageStats,
  setShowLabelManager,
  setShowSettings,
  setShowKeyboardShortcuts,
  setShowUsageStats,
  anyModalOpen,
}: UseAppKeyboardShortcutsParams) {
  // Focused chat index for j/k navigation (separate from selected chat)
  const [focusedIndex, setFocusedIndex] = useState<number>(-1);

  // Reset focused index when view changes
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
      searchInputRef,
      setShowLabelManager,
      setShowSettings,
      setShowKeyboardShortcuts,
      setShowUsageStats,
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

  return { focusedIndex };
}
