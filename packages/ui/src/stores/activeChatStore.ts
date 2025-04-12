import { Chat, ChatCreateInput, ChatParticipantShape, ChatShape, DUMMY_ID, FindByIdInput, SEEDED_IDS, Session, endpointsChat, uuidValidate } from "@local/shared";
import { TFunction } from "i18next";
import { useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";
import { SessionContext } from "../contexts/session.js";
import { useMessageActions, useMessageTree } from "../hooks/messages.js";
import { useChatTasks } from "../hooks/tasks.js";
import { useSocketChat } from "../hooks/useSocketChat.js";
import { getCurrentUser } from "../utils/authentication/session.js";
import { getUserLanguages } from "../utils/display/translationTools.js";
import { getCookieMatchingChat, setCookieMatchingChat } from "../utils/localStorage.js";
import { CHAT_DEFAULTS, chatInitialValues, transformChatValues, withModifiableMessages, withYourMessages } from "../views/objects/chat/ChatCrud.js";

interface ActiveChatState {
    /** 
     * The active chat, or null if there isn't one yet.
     */
    chat: ChatShape | null;
    /**
     * Indicates that the chat is a bot-only chat. 
     * 
     * Lets us know if editing a message should create a new version
     */
    isBotOnlyChat: boolean;
    /**
     * True if we're loading a chat or creating a new chat
     */
    isLoading: boolean;
    /**
     * The last message in the current message tree for the active chat, 
     * or null if there is no active chat or there are no messages in the chat.
     */
    latestMessageId: string | null;
    /**
     * The chat's participants
     */
    participants: Omit<ChatParticipantShape, "chat">[];
    /**
     * Users/bots currently typing in the chat
     */
    usersTyping: Omit<ChatParticipantShape, "chat">[];
    /**
     * Resets the active chat by creating a new empty chat
     * 
     * @param session The user session
     * @param translator Translation function
     * @param language The language to use
     */
    resetActiveChat: (session: Session, translator: TFunction, language?: string) => Promise<void>;
    /**
     * Sets the active chat
     */
    setActiveChat: (chat: ChatShape | null, session: Session) => void;
    /**
     * Updates the latest message ID
     */
    setLatestMessageId: (id: string) => void;
    /**
     * Updates the chat participants
     */
    setParticipants: (participants: Omit<ChatParticipantShape, "chat">[]) => void;
    /**
     * Updates the users/bots currently typing in the chat
     */
    setUsersTyping: (usersTyping: Omit<ChatParticipantShape, "chat">[]) => void;
    /**
     * Gets chat info from cache or creates a new chat
     */
    initializeChat: (session: Session, translator: TFunction) => Promise<void>;
    /**
     * Check if a chat is a bot-only chat (has only bots and the current user)
     */
    calculateIsBotOnlyChat: (chat: ChatShape | null, session: Session) => boolean;
}

export const useActiveChatStore = create<ActiveChatState>()((set, get) => {
    // Loading state tracking
    const isFetchingChat = { current: false };
    const hasTriedCreatingNewChat = { current: false };

    return {
        chat: null,
        isBotOnlyChat: false,
        isLoading: false,
        latestMessageId: null,
        participants: [],
        usersTyping: [],

        resetActiveChat: async (session, translator, language = "en") => {
            if (!session) return;

            const { languages: userLanguages } = getCurrentUser(session);

            const chatToUse = chatInitialValues(
                session,
                translator,
                language || (userLanguages ? userLanguages[0] : "en"),
                CHAT_DEFAULTS,
            );

            try {
                const inputs = transformChatValues(
                    withModifiableMessages(chatToUse, session),
                    withYourMessages(chatToUse, session),
                    true,
                );

                const response = await fetchData<ChatCreateInput, Chat>({
                    ...endpointsChat.createOne,
                    inputs: inputs as ChatCreateInput, // Type assertion to fix type error
                });

                if (response.errors) {
                    ServerResponseParser.displayErrors(response.errors);
                    throw new Error("Failed to create new chat");
                }

                const data = response.data;
                if (data) {
                    set({
                        chat: { ...data, messages: [] },
                        participants: data.participants || [],
                        usersTyping: [],
                    });

                    const userId = getCurrentUser(session).id;
                    if (userId) {
                        setCookieMatchingChat(data.id, [userId, SEEDED_IDS.User.Valyxa]);
                    }
                }
            } catch (error) {
                console.error("Error creating chat:", error);
            }
        },

        setActiveChat: (chat, session) => {
            if (chat && chat.participants) {
                set({
                    chat,
                    participants: chat.participants,
                });
            } else {
                set({
                    chat: null,
                    participants: [],
                });
            }

            // Update the cache
            if (!chat || !session) return;

            const userId = getCurrentUser(session).id;
            const participantIds = chat.participants?.map(p => p.user?.id) ?? [];

            if (userId && participantIds.length > 1 && participantIds.includes(userId)) {
                // Must have more than yourself
                setCookieMatchingChat(chat.id, participantIds);
            }
        },

        setLatestMessageId: (id) => {
            set({ latestMessageId: id });
        },

        setParticipants: (participants) => {
            set({ participants });
        },

        setUsersTyping: (usersTyping) => {
            set({ usersTyping });
        },

        initializeChat: async (session, translator) => {
            if (!session) return;

            const userId = getCurrentUser(session).id;
            if (!userId || isFetchingChat.current) return;

            // Check local storage for an existing chat
            const existingChatId = getCookieMatchingChat([userId, SEEDED_IDS.User.Valyxa]);

            // If stored chat is valid, fetch it
            if (existingChatId && existingChatId !== DUMMY_ID && uuidValidate(existingChatId)) {
                isFetchingChat.current = true;
                set({ isLoading: true });

                try {
                    const response = await fetchData<FindByIdInput, Chat>({
                        ...endpointsChat.findOne,
                        inputs: { id: existingChatId },
                    });

                    if (response.errors) {
                        // If we get an error indicating that the stored chat doesn't exist, create a new chat
                        if (ServerResponseParser.hasErrorCode(response, "NotFound") && !hasTriedCreatingNewChat.current) {
                            hasTriedCreatingNewChat.current = true;
                            await get().resetActiveChat(session, translator);
                        } else {
                            ServerResponseParser.displayErrors(response.errors);
                        }
                    } else if (response.data) {
                        set({
                            chat: { ...response.data, messages: [] },
                            participants: response.data.participants || [],
                            isLoading: false,
                        });
                    } else {
                        // No data and no errors, consider this a failed fetch
                        set({ isLoading: false });
                    }
                } catch (error) {
                    console.error("Error fetching chat:", error);
                    set({ isLoading: false });
                } finally {
                    isFetchingChat.current = false;
                    set({ isLoading: false });
                }
            }
            // Otherwise, create a new chat
            else if (!hasTriedCreatingNewChat.current) {
                hasTriedCreatingNewChat.current = true;
                await get().resetActiveChat(session, translator);
            }
        },

        calculateIsBotOnlyChat: (chat, session) => {
            if (!chat || !session) return false;

            return chat.participants?.every(
                p => p.user?.isBot || p.user?.id === getCurrentUser(session).id,
            ) ?? false;
        },
    };
});

/**
 * Custom hook for using the active chat store with session context and socket integration.
 * This hook combines the Zustand store with the useSocketChat hook for socket event handling.
 */
export function useActiveChat({
    setMessage,
}: {
    setMessage?: (message: string) => unknown;
}) {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const languages = useMemo(() => getUserLanguages(session), [session]);

    const store = useActiveChatStore();
    const messageTree = useMessageTree(store.chat?.id);
    const taskInfo = useChatTasks({ chatId: store.chat?.id });
    const isBotOnlyChat = session !== null && session !== undefined && store.calculateIsBotOnlyChat(store.chat, session);

    // Add a ref to track if initialization has been attempted
    const hasInitialized = useRef(false);

    // Listen for socket events and update the store
    const { messageStream } = useSocketChat({
        addMessages: messageTree.addMessages,
        chat: store.chat,
        editMessage: messageTree.editMessage,
        participants: store.participants,
        removeMessages: messageTree.removeMessages,
        setParticipants: store.setParticipants,
        setUsersTyping: store.setUsersTyping,
        usersTyping: store.usersTyping,
    });
    const updateMessage = useCallback((updatedMessage: string) => {
        setMessage?.(updatedMessage);
    }, [setMessage]);
    // Set up message actions (handles complex logic like updating the message tree)
    const messageActions = useMessageActions({
        activeTask: taskInfo.activeTask,
        addMessages: messageTree.addMessages,
        chat: store.chat,
        contexts: taskInfo.contexts[taskInfo.activeTask.taskId] || [],
        editMessage: messageTree.editMessage,
        isBotOnlyChat,
        language: languages[0],
        setMessage: updateMessage,
    });

    // Wrapped functions that include session
    const resetActiveChat = useCallback(() => {
        if (session) {
            return store.resetActiveChat(session, t);
        }
        return Promise.resolve();
    }, [session, store, t]);

    const setActiveChat = useCallback((newChat: ChatShape | null) => {
        if (session) {
            store.setActiveChat(newChat, session);
        }
    }, [session, store]);

    // Initialize chat when hook is mounted
    useEffect(() => {
        if (session && !hasInitialized.current) {
            hasInitialized.current = true;
            store.initializeChat(session, t);
        }
    }, [session, t]);

    return {
        chat: store.chat,
        isBotOnlyChat,
        isLoading: store.isLoading,
        latestMessageId: store.latestMessageId,
        participants: store.participants,
        usersTyping: store.usersTyping,
        resetActiveChat,
        setActiveChat,
        messageActions,
        messageTree,
        messageStream,
        taskInfo,
    };
}
