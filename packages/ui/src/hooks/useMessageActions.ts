import { Chat, DUMMY_ID, LlmTask, LlmTaskInfo, RegenerateResponseInput, StartTaskInput, Success, VALYXA_ID, endpointPostRegenerateResponse, endpointPostStartTask, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { setCookieTaskForMessage } from "utils/cookies";
import { PubSub } from "utils/pubsub";
import { ChatShape } from "utils/shape/models/chat";
import { ChatMessageShape } from "utils/shape/models/chatMessage";
import { useLazyFetch } from "./useLazyFetch";
import { MessageTree } from "./useMessageTree";

type UseMessageActionsProps = {
    chat: ChatShape;
    handleChatUpdate: (updatedChat?: ChatShape) => Promise<Chat>;
    language: string;
    tasks: Record<string, LlmTaskInfo[]>;
    tree: MessageTree<ChatMessageShape>;
    updateTasksForMessage: (messageId: string, tasks: LlmTaskInfo[]) => unknown
};

/**
 * Performs various chat message actions by 
 * updating the server database and local tree structure
 */
export const useMessageActions = ({
    chat,
    handleChatUpdate,
    language,
    tasks,
    tree,
    updateTasksForMessage,
}: UseMessageActionsProps) => {
    const session = useContext(SessionContext);

    /** Commit a new message */
    const postMessage = useCallback((text: string) => {
        if (text.trim() === "") return;
        const newMessage: ChatMessageShape = {
            __typename: "ChatMessage" as const,
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            status: "sending",
            versionIndex: 0,
            chat: {
                __typename: "Chat" as const,
                id: chat.id,
            },
            reactionSummaries: [],
            translations: [{
                __typename: "ChatMessageTranslation" as const,
                id: DUMMY_ID,
                language,
                text,
            }],
            user: {
                __typename: "User" as const,
                id: getCurrentUser(session).id ?? "",
                isBot: false,
                name: getCurrentUser(session).name ?? undefined,
            },
        };
        PubSub.get().publish("chatMessage", {
            chatId: chat.id,
            data: { newMessage },
        });
        handleChatUpdate({
            ...chat,
            messages: [...(chat.messages ?? []), newMessage],
        }).then(() => {
            PubSub.get().publish("chatMessage", {
                chatId: chat.id,
                data: { updatedMessage: { ...newMessage, status: "sent" } },
            });
        }).catch(() => {
            PubSub.get().publish("chatMessage", {
                chatId: chat.id,
                data: { updatedMessage: { ...newMessage, status: "failed" } },
            });
        });
    }, [chat, language, handleChatUpdate, session]);

    /** Commit an existing message */
    const putMessage = useCallback((updatedMessage: ChatMessageShape) => {
        const isOwn = updatedMessage.user?.id === getCurrentUser(session).id;
        const existingMessage = chat.messages?.find((message) => message.id === updatedMessage.id);
        if (!existingMessage) return;
        PubSub.get().publish("chatMessage", {
            chatId: chat.id,
            data: { updatedMessage },
        });
        handleChatUpdate({
            ...chat,
            messages: (chat.messages ?? []).map((message) => {
                if (message.id === updatedMessage.id) {
                    return updatedMessage;
                }
                return message;
            }),
        }).catch(() => {
            PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
            // If own message, mark as fail
            if (isOwn) {
                PubSub.get().publish("chatMessage", {
                    chatId: chat.id,
                    data: { updatedMessage: { ...updatedMessage, status: "failed" } },
                });
            }
            // Otherwise, reverse
            else {
                PubSub.get().publish("chatMessage", {
                    chatId: chat.id,
                    data: { updatedMessage: existingMessage },
                });
            }
        });
    }, [chat, handleChatUpdate, session]);

    const [regenerate] = useLazyFetch<RegenerateResponseInput, Success>(endpointPostRegenerateResponse);
    /** Regenerate a bot response */
    const regenerateResponse = useCallback((message: ChatMessageShape) => {
        fetchLazyWrapper<RegenerateResponseInput, Success>({
            fetch: regenerate,
            inputs: { messageId: message.id },
            successCondition: (data) => data && data.success === true,
            errorMessage: () => ({ messageKey: "ActionFailed" }),
        });
    }, [regenerate]);

    const [startTask] = useLazyFetch<StartTaskInput, Success>(endpointPostStartTask);
    /** 
     * Handle a suggested task, depending on its state
     */
    const respondToTask = useCallback((task: LlmTaskInfo) => {
        // Ignore if status is "completed" or "failed"
        if (["completed", "failed"].includes(task.status)) {
            console.warn("Ignoring task: invalid status", task);
            return;
        }
        // Ignore if this isn't a valid task type ("Start" is a placeholder task for chatting 
        // with the user until we have a real task to perform, so we don't want to trigger it here)
        if (["Start"].includes(task.task)) {
            console.warn("Ignoring task: invalid task type", task);
            return;
        }
        // Only respond if we can find the message
        if (!task.messageId) {
            console.warn("Ignoring task: no message ID", task);
            return;
        }
        const messageNode = tree.map.get(task.messageId);
        if (!messageNode) {
            console.warn("Ignoring task: could not find associated message", task, tree.map);
            return;
        }
        // If status is "suggested", trigger the task
        if (task.status === "suggested") {
            const { message } = messageNode;
            const originalTask = { ...task };
            const originalTaskList = tasks[message.id] ?? [];
            const updatedTask = {
                ...task,
                lastUpdated: new Date().toISOString(),
                status: "running" as const,
            };
            const updatedTaskList = (tasks[message.id] ?? []).map((t) => t.id === task.id ? updatedTask : t);
            setCookieTaskForMessage(message.id, updatedTask);
            updateTasksForMessage(message.id, updatedTaskList);
            fetchLazyWrapper<StartTaskInput, Success>({
                fetch: startTask,
                inputs: {
                    botId: message.user?.id ?? VALYXA_ID,
                    label: task.label,
                    messageId: message.id ?? "",
                    properties: task.properties ?? {},
                    task: task.task as LlmTask,
                    taskId: task.id,
                },
                spinnerDelay: null, // Disable spinner since this is a background task
                successCondition: (data) => data && data.success === true,
                errorMessage: () => ({ messageKey: "ActionFailed" }),
                onError: () => {
                    setCookieTaskForMessage(message.id, originalTask);
                    updateTasksForMessage(message.id, originalTaskList);
                },
            });
        }
        // If status is "running", attempt to pause/stop the task
        else if (task.status === "running") {
            //TODO
        }
    }, [startTask, tasks, tree.map, updateTasksForMessage]);

    return {
        postMessage,
        putMessage,
        regenerateResponse,
        respondToTask,
    };
};
