/**
 * useChats - Main orchestration hook for chat management.
 *
 * Composes focused hooks:
 * - useChatQueries: Data fetching (chat list, single chat, models)
 * - useChatMutations: CRUD mutations (create, update, delete, toggle, etc.)
 * - useChatActions: Message operations (send, regenerate, edit, branch, fork)
 * - useCompletion: AI streaming and tool calls
 * - useLabels: Label CRUD and chat-label associations
 *
 * SEAM: For testing, mock the individual hooks or the API functions.
 */
import { useState, useMemo, useRef } from "react";
import { useCompletion, type ActiveToolCall, type PendingApproval } from "./useCompletion";
import { useLabels } from "./useLabels";
import { useChatListQuery, useSelectedChatQuery, useModelsQuery } from "./useChatQueries";
import { useChatMutations } from "./useChatMutations";
import { useChatActions } from "./useChatActions";
import { CacheUpdateManager, createQueryClientAdapter } from "../lib/cache";
import { useQueryClient } from "@tanstack/react-query";

export type View = "inbox" | "starred" | "archived";

// Re-export for convenience
export type { ActiveToolCall, PendingApproval };

export interface UseChatsOptions {
  initialChatId?: string;
  onChatChange?: (chatId: string | null) => void;
}

export function useChats(options: UseChatsOptions = {}) {
  const { initialChatId, onChatChange } = options;
  const queryClient = useQueryClient();
  const [selectedChatId, setSelectedChatId] = useState<string | null>(initialChatId || null);
  const [currentView, setCurrentView] = useState<View>("inbox");

  // Cache manager for coordinating invalidations with streaming
  const cacheManagerRef = useRef<CacheUpdateManager | null>(null);
  if (!cacheManagerRef.current) {
    cacheManagerRef.current = new CacheUpdateManager(createQueryClientAdapter(queryClient));
  }
  const cacheManager = cacheManagerRef.current;

  // Delegate to focused hooks
  const completion = useCompletion();
  const labelOps = useLabels();
  const { chats, loadingChats, chatsError } = useChatListQuery(currentView, selectedChatId);
  const { chatData, loadingChat, chatError } = useSelectedChatQuery(selectedChatId);
  const { models } = useModelsQuery();

  const mutations = useChatMutations({
    selectedChatId, currentView, setSelectedChatId, onChatChange,
  });

  const actions = useChatActions({
    selectedChatId,
    setSelectedChatId,
    onChatChange,
    completion,
    cacheManager,
    chats,
    toggleReadMutate: mutations.toggleReadMutation.mutate,
    selectBranchMutate: mutations.selectBranchMutation.mutate,
    forkChatMutate: mutations.forkChatMutation.mutate,
  });

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

  return useMemo(
    () => ({
      // State
      selectedChatId, currentView, isGenerating, streamingContent,
      activeToolCalls, pendingApprovals, awaitingApprovals,
      isRegenerating: actions.isRegenerating, regeneratingContent: actions.regeneratingContent,
      generatedImages,
      editingMessage: actions.editingMessage, isEditing: actions.isEditing,

      // Data
      chats, chatData, models, labels,

      // Loading states
      loadingChats, loadingChat,

      // Errors
      chatsError, chatError,

      // Actions
      setCurrentView,
      selectChat: actions.selectChat,
      sendMessage: actions.sendMessage,
      createChatWithMessage: actions.createChatWithMessage,

      // Chat mutations
      createChat: mutations.createChatAction,
      deleteChat: mutations.deleteChatAction,
      deleteAllChats: mutations.deleteAllChatsMutation.mutateAsync,
      updateChat: mutations.updateChatAction,
      toggleRead: mutations.toggleReadAction,
      toggleArchive: mutations.toggleArchiveAction,
      toggleStar: mutations.toggleStarAction,
      autoNameChat: mutations.autoNameChatAction,

      // Branching operations
      regenerateMessage: actions.regenerateMessage,
      selectBranch: actions.selectBranch,

      // Edit operations
      setEditingMessage: actions.setEditingMessage,
      editMessageAndComplete: actions.editMessageAndComplete,
      cancelEdit: actions.cancelEdit,

      // Bulk operations
      bulkOperate: mutations.bulkOperateAction,

      // Fork conversation
      forkConversation: actions.forkConversation,

      // Tool approval actions
      approveTool: actions.approveTool,
      rejectTool: actions.rejectTool,

      // Label operations (delegated)
      createLabel: createLabelAction, deleteLabel: deleteLabelAction,
      assignLabel: assignLabelAction, removeLabel: removeLabelAction,

      // Mutation states
      isCreatingChat: mutations.isCreatingChat, isDeletingChat: mutations.isDeletingChat,
      isDeletingAllChats: mutations.isDeletingAllChats, isUpdatingChat: mutations.isUpdatingChat,
      isAutoNaming: mutations.isAutoNaming, isSelectingBranch: mutations.isSelectingBranch,
      isBulkOperating: mutations.isBulkOperating, isForking: mutations.isForking,
    }),
    [
      selectedChatId, currentView, isGenerating, streamingContent,
      activeToolCalls, pendingApprovals, awaitingApprovals,
      actions.isRegenerating, actions.regeneratingContent, generatedImages,
      actions.editingMessage, actions.isEditing,
      chats, chatData, models, labels, loadingChats, loadingChat, chatsError, chatError,
      setCurrentView, actions.selectChat, actions.sendMessage, actions.createChatWithMessage,
      mutations.createChatAction, mutations.deleteChatAction, mutations.deleteAllChatsMutation.mutateAsync,
      mutations.updateChatAction, mutations.toggleReadAction, mutations.toggleArchiveAction,
      mutations.toggleStarAction, mutations.autoNameChatAction,
      actions.regenerateMessage, actions.selectBranch,
      actions.setEditingMessage, actions.editMessageAndComplete, actions.cancelEdit,
      mutations.bulkOperateAction, actions.forkConversation,
      actions.approveTool, actions.rejectTool,
      createLabelAction, deleteLabelAction, assignLabelAction, removeLabelAction,
      mutations.isCreatingChat, mutations.isDeletingChat, mutations.isDeletingAllChats,
      mutations.isUpdatingChat, mutations.isAutoNaming, mutations.isSelectingBranch,
      mutations.isBulkOperating, mutations.isForking,
    ]
  );
}
