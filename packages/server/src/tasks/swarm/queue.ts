// This is a comment to trigger a re-lint
/* eslint-disable no-magic-numbers */
import { type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
// Import QueueService type only to avoid circular dependency
type QueueServiceType = import("../queues.js").QueueService;
import { type SwarmTask, type SwarmExecutionTask } from "../taskTypes.js";

// Re-export the limits from the separate file to maintain backward compatibility
export { SWARM_QUEUE_LIMITS } from "./limits.js";

// Updated determinePriority function for swarm tasks
const BASE_PRIORITY = 100; // Default priority for swarm tasks
const PREMIUM_USER_PRIORITY_ADJUSTMENT = -20; // Premium users get higher priority (lower number)

/**
 * Compute dynamic priority for swarm tasks.
 * Priority is based on whether the user has a premium account.
 * 
 * NOTE: You must enqueue using `processSwarm` for priority to be applied.
 */
function determinePriority(payload: Omit<SwarmTask, "type" | "status">): number {
    let priority = BASE_PRIORITY;
    if (payload.userData?.hasPremium) {
        priority += PREMIUM_USER_PRIORITY_ADJUSTMENT;
    }
    return Math.max(0, priority); // Ensure priority doesn\'t go below 0
}

/**
 * Schedule or update a swarm task in the queue.
 * @param data Omitted fields status will be set internally.
 * @param queueService The QueueService instance to use
 */
export function processSwarm(
    data: Omit<SwarmTask, "status">,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.swarm.addTask(
        { ...data, status: "Scheduled" },
        { priority: determinePriority(data) },
    );
}

/**
 * Fetch statuses for multiple swarm jobs.
 * @param taskIds Array of swarm job IDs
 * @param queueService The QueueService instance to use
 */
export async function getSwarmTaskStatuses(
    taskIds: string[],
    queueService: QueueServiceType,
): Promise<TaskStatusInfo[]> {
    const statuses = await queueService.getTaskStatuses(taskIds, "swarm");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}

/**
 * Schedule a new three-tier swarm execution
 * @param data Omitted fields status will be set internally.
 * @param queueService The QueueService instance to use
 */
export function processNewSwarmExecution(
    data: Omit<SwarmExecutionTask, "status">,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.swarm.addTask(
        { ...data, status: "Scheduled" },
        { priority: determinePriority(data) },
    );
}

/**
 * Change the status of an existing swarm job.
 * @param taskId The task ID
 * @param status The new status
 * @param userId ID of the user requesting the status change
 * @param queueService The QueueService instance to use
 */
export async function changeSwarmTaskStatus(
    taskId: string,
    status: string,
    userId: string,
    queueService: QueueServiceType,
): Promise<Success> {
    return queueService.swarm.changeTaskStatus<SwarmTask>(taskId, status, userId);
}
