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

type UseMessageActionsProps = {
    chat: ChatShape;
    handleChatUpdate: (updatedChat?: ChatShape) => Promise<Chat>;
    language: string;
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
        console.log("in respondToTask", task);
        // Ignore if status is "completed" or "failed"
        if (["completed", "failed"].includes(task.status)) return;
        // Ignore if this isn't a task type that can be started/stopped
        if (["Start"].includes(task.task)) return;
        // Only respond if we can find the message
        if (!task.messageId) return;
        const message = chat.messages?.find((message) => message.id === task.messageId);
        if (!message) return;
        // If status is "suggested", trigger the task
        if (task.status === "suggested") {
            setCookieTaskForMessage(message.id, { ...task, status: "running" });
            updateTasksForMessage(message.id, [{ ...task, status: "running" }]);
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
                successCondition: (data) => data && data.success === true,
                errorMessage: () => ({ messageKey: "ActionFailed" }),
                onError: () => {
                    setCookieTaskForMessage(message.id, { ...task, status: "suggested" });
                    updateTasksForMessage(message.id, [{ ...task, status: "suggested" }]);
                },
            });
        }
        // If status is "running", attempt to pause/stop the task
        else if (task.status === "running") {
            //TODO
        }
    }, [chat, startTask, updateTasksForMessage]);

    return {
        postMessage,
        putMessage,
        regenerateResponse,
        respondToTask,
    };
};
