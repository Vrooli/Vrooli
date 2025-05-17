import { Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import { addJobToQueue, changeTaskStatus, getTaskStatuses } from "../queueHelper.js";
import { SandboxProcessPayload } from "./types.js";

export type SandboxTestPayload = {
    __process: "Test";
};

export type SandboxRequestPayload = SandboxProcessPayload | SandboxTestPayload;

export function runSandboxedCode(data: SandboxProcessPayload): Promise<Success> {
    return addJobToQueue(sandboxQueue.getQueue(), data, {});
}

/**
 * Update a task's status
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export async function changeSandboxTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return changeTaskStatus(sandboxQueue.getQueue(), jobId, status, userId);
}

/**
 * Get the statuses of multiple sandbox tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getSandboxTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    return getTaskStatuses(sandboxQueue.getQueue(), taskIds);
}
