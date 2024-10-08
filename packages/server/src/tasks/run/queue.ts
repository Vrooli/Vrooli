/* eslint-disable no-magic-numbers */
import { MINUTES_1_MS, RunFrom, RunRequestPayloadBase, Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, changeTaskStatus, getProcessPath, getTaskStatuses } from "../queueHelper";

export type RunProjectPayload = RunRequestPayloadBase & {
    __process: "Project";
    projectVersionId: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
};

export type RunRoutinePayload = RunRequestPayloadBase & {
    __process: "Routine";
    /** Inputs and outputs to use on current step, if not already in run object */
    formValues?: Record<string, unknown>;
    routineVersionId: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
};

export type RunTestPayload = {
    __process: "Test";
};

export type RunRequestPayload = RunProjectPayload | RunRoutinePayload | RunTestPayload;

let logger: winston.Logger;
let runProcess: (job: Bull.Job<RunRequestPayload>) => Promise<unknown>;
let runQueue: Bull.Queue<RunRequestPayload>;
const FOLDER = "run";

// Call this on server startup
export async function setupRunQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).REDIS_URL;
        runProcess = (await import(getProcessPath(FOLDER))).runProcess;

        // Initialize the Bull queue
        runQueue = new Bull<RunRequestPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        runQueue.process(runProcess);
        // Verify that the queue is working
        addJobToQueue(runQueue, { __process: "Test" }, { timeout: MINUTES_1_MS });
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
 * Determines the queue priority for a run based on the run's properties. 
 * Lower numbers are higher priority.
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
        case RunFrom.RunView:
            priority -= 20;
            break;
        case RunFrom.Api:
            priority -= 16;
            break;
        case RunFrom.Chat:
            priority -= 14;
            break;
        case RunFrom.Webhook:
            priority -= 10;
            break;
        case RunFrom.Bot:
            priority -= 7;
            break;
        case RunFrom.Schedule:
            priority -= 3;
            break;
        case RunFrom.Test:
            priority -= 1;
            break;
    }

    // Increase priority if the task is time-sensitive
    if (payload.isTimeSensitive) {
        // This puts scheduled runs from non-premium users above everything except RunView runs, 
        // and scheduled runs from premium users above everything
        priority -= 15;
    }

    // Increase priority if the user is a premium user
    if (payload.userData.hasPremium) {
        // This puts scheduled runs from premium users above bot runs
        priority -= 5;
    }

    // Ensure priority is non-negative
    return Math.max(0, priority);
}

export function processRunProject(
    data: Omit<RunProjectPayload, "__process" | "status">,
): Promise<Success> {
    return addJobToQueue(
        runQueue,
        { ...data, __process: "Project", status: "Scheduled" },
        { jobId: data.taskId, timeout: MINUTES_1_MS, priority: determinePriority(data) },
    );
}

export function processRunRoutine(
    data: Omit<RunRoutinePayload, "__process" | "status">,
): Promise<Success> {
    return addJobToQueue(
        runQueue,
        { ...data, __process: "Routine", status: "Scheduled" },
        { jobId: data.taskId, timeout: MINUTES_1_MS, priority: determinePriority(data) },
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
    return changeTaskStatus(runQueue, jobId, status, userId);
}

/**
 * Get the statuses of multiple run tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getRunTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    return getTaskStatuses(runQueue, taskIds);
}
