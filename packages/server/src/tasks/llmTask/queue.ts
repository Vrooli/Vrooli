import { MINUTES_1_MS, ServerLlmTaskInfo, Success, TaskStatus, TaskStatusInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, changeTaskStatus, getProcessPath, getTaskStatuses } from "../queueHelper";

export type LlmTaskProcessPayload = {
    __process: "LlmTask";
    /** The chat the command was triggered in */
    chatId?: string | null;
    /** The language the command was triggered in */
    language: string;
    /** The status of the job process */
    status: TaskStatus | `${TaskStatus}`;
    /** The task to be run */
    taskInfo: ServerLlmTaskInfo;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

let logger: winston.Logger;
let llmTaskProcess: (job: Bull.Job<LlmTaskProcessPayload>) => Promise<unknown>;
let llmTaskQueue: Bull.Queue<LlmTaskProcessPayload>;
const FOLDER = "llmTask";

// Call this on server startup
export async function setupLlmTaskQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).REDIS_URL;
        llmTaskProcess = (await import(getProcessPath(FOLDER))).llmTaskProcess;

        // Initialize the Bull queue
        llmTaskQueue = new Bull<LlmTaskProcessPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        llmTaskQueue.process(llmTaskProcess);
    } catch (error) {
        const errorMessage = "Failed to setup command queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0220", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export async function processLlmTask(data: Omit<LlmTaskProcessPayload, "status">): Promise<Success> {
    return addJobToQueue(
        llmTaskQueue,
        { ...data, status: "Scheduled" },
        { jobId: data.taskInfo.taskId, timeout: MINUTES_1_MS },
    );
}

/**
 * Update a task's status
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export async function changeLlmTaskStatus(
    jobId: string,
    status: TaskStatus | `${TaskStatus}`,
    userId: string,
): Promise<Success> {
    return changeTaskStatus(llmTaskQueue, jobId, status, userId);
}

/**
 * Get the statuses of multiple LLM tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export async function getLlmTaskStatuses(taskIds: string[]): Promise<TaskStatusInfo[]> {
    return getTaskStatuses(llmTaskQueue, taskIds);
}
