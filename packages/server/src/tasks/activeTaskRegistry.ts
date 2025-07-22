import { SECONDS_1_MS } from "@vrooli/shared";
import { logger } from "../events/logger.js";

/**
 * Interface for a managed task state machine.
 * 
 * Supports pausing, stopping, and resuming tasks.
 */
export interface ManagedTaskStateMachine {
    /** Returns the unique ID of the task it manages */
    getTaskId(): string;
    /** Returns the current status of the task */
    getState(): string;
    /** Returns true if pause was initiated successfully */
    requestPause(): Promise<boolean>;
    /** Returns true if stop was initiated successfully */
    requestStop(reason: string): Promise<boolean>;
    /** Optional: for user-specific actions/notifications */
    getAssociatedUserId?(): string | undefined;
}

/**
 * Limits and thresholds for prioritized queues.
 */
export type ActiveTaskRegistryLimits = {
    /** The maximum number of active jobs in the queue */
    maxActive: number;
    /** How often to check for high load conditions (milliseconds) */
    highLoadCheckIntervalMs: number;
    /** The threshold for high load (percentage of maxActive, from 0 to 1) */
    highLoadThresholdPercentage: number;
    /** How long until a task is considered long running (free tier) */
    longRunningThresholdFreeMs: number;
    /** How long until a task is considered long running (premium tier) */
    longRunningThresholdPremiumMs: number;
    /**
     * How long to wait before giving up on a run and marking it as failed.
     * 
     * NOTE 1: This should be a long time, so that runs can execute for a long time in low-load conditions.
     */
    taskTimeoutMs: number;
    /**
     * How long to wait after a run has been paused to give it a chance to finish gracefully.
     */
    shutdownGracePeriodMs: number;
    /** Action to take when a task first exceeds the long-running threshold */
    onLongRunningFirstThreshold: "pause" | "stop";
    /** Number of times to retry pause before giving up (0 means no retries) */
    longRunningPauseRetries?: number;
    /** Number of times to retry stop before giving up (0 means no retries) */
    longRunningStopRetries?: number;
    // In the future, we could add a field for subsequent actions if the task continues to be long-running after the first action
}

/**
 * Base representation of an active task record from any registry.
 */
export interface BaseActiveTaskRecord {
    /** Whether the user has premium status. Premium users have a higher priority for long-running tasks. */
    hasPremium: boolean;
    /** The time the task was started. */
    startTime: number;
    /** The unique ID of the task. */
    id: string;
    /** The ID of the user who started the task. */
    userId: string;
}

/**
 * Base abstract class for an active task registry.
 * 
 * This is useful for long-running tasks that require prioritization, pausing/cancelling/resuming, live configuration updates, etc.
 * 
 * Specific registries like ActiveRunsRegistry and ActiveSwarmRegistry should extend this class.
 */
export abstract class BaseActiveTaskRegistry<
    Record extends BaseActiveTaskRecord,
    SM extends ManagedTaskStateMachine
> {
    /** Internal map for fast lookup by runId. */
    private recordsMap = new Map<string, SM>();
    /** Ordered list (oldest first) of task records. */
    private recordsList: Record[] = [];

    /**
     * Adds a new task record to the registry.
     *
     * @param record - The task record to add.
     * @param stateMachine - The state machine to use to manage the task.
     * @throws An error if the task record already exists.
     */
    public add(record: Record, stateMachine: SM): void {
        if (this.recordsMap.has(record.id)) {
            throw new Error(`Task with id ${record.id} already exists in the registry.`);
        }
        this.recordsMap.set(record.id, stateMachine);
        this.recordsList.push(record);
    }

    /**
     * Removes a task record from the registry.
     *
     * @param id - The unique ID of the task.
     * @returns true if the task was removed, false if it wasn't found.
    */
    public remove(id: string): boolean {
        if (!this.recordsMap.has(id)) {
            return false;
        }
        this.recordsMap.delete(id);
        // Remove the record from the list.
        const index = this.recordsList.findIndex(record => record.id === id);
        if (index !== -1) {
            this.recordsList.splice(index, 1);
        }
        return true;
    }

    /**
     * Retrieves the StateMachine for a given task ID
     *
     * @param id - The unique ID of the task.
     * @returns The StateMachine instance, or undefined if not found.
     */
    public get(id: string): SM | undefined {
        return this.recordsMap.get(id);
    }

    /**
     * Returns the list of active task records in order (oldest first).
     * 
     * NOTE: Because we push new records to the end, recordsList is already in insertion order.
     */
    public getOrderedRecords(): Record[] {
        return this.recordsList;
    }

    /**
     * Returns the number of active tasks in the registry.
     */
    public count(): number {
        return this.recordsMap.size;
    }

    /**
     * Clears all tasks from the registry.
     */
    public clear(): void {
        this.recordsMap.clear();
        this.recordsList = [];
    }
}

