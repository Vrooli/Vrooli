/* eslint-disable no-magic-numbers */
import { RunTriggeredFrom, type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
// Import QueueService type only to avoid circular dependency
type QueueServiceType = import("../queues.js").QueueService;
import { QueueTaskType, type RunTask } from "../taskTypes.js";

// Re-export the limits from the separate file to maintain backward compatibility
export { RUN_QUEUE_LIMITS } from "./limits.js";

/**
 * Schedule or update a run job in the queue.
 * @param data Omitted fields type and status will be set internally.
 * @param queueService The QueueService instance to use
 */
export async function processRun(
    data: Omit<RunTask, "type" | "status">,
    queueService: QueueServiceType,
): Promise<Success> {
    // Compute dynamic priority for run tasks inline to support async operations
    let priority = 100;
    switch (data.runFrom) {
        case RunTriggeredFrom.RunView: priority -= 20; break;
        case RunTriggeredFrom.Api: priority -= 16; break;
        case RunTriggeredFrom.Chat: priority -= 14; break;
        case RunTriggeredFrom.Webhook: priority -= 10; break;
        case RunTriggeredFrom.Bot: priority -= 7; break;
        case RunTriggeredFrom.Schedule: priority -= 3; break;
        case RunTriggeredFrom.Test: priority -= 1; break;
    }
    if (data.config?.isTimeSensitive) priority -= 15;
    if (data.userData.hasPremium) priority -= 5;
    if (!data.isNewRun) {
        // Lazy import to avoid circular dependency
        const { activeRunRegistry } = await import("./process.js");
        if (activeRunRegistry.get(data.runId)) priority -= 10;
    }
    priority = Math.max(0, priority);

    return queueService.run.addTask(
        { ...data, type: QueueTaskType.RUN_START, status: "Scheduled" },
        { priority },
    );
}

/**
 * Change the status of an existing run job.
 * @param jobId The job ID (task ID)
 * @param status The new status
 * @param userId ID of the user requesting the status change
 * @param queueService The QueueService instance to use
 */
export function changeRunTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.changeTaskStatus(jobId, status, userId, "run");
}

/**
 * Fetch statuses for multiple run jobs.
 * @param taskIds Array of run job IDs
 * @param queueService The QueueService instance to use
 */
export async function getRunTaskStatuses(
    taskIds: string[],
    queueService: QueueServiceType,
): Promise<TaskStatusInfo[]> {
    const statuses = await queueService.getTaskStatuses(taskIds, "run");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}
