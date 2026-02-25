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
import { useState, useCallback, useMemo, useRef } from "react";
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
  type Chat,
  type Model,
} from "../lib/api";
import { useCompletion, type ActiveToolCall, type PendingApproval } from "./useCompletion";
import { useLabels } from "./useLabels";
import { getDefaultModel } from "../components/settings/Settings";
import type { MessagePayload } from "../components/chat/MessageInput";
import { CacheUpdateManager, createQueryClientAdapter } from "../lib/cache";

// Stable empty arrays to prevent infinite re-render loops
// CRITICAL: Using `= []` in destructuring creates a NEW array on every render,
// which changes the reference and triggers useMemo/useCallback dependencies
const EMPTY_CHATS: Chat[] = [];
const EMPTY_MODELS: Model[] = [];

export type View = "inbox" | "starred" | "archived";

// Re-export for convenience
export type { ActiveToolCall, PendingApproval };

export interface UseChatsOptions {
  /** Initial chat ID from URL - will be selected once chats are loaded */
  initialChatId?: string;
  /** Callback when selected chat changes - used for URL sync */
  onChatChange?: (chatId: string | null) => void;
  /** Callback when a template's suggested tool is used (template should be deactivated) */
  onTemplateDeactivated?: () => void;
}

export function useChats(options: UseChatsOptions = {}) {
  const { initialChatId, onChatChange, onTemplateDeactivated } = options;
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(initialChatId || null);
  const [currentView, setCurrentView] = useState<View>("inbox");

  // Cache manager for coordinating invalidations with streaming
  // Use ref to maintain stable instance across renders
  const cacheManagerRef = useRef<CacheUpdateManager | null>(null);
  if (!cacheManagerRef.current) {
    cacheManagerRef.current = new CacheUpdateManager(createQueryClientAdapter(queryClient));
  }
  const cacheManager = cacheManagerRef.current;

  // Delegate to focused hooks
  const completion = useCompletion({ onTemplateDeactivated });
  const labelOps = useLabels();

  // Fetch chats based on current view
  // NOTE: Use stable EMPTY_CHATS constant instead of `= []` to prevent
  // creating new array references on every render when data is undefined
  // CRITICAL: Use aggressive caching to prevent cascading re-renders during
  // rapid state transitions (e.g., fresh chat message send).
  const {
    data: chatsData,
    isLoading: loadingChats,
    error: chatsError,
  } = useQuery({
    queryKey: ["chats", currentView],
    queryFn: () =>
      fetchChats({
        archived: currentView === "archived",
        starred: currentView === "starred",
      }),
    staleTime: 5000, // 5 seconds - data considered fresh
    refetchInterval: selectedChatId ? 30000 : 10000,
    refetchOnMount: false, // Don't refetch when component mounts
    refetchOnWindowFocus: false, // Don't refetch on window focus
  });
  const chats = chatsData ?? EMPTY_CHATS;

  // Fetch selected chat with messages
  // CRITICAL: staleTime prevents rapid refetches during chat transitions.
  // Without this, each cache update from refetchQueries triggers re-renders
  // in all subscribers, which can cascade and cause "too many re-renders" errors.
  const {
    data: chatData,
    isLoading: loadingChat,
    error: chatError,
  } = useQuery({
    queryKey: ["chat", selectedChatId],
    queryFn: () => (selectedChatId ? fetchChat(selectedChatId) : null),
    enabled: !!selectedChatId,
    staleTime: 1000, // Data considered fresh for 1 second
    refetchOnMount: false, // Don't refetch when component mounts
    refetchOnWindowFocus: false, // Don't refetch on window focus
  });

  // Fetch available models
  // NOTE: Use stable EMPTY_MODELS constant instead of `= []`
  // Models rarely change - use aggressive caching
  const { data: modelsData } = useQuery({
    queryKey: ["models"],
    queryFn: fetchModels,
    staleTime: 300_000, // 5 minutes - models rarely change
    gcTime: 600_000, // 10 minutes - keep in cache
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
  const models = modelsData ?? EMPTY_MODELS;

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

      // Mark streaming active to defer invalidations
      cacheManager.startStreaming(chatId);

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
      } catch (error) {
        if (error instanceof DOMException && error.name === "AbortError") {
          console.log("Regeneration aborted");
        } else {
          console.error("Regeneration failed:", error);
        }
      } finally {
        setIsRegenerating(false);
        setRegeneratingContent("");
        // Single batched invalidation when streaming ends
        await cacheManager.endStreaming(chatId);
      }
    },
    [isRegenerating, completion.isGenerating, cacheManager]
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

      // Mark streaming active to defer all invalidations
      cacheManager.startStreaming(selectedChatId);

      try {
        // Create the edited message (new sibling)
        await apiEditMessage(selectedChatId, messageId, {
          content: payload.content.trim(),
          attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
          web_search: payload.webSearchEnabled ? true : undefined,
        });

        // Queue invalidation (will be batched at end of streaming)
        await cacheManager.invalidateChat(selectedChatId, ["chats"]);

        // Run AI completion (this uses useCompletion for proper streaming)
        await completion.runCompletion(selectedChatId);
      } catch (error) {
        console.error("Edit message failed:", error);
      } finally {
        setIsEditing(false);
        // Single batched invalidation when streaming ends
        await cacheManager.endStreaming(selectedChatId);
      }
    },
    // CRITICAL: Use completion.runCompletion, not the whole completion object (see sendMessageAndComplete)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, isEditing, completion.runCompletion, cacheManager]
  );

  // Cancel edit mode
  const cancelEdit = useCallback(() => {
    setEditingMessage(null);
  }, []);

  // Send message and run completion
  const sendMessageAndComplete = useCallback(
    async (chatId: string, payload: MessagePayload) => {
      // Add user message with attachments, web search, and skills settings
      const newMessage = await addMessage(chatId, {
        role: "user",
        content: payload.content.trim(),
        attachment_ids: payload.attachmentIds.length > 0 ? payload.attachmentIds : undefined,
        web_search: payload.webSearchEnabled ? true : undefined,
        skill_ids: payload.skillIds && payload.skillIds.length > 0 ? payload.skillIds : undefined,
      });

      // CRITICAL: Optimistically update cache with active_leaf tracking.
      // This avoids refetchQueries cascading re-renders while tracking the expected leaf.
      cacheManager.optimisticAddMessage(chatId, newMessage, newMessage.id);

      // CRITICAL: Defer runCompletion to the next event loop tick.
      // The optimistic update triggers synchronous re-renders in react-query subscribers.
      // Using setTimeout(0) ensures React finishes the current render batch first.
      await new Promise<void>((resolve) => {
        setTimeout(() => {
          resolve();
        }, 0);
      });

      // Mark streaming active - defers all invalidations until completion
      cacheManager.startStreaming(chatId);

      try {
        await completion.runCompletion(chatId, {
          forcedTool: payload.forcedTool,
          skills: payload.skills,
        });

      } catch (error) {
        console.error("Chat completion failed:", error);
      } finally {
        // CRITICAL: Single batched invalidation when streaming ends.
        // This consolidates all deferred invalidations into one operation,
        // preventing cascading re-renders that cause "too many re-renders" errors.
        await cacheManager.endStreaming(chatId);
      }
    },
    // CRITICAL: Depend on completion.runCompletion specifically, NOT the whole completion object.
    // While useCompletion's return is memoized, its values change when state changes (e.g., isGenerating).
    // Using the whole object would cause sendMessageAndComplete to be recreated whenever any
    // completion state changes, cascading through all callbacks that depend on it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [completion.runCompletion]
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
        // NOTE: Don't invalidate ["chats"] here - sendMessageAndComplete handles it at the end.
        // Removing this prevents redundant invalidations that cause render storms during
        // the fresh chat transition.

        await sendMessageAndComplete(chatId, payload);
      } catch (error) {
        console.error("Failed to create chat with message:", error);
      }
    },
    [completion.isGenerating, sendMessageAndComplete, onChatChange]
  );

  // Send message to existing chat
  const sendMessage = useCallback(
    async (payload: MessagePayload) => {
      const hasContent = payload.content.trim() || payload.attachmentIds.length > 0;
      if (!selectedChatId || !hasContent || completion.isGenerating) return;

      await sendMessageAndComplete(selectedChatId, payload);
    },
    [selectedChatId, completion.isGenerating, sendMessageAndComplete]
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
      // Single consolidated invalidation after approval
      // Uses cache manager to batch if multiple approvals happen rapidly
      await cacheManager.invalidateChat(selectedChatId, ["chats"]);
      // If auto-continued, backend triggered new completion - already handled by single invalidation
      if (result?.auto_continued) {
        // No need for extra invalidation - the one above is sufficient
      }
    },
    // CRITICAL: Use completion.approveTool, not the whole completion object
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, completion.approveTool, cacheManager]
  );

  const rejectTool = useCallback(
    async (toolCallId: string, reason?: string) => {
      if (!selectedChatId) return;
      await completion.rejectTool(selectedChatId, toolCallId, reason);
      // Single invalidation after rejection
      await cacheManager.invalidateChat(selectedChatId);
    },
    // CRITICAL: Use completion.rejectTool, not the whole completion object
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [selectedChatId, completion.rejectTool, cacheManager]
  );

  // CRITICAL: Memoize action functions that wrap mutation.mutate to prevent
  // creating new references on every render. These are passed down to children
  // and would cause cascading re-renders without memoization.
  const createChatAction = useCallback(
    (params?: Parameters<typeof createChat>[0]) => createChatMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps -- mutation.mutate identity changes but mutation object is stable
    [createChatMutation.mutate]
  );
  const deleteChatAction = useCallback(
    (chatId: string) => deleteChatMutation.mutate(chatId),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [deleteChatMutation.mutate]
  );
  const updateChatAction = useCallback(
    (params: Parameters<typeof updateChatMutation.mutate>[0]) => updateChatMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [updateChatMutation.mutate]
  );
  const toggleReadAction = useCallback(
    (params: Parameters<typeof toggleReadMutation.mutate>[0]) => toggleReadMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [toggleReadMutation.mutate]
  );
  const toggleArchiveAction = useCallback(
    (params: Parameters<typeof toggleArchiveMutation.mutate>[0]) => toggleArchiveMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [toggleArchiveMutation.mutate]
  );
  const toggleStarAction = useCallback(
    (params: Parameters<typeof toggleStarMutation.mutate>[0]) => toggleStarMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [toggleStarMutation.mutate]
  );
  const autoNameChatAction = useCallback(
    (chatId: string) => autoNameChatMutation.mutate(chatId),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [autoNameChatMutation.mutate]
  );
  const bulkOperateAction = useCallback(
    (params: Parameters<typeof bulkOperateMutation.mutate>[0]) => bulkOperateMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [bulkOperateMutation.mutate]
  );

  // Extract pending states outside useMemo to avoid including mutation objects in deps
  const isCreatingChat = createChatMutation.isPending;
  const isDeletingChat = deleteChatMutation.isPending;
  const isDeletingAllChats = deleteAllChatsMutation.isPending;
  const isUpdatingChat = updateChatMutation.isPending;
  const isAutoNaming = autoNameChatMutation.isPending;
  const isSelectingBranch = selectBranchMutation.isPending;
  const isBulkOperating = bulkOperateMutation.isPending;
  const isForking = forkChatMutation.isPending;

  // Extract completion state values for stable dependencies
  const isGenerating = completion.isGenerating;
  const streamingContent = completion.streamingContent;
  const activeToolCalls = completion.activeToolCalls;
  const pendingApprovals = completion.pendingApprovals;
  const awaitingApprovals = completion.awaitingApprovals;
  const generatedImages = completion.generatedImages;

  // Extract label ops for stable dependencies
  const labels = labelOps.labels;
  const createLabelAction = labelOps.createLabel;
  const deleteLabelAction = labelOps.deleteLabel;
  const assignLabelAction = labelOps.assignLabel;
  const removeLabelAction = labelOps.removeLabel;

  // CRITICAL: Memoize the return object to prevent creating new object references
  // on every render. Without this, every render creates a new object that triggers
  // re-renders in AppContent and all children, potentially causing "too many re-renders"
  // errors during rapid state transitions.
  return useMemo(
    () => ({
      // State
      selectedChatId,
      currentView,
      isGenerating,
      streamingContent,
      activeToolCalls,
      pendingApprovals,
      awaitingApprovals,
      isRegenerating,
      regeneratingContent,
      generatedImages,
      // Edit state
      editingMessage,
      isEditing,

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
      createChatWithMessage,

      // Chat mutations - use memoized callbacks
      createChat: createChatAction,
      deleteChat: deleteChatAction,
      deleteAllChats: deleteAllChatsMutation.mutateAsync,
      updateChat: updateChatAction,
      toggleRead: toggleReadAction,
      toggleArchive: toggleArchiveAction,
      toggleStar: toggleStarAction,
      autoNameChat: autoNameChatAction,

      // Branching operations (ChatGPT-style regeneration)
      regenerateMessage,
      selectBranch,

      // Edit operations
      setEditingMessage,
      editMessageAndComplete,
      cancelEdit,

      // Bulk operations
      bulkOperate: bulkOperateAction,

      // Fork conversation
      forkConversation,

      // Tool approval actions
      approveTool,
      rejectTool,

      // Label operations (delegated)
      createLabel: createLabelAction,
      deleteLabel: deleteLabelAction,
      assignLabel: assignLabelAction,
      removeLabel: removeLabelAction,

      // Mutation states
      isCreatingChat,
      isDeletingChat,
      isDeletingAllChats,
      isUpdatingChat,
      isAutoNaming,
      isSelectingBranch,
      isBulkOperating,
      isForking,
    }),
    [
      selectedChatId,
      currentView,
      isGenerating,
      streamingContent,
      activeToolCalls,
      pendingApprovals,
      awaitingApprovals,
      isRegenerating,
      regeneratingContent,
      generatedImages,
      editingMessage,
      isEditing,
      chats,
      chatData,
      models,
      labels,
      loadingChats,
      loadingChat,
      chatsError,
      chatError,
      setCurrentView,
      selectChat,
      sendMessage,
      createChatWithMessage,
      createChatAction,
      deleteChatAction,
      deleteAllChatsMutation.mutateAsync,
      updateChatAction,
      toggleReadAction,
      toggleArchiveAction,
      toggleStarAction,
      autoNameChatAction,
      regenerateMessage,
      selectBranch,
      setEditingMessage,
      editMessageAndComplete,
      cancelEdit,
      bulkOperateAction,
      forkConversation,
      approveTool,
      rejectTool,
      createLabelAction,
      deleteLabelAction,
      assignLabelAction,
      removeLabelAction,
      isCreatingChat,
      isDeletingChat,
      isDeletingAllChats,
      isUpdatingChat,
      isAutoNaming,
      isSelectingBranch,
      isBulkOperating,
      isForking,
    ]
  );
}
