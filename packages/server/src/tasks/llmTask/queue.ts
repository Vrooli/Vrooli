import { SessionUserToken } from "@local/server";
import { LlmTaskInfo } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
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

export const processLlmTask = (data: LlmTaskProcessPayload) => {
    llmTaskQueue.add(data, { timeout: 1000 * 60 * 1 });
};
