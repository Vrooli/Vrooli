import { useState, useEffect, useCallback } from "react";

/**
 * Hook for URL-based chat routing.
 * - Updates URL when a chat is selected: /chat/:chatId
 * - Reads chat ID from URL on page load
 * - Handles browser back/forward navigation
 */
export function useChatRoute() {
  // Parse chat ID from current URL
  const getChatIdFromUrl = useCallback((): string => {
    const path = window.location.pathname;
    const match = path.match(/^\/chat\/([a-zA-Z0-9-]+)$/);
    return match ? match[1] : "";
  }, []);

  const [initialChatId] = useState<string>(getChatIdFromUrl);

  // Update URL when chat is selected
  const setChatInUrl = useCallback((chatId: string) => {
    const currentPath = window.location.pathname;
    const newPath = chatId ? `/chat/${chatId}` : "/";

    // Only update if path actually changed
    if (currentPath !== newPath) {
      window.history.pushState({ chatId }, "", newPath);
    }
  }, []);

  // Replace URL without adding to history (for initial load or corrections)
  const replaceChatInUrl = useCallback((chatId: string) => {
    const currentPath = window.location.pathname;
    const newPath = chatId ? `/chat/${chatId}` : "/";

    if (currentPath !== newPath) {
      window.history.replaceState({ chatId }, "", newPath);
    }
  }, []);

  return {
    initialChatId,
    setChatInUrl,
    replaceChatInUrl,
    getChatIdFromUrl,
  };
}

/**
 * Hook to listen for browser back/forward navigation.
 * Calls the callback with the chat ID from the URL when navigation occurs.
 */
export function usePopStateListener(onNavigate: (chatId: string) => void) {
  useEffect(() => {
    const handlePopState = () => {
      const path = window.location.pathname;
      const match = path.match(/^\/chat\/([a-zA-Z0-9-]+)$/);
      const chatId = match ? match[1] : "";
      onNavigate(chatId);
    };

    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, [onNavigate]);
}
