import { SessionUserToken } from "@local/server";
import Bull from "bull";
import winston from "winston";
import { LlmCommand } from "../llm/commands";

export type CommandProcessPayload = {
    /** The command to be run */
    command: LlmCommand;
    /** The chat the command was triggered in */
    chatId: string
    /** The language the command was triggered in */
    language: string;
    /** The user who's running the command (not the bot) */
    userData: SessionUserToken;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let commandProcess: (job: Bull.Job<CommandProcessPayload>) => Promise<unknown>;
let commandQueue: Bull.Queue<CommandProcessPayload>;

// Call this on server startup
export async function setupCommandQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        commandProcess = processModule.commandProcess;

        // Initialize the Bull queue
        commandQueue = new Bull<CommandProcessPayload>("command", {
            redis: { port: PORT, host: HOST },
        });
        commandQueue.process(commandProcess);
    } catch (error) {
        const errorMessage = "Failed to setup command queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0220", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export const processCommand = (data: CommandProcessPayload) => {
    commandQueue.add(data, { timeout: 1000 * 60 * 1 });
};
