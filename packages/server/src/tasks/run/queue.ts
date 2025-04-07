/* eslint-disable no-magic-numbers */
import { MINUTES_1_MS, MINUTES_5_MS, RunConfig, RunProgress, RunStatus, RunTriggeredFrom, SessionUser, Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { BaseQueue } from "../base/queue.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, changeTaskStatus, getProcessPath, getTaskStatuses } from "../queueHelper.js";
import { RUN_SHUTDOWN_GRACE_PERIOD_MS, RUN_TIMEOUT_MS, activeRunsRegistry } from "./process.js";

/**
 * The number of runs that can be active at once.
 * 
 * NOTE: This is not the number of runs that can be in the queue at once.
 */
const MAX_ACTIVE_RUNS = 100;
/**
 * The numbers of runs in the queue for the system to be considered in a "high load" state.
 * 
 * When this number is exceeded, active runs that have been running for a while will be paused,
 * so that new runs have a chance to start.
 */
const HIGH_LOAD_THRESHOLD = Math.floor(MAX_ACTIVE_RUNS * 0.8);
/**
 * How often to check for long-running runs under high load.
 */
const HIGH_LOAD_CHECK_INTERVAL_MS = MINUTES_1_MS;
/**
 * How long in milliseconds an active run - by a FREE user - can run before it is consider "long running",
 * and has the potential to be paused to make room for new runs.
 */
const LONG_RUNNING_THRESHOLD_FREE_MS = MINUTES_1_MS;
/**
 * How long in milliseconds an active run - by a PREMIUM user - can run before it is consider "long running",
 * and has the potential to be paused to make room for new runs.
 */
const LONG_RUNNING_THRESHOLD_PREMIUM_MS = MINUTES_5_MS;
/**
 * How long the Bull queue will wait before giving up on a run and marking it as failed.
 * 
 * NOTE 1: This should be a long time, so that runs can execute for a long time in low-load conditions.
 * 
 * NOTE 2: On the run process level, we also have a timeout. This should be longer than that timeout. 
 * This is because this timeout kills the process immediately, while the run process timeout gives the 
 * process a chance to safely pause itself before being killed.
 */
const JOB_TIMEOUT_MS = RUN_TIMEOUT_MS + MINUTES_1_MS;

/**
 * Base object for requesting a run from the server
 */
export type RunRequestPayloadBase = Pick<RunProgress, "runId" | "type"> & {
    /**
     * The configuration for the run.
     * 
     * Should be provided if the run is new, but can be omitted if the run is being resumed.
     */
    config?: RunConfig;
    /**
     * Indicates if the run is new.
     */
    isNewRun: boolean;
    /**
     * What triggered the run. This is a factor in determining
     * the queue priority of the task.
     */
    runFrom: RunTriggeredFrom;
    /**
     * ID of the user who started the run. Either a human if it matches the 
     * session user, or a bot if it doesn't.
     */
    startedById: string;
    /** The latest status of the command. */
    status: TaskStatus | `${TaskStatus}`;
    /**
     * Unique ID to track task 
     * 
     * NOTE: This is the ID of the task job for managing the task queue, not the LlmTask task ID
     */
    taskId: string;
}

export type RunProjectPayload = RunRequestPayloadBase & {
    __process: "Project";
    projectVersionId: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUser;
};

export type RunRoutinePayload = RunRequestPayloadBase & {
    __process: "Routine";
    /** Inputs and outputs to use on current step, if not already in run object */
    formValues?: Record<string, unknown>;
    routineVersionId: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUser;
};

export type RunTestPayload = {
    __process: "Test";
};

export type RunRequestPayload = RunProjectPayload | RunRoutinePayload | RunTestPayload;

let logger: winston.Logger;
let runProcess: (job: Bull.Job<RunRequestPayload>) => Promise<unknown>;
export let runQueue: BaseQueue<RunRequestPayload>;
const FOLDER = "run";

// Call this on server startup
export async function setupRunQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).getRedisUrl();
        runProcess = (await import(getProcessPath(FOLDER))).runProcess;

        // Initialize the Bull queue
        runQueue = new BaseQueue<RunRequestPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        runQueue.getQueue().process(MAX_ACTIVE_RUNS, runProcess);
        // Verify that the queue is working
        addJobToQueue(runQueue.getQueue(), { __process: "Test" }, { timeout: JOB_TIMEOUT_MS });
        // Set up a periodic check to gracefully pause long-running runs under periods of high load.
        setInterval(checkLongRunningRuns, HIGH_LOAD_CHECK_INTERVAL_MS);
    } catch (error) {
        const errorMessage = "Failed to setup run queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0567", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

/**
 * Periodically check the active runs. If the system is under high load,
 * then for runs that have been active longer than their threshold (free vs premium),
 * call the run state machine’s pause method.
 */
function checkLongRunningRuns() {
    // Don't pause runs if the system is not under high load
    if (activeRunsRegistry.count() < HIGH_LOAD_THRESHOLD) {
        return;
    }

    // Compare runs against the current time
    const now = Date.now();
    // Get the runs in order of when they were added to the registry
    const orderedRuns = activeRunsRegistry.getOrderedRuns();
    // activeRunsRegistry.getOrderedRuns().forEach((record, index) => {
    for (const record of orderedRuns) {
        const timeElapsedMs = now - record.startTime;
        // If the time elapsed is shorter than the lowest possible threshold, 
        // then we know that every subsequent run will also be below its threshold. 
        // This means we can stop checking.
        if (timeElapsedMs < LONG_RUNNING_THRESHOLD_FREE_MS) {
            return;
        }
        const thresholdMs = record.hasPremium ? LONG_RUNNING_THRESHOLD_PREMIUM_MS : LONG_RUNNING_THRESHOLD_FREE_MS;
        if (timeElapsedMs > thresholdMs) {
            logger.warning(`Pausing long-running run ${record.runId} after ${timeElapsedMs}ms under high load.`, { trace: "0568" });
            const run = activeRunsRegistry.get(record.runId);
            if (run) {
                // This is idempotent, so it's safe to call multiple times
                run.stopRun(RunStatus.Paused);
            }
            // After a grace period, remove it from the registry if it’s still active.
            setTimeout(() => {
                const run = activeRunsRegistry.get(record.runId);
                if (run) {
                    activeRunsRegistry.remove(record.runId);
                    logger.warning(`Run ${record.runId} did not pause gracefully within the grace period and has been removed.`, { trace: "0570" });
                }
            }, RUN_SHUTDOWN_GRACE_PERIOD_MS);
        }
    }
}

/**
 * Determines the queue priority for a run based on the run's properties. 
 * Lower numbers are higher priority.
 * 
 * @param payload The payload for the run.
 * @returns The priority for the run.
 */
function determinePriority(
    payload: Omit<RunProjectPayload, "__process" | "status"> | Omit<RunRoutinePayload, "__process" | "status">,
): number {
    // Start with a base priority that is high enough to not go below 0 
    // even with all the adjustments, and leaves enough of a buffer for 
    // future adjustments
    let priority = 100;

    // Adjust base priority based on where the run is triggered from
    switch (payload.runFrom) {
        case RunTriggeredFrom.RunView:
            priority -= 20;
            break;
        case RunTriggeredFrom.Api:
            priority -= 16;
            break;
        case RunTriggeredFrom.Chat:
            priority -= 14;
            break;
        case RunTriggeredFrom.Webhook:
            priority -= 10;
            break;
        case RunTriggeredFrom.Bot:
            priority -= 7;
            break;
        case RunTriggeredFrom.Schedule:
            priority -= 3;
            break;
        case RunTriggeredFrom.Test:
            priority -= 1;
            break;
    }

    // Increase priority if the task is time-sensitive
    // TODO don't think we charge the user for this yet, but we should
    if (payload.config?.isTimeSensitive) {
        // This puts scheduled runs from non-premium users above everything except RunView runs, 
        // and scheduled runs from premium users above everything
        priority -= 15;
    }

    // Increase priority if the user is a premium user
    if (payload.userData.hasPremium) {
        // This puts scheduled runs from premium users above bot runs
        priority -= 5;
    }

    // Increase priority for existing runs that are already in the active run registry
    if (!payload.isNewRun && activeRunsRegistry.get(payload.runId)) {
        priority -= 10;
    }

    // Ensure priority is non-negative
    return Math.max(0, priority);
}

export function processRunProject(
    data: Omit<RunProjectPayload, "__process" | "status">,
): Promise<Success> {
    return addJobToQueue(
        runQueue.getQueue(),
        { ...data, __process: "Project", status: "Scheduled" },
        { jobId: data.taskId, timeout: JOB_TIMEOUT_MS, priority: determinePriority(data) },
    );
}

export function processRunRoutine(
    data: Omit<RunRoutinePayload, "__process" | "status">,
): Promise<Success> {
    return addJobToQueue(
        runQueue.getQueue(),
        { ...data, __process: "Routine", status: "Scheduled" },
        { jobId: data.taskId, timeout: JOB_TIMEOUT_MS, priority: determinePriority(data) },
    );
}


/**
 * Update a task's status
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export async function changeRunTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return changeTaskStatus(runQueue.getQueue(), jobId, status, userId);
}

/**
 * Get the statuses of multiple run tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getRunTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    return getTaskStatuses(runQueue.getQueue(), taskIds);
}
