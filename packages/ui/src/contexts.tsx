import { Chat, ChatCreateInput, ChatParticipantShape, ChatShape, DUMMY_ID, FindByIdInput, Session, VALYXA_ID, endpointGetChat, endpointPostChat, noop, uuidValidate } from "@local/shared";
import { fetchLazyWrapper, hasErrorCode } from "api";
import { useLazyFetch } from "hooks/useLazyFetch";
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieMatchingChat, setCookieMatchingChat } from "utils/localStorage";
import { CHAT_DEFAULTS, chatInitialValues, transformChatValues, withModifiableMessages, withYourMessages } from "views/objects/chat";

export const SessionContext = createContext<Session | undefined>(undefined);

type ZIndexContextType = {
    getZIndex: () => number;
    releaseZIndex: () => unknown;
};

export const DEFAULT_Z_INDEX = 1000;
const Z_INDEX_INCREMENT = 5;

export const ZIndexContext = createContext<ZIndexContextType | undefined>(undefined);

export function ZIndexProvider({ children }) {
    const stack = useRef<number[]>([]);

    function getZIndex() {
        const newZIndex = (stack.current.length > 0 ? stack.current[stack.current.length - 1] : DEFAULT_Z_INDEX) + Z_INDEX_INCREMENT;
        stack.current.push(newZIndex);
        return newZIndex;
    }

    function releaseZIndex() {
        stack.current.pop();
    }

    const value = useMemo(() => ({ getZIndex, releaseZIndex }), []);

    return (
        <ZIndexContext.Provider value={value}>
            {children}
        </ZIndexContext.Provider>
    );
}

export type ActiveChatContext = {
    /** 
     * The active chat, or null if there isn't one yet.
     */
    chat: ChatShape | null
    /**
     * Indicates that the chat is a bot-only chat. 
     * Should always be true for the active chat
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
     * Creates a new active chat.
     */
    resetChat: () => unknown;
    /**
     * Updates the latest message ID.
     * @param latestMessageId The new latest message ID
     */
    setLatestMessageId: (latestMessageId: string) => unknown;
    /**
     * Update the chat participants
     */
    setParticipants: (participants: Omit<ChatParticipantShape, "chat">[]) => unknown;
    /**
     * Updates the users currently typing in the chat
     */
    setUsersTyping: (usersTyping: Omit<ChatParticipantShape, "chat">[]) => unknown;
    /**
     * Users currently typing in the chat
     */
    usersTyping: Omit<ChatParticipantShape, "chat">[];
}

export const ActiveChatContext = createContext<ActiveChatContext>({
    chat: null,
    isBotOnlyChat: true,
    isLoading: false,
    latestMessageId: null,
    participants: [],
    resetChat: noop,
    setLatestMessageId: noop,
    setParticipants: noop,
    setUsersTyping: noop,
    usersTyping: [],
});

export function ActiveChatProvider({ children }) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { id: userId, languages: userLanguages } = getCurrentUser(session);

    const [getChat, { data: loadedChat, loading: isChatLoading }] = useLazyFetch<FindByIdInput, Chat>(endpointGetChat);
    const [createChat, { loading: isCreatingChat }] = useLazyFetch<ChatCreateInput, Chat>(endpointPostChat);
    const isLoading = useMemo(function isLoadingMemo() {
        return isChatLoading || isCreatingChat;
    }, [isChatLoading, isCreatingChat]);

    const [chat, setChat] = useState<ChatShape | null>(null);
    const [latestMessageId, setLatestMessageId] = useState<string | null>(null);

    const [participants, setParticipants] = useState<Omit<ChatParticipantShape, "chat">[]>([]);
    const [usersTyping, setUsersTyping] = useState<Omit<ChatParticipantShape, "chat">[]>([]);

    useEffect(function setParticipantsEffect() {
        if (loadedChat?.participants) {
            setParticipants(loadedChat.participants);
        }
    }, [loadedChat?.participants]);

    const isBotOnlyChat = useMemo(function isBotOnlyChatMemo() {
        return chat?.participants?.every(p => p.user?.isBot || p.user?.id === getCurrentUser(session).id) ?? false;
    }, [chat, session]);

    const hasTriedCreatingNewChat = useRef(false);
    const isFetchingChat = useRef(false);

    const resetChat = useCallback(function resetChatCallback() {
        const chatToUse = chatInitialValues(session, t, userLanguages ? userLanguages[0] : "en", CHAT_DEFAULTS);
        fetchLazyWrapper<ChatCreateInput, Chat>({
            fetch: createChat,
            inputs: transformChatValues(withModifiableMessages(chatToUse, session), withYourMessages(chatToUse, session), true),
            onSuccess: (data) => {
                setChat({ ...data, messages: [] });
                setUsersTyping([]);
                const userId = getCurrentUser(session).id;
                if (userId) setCookieMatchingChat(data.id, [userId, VALYXA_ID]);
            },
        });
    }, [createChat, userLanguages, session, t]);

    useEffect(() => {
        if (!userId || isFetchingChat.current) return;

        // Check local storage for an existing chat
        const existingChatId = getCookieMatchingChat([userId, VALYXA_ID]);

        async function initializeChat() {
            if (!userId) return;

            // If stored chat is valid, fetch it
            if (existingChatId && existingChatId !== DUMMY_ID && uuidValidate(existingChatId)) {
                isFetchingChat.current = true;
                fetchLazyWrapper<FindByIdInput, Chat>({
                    fetch: getChat,
                    inputs: { id: existingChatId },
                    onSuccess: (data) => {
                        setChat({ ...data, messages: [] });
                    },
                    onError: (response) => {
                        // If we get an error indicating that the stored chat doesn't exist, create a new chat
                        if (hasErrorCode(response, "NotFound") && !hasTriedCreatingNewChat.current) {
                            hasTriedCreatingNewChat.current = true;
                            resetChat();
                        }
                    },
                    onCompleted: () => {
                        isFetchingChat.current = false;
                    },
                });
            }
            // Otherwise, create a new chat
            else if (!hasTriedCreatingNewChat.current) {
                hasTriedCreatingNewChat.current = true;
                resetChat();
            }
        }

        initializeChat();
    }, [getChat, resetChat, userId]);

    const value = useMemo<ActiveChatContext>(() => ({
        chat,
        isBotOnlyChat,
        isLoading,
        latestMessageId,
        participants,
        resetChat,
        setLatestMessageId,
        setParticipants,
        setUsersTyping,
        usersTyping,
    }), [chat, isBotOnlyChat, isLoading, latestMessageId, participants, resetChat, usersTyping]);

    return (
        <ActiveChatContext.Provider value={value}>
            {children}
        </ActiveChatContext.Provider>
    );
}
