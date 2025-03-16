import { Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, changeTaskStatus, getProcessPath, getTaskStatuses } from "../queueHelper.js";
import { SandboxProcessPayload } from "./types.js";

export type SandboxTestPayload = {
    __process: "Test";
};

export type SandboxRequestPayload = SandboxProcessPayload | SandboxTestPayload;

let logger: winston.Logger;
let sandboxProcess: (job: Bull.Job<SandboxProcessPayload>) => Promise<unknown>;
let sandboxQueue: Bull.Queue<SandboxProcessPayload>;
const FOLDER = "sandbox";

// Call this on server startup
export async function setupSandboxQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).getRedisUrl();
        sandboxProcess = (await import(getProcessPath(FOLDER))).sandboxProcess;

        // Initialize the Bull queue
        sandboxQueue = new Bull<SandboxProcessPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        sandboxQueue.process(sandboxProcess);
    } catch (error) {
        const errorMessage = "Failed to setup sandbox queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0553", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export function runSandboxedCode(data: SandboxProcessPayload): Promise<Success> {
    return addJobToQueue(sandboxQueue, data, {});
}

/**
 * Update a task's status
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export async function changeSandboxTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return changeTaskStatus(sandboxQueue, jobId, status, userId);
}

/**
 * Get the statuses of multiple sandbox tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getSandboxTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    return getTaskStatuses(sandboxQueue, taskIds);
}
