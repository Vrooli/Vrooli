import { LlmTaskInfo, MINUTES_10_MS } from "@local/shared";

/** How long a task can be running until it is considered stale */
export const STALE_TASK_THRESHOLD_MS = MINUTES_10_MS;

/**
 * Determines if an Llm task is stale, meaning its status is 
 * probably out of date
 * @param taskInfo The task info to check
 * @returns True if the task is stale, false otherwise
 */
export const isTaskStale = (taskInfo: LlmTaskInfo): boolean => {
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
};
