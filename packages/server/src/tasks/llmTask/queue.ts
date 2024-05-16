import { SessionUserToken } from "@local/server";
import { HOURS_1_S, MINUTES_1_MS, ServerLlmTaskInfo } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";

type LlmTaskStatus = "scheduled" | "running" | "canceled" | "completed" | "failed";
export type LlmTaskProcessPayload = {
    /** The task to be run */
    taskInfo: ServerLlmTaskInfo;
    /** The chat the command was triggered in */
    chatId?: string | null;
    /** The language the command was triggered in */
    language: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
    /** The status of the job process */
    status: LlmTaskStatus;
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

export const processLlmTask = (data: Omit<LlmTaskProcessPayload, "status">) => {
    llmTaskQueue.add(
        { ...data, status: "scheduled" },
        { jobId: data.taskInfo.id, timeout: MINUTES_1_MS },
    );
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
    status: "scheduled" | "canceled" | "running",
    userId: string,
): Promise<{ success: boolean, message: string }> => {
    try {
        const job = await llmTaskQueue.getJob(jobId);
        if (!job) {
            // Job not found in queue, handle based on desired status
            if (status === "canceled") {
                logger.info(`LLM task with jobId ${jobId} not found, but considered canceled.`);
                return { success: true, message: "Task not found but considered canceled." };
            } else {
                throw new Error(`LLM task with jobId ${jobId} not found.`);
            }
        }

        // Check if the user has permission to change the status
        if (job.data.userData.id !== userId) {
            throw new Error(`User ${userId} is not authorized to change the status of job ${jobId}.`);
        }

        if (status === "scheduled") {
            const currentState = await job.getState();
            if (currentState === "failed" || currentState === "waiting") {
                // Add the job back to the queue if it failed or is waiting
                await job.update({ ...job.data, status: "scheduled" });
                await llmTaskQueue.add(job.data);
                logger.info(`LLM task with jobId ${jobId} rescheduled.`);
                return { success: true, message: "Task rescheduled." };
            } else {
                throw new Error(`LLM task with jobId ${jobId} cannot be rescheduled from state ${currentState}.`);
            }
        } else if (status === "canceled") {
            await job.update({ ...job.data, status: "canceled" });
            await job.remove();
            logger.info(`LLM task with jobId ${jobId} canceled.`);
            return { success: true, message: "Task canceled." };
        }
    } catch (error) {
        logger.error(`Failed to change status for LLM task with jobId ${jobId}.`, { error });
        return { success: false, message: (error as { message?: string }).message || "Failed to change status." };
    }
    logger.error(`Failed to change status for LLM task with jobId ${jobId}.`);
    return { success: false, message: "Failed to change status." };
};

export const getLlmTaskStatus = async (jobId: string) => {
    try {
        const job = await llmTaskQueue.getJob(jobId);
        if (job) {
            const state = await job.getState();
            const progress = await job.progress();
            const result = await job.finished().catch(err => ({ error: err.message }));
            return { state, progress, result };
        } else {
            return { state: "not found" };
        }
    } catch (error) {
        logger.error(`Failed to get status for LLM task with jobId ${jobId}.`, { error });
        throw error;
    }
};
