import { LlmTask, MINUTES_1_MS, StartLlmTaskInput, Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { PreMapMessageData, PreMapUserData } from "../../utils/chat.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper";

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
export type StartLlmTaskPayload = StartLlmTaskInput & {
    __process: "StartTask";
    /**
     * The user data of the user who triggered the task
     */
    userData: SessionUserToken;

}

export type LlmTestPayload = {
    __process: "Test";
}

export type LlmRequestPayload = RequestBotMessagePayload | RequestAutoFillPayload | StartLlmTaskPayload | LlmTestPayload;

let logger: winston.Logger;
let llmProcess: (job: Bull.Job<LlmRequestPayload>) => Promise<unknown>;
let llmQueue: Bull.Queue<LlmRequestPayload>;
const FOLDER = "llm";

// Call this on server startup
export async function setupLlmQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).REDIS_URL;
        llmProcess = (await import(getProcessPath(FOLDER))).llmProcess;

        // Initialize the Bull queue
        llmQueue = new Bull<LlmRequestPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        llmQueue.process(llmProcess);
        // Verify that the queue is working
        addJobToQueue(llmQueue, { __process: "Test" }, { timeout: MINUTES_1_MS });
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
export function requestStartLlmTask(
    props: Omit<StartLlmTaskPayload, "__process">,
): Promise<Success> {
    return addJobToQueue(llmQueue,
        { ...props, __process: "StartTask" as const },
        { timeout: MINUTES_1_MS });
}
