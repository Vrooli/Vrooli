import { useEffect, useRef, useState } from "react";
import type { Message } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useCompletion";

interface UseScrollManagementOptions {
  messages: Message[];
  streamingContent: string;
  activeToolCalls: ActiveToolCall[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
}

export function useScrollManagement({
  messages,
  streamingContent,
  activeToolCalls,
  scrollToMessageId,
  onScrollComplete,
}: UseScrollManagementOptions) {
  const endRef = useRef<HTMLDivElement>(null);
  const messageRefs = useRef<Map<string, HTMLDivElement>>(new Map());

  // Track which message is highlighted (from search result navigation)
  // Using React state instead of classList.add/remove because Tailwind CSS
  // purges dynamically-added class names that don't appear in source files.
  const [highlightedMessageId, setHighlightedMessageId] = useState<string | null>(null);

  // Ref for debouncing auto-scroll to prevent rapid scroll triggers during streaming
  const scrollTimeoutRef = useRef<number | null>(null);

  // Clear stale message refs when messages change to prevent memory leaks
  // and stale data issues when switching between chats
  useEffect(() => {
    const currentMessageIds = new Set(messages.map((m) => m.id));
    // Remove refs for messages that are no longer in the list
    for (const id of messageRefs.current.keys()) {
      if (!currentMessageIds.has(id)) {
        messageRefs.current.delete(id);
      }
    }
  }, [messages]);

  // Scroll to specific message when navigating from search.
  // Uses retry polling because when navigating to a new chat, messages load async
  // and the target DOM element may not exist yet on first attempt.
  useEffect(() => {
    if (!scrollToMessageId) return;

    let attempts = 0;
    const maxAttempts = 30; // ~3 seconds
    let timerId: number;
    let highlightTimerId: number;

    const tryScroll = () => {
      const messageEl = messageRefs.current.get(scrollToMessageId);
      if (messageEl) {
        messageEl.scrollIntoView({ behavior: "smooth", block: "center" });
        setHighlightedMessageId(scrollToMessageId);
        highlightTimerId = window.setTimeout(() => {
          setHighlightedMessageId(null);
          onScrollComplete?.();
        }, 2000);
        return;
      }
      attempts++;
      if (attempts < maxAttempts) {
        timerId = window.setTimeout(tryScroll, 100);
      } else {
        onScrollComplete?.();
      }
    };

    // Try immediately first (element may already be in DOM for same-chat scrolls),
    // then retry with delay for cross-chat navigation where messages load async.
    tryScroll();
    return () => {
      clearTimeout(timerId);
      clearTimeout(highlightTimerId);
    };
  }, [scrollToMessageId, onScrollComplete]);

  // Auto-scroll to end for new messages/streaming
  // DEBOUNCED: Use timeout to prevent rapid scroll triggers during streaming which can
  // interact with browser reflow and cause React reconciliation issues.
  // Uses stable primitive dependencies instead of array references to prevent
  // unnecessary effect runs.
  const hasStreamingContent = Boolean(streamingContent);
  useEffect(() => {
    if (!scrollToMessageId) {
      // Clear any pending scroll
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
      // Debounce the scroll with a small delay
      scrollTimeoutRef.current = window.setTimeout(() => {
        endRef.current?.scrollIntoView({ behavior: "smooth" });
      }, 50);
    }
    return () => {
      if (scrollTimeoutRef.current) {
        clearTimeout(scrollTimeoutRef.current);
      }
    };
  }, [messages.length, hasStreamingContent, activeToolCalls.length, scrollToMessageId]);

  return {
    endRef,
    messageRefs,
    highlightedMessageId,
  };
}
