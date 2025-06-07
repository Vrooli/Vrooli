/* eslint-disable no-magic-numbers */
import { type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
import type { ActiveTaskRegistryLimits } from "../activeTaskRegistry.js";
import { QueueService } from "../queues.js";
import { type SwarmTask } from "../taskTypes.js";

export const SWARM_QUEUE_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: parseInt(process.env.WORKER_SWARM_MAX_ACTIVE || "10"),
    highLoadCheckIntervalMs: parseInt(process.env.WORKER_SWARM_HIGH_LOAD_CHECK_INTERVAL_MS || "5000"),
    highLoadThresholdPercentage: parseFloat(process.env.WORKER_SWARM_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0.8"),
    longRunningThresholdFreeMs: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_FREE_MS || "60000"),
    longRunningThresholdPremiumMs: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_THRESHOLD_PREMIUM_MS || "300000"),
    taskTimeoutMs: parseInt(process.env.WORKER_SWARM_TASK_TIMEOUT_MS || "600000"),
    shutdownGracePeriodMs: parseInt(process.env.WORKER_SWARM_SHUTDOWN_GRACE_PERIOD_MS || "30000"),
    onLongRunningFirstThreshold: (process.env.WORKER_SWARM_ON_LONG_RUNNING_FIRST_THRESHOLD || "pause") as "pause" | "stop",
    longRunningPauseRetries: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_PAUSE_RETRIES || "1"),
    longRunningStopRetries: parseInt(process.env.WORKER_SWARM_LONG_RUNNING_STOP_RETRIES || "0"),
};

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
 */
export function processSwarm(
    data: Omit<SwarmTask, "status">,
): Promise<Success> {
    return QueueService.get().swarm.addTask(
        { ...data, status: "Scheduled" },
        { priority: determinePriority(data) },
    );
}

/**
 * Fetch statuses for multiple swarm jobs.
 */
export async function getSwarmTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    const statuses = await QueueService.get().getTaskStatuses(taskIds, "swarm");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}

/**
 * Change the status of an existing swarm job.
 */
export async function changeSwarmTaskStatus(
    taskId: string,
    status: string,
    userId: string,
): Promise<Success> {
    return QueueService.get().swarm.changeTaskStatus<SwarmTask>(taskId, status, userId);
}
