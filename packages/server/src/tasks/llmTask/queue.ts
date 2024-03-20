import { SessionUserToken } from "@local/server";
import { LlmTaskInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";

export type LlmTaskProcessPayload = {
    /** The task to be run */
    taskInfo: Omit<LlmTaskInfo, "status">;
    /** The chat the command was triggered in */
    chatId?: string | null;
    /** The language the command was triggered in */
    language: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let llmTaskProcess: (job: Bull.Job<LlmTaskProcessPayload>) => Promise<unknown>;
let llmTaskQueue: Bull.Queue<LlmTaskProcessPayload>;

// Call this on server startup
export async function setupLlmTaskQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        llmTaskProcess = processModule.llmTaskProcess;

        // Initialize the Bull queue
        llmTaskQueue = new Bull<LlmTaskProcessPayload>("command", {
            redis: { port: PORT, host: HOST },
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

export const processLlmTask = (data: LlmTaskProcessPayload) => {
    llmTaskQueue.add(data, { timeout: 1000 * 60 * 1 });
};
