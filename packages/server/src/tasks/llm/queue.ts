import { HOURS_1_S, LlmTask, MINUTES_1_MS, Success } from "@local/shared";
import Bull from "bull";
import path from "path";
import { fileURLToPath } from "url";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { PreMapMessageData, PreMapUserData } from "../../utils/chat.js";
import { addJobToQueue } from "../queueHelper.js";

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
// TODO can provide state management to bot message payload by adding a routineId field and some field that describes our spot in the routine. This plus passing in data (for autofilling forms, etc.) and passing data to the next step in the routine will allow us to automate routines. Then we can build all of the routines needed to build/improve routines, and we should be good to go!

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

/**
 * Payload for starting a suggested task
 */
export type StartTaskPayload = {
    __process: "StartTask";
    /**
     * The ID of the bot the task will be performed by
     */
    botId: string;
    /**
     * Label for the task, to provide in notifications
     */
    label: string;
    /**
     * The ID of the message the task was suggested in. 
     * Used to grab the relevant chat context
     */
    messageId: string;
    /**
     * Any properties provided with the task
     */
    properties: object;
    /**
     * The task to start
     */
    task: LlmTask | `${LlmTask}`;
    /**
     * The ID of the task, so we can update its status in the UI
     */
    taskId: string;
    /**
     * The user data of the user who triggered the task
     */
    userData: SessionUserToken;

}

export type LlmRequestPayload = RequestBotMessagePayload | RequestAutoFillPayload | StartTaskPayload;

let logger: winston.Logger;
let llmProcess: (job: Bull.Job<LlmRequestPayload>) => Promise<unknown>;
let llmQueue: Bull.Queue<LlmRequestPayload>;
const dirname = path.dirname(fileURLToPath(import.meta.url));
const importExtension = process.env.NODE_ENV === "test" ? ".ts" : ".js";

// Call this on server startup
export async function setupLlmQueue() {
    try {
        const loggerPath = path.join(dirname, "../../events/logger" + importExtension);
        const loggerModule = await import(loggerPath);
        logger = loggerModule.logger;

        const redisConnPath = path.join(dirname, "../../redisConn" + importExtension);
        const redisConnModule = await import(redisConnPath);
        const REDIS_URL = redisConnModule.REDIS_URL;

        const processPath = path.join(dirname, "./process" + importExtension);
        const processModule = await import(processPath);
        llmProcess = processModule.llmProcess;

        // Initialize the Bull queue
        llmQueue = new Bull<LlmRequestPayload>("llm", {
            redis: REDIS_URL,
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
export function requestBotResponse(
    props: Omit<RequestBotMessagePayload, "__process">,
): Promise<Success> {
    return addJobToQueue(llmQueue,
        { ...props, __process: "BotMessage" as const },
        { timeout: MINUTES_1_MS });
}

/**
 * Requests autofill for a form. Handles response generation and processing,
 * websocket events, and any other logic
 */
export function requestAutoFill(
    props: Omit<RequestAutoFillPayload, "__process">,
): Promise<Success> {
    return addJobToQueue(llmQueue,
        { ...props, __process: "AutoFill" as const },
        { timeout: MINUTES_1_MS });
}

/**
 * Requests a specific task to be started. Handles response generation and processing,
 * websocket events, and any other logic
 */
export function requestStartTask(
    props: Omit<StartTaskPayload, "__process">,
): Promise<Success> {
    return addJobToQueue(llmQueue,
        { ...props, __process: "StartTask" as const },
        { timeout: MINUTES_1_MS });
}
