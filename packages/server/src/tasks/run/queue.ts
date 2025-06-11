/* eslint-disable no-magic-numbers */
import { HOURS_2_MS, MINUTES_1_MS, MINUTES_5_MS, RunTriggeredFrom, SECONDS_10_MS, type Success, type TaskStatus, type TaskStatusInfo } from "@vrooli/shared";
import type { ActiveTaskRegistryLimits } from "../activeTaskRegistry.js";
import { QueueService } from "../queues.js";
import { QueueTaskType, type RunTask } from "../taskTypes.js";
import { activeRunsRegistry } from "./process.js";

export const RUN_QUEUE_LIMITS: ActiveTaskRegistryLimits = {
    maxActive: parseInt(process.env.WORKER_RUN_MAX_ACTIVE || "1000"),
    highLoadCheckIntervalMs: parseInt(process.env.WORKER_RUN_HIGH_LOAD_CHECK_INTERVAL_MS || MINUTES_1_MS.toString()),
    highLoadThresholdPercentage: parseFloat(process.env.WORKER_RUN_HIGH_LOAD_THRESHOLD_PERCENTAGE || "0.8"),
    longRunningThresholdFreeMs: parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_FREE_MS || MINUTES_1_MS.toString()),
    longRunningThresholdPremiumMs: parseInt(process.env.WORKER_RUN_LONG_RUNNING_THRESHOLD_PREMIUM_MS || MINUTES_5_MS.toString()),
    taskTimeoutMs: parseInt(process.env.WORKER_RUN_TASK_TIMEOUT_MS || HOURS_2_MS.toString()),
    shutdownGracePeriodMs: parseInt(process.env.WORKER_RUN_SHUTDOWN_GRACE_PERIOD_MS || SECONDS_10_MS.toString()),
    onLongRunningFirstThreshold: (process.env.WORKER_RUN_ON_LONG_RUNNING_FIRST_THRESHOLD || "pause") as "pause" | "stop",
    longRunningPauseRetries: parseInt(process.env.WORKER_RUN_LONG_RUNNING_PAUSE_RETRIES || "1"),
    longRunningStopRetries: parseInt(process.env.WORKER_RUN_LONG_RUNNING_STOP_RETRIES || "0"),
};

/**
 * Compute dynamic priority for run tasks.
 * 
 * NOTE: You must enqueue using `processRun` for priority to be applied.
 */
function determinePriority(payload: Omit<RunTask, "type" | "status">): number {
    let priority = 100;
    switch (payload.runFrom) {
        case RunTriggeredFrom.RunView: priority -= 20; break;
        case RunTriggeredFrom.Api: priority -= 16; break;
        case RunTriggeredFrom.Chat: priority -= 14; break;
        case RunTriggeredFrom.Webhook: priority -= 10; break;
        case RunTriggeredFrom.Bot: priority -= 7; break;
        case RunTriggeredFrom.Schedule: priority -= 3; break;
        case RunTriggeredFrom.Test: priority -= 1; break;
    }
    if (payload.config?.isTimeSensitive) priority -= 15;
    if (payload.userData.hasPremium) priority -= 5;
    if (!payload.isNewRun && activeRunsRegistry.get(payload.runId)) priority -= 10;
    return Math.max(0, priority);
}

/**
 * Schedule or update a run job in the queue.
 */
export function processRun(
    data: Omit<RunTask, "type" | "status">,
): Promise<Success> {
    return QueueService.get().run.addTask(
        { ...data, type: QueueTaskType.RUN_START, status: "Scheduled" },
        { priority: determinePriority(data) },
    );
}

/**
 * Change the status of an existing run job.
 */
export function changeRunTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return QueueService.get().changeTaskStatus(jobId, status, userId, "run");
}

/**
 * Fetch statuses for multiple run jobs.
 */
export async function getRunTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    const statuses = await QueueService.get().getTaskStatuses(taskIds, "run");
    return statuses.map(s => ({
        __typename: "TaskStatusInfo" as const,
        id: s.id,
        status: s.status as TaskStatus | null,
        queueName: s.queueName,
    }));
}
