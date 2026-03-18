import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import type { MessagePayload } from "../components/chat/MessageInput";

/**
 * Memoized callback props for ChatView to prevent creating new function
 * references on every render, which would cause unnecessary child re-renders.
 */
export function useChatViewCallbacks({
  selectedChatId,
  updateChat,
  toggleRead,
  toggleStar,
  toggleArchive,
  deleteChat,
  assignLabel,
  removeLabel,
  regenerateMessage,
  editingMessage,
  editMessageAndComplete,
  setScrollToMessageId,
}: {
  selectedChatId: string | null;
  updateChat: (args: { chatId: string; data: Record<string, unknown> }) => void;
  toggleRead: (args: { chatId: string }) => void;
  toggleStar: (args: { chatId: string }) => void;
  toggleArchive: (args: { chatId: string }) => void;
  deleteChat: (chatId: string) => void;
  assignLabel: (args: { chatId: string; labelId: string }) => void;
  removeLabel: (args: { chatId: string; labelId: string }) => void;
  regenerateMessage: (chatId: string, messageId: string) => void;
  editingMessage: { id: string } | null;
  editMessageAndComplete: (messageId: string, payload: MessagePayload) => void;
  setScrollToMessageId: (id: string | null) => void;
}) {
  const queryClient = useQueryClient();

  const handleScrollComplete = useCallback(() => {
    setScrollToMessageId(null);
  }, [setScrollToMessageId]);

  const handleUpdateChatFromView = useCallback(
    (data: Record<string, unknown>) => {
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
    (payload: MessagePayload) => {
      if (editingMessage) {
        editMessageAndComplete(editingMessage.id, payload);
      }
    },
    [editingMessage, editMessageAndComplete]
  );

  const handleRefreshChat = useCallback(() => {
    if (selectedChatId) {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    }
  }, [selectedChatId, queryClient]);

  return {
    handleScrollComplete,
    handleUpdateChatFromView,
    handleToggleReadFromView,
    handleToggleStarFromView,
    handleToggleArchiveFromView,
    handleDeleteChatFromView,
    handleAssignLabelFromView,
    handleRemoveLabelFromView,
    handleRegenerateMessageFromView,
    handleSubmitEditFromView,
    handleRefreshChat,
  };
}
