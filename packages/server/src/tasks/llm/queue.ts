import Bull from "bull";
import winston from "winston";
import { PreMapMessageData, PreMapUserData } from "../../models/base/chatMessage.js";
import { SessionUserToken } from "../../types.js";

export type RequestBotResponsePayload = {
    chatId: string;
    messageId: string;
    message: PreMapMessageData;
    respondingBotId: string;
    participantsData: Record<string, PreMapUserData>;
    userData: SessionUserToken;
}

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let llmProcess: (job: Bull.Job<RequestBotResponsePayload>) => Promise<unknown>;
let llmQueue: Bull.Queue<RequestBotResponsePayload>;

// Call this on server startup
export async function setupLlmQueue() {
    try {
        const loggerModule = await import("../../events/logger.js");
        logger = loggerModule.logger;

        const redisConnModule = await import("../../redisConn.js");
        HOST = redisConnModule.HOST;
        PORT = redisConnModule.PORT;

        const processModule = await import("./process.js");
        llmProcess = processModule.llmProcess;

        // Initialize the Bull queue
        llmQueue = new Bull<RequestBotResponsePayload>("llm", {
            redis: { port: PORT, host: HOST }
        });
        llmQueue.process(llmProcess);
    } catch (error) {
        const errorMessage = "Failed to setup sms queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0212", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

/**
 * Responds to a chat message. Handles response generation and processing, 
 * websocket events, and any other logic
 */
export function requestBotResponse(props: RequestBotResponsePayload) {
    llmQueue.add(props, { timeout: 1000 * 60 * 3 });
}
