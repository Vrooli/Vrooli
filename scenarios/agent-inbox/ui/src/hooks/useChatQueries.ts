/**
 * useChatQueries - React Query hooks for fetching chat data.
 *
 * Extracted from useChats.ts for modularity.
 */
import { useQuery } from "@tanstack/react-query";
import {
  fetchChats,
  fetchChat,
  fetchModels,
  type Chat,
  type Model,
} from "../lib/api";
import type { View } from "./useChats";

// Stable empty arrays to prevent infinite re-render loops
const EMPTY_CHATS: Chat[] = [];
const EMPTY_MODELS: Model[] = [];

export function useChatListQuery(currentView: View, selectedChatId: string | null) {
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
    staleTime: 5000,
    refetchInterval: selectedChatId ? 30000 : 10000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
  const chats = chatsData ?? EMPTY_CHATS;

  return { chats, loadingChats, chatsError };
}

export function useSelectedChatQuery(selectedChatId: string | null) {
  const {
    data: chatData,
    isLoading: loadingChat,
    error: chatError,
  } = useQuery({
    queryKey: ["chat", selectedChatId],
    queryFn: () => (selectedChatId ? fetchChat(selectedChatId) : null),
    enabled: !!selectedChatId,
    staleTime: 1000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });

  return { chatData, loadingChat, chatError };
}

export function useModelsQuery() {
  const { data: modelsData } = useQuery({
    queryKey: ["models"],
    queryFn: fetchModels,
    staleTime: 300_000,
    gcTime: 600_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
  });
  const models = modelsData ?? EMPTY_MODELS;

  return { models };
}
