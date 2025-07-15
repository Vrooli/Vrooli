import { DUMMY_ID, JOIN_CHAT_ROOM_ERRORS, type ChatMessageShape, type ChatParticipantShape, type ChatShape, type ChatSocketEventPayloads, type Session, type StreamErrorPayload } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useRef, useState } from "react";
import { SocketService } from "../api/socket.js";
import { SessionContext } from "../contexts/session.js";
import { useDebugStore } from "../stores/debugStore.js";
import { getCurrentUser } from "../utils/authentication/session.js";
import { removeCookieMatchingChat, removeCookiesWithChatId, setCookieMatchingChat, updateCookiePartialTaskForChat, upsertCookieTaskForChat } from "../utils/localStorage.js";
import { PubSub } from "../utils/pubsub.js";
import { useThrottle } from "./useThrottle.js";

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

// Add constants for magic numbers
const MAX_MESSAGE_PREVIEW_LENGTH = 50;

// Track active join attempts globally to prevent duplicates
const activeJoinAttempts = new Map<string, boolean>();

interface ClientAccumulatedStreamState {
    __type: "stream" | "end" | "error";
    botId?: string;
    accumulatedMessage: string;
    error?: StreamErrorPayload;
}

export function processMessages(
    { added, updated, removed }: ChatSocketEventPayloads["messages"],
    addMessages: UseSocketChatProps["addMessages"],
    editMessage: UseSocketChatProps["editMessage"],
    removeMessages: UseSocketChatProps["removeMessages"],
    chatId?: string | null,
) {
    // Add debug data for received messages
    if (Array.isArray(added) && added.length > 0) {
        // Debug: Track new messages
        useDebugStore.getState().addData(`chat-messages-${chatId || "unknown"}`, {
            event: "received-messages",
            count: added.length,
            messages: added.map(m => ({
                id: m.id,
                // Safe access to content field which might not exist in the type
                preview: typeof m["content"] === "string"
                    ? m["content"].substring(0, MAX_MESSAGE_PREVIEW_LENGTH) +
                    (m["content"].length > MAX_MESSAGE_PREVIEW_LENGTH ? "..." : "")
                    : "(no content)",
            })),
            timestamp: new Date().toISOString(),
        });

        addMessages(added.map(m => ({ ...m, status: "sent" })));
    }
    if (Array.isArray(updated) && updated.length > 0) {
        // Debug: Track updated messages
        useDebugStore.getState().addData(`chat-messages-${chatId || "unknown"}`, {
            event: "updated-messages",
            count: updated.length,
            messages: updated.map(m => ({ id: m.id })),
            timestamp: new Date().toISOString(),
        });

        updated.forEach(message => {
            editMessage({ ...message, status: "sent" });
        });
    }
    if (Array.isArray(removed) && removed.length > 0) {
        // Debug: Track removed messages
        useDebugStore.getState().addData(`chat-messages-${chatId || "unknown"}`, {
            event: "removed-messages",
            count: removed.length,
            messageIds: removed,
            timestamp: new Date().toISOString(),
        });

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
    const existingPublicIds = participants.map(p => p.user.publicId);
    removeCookieMatchingChat(existingPublicIds);

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

    const updatedPublicIds = updatedParticipants.map(p => p.user.publicId);
    setCookieMatchingChat(chat.id, updatedPublicIds);

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
    payload: ChatSocketEventPayloads["responseStream"],
    messageStreamRef: React.MutableRefObject<ClientAccumulatedStreamState | null>,
    setMessageStream: (stream: ClientAccumulatedStreamState | null) => void,
    throttledSetMessageStream: (stream: ClientAccumulatedStreamState | null) => void,
) {
    const { __type, botId, chunk, error: streamError } = payload;

    // Initialize or reset ref if botId changes or ref is null
    if (!messageStreamRef.current || (botId && messageStreamRef.current.botId !== botId)) {
        messageStreamRef.current = { __type, botId, accumulatedMessage: "" };
        if (__type === "error" && streamError) {
            messageStreamRef.current.error = streamError;
        }
    } else {
        // Update type for existing stream if it's not null
        messageStreamRef.current.__type = __type;
    }

    if (__type === "stream" && chunk) {
        messageStreamRef.current.accumulatedMessage += chunk;
        messageStreamRef.current.error = undefined; // Clear previous error if now streaming
    } else if (__type === "error" && streamError) {
        messageStreamRef.current.error = streamError;
        // Optionally, set accumulatedMessage from error or leave as is
        // messageStreamRef.current.accumulatedMessage = streamError.message;
    }

    if (__type === "end") {
        messageStreamRef.current = null;
        setMessageStream(null); // Immediate update
        return; // Avoid throttled update if ended
    }

    // Update state on throttle only if ref is not null
    if (messageStreamRef.current) {
        throttledSetMessageStream({ ...messageStreamRef.current });
    }
}

export function processModelReasoningStream(
    payload: ChatSocketEventPayloads["modelReasoningStream"],
    reasoningStreamRef: React.MutableRefObject<ClientAccumulatedStreamState | null>,
    setReasoningStream: (stream: ClientAccumulatedStreamState | null) => void,
    throttledSetReasoningStream: (stream: ClientAccumulatedStreamState | null) => void,
) {
    const { __type, botId, chunk, error: streamError } = payload;

    if (!reasoningStreamRef.current || (botId && reasoningStreamRef.current.botId !== botId)) {
        reasoningStreamRef.current = { __type, botId, accumulatedMessage: "" };
        if (__type === "error" && streamError) {
            reasoningStreamRef.current.error = streamError;
        }
    } else {
        reasoningStreamRef.current.__type = __type;
    }

    if (__type === "stream" && chunk) {
        reasoningStreamRef.current.accumulatedMessage += chunk;
        reasoningStreamRef.current.error = undefined;
    } else if (__type === "error" && streamError) {
        reasoningStreamRef.current.error = streamError;
    }

    if (__type === "end") {
        reasoningStreamRef.current = null;
        setReasoningStream(null);
        return;
    }

    if (reasoningStreamRef.current) {
        throttledSetReasoningStream({ ...reasoningStreamRef.current });
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
    const addDebugData = useDebugStore(state => state.addData);
    // Add a ref to track if we've already joined a chat
    const joinedChatsRef = useRef<Set<string>>(new Set());

    // Handle connection/disconnection
    useEffect(function connectToChatEffect() {
        if (!chat?.id || chat.id === DUMMY_ID) return;
        const debugTrace = `chat-room-${chat.id}`;

        // Prevent multiple simultaneous join attempts for the same chat
        if (activeJoinAttempts.get(chat.id)) {
            console.debug(`Prevented duplicate join attempt for chat ${chat.id}`);
            return;
        }

        // Only join if we haven't joined this chat before
        if (!joinedChatsRef.current.has(chat.id)) {
            // Mark this chat as having an active join attempt
            activeJoinAttempts.set(chat.id, true);

            // Debug: log chat room join attempt
            addDebugData(debugTrace, {
                event: "join-attempt",
                chatId: chat.id,
                timestamp: new Date().toISOString(),
            });

            SocketService.get().emitEvent("joinChatRoom", { chatId: chat.id }, (response) => {
                // Clear the active join attempt flag
                activeJoinAttempts.delete(chat.id);

                if (response.error) {
                    // Debug: log join error
                    addDebugData(debugTrace, {
                        event: "join-error",
                        chatId: chat.id,
                        error: response.error,
                        timestamp: new Date().toISOString(),
                    });

                    PubSub.get().publish("snack", { messageKey: "ChatRoomJoinFailed", severity: "Error" });
                    // If the response indicates that the chat was deleted or is unauthorized,
                    // we should remove all references to this chat from local storage
                    if (response.error === JOIN_CHAT_ROOM_ERRORS.ChatNotFoundOrUnauthorized) {
                        removeCookiesWithChatId(chat.id);
                    }
                } else {
                    // Mark this chat as joined
                    joinedChatsRef.current.add(chat.id);

                    // Debug: log successful join
                    addDebugData(debugTrace, {
                        event: "join-success",
                        chatId: chat.id,
                        timestamp: new Date().toISOString(),
                    });
                }
            });
        }

        // Always return a cleanup function that properly leaves the chat room
        return () => {
            // Only attempt to leave if we previously joined and there's no active join attempt
            if (joinedChatsRef.current.has(chat.id) && !activeJoinAttempts.get(chat.id)) {
                // Debug: log chat room leave
                addDebugData(debugTrace, {
                    event: "leave-attempt",
                    chatId: chat.id,
                    timestamp: new Date().toISOString(),
                });

                SocketService.get().emitEvent("leaveChatRoom", { chatId: chat.id }, (response) => {
                    if (response.error) {
                        // Debug: log leave error
                        addDebugData(debugTrace, {
                            event: "leave-error",
                            chatId: chat.id,
                            error: response.error,
                            timestamp: new Date().toISOString(),
                        });

                        console.error("Failed to leave chat room", response.error);
                    } else {
                        // Remove chat from joined set
                        joinedChatsRef.current.delete(chat.id);

                        // Debug: log successful leave
                        addDebugData(debugTrace, {
                            event: "leave-success",
                            chatId: chat.id,
                            timestamp: new Date().toISOString(),
                        });
                    }
                });
            }
        };
    }, [chat?.id, addDebugData]);

    const messageStreamRef = useRef<ClientAccumulatedStreamState | null>(null);
    const [messageStream, setMessageStream] = useState<ClientAccumulatedStreamState | null>(null);
    const throttledSetMessageStreamHander = useCallback((stream: ClientAccumulatedStreamState | null) => {
        // Create copy of message to ensure proper re-rendering
        const streamCopy = stream ? { ...stream } : null;
        setMessageStream(streamCopy);
    }, []);
    const throttledSetMessageStream = useThrottle(throttledSetMessageStreamHander, MESSAGE_STREAM_UPDATE_THROTTLE_MS);

    const modelReasoningStreamRef = useRef<ClientAccumulatedStreamState | null>(null);
    const [modelReasoningStream, setModelReasoningStream] = useState<ClientAccumulatedStreamState | null>(null);
    const throttledSetModelReasoningStreamHandler = useCallback((stream: ClientAccumulatedStreamState | null) => {
        const streamCopy = stream ? { ...stream } : null;
        setModelReasoningStream(streamCopy);
    }, []);
    const throttledSetModelReasoningStream = useThrottle(throttledSetModelReasoningStreamHandler, MESSAGE_STREAM_UPDATE_THROTTLE_MS);

    const [botStatuses, setBotStatuses] = useState<Record<string, ChatSocketEventPayloads["botStatusUpdate"]>>({});

    // Store refs for parameters to reduce the number of dependencies of socket event handlers. 
    // This reduces the number of times the socket events are connected/disconnected.
    const chatRef = useRef(chat);
    chatRef.current = chat;
    const participantsRef = useRef(participants);
    participantsRef.current = participants;
    const usersTypingRef = useRef(usersTyping);
    usersTypingRef.current = usersTyping;

    // Handle incoming data
    useEffect(() => SocketService.get().onEvent("messages", (payload) => processMessages(payload, addMessages, editMessage, removeMessages, chat?.id)), [addMessages, editMessage, removeMessages, chat?.id]);
    useEffect(() => SocketService.get().onEvent("typing", (payload) => processTypingUpdates(payload, usersTypingRef.current, participantsRef.current, session, setUsersTyping)), [session, setUsersTyping]);
    useEffect(() => SocketService.get().onEvent("llmTasks", (payload) => processLlmTasks(payload, chat?.id)), [chat?.id]);
    useEffect(() => SocketService.get().onEvent("participants", (payload) => processParticipantsUpdates(payload, participantsRef.current, chatRef.current, setParticipants)), [setParticipants]);
    useEffect(() => SocketService.get().onEvent("responseStream", (payload) => processResponseStream(payload, messageStreamRef, setMessageStream, throttledSetMessageStream)), [throttledSetMessageStream]);
    useEffect(() => SocketService.get().onEvent("modelReasoningStream", (payload) => processModelReasoningStream(payload, modelReasoningStreamRef, setModelReasoningStream, throttledSetModelReasoningStream)), [throttledSetModelReasoningStream]);

    useEffect(() => {
        // eslint-disable-next-line func-style
        const handleBotStatusUpdate = (payload: ChatSocketEventPayloads["botStatusUpdate"]) => {
            if (chatRef.current && payload.chatId === chatRef.current.id) {
                setBotStatuses(prevStatuses => ({
                    ...prevStatuses,
                    [payload.botId]: payload,
                }));
            }
        };

        const cleanup = SocketService.get().onEvent("botStatusUpdate", handleBotStatusUpdate);

        return () => {
            if (typeof cleanup === "function") {
                cleanup();
            }
        };
    }, [chatRef.current?.id]);

    const requestResponseCancellation = useCallback((chatIdToCancel: string) => {
        if (!chatIdToCancel) {
            console.warn("requestResponseCancellation: chatIdToCancel is required.");
            return;
        }
        addDebugData(`chat-room-${chatIdToCancel}`, {
            event: "request-cancellation-sent",
            chatId: chatIdToCancel,
            timestamp: new Date().toISOString(),
        });
        SocketService.get().emitEvent("requestCancellation", { chatId: chatIdToCancel }, (response) => {
            if (response.error) {
                addDebugData(`chat-room-${chatIdToCancel}`, {
                    event: "request-cancellation-error",
                    chatId: chatIdToCancel,
                    error: response.error,
                    message: response.message,
                    timestamp: new Date().toISOString(),
                });
                console.error("Failed to request response cancellation:", response.error, response.message);
                // Optionally, show a snackbar or notification to the user about the failure
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error", messageVariables: { error: response.message || response.error } });
            } else {
                addDebugData(`chat-room-${chatIdToCancel}`, {
                    event: "request-cancellation-acked",
                    chatId: chatIdToCancel,
                    message: response.message,
                    timestamp: new Date().toISOString(),
                });
                // Optionally, show a snackbar or notification for acknowledgement
                // PubSub.get().publish("snack", { messageKey: "RequestCancellationSent", severity: "Info" });
            }
        });
    }, [addDebugData]);

    return { messageStream, modelReasoningStream, botStatuses, requestResponseCancellation };
}
