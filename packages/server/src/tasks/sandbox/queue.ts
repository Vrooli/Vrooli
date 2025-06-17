import { type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
import { QueueTaskType, type SandboxTask } from "../taskTypes.js";

// Import QueueService type only to avoid circular dependency
type QueueServiceType = import("../queues.js").QueueService;

/**
 * Schedule a sandbox execution job.
 * @param data Omitted fields __process and status will be set internally.
 * @param queueService The QueueService instance to use
 */
export function processSandbox(
    data: Omit<SandboxTask, "type" | "status">,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.sandbox.addTask(
        { ...data, type: QueueTaskType.SANDBOX_EXECUTION, status: "Scheduled" },
    );
}

/**
 * Change the status of an existing sandbox job.
 * @param jobId The job ID (task ID)
 * @param status The new status
 * @param userId ID of the user requesting the status change
 * @param queueService The QueueService instance to use
 */
export async function changeSandboxTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.changeTaskStatus(jobId, status, userId, "sandbox");
}

/**
 * Fetch statuses for multiple sandbox jobs.
 * @param taskIds Array of sandbox job IDs
 * @param queueService The QueueService instance to use
 */
export async function getSandboxTaskStatuses(
    taskIds: string[],
    queueService: QueueServiceType,
): Promise<TaskStatusInfo[]> {
    const statuses = await queueService.getTaskStatuses(taskIds, "sandbox");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}