/**
 * Helper function to attempt an action (pause or stop) with retries.
 * @param actionFn - The function to call (sm.requestPause or sm.requestStop).
 * @param retries - The number of retries allowed.
 * @param taskTypeName - Name of the task type for logging.
 * @param taskId - ID of the task for logging.
 * @param actionName - Name of the action for logging ("pause" or "stop").
 */
async function attemptActionWithRetries(
    actionFn: () => Promise<boolean>,
    retries: number | undefined,
    taskTypeName: string,
    taskId: string,
    actionName: "pause" | "stop",
): Promise<void> {
    const maxAttempts = (retries ?? 0) + 1;
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
        try {
            const success = await actionFn();
            if (success) {
                return;
            } else {
                logger.warn(`${actionName.charAt(0).toUpperCase() + actionName.slice(1)} request for ${taskId} failed or was not applicable (attempt ${attempt}/${maxAttempts}).`);
            }
        } catch (err) {
            logger.error(`Error requesting ${actionName} for ${taskTypeName} task ${taskId} (attempt ${attempt}/${maxAttempts})`, { error: err });
        }
        if (attempt < maxAttempts) {
            // Optional: add a small delay between retries if needed
            // await new Promise(resolve => setTimeout(resolve, 1000)); // e.g., 1 second delay
        }
    }
    logger.error(`Failed to ${actionName} ${taskTypeName} task ${taskId} after ${maxAttempts} attempts.`);
}

/**
 * Generic function to check for long-running tasks in a given registry.
 * @param registry - The active task registry (e.g., activeRunsRegistry, activeSwarmRegistry).
 * @param limits - The ActiveTaskRegistryLimits for this type of task.
 * @param taskTypeName - A string name for the task type, for logging (e.g., "Run", "Swarm").
 * @param getOrderedRecordsMethod - The specific method on the registry to get ordered records.
 */
