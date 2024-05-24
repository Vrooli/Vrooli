import { ChatParticipant, ChatSocketEventPayloads, DUMMY_ID, LlmTask, LlmTaskInfo, Session } from "@local/shared";
import { emitSocketEvent, onSocketEvent } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useContext, useEffect, useRef, useState } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieTasksForMessage, removeCookieMatchingChat, setCookieMatchingChat, setCookieTaskForMessage } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { useThrottle } from "./useThrottle";

type ParticipantWithoutChat = Omit<ChatParticipant, "chat">;

type UseWebSocketEventsProps = {
    addMessages: (newMessages: ChatMessageShape[]) => unknown;
    chat: ChatShape;
    editMessage: (editedMessage: ChatMessageShape) => unknown;
    participants: ParticipantWithoutChat[];
    removeMessages: (deletedIds: string[]) => unknown;
    setParticipants: (participants: ParticipantWithoutChat[]) => unknown;
    setUsersTyping: (updatedParticipants: ParticipantWithoutChat[]) => unknown;
    /** The active task being performed, if any */
    task?: LlmTask | `${LlmTask}` | string;
    updateTasksForMessage: (messageId: string, tasks: LlmTaskInfo[]) => unknown;
    usersTyping: ParticipantWithoutChat[];
}

export const processMessages = (
    { added, deleted, edited }: ChatSocketEventPayloads['messages'],
    addMessages: UseWebSocketEventsProps['addMessages'],
    removeMessages: UseWebSocketEventsProps['removeMessages'],
    editMessage: UseWebSocketEventsProps['editMessage'],
) => {
    if (Array.isArray(added) && added.length > 0) {
        addMessages(added.map(m => ({ ...m, status: "sent" })));
    }
    if (Array.isArray(deleted) && deleted.length > 0) {
        removeMessages(deleted);
    }
    if (Array.isArray(edited) && edited.length > 0) {
        edited.forEach(message => {
            editMessage({ ...message, status: "sent" });
        });
    }
};

export const processTypingUpdates = (
    { starting, stopping }: ChatSocketEventPayloads['typing'],
    usersTyping: UseWebSocketEventsProps['usersTyping'],
    participants: UseWebSocketEventsProps['participants'],
    session: Session | undefined,
    setUsersTyping: UseWebSocketEventsProps['setUsersTyping'],
) => {
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
};

export const processParticipantsUpdates = (
    { joining, leaving }: ChatSocketEventPayloads['participants'],
    participants: UseWebSocketEventsProps['participants'],
    chat: UseWebSocketEventsProps['chat'],
    task: UseWebSocketEventsProps['task'],
    setParticipants: UseWebSocketEventsProps['setParticipants'],
) => {
    // Remove cache data for old participants group
    const existingUserIds = participants.map(p => p.user.id);
    removeCookieMatchingChat(existingUserIds, task);

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
    setCookieMatchingChat(chat.id, updatedUserIds, task);

    // Don't update if the participants are the same
    if (updatedParticipants.length === participants.length && updatedParticipants.every((p, i) => p.user.id === participants[i].user.id)) {
        return;
    }
    setParticipants(updatedParticipants);
};

