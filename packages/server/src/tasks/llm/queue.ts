import { LlmTask } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { PreMapMessageData, PreMapUserData } from "../../models/base/chatMessage.js";
import { SessionUserToken } from "../../types.js";

/**
 * Payload for generating a bot response in a chat
 */
export type RequestBotMessagePayload = {
    __process: "BotMessage";
    /**
     * The chat we're responding in
     */
    chatId: string;
    /** 
     * The message to respond to. If null, we 
     * assume that there are no messages in the chat
     * (and thus we are about to create the initial message).
     */
    parent: PreMapMessageData | null;
    /**
     * Information about other participants in the chat
     */
    participantsData: Record<string, PreMapUserData>;
    /**
     * The ID of the bot that should generate the response
     */
    respondingBotId: string;
    /**
     * The current task being performed
     */
    task: LlmTask | `${LlmTask}`;
    /**
     * The user data of the user who triggered the bot response
     */
    userData: SessionUserToken;
}

/**
 * Payload for generating a bot response for autofill of a form
 */
export type RequestAutoFillPayload = {
    __process: "AutoFill";
    /**
     * The known form data
     */
    data: object;
    /**
     * The task being performed. For forms, this should be an "Add" 
     * or "Update" task, but technically any task besides "Start" 
     * could be used.
     */
    task: LlmTask | `${LlmTask}`;
    /**
    * The user data of the user who triggered the bot response
    */
    userData: SessionUserToken;
}

export type LlmRequestPayload = RequestBotMessagePayload | RequestAutoFillPayload;

let logger: winston.Logger;
let HOST: string;
let PORT: number;
let llmProcess: (job: Bull.Job<LlmRequestPayload>) => Promise<unknown>;
let llmQueue: Bull.Queue<LlmRequestPayload>;

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
        llmQueue = new Bull<LlmRequestPayload>("llm", {
            redis: { port: PORT, host: HOST },
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
export const requestBotResponse = (props: Omit<RequestBotMessagePayload, "__process">) => {
    llmQueue.add({ ...props, __process: "BotMessage" as const }, { timeout: 1000 * 60 * 3 });
};


/**
 * Requests autofill for a form. Handles response generation and processing,
 * websocket events, and any other logic
 */
export const requestAutoFill = (props: Omit<RequestAutoFillPayload, "__process">) => {
    llmQueue.add({ ...props, __process: "AutoFill" as const }, { timeout: 1000 * 60 * 3 });
};
