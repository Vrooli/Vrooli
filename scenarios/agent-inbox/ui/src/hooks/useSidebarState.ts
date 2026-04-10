import { useState, useCallback } from "react";

// Sidebar collapsed state persistence (desktop)
const SIDEBAR_COLLAPSED_KEY = "agent-inbox:sidebar-collapsed";

function getSidebarCollapsed(): boolean {
  if (typeof window !== "undefined") {
    return localStorage.getItem(SIDEBAR_COLLAPSED_KEY) === "true";
  }
  return false;
}

function setSidebarCollapsedStorage(collapsed: boolean): void {
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

export interface SidebarState {
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  chatListOpen: boolean;
  setChatListOpen: (open: boolean) => void;
  sidebarCollapsed: boolean;
  handleToggleSidebarCollapsed: () => void;
}

export function useSidebarState(): SidebarState {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [chatListOpen, setChatListOpenState] = useState(getChatListOpen);
  const [sidebarCollapsed, setSidebarCollapsedState] = useState(getSidebarCollapsed);

  // Wrapper to persist chat list state to localStorage
  const setChatListOpen = useCallback((open: boolean) => {
    setChatListOpenState(open);
    setChatListOpenStorage(open);
  }, []);

  const handleToggleSidebarCollapsed = useCallback(() => {
    setSidebarCollapsedState((prev) => {
      const newValue = !prev;
      setSidebarCollapsedStorage(newValue);
      return newValue;
    });
  }, []);

  return {
    sidebarOpen,
    setSidebarOpen,
    chatListOpen,
    setChatListOpen,
    sidebarCollapsed,
    handleToggleSidebarCollapsed,
  };
}
