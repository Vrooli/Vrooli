import { LanguageModelResponseMode, LlmTask, MINUTES_1_MS, RunContext, Success, TaskContextInfo } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { SessionUserToken } from "../../types.js";
import { PreMapUserData } from "../../utils/chat.js";
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
     * The mode to use when generating the response
     */
    mode: LanguageModelResponseMode;
    /** 
     * The ID of the message that triggered the bot response. 
     * 
     * NOTE: This should be stored in the database already. Otherwise, 
     * we cannot use it to collect all previous messages for the LLM context.
     */
    parentId?: string | null;
    /** 
     * The text of the message you typed that triggered the bot response. 
     * 
     * If provided, we check for and process any commands in the message. 
     * If the message was only commands, we skip generating a response.
     * 
     * NOTE: You should skip this if you don't want to process the commands 
     * in the parent message, even if there is a parent.
     */
    parentMessage: string | null;
    /**
     * Information about other participants in the chat, if already known
     */
    participantsData?: Record<string, PreMapUserData>;
    /**
     * The ID of the bot that should generate the response
     */
    respondingBotId: string;
    /**
     * The context for the current run, if any
     */
    runContext?: RunContext;
    /**
     * If true, we'll always suggest tasks instead of running them immediately. 
     * This is useful for things like autofilling forms.
     */
    shouldNotRunTasks: boolean;
    /**
     * The current task being performed
     */
    task: LlmTask | `${LlmTask}`;
    /**
     * Any context data for the task
     */
    taskContexts: TaskContextInfo[];
    /**
     * The user data of the user who triggered the bot response
     */
    userData: SessionUserToken;
}
// TODO can provide state management to bot message payload by adding a routineId field and some field that describes our spot in the routine. This plus passing in data (for autofilling forms, etc.) and passing data to the next step in the routine will allow us to automate routines. Then we can build all of the routines needed to build/improve routines, and we should be good to go!

export type LlmTestPayload = {
    __process: "Test";
}

export type LlmRequestPayload = RequestBotMessagePayload | LlmTestPayload;

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
