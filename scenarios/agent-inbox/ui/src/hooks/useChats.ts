import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchChats,
  fetchChat,
  fetchModels,
  fetchLabels,
  createChat,
  deleteChat,
  updateChat,
  addMessage,
  completeChat,
  toggleRead,
  toggleArchive,
  toggleStar,
  createLabel,
  deleteLabel,
  assignLabel,
  removeLabel,
} from "../lib/api";

export type View = "inbox" | "starred" | "archived";

export function useChats() {
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(null);
  const [currentView, setCurrentView] = useState<View>("inbox");
  const [isGenerating, setIsGenerating] = useState(false);
  const [streamingContent, setStreamingContent] = useState("");

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

  // Fetch labels
  const { data: labels = [] } = useQuery({
    queryKey: ["labels"],
    queryFn: fetchLabels,
  });

  // Mutations
  const createChatMutation = useMutation({
    mutationFn: createChat,
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

  // Label mutations
  const createLabelMutation = useMutation({
    mutationFn: createLabel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["labels"] });
    },
  });

  const deleteLabelMutation = useMutation({
    mutationFn: deleteLabel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["labels"] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const assignLabelMutation = useMutation({
    mutationFn: ({ chatId, labelId }: { chatId: string; labelId: string }) => assignLabel(chatId, labelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const removeLabelMutation = useMutation({
    mutationFn: ({ chatId, labelId }: { chatId: string; labelId: string }) => removeLabel(chatId, labelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const sendMessage = useCallback(
    async (content: string) => {
      if (!selectedChatId || !content.trim() || isGenerating) return;

      // Add user message
      await addMessage(selectedChatId, { role: "user", content: content.trim() });
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });

      // Start AI completion with streaming
      setIsGenerating(true);
      setStreamingContent("");

      try {
        await completeChat(selectedChatId, {
          stream: true,
          onChunk: (chunk) => {
            setStreamingContent((prev) => prev + chunk);
          },
        });

        queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
        queryClient.invalidateQueries({ queryKey: ["chats"] });
      } catch (error) {
        console.error("Chat completion failed:", error);
      } finally {
        setIsGenerating(false);
        setStreamingContent("");
      }
    },
    [selectedChatId, isGenerating, queryClient]
  );

  const selectChat = useCallback(
    (chatId: string) => {
      setSelectedChatId(chatId);
      // Mark as read when selecting
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
    isGenerating,
    streamingContent,

    // Data
    chats,
    chatData,
    models,
    labels,

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

    // Mutations
    createChat: createChatMutation.mutate,
    deleteChat: deleteChatMutation.mutate,
    updateChat: updateChatMutation.mutate,
    toggleRead: toggleReadMutation.mutate,
    toggleArchive: toggleArchiveMutation.mutate,
    toggleStar: toggleStarMutation.mutate,
    createLabel: createLabelMutation.mutate,
    deleteLabel: deleteLabelMutation.mutate,
    assignLabel: assignLabelMutation.mutate,
    removeLabel: removeLabelMutation.mutate,

    // Mutation states
    isCreatingChat: createChatMutation.isPending,
    isDeletingChat: deleteChatMutation.isPending,
    isUpdatingChat: updateChatMutation.isPending,
  };
}
