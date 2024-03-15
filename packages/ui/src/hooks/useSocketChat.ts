import { ChatParticipant, DUMMY_ID, LlmTask, LlmTaskInfo } from "@local/shared";
import { emitSocketEvent, onSocketEvent, socket } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useContext, useEffect } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { getCookieTasksForMessage, removeCookieMatchingChat, setCookieMatchingChat, setCookieTaskForMessage } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";

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

    // Handle incoming data
    useEffect(() => {
        onSocketEvent("messages", ({ added, deleted, edited }) => {
            if (added) {
                addMessages(added.map(m => ({ ...m, status: "sent" })));
            }
            if (deleted) {
                removeMessages(deleted);
            }
            if (edited) {
                for (const message of edited) {
                    editMessage({ ...message, status: "sent" });
                }
            }
        });
        onSocketEvent("typing", ({ starting, stopping }) => {
            // Add every user that's typing
            const newTyping = [...usersTyping];
            for (const id of starting ?? []) {
                // Never add yourself
                if (id === getCurrentUser(session).id) continue;
                if (newTyping.some(p => p.user.id === id)) continue;
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
            setUsersTyping(newTyping);
        });
        onSocketEvent("llmTasks", ({ tasks }) => {
            const tasksByMessageId: Record<string, LlmTaskInfo[]> = tasks.reduce((acc, task) => {
                if (!task.messageId) return acc;
                if (!acc[task.messageId]) acc[task.messageId] = [];
                acc[task.messageId].push(task);
                return acc;
            }, {});

            // Update cache and invoke callback for each messageId
            Object.entries(tasksByMessageId).forEach(([messageId, tasksForMessage]) => {
                tasksForMessage.forEach(task => {
                    setCookieTaskForMessage(messageId, task);
                });
                const updatedTasks = getCookieTasksForMessage(messageId);
                console.log("got message tasks", updatedTasks, messageId, tasksByMessageId);
                if (updatedTasks) {
                    updateTasksForMessage(messageId, updatedTasks.tasks);
                }
            });
        });
        onSocketEvent("participants", ({ joining, leaving }) => {
            // Remove cache data for old participants group
            const existingUserIds = participants.map(p => p.user.id);
            removeCookieMatchingChat(existingUserIds, task);

            // Create updated participants list
            let updatedParticipants = [...participants] as ParticipantWithoutChat[];
            if (joining) {
                updatedParticipants = [...updatedParticipants, ...joining];
            }
            if (leaving) {
                // Should be filtering by participant's user ID, but we'll also filter by participant ID just in case
                updatedParticipants = updatedParticipants.filter(participant => !leaving.includes(participant.user.id) && !leaving.includes(participant.id));
            }

            // Update the participants in the chat and cache
            setParticipants(updatedParticipants);
            const updatedUserIds = updatedParticipants.map(p => p.user.id);
            setCookieMatchingChat(chat.id, updatedUserIds, task);
        });
        return () => {
            // Remove event handlers
            socket.off("messages");
            socket.off("typing");
        };
    }, [addMessages, editMessage, chat.id, participants, removeMessages, session, usersTyping, setUsersTyping, task, updateTasksForMessage, setParticipants]);
};
