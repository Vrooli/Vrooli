import { HOURS_1_S, LlmTaskStatus, LlmTaskStatusInfo, MINUTES_1_MS, ServerLlmTaskInfo, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { SessionUserToken } from "../../types";
import { addJobToQueue } from "../queueHelper";

export type LlmTaskProcessPayload = {
    /** The chat the command was triggered in */
    chatId?: string | null;
    /** The language the command was triggered in */
    language: string;
    /** The status of the job process */
    status: LlmTaskStatus | `${LlmTaskStatus}`;
    /** The task to be run */
    taskInfo: ServerLlmTaskInfo;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let llmTaskProcess: (job: Bull.Job<LlmTaskProcessPayload>) => Promise<unknown>;
let llmTaskQueue: Bull.Queue<LlmTaskProcessPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export const setupLlmTaskQueue = async () => {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        llmTaskProcess = processModule.llmTaskProcess;

        // Initialize the Bull queue
        llmTaskQueue = new Bull<LlmTaskProcessPayload>("command", {
            redis: { port: PORT, host: HOST },
            defaultJobOptions: {
                removeOnComplete: {
                    age: HOURS_1_S,
                    count: 10_000,
                },
                removeOnFail: {
                    age: HOURS_1_S,
                    count: 10_000,
                },
            },
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
};

export const processLlmTask = async (data: Omit<LlmTaskProcessPayload, "status">): Promise<Success> => {
    return addJobToQueue(llmTaskQueue, { ...data, status: "Scheduled" }, { jobId: data.taskInfo.id, timeout: MINUTES_1_MS });
};

/**
 * Update a task's status
 * @param jobId The job ID (also the task ID) of the task
 * @param status The new status of the task
 * @param userId The user ID of the user who triggered the task. 
 * Only they are allowed to change the status of the task.
 */
export const changeLlmTaskStatus = async (
    jobId: string,
    status: `${LlmTaskStatus}`,
    userId: string,
): Promise<Success> => {
    try {
        const job = await llmTaskQueue.getJob(jobId);
        if (!job) {
            // If job isn't found but we're changing to a start or end state, consider it a success
            if (["Completed", "Failed", "Suggested"].includes(status)) {
                logger.info(`LLM task with jobId ${jobId} not found, but considered a success.`);
                return { __typename: "Success" as const, success: true };
            }
            // Otherwise, consider it an error 
            else {
                logger.error(`LLM task with jobId ${jobId} not found.`);
                return { __typename: "Success" as const, success: false };
            }
        }

        // Check if the user has permission to change the status
        if (job.data.userData.id !== userId) {
            logger.error(`User ${userId} is not authorized to change the status of job ${jobId}.`);
            return { __typename: "Success" as const, success: false };
        }

        // Update the job status
        await job.update({ ...job.data, status });
        return { __typename: "Success" as const, success: true };
    } catch (error) {
        logger.error(`Failed to change status for LLM task with jobId ${jobId}.`, { error });
        return { __typename: "Success" as const, success: false };
    }
};

/**
 * Get the statuses of multiple LLM tasks.
 * @param taskIds Array of task IDs for which to fetch the statuses.
 * @returns Promise that resolves to an array of objects with task ID and status.
 */
export const getLlmTaskStatuses = async (taskIds: string[]): Promise<LlmTaskStatusInfo[]> => {
    const taskStatusInfos = await Promise.all(taskIds.map(async (taskId) => {
        try {
            const job = await llmTaskQueue.getJob(taskId);
            if (job) {
                return { __typename: "LlmTaskStatusInfo" as const, id: taskId, status: job.data.status as LlmTaskStatus };
            } else {
                return { __typename: "LlmTaskStatusInfo" as const, id: taskId, status: null };  // Task not found, return null status
            }
        } catch (error) {
            console.error(`Failed to retrieve job ${taskId}:`, error);
            return { __typename: "LlmTaskStatusInfo" as const, id: taskId, status: null };  // Return null if there is an error fetching the job
        }
    }));

    return taskStatusInfos;
};
