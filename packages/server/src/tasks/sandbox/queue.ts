import { type Success, type TaskStatus, type TaskStatusInfo } from "@local/shared";
import { QueueService } from "../queues.js";
import { QueueTaskType, type SandboxTask } from "../taskTypes.js";

/**
 * Schedule a sandbox execution job.
 * @param data Omitted fields __process and status will be set internally.
 */
export function processSandbox(
    data: Omit<SandboxTask, "type" | "status">,
): Promise<Success> {
    return QueueService.get().sandbox.add(
        { ...data, type: QueueTaskType.SANDBOX_EXECUTION, status: "Scheduled" },
    )
        .then(() => ({ __typename: "Success" as const, success: true }))
        .catch(() => ({ __typename: "Success" as const, success: false }));
}

/**
 * Change the status of an existing sandbox job.
 * @param jobId The job ID (task ID)
 * @param status The new status
 * @param userId ID of the user requesting the status change
 */
export async function changeSandboxTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return QueueService.get().changeTaskStatus(jobId, status, userId, "sandbox");
}

/**
 * Fetch statuses for multiple sandbox jobs.
 * @param taskIds Array of sandbox job IDs
 */
export async function getSandboxTaskStatuses(
    taskIds: string[],
): Promise<TaskStatusInfo[]> {
    const statuses = await QueueService.get().getTaskStatuses(taskIds, "sandbox");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}
