/* eslint-disable no-magic-numbers */
import { RunTriggeredFrom, type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
import { QueueTaskType, type RunTask } from "../taskTypes.js";

// Import QueueService type only to avoid circular dependency
type QueueServiceType = import("../queues.js").QueueService;

// Re-export the limits from the separate file to maintain backward compatibility
export { RUN_QUEUE_LIMITS } from "./limits.js";

// Priority constants for run tasks 
const BASE_PRIORITY = 100; // Default priority for run tasks
const RUN_VIEW_PRIORITY_ADJUSTMENT = -20; // Interactive runs get highest priority
const API_PRIORITY_ADJUSTMENT = -16;
const CHAT_PRIORITY_ADJUSTMENT = -14;
const WEBHOOK_PRIORITY_ADJUSTMENT = -10;
const BOT_PRIORITY_ADJUSTMENT = -7;
const SCHEDULE_PRIORITY_ADJUSTMENT = -3;
const TEST_PRIORITY_ADJUSTMENT = -1;
const TIME_SENSITIVE_PRIORITY_ADJUSTMENT = -15; // Time-sensitive tasks get priority boost
const PREMIUM_USER_PRIORITY_ADJUSTMENT = -5; // Premium users get priority boost
const RESUMED_RUN_PRIORITY_ADJUSTMENT = -10; // Resumed runs get priority boost

/**
 * Compute dynamic priority for run tasks.
 * Priority is based on multiple factors including:
 * - Where the run was triggered from (interactive vs automated)
 * - Whether the user has premium
 * - Whether the task is time-sensitive
 * - Whether it's a resumed run
 * 
 * NOTE: You must enqueue using `processRun` for priority to be applied.
 */
async function determinePriority(data: Omit<RunTask, "status">): Promise<number> {
    const { context, input } = data;
    let priority = BASE_PRIORITY;

    // Apply priority adjustments based on trigger source
    switch (input.runFrom) {
        case RunTriggeredFrom.RunView:
            priority += RUN_VIEW_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Api:
            priority += API_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Chat:
            priority += CHAT_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Webhook:
            priority += WEBHOOK_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Bot:
            priority += BOT_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Schedule:
            priority += SCHEDULE_PRIORITY_ADJUSTMENT;
            break;
        case RunTriggeredFrom.Test:
            priority += TEST_PRIORITY_ADJUSTMENT;
            break;
    }

    // Apply additional priority adjustments
    if (input.config?.isTimeSensitive) {
        priority += TIME_SENSITIVE_PRIORITY_ADJUSTMENT;
    }
    if (context.userData.hasPremium) {
        priority += PREMIUM_USER_PRIORITY_ADJUSTMENT;
    }
    if (!input.isNewRun) {
        // Lazy import to avoid circular dependency
        const { activeRunRegistry } = await import("./process.js");
        if (activeRunRegistry.get(input.runId)) {
            priority += RESUMED_RUN_PRIORITY_ADJUSTMENT;
        }
    }

    return Math.max(0, priority); // Ensure priority doesn't go below 0
}

/**
 * Schedule or update a run job in the queue.
 * @param data The complete RunExecutionInput structure (without status field)
 * @param queueService The QueueService instance to use
 */
export async function processRun(
    data: Omit<RunTask, "status">,
    queueService: QueueServiceType,
): Promise<Success> {
    const priority = await determinePriority(data);
    return queueService.run.addTask(
        { ...data, type: QueueTaskType.RUN_START, status: "Scheduled" as const },
        { priority },
    );
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

/**
 * Change the status of an existing run job.
 * @param jobId The job ID (task ID)
 * @param status The new status
 * @param userId ID of the user requesting the status change
 * @param queueService The QueueService instance to use
 */
export async function changeRunTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.changeTaskStatus(jobId, status, userId, "run");
}
