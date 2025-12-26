/**
 * useChats - Main orchestration hook for chat management.
 *
 * This hook combines:
 * - Chat list fetching and filtering by view
 * - Single chat selection and operations
 * - Completion orchestration (delegates to useCompletion)
 * - Label operations (delegates to useLabels)
 *
 * ARCHITECTURE NOTE:
 * This hook serves as the main entry point for chat functionality.
 * Complex sub-features are extracted to focused hooks:
 * - useCompletion: AI streaming and tool calls
 * - useLabels: Label CRUD and chat-label associations
 *
 * SEAM: For testing, mock the individual hooks or the API functions.
 */
import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchChats,
  fetchChat,
  fetchModels,
  createChat,
  deleteChat,
  deleteAllChats,
  updateChat,
  addMessage,
  toggleRead,
  toggleArchive,
  toggleStar,
  autoNameChat,
} from "../lib/api";
import { useCompletion, type ActiveToolCall } from "./useCompletion";
import { useLabels } from "./useLabels";
import { getDefaultModel } from "../components/settings/Settings";

export type View = "inbox" | "starred" | "archived";

// Re-export for convenience
export type { ActiveToolCall };

export function useChats() {
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [currentView, setCurrentView] = useState<View>("inbox");

  // Delegate to focused hooks
  const completion = useCompletion();
  const labelOps = useLabels();

  // Fetch chats based on current view
  const {
    data: chats = [],
    isLoading: loadingChats,
    error: chatsError,
  } = useQuery({
    queryKey: ["chats", currentView],
    queryFn: () =>
      fetchChats({
        archived: currentView === "archived",
        starred: currentView === "starred",
      }),
    refetchInterval: 10000,
  });

  // Fetch selected chat with messages
  const {
    data: chatData,
    isLoading: loadingChat,
    error: chatError,
  } = useQuery({
    queryKey: ["chat", selectedChatId],
    queryFn: () => (selectedChatId ? fetchChat(selectedChatId) : null),
    enabled: !!selectedChatId,
  });

  // Fetch available models
  const { data: models = [] } = useQuery({
    queryKey: ["models"],
    queryFn: fetchModels,
  });

  // Chat mutations
  const createChatMutation = useMutation({
    mutationFn: (params: Parameters<typeof createChat>[0] = {}) => {
      // Use default model if not specified
      if (!params.model) {
        params.model = getDefaultModel();
      }
      return createChat(params);
    },
    onSuccess: (newChat) => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(newChat.id);
    },
  });

  const deleteChatMutation = useMutation({
    mutationFn: deleteChat,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(null);
    },
  });

  const deleteAllChatsMutation = useMutation({
    mutationFn: deleteAllChats,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      queryClient.invalidateQueries({ queryKey: ["chat"] });
      setSelectedChatId(null);
    },
  });

  const updateChatMutation = useMutation({
    mutationFn: ({ chatId, data }: { chatId: string; data: { name?: string; model?: string } }) =>
      updateChat(chatId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const toggleReadMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleRead(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    },
  });

  const toggleArchiveMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleArchive(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      if (currentView === "inbox" || currentView === "archived") {
        setSelectedChatId(null);
      }
    },
  });

  const toggleStarMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleStar(chatId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const autoNameChatMutation = useMutation({
    mutationFn: (chatId: string) => autoNameChat(chatId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  // Send message and run completion
  const sendMessageAndComplete = useCallback(
    async (chatId: string, content: string, needsAutoName: boolean) => {
      // Add user message
      await addMessage(chatId, { role: "user", content: content.trim() });
      queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });

      // Run AI completion
      try {
        await completion.runCompletion(chatId);
        queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
        queryClient.invalidateQueries({ queryKey: ["chats"] });

        // Auto-name if needed
        if (needsAutoName) {
          try {
            await autoNameChat(chatId);
            queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
            queryClient.invalidateQueries({ queryKey: ["chats"] });
          } catch (e) {
            console.error("Auto-naming failed:", e);
          }
        }
      } catch (error) {
        console.error("Chat completion failed:", error);
      }
    },
    [queryClient, completion]
  );

  // Create a new chat and immediately send a message
  const createChatWithMessage = useCallback(
    async (content: string) => {
      if (!content.trim() || completion.isGenerating) return;

      try {
        const defaultModel = getDefaultModel();
        const newChat = await createChat({ model: defaultModel });
        const chatId = newChat.id;

        setSelectedChatId(chatId);
        queryClient.invalidateQueries({ queryKey: ["chats"] });

        await sendMessageAndComplete(chatId, content, true);
      } catch (error) {
        console.error("Failed to create chat with message:", error);
      }
    },
    [completion.isGenerating, queryClient, sendMessageAndComplete]
  );

  // Send message to existing chat
  const sendMessage = useCallback(
    async (content: string) => {
      if (!selectedChatId || !content.trim() || completion.isGenerating) return;

      const currentChat = chats.find((c) => c.id === selectedChatId);
      const needsAutoName = currentChat?.name === "New Chat";

      await sendMessageAndComplete(selectedChatId, content, needsAutoName);
    },
    [selectedChatId, completion.isGenerating, chats, sendMessageAndComplete]
  );

  // Select chat and mark as read
  const selectChat = useCallback(
    (chatId: string) => {
      setSelectedChatId(chatId);
      const chat = chats.find((c) => c.id === chatId);
      if (chat && !chat.is_read) {
        toggleReadMutation.mutate({ chatId, value: true });
      }
    },
    [chats, toggleReadMutation]
  );

  return {
    // State
    selectedChatId,
    currentView,
    isGenerating: completion.isGenerating,
    streamingContent: completion.streamingContent,
    activeToolCalls: completion.activeToolCalls,

    // Data
    chats,
    chatData,
    models,
    labels: labelOps.labels,

    // Loading states
    loadingChats,
    loadingChat,

    // Errors
    chatsError,
    chatError,

    // Actions
    setCurrentView,
    selectChat,
    sendMessage,
    createChatWithMessage,

    // Chat mutations
    createChat: createChatMutation.mutate,
    deleteChat: deleteChatMutation.mutate,
    deleteAllChats: deleteAllChatsMutation.mutateAsync,
    updateChat: updateChatMutation.mutate,
    toggleRead: toggleReadMutation.mutate,
    toggleArchive: toggleArchiveMutation.mutate,
    toggleStar: toggleStarMutation.mutate,
    autoNameChat: autoNameChatMutation.mutate,

    // Label operations (delegated)
    createLabel: labelOps.createLabel,
    deleteLabel: labelOps.deleteLabel,
    assignLabel: labelOps.assignLabel,
    removeLabel: labelOps.removeLabel,

    // Mutation states
    isCreatingChat: createChatMutation.isPending,
    isDeletingChat: deleteChatMutation.isPending,
    isDeletingAllChats: deleteAllChatsMutation.isPending,
    isUpdatingChat: updateChatMutation.isPending,
    isAutoNaming: autoNameChatMutation.isPending,
  };
}