export async function checkLongRunningTasksInRegistry(
    registry: BaseActiveTaskRegistry<BaseActiveTaskRecord, ManagedTaskStateMachine>,
    limits: ActiveTaskRegistryLimits,
    taskTypeName: string,
): Promise<number> {
    const now = Date.now();
    if (registry.count() === 0) {
        return 0; // Return count of problematic tasks
    }

    const tasksToProcess = registry.getOrderedRecords();
    let longRunningTaskCount = 0;

    if (!tasksToProcess || !Array.isArray(tasksToProcess)) {
        logger.warn(`[checkLongRunningTasksInRegistry] getOrderedRecords for ${taskTypeName} returned an unexpected value. Skipping check.`);
        return 0;
    }

    for (const taskInfo of tasksToProcess) {
        if (!taskInfo || typeof taskInfo.startTime !== "number" || typeof taskInfo.id !== "string") {
            logger.warn(`[checkLongRunningTasksInRegistry] Encountered invalid ${taskTypeName} taskInfo object.`, { taskInfo });
            continue;
        }
        if (typeof taskInfo.userId !== "string" || taskInfo.userId.trim() === "") {
            logger.warn(`[checkLongRunningTasksInRegistry] ${taskTypeName} task ${taskInfo.id} is missing a valid userId. Skipping notifications for this task.`, { taskInfo });
            // Continue processing for pause/stop logic even if notifications can't be sent.
        }

        const threshold = taskInfo.hasPremium ? limits.longRunningThresholdPremiumMs : limits.longRunningThresholdFreeMs;
        const durationMs = now - taskInfo.startTime;

        if (durationMs > threshold) {
            longRunningTaskCount++;
            const warningMessage = `${taskTypeName} task ${taskInfo.id} (user: ${taskInfo.userId}) has been active for ${Math.floor(durationMs / SECONDS_1_MS)}s (threshold: ${threshold / SECONDS_1_MS}s).`;
            logger.warn(
                warningMessage,
                {
                    taskId: taskInfo.id,
                    taskType: taskTypeName,
                    startTime: new Date(taskInfo.startTime).toISOString(),
                    durationMs,
                    thresholdMs: threshold,
                    hasPremium: taskInfo.hasPremium,
                    userId: taskInfo.userId,
                },
            );

            // Action 1: Send notification to user
            if (taskInfo.userId) {
                try {
                    // Use lazy loading to avoid circular dependency with notify/queues
                    const { Notify } = await import("../notify/notify.js");

                    Notify(undefined) // Pass undefined for languages, or retrieve user languages if possible
                        .pushLongRunningTaskWarning(
                            taskInfo.id,
                            taskTypeName,
                            durationMs,
                            threshold,
                        )
                        .toUser(taskInfo.userId)
                        .catch(error => {
                            logger.error(`Failed to send long-running task warning notification for ${taskTypeName} task ${taskInfo.id}`, { error, taskId: taskInfo.id, userId: taskInfo.userId });
                        });
                    logger.info(`Sent/Queued long-running task warning notification for ${taskTypeName} task ${taskInfo.id} for user ${taskInfo.userId} via Notify service.`);
                } catch (error) {
                    logger.error(`Error initiating long-running task warning notification for ${taskTypeName} task ${taskInfo.id}`, { error, taskId: taskInfo.id, userId: taskInfo.userId });
                }
            } else {
                logger.info(`Cannot send notification for long-running ${taskTypeName} task ${taskInfo.id} because userId is missing.`);
            }

            const sm = registry.get(taskInfo.id);
            if (sm) {
                const status = sm.getState();
                const pausableStoppableStates = ["RUNNING", "IDLE", "STARTING"]; // These are examples, adjust based on actual state machine logic

                if (pausableStoppableStates.includes(status)) {
                    const action = limits.onLongRunningFirstThreshold;
                    const stopReason = `Exceeded long-running threshold of ${Math.floor(threshold / SECONDS_1_MS)}s`;

                    if (action === "pause") {
                        await attemptActionWithRetries(
                            () => sm.requestPause(),
                            limits.longRunningPauseRetries,
                            taskTypeName,
                            taskInfo.id,
                            "pause",
                        );
                    } else {
                        await attemptActionWithRetries(
                            () => sm.requestStop(stopReason),
                            limits.longRunningStopRetries,
                            taskTypeName,
                            taskInfo.id,
                            "stop",
                        );
                    }
                } else {
                    logger.info(`Long-running ${taskTypeName} task ${taskInfo.id} is in status '${status}', not attempting pause/stop.`);
                }
            } else {
                logger.warn(`Could not retrieve state machine for long-running ${taskTypeName} task ${taskInfo.id} to attempt actions.`);
            }

            // Action 3: Forceful Termination Warning (BullMQ handles actual timeout)
            const TIMEOUT_WARNING_THRESHOLD_PERCENTAGE = 0.8;
            if (limits.taskTimeoutMs > 0 && durationMs > limits.taskTimeoutMs * TIMEOUT_WARNING_THRESHOLD_PERCENTAGE) {
                logger.warn(
                    `${taskTypeName} task ${taskInfo.id} is approaching its hard timeout of ${limits.taskTimeoutMs / SECONDS_1_MS}s. Current duration: ${Math.floor(durationMs / SECONDS_1_MS)}s.`,
                );
            }
        }
    }
    // In the future, further refinements can be made to the helper function, e.g. exponential backoff for retries.

    return longRunningTaskCount;
}
