import { useState, useCallback, useRef, useEffect } from "react";
import type { Chat, BulkOperation } from "./types";

interface UseBulkSelectionOptions {
  displayChats: Chat[];
  selectedChatId: string | null;
  currentView: string;
  onBulkOperate?: (chatIds: string[], operation: BulkOperation, labelId?: string) => void;
}

export function useBulkSelection({
  displayChats,
  selectedChatId,
  currentView,
  onBulkOperate,
}: UseBulkSelectionOptions) {
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedChatIds, setSelectedChatIds] = useState<Set<string>>(new Set());
  const lastSelectedIndexRef = useRef<number>(-1);

  // Exit selection mode when view changes
  useEffect(() => {
    setSelectionMode(false);
    setSelectedChatIds(new Set());
    lastSelectedIndexRef.current = -1;
  }, [currentView]);

  // Update anchor when user navigates to a chat (for shift+click to work from viewed chat)
  useEffect(() => {
    if (selectedChatId && !selectionMode) {
      const idx = displayChats.findIndex((c) => c.id === selectedChatId);
      if (idx !== -1) {
        lastSelectedIndexRef.current = idx;
      }
    }
  }, [selectedChatId, displayChats, selectionMode]);

  // Toggle selection for a chat with shift+click support
  const toggleChatSelection = useCallback(
    (chatId: string, index: number, event: React.MouseEvent) => {
      event.stopPropagation();

      // Always enter selection mode
      setSelectionMode(true);

      if (event.shiftKey) {
        let anchorIndex = lastSelectedIndexRef.current;
        if (anchorIndex === -1 && selectedChatId) {
          anchorIndex = displayChats.findIndex((c) => c.id === selectedChatId);
        }
        if (anchorIndex === -1) {
          anchorIndex = 0;
        }

        const start = Math.min(anchorIndex, index);
        const end = Math.max(anchorIndex, index);
        const rangeIds = displayChats.slice(start, end + 1).map((c) => c.id);

        setSelectedChatIds((prev) => new Set([...prev, ...rangeIds]));
      } else {
        setSelectedChatIds((prev) => {
          const next = new Set(prev);
          if (next.has(chatId)) {
            next.delete(chatId);
          } else {
            next.add(chatId);
          }
          return next;
        });
      }

      lastSelectedIndexRef.current = index;
    },
    [displayChats, selectedChatId]
  );

  const selectAll = useCallback(() => {
    setSelectedChatIds(new Set(displayChats.map((c) => c.id)));
  }, [displayChats]);

  const deselectAll = useCallback(() => {
    setSelectedChatIds(new Set());
  }, []);

  const exitSelectionMode = useCallback(() => {
    setSelectionMode(false);
    setSelectedChatIds(new Set());
  }, []);

  const handleBulkOperation = useCallback(
    (operation: BulkOperation) => {
      if (!onBulkOperate || selectedChatIds.size === 0) return;
      onBulkOperate(Array.from(selectedChatIds), operation);
      exitSelectionMode();
    },
    [onBulkOperate, selectedChatIds, exitSelectionMode]
  );

  return {
    selectionMode,
    setSelectionMode,
    selectedChatIds,
    toggleChatSelection,
    selectAll,
    deselectAll,
    exitSelectionMode,
    handleBulkOperation,
  };
}
