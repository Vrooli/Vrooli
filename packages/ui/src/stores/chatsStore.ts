import { Chat, ChatSearchInput, ChatSearchResult, ChatSortBy, endpointsChat } from "@local/shared";
import { useContext, useEffect } from "react";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { SessionContext } from "../contexts/session.js";
import { checkIfLoggedIn } from "../utils/authentication/session.js";

interface ChatsState {
    chats: Chat[];
    isLoading: boolean;
    error: string | null;
    /** Fetch chats from the server */
    fetchChats: (signal?: AbortSignal) => Promise<Chat[]>;
    /** Set chats list */
    setChats: (chats: Chat[] | ((prev: Chat[]) => Chat[])) => void;
    /** Add a new chat to the list */
    addChat: (chat: Chat) => void;
    /** Remove a chat from the list by ID */
    removeChat: (chatId: string) => void;
    /** Update an existing chat in the list */
    updateChat: (updatedChat: Chat) => void;
    /** Clear all chats and reset state */
    clearChats: () => void;
}

export const useChatsStore = create<ChatsState>()((set, get) => ({
    chats: [],
    isLoading: false,
    error: null,
    fetchChats: async (signal) => {
        // Avoid refetching if already fetched or in progress.
        if (get().chats.length > 0 || get().isLoading) {
            return get().chats;
        }
        set({ isLoading: true, error: null });
        try {
            // TODO: Implement pagination/infinite scroll if needed later
            const response = await fetchData<ChatSearchInput, ChatSearchResult>({
                ...endpointsChat.findMany,
                inputs: { sortBy: ChatSortBy.DateUpdatedDesc }, // Use the correct enum value
                signal,
            });
            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                set({ error: "Failed to fetch chats", isLoading: false });
                throw new Error("Failed to fetch chats");
            }
            const chats = response.data?.edges.map(edge => edge.node) ?? [];

            set({ chats, isLoading: false });
            return chats;
        } catch (error) {
            if ((error as { name?: string }).name === "AbortError") return [];
            console.error("Error fetching chats:", error);
            set({ error: "Error fetching chats", isLoading: false });
            return [];
        }
    },
    setChats: (chats) => {
        set((state) => ({
            chats:
                typeof chats === "function"
                    ? chats(state.chats)
                    : chats,
        }));
    },
    addChat: (chat) => {
        // Add new chat, potentially sorting by updatedAt might be better
        set((state) => ({
            chats: [chat, ...state.chats], // Add to beginning for now
        }));
    },
    removeChat: (chatId) => {
        set((state) => ({
            chats: state.chats.filter((chat) => chat.id !== chatId),
        }));
    },
    updateChat: (updatedChat) => {
        set((state) => ({
            chats: state.chats.map((chat) =>
                chat.id === updatedChat.id ? updatedChat : chat,
            ),
        }));
    },
    clearChats: () => {
        set({ chats: [], isLoading: false, error: null });
    },
}));

/**
 * Custom hook to manage chat fetching and potentially real-time updates.
 * This hook handles:
 * 1. Fetching initial chats when the user logs in and the store is empty.
 * 2. Clearing chats from the store when the user logs out.
 *
 * Components needing the chat list should consume it directly from the store:
 * `const chats = useChatsStore(state => state.chats);`
 */
export function useChats(): void {
    const fetchChats = useChatsStore(state => state.fetchChats);
    const clearChatsStore = useChatsStore(state => state.clearChats);
    const getStoreState = useChatsStore.getState;

    const session = useContext(SessionContext);
    const isLoggedIn = checkIfLoggedIn(session);

    useEffect(function fetchExistingChatsEffect() {
        const abortController = new AbortController();

        if (!isLoggedIn) {
            if (getStoreState().chats.length > 0) {
                clearChatsStore();
            }
        } else {
            const currentState = getStoreState();
            if (currentState.chats.length === 0 && !currentState.isLoading && !currentState.error) {
                fetchChats(abortController.signal).catch(error => {
                    if ((error as { name?: string }).name !== "AbortError") {
                        console.error("Initial chat fetch failed:", error);
                    }
                });
            }
        }

        return () => {
            abortController.abort();
        };
    }, [isLoggedIn, fetchChats, clearChatsStore, session]);

    // Add WebSocket listener for real-time updates if needed later
    // useEffect(function listenForChatsEffect() { ... });
} 
