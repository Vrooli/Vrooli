import { ChatMessageShape, ChatParticipantShape, ChatShape, ChatSocketEventPayloads, DUMMY_ID, JOIN_CHAT_ROOM_ERRORS, Session } from "@local/shared";
import { SocketService } from "api/socket";
import { SessionContext } from "contexts";
import { useCallback, useContext, useEffect, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { removeCookieMatchingChat, removeCookiesWithChatId, setCookieMatchingChat, updateCookiePartialTaskForChat, upsertCookieTaskForChat } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { useThrottle } from "./useThrottle";

type ParticipantWithoutChat = Omit<ChatParticipantShape, "chat">;

type UseSocketChatProps = {
    addMessages: (newMessages: ChatMessageShape[]) => unknown;
    chat?: ChatShape | null;
    editMessage: (editedMessage: (Partial<ChatMessageShape> & { id: string })) => unknown;
    participants: ParticipantWithoutChat[];
    removeMessages: (deletedIds: string[]) => unknown;
    setParticipants: (participants: ParticipantWithoutChat[]) => unknown;
    setUsersTyping: (updatedParticipants: ParticipantWithoutChat[]) => unknown;
    usersTyping: ParticipantWithoutChat[];
}

export function processMessages(
    { added, updated, removed }: ChatSocketEventPayloads["messages"],
    addMessages: UseSocketChatProps["addMessages"],
    editMessage: UseSocketChatProps["editMessage"],
    removeMessages: UseSocketChatProps["removeMessages"],
) {
    if (Array.isArray(added) && added.length > 0) {
        addMessages(added.map(m => ({ ...m, status: "sent" })));
    }
    if (Array.isArray(updated) && updated.length > 0) {
        updated.forEach(message => {
            editMessage({ ...message, status: "sent" });
        });
    }
    if (Array.isArray(removed) && removed.length > 0) {
        removeMessages(removed);
    }
}

export function processTypingUpdates(
    { starting, stopping }: ChatSocketEventPayloads["typing"],
    usersTyping: UseSocketChatProps["usersTyping"],
    participants: UseSocketChatProps["participants"],
    session: Session | undefined,
    setUsersTyping: UseSocketChatProps["setUsersTyping"],
) {
    // Add every user that's typing
    const newTyping = JSON.parse(JSON.stringify(usersTyping)) as ParticipantWithoutChat[];
    for (const id of starting ?? []) {
        // Don't add yourself
        if (id === getCurrentUser(session).id) continue;
        // Don't add duplicates
        if (newTyping.some(p => p.user.id === id)) continue;
        // Don't add users that aren't in the chat
        const participant = participants.find(p => p.user.id === id);
        if (!participant) continue;
        newTyping.push(participant);
    }
    // Remove every user that stopped typing
    for (const id of stopping ?? []) {
        const index = newTyping.findIndex(p => p.user.id === id);
        if (index === -1) continue;
        newTyping.splice(index, 1);
    }
    // If newTyping is the same as usersTyping, don't update
    if (newTyping.length === usersTyping.length && newTyping.every((p, i) => p.user.id === usersTyping[i].user.id)) {
        return;
    }
    setUsersTyping(newTyping);
}

export function processParticipantsUpdates(
    { joining, leaving }: ChatSocketEventPayloads["participants"],
    participants: UseSocketChatProps["participants"],
    chat: UseSocketChatProps["chat"],
    setParticipants: UseSocketChatProps["setParticipants"],
) {
    if (!chat) return;

    // Remove cache data for old participants group
    const existingUserIds = participants.map(p => p.user.id);
    removeCookieMatchingChat(existingUserIds);

    // Create updated participants list
    let updatedParticipants = [...participants];
    if (joining) {
        // Filter out joining participants who are already present
        const joiningFiltered = joining.filter(j => !participants.some(p => p.user.id === j.user.id));
        updatedParticipants = [...updatedParticipants, ...joiningFiltered];
    }
    if (leaving) {
        updatedParticipants = updatedParticipants.filter(participant => !leaving.includes(participant.user.id) && !leaving.includes(participant.id));
    }

    const updatedUserIds = updatedParticipants.map(p => p.user.id);
    setCookieMatchingChat(chat.id, updatedUserIds);

    // Don't update if the participants are the same
    if (updatedParticipants.length === participants.length && updatedParticipants.every((p, i) => p.user.id === participants[i].user.id)) {
        return;
    }
    setParticipants(updatedParticipants);
}

/**
 * Processes incoming LLM tasks from the server, and sends them through 
 * a pubsub event.
 * @param payload The incoming LLM task data. Contains an array of full tasks and partial updates.
 * @param chatId The chat ID to which the tasks belong.
 */
export function processLlmTasks(
    payload: ChatSocketEventPayloads["llmTasks"],
    chatId: string | null | undefined,
) {
    if (!chatId) return;

    if (Array.isArray(payload.tasks) && payload.tasks.length > 0) {
        // Filter out tasks with no ID
        payload.tasks = payload.tasks.filter(task => !!task.taskId);
        PubSub.get().publish("chatTask", {
            chatId,
            tasks: {
                add: {
                    inactive: {
                        behavior: "onlyIfNoTaskType",
                        value: payload.tasks,
                    },
                },
            },
        });
        for (const task of payload.tasks) {
            upsertCookieTaskForChat(chatId, task);
        }
    }
    if (Array.isArray(payload.updates) && payload.updates.length > 0) {
        PubSub.get().publish("chatTask", {
            chatId,
            tasks: {
                update: payload.updates,
            },
        });
        for (const update of payload.updates) {
            updateCookiePartialTaskForChat(chatId, {
                ...update,
                lastUpdated: new Date().toISOString(),
            });
        }
    }
}

export function processResponseStream(
    { __type, botId, message }: ChatSocketEventPayloads["responseStream"],
    messageStreamRef: React.MutableRefObject<ChatSocketEventPayloads["responseStream"] | null>,
    setMessageStream: (stream: ChatSocketEventPayloads["responseStream"] | null) => void,
    throttledSetMessageStream: (stream: ChatSocketEventPayloads["responseStream"] | null) => void,
) {
    // Initialize ref if it doesn't exist
    if (!messageStreamRef.current) {
        messageStreamRef.current = { __type, message: "" };
    }

    // Add __type to stream
    messageStreamRef.current.__type = __type;

    // Check and update botId if new botId is provided and it is different from current
    if (botId && messageStreamRef.current.botId !== botId) {
        messageStreamRef.current.botId = botId;
        // This indicates the start of a new message with a new bot, so clear any old message
        messageStreamRef.current.message = "";
    }

    // Add to message if we're still streaming or it's an error. 
    // We keep the stream if there's an error in case the user want to 
    // read what part of the message was received
    if (__type === "stream" || __type === "error") {
        messageStreamRef.current.message += message;
    }

    // Remove message if it's the end. The full message will be received through the rest endpoint
    if (__type === "end") {
        messageStreamRef.current = null;
        // Update state immediately
        setMessageStream(null);
    } else {
        // Update state on throttle
        throttledSetMessageStream(messageStreamRef.current);
    }
}

const MESSAGE_STREAM_UPDATE_THROTTLE_MS = 100;

/** 
 * Handles the modification of a chat through web socket events, 
 * as well as the relevant localStorage caching.
 */
export function useSocketChat({
    addMessages,
    chat,
    editMessage,
    participants,
    removeMessages,
    setParticipants,
    setUsersTyping,
    usersTyping,
}: UseSocketChatProps) {
    const session = useContext(SessionContext);

    // Handle connection/disconnection
    useEffect(function connectToChatEffect() {
        if (!chat?.id || chat.id === DUMMY_ID) return;

        SocketService.get().emitEvent("joinChatRoom", { chatId: chat.id }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "ChatRoomJoinFailed", severity: "Error" });
                // If the response indicates that the chat was deleted or is unauthorized,
                // we should remove all references to this chat from local storage
                if (response.error === JOIN_CHAT_ROOM_ERRORS.ChatNotFoundOrUnauthorized) {
                    removeCookiesWithChatId(chat.id);
                }
            }
        });

        return () => {
            SocketService.get().emitEvent("leaveChatRoom", { chatId: chat.id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave chat room", response.error);
                }
            });
        };
    }, [chat?.id]);

    const messageStreamRef = useRef<ChatSocketEventPayloads["responseStream"] | null>(null);
    const [messageStream, setMessageStream] = useState<ChatSocketEventPayloads["responseStream"] | null>(null);
    const throttledSetMessageStreamHander = useCallback((messageStream: ChatSocketEventPayloads["responseStream"] | null) => {
        // Create copy of message to ensure proper re-rendering
        const messageStreamCopy = messageStream ? { ...messageStream } : null;
        setMessageStream(messageStreamCopy);
    }, []);
    const throttledSetMessageStream = useThrottle(throttledSetMessageStreamHander, MESSAGE_STREAM_UPDATE_THROTTLE_MS);

    // Store refs for parameters to reduce the number of dependencies of socket event handlers. 
    // This reduces the number of times the socket events are connected/disconnected.
    const chatRef = useRef(chat);
    chatRef.current = chat;
    const participantsRef = useRef(participants);
    participantsRef.current = participants;
    const usersTypingRef = useRef(usersTyping);
    usersTypingRef.current = usersTyping;

    // Handle incoming data
    useEffect(() => SocketService.get().onEvent("messages", (payload) => processMessages(payload, addMessages, editMessage, removeMessages)), [addMessages, editMessage, removeMessages]);
    useEffect(() => SocketService.get().onEvent("typing", (payload) => processTypingUpdates(payload, usersTypingRef.current, participantsRef.current, session, setUsersTyping)), [session, setUsersTyping]);
    useEffect(() => SocketService.get().onEvent("llmTasks", (payload) => processLlmTasks(payload, chat?.id)), [chat?.id]);
    useEffect(() => SocketService.get().onEvent("participants", (payload) => processParticipantsUpdates(payload, participantsRef.current, chatRef.current, setParticipants)), [setParticipants]);
    useEffect(() => SocketService.get().onEvent("responseStream", (payload) => processResponseStream(payload, messageStreamRef, setMessageStream, throttledSetMessageStream)), [throttledSetMessageStream]);

    return { messageStream };
}
