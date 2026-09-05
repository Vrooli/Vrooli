/**
 * useChatMutations - React Query mutations for chat CRUD operations.
 *
 * Extracted from useChats.ts for modularity.
 */
import { useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  createChat,
  deleteChat,
  deleteAllChats,
  updateChat,
  toggleRead,
  toggleArchive,
  toggleStar,
  autoNameChat,
  selectBranch as apiSelectBranch,
  bulkOperateChats as apiBulkOperateChats,
  forkChat as apiForkChat,
  type BulkOperation,
} from "../lib/api";
import { getDefaultModel } from "../components/settings/Settings";

export interface UseChatMutationsOptions {
  selectedChatId: string | null;
  currentView: string;
  setSelectedChatId: (id: string | null) => void;
  onChatChange?: (chatId: string | null) => void;
}

export function useChatMutations({
  selectedChatId,
  currentView,
  setSelectedChatId,
  onChatChange,
}: UseChatMutationsOptions) {
  const queryClient = useQueryClient();

  const createChatMutation = useMutation({
    mutationFn: (params: Parameters<typeof createChat>[0] = {}) => {
      if (!params.model) {
        params.model = getDefaultModel();
      }
      return createChat(params);
    },
    onSuccess: (newChat) => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(newChat.id);
      onChatChange?.(newChat.id);
    },
  });

  const deleteChatMutation = useMutation({
    mutationFn: deleteChat,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(null);
      onChatChange?.(null);
    },
  });

  const deleteAllChatsMutation = useMutation({
    mutationFn: deleteAllChats,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      void queryClient.invalidateQueries({ queryKey: ["chat"] });
      setSelectedChatId(null);
      onChatChange?.(null);
    },
  });

  const updateChatMutation = useMutation({
    mutationFn: ({ chatId, data }: { chatId: string; data: { name?: string; model?: string } }) =>
      updateChat(chatId, data),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const toggleReadMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleRead(chatId, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      void queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
    },
  });

  const toggleArchiveMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleArchive(chatId, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      if (currentView === "inbox" || currentView === "archived") {
        setSelectedChatId(null);
        onChatChange?.(null);
      }
    },
  });

  const toggleStarMutation = useMutation({
    mutationFn: ({ chatId, value }: { chatId: string; value?: boolean }) => toggleStar(chatId, value),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const autoNameChatMutation = useMutation({
    mutationFn: (chatId: string) => autoNameChat(chatId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["chat", selectedChatId] });
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
    },
  });

  const selectBranchMutation = useMutation({
    mutationFn: ({ chatId, messageId }: { chatId: string; messageId: string }) =>
      apiSelectBranch(chatId, messageId),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: ["chat", variables.chatId] });
    },
  });

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
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      void queryClient.invalidateQueries({ queryKey: ["chat"] });
    },
  });

  const forkChatMutation = useMutation({
    mutationFn: ({ chatId, messageId }: { chatId: string; messageId: string }) =>
      apiForkChat(chatId, messageId),
    onSuccess: (newChat) => {
      void queryClient.invalidateQueries({ queryKey: ["chats"] });
      setSelectedChatId(newChat.id);
      onChatChange?.(newChat.id);
    },
  });

  // Memoized action wrappers
  const createChatAction = useCallback(
    (params?: Parameters<typeof createChat>[0]) => createChatMutation.mutate(params),
    // eslint-disable-next-line react-hooks/exhaustive-deps
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

  return {
    // Raw mutations (for direct access)
    createChatMutation,
    deleteChatMutation,
    deleteAllChatsMutation,
    updateChatMutation,
    toggleReadMutation,
    toggleArchiveMutation,
    toggleStarMutation,
    autoNameChatMutation,
    selectBranchMutation,
    bulkOperateMutation,
    forkChatMutation,

    // Memoized action wrappers
    createChatAction,
    deleteChatAction,
    updateChatAction,
    toggleReadAction,
    toggleArchiveAction,
    toggleStarAction,
    autoNameChatAction,
    bulkOperateAction,

    // Pending states
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
