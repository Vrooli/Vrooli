import { BUSINESS_NAME, Success } from "@local/shared";
import Bull from "bull";
import winston from "winston";
import { CustomError } from "../../events/error.js";
import { DEFAULT_JOB_OPTIONS, LOGGER_PATH, REDIS_CONN_PATH, addJobToQueue, getProcessPath } from "../queueHelper.js";

export type SmsProcessPayload = {
    to: string[];
    body: string;
}

let logger: winston.Logger;
let smsProcess: (job: Bull.Job<SmsProcessPayload>) => Promise<unknown>;
let smsQueue: Bull.Queue<SmsProcessPayload>;
const FOLDER = "sms";

// Call this on server startup
export async function setupSmsQueue() {
    try {
        logger = (await import(LOGGER_PATH)).logger;
        const REDIS_URL = (await import(REDIS_CONN_PATH)).getRedisUrl();
        smsProcess = (await import(getProcessPath(FOLDER))).smsProcess;

        // Initialize the Bull queue
        smsQueue = new Bull<SmsProcessPayload>(FOLDER, {
            redis: REDIS_URL,
            defaultJobOptions: DEFAULT_JOB_OPTIONS,
        });
        smsQueue.process(smsProcess);
    } catch (error) {
        const errorMessage = "Failed to setup sms queue";
        if (logger) {
            logger.error(errorMessage, { trace: "0211", error });
        } else {
            console.error(errorMessage, error);
        }
    }
}

export function sendSms(to: string[], body: string): Promise<Success> {
    // Must include at least one "to" number
    if (to.length === 0) {
        throw new CustomError("0353", "InternalError");
    }
    return addJobToQueue(smsQueue, { to, body }, {});
}

/** Adds a verification code text to a task queue */
export function sendSmsVerification(phoneNumber: string, code: string): Promise<Success> {
    return addJobToQueue(smsQueue, {
        to: [phoneNumber],
        body: `${code} is your ${BUSINESS_NAME} verification code`,
    }, {});
}