export const processLlmTasks = (
    { tasks, updates }: ChatSocketEventPayloads["llmTasks"],
    updateTasksForMessage: UseWebSocketEventsProps["updateTasksForMessage"],
) => {
    // Combine full tasks and updates into a single operation per messageId
    const combinedTasksByMessageId: Record<string, LlmTaskInfo[]> = {};

    // Initialize tasks
    (tasks ?? []).forEach(task => {
        if (!task.messageId || !task.id) return;
        if (!combinedTasksByMessageId[task.messageId]) combinedTasksByMessageId[task.messageId] = [];
        combinedTasksByMessageId[task.messageId].push(task);
    });

    // Apply updates
    (updates ?? []).forEach(update => {
        if (!update.messageId || !update.id) return;
        if (!combinedTasksByMessageId[update.messageId]) {
            // If no tasks were initially found for this messageId, initialize the array
            combinedTasksByMessageId[update.messageId] = [];
        }
        const existingTaskIndex = combinedTasksByMessageId[update.messageId].findIndex(task => task.id === update.id);
        if (existingTaskIndex > -1) {
            // Merge the update into the existing task
            combinedTasksByMessageId[update.messageId][existingTaskIndex] = {
                ...combinedTasksByMessageId[update.messageId][existingTaskIndex],
                ...update,
                lastUpdated: new Date().toISOString(),
            };
        } else {
            // Handle case where update is for a new task not yet in tasks
            combinedTasksByMessageId[update.messageId].push({
                ...update,
                lastUpdated: new Date().toISOString(),
            } as LlmTaskInfo);
        }
    });

    // Update cache and invoke callback for each messageId using helper functions
    Object.entries(combinedTasksByMessageId).forEach(([messageId, tasksForMessage]) => {
        tasksForMessage.forEach(task => {
            setCookieTaskForMessage(messageId, task); // Set or update task using helper
        });
        const updatedTasks = getCookieTasksForMessage(messageId);
        if (updatedTasks) {
            updateTasksForMessage(messageId, updatedTasks.tasks);
        }
    });
};

export const processResponseStream = (
    { __type, botId, message }: ChatSocketEventPayloads['responseStream'],
    messageStreamRef: React.MutableRefObject<ChatSocketEventPayloads["responseStream"] | null>,
    setMessageStream: (stream: ChatSocketEventPayloads["responseStream"] | null) => void,
    throttledSetMessageStream: (stream: ChatSocketEventPayloads["responseStream"] | null) => void,
) => {
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
};

/** 
 * Handles the modification of a chat through web socket events, 
 * as well as the relevant localStorage caching.
 */
export const useSocketChat = ({
    addMessages,
    chat,
    editMessage,
    participants,
    removeMessages,
    setParticipants,
    setUsersTyping,
    task,
    updateTasksForMessage,
    usersTyping,
}: UseWebSocketEventsProps) => {
    const session = useContext(SessionContext);

    // Handle connection/disconnection
    useEffect(() => {
        if (!chat?.id || chat.id === DUMMY_ID) return;

        emitSocketEvent("joinChatRoom", { chatId: chat.id }, (response) => {
            if (response.error) {
                PubSub.get().publish("snack", { messageKey: "ChatRoomJoinFailed", severity: "Error" });
            }
        });

        return () => {
            emitSocketEvent("leaveChatRoom", { chatId: chat.id }, (response) => {
                if (response.error) {
                    console.error("Failed to leave chat room", response.error);
                }
            });
        };
    }, [chat?.id]);

    const messageStreamRef = useRef<ChatSocketEventPayloads["responseStream"] | null>(null);
    const [messageStream, setMessageStream] = useState<ChatSocketEventPayloads["responseStream"] | null>(null);
    const throttledSetMessageStream = useThrottle((messageStream: ChatSocketEventPayloads["responseStream"] | null) => {
        // Create copy of message to ensure proper re-rendering
        const messageStreamCopy = messageStream ? { ...messageStream } : null;
        setMessageStream(messageStreamCopy);
    }, 100);

    // Handle incoming data
    useEffect(() => {
        const cleanupMessages = onSocketEvent("messages", (payload) => processMessages(payload, addMessages, removeMessages, editMessage));
        const cleanupTyping = onSocketEvent("typing", (payload) => processTypingUpdates(payload, usersTyping, participants, session, setUsersTyping));
        const cleanupLlmTasks = onSocketEvent("llmTasks", (payload) => processLlmTasks(payload, updateTasksForMessage));
        const cleanupParticipants = onSocketEvent("participants", (payload) => processParticipantsUpdates(payload, participants, chat, task, setParticipants));
        const cleanupResponseStream = onSocketEvent("responseStream", (payload) => processResponseStream(payload, messageStreamRef, setMessageStream, throttledSetMessageStream));
        return () => {
            cleanupMessages();
            cleanupTyping();
            cleanupLlmTasks();
            cleanupParticipants();
            cleanupResponseStream();
        };
    }, [addMessages, editMessage, chat.id, participants, removeMessages, session, usersTyping, setUsersTyping, task, updateTasksForMessage, setParticipants]);

    return { messageStream };
};
