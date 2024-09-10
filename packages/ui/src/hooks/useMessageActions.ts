import { CancelTaskInput, Chat, ChatMessageShape, ChatShape, DUMMY_ID, LlmTask, LlmTaskInfo, RegenerateResponseInput, StartLlmTaskInput, Success, TaskType, VALYXA_ID, endpointPostCancelTask, endpointPostRegenerateResponse, endpointPostStartLlmTask, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { SessionContext } from "contexts";
import { useCallback, useContext } from "react";
import { getCurrentUser } from "utils/authentication/session";
import { setCookieTaskForMessage } from "utils/cookies";
import { PubSub } from "utils/pubsub";
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
export function useMessageActions({
    chat,
    handleChatUpdate,
    language,
    tasks,
    tree,
    updateTasksForMessage,
}: UseMessageActionsProps) {
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
        console.log("yeeeet in postMessage", newMessage);
        handleChatUpdate({
            ...chat,
            messages: [...(chat.messages ?? []), newMessage],
        }).catch(() => {
            PubSub.get().publish("snack", { messageKey: "ActionFailed", severity: "Error" });
        });
    }, [chat, language, handleChatUpdate, session]);

    /** Commit an existing message */
    const putMessage = useCallback((updatedMessage: ChatMessageShape) => {
        console.log("yeeet in putMessage");
        const isOwn = updatedMessage.user?.id === getCurrentUser(session).id;
        const existingMessage = chat.messages?.find((message) => message.id === updatedMessage.id);
        if (!existingMessage) return;
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

    const [startTask] = useLazyFetch<StartLlmTaskInput, Success>(endpointPostStartLlmTask);
    const [cancelTask] = useLazyFetch<CancelTaskInput, Success>(endpointPostCancelTask);

    /** 
     * Handle a suggested task, depending on its state
     */
    const respondToTask = useCallback((task: LlmTaskInfo) => {
        // Ignore if status is "Completed"
        if (["Completed"].includes(task.status)) {
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
        const { message } = messageNode;
        const originalTask = { ...task };
        const originalTaskList = tasks[message.id] ?? [];
        // If task is not running and not completed, start the task
        if (["Suggested", "Canceled", "Failed"].includes(task.status)) {
            const updatedTask = {
                ...task,
                lastUpdated: new Date().toISOString(),
                status: "Running" as const,
            };
            const updatedTaskList = (tasks[message.id] ?? []).map((t) => t.taskId === task.taskId ? updatedTask : t);
            setCookieTaskForMessage(message.id, updatedTask);
            updateTasksForMessage(message.id, updatedTaskList);
            fetchLazyWrapper<StartLlmTaskInput, Success>({
                fetch: startTask,
                inputs: {
                    botId: message.user?.id ?? VALYXA_ID,
                    label: task.label,
                    messageId: message.id ?? "",
                    properties: task.properties ?? {},
                    task: task.task as LlmTask,
                    taskId: task.taskId,
                },
                spinnerDelay: null, // Disable spinner since this is a background task
                successCondition: (data) => data && data.success === true,
                errorMessage: () => ({ messageKey: "ActionFailed" }),
                onError: () => {
                    setCookieTaskForMessage(message.id, originalTask);
                    updateTasksForMessage(message.id, originalTaskList);
                },
                // Socket event should update task data on success, so we don't need to do anything here
            });
        }
        // If status is "Running", attempt to cancel the task
        else if (["Running", "Canceling"].includes(task.status)) {
            const updatedTask = {
                ...task,
                lastUpdated: new Date().toISOString(),
                status: "Canceling" as const,
            };
            const updatedTaskList = (tasks[message.id] ?? []).map((t) => t.taskId === task.taskId ? updatedTask : t);
            setCookieTaskForMessage(message.id, updatedTask);
            updateTasksForMessage(message.id, updatedTaskList);
            fetchLazyWrapper<CancelTaskInput, Success>({
                fetch: cancelTask,
                inputs: { taskId: task.taskId, taskType: TaskType.Llm },
                spinnerDelay: null, // Disable spinner since this is a background task
                successCondition: (data) => data && data.success === true,
                errorMessage: () => ({ messageKey: "ActionFailed" }),
                onError: () => {
                    setCookieTaskForMessage(message.id, originalTask);
                    updateTasksForMessage(message.id, originalTaskList);
                },
                onSuccess: () => {
                    const canceledTask = {
                        ...task,
                        lastUpdated: new Date().toISOString(),
                        status: "Suggested" as const,
                    };
                    const canceledTaskList = (tasks[message.id] ?? []).map((t) => t.taskId === task.taskId ? canceledTask : t);
                    setCookieTaskForMessage(message.id, canceledTask);
                    updateTasksForMessage(message.id, canceledTaskList);
                },
            });
        } else {
            console.warn("Ignoring task: invalid status", task);
        }
    }, [cancelTask, startTask, tasks, tree.map, updateTasksForMessage]);

    return {
        postMessage,
        putMessage,
        regenerateResponse,
        respondToTask,
    };
}
