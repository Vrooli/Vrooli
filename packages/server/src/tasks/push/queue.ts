import { Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper";

export type PushSubscription = {
    endpoint: string;
    keys: {
        p256dh: string;
        auth: string;
    };
}

export type PushPayload = {
    body: string;
    icon?: string | null;
    link: string | null | undefined;
    title: string | null | undefined;
}

let logger: winston.Logger;
let pushProcess: (job: Bull.Job<PushSubscription & PushPayload>) => Promise<unknown>;
let pushQueue: Bull.Queue<PushSubscription & PushPayload>;
const FOLDER = "push";

// Call this on server startup
export async function setupPushQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).REDIS_URL;
        pushProcess = (await import(getProcessPath(FOLDER))).pushProcess;

        // Initialize the Bull queue
        pushQueue = new Bull<PushSubscription & PushPayload>("push", {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        pushQueue.process(pushProcess);
    } catch (error) {
        const errorMessage = "Failed to setup push queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0205", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export function sendPush(
    subscription: PushSubscription,
    payload: PushPayload,
    delay = 0,
): Promise<Success> {
    return addJobToQueue(pushQueue, {
        ...payload,
        ...subscription,
    }, { delay });
}
