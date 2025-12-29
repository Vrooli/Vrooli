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
  regenerateMessage as apiRegenerateMessage,
  editMessage as apiEditMessage,
  selectBranch as apiSelectBranch,
  bulkOperateChats as apiBulkOperateChats,
  forkChat as apiForkChat,
  type StreamingEvent,
  type BulkOperation,
  type Message,
} from "../lib/api";
import { useCompletion, type ActiveToolCall, type PendingApproval } from "./useCompletion";
import { useLabels } from "./useLabels";
import { getDefaultModel } from "../components/settings/Settings";
import type { MessagePayload } from "../components/chat/MessageInput";

export type View = "inbox" | "starred" | "archived";

// Re-export for convenience
export type { ActiveToolCall, PendingApproval };

export interface UseChatsOptions {
  /** Initial chat ID from URL - will be selected once chats are loaded */
  initialChatId?: string;
  /** Callback when selected chat changes - used for URL sync */
  onChatChange?: (chatId: string | null) => void;
}

export function useChats(options: UseChatsOptions = {}) {
  const { initialChatId, onChatChange } = options;
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(initialChatId || null);
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
      onChatChange?.(newChat.id);
    },
  });

  const deleteChatMutation = useMutation({
    mutationFn: deleteChat,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(null);
      onChatChange?.(null);
    },
  });

  const deleteAllChatsMutation = useMutation({
    mutationFn: deleteAllChats,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      queryClient.invalidateQueries({ queryKey: ["chat"] });
      setSelectedChatId(null);
      onChatChange?.(null);
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
        onChatChange?.(null);
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

  // Branch selection mutation
  const selectBranchMutation = useMutation({
    mutationFn: ({ chatId, messageId }: { chatId: string; messageId: string }) =>
      apiSelectBranch(chatId, messageId),
    onSuccess: (_data, variables) => {
      // Use chatId from variables, not from closure, to avoid stale data
      queryClient.invalidateQueries({ queryKey: ["chat", variables.chatId] });
    },
  });

  // State for regeneration streaming
  const [isRegenerating, setIsRegenerating] = useState(false);
  const [regeneratingContent, setRegeneratingContent] = useState("");

  // State for message editing
  const [editingMessage, setEditingMessage] = useState<Message | null>(null);
  const [isEditing, setIsEditing] = useState(false);

  // Regenerate a message (ChatGPT-style branching)
  const regenerateMessage = useCallback(
    async (chatId: string, messageId: string) => {
      if (isRegenerating || completion.isGenerating) return;

      setIsRegenerating(true);
      setRegeneratingContent("");

      const abortController = new AbortController();

      try {
        await apiRegenerateMessage(chatId, messageId, {
          stream: true,
          signal: abortController.signal,
          onChunk: (content: string) => {
            setRegeneratingContent((prev) => prev + content);
          },
          onEvent: (event: StreamingEvent) => {
            // Handle tool calls and other events if needed
            if (event.type === "error") {
              console.error("Regeneration error:", event.error);
            }
          },
        });

        // Refresh chat data after regeneration completes
        queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
        queryClient.invalidateQueries({ queryKey: ["chats"] });
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          console.log("Regeneration aborted");
        } else {
          console.error("Regeneration failed:", error);
        }
      } finally {
        setIsRegenerating(false);
        setRegeneratingContent("");
      }
    },
    [isRegenerating, completion.isGenerating, queryClient]
  );

  // Select a different message branch
  const selectBranch = useCallback(
    (messageId: string) => {
      console.log("[useChats] selectBranch called", { selectedChatId, messageId });
      if (!selectedChatId) {
        console.log("[useChats] No selectedChatId, returning");
        return;
      }
      console.log("[useChats] Calling selectBranchMutation.mutate");
      selectBranchMutation.mutate({ chatId: selectedChatId, messageId });
    },
    [selectedChatId, selectBranchMutation]
  );

  // Edit a message (creates sibling and triggers new AI response)
  const editMessageAndComplete = useCallback(
    async (messageId: string, payload: MessagePayload) => {
      if (!selectedChatId || isEditing || completion.isGenerating) return;

      setIsEditing(true);
      // Clear editing state immediately so the input clears
      setEditingMessage(null);

      try {
        // Create the edited message (new sibling)
        await apiEditMessage(selectedChatId, messageId, {
          content: payload.content.trim(),
          attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
          web_search: payload.webSearchEnabled ? true : undefined,
        });

        // Refresh chat data immediately to show the new message
        queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
        queryClient.invalidateQueries({ queryKey: ["chats"] });

        // Run AI completion (this uses useCompletion for proper streaming)
        await completion.runCompletion(selectedChatId);

        // Refresh again after completion
        queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
        queryClient.invalidateQueries({ queryKey: ["chats"] });
      } catch (error) {
        console.error("Edit message failed:", error);
      } finally {
        setIsEditing(false);
      }
    },
    [selectedChatId, isEditing, completion, queryClient]
  );

  // Cancel edit mode
  const cancelEdit = useCallback(() => {
    setEditingMessage(null);
  }, []);

  // Send message and run completion
  const sendMessageAndComplete = useCallback(
    async (chatId: string, payload: MessagePayload, needsAutoName: boolean) => {
      // Add user message with attachments and web search settings
      await addMessage(chatId, {
        role: "user",
        content: payload.content.trim(),
        attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
        web_search: payload.webSearchEnabled ? true : undefined,
      });
      queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });

      // Run AI completion with optional forced tool
      try {
        await completion.runCompletion(chatId, { forcedTool: payload.forcedTool });
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
    async (payload: MessagePayload) => {
      const hasContent = payload.content.trim() || payload.attachmentIds.length > 0;
      if (!hasContent || completion.isGenerating) return;

      try {
        const defaultModel = getDefaultModel();
        const newChat = await createChat({ model: defaultModel });
        const chatId = newChat.id;

        setSelectedChatId(chatId);
        onChatChange?.(chatId);
        queryClient.invalidateQueries({ queryKey: ["chats"] });

        await sendMessageAndComplete(chatId, payload, true);
      } catch (error) {
        console.error("Failed to create chat with message:", error);
      }
    },
    [completion.isGenerating, queryClient, sendMessageAndComplete, onChatChange]
  );

  // Send message to existing chat
  const sendMessage = useCallback(
    async (payload: MessagePayload) => {
      const hasContent = payload.content.trim() || payload.attachmentIds.length > 0;
      if (!selectedChatId || !hasContent || completion.isGenerating) return;

      const currentChat = chats.find((c) => c.id === selectedChatId);
      const needsAutoName = currentChat?.name === "New Chat";

      await sendMessageAndComplete(selectedChatId, payload, needsAutoName);
    },
    [selectedChatId, completion.isGenerating, chats, sendMessageAndComplete]
  );

  // Select chat and mark as read
  const selectChat = useCallback(
    (chatId: string) => {
      const newId = chatId || null;
      setSelectedChatId(newId);
      onChatChange?.(newId);
      const chat = chats.find((c) => c.id === chatId);
      if (chat && !chat.is_read) {
        toggleReadMutation.mutate({ chatId, value: true });
      }
    },
    [chats, toggleReadMutation, onChatChange]
  );

  // Bulk operations mutation
  const bulkOperateMutation = useMutation({
    mutationFn: ({
      chatIds,
      operation,
      labelId,
    }: {
      chatIds: string[];
      operation: BulkOperation;
      labelId?: string;
    }) => apiBulkOperateChats(chatIds, operation, labelId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      queryClient.invalidateQueries({ queryKey: ["chat"] });
    },
  });

  // Fork chat mutation
  const forkChatMutation = useMutation({
    mutationFn: ({ chatId, messageId }: { chatId: string; messageId: string }) =>
      apiForkChat(chatId, messageId),
    onSuccess: (newChat) => {
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(newChat.id);
      onChatChange?.(newChat.id);
    },
  });

  // Fork conversation from a specific message
  const forkConversation = useCallback(
    (messageId: string) => {
      if (!selectedChatId) return;
      forkChatMutation.mutate({ chatId: selectedChatId, messageId });
    },
    [selectedChatId, forkChatMutation]
  );

  // Tool approval actions that wrap completion methods with the current chatId
  const approveTool = useCallback(
    async (toolCallId: string) => {
      if (!selectedChatId) return;
      const result = await completion.approveTool(selectedChatId, toolCallId);
      // Refetch chat after approval to get updated messages
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      queryClient.invalidateQueries({ queryKey: ["chats"] });
      // If auto-continued, the backend triggered a new completion
      // which will add new messages, so we refetch
      if (result?.auto_continued) {
        queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      }
    },
    [selectedChatId, completion, queryClient]
  );

  const rejectTool = useCallback(
    async (toolCallId: string, reason?: string) => {
      if (!selectedChatId) return;
      await completion.rejectTool(selectedChatId, toolCallId, reason);
      // Refetch chat after rejection
      queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    },
    [selectedChatId, completion, queryClient]
  );

  return {
    // State
    selectedChatId,
    currentView,
    isGenerating: completion.isGenerating,
    streamingContent: completion.streamingContent,
    activeToolCalls: completion.activeToolCalls,
    pendingApprovals: completion.pendingApprovals,
    awaitingApprovals: completion.awaitingApprovals,
    isRegenerating,
    regeneratingContent,
    generatedImages: completion.generatedImages,
    // Edit state
    editingMessage,
    isEditing,

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

    // Branching operations (ChatGPT-style regeneration)
    regenerateMessage,
    selectBranch,

    // Edit operations
    setEditingMessage,
    editMessageAndComplete,
    cancelEdit,

    // Bulk operations
    bulkOperate: bulkOperateMutation.mutate,

    // Fork conversation
    forkConversation,

    // Tool approval actions
    approveTool,
    rejectTool,

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
    isSelectingBranch: selectBranchMutation.isPending,
    isBulkOperating: bulkOperateMutation.isPending,
    isForking: forkChatMutation.isPending,
  };
}
