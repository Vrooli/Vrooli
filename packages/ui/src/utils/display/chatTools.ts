import { AITaskInfo, LlmTask, MINUTES_10_MS, uuid } from "@local/shared";
import { getExistingAIConfig } from "api/ai.js";
import { SetLocation } from "route";

/** How long a task can be running until it is considered stale */
export const STALE_TASK_THRESHOLD_MS = MINUTES_10_MS;

/**
 * Determines if an Llm task is stale, meaning its status is 
 * probably out of date
 * @param taskInfo The task info to check
 * @returns True if the task is stale, false otherwise
 */
export function isTaskStale(taskInfo: AITaskInfo): boolean {
    if (!["Running", "Canceling"].includes(taskInfo.status)) {
        return false;
    }
    if (!taskInfo.lastUpdated) {
        return false;
    }
    const lastUpdated = new Date(taskInfo.lastUpdated);
    const now = new Date();
    const diff = now.getTime() - lastUpdated.getTime();
    return diff > STALE_TASK_THRESHOLD_MS;
}

/**
 * Generates task info for a suggested task, using only the LlmTask string
 * @param task The LlmTask to generate info for
 * @returns The task info
 */
export function taskToTaskInfo(task: LlmTask): AITaskInfo {
    const configData = getExistingAIConfig();
    let label: string = task;
    if (configData && configData.task) {
        const config = configData.task.config;
        if (config && config[task]) {
            label = config[task].label;
        }
    }
    return {
        label,
        lastUpdated: new Date().toISOString(),
        status: "Suggested",
        task,
        taskId: `task-${uuid()}`,
        properties: null,
    };
}

/**
 * Handles what happens when you click on a suggested or active task, where its 
 * behavior depends on the task's status.
 * 
 * Here's each status' behavior:
 * - "Canceling" => "Canceling" - reattempts to cancel
 * - "Completed" => "Completed" - 
 * 
 * @param task The task to handle
 * @param chatId The ID of the chat this task is associated with
 * @param setLocation The function to set the location of the page
 */
export function handleTaskClick(
    task: AITaskInfo,
    chatId: string,
    setLocation: SetLocation,
) {
    return;//TODO instead of doing this on click, have click only select/unselect. Active task can show icon for this stuff, such as open, remove, run, etc.
    // if (["Completed"].includes(task.status)) {
    //     // If the result link is available, open it
    //     if (task.resultLink) {
    //         openLink(setLocation, task.resultLink);
    //     } else {
    //         console.warn("Ignoring completed task: no result link", task);
    //     }
    //     return;
    // }
    // // Ignore if this isn't a valid task type ("Start" is a placeholder task for chatting 
    // // with the user until we have a real task to perform, so we don't want to trigger it here)
    // if (["Start"].includes(task.task)) {
    //     console.warn("Ignoring task: invalid task type", task);
    //     return;
    // }
    // const originalTask = { ...task };
    // // If task is not running and not completed, start the task
    // if (["Suggested", "Canceled", "Failed"].includes(task.status)) {
    //     let updatedTask = {
    //         ...task,
    //         lastUpdated: new Date().toISOString(),
    //         status: "Running" as const,
    //     } as LlmTaskInfo;
    //     setCookieTaskForChat(chatId, updatedTask);
    //     PubSub.get().publish("chatTask", {
    //         chatId,
    //         setActiveTask: { behavior: "force", value: updatedTask },
    //     });
    //     fetchLazyWrapper<StartLlmTaskInput, Success>({
    //         fetch: startTask,
    //         inputs: {
    //             botId: message.user?.id ?? VALYXA_ID,
    //             label: task.label,
    //             messageId: message.id ?? "",
    //             properties: task.properties ?? {},
    //             task: task.task as LlmTask,
    //             taskId: task.taskId,
    //         },
    //         spinnerDelay: null, // Disable spinner since this is a background task
    //         successCondition: (data) => data && data.success === true,
    //         errorMessage: () => ({ messageKey: "ActionFailed" }),
    //         onError: () => {
    //             updatedTask = {
    //                 ...updatedTask,
    //                 lastUpdated: new Date().toISOString(),
    //                 status: "Failed" as const,
    //             };
    //             setCookieTaskForChat(chatId, updatedTask);
    //             PubSub.get().publish("chatTask", {
    //                 chatId,
    //                 setActiveTask: { behavior: "force", value: null },
    //                 addSuggestedTasks: { behavior: "add", value: [updatedTask] },
    //             });
    //         },
    //         // Socket event should update task data on success, so we don't need to do anything here
    //     });
    // }
    // // If status is "Running", attempt to cancel the task
    // else if (["Running", "Canceling"].includes(task.status)) {
    //     const updatedTask = {
    //         ...task,
    //         lastUpdated: new Date().toISOString(),
    //         status: "Canceling" as const,
    //     };
    //     const updatedTaskList = (tasks[message.id] ?? []).map((t) => t.taskId === task.taskId ? updatedTask : t);
    //     setCookieTaskForMessage(message.id, updatedTask);
    //     updateTasksForMessage(message.id, updatedTaskList);
    //     fetchLazyWrapper<CancelTaskInput, Success>({
    //         fetch: cancelTask,
    //         inputs: { taskId: task.taskId, taskType: TaskType.Llm },
    //         spinnerDelay: null, // Disable spinner since this is a background task
    //         successCondition: (data) => data && data.success === true,
    //         errorMessage: () => ({ messageKey: "ActionFailed" }),
    //         onError: () => {
    //             setCookieTaskForMessage(message.id, originalTask);
    //             updateTasksForMessage(message.id, originalTaskList);
    //         },
    //         onSuccess: () => {
    //             const canceledTask = {
    //                 ...task,
    //                 lastUpdated: new Date().toISOString(),
    //                 status: "Suggested" as const,
    //             };
    //             const canceledTaskList = (tasks[message.id] ?? []).map((t) => t.taskId === task.taskId ? canceledTask : t);
    //             setCookieTaskForMessage(message.id, canceledTask);
    //             updateTasksForMessage(message.id, canceledTaskList);
    //         },
    //     });
    // } else {
    //     console.warn("Ignoring task: invalid status", task);
    // }
}
